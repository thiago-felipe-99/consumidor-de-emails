package rabbit

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Config struct {
	User     string
	Password string
	Host     string
	Port     string
	Vhost    string
}

type Rabbit struct {
	connection *amqp.Connection
	channel    *amqp.Channel
}

func (rabbit *Rabbit) Close() error {
	err := rabbit.channel.Close()
	if err != nil {
		return fmt.Errorf("error on closing channel: %w", err)
	}

	err = rabbit.connection.Close()
	if err != nil {
		return fmt.Errorf("error on closing RabbitMQ connection: %w", err)
	}

	return nil
}

func (rabbit *Rabbit) CreateQueue(name string, maxRetries int) error {
	dlx := name + "-dlx"

	queueArgs := amqp.Table{}
	queueArgs["x-dead-letter-exchange"] = name + "-dlx"
	queueArgs["x-dead-letter-routing-key"] = "dead-message"
	queueArgs["x-delivery-limit"] = maxRetries
	queueArgs["x-queue-type"] = "quorum"

	_, err := rabbit.channel.QueueDeclare(name, true, false, false, false, queueArgs)
	if err != nil {
		return fmt.Errorf("error declaring RabbitMQ queue: %w", err)
	}

	_, err = rabbit.channel.QueueDeclare(dlx, true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("error declaring RabbitMQ dlx queue: %w", err)
	}

	err = rabbit.channel.ExchangeDeclare(dlx, "direct", true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("error declaring RabbitMQ dlx exchange: %w", err)
	}

	err = rabbit.channel.QueueBind(dlx, "dead-message", dlx, false, nil)
	if err != nil {
		return fmt.Errorf("error binding dlx queue with dlx exchange: %w", err)
	}

	return nil
}

func (rabbit *Rabbit) SendMessage(ctx context.Context, queue string, message any) error {
	messageEncoding, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("error enconding message: %w", err)
	}

	publish := amqp.Publishing{
		ContentType: "application/json",
		Body:        messageEncoding,
	}

	err = rabbit.channel.PublishWithContext(ctx, "", queue, false, false, publish)
	if err != nil {
		return fmt.Errorf("error sending message: %w", err)
	}

	return nil
}

func New(config Config) (*Rabbit, error) {
	url := fmt.Sprintf(
		"amqp://%s:%s@%s:%s/%s",
		config.User,
		config.Password,
		config.Host,
		config.Port,
		config.Vhost,
	)

	rabbit, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("error connecting to RabbitMQ: %w", err)
	}

	channel, err := rabbit.Channel()
	if err != nil {
		rabbit.Close()

		return nil, fmt.Errorf("error opening RabbitMQ channel: %w ", err)
	}

	return &Rabbit{
		connection: rabbit,
		channel:    channel,
	}, nil
}
