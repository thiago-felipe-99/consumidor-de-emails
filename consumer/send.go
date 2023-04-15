package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	htmplate "html/template"
	"log"
	"regexp"
	"time"

	"github.com/microcosm-cc/bluemonday"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/russross/blackfriday/v2"
	"github.com/wneessen/go-mail"
)

type receiver struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type template struct {
	Name string         `json:"name"`
	Data map[string]any `json:"data"`
}

type email struct {
	Receivers       []receiver       `json:"receivers"`
	BlindReceivers  []receiver       `json:"blindReceivers"`
	Subject         string           `json:"subject"`
	Message         string           `json:"message"`
	Template        template         `json:"template"`
	Type            mail.ContentType `json:"type"`
	Attachments     []string         `json:"attachments"`
	attachmentsSize int
	messageQueue    amqp.Delivery
	messageMail     *mail.Msg
	error           error
}

type errorQuantity struct {
	error
	quantity int
}

type sendStatus struct {
	successfully int
	failed       int
	errors       []errorQuantity
}

type send struct {
	*cache
	*templateCache
	*sender
	*metrics
	*smtp
	status    chan sendStatus
	maxReties int64
}

func newSend(
	cache *cache,
	templateCache *templateCache,
	sender *sender,
	smtp *smtp,
	metrics *metrics,
	maxReties int64,
) *send {
	return &send{
		cache:         cache,
		templateCache: templateCache,
		sender:        sender,
		metrics:       metrics,
		smtp:          smtp,
		status:        make(chan sendStatus),
		maxReties:     maxReties,
	}
}

func proccessQueue(queue []amqp.Delivery) ([]email, []email) {
	ready, failed := []email{}, []email{}

	for _, message := range queue {
		email := email{
			messageQueue: message,
		}

		err := json.Unmarshal(message.Body, &email)
		if err != nil {
			email.error = fmt.Errorf("error converting a message to an email: %w", err)
			failed = append(failed, email)
		} else {
			ready = append(ready, email)
		}
	}

	return ready, failed
}

func emailFailed(index int, ready, failed []email) ([]email, []email) {
	failed = append(failed, ready[index])

	ready[index], ready[len(ready)-1] = ready[len(ready)-1], ready[index]

	return ready[:len(ready)-1], failed
}

func emailFailedUniqErr(err error, ready, failed []email) []email {
	for _, email := range ready {
		email.error = err
		failed = append(failed, email)
	}

	return failed
}

func getTemplateHTML(template template, cache *templateCache) (string, error) {
	markdown, err := cache.get(template.Name)
	if err != nil {
		return "", fmt.Errorf("error getting template from cache: %w", err)
	}

	rawHTML := blackfriday.Run(markdown)

	regex := regexp.MustCompile(`({{)( *)(\w+)( *)(}})`)
	replace := []byte("$1 index . \"$3\" $5")

	replaceHTML := regex.ReplaceAll(rawHTML, replace)

	templateHTML, err := htmplate.New("template").Parse(string(replaceHTML))
	if err != nil {
		return "", fmt.Errorf("erro parsing HTML: %w", err)
	}

	buffer := bytes.NewBuffer(make([]byte, len(replaceHTML)))

	err = templateHTML.Execute(buffer, template.Data)
	if err != nil {
		return "", fmt.Errorf("error executing template: %w", err)
	}

	html := bluemonday.UGCPolicy().SanitizeReader(buffer)

	return html.String(), nil
}

func createEmailMessage(
	cache *cache,
	template *templateCache,
	sender *sender,
	email email,
) (*mail.Msg, int, error) {
	message := mail.NewMsg()
	attachmentsSize := 0

	err := message.EnvelopeFromFormat(sender.Name, sender.Email)
	if err != nil {
		return nil, 0, fmt.Errorf("error adding email sender: %w", err)
	}

	for _, receiver := range email.Receivers {
		err = message.AddToFormat(receiver.Name, receiver.Email)
		if err != nil {
			return nil, 0, fmt.Errorf("error adding email receiver: %w", err)
		}
	}

	for _, receiver := range email.BlindReceivers {
		err = message.AddBccFormat(receiver.Name, receiver.Email)
		if err != nil {
			return nil, 0, fmt.Errorf("error adding email blind receiver: %w", err)
		}
	}

	for _, attachment := range email.Attachments {
		file, err := cache.getFile(attachment)
		if err != nil {
			return nil, 0, fmt.Errorf("error getting attachment from cache: %w", err)
		}

		attachmentsSize += len(file)
		message.AttachReadSeeker(attachment, bytes.NewReader(file))
	}

	message.Subject(email.Subject)

	if email.Template.Name != "" {
		html, err := getTemplateHTML(email.Template, template)
		if err != nil {
			return nil, 0, fmt.Errorf("error getting template HTML: %w", err)
		}

		message.SetBodyString(mail.TypeTextHTML, html)
	} else {
		message.SetBodyString(email.Type, email.Message)
	}

	return message, attachmentsSize, nil
}

func proccessEmails(
	cache *cache,
	templateCache *templateCache,
	sender *sender,
	ready, failed []email,
) ([]email, []email) {
	for index := len(ready) - 1; index >= 0; index-- {
		message, attachmentsSize, err := createEmailMessage(
			cache,
			templateCache,
			sender,
			ready[index],
		)
		if err != nil {
			ready[index].error = err
			ready, failed = emailFailed(index, ready, failed)
		} else {
			ready[index].messageMail = message
			ready[index].attachmentsSize = attachmentsSize
		}
	}

	return ready, failed
}

func sendEmails(smtp *smtp, ready, failed []email) ([]email, []email) {
	clientOption := []mail.Option{
		mail.WithPort(smtp.Port),
		mail.WithSMTPAuth(mail.SMTPAuthPlain),
		mail.WithUsername(smtp.User),
		mail.WithPassword(smtp.Password),
		mail.WithTLSPolicy(mail.TLSMandatory),
	}

	client, err := mail.NewClient(smtp.Host, clientOption...)
	if err != nil {
		return []email{}, emailFailedUniqErr(err, ready, failed)
	}

	messages := []*mail.Msg{}
	for _, email := range ready {
		messages = append(messages, email.messageMail)
	}

	err = client.DialWithContext(context.Background())
	if err != nil {
		return []email{}, emailFailedUniqErr(err, ready, failed)
	}

	err = client.Send(messages...)
	if err != nil {
		for index, email := range ready {
			if email.messageMail.HasSendError() {
				ready[index].error = email.messageMail.SendError()
				ready, failed = emailFailed(index, ready, failed)
			}
		}

		return ready, failed
	}

	return ready, failed
}

func proccessAcknowledgment(ready, failed []email) ([]email, []email) {
	for index := len(ready) - 1; index >= 0; index-- {
		err := ready[index].messageQueue.Ack(false)
		if err != nil {
			ready[index].error = err
			ready, failed = emailFailed(index, ready, failed)
		}
	}

	return ready, failed
}

func appendIfMissing(items []errorQuantity, item error) []errorQuantity {
	for index, ele := range items {
		if errors.Is(item, ele.error) {
			items[index].quantity++

			return items
		}
	}

	return append(items, errorQuantity{error: item, quantity: 1})
}

func proccessNotAcknowledgment(emails []email) []errorQuantity {
	errs := []errorQuantity{}

	for _, email := range emails {
		err := email.messageQueue.Nack(false, true)
		if err != nil {
			errs = appendIfMissing(
				errs,
				fmt.Errorf("error resending message to the queue: %w", err),
			)
		}

		errs = appendIfMissing(errs, email.error)
	}

	return errs
}

func setMetrics(metrics *metrics, timeInit time.Time, ready, failed []email, maxRetries int64) {
	receivedBytes := 0
	sentEmails := 0
	sentBytes := 0
	sentAttachment := 0
	sentAttachmentsBytes := 0
	sentWithAttachemnt := 0
	sentMaxRetries := 0

	for _, email := range failed {
		receivedBytes += len(email.messageQueue.Body)

		if value, okay := email.messageQueue.Headers["x-delivery-count"]; okay {
			if retries, okay := value.(int64); okay {
				if retries >= maxRetries {
					sentMaxRetries++
				}
			}
		}
	}

	for _, email := range ready {
		receivedBytes += len(email.messageQueue.Body)
		sentBytes += len(email.Message)
		sentEmails++

		attachmentsSize := len(email.Attachments)
		if attachmentsSize > 0 {
			sentAttachment += attachmentsSize
			sentAttachmentsBytes += email.attachmentsSize
			sentWithAttachemnt++
		}
	}

	metrics.emailsReceived.Add(float64(len(ready) + len(failed)))
	metrics.emailsReceivedBytes.Add(float64(receivedBytes))
	metrics.emailsSent.Add(float64(sentEmails))
	metrics.emailsSentBytes.Add(float64(sentBytes))
	metrics.emailsSentAttachment.Add(float64(sentAttachment))
	metrics.emailsSentAttachmentBytes.Add(float64(sentAttachmentsBytes))
	metrics.emailsSentWithAttachment.Add(float64(sentWithAttachemnt))
	metrics.emailsResent.Add(float64(len(failed)))
	metrics.emailsSentMaxRetries.Add(float64(sentMaxRetries))
	metrics.emailsSentTimeSeconds.Observe(time.Since(timeInit).Seconds())
}

func (send *send) emails(queue []amqp.Delivery) {
	timeInit := time.Now()

	emailsReady, emailsFailed := proccessQueue(queue)
	emailsReady, emailsFailed = proccessEmails(
		send.cache,
		send.templateCache,
		send.sender,
		emailsReady,
		emailsFailed,
	)
	emailsReady, emailsFailed = sendEmails(send.smtp, emailsReady, emailsFailed)
	emailsReady, emailsFailed = proccessAcknowledgment(emailsReady, emailsFailed)

	err := proccessNotAcknowledgment(emailsFailed)

	send.status <- sendStatus{
		successfully: len(emailsReady),
		failed:       len(emailsFailed),
		errors:       err,
	}

	setMetrics(send.metrics, timeInit, emailsReady, emailsFailed, send.maxReties)
}

func (send *send) copyQueueAndSendEmails(queue []amqp.Delivery) []amqp.Delivery {
	buffer := make([]amqp.Delivery, len(queue))
	copy(buffer, queue)

	log.Printf("[INFO] - Sending %d emails", len(buffer))

	go send.emails(buffer)

	return queue[:0]
}
