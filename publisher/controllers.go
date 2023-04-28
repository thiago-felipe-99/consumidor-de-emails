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

type queueController struct {
	rabbit     *rabbit.Rabbit
	validate   *validator.Validate
	database   *database
	translator *ut.UniversalTranslator
	languages  []string
	core       *queueCore
}

func (controller *queueController) bodyParser(body any, handler *fiber.Ctx) error {
	err := handler.BodyParser(body)
	if err != nil {
		return errBodyValidate
	}

	err = controller.validate.Struct(body)
	if err != nil {
		validationErrs := validator.ValidationErrors{}

		okay := errors.As(err, &validationErrs)
		if !okay {
			return errBodyValidate
		}

		accept := handler.AcceptsLanguages(controller.languages...)
		if accept == "" {
			accept = controller.languages[0]
		}

		language, _ := controller.translator.GetTranslator(accept)

		messages := validationErrs.Translate(language)

		messageSend := ""
		for _, message := range messages {
			messageSend += "\n" + message
		}

		return errors.New(messageSend) //nolint: goerr113
	}

	return nil
}

type queueModel struct {
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
func (controller *queueController) create() func(*fiber.Ctx) error {
	return func(handler *fiber.Ctx) error {
		body := &queueModel{
			MaxRetries: 10, //nolint:gomnd
		}

		err := handler.BodyParser(body)
		if err != nil {
			return handler.Status(fiber.StatusBadRequest).JSON(sent{err.Error()})
		}

		err = controller.core.create(*body)
		if err != nil {
			if errors.Is(err, errQueueAlreadyExist) {
				return handler.Status(fiber.StatusConflict).JSON(sent{errQueueAlreadyExist.Error()})
			}

			modelInvalid := modelInvalidError{}
			if okay := errors.As(err, &modelInvalid); okay {
				accept := handler.AcceptsLanguages(controller.languages...)
				if accept == "" {
					accept = controller.languages[0]
				}

				language, _ := controller.translator.GetTranslator(accept)

				return handler.Status(fiber.StatusBadRequest).
					JSON(sent{modelInvalid.Translate(language)})
			}

			log.Printf("[ERROR] - Error creating queue: %s", err)

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
func (controller *queueController) getAll() func(*fiber.Ctx) error {
	return func(handler *fiber.Ctx) error {
		queues, err := controller.database.getQueues()
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
func (controller *queueController) delete() func(*fiber.Ctx) error {
	return func(handler *fiber.Ctx) error {
		name := handler.Params("name")

		exist, err := controller.database.existQueue(name)
		if err != nil {
			log.Printf("[ERROR] - Error checking queue: %s", err)

			return handler.Status(fiber.StatusInternalServerError).
				JSON(sent{"error checking queue"})
		}

		if !exist {
			return handler.Status(fiber.StatusNotFound).JSON(sent{errQueueDontExist.Error()})
		}

		err = controller.rabbit.DeleteQueueWithDLX(name, dlxName(name))
		if err != nil {
			log.Printf("[ERROR] - Error deleting queue from RabbitMQ: %s", err)

			return handler.Status(fiber.StatusInternalServerError).
				JSON(sent{"error deleting queue from RabbitMQ"})
		}

		err = controller.database.deleteQueue(name)
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
func (controller *queueController) send() func(*fiber.Ctx) error {
	return func(handler *fiber.Ctx) error {
		name := handler.Params("name")

		queueExist, err := controller.database.existQueue(name)
		if err != nil {
			log.Printf("[ERROR] - Error checking queue: %s", err)

			return handler.Status(fiber.StatusInternalServerError).
				JSON(sent{"error verifying queue"})
		}

		if !queueExist {
			return handler.Status(fiber.StatusNotFound).JSON(sent{errQueueDontExist.Error()})
		}

		body := &email{}

		err = controller.bodyParser(body, handler)
		if err != nil {
			return handler.Status(fiber.StatusBadRequest).JSON(sent{err.Error()})
		}

		err = controller.rabbit.SendMessage(context.Background(), name, body)
		if err != nil {
			log.Printf("[ERROR] - Error sending email: %s", err)

			return handler.Status(fiber.StatusInternalServerError).
				JSON(sent{"error send email"})
		}

		err = controller.database.saveEmail(*body)
		if err != nil {
			log.Printf("[ERROR] - Error saving email: %s", err)

			return handler.Status(fiber.StatusInternalServerError).
				JSON(sent{"error savving email"})
		}

		return handler.JSON(sent{"email sent"})
	}
}
