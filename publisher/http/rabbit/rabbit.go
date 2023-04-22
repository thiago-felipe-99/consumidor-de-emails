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
	ErrConnectionClosed = errors.New("closed connection with RabbitMQ")
	ErrEncondingMessage = errors.New("error enconding message")
	ErrSendingMessage   = errors.New("error sending message")
)

type ErrMaxRetries struct {
	errors []error
}

func (errMaxRetries *ErrMaxRetries) add(err error) {
	errMaxRetries.errors = append(errMaxRetries.errors, err)
}

func (errMaxRetries *ErrMaxRetries) Error() string {
	errMaxRetries.errors = append([]error{errors.New("error max retries")}, errMaxRetries.errors...)
	return errors.Join(errMaxRetries.errors...).Error()
}

type Config struct {
	User     string
	Password string
	Host     string
	Port     string
	Vhost    string
}

type Rabbit struct {
	done                  chan bool
	close                 bool
	maxPublishRetries     int
	maxCreateQueueRetries int
	connection            *amqp.Connection
	channel               *amqp.Channel
	notifyClose           chan amqp.Error
}

func (rabbit *Rabbit) Close() error {
	if rabbit.close {
		return errors.New("connection already closed")
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

func (rabbit *Rabbit) retries(
	ctx context.Context,
	maxRetries int,
	errsReturn []error,
	try func() error,
) error {
	resendDelay := time.Second
	errMaxRetries := &ErrMaxRetries{}

	for retries := 0; retries < maxRetries; retries++ {
		err := try()
		if err != nil {
			for _, errReturn := range errsReturn {
				if errors.Is(err, errReturn) {
					return err
				}
			}

			errMaxRetries.add(err)

			select {
			case <-rabbit.done:
				return ErrConnectionClosed
			case <-time.After(resendDelay):
				resendDelay *= 2
			}

			continue
		}

		return nil
	}

	return errMaxRetries
}

func (rabbit *Rabbit) CreateQueue(name string, maxRetries int) error {
	errsReturn := []error{}

	createQueue := func() error {
		return rabbit.createQueue(name, maxRetries)
	}

	return rabbit.retries(
		context.Background(),
		rabbit.maxCreateQueueRetries,
		errsReturn,
		createQueue,
	)
}

func (rabbit *Rabbit) createQueue(name string, maxRetries int) error {
	if rabbit.close {
		return ErrConnectionClosed
	}

	dlx := name + "-dlx"

	queueArgs := amqp.Table{}
	queueArgs["x-dead-letter-exchange"] = name + "-dlx"
	queueArgs["x-dead-letter-routing-key"] = "dead-message"
	queueArgs["x-delivery-limit"] = maxRetries + 1
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
	errsReturn := []error{ErrEncondingMessage}

	sendMessage := func() error {
		return rabbit.sendMessage(ctx, queue, message)
	}

	return rabbit.retries(ctx, rabbit.maxPublishRetries, errsReturn, sendMessage)
}

func (rabbit *Rabbit) sendMessage(ctx context.Context, queue string, message any) error {
	if rabbit.close {
		return ErrConnectionClosed
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	messageEncoding, err := json.Marshal(message)
	if err != nil {
		return errors.Join(ErrEncondingMessage, err)
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
		return errors.Join(ErrSendingMessage, err)
	}

	done, err := confirm.WaitContext(ctx)
	if err != nil {
		return err
	}

	if !done {
		return errors.New("timeout sending message")
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
		done:                  make(chan bool),
		close:                 false,
		maxPublishRetries:     5,
		maxCreateQueueRetries: 3,
		connection:            rabbit,
		channel:               channel,
		notifyClose:           make(chan amqp.Error),
	}, nil
}
