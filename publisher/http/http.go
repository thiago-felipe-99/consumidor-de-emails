//nolint:wrapcheck
package http

import (
	"context"
	"errors"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/thiago-felipe-99/mail/rabbit"
)

var (
	errQueueAlreadyExist = errors.New("queue already exist")
	errQueueDontExist    = errors.New("queue dont exist")
)

type queues []string

func (queues *queues) add(name string) {
	if queues.exist(name) {
		return
	}

	*queues = append(*queues, name)
}

func (queues *queues) exist(name string) bool {
	for _, queue := range *queues {
		if queue == name {
			return true
		}
	}

	return false
}

type receiver struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type template struct {
	Name string            `json:"name"`
	Data map[string]string `json:"data"`
}

type email struct {
	Receivers      []receiver `json:"receivers"`
	BlindReceivers []receiver `json:"blindReceivers"`
	Subject        string     `json:"subject"`
	Message        string     `json:"message"`
	Template       template   `json:"template"`
	Attachments    []string   `json:"attachments"`
}

func createQueue(rabbit *rabbit.Rabbit, queues queues) func(*fiber.Ctx) error {
	return func(handler *fiber.Ctx) error {
		body := &struct {
			Name       string `json:"name"`
			MaxRetries int64  `json:"maxRetries"`
		}{
			MaxRetries: 10, //nolint:gomnd
		}

		err := handler.BodyParser(body)
		if err != nil {
			return err
		}

		if queues.exist(body.Name) {
			return handler.Status(fiber.StatusConflict).SendString(errQueueAlreadyExist.Error())
		}

		err = rabbit.CreateQueue(body.Name, body.MaxRetries)
		if err != nil {
			log.Printf("[ERROR] - Error creating queue: %s", err)

			return handler.Status(fiber.StatusInternalServerError).
				SendString("error creating queue")
		}

		queues.add(body.Name)

		return nil
	}
}

func sendEmail(rabbit *rabbit.Rabbit, queues queues) func(*fiber.Ctx) error {
	return func(handler *fiber.Ctx) error {
		queue := handler.Params("name")

		if !queues.exist(queue) {
			return handler.Status(fiber.StatusNotFound).SendString(errQueueDontExist.Error())
		}

		body := &email{}

		err := handler.BodyParser(body)
		if err != nil {
			return err
		}

		err = rabbit.SendMessage(context.Background(), queue, body)
		if err != nil {
			log.Printf("[ERROR] - Error creating queue: %s", err)

			return handler.Status(fiber.StatusInternalServerError).
				SendString("error creating queue")
		}

		return nil
	}
}

func CreateServer(rabbit *rabbit.Rabbit) *fiber.App {
	queues := queues{}

	app := fiber.New()

	app.Use(recover.New())

	app.Post("/email/queue", createQueue(rabbit, queues))
	app.Post("/email/queue/:name/add", sendEmail(rabbit, queues))

	return app
}
