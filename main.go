package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	serverWriteTimeout = 10 * time.Second
	serverReadTImeout  = 5 * time.Second
)

func main() {
	configs, err := getConfigurations()
	if err != nil {
		log.Printf("[ERROR] - Error reading the configurations: %s", err)

		return
	}

	cache, err := newCache(configs)
	if err != nil {
		log.Printf("[ERROR] - Error creating the files cache: %s", err)

		return
	}

	rabbitURL := fmt.Sprintf(
		"amqp://%s:%s@%s:%d/%s",
		configs.Rabbit.User,
		configs.Rabbit.Password,
		configs.Rabbit.Host,
		configs.Rabbit.Port,
		configs.Rabbit.Vhost,
	)

	rabbit, err := amqp.Dial(rabbitURL)
	if err != nil {
		log.Printf("[ERROR] - Error connecting to RabbitMQ: %s", err)

		return
	}
	defer rabbit.Close()

	channel, err := rabbit.Channel()
	if err != nil {
		log.Printf("[ERROR] - Error opening RabbitMQ channel: %s", err)

		return
	}
	defer channel.Close()

	err = channel.Qos(configs.Buffer.Size*configs.Buffer.Quantity, 0, false)
	if err != nil {
		log.Printf("[ERROR] - Error configuring consumer queue size: %s", err)

		return
	}

	queue, err := channel.Consume(configs.Rabbit.Queue, "", false, false, false, false, nil)
	if err != nil {
		log.Printf("[ERROR] - Error registering consumer: %s", err)

		return
	}

	var wait chan struct{}

	metrics := newMetrics()
	registryMetrics := prometheus.NewRegistry()

	registryMetrics.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		metrics.emailsReceived,
		metrics.emailsReceivedBytes,
		metrics.emailsSent,
		metrics.emailsSentBytes,
		metrics.emailsSentAttachment,
		metrics.emailsSentAttachmentBytes,
		metrics.emailsSentWithAttachment,
		metrics.emailsResent,
		metrics.emailsSentTimeSeconds,
		metrics.emailsCacheAttachment,
		metrics.emailsCacheAttachmentBytes,
	)

	go func() {
		http.Handle("/metrics", promhttp.HandlerFor(registryMetrics, promhttp.HandlerOpts{
			EnableOpenMetrics: true,
		}))

		server := &http.Server{
			WriteTimeout: serverWriteTimeout,
			ReadTimeout:  serverReadTImeout,
		}

		err := server.ListenAndServe()
		if err != nil {
			log.Fatalf("[ERROR] - Error starting metrics server")
		}

		log.Printf("[INFO] - Metrics server started successfully")
	}()

	go func() {
		bufferQueue := []amqp.Delivery{}
		timeout := time.NewTicker(time.Duration(configs.Timeout) * time.Second)
		send := newSend(cache, &configs.Sender, &configs.SMTP, metrics)

		for {
			select {
			case message := <-queue:
				bufferQueue = append(bufferQueue, message)

				timeout.Reset(time.Duration(configs.Timeout) * time.Second)

				if len(bufferQueue) >= configs.Buffer.Size {
					buffer := make([]amqp.Delivery, len(bufferQueue))
					copy(buffer, bufferQueue)

					log.Printf("[INFO] - Sending %d emails", len(buffer))

					go send.emails(buffer)

					bufferQueue = bufferQueue[:0]
				}

			case <-timeout.C:
				if len(bufferQueue) > 0 {
					buffer := make([]amqp.Delivery, len(bufferQueue))
					copy(buffer, bufferQueue)

					log.Printf("[INFO] - Sending %d emails", len(buffer))

					go send.emails(buffer)

					bufferQueue = bufferQueue[:0]
				}
			}
		}
	}()

	log.Printf("[INFO] - Server started successfully")
	<-wait
}
