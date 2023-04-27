package main

import (
	"log"

	"github.com/thiago-felipe-99/mail/publisher/http"
	"github.com/thiago-felipe-99/mail/rabbit"
)

func main() {
	rabbitConfig := rabbit.Config{
		User:     "rabbit",
		Password: "rabbit",
		Host:     "localhost",
		Port:     "5672",
		PortAPI:  "15672",
		Vhost:    "email",
	}

	rabbitConnection := rabbit.New(rabbitConfig)
	defer rabbitConnection.Close()

	go rabbitConnection.HandleConnection()

	queus, err := rabbit.NewQueues(rabbitConfig)
	if err != nil {
		log.Printf("[ERROR] - Error creating queues: %s", err)

		return
	}

	server := http.CreateServer(rabbitConnection, queus)

	err = server.Listen(":8080")
	if err != nil {
		log.Printf("[ERROR] - Error listen HTTP server: %s", err)

		return
	}
}
