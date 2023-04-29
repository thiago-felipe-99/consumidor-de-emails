package main

import (
	"errors"
	"fmt"

	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	"github.com/thiago-felipe-99/mail/rabbit"
)

func dlxName(name string) string {
	return name + "-dlx"
}

type queueCore struct {
	rabbit   *rabbit.Rabbit
	validate *validator.Validate
	database *database
}

type modelInvalidError struct {
	invalid validator.ValidationErrors
}

func (err modelInvalidError) Error() string {
	return err.invalid.Error()
}

func (err modelInvalidError) Translate(language ut.Translator) string {
	messages := err.invalid.Translate(language)

	messageSend := ""
	for _, message := range messages {
		messageSend += "\n" + message
	}

	return messageSend
}

func (core *queueCore) create(queue queueBody) error {
	err := core.validate.Struct(queue)
	if err != nil {
		validationErrs := validator.ValidationErrors{}

		okay := errors.As(err, &validationErrs)
		if !okay {
			return errBodyValidate
		}

		return &modelInvalidError{validationErrs}
	}

	name, dlx := queue.Name, dlxName(queue.Name)

	queueExist, err := core.database.existQueue(name)
	if err != nil {
		return fmt.Errorf("error checking queue: %w", err)
	}

	dlxExist, err := core.database.existQueue(dlx)
	if err != nil {
		return fmt.Errorf("error checking queue: %w", err)
	}

	if queueExist || dlxExist {
		return errQueueAlreadyExist
	}

	err = core.rabbit.CreateQueueWithDLX(name, dlx, queue.MaxRetries)
	if err != nil {
		return fmt.Errorf("error creating queue: %w", err)
	}

	err = core.database.addQueue(name, dlx, queue.MaxRetries)
	if err != nil {
		return fmt.Errorf("error adding queue on database: %w", err)
	}

	return nil
}

func (core *queueCore) getAll() ([]queueModel, error) {
	queues, err := core.database.getQueues()
	if err != nil {
		return nil, fmt.Errorf("error getting all queues: %w", err)
	}

	return queues, nil
}
