package core

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/thiago-felipe-99/mail/publisher/data"
	"github.com/thiago-felipe-99/mail/publisher/model"
	"github.com/thiago-felipe-99/mail/rabbit"
)

var (
	ErrInvalidName              = errors.New("was sent a invalid name")
	ErrQueueAlreadyExist        = errors.New("queue already exist")
	ErrQueueDoesNotExist        = errors.New("queue does not exist")
	ErrBodyValidate             = errors.New("unable to parse body")
	ErrTemplateNameAlreadyExist = errors.New("template name already exist")
	ErrMaxSizeTemplate          = errors.New("template has a max size of 1MB")
	ErrMissingFieldTemplates    = errors.New("missing fields from template")
	ErrTemplateDoesNotExist     = errors.New("template does not exist")
)

const maxSizeTemplate = 1024 * 1024

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

func validate(validate *validator.Validate, data any) error {
	err := validate.Struct(data)
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

func dlxName(name string) string {
	return name + "-dlx"
}

type Queue struct {
	template  *Template
	rabbit    *rabbit.Rabbit
	database  *data.Queue
	validator *validator.Validate
}

func (core *Queue) Create(partial model.QueuePartial) error {
	err := validate(core.validator, partial)
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

	err = core.database.Create(queue)
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
		return ErrQueueDoesNotExist
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

	err := validate(core.validator, email)
	if err != nil {
		return err
	}

	queueExist, err := core.database.Exist(queue)
	if err != nil {
		return fmt.Errorf("error checking queue: %w", err)
	}

	if !queueExist {
		return ErrQueueDoesNotExist
	}

	if email.Template != nil {
		exist, err := core.template.Exist(email.Template.Name)
		if err != nil {
			return fmt.Errorf("error checking if template exist: %w", err)
		}

		if !exist {
			return ErrTemplateDoesNotExist
		}

		fields, err := core.template.GetFields(email.Template.Name)
		if err != nil {
			return fmt.Errorf("error getting templates fields: %w", err)
		}

		for _, field := range fields {
			if _, found := email.Template.Data[field]; !found {
				return ErrMissingFieldTemplates
			}
		}
	}

	err = core.rabbit.SendMessage(context.Background(), queue, email)
	if err != nil {
		return fmt.Errorf("error sending email: %w", err)
	}

	email.ID = uuid.New()

	err = core.database.SaveEmail(email)
	if err != nil {
		return fmt.Errorf("error saving email: %w", err)
	}

	return nil
}

func NewQueue(
	template *Template,
	rabbit *rabbit.Rabbit,
	database *data.Queue,
	validate *validator.Validate,
) *Queue {
	return &Queue{
		template:  template,
		rabbit:    rabbit,
		database:  database,
		validator: validate,
	}
}

type Template struct {
	minio       *minio.Client
	bucket      string
	database    *data.Template
	validate    *validator.Validate
	regexFields *regexp.Regexp
}

func (core *Template) Create(partial model.TemplatePartial) error {
	err := validate(core.validate, partial)
	if err != nil {
		return err
	}

	if len(partial.Name) > maxSizeTemplate {
		return ErrMaxSizeTemplate
	}

	exist, err := core.database.Exist(partial.Name)
	if err != nil {
		return fmt.Errorf("error checking if template exist: %w", err)
	}

	if exist {
		return ErrTemplateNameAlreadyExist
	}

	fieldsRaw := core.regexFields.FindAllString(partial.Template, -1)
	fields := make([]string, 0, len(fieldsRaw))
	existField := func(fields []string, find string) bool {
		for _, field := range fields {
			if field == find {
				return true
			}
		}

		return false
	}

	for _, field := range fieldsRaw {
		field = strings.Trim(field, "{} ")

		if !existField(fields, field) {
			fields = append(fields, field)
		}
	}

	template := model.Template{
		ID:       uuid.New(),
		Name:     partial.Name,
		Template: partial.Template,
		Fields:   fields,
	}

	templateReader := strings.NewReader(template.Template)

	_, err = core.minio.PutObject(
		context.Background(),
		core.bucket,
		template.Name,
		templateReader,
		templateReader.Size(),
		minio.PutObjectOptions{
			ContentType: "text/markdown",
		},
	)
	if err != nil {
		return fmt.Errorf("error adding template on minio: %w", err)
	}

	err = core.database.Create(template)
	if err != nil {
		return fmt.Errorf("error adding template on database: %w", err)
	}

	return nil
}

func (core *Template) Exist(name string) (bool, error) {
	exist, err := core.database.Exist(name)
	if err != nil {
		return false, fmt.Errorf("error checking if template exist in database: %w", err)
	}

	return exist, nil
}

func (core *Template) GetFields(name string) ([]string, error) {
	template, err := core.database.Get(name)
	if err != nil {
		return nil, fmt.Errorf("error getting template: %w", err)
	}

	return template.Fields, nil
}

func (core *Template) GetAll() ([]model.Template, error) {
	templates, err := core.database.GetAll()
	if err != nil {
		return nil, fmt.Errorf("error getting templates from database: %w", err)
	}

	return templates, nil
}

func NewTemplate(
	database *data.Template,
	minio *minio.Client,
	bucket string,
	validate *validator.Validate,
) *Template {
	return &Template{
		database:    database,
		minio:       minio,
		bucket:      bucket,
		validate:    validate,
		regexFields: regexp.MustCompile(`{{ *(\w|\d)+ *}}`),
	}
}
