//nolint:wrapcheck
package main

import (
	"errors"
	"log"

	ut "github.com/go-playground/universal-translator"
	"github.com/gofiber/fiber/v2"
)

var (
	errQueueAlreadyExist = errors.New("queue already exist")
	errQueueDontExist    = errors.New("queue dont exist")
	errBodyValidate      = errors.New("unable to parse body")
)

type sent struct {
	Message string `json:"message" bson:"message"`
}

type queueController struct {
	translator *ut.UniversalTranslator
	languages  []string
	core       *queueCore
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
// @Param			queue	body		queueBody	true	"queue params"
// @Router			/email/queue [post]
// @Description	Creating a RabbitMQ queue with DLX.
func (controller *queueController) create(handler *fiber.Ctx) error {
	body := &queueBody{
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

// Getting all RabbitMQ queues
//
// @Summary		Get queues
// @Tags			queue
// @Accept			json
// @Produce		json
// @Success		200		{array}	queueModel "all queues"
// @Failure		500		{object}	sent "internal server error"
// @Router			/email/queue [get]
// @Description	Getting all RabbitMQ queues.
func (controller *queueController) getAll(handler *fiber.Ctx) error {
	queues, err := controller.core.getAll()
	if err != nil {
		log.Printf("[ERROR] - Error getting all queues: %s", err)

		return handler.Status(fiber.StatusInternalServerError).
			JSON(sent{"error getting queue"})
	}

	return handler.JSON(queues)
}

// Delete a queue with DLX
//
// @Summary		Delete queues
// @Tags			queue
// @Accept			json
// @Produce		json
// @Success		204		{onject}	sent "queue deleted"
// @Failure		404		{object}	sent "queue dont exist"
// @Failure		500		{object}	sent "internal server error"
// @Router			/email/queue/{name} [delete]
// @Description	Delete a queue with DLX.
func (controller *queueController) delete(handler *fiber.Ctx) error {
	err := controller.core.delete(handler.Params("name"))
	if err != nil {
		if errors.Is(err, errQueueDontExist) {
			return handler.Status(fiber.StatusNotFound).JSON(sent{errQueueDontExist.Error()})
		}

		log.Printf("[ERROR] - Error deleting queue: %s", err)

		return handler.Status(fiber.StatusInternalServerError).
			JSON(sent{"error deletring queue"})
	}

	return handler.JSON(sent{"queue deleted"})
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
// @Param			queue	body	emailModel	true	"email"
// @Router			/email/queue/{name}/sendEmail [post]
// @Description	Sends an email to the RabbitMQ queue.
func (controller *queueController) sendEmail(handler *fiber.Ctx) error {
	body := &emailModel{}

	err := handler.BodyParser(body)
	if err != nil {
		return handler.Status(fiber.StatusBadRequest).JSON(sent{err.Error()})
	}

	err = controller.core.sendEmail(handler.Params("name"), *body)
	if err != nil {
		if errors.Is(err, errQueueDontExist) {
			return handler.Status(fiber.StatusConflict).JSON(sent{errQueueDontExist.Error()})
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

		log.Printf("[ERROR] - Error sending email: %s", err)

		return handler.Status(fiber.StatusInternalServerError).
			JSON(sent{"error sending email"})
	}

	return handler.JSON(sent{"email sent"})
}
