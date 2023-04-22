package main

import (
	"log"

	"github.com/thiago-felipe-99/mail/publisher/http"
)

func main() {
	server := http.CreateServer()

	err := server.Listen(":8080")
	if err != nil {
		log.Panicf("[ERROR] - Error listen HTTP server: %s", err)
	}
}
