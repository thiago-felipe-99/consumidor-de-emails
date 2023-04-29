package core

import (
	"context"
	"errors"
	"fmt"
	"time"

	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/thiago-felipe-99/mail/publisher/data"
	"github.com/thiago-felipe-99/mail/publisher/model"
	"github.com/thiago-felipe-99/mail/rabbit"
)

var (
	ErrInvalidName              = errors.New("was sent a invalid name")
	ErrQueueAlreadyExist        = errors.New("queue already exist")
	ErrQueueDontExist           = errors.New("queue does not exist")
	ErrBodyValidate             = errors.New("unable to parse body")
	ErrTemplateNameAlreadyExist = errors.New("template name already exist")
)

func dlxName(name string) string {
	return name + "-dlx"
}

type Queue struct {
	rabbit    *rabbit.Rabbit
	database  *data.Queue
	validator *validator.Validate
}

type ModelInvalidError struct {
	invalid validator.ValidationErrors
}

func (err ModelInvalidError) Error() string {
	return err.invalid.Error()
}

func (err ModelInvalidError) Translate(language ut.Translator) string {
	messages := err.invalid.Translate(language)

	messageSend := ""
	for _, message := range messages {
		messageSend += ", " + message
	}

	return messageSend[2:]
}

func (core *Queue) validate(data any) error {
	err := core.validator.Struct(data)
	if err != nil {
		validationErrs := validator.ValidationErrors{}

		okay := errors.As(err, &validationErrs)
		if !okay {
			return ErrBodyValidate
		}

		return ModelInvalidError{validationErrs}
	}

	return nil
}

func (core *Queue) Create(partial model.QueuePartial) error {
	err := core.validate(partial)
	if err != nil {
		return err
	}

	queue := model.Queue{
		ID:         uuid.New(),
		Name:       partial.Name,
		DLX:        dlxName(partial.Name),
		MaxRetries: partial.MaxRetries,
		CreatedAt:  time.Now(),
	}

	queueExist, err := core.database.Exist(queue.Name)
	if err != nil {
		return fmt.Errorf("error checking queue: %w", err)
	}

	dlxExist, err := core.database.Exist(queue.Name)
	if err != nil {
		return fmt.Errorf("error checking queue: %w", err)
	}

	if queueExist || dlxExist {
		return ErrQueueAlreadyExist
	}

	err = core.rabbit.CreateQueueWithDLX(queue.Name, queue.DLX, queue.MaxRetries)
	if err != nil {
		return fmt.Errorf("error creating queue: %w", err)
	}

	err = core.database.Add(queue)
	if err != nil {
		return fmt.Errorf("error adding queue on database: %w", err)
	}

	return nil
}

func (core *Queue) GetAll() ([]model.Queue, error) {
	queues, err := core.database.GetAll()
	if err != nil {
		return nil, fmt.Errorf("error getting all queues: %w", err)
	}

	return queues, nil
}

func (core *Queue) Delete(name string) error {
	if len(name) == 0 {
		return ErrInvalidName
	}

	exist, err := core.database.Exist(name)
	if err != nil {
		return fmt.Errorf("error checking queue: %w", err)
	}

	if !exist {
		return ErrQueueDontExist
	}

	err = core.rabbit.DeleteQueueWithDLX(name, dlxName(name))
	if err != nil {
		return fmt.Errorf("error deleting queue from RabbitMQ: %w", err)
	}

	err = core.database.Delete(name)
	if err != nil {
		return fmt.Errorf("error deleting queue from database: %w", err)
	}

	return nil
}

func (core *Queue) SendEmail(queue string, email model.Email) error {
	if len(queue) == 0 {
		return ErrInvalidName
	}

	err := core.validate(email)
	if err != nil {
		return err
	}

	queueExist, err := core.database.Exist(queue)
	if err != nil {
		return fmt.Errorf("error checking queue: %w", err)
	}

	if !queueExist {
		return ErrQueueDontExist
	}

	err = core.rabbit.SendMessage(context.Background(), queue, email)
	if err != nil {
		return fmt.Errorf("error sending email: %w", err)
	}

	err = core.database.SaveEmail(email)
	if err != nil {
		return fmt.Errorf("error saving email: %w", err)
	}

	return nil
}

func NewQueue(rabbit *rabbit.Rabbit, database *data.Queue, validate *validator.Validate) *Queue {
	return &Queue{
		rabbit:    rabbit,
		validator: validate,
		database:  database,
	}
}

type Template struct {
	// minio
	database  *data.Template
	validator *validator.Validate
}

func (core *Template) Create(_ model.TemplatePartial) error {
	return nil
}

func NewTemplate(database *data.Template, validate *validator.Validate) *Template {
	return &Template{
		database:  database,
		validator: validate,
	}
}
