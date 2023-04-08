package main

import (
	"encoding/json"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/wneessen/go-mail"
)

type receiver struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type email struct {
	Receiver      receiver         `json:"receiver"`
	Subject       string           `json:"subject"`
	Message       string           `json:"message"`
	Type          mail.ContentType `json:"type"`
	Attachments   []string         `json:"attachments"`
	messageRabbit amqp.Delivery
}

type send struct {
	*cache
	*sender
	*smtp
	metrics *metrics
}

func newSend(cache *cache, sender *sender, smtp *smtp, metrics *metrics) *send {
	return &send{
		cache:   cache,
		sender:  sender,
		smtp:    smtp,
		metrics: metrics,
	}
}

func (send *send) emailsToQueue(description string, err error, emails []email) {
	send.metrics.emailsResent.Add(float64(len(emails)))

	log.Printf("[ERROR] - Error processing a batch of emails, resending them to the queue")
	log.Printf("[ERROR] - %s: %s", description, err)

	for _, email := range emails {
		err = email.messageRabbit.Nack(false, true)
		if err != nil {
			log.Printf("[ERROR] Error resending message to the queue: %s", err)
		}
	}
}

func (send *send) messageToQueue(description string, err error, message amqp.Delivery) {
	send.metrics.emailsResent.Inc()

	log.Printf("[ERROR] - Error processing a message, resending it to the queue")
	log.Printf("[ERROR] - %s: %s", description, err)

	err = message.Nack(false, true)
	if err != nil {
		log.Printf("[ERROR] Error resending message to the queue: %s", err)
	}
}

func (send *send) emails(queue []amqp.Delivery) {
	timeInit := time.Now()

	send.metrics.emailsReceived.Add(float64(len(queue)))

	emails := []email{}
	bytesReceived := 0

	for _, message := range queue {
		bytesReceived += len(message.Body)

		email := email{}

		err := json.Unmarshal(message.Body, &email)
		if err != nil {
			description := "Error converting a message to an email"
			send.messageToQueue(description, err, message)
		} else {
			email.messageRabbit = message
			emails = append(emails, email)
		}
	}

	send.metrics.emailsReceivedBytes.Add(float64(bytesReceived))

	clientOption := []mail.Option{
		mail.WithPort(send.smtp.Port),
		mail.WithSMTPAuth(mail.SMTPAuthPlain),
		mail.WithUsername(send.smtp.User),
		mail.WithPassword(send.smtp.Password),
		mail.WithTLSPolicy(mail.TLSMandatory),
	}

	client, err := mail.NewClient(send.smtp.Host, clientOption...)
	if err != nil {
		descricao := "Error creating an email client"
		send.emailsToQueue(descricao, err, emails)

		return
	}

	messages := []*mail.Msg{}
	emailsReady := []email{}

	for _, email := range emails {
		message := mail.NewMsg()

		err = message.EnvelopeFromFormat(send.sender.Name, send.sender.Email)
		if err != nil {
			description := "Error adding email sender"
			send.messageToQueue(description, err, email.messageRabbit)

			continue
		}

		err = message.AddToFormat(email.Receiver.Name, email.Receiver.Email)
		if err != nil {
			description := "Error adding email receiver"
			send.messageToQueue(description, err, email.messageRabbit)

			continue
		}

		message.Subject(email.Subject)
		message.SetBodyString(email.Type, email.Message)

		messages = append(messages, message)
		emailsReady = append(emailsReady, email)
	}

	err = client.DialAndSend(messages...)
	if err != nil {
		description := "Error seding emails"
		send.emailsToQueue(description, err, emailsReady)

		return
	}

	emailsSent := 0
	bytesSent := 0

	for _, email := range emailsReady {
		err := email.messageRabbit.Ack(false)
		if err != nil {
			log.Printf("[ERROR] - Error sending a termination message to RabbitMQ: %s", err)
		} else {
			emailsSent++
			bytesSent += len(email.Message)
		}
	}

	tempoDecorrido := time.Since(timeInit).Seconds()

	send.metrics.emailsSent.Add(float64(emailsSent))
	send.metrics.emailsSentTimeSeconds.Observe(tempoDecorrido)
	send.metrics.emailsSentBytes.Add(float64(bytesSent))

	log.Printf("[INFO] - Foram enviado %d emails", emailsSent)
}

func (send *send) copyQueueAndSendEmails(queue []amqp.Delivery) []amqp.Delivery {
	buffer := make([]amqp.Delivery, len(queue))
	copy(buffer, queue)

	log.Printf("[INFO] - Sending %d emails", len(buffer))

	go send.emails(buffer)

	return queue[:0]
}
