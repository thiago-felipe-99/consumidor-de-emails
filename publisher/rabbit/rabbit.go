package rabbit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

var (
	ErrAlreadyClosed    = errors.New("connection already closed")
	ErrConnectionClosed = errors.New("closed connection with RabbitMQ")
	ErrEncondingMessage = errors.New("error encoding message")
	ErrSendingMessage   = errors.New("error sending message")
	ErrTimeoutMessage   = errors.New("timeout sending message")
	ErrMaxRetries       = errors.New("error max retries")
)

type MaxRetriesError struct {
	errors []error
}

func (maxRetriesError *MaxRetriesError) add(err error) {
	maxRetriesError.errors = append(maxRetriesError.errors, err)
}

func (maxRetriesError *MaxRetriesError) Error() string {
	maxRetriesError.errors = append([]error{ErrMaxRetries}, maxRetriesError.errors...)

	return errors.Join(maxRetriesError.errors...).Error()
}

type Config struct {
	User     string
	Password string
	Host     string
	Port     string
	Vhost    string
}

type Rabbit struct {
	url                   string
	done                  chan bool
	close                 bool
	maxPublishRetries     int
	maxCreateQueueRetries int
	connection            *amqp.Connection
	channel               *amqp.Channel
	notifyConnectionClose chan *amqp.Error
	notifyChannelClose    chan *amqp.Error
	timeoutSendMessage    time.Duration
}

func (rabbit *Rabbit) Close() error {
	if rabbit.close {
		return ErrAlreadyClosed
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

func (rabbit *Rabbit) retries(maxRetries int, errsReturn []error, try func() error) error {
	resendDelay := time.Second
	errMaxRetries := &MaxRetriesError{}

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

	return rabbit.retries(rabbit.maxCreateQueueRetries, errsReturn, createQueue)
}

func (rabbit *Rabbit) createQueue(name string, maxRetries int) error {
	if rabbit.close {
		return ErrConnectionClosed
	}

	dlx := name + "-dlx"

	queueArgs := amqp.Table{
		"x-dead-letter-exchange":    name + "-dlx",
		"x-dead-letter-routing-key": "dead-message",
		"x-delivery-limit":          maxRetries,
		"x-queue-type":              "quorum",
	}

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

	return rabbit.retries(rabbit.maxPublishRetries, errsReturn, sendMessage)
}

func (rabbit *Rabbit) sendMessage(ctx context.Context, queue string, message any) error {
	if rabbit.close {
		return ErrConnectionClosed
	}

	ctx, cancel := context.WithTimeout(ctx, rabbit.timeoutSendMessage)
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
		return errors.Join(ErrTimeoutMessage, err)
	}

	if !done {
		return ErrTimeoutMessage
	}

	return nil
}

func (rabbit *Rabbit) HandleConnection() {
	recreatDelay := time.Second

	for {
		log.Println("[INFO] - Trying to connect with RabbitMQ")

		err := rabbit.createConnection()
		if err != nil {
			log.Printf("[ERROR] - Error creating RabbitMQ connection: %s", err)

			select {
			case <-rabbit.done:
				log.Println("[INFO] - Connection was terminated")

				return

			case <-time.After(recreatDelay):
				recreatDelay *= 2
			}

			continue
		}

		log.Println("[INFO] - Connection to RabbitMQ successfully established")

		recreatDelay = time.Second

		select {
		case <-rabbit.done:
			log.Println("[INFO] - Connection was terminated")

			return

		case <-rabbit.notifyConnectionClose:
			log.Println("[INFO] - Connection was closed, recreating connection")

		case <-rabbit.notifyChannelClose:
			log.Println("[INFO] - Channel was closed, recreating channel")
		}
	}
}

func (rabbit *Rabbit) createConnection() error {
	rabbit.close = true

	if rabbit.connection == nil || rabbit.connection.IsClosed() {
		connection, err := amqp.Dial(rabbit.url)
		if err != nil {
			return fmt.Errorf("error creating connection: %w", err)
		}

		rabbit.connection = connection
		rabbit.notifyConnectionClose = make(chan *amqp.Error, 1)
		rabbit.connection.NotifyClose(rabbit.notifyConnectionClose)
	}

	channel, err := rabbit.connection.Channel()
	if err != nil {
		rabbit.connection.Close()

		return fmt.Errorf("failed to open RabbitMQ channel: %w", err)
	}

	err = channel.Confirm(false)
	if err != nil {
		rabbit.connection.Close()

		return fmt.Errorf("failed to active channel confirm: %w", err)
	}

	rabbit.channel = channel
	rabbit.notifyChannelClose = make(chan *amqp.Error, 1)
	rabbit.channel.NotifyClose(rabbit.notifyChannelClose)

	rabbit.close = false

	return nil
}

func New(config Config) *Rabbit {
	url := fmt.Sprintf(
		"amqp://%s:%s@%s:%s/%s",
		config.User,
		config.Password,
		config.Host,
		config.Port,
		config.Vhost,
	)

	//nolint: gomnd
	rabbit := &Rabbit{
		url:                   url,
		done:                  make(chan bool),
		close:                 true,
		maxPublishRetries:     5,
		maxCreateQueueRetries: 3,
		timeoutSendMessage:    5 * time.Second,
	}

	return rabbit
}
