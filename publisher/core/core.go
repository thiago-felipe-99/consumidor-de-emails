package core

import (
	"errors"
	"time"

	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	"github.com/minio/minio-go/v7"
	"github.com/thiago-felipe-99/mail/publisher/data"
	"github.com/thiago-felipe-99/mail/rabbit"
)

var (
	ErrInvalidName                   = errors.New("was sent a invalid name")
	ErrUserAlreadyExist              = errors.New("user already exist")
	ErrUserDoesNotExist              = errors.New("user does not exist")
	ErrUserWrongPassword             = errors.New("was sent a wrong password")
	ErrUserSessionDoesNotExist       = errors.New("user session does not exist")
	ErrUserSessionDeleted            = errors.New("user session deleted")
	ErrUserIsNotAdmin                = errors.New("user is not admin")
	ErrUserIsProtected               = errors.New("user is protected")
	ErrQueueAlreadyExist             = errors.New("queue already exist")
	ErrQueueDoesNotExist             = errors.New("queue does not exist")
	ErrBodyValidate                  = errors.New("unable to parse body")
	ErrTemplateNameAlreadyExist      = errors.New("template name already exist")
	ErrMaxSizeTemplate               = errors.New("template has a max size of 1MB")
	ErrMissingFieldTemplates         = errors.New("missing fields from template")
	ErrTemplateDoesNotExist          = errors.New("template does not exist")
	ErrAttachmentDoesNotExist        = errors.New("attachment does not exist")
	ErrAttachmentDoesNotExistOnMinio = errors.New("attachment does not exist on minio")
	ErrMaxSizeAttachment             = errors.New("attachment has a max size")
	ErrUploadAlreadyConfirmed        = errors.New("upload already confirmed")
	ErrEmailListAlreadyExist         = errors.New("email list already exist")
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

type Cores struct {
	*User
	*Queue
	*Template
	*Attachment
}

func NewCores(
	databases *data.Databases,
	validate *validator.Validate,
	sessionDuration time.Duration,
	rabbit *rabbit.Rabbit,
	minio *minio.Client,
	bukcetTemplate string,
	bukcetAttachment string,
	maxEntrySize int,
) *Cores {
	template := newTemplate(databases.Template, minio, bukcetTemplate, validate)
	attachment := newAttachment(
		minio,
		bukcetAttachment,
		databases.Attachment,
		validate,
		maxEntrySize,
	)

	return &Cores{
		User:       newUser(databases.User, validate, sessionDuration),
		Template:   template,
		Queue:      newQueue(template, attachment, rabbit, databases.Queue, validate),
		Attachment: attachment,
	}
}
