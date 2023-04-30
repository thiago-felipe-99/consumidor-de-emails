package main

import (
	"log"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/thiago-felipe-99/mail/publisher/controllers"
	"github.com/thiago-felipe-99/mail/publisher/core"
	"github.com/thiago-felipe-99/mail/publisher/data"
	_ "github.com/thiago-felipe-99/mail/publisher/docs"
	"github.com/thiago-felipe-99/mail/rabbit"
)

const sessionDuration = 5 * time.Second

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

	minio, err := minio.New("localhost:9000", &minio.Options{
		Creds: credentials.NewStaticV4("minio", "miniominio", ""),
	})
	if err != nil {
		log.Printf("[ERROR] - Error connecting with the Minio: %s", err)

		return
	}

	mongoClient, err := data.NewMongoClient(
		"mongodb://mongo:mongo@localhost:27017/?connectTimeoutMS=10000&timeoutMS=5000&maxIdleTimeMS=100",
	)
	if err != nil {
		log.Printf("[ERROR] - Error creating datase: %s", err)

		return
	}

	databases := data.NewDatabases(mongoClient)

	queues, err := databases.Queue.GetAll()
	if err != nil {
		log.Printf("[ERROR] - Error getting queues: %s", err)

		return
	}

	for _, queue := range queues {
		err := rabbitConnection.CreateQueueWithDLX(queue.Name, queue.DLX, queue.MaxRetries)
		if err != nil {
			log.Printf("[ERROR] - Error creating queue: %s", err)

			return
		}
	}

	validate := validator.New()

	cores := core.NewCores(
		databases,
		validate,
		sessionDuration,
		rabbitConnection,
		minio,
		"template",
	)

	server, err := controllers.CreateHTTPServer(validate, cores)
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
