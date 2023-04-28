//nolint:wrapcheck
package main

import (
	"context"
	"errors"
	"log"

	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/thiago-felipe-99/mail/rabbit"
)

var (
	errQueueAlreadyExist = errors.New("queue already exist")
	errQueueDontExist    = errors.New("queue dont exist")
	errBodyValidate      = errors.New("unable to parse body")
)

type sent struct {
	Message string `json:"message" bson:"message"`
}

type receiver struct {
	Name  string `json:"name"  bson:"name"  validate:"required"`
	Email string `json:"email" bson:"email" validate:"required,email"`
}

type template struct {
	Name string            `json:"name" bson:"name" validate:"required"`
	Data map[string]string `json:"data" bson:"data"`
}

//nolint:lll
type email struct {
	ID             uuid.UUID  `json:"-"              bson:"_id"`
	Receivers      []receiver `json:"receivers"      bson:"receivers"       validate:"required_without=BlindReceivers"`
	BlindReceivers []receiver `json:"blindReceivers" bson:"blind_receivers" validate:"required_without=Receivers"`
	Subject        string     `json:"subject"        bson:"subject"         validate:"required"`
	Message        string     `json:"message"        bson:"message"         validate:"required_without=Template,excluded_with=Template"`
	Template       *template  `json:"template"       bson:"template"        validate:"required_without=Message,excluded_with=Message"`
	Attachments    []string   `json:"attachments"    bson:"attachments"`
}

type queue struct {
	rabbit     *rabbit.Rabbit
	validate   *validator.Validate
	translator *ut.UniversalTranslator
	languages  []string
	database   *database
}

func dlxName(name string) string {
	return name + "-dlx"
}

func (queue *queue) bodyParser(body any, handler *fiber.Ctx) error {
	err := handler.BodyParser(body)
	if err != nil {
		return errBodyValidate
	}

	err = queue.validate.Struct(body)
	if err != nil {
		validationErrs := validator.ValidationErrors{}

		okay := errors.As(err, &validationErrs)
		if !okay {
			return errBodyValidate
		}

		accept := handler.AcceptsLanguages(queue.languages...)
		if accept == "" {
			accept = queue.languages[0]
		}

		language, _ := queue.translator.GetTranslator(accept)

		messages := validationErrs.Translate(language)

		messageSend := ""
		for _, message := range messages {
			messageSend += "\n" + message
		}

		return errors.New(messageSend) //nolint: goerr113
	}

	return nil
}

type queueBody struct {
	Name       string `json:"name"       validate:"required"`
	MaxRetries int64  `json:"maxRetries"`
}

// Creating a RabbitMQ queue with DLX
//
// @Summary		Creating queue
// @Tags			queue
// @Accept			json
// @Produce		json
// @Success		200		{object}	sent "create queue successfully"
// @Failure		400		{object}	sent "an invalid queue param was sent"
// @Failure		409		{object}	sent "queue already exist"
// @Failure		500		{object}	sent "internal server error"
// @Param			queue	body		queue	true	"queue params"
// @Router			/email/queue [post]
// @Description	Creating a RabbitMQ queue with DLX.
func (queue *queue) create() func(*fiber.Ctx) error {
	return func(handler *fiber.Ctx) error {
		body := &queueBody{
			MaxRetries: 10, //nolint:gomnd
		}

		err := queue.bodyParser(body, handler)
		if err != nil {
			return handler.Status(fiber.StatusBadRequest).JSON(sent{err.Error()})
		}

		name, dlx := body.Name, dlxName(body.Name)

		queueExist, err := queue.database.existQueue(name)
		if err != nil {
			log.Printf("[ERROR] - Error checking queue: %s", err)

			return handler.Status(fiber.StatusInternalServerError).
				JSON(sent{"error checking queue"})
		}

		dlxExist, err := queue.database.existQueue(dlx)
		if err != nil {
			log.Printf("[ERROR] - Error checking queue: %s", err)

			return handler.Status(fiber.StatusInternalServerError).
				JSON(sent{"error checking queue"})
		}

		if queueExist || dlxExist {
			return handler.Status(fiber.StatusConflict).JSON(sent{errQueueAlreadyExist.Error()})
		}

		err = queue.rabbit.CreateQueueWithDLX(name, dlx, body.MaxRetries)
		if err != nil {
			log.Printf("[ERROR] - Error creating queue: %s", err)

			return handler.Status(fiber.StatusInternalServerError).
				JSON(sent{"error creating queue"})
		}

		err = queue.database.addQueue(name, dlx, body.MaxRetries)
		if err != nil {
			log.Printf("[ERROR] - Error adding queue on database: %s", err)

			return handler.Status(fiber.StatusInternalServerError).
				JSON(sent{"error creating queue"})
		}

		return handler.Status(fiber.StatusCreated).JSON(sent{"queue created"})
	}
}

// Getting all RabbitMQ queues
//
// @Summary		Get queues
// @Tags			queue
// @Accept			json
// @Produce		json
// @Success		200		{array}	queueData "all queues"
// @Failure		500		{object}	sent "internal server error"
// @Router			/email/queue [get]
// @Description	Getting all RabbitMQ queues.
func (queue *queue) getAll() func(*fiber.Ctx) error {
	return func(handler *fiber.Ctx) error {
		queues, err := queue.database.getQueues()
		if err != nil {
			log.Printf("[ERROR] - Error getting all queues: %s", err)

			return handler.Status(fiber.StatusInternalServerError).
				JSON(sent{"error getting queue"})
		}

		return handler.JSON(queues)
	}
}

// Delete a queue with DLX
//
// @Summary		Delete queues
// @Tags			queue
// @Accept			json
// @Produce		json
// @Success		204		{array}	queueData "queue deleted"
// @Failure		404		{object}	sent "queue dont exist"
// @Failure		500		{object}	sent "internal server error"
// @Router			/email/queue/{name} [delete]
// @Description	Delete a queue with DLX.
func (queue *queue) delete() func(*fiber.Ctx) error {
	return func(handler *fiber.Ctx) error {
		name := handler.Params("name")

		exist, err := queue.database.existQueue(name)
		if err != nil {
			log.Printf("[ERROR] - Error checking queue: %s", err)

			return handler.Status(fiber.StatusInternalServerError).
				JSON(sent{"error checking queue"})
		}

		if !exist {
			return handler.Status(fiber.StatusNotFound).JSON(sent{errQueueDontExist.Error()})
		}

		err = queue.rabbit.DeleteQueueWithDLX(name, dlxName(name))
		if err != nil {
			log.Printf("[ERROR] - Error deleting queue from RabbitMQ: %s", err)

			return handler.Status(fiber.StatusInternalServerError).
				JSON(sent{"error deleting queue from RabbitMQ"})
		}

		err = queue.database.deleteQueue(name)
		if err != nil {
			log.Printf("[ERROR] - Error deleting queue: %s", err)

			return handler.Status(fiber.StatusInternalServerError).
				JSON(sent{"error deleting queue"})
		}

		return handler.JSON(sent{"queue deleted"})
	}
}

// Sends an email to the RabbitMQ queue
//
// @Summary		Sends email
// @Tags			queue
// @Accept			json
// @Produce		json
// @Success		200		{object}	sent "email sent successfully"
// @Failure		400		{object}	sent "an invalid email param was sent"
// @Failure		404		{object}	sent "queue does not exist"
// @Failure		500		{object}	sent "internal server error"
// @Param			name	path	string		true	"queue name"
// @Param			queue	body	email	true	"email"
// @Router			/email/queue/{name}/send [post]
// @Description	Sends an email to the RabbitMQ queue.
func (queue *queue) send() func(*fiber.Ctx) error {
	return func(handler *fiber.Ctx) error {
		name := handler.Params("name")

		queueExist, err := queue.database.existQueue(name)
		if err != nil {
			log.Printf("[ERROR] - Error checking queue: %s", err)

			return handler.Status(fiber.StatusInternalServerError).
				JSON(sent{"error verifying queue"})
		}

		if !queueExist {
			return handler.Status(fiber.StatusNotFound).JSON(sent{errQueueDontExist.Error()})
		}

		body := &email{}

		err = queue.bodyParser(body, handler)
		if err != nil {
			return handler.Status(fiber.StatusBadRequest).JSON(sent{err.Error()})
		}

		err = queue.rabbit.SendMessage(context.Background(), name, body)
		if err != nil {
			log.Printf("[ERROR] - Error sending email: %s", err)

			return handler.Status(fiber.StatusInternalServerError).
				JSON(sent{"error send email"})
		}

		err = queue.database.saveEmail(*body)
		if err != nil {
			log.Printf("[ERROR] - Error saving email: %s", err)

			return handler.Status(fiber.StatusInternalServerError).
				JSON(sent{"error savving email"})
		}

		return handler.JSON(sent{"email sent"})
	}
}
