//nolint:wrapcheck
package controllers

import (
	"errors"
	"log"

	ut "github.com/go-playground/universal-translator"
	"github.com/gofiber/fiber/v2"
	"github.com/thiago-felipe-99/mail/publisher/core"
	"github.com/thiago-felipe-99/mail/publisher/model"
)

type sent struct {
	Message string `json:"message" bson:"message"`
}

type Queue struct {
	translator *ut.UniversalTranslator
	languages  []string
	core       *core.Queue
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
// @Param			queue	body		model.QueuePartial	true	"queue params"
// @Router			/email/queue [post]
// @Description	Creating a RabbitMQ queue with DLX.
func (controller *Queue) create(handler *fiber.Ctx) error {
	body := &model.QueuePartial{
		MaxRetries: 10, //nolint:gomnd
	}

	err := handler.BodyParser(body)
	if err != nil {
		return handler.Status(fiber.StatusBadRequest).JSON(sent{err.Error()})
	}

	err = controller.core.Create(*body)
	if err != nil {
		modelInvalid := core.ModelInvalidError{}
		if okay := errors.As(err, &modelInvalid); okay {
			accept := handler.AcceptsLanguages(controller.languages...)
			if accept == "" {
				accept = controller.languages[0]
			}

			language, _ := controller.translator.GetTranslator(accept)

			return handler.Status(fiber.StatusBadRequest).
				JSON(sent{modelInvalid.Translate(language)})
		}

		if errors.Is(err, core.ErrQueueAlreadyExist) {
			return handler.Status(fiber.StatusConflict).
				JSON(sent{core.ErrQueueAlreadyExist.Error()})
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
// @Success		200		{array}	model.Queue "all queues"
// @Failure		500		{object}	sent "internal server error"
// @Router			/email/queue [get]
// @Description	Getting all RabbitMQ queues.
func (controller *Queue) getAll(handler *fiber.Ctx) error {
	queues, err := controller.core.GetAll()
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
func (controller *Queue) delete(handler *fiber.Ctx) error {
	err := controller.core.Delete(handler.Params("name"))
	if err != nil {
		if errors.Is(err, core.ErrQueueDontExist) {
			return handler.Status(fiber.StatusNotFound).JSON(sent{core.ErrQueueDontExist.Error()})
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
// @Param			queue	body	model.Email	true	"email"
// @Router			/email/queue/{name}/send [post]
// @Description	Sends an email to the RabbitMQ queue.
func (controller *Queue) sendEmail(handler *fiber.Ctx) error {
	body := &model.Email{}

	err := handler.BodyParser(body)
	if err != nil {
		return handler.Status(fiber.StatusBadRequest).JSON(sent{err.Error()})
	}

	err = controller.core.SendEmail(handler.Params("name"), *body)
	if err != nil {
		modelInvalid := core.ModelInvalidError{}
		if okay := errors.As(err, &modelInvalid); okay {
			accept := handler.AcceptsLanguages(controller.languages...)
			if accept == "" {
				accept = controller.languages[0]
			}

			language, _ := controller.translator.GetTranslator(accept)

			return handler.Status(fiber.StatusBadRequest).
				JSON(sent{modelInvalid.Translate(language)})
		}

		if errors.Is(err, core.ErrQueueDontExist) {
			return handler.Status(fiber.StatusNotFound).JSON(sent{core.ErrQueueDontExist.Error()})
		}

		log.Printf("[ERROR] - Error sending email: %s", err)

		return handler.Status(fiber.StatusInternalServerError).
			JSON(sent{"error sending email"})
	}

	return handler.JSON(sent{"email sent"})
}

type Template struct {
	translator *ut.UniversalTranslator
	languages  []string
	core       *core.Template
}

// Creating a email template
//
// @Summary		Creating template
// @Tags			template
// @Accept			json
// @Produce		json
// @Success		200		{object}	sent "create template successfully"
// @Failure		400		{object}	sent "an invalid template param was sent"
// @Failure		409		{object}	sent "template name already exist"
// @Failure		500		{object}	sent "internal server error"
// @Param			template	body		model.TemplatePartial	true	"template params"
// @Router			/email/template [post]
// @Description	Creating a email template.
func (controller *Template) create(handler *fiber.Ctx) error {
	body := &model.TemplatePartial{}

	err := handler.BodyParser(body)
	if err != nil {
		return handler.Status(fiber.StatusBadRequest).JSON(sent{err.Error()})
	}

	err = controller.core.Create(*body)
	if err != nil {
		modelInvalid := core.ModelInvalidError{}
		if okay := errors.As(err, &modelInvalid); okay {
			accept := handler.AcceptsLanguages(controller.languages...)
			if accept == "" {
				accept = controller.languages[0]
			}

			language, _ := controller.translator.GetTranslator(accept)

			return handler.Status(fiber.StatusBadRequest).
				JSON(sent{modelInvalid.Translate(language)})
		}

		if errors.Is(err, core.ErrTemplateNameAlreadyExist) {
			return handler.Status(fiber.StatusConflict).
				JSON(sent{core.ErrTemplateNameAlreadyExist.Error()})
		}

		log.Printf("[ERROR] - Error creating template: %s", err)

		return handler.Status(fiber.StatusInternalServerError).
			JSON(sent{"error creating template"})
	}

	return handler.Status(fiber.StatusCreated).JSON(sent{"template created"})
}

