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

func newRabbit(configs *configurations) (<-chan amqp.Delivery, func(), error) {
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
		return nil, nil, fmt.Errorf("error connecting to RabbitMQ: %w", err)
	}

	channel, err := rabbit.Channel()
	if err != nil {
		rabbit.Close()

		return nil, nil, fmt.Errorf("error opening RabbitMQ channel: %w", err)
	}

	closeRabbit := func() {
		channel.Close()
		rabbit.Close()
	}

	err = channel.Qos(configs.Buffer.Size*configs.Buffer.Quantity, 0, false)
	if err != nil {
		closeRabbit()

		return nil, nil, fmt.Errorf("error configuring consumer queue size: %w", err)
	}

	queue, err := channel.Consume(configs.Rabbit.Queue, "", false, false, false, false, nil)
	if err != nil {
		closeRabbit()

		return nil, nil, fmt.Errorf("error registering consumer: %w", err)
	}

	return queue, closeRabbit, nil
}

func serverMetrics(metrics *metrics) {
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
}

func processQueue(
	queue <-chan amqp.Delivery,
	send *send,
	timeout time.Duration,
	bufferSize int,
) {
	buffer := []amqp.Delivery{}
	ticker := time.NewTicker(timeout)

	for {
		select {
		case message := <-queue:
			buffer = append(buffer, message)

			ticker.Reset(timeout)

			if len(buffer) >= bufferSize {
				buffer = send.copyQueueAndSendEmails(buffer)
			}

		case <-ticker.C:
			if len(buffer) > 0 {
				buffer = send.copyQueueAndSendEmails(buffer)
			}
		}
	}
}

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

	queue, closeRabbit, err := newRabbit(configs)
	if err != nil {
		log.Printf("[ERROR] - Error creating queue: %s", err)

		return
	}

	defer closeRabbit()

	metrics := newMetrics()

	send, err := newSend(cache, &configs.Sender, &configs.SMTP, metrics)
	if err != nil {
		log.Printf("[ERROR] - Error creating sender: %s", err)
	}

	timeout := time.Duration(configs.Timeout) * time.Second

	var wait chan struct{}

	go serverMetrics(metrics)

	go processQueue(queue, send, timeout, configs.Buffer.Size)

	log.Printf("[INFO] - Server started successfully")
	<-wait
}
