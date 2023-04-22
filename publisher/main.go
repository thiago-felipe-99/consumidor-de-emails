package main

import (
	"log"

	"github.com/thiago-felipe-99/mail/publisher/http"
	"github.com/thiago-felipe-99/mail/publisher/http/rabbit"
)

func main() {
	rabbitConfig := rabbit.Config{
		User:     "rabbit",
		Password: "rabbit",
		Host:     "localhost",
		Port:     "5672",
		Vhost:    "email",
	}

	rabbitConnection, err := rabbit.New(rabbitConfig)
	if err != nil {
		log.Printf("[ERROR] - Erro creating connection with RabbitMQ: %s", err)

		return
	}
	defer rabbitConnection.Close()

	server := http.CreateServer(rabbitConnection)

	err = server.Listen(":8080")
	if err != nil {
		log.Printf("[ERROR] - Error listen HTTP server: %s", err)

		return
	}
}
