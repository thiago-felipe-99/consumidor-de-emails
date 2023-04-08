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
		log.Printf("[ERRO] - Erro ao ler as configurações: %s", err)

		return
	}

	cache, err := novoCache(configs)
	if err != nil {
		log.Printf("[ERRO] - Erro ao criar o cache de arquivos: %s", err)

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
		log.Printf("[ERRO] - Erro ao conectar com o Rabbit: %s", err)

		return
	}
	defer rabbit.Close()

	canal, err := rabbit.Channel()
	if err != nil {
		log.Printf("[ERRO] - Erro ao abrir o canal do Rabbit: %s", err)

		return
	}
	defer canal.Close()

	err = canal.Qos(configs.Buffer.Size*configs.Buffer.Quantity, 0, false)
	if err != nil {
		log.Printf("[ERRO] - Erro ao configurar o tamanho da fila do consumidor: %s", err)

		return
	}

	fila, err := canal.Consume(configs.Rabbit.Queue, "", false, false, false, false, nil)
	if err != nil {
		log.Printf("[ERRO] - Erro ao registrar o consumidor: %s", err)

		return
	}

	var esperar chan struct{}

	metricas := criarMetricas()
	registryMetricas := prometheus.NewRegistry()

	registryMetricas.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		metricas.emailsRecebidos,
		metricas.emailsRecebidosBytes,
		metricas.emailsEnviados,
		metricas.emailsEnviadosBytes,
		metricas.emailsAnexosEnviados,
		metricas.emailsAnexosEnviadosBytes,
		metricas.emailsEnviadosComAnexo,
		metricas.emailsReenviados,
		metricas.emailsTempoDeEnvioSegundos,
		metricas.emailsCacheAnexos,
		metricas.emailsCacheAnexosBytes,
	)

	go func() {
		http.Handle("/metrics", promhttp.HandlerFor(registryMetricas, promhttp.HandlerOpts{
			EnableOpenMetrics: true,
		}))

		server := &http.Server{
			WriteTimeout: serverWriteTimeout,
			ReadTimeout:  serverReadTImeout,
		}

		err := server.ListenAndServe()
		if err != nil {
			log.Fatalf("[ERRO] - Erro ao inicializar servidor de metricas")
		}

		log.Printf("[INFO] - Servidor de metricas inicializado com sucesso")
	}()

	go func() {
		bufferFila := []amqp.Delivery{}
		timeout := time.NewTicker(time.Duration(configs.Timeout) * time.Second)
		enviar := novoEnviar(cache, &configs.Sender, &configs.SMTP, metricas)

		for {
			select {
			case mensagen := <-fila:
				bufferFila = append(bufferFila, mensagen)

				timeout.Reset(time.Duration(configs.Timeout) * time.Second)

				if len(bufferFila) >= configs.Buffer.Size {
					buffer := make([]amqp.Delivery, len(bufferFila))
					copy(buffer, bufferFila)

					log.Printf("[INFO] - Fazendo envio de %d emails", len(buffer))

					go enviar.emails(buffer)

					bufferFila = bufferFila[:0]
				}

			case <-timeout.C:
				if len(bufferFila) > 0 {
					buffer := make([]amqp.Delivery, len(bufferFila))
					copy(buffer, bufferFila)

					log.Printf("[INFO] - Fazendo envio de %d emails", len(buffer))

					go enviar.emails(buffer)

					bufferFila = bufferFila[:0]
				}
			}
		}
	}()

	log.Printf("[INFO] - Servidor inicializado com sucesso")
	<-esperar
}
