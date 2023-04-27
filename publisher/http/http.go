//nolint:wrapcheck
package http

import (
	"context"
	"errors"
	"log"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/thiago-felipe-99/mail/rabbit"
)

var (
	errQueueAlreadyExist = errors.New("queue already exist")
	errQueueDontExist    = errors.New("queue dont exist")
	errBodyValidate      = errors.New("unable to parse body")
)

type receiver struct {
	Name  string `json:"name"  validate:"required"`
	Email string `json:"email" validate:"required,email"`
}

type template struct {
	Name string            `json:"name" validate:"required"`
	Data map[string]string `json:"data"`
}

type email struct {
	Receivers      []receiver `json:"receivers"      validate:"required_without=BlindReceivers"`
	BlindReceivers []receiver `json:"blindReceivers" validate:"required_without=Receivers"`
	Subject        string     `json:"subject"        validate:"required"`
	Message        string     `json:"message"        validate:"required_without=Template,excluded_with=Template"`
	Template       *template  `json:"template"       validate:"required_without=Message,excluded_with=Message"`
	Attachments    []string   `json:"attachments"`
}

type queue struct {
	rabbit   *rabbit.Rabbit
	queues   *rabbit.Queues
	validate *validator.Validate
}

func (queue *queue) create() func(*fiber.Ctx) error {
	return func(handler *fiber.Ctx) error {
		body := &struct {
			Name       string `json:"name" validate:"required"`
			MaxRetries int64  `json:"maxRetries"`
		}{
			MaxRetries: 10, //nolint:gomnd
		}

		err := handler.BodyParser(body)
		if err != nil {
			return err
		}

		err = queue.validate.Struct(body)
		if err != nil {
			validationErrs := validator.ValidationErrors{}

			okay := errors.As(err, &validationErrs)
			if !okay {
				return errBodyValidate
			}

			return handler.Status(fiber.StatusBadRequest).SendString(err.Error())
		}

		if queue.queues.Exist(body.Name) {
			return handler.Status(fiber.StatusConflict).SendString(errQueueAlreadyExist.Error())
		}

		err = queue.rabbit.CreateQueue(body.Name, body.MaxRetries)
		if err != nil {
			log.Printf("[ERROR] - Error creating queue: %s", err)

			return handler.Status(fiber.StatusInternalServerError).
				SendString("error creating queue")
		}

		queue.queues.Add(body.Name)

		return nil
	}
}

func (queue *queue) send() func(*fiber.Ctx) error {
	return func(handler *fiber.Ctx) error {
		name := handler.Params("name")

		if !queue.queues.Exist(name) {
			return handler.Status(fiber.StatusNotFound).SendString(errQueueDontExist.Error())
		}

		body := &email{}

		err := handler.BodyParser(body)
		if err != nil {
			return err
		}

		err = queue.validate.Struct(body)
		if err != nil {
			validationErrs := validator.ValidationErrors{}

			okay := errors.As(err, &validationErrs)
			if !okay {
				return errBodyValidate
			}

			return handler.Status(fiber.StatusBadRequest).SendString(err.Error())
		}

		err = queue.rabbit.SendMessage(context.Background(), name, body)
		if err != nil {
			log.Printf("[ERROR] - Error creating queue: %s", err)

			return handler.Status(fiber.StatusInternalServerError).
				SendString("error creating queue")
		}

		return nil
	}
}

func CreateServer(rabbit *rabbit.Rabbit, queues *rabbit.Queues) *fiber.App {
	app := fiber.New()

	app.Use(recover.New())

	queue := queue{
		rabbit:   rabbit,
		queues:   queues,
		validate: validator.New(),
	}

	app.Post("/email/queue", queue.create())
	app.Post("/email/queue/:name/add", queue.send())

	return app
}
