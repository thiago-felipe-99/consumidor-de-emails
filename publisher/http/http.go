package http

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/thiago-felipe-99/mail/publisher/http/rabbit"
)

func createQueue(rabbit *rabbit.Rabbit) func(*fiber.Ctx) error {
	return func(handler *fiber.Ctx) error {
		params := &struct {
			Name       string `json:"name"`
			MaxRetries int    `json:"maxRetries"`
    }{
      MaxRetries: 10,
    }

		err := handler.BodyParser(params)
		if err != nil {
			return err
		}

		err = rabbit.CreateQueue(params.Name, params.MaxRetries)
		if err != nil {
			log.Printf("[ERROR] - Error creating queue: %s", err)

			return handler.Status(fiber.StatusInternalServerError).
				SendString("error creating queue")
		}

		return nil
	}
}

func CreateServer(rabbit *rabbit.Rabbit) *fiber.App {
	app := fiber.New()

	app.Post("/email/queue", createQueue(rabbit))

	return app
}
