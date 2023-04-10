package main

import (
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func newRabbit(configs *configurations) (<-chan amqp.Delivery, func(), error) {
	rabbitURL := fmt.Sprintf(
		"amqp://%s:%s@%s:%d/%s",
		configs.Rabbit.User,
		configs.Rabbit.Password,
		configs.Rabbit.Host,
		configs.Rabbit.Port,
		configs.Rabbit.Vhost,
	)

	rabbit, err := amqp.Dial(rabbitURL)
	if err != nil {
		return nil, nil, fmt.Errorf("error connecting to RabbitMQ: %w", err)
	}

	channel, err := rabbit.Channel()
	if err != nil {
		rabbit.Close()

		return nil, nil, fmt.Errorf("error opening RabbitMQ channel: %w", err)
	}

	closeRabbit := func() {
		channel.Close()
		rabbit.Close()
	}

	dlx := configs.Rabbit.Queue + "-dlx"
	queueArgs := amqp.Table{}
	queueArgs["x-dead-letter-exchange"] = dlx
	queueArgs["x-dead-letter-routing-key"] = "dead-message"
	queueArgs["x-queue-type"] = "quorum"
	queueArgs["x-delivery-limit"] = configs.Rabbit.MaxRetries

	_, err = channel.QueueDeclare(configs.Rabbit.Queue, true, false, false, false, queueArgs)
	if err != nil {
		closeRabbit()

		return nil, nil, fmt.Errorf("error declaring RabbitMQ queue: %w", err)
	}

	_, err = channel.QueueDeclare(dlx, true, false, false, false, nil)
	if err != nil {
		closeRabbit()

		return nil, nil, fmt.Errorf("error declaring RabbitMQ dlx queue: %w", err)
	}

	err = channel.ExchangeDeclare(dlx, "direct", true, false, false, false, nil)
	if err != nil {
		closeRabbit()

		return nil, nil, fmt.Errorf("error declaring RabbitMQ dlx exchange: %w", err)
	}

	err = channel.QueueBind(dlx, "dead-message", dlx, false, nil)
	if err != nil {
		closeRabbit()

		return nil, nil, fmt.Errorf("error binding dlx queue with dlx exchange: %w", err)
	}

	err = channel.Qos(configs.Buffer.Size*configs.Buffer.Quantity, 0, false)
	if err != nil {
		closeRabbit()

		return nil, nil, fmt.Errorf("error configuring consumer queue size: %w", err)
	}

	queue, err := channel.Consume(configs.Rabbit.Queue, "", false, false, false, false, nil)
	if err != nil {
		closeRabbit()

		return nil, nil, fmt.Errorf("error registering consumer: %w", err)
	}

	return queue, closeRabbit, nil
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

	queue, closeRabbit, err := newRabbit(configs)
	if err != nil {
		log.Printf("[ERROR] - Error creating queue: %s", err)

		return
	}

	defer closeRabbit()

	metrics := newMetrics()
	send := newSend(cache, template, &configs.Sender, &configs.SMTP, metrics, configs.Rabbit.MaxRetries)
	timeout := time.Duration(configs.Timeout) * time.Second

	var wait chan struct{}

	go serverMetrics(metrics)

	go cacheMetrics(cache, metrics)

	go getMessages(queue, send, timeout, configs.Buffer.Size)

	go logSend(send)

	log.Printf("[INFO] - Server started successfully")
	<-wait
}
