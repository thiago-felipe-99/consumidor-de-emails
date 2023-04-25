//nolint:wrapcheck
package http

import (
	"context"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/thiago-felipe-99/mail/rabbit"
)

func createQueue(rabbit *rabbit.Rabbit, queues []string) func(*fiber.Ctx) error {
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

		err = rabbit.CreateQueue(body.Name, body.MaxRetries)
		if err != nil {
			log.Printf("[ERROR] - Error creating queue: %s", err)

			return handler.Status(fiber.StatusInternalServerError).
				SendString("error creating queue")
		}

		queues = append(queues, body.Name)

		return nil
	}
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

func sendEmail(rabbit *rabbit.Rabbit) func(*fiber.Ctx) error {
	return func(handler *fiber.Ctx) error {
		queue := handler.Params("name")

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
	queues := []string{}

	app := fiber.New()

	app.Use(recover.New())

	app.Post("/email/queue", createQueue(rabbit, queues))
	app.Post("/email/queue/:name/add", sendEmail(rabbit))

	return app
}
