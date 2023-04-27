package main

import (
	"log"
	"time"

	"github.com/thiago-felipe-99/mail/publisher/http"
	"github.com/thiago-felipe-99/mail/rabbit"
)

func updateQueues(queues *rabbit.Queues) {
	for {
		err := queues.UpdateQueues()
		if err != nil {
			log.Printf("[ERROR] - Error updating queues: %s", err)
		}

		time.Sleep(time.Second * 15) //nolint: gomnd
	}
}

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

	queues := rabbit.NewQueues(rabbitConfig)

	go updateQueues(queues)

	server, err := http.CreateServer(rabbitConnection, queues)
	if err != nil {
		log.Printf("[ERROR] - Error create server: %s", err)

		return
	}

	err = server.Listen(":8080")
	if err != nil {
		log.Printf("[ERROR] - Error listen HTTP server: %s", err)

		return
	}
}
