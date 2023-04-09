package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/wneessen/go-mail"
)

type receiver struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type email struct {
	Receiver        receiver         `json:"receiver"`
	Subject         string           `json:"subject"`
	Message         string           `json:"message"`
	Type            mail.ContentType `json:"type"`
	Attachments     []string         `json:"attachments"`
	attachmentsSize int
	messageQueue    amqp.Delivery
	messageMail     *mail.Msg
	error           error
}

type send struct {
	*cache
	*sender
	*metrics
	*smtp
	infos  chan string
	errors chan string
}

func newSend(cache *cache, sender *sender, smtp *smtp, metrics *metrics) *send {
	return &send{
		cache:   cache,
		sender:  sender,
		metrics: metrics,
		smtp:    smtp,
		infos:   make(chan string),
		errors:  make(chan string),
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

func proccesEmails(cache *cache, sender *sender, emails, failed []email) ([]email, []email) {
	ready := []email{}

emailToMessage:
	for _, email := range emails {
		message := mail.NewMsg()

		err := message.EnvelopeFromFormat(sender.Name, sender.Email)
		if err != nil {
			email.error = fmt.Errorf("error adding email sender: %w", err)
			failed = append(failed, email)

			continue
		}

		err = message.AddToFormat(email.Receiver.Name, email.Receiver.Email)
		if err != nil {
			email.error = fmt.Errorf("error adding email receiver: %w", err)
			failed = append(failed, email)

			continue
		}

		for _, attachment := range email.Attachments {
			file, err := cache.getFile(attachment)
			if err != nil {
				email.error = fmt.Errorf("error getting attachment from cache: %w", err)
				failed = append(failed, email)

				continue emailToMessage
			}

			email.attachmentsSize += len(file)
			message.AttachReadSeeker(attachment, bytes.NewReader(file))
		}

		message.Subject(email.Subject)
		message.SetBodyString(email.Type, email.Message)

		email.messageMail = message

		ready = append(ready, email)
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
		for _, email := range ready {
			email.error = err
			failed = append(failed, email)
		}

		return []email{}, failed
	}

	messages := []*mail.Msg{}
	for _, email := range ready {
		messages = append(messages, email.messageMail)
	}

	err = client.DialAndSend(messages...)
	if err != nil {
		for _, email := range ready {
			email.error = err
			failed = append(failed, email)
		}

		return []email{}, failed
	}

	return ready, failed
}

func proccessAcknowledgment(emails, failed []email) ([]email, []email) {
	ready := []email{}

	for _, email := range emails {
		err := email.messageQueue.Ack(false)
		if err != nil {
			email.error = fmt.Errorf("error sending a termination message to RabbitMQ: %w", err)
			failed = append(failed, email)
		} else {
			ready = append(ready, email)
		}
	}

	return ready, failed
}

func proccessNotAcknowledgment(emails []email) []error {
	errs := []error{}

	for _, email := range emails {
		err := email.messageQueue.Nack(false, true)
		if err != nil {
			errs = append(errs, fmt.Errorf("error resending message to the queue: %w", err))
		}

		errs = append(errs, email.error)
	}

	return errs
}

func (send *send) setMetrics(timeInit time.Time, ready, failed []email) {
	receivedBytes := 0
	sentEmails := 0
	sentBytes := 0
	sentAttachment := 0
	sentAttachmentsBytes := 0
	sentWithAttachemnt := 0

	for _, email := range failed {
		receivedBytes += len(email.messageQueue.Body)
	}

	for _, email := range ready {
		receivedBytes += len(email.messageQueue.Body)
		sentEmails++
		sentBytes += len(email.Message)

		attachmentsSize := len(email.Attachments)
		if attachmentsSize > 0 {
			sentAttachment += attachmentsSize
			sentAttachmentsBytes += email.attachmentsSize
			sentWithAttachemnt++
		}
	}

	send.metrics.emailsReceived.Add(float64(len(ready) + len(failed)))
	send.metrics.emailsReceivedBytes.Add(float64(receivedBytes))
	send.metrics.emailsSent.Add(float64(sentEmails))
	send.metrics.emailsSentBytes.Add(float64(sentBytes))
	send.metrics.emailsSentAttachment.Add(float64(sentAttachment))
	send.metrics.emailsSentAttachmentBytes.Add(float64(sentAttachmentsBytes))
	send.metrics.emailsSentWithAttachment.Add(float64(sentWithAttachemnt))
  send.metrics.emailsSentTimeSeconds.Observe(time.Since(timeInit).Seconds())
	send.metrics.emailsResent.Add(float64(len(failed)))
}

func (send *send) emails(queue []amqp.Delivery) []error {
	timeInit := time.Now()

	emailsReady, emailsFailed := proccessQueue(queue)
	emailsReady, emailsFailed = proccesEmails(send.cache, send.sender, emailsReady, emailsFailed)
	emailsReady, emailsFailed = sendEmails(send.smtp, emailsReady, emailsFailed)
	emailsReady, emailsFailed = proccessAcknowledgment(emailsReady, emailsFailed)

	err := proccessNotAcknowledgment(emailsFailed)

	send.setMetrics(timeInit, emailsReady, emailsFailed)

	send.infos <- fmt.Sprintf("Has been sent %d emails", len(emailsReady))

	return err
}

func (send *send) copyQueueAndSendEmails(queue []amqp.Delivery) []amqp.Delivery {
	buffer := make([]amqp.Delivery, len(queue))
	copy(buffer, queue)

	send.infos <- fmt.Sprintf("Sending %d emails", len(buffer))

	go send.emails(buffer)

	return queue[:0]
}
