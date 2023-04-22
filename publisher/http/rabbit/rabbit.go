package rabbit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

var (
	ErrConnectionClose  = errors.New("closed connection with RabbitMQ")
	ErrEncondingMessage = errors.New("error enconding message")
	ErrSendingMessage   = errors.New("error sending message")
	ErrMaxRetries       = errors.New("error max retries")
)

type Config struct {
	User     string
	Password string
	Host     string
	Port     string
	Vhost    string
}

type Rabbit struct {
	done              chan bool
	close             bool
	maxPublishRetries int
	connection        *amqp.Connection
	channel           *amqp.Channel
	notifyClose       chan amqp.Error
}

func (rabbit *Rabbit) Close() error {
	if rabbit.close {
		return errors.New("connection already close")
	}

	close(rabbit.done)

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
	if rabbit.close {
		return ErrConnectionClose
	}

	resendDelay := time.Second

	for retries := 0; retries < rabbit.maxPublishRetries; retries++ {
		chanConfirm, err := rabbit.sendMessage(ctx, queue, message)
		if err != nil {
			if errors.Is(err, ErrEncondingMessage) {
				return err
			}

			select {
			case <-rabbit.done:
				return ErrConnectionClose
			case <-time.After(resendDelay):
				resendDelay *= 2
			}

			continue
		}

		ctx, cancel := context.WithTimeout(ctx, resendDelay)
		defer cancel()

		done, err := chanConfirm.WaitContext(ctx)
		if err != nil {
			continue
		}

		if done {
			return nil
		}
	}

	return ErrMaxRetries
}

func (rabbit *Rabbit) sendMessage(
	ctx context.Context,
	queue string,
	message any,
) (*amqp.DeferredConfirmation, error) {
	if rabbit.close {
		return nil, ErrConnectionClose
	}

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	messageEncoding, err := json.Marshal(message)
	if err != nil {
		return nil, errors.Join(ErrEncondingMessage, err)
	}

	publish := amqp.Publishing{
		ContentType: "application/json",
		Body:        messageEncoding,
	}

	confirm, err := rabbit.channel.PublishWithDeferredConfirmWithContext(
		ctx,
		"",
		queue,
		false,
		false,
		publish,
	)
	if err != nil {
		return nil, errors.Join(ErrSendingMessage, err)
	}

	return confirm, err
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
		done:              make(chan bool),
		close:             false,
		maxPublishRetries: 5,
		connection:        rabbit,
		channel:           channel,
		notifyClose:       make(chan amqp.Error),
	}, nil
}
