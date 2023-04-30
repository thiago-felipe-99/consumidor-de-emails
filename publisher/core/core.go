package core

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/alexedwards/argon2id"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/thiago-felipe-99/mail/publisher/data"
	"github.com/thiago-felipe-99/mail/publisher/model"
	"github.com/thiago-felipe-99/mail/rabbit"
)

var (
	ErrUserAlreadyExist         = errors.New("user already exist")
	ErrUserDoesNotExist         = errors.New("user does not exist")
	ErrUserSessionDoesNotExist  = errors.New("user session does not exist")
	ErrDifferentPassword        = errors.New("was sent a different password")
	ErrInvalidName              = errors.New("was sent a invalid name")
	ErrQueueAlreadyExist        = errors.New("queue already exist")
	ErrQueueDoesNotExist        = errors.New("queue does not exist")
	ErrBodyValidate             = errors.New("unable to parse body")
	ErrTemplateNameAlreadyExist = errors.New("template name already exist")
	ErrMaxSizeTemplate          = errors.New("template has a max size of 1MB")
	ErrMissingFieldTemplates    = errors.New("missing fields from template")
	ErrTemplateDoesNotExist     = errors.New("template does not exist")
)

const (
	argon2idParamMemory      = 128 * 1024
	argon2idParamIterations  = 2
	argon2idParamSaltLength  = 32
	argon2idParamKeyLength   = 64
	argon2idParamParallelism = 4
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

type User struct {
	database        *data.User
	validator       *validator.Validate
	argon2id        argon2id.Params
	durationSession time.Duration
}

func (core *User) Create(user model.User) error {
	err := validate(core.validator, user)
	if err != nil {
		return err
	}

	user.ID = uuid.New()

	exist, err := core.database.Exist(user.Name, user.Email)
	if err != nil {
		return fmt.Errorf("error checking if user exist in database: %w", err)
	}

	if exist {
		return ErrUserAlreadyExist
	}

	hash, err := argon2id.CreateHash(user.Password, &core.argon2id)
	if err != nil {
		return fmt.Errorf("error creating password hash: %w", err)
	}

	user.Password = hash

	err = core.database.Create(user)
	if err != nil {
		return fmt.Errorf("error creating user in database: %w", err)
	}

	return nil
}

func (core *User) NewSession(partial model.UserPartial) (*model.UserSession, error) {
	err := validate(core.validator, partial)
	if err != nil {
		return nil, err
	}

	exist, err := core.database.Exist(partial.Name, partial.Email)
	if err != nil {
		return nil, fmt.Errorf("error checking if user exist in database: %w", err)
	}

	if !exist {
		return nil, ErrUserDoesNotExist
	}

	user, err := core.database.GetByNameOrEmail(partial.Name, partial.Email)
	if err != nil {
		return nil, fmt.Errorf("error getting user in database: %w", err)
	}

	equals, err := argon2id.ComparePasswordAndHash(partial.Password, user.Password)
	if err != nil {
		return nil, fmt.Errorf("error comparing password with hash: %w", err)
	}

	if !equals {
		return nil, ErrDifferentPassword
	}

	session := model.UserSession{
		ID:         uuid.New(),
		UserID:     user.ID,
		CreateadAt: time.Now(),
		Expires:    time.Now().Add(core.durationSession),
		DeletedAt:  time.Now().Add(core.durationSession),
	}

	err = core.database.SaveSession(session)
	if err != nil {
		return nil, fmt.Errorf("error saving session in database: %w", err)
	}

	return &session, nil
}

func (core *User) RefreshSession(sessionID string) (*model.UserSession, error) {
	exist, err := core.database.ExistSession(sessionID)
	if err != nil {
		return nil, fmt.Errorf("error checking if session exist in database: %w", err)
	}

	if !exist {
		return nil, ErrUserSessionDoesNotExist
	}

	currentSession, err := core.database.GetSession(sessionID)
	if err != nil {
		return nil, fmt.Errorf("error getting session from database: %w", err)
	}

	if currentSession.DeletedAt.Before(time.Now()) {
		return nil, ErrUserSessionDoesNotExist
	}

	currentSession.DeletedAt = time.Now()

	exist, err = core.database.ExistByID(currentSession.UserID)
	if err != nil {
		return nil, fmt.Errorf("error checking if user exist in database: %w", err)
	}

	if !exist {
		return nil, ErrUserSessionDoesNotExist
	}

	err = core.database.UpdateSession(*currentSession)
	if err != nil {
		return nil, fmt.Errorf("error updating session in database: %w", err)
	}

	newSession := model.UserSession{
		ID:         uuid.New(),
		UserID:     currentSession.UserID,
		CreateadAt: time.Now(),
		Expires:    time.Now().Add(core.durationSession),
		DeletedAt:  time.Now().Add(core.durationSession),
	}

	err = core.database.SaveSession(newSession)
	if err != nil {
		return nil, fmt.Errorf("error saving session in database: %w", err)
	}

	return &newSession, nil
}

func NewUser(
	database *data.User,
	validate *validator.Validate,
	durationSession time.Duration,
) *User {
	return &User{
		database:  database,
		validator: validate,
		argon2id: argon2id.Params{
			Memory:      argon2idParamMemory,
			Iterations:  argon2idParamIterations,
			Parallelism: argon2idParamParallelism,
			SaltLength:  argon2idParamSaltLength,
			KeyLength:   argon2idParamKeyLength,
		},
		durationSession: durationSession,
	}
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
		return fmt.Errorf("error checking if queue exist in database: %w", err)
	}

	dlxExist, err := core.database.Exist(queue.Name)
	if err != nil {
		return fmt.Errorf("error checking if dlx queue exist in database: %w", err)
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
		return fmt.Errorf("error creating queue in database: %w", err)
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
		return fmt.Errorf("error checking if queue exist: %w", err)
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
		return fmt.Errorf("error checking if queue exist: %w", err)
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
		return fmt.Errorf("error saving email in database: %w", err)
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

func (core *Template) getFields(template string) []string {
	fieldsRaw := core.regexFields.FindAllString(template, -1)
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

	return fields
}

func (core *Template) Create(partial model.TemplatePartial) error {
	err := validate(core.validate, partial)
	if err != nil {
		return err
	}

	if len(partial.Template) > maxSizeTemplate {
		return ErrMaxSizeTemplate
	}

	exist, err := core.database.Exist(partial.Name)
	if err != nil {
		return fmt.Errorf("error checking if template exist: %w", err)
	}

	if exist {
		return ErrTemplateNameAlreadyExist
	}

	template := model.Template{
		ID:       uuid.New(),
		Name:     partial.Name,
		Template: partial.Template,
		Fields:   core.getFields(partial.Template),
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
		return fmt.Errorf("error creating template in Minio: %w", err)
	}

	err = core.database.Create(template)
	if err != nil {
		return fmt.Errorf("error creating template in database: %w", err)
	}

	return nil
}

func (core *Template) Update(name string, partial model.TemplatePartial) error {
	err := validate(core.validate, partial)
	if err != nil {
		return err
	}

	if len(partial.Template) > maxSizeTemplate {
		return ErrMaxSizeTemplate
	}

	exist, err := core.database.Exist(name)
	if err != nil {
		return fmt.Errorf("error checking if template exist: %w", err)
	}

	if !exist {
		return ErrTemplateDoesNotExist
	}

	templateID, err := core.database.GetID(name)
	if err != nil {
		return fmt.Errorf("error getting template ID: %w", err)
	}

	template := model.Template{
		ID:       templateID,
		Name:     name,
		Template: partial.Template,
		Fields:   core.getFields(partial.Template),
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
		return fmt.Errorf("error updating template in Minio: %w", err)
	}

	err = core.database.Update(template)
	if err != nil {
		return fmt.Errorf("error updating template in database: %w", err)
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
		return nil, fmt.Errorf("error getting template from database: %w", err)
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

func (core *Template) Get(name string) (*model.Template, error) {
	if len(name) == 0 {
		return nil, ErrInvalidName
	}

	exist, err := core.database.Exist(name)
	if err != nil {
		return nil, fmt.Errorf("error checking if template exist: %w", err)
	}

	if !exist {
		return nil, ErrTemplateDoesNotExist
	}

	template, err := core.database.Get(name)
	if err != nil {
		return nil, fmt.Errorf("error getting template from database: %w", err)
	}

	return template, nil
}

func (core *Template) Delete(name string) error {
	if len(name) == 0 {
		return ErrInvalidName
	}

	exist, err := core.database.Exist(name)
	if err != nil {
		return fmt.Errorf("error checking if template exist: %w", err)
	}

	if !exist {
		return ErrTemplateDoesNotExist
	}

	err = core.minio.RemoveObject(
		context.Background(),
		core.bucket,
		name,
		minio.RemoveObjectOptions{},
	)
	if err != nil {
		return fmt.Errorf("error deleting template from Minio: %w", err)
	}

	err = core.database.Delete(name)
	if err != nil {
		return fmt.Errorf("error deleting template from database: %w", err)
	}

	return nil
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
