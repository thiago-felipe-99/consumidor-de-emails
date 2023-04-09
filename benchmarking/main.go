package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	amqp "github.com/rabbitmq/amqp091-go"
)

type rabbit struct {
	user, password, host, port, vhost, queue string
}

type configurations struct {
	rabbit
	messagesQuantity  int
	contentType, body string
}

func getConfigurations() (*configurations, error) {
	err := godotenv.Load()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	quantidadeDeMensagens, err := strconv.Atoi(os.Getenv("MESSAGES_QUANTITY"))
	if err != nil {
		return nil, err
	}

	config := &configurations{
		rabbit: rabbit{
			user:     os.Getenv("RABBIT_USER"),
			password: os.Getenv("RABBIT_PASSWORD"),
			host:     os.Getenv("RABBIT_HOST"),
			port:     os.Getenv("RABBIT_PORT"),
			vhost:    os.Getenv("RABBIT_VHOST"),
			queue:    os.Getenv("RABBIT_QUEUE"),
		},
		messagesQuantity: quantidadeDeMensagens,
		contentType:      os.Getenv("CONTENT_TYPE"),
		body:             os.Getenv("BODY"),
	}

	return config, nil
}

func main() {
	configs, err := getConfigurations()
	if err != nil {
		log.Printf("[ERROR] - Error reading the configurations: %s", err)

		return
	}

	rabbitURL := fmt.Sprintf(
		"amqp://%s:%s@%s:%s/%s",
		configs.rabbit.user,
		configs.rabbit.password,
		configs.rabbit.host,
		configs.rabbit.port,
		configs.rabbit.vhost,
	)

	rabbit, err := amqp.Dial(rabbitURL)
	if err != nil {
		log.Printf("[ERROR] - Error connecting to RabbitMQ: %s", err)

		return
	}
	defer rabbit.Close()

	channel, err := rabbit.Channel()
	if err != nil {
		log.Printf("[ERROR] - Error opening RabbitMQ channel: %s", err)

		return
	}
	defer channel.Close()

	dlx := configs.rabbit.queue + "-dlx"
	queueArgs := amqp.Table{}
	queueArgs["x-dead-letter-exchange"] = configs.rabbit.queue + "-dlx"
	queueArgs["x-dead-letter-routing-key"] = "dead-message"
	queueArgs["x-delivery-limit"] = 2
	queueArgs["x-queue-type"] = "quorum"

	queue, err := channel.QueueDeclare(configs.rabbit.queue, true, false, false, false, queueArgs)
	if err != nil {
		log.Println("[ERROR] - Error declaring RabbitMQ queue: %w", err)

		return
	}

	_, err = channel.QueueDeclare(dlx, true, false, false, false, nil)
	if err != nil {
		log.Println("[ERROR] - Error declaring RabbitMQ dlx queue: %w", err)

		return
	}

	err = channel.ExchangeDeclare(dlx, "direct", true, false, false, false, nil)
	if err != nil {
		log.Println("[ERROR] - Error declaring RabbitMQ dlx exchange: %w", err)

		return
	}

	err = channel.QueueBind(dlx, "dead-message", dlx, false, nil)
	if err != nil {
		log.Println("[ERROR] - Error binding dlx queue with dlx exchange: %w", err)

		return
	}

	log.Printf(
		"[INFO] - The queue '%s' has %d messages and %d consumers",
		queue.Name,
		queue.Messages,
		queue.Consumers,
	)

	message := amqp.Publishing{
		ContentType: configs.contentType,
		Body:        []byte(configs.body),
	}

	for i := 1; i <= configs.messagesQuantity; i++ {
		err := channel.PublishWithContext(
			context.Background(),
			"",
			queue.Name,
			false,
			false,
			message,
		)
		if err != nil {
			log.Printf("[ERROR] - Error sending message to queue: %s", err)

			return
		}
	}

	log.Printf(
		"[INFO] - %d messages were sent to queue '%s'",
		configs.messagesQuantity,
		queue.Name,
	)
}
