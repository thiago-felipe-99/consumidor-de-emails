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

type expectError struct {
	err    error
	status int
}

type okay struct {
	message string
	status  int
}

func callingCore(
	coreFunc func() error,
	expectErrors []expectError,
	unexpectMessageError string,
	okay okay,
	language ut.Translator,
	handler *fiber.Ctx,
) error {
	err := coreFunc()
	if err != nil {
		modelInvalid := core.ModelInvalidError{}
		if okay := errors.As(err, &modelInvalid); okay {
			return handler.Status(fiber.StatusBadRequest).
				JSON(sent{modelInvalid.Translate(language)})
		}

		for _, expectError := range expectErrors {
			if errors.Is(err, expectError.err) {
				return handler.Status(expectError.status).JSON(sent{expectError.err.Error()})
			}
		}

		log.Printf("[ERROR] - %s: %s", unexpectMessageError, err)

		return handler.Status(fiber.StatusInternalServerError).
			JSON(sent{unexpectMessageError})
	}

	return handler.Status(okay.status).JSON(sent{okay.message})
}

type Queue struct {
	core       *core.Queue
	translator *ut.UniversalTranslator
	languages  []string
}

func (controller *Queue) getTranslator(handler *fiber.Ctx) ut.Translator { //nolint:ireturn
	accept := handler.AcceptsLanguages(controller.languages...)
	if accept == "" {
		accept = controller.languages[0]
	}

	language, _ := controller.translator.GetTranslator(accept)

	return language
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

	funcCore := func() error { return controller.core.Create(*body) }

	expectErrors := []expectError{{core.ErrQueueAlreadyExist, fiber.StatusConflict}}

	unexpectMessageError := "error creating queue"

	okay := okay{"queue created", fiber.StatusCreated}

	return callingCore(
		funcCore,
		expectErrors,
		unexpectMessageError,
		okay,
		controller.getTranslator(handler),
		handler,
	)
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
			JSON(sent{"error getting all queues"})
	}

	return handler.JSON(queues)
}

// Delete a queue with DLX
//
// @Summary		Delete queues
// @Tags			queue
// @Accept			json
// @Produce		json
// @Success		204		{object}	sent "queue deleted"
// @Failure		404		{object}	sent "queue dont exist"
// @Failure		500		{object}	sent "internal server error"
// @Router			/email/queue/{name} [delete]
// @Description	Delete a queue with DLX.
func (controller *Queue) delete(handler *fiber.Ctx) error {
	funcCore := func() error { return controller.core.Delete(handler.Params("name")) }

	expectErrors := []expectError{{core.ErrQueueDoesNotExist, fiber.StatusNotFound}}

	unexpectMessageError := "error deleting queue"

	okay := okay{"queue deleted", fiber.StatusOK}

	return callingCore(
		funcCore,
		expectErrors,
		unexpectMessageError,
		okay,
		controller.getTranslator(handler),
		handler,
	)
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

	funcCore := func() error { return controller.core.SendEmail(handler.Params("name"), *body) }

	expectErrors := []expectError{
		{core.ErrQueueDoesNotExist, fiber.StatusNotFound},
		{core.ErrMissingFieldTemplates, fiber.StatusBadRequest},
		{core.ErrTemplateDoesNotExist, fiber.StatusBadRequest},
	}

	unexpectMessageError := "error sending email"

	okay := okay{"email sent", fiber.StatusOK}

	return callingCore(
		funcCore,
		expectErrors,
		unexpectMessageError,
		okay,
		controller.getTranslator(handler),
		handler,
	)
}

type Template struct {
	core       *core.Template
	translator *ut.UniversalTranslator
	languages  []string
}

func (controller *Template) getTranslator(handler *fiber.Ctx) ut.Translator { //nolint:ireturn
	accept := handler.AcceptsLanguages(controller.languages...)
	if accept == "" {
		accept = controller.languages[0]
	}

	language, _ := controller.translator.GetTranslator(accept)

	return language
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

	funcCore := func() error { return controller.core.Create(*body) }

	expectErrors := []expectError{
		{core.ErrTemplateNameAlreadyExist, fiber.StatusConflict},
		{core.ErrMaxSizeTemplate, fiber.StatusBadRequest},
	}

	unexpectMessageError := "error creating template"

	okay := okay{"template created", fiber.StatusCreated}

	return callingCore(
		funcCore,
		expectErrors,
		unexpectMessageError,
		okay,
		controller.getTranslator(handler),
		handler,
	)
}

// Getting all email templates
//
// @Summary		Get templates
// @Tags			template
// @Accept			json
// @Produce		json
// @Success		200		{array}	model.Template "all templates"
// @Failure		500		{object}	sent "internal server error"
// @Router			/email/template [get]
// @Description	Getting all email templates.
func (controller *Template) getAll(handler *fiber.Ctx) error {
	templates, err := controller.core.GetAll()
	if err != nil {
		log.Printf("[ERROR] - Error getting all templates: %s", err)

		return handler.Status(fiber.StatusInternalServerError).
			JSON(sent{"error getting all templates"})
	}

	return handler.JSON(templates)
}
