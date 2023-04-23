package main

import (
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/thiago-felipe-99/mail/rabbit"
)

func newRabbit(configs *configurations) *rabbit.Rabbit {
	config := rabbit.Config{
		User:     configs.Rabbit.User,
		Password: configs.Rabbit.Password,
		Host:     configs.Rabbit.Host,
		Port:     fmt.Sprint(configs.Rabbit.Port),
		Vhost:    configs.Rabbit.Vhost,
	}

	rabbit := rabbit.New(config)

	return rabbit
}

func consumeMessages(rabbit *rabbit.Rabbit, configs *configurations, queue chan<- amqp.Delivery) {
	sleep := time.Second

	for {
		err := rabbit.CreateQueue(configs.Rabbit.Queue, configs.Rabbit.MaxRetries)
		if err != nil {
			log.Printf("[ERROR] - Erro creating queue: %s", err)

			time.Sleep(sleep)
			sleep *= 2

			continue
		}

		sleep = time.Second

		messages, err := rabbit.Consume(
			configs.Rabbit.Queue,
			configs.Buffer.Size*configs.Buffer.Quantity,
		)
		if err != nil {
			log.Printf("[ERROR] - Error consuming the queue: %s", err)

			continue
		}

		log.Printf("[INFO] - Consuming the queue")

		for message := range messages {
			queue <- message
		}
	}
}

func getMessages(
	queue <-chan amqp.Delivery,
	send *send,
	timeout time.Duration,
	bufferSize int,
) {
	buffer := []amqp.Delivery{}
	ticker := time.NewTicker(timeout)

	for {
		select {
		case message := <-queue:
			buffer = append(buffer, message)

			ticker.Reset(timeout)

			if len(buffer) >= bufferSize {
				buffer = send.copyQueueAndSendEmails(buffer)
			}

		case <-ticker.C:
			if len(buffer) > 0 {
				buffer = send.copyQueueAndSendEmails(buffer)
			}
		}
	}
}

func logSend(send *send) {
	for status := range send.status {
		if status.successfully > 0 {
			log.Printf("[INFO] - Were sent %d successfully emails", status.successfully)
		}

		if status.failed > 0 {
			log.Printf("[ERROR] - Failed to send %d emails", status.failed)
		}

		for _, err := range status.errors {
			log.Printf("[ERROR] - %d errors with message: %s", err.quantity, err.error)
		}
	}
}

func main() {
	configs, err := getConfigurations()
	if err != nil {
		log.Printf("[ERROR] - Error reading the configurations: %s", err)

		return
	}

	cache, err := newCache(configs)
	if err != nil {
		log.Printf("[ERROR] - Error creating the files cache: %s", err)

		return
	}

	template, err := newTemplate(configs)
	if err != nil {
		log.Printf("[ERROR] - Error creating the files cache: %s", err)

		return
	}

	template.setAll()

	queue := make(chan amqp.Delivery)

	rabbit := newRabbit(configs)

	go rabbit.HandleConnection()

	go consumeMessages(rabbit, configs, queue)

	defer rabbit.Close()

	metrics := newMetrics()
	send := newSend(
		cache,
		template,
		&configs.Sender,
		&configs.SMTP,
		metrics,
		configs.Rabbit.MaxRetries,
	)
	timeout := time.Duration(configs.Timeout) * time.Second

	var wait chan struct{}

	go serverMetrics(metrics)

	go cacheMetrics(cache, metrics)

	go getMessages(queue, send, timeout, configs.Buffer.Size)

	go logSend(send)

	log.Printf("[INFO] - Server started successfully")
	<-wait
}
