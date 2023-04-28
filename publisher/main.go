package main

import (
	"log"

	_ "github.com/lib/pq"
	_ "github.com/thiago-felipe-99/mail/publisher/docs"
	"github.com/thiago-felipe-99/mail/rabbit"
)

// @title			Publisher Emails
// @version		1.0
// @host			localhost:8080
// @BasePath		/
// @description	This is an api that publishes emails in RabbitMQ.
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

	database, err := newDatabase()
	if err != nil {
		log.Printf("[ERROR] - Error creating datase: %s", err)

		return
	}

	server, err := createHTTPServer(rabbitConnection, database)
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
