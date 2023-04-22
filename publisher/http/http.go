package http

import (
	"log"

	"github.com/gofiber/fiber/v2"
)

type QueueParams struct {
	Name string `json:"name"`
}

func createQueue(handler *fiber.Ctx) error {
	params := &QueueParams{}

  err :=  handler.BodyParser(params)
  if err != nil {
    return err
  }

  log.Println(params)

  return nil
}

func CreateServer() *fiber.App {
	app := fiber.New()

	app.Post("/email/queue", createQueue)

	return app
}
