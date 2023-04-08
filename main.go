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

func main() {
	configs, err := pegarConfiguracoes()
	if err != nil {
		log.Fatalf("[ERRO] - Erro ao ler as configurações: %s", err)
	}

	cache, err := novoCache(configs)
	if err != nil {
		log.Fatalf("[ERRO] - Erro ao criar o cache de arquivos: %s", err)
	}

	rabbitURL := fmt.Sprintf(
		"amqp://%s:%s@%s:%d/%s",
		configs.rabbit.user,
		configs.rabbit.senha,
		configs.rabbit.host,
		configs.rabbit.porta,
		configs.rabbit.vhost,
	)

	rabbit, err := amqp.Dial(rabbitURL)
	if err != nil {
		log.Fatalf("[ERRO] - Erro ao conectar com o Rabbit: %s", err)
	}
	defer rabbit.Close()

	canal, err := rabbit.Channel()
	if err != nil {
		log.Fatalf("[ERRO] - Erro ao abrir o canal do Rabbit: %s", err)
	}
	defer canal.Close()

	err = canal.Qos(configs.buffer.tamanho*configs.buffer.quantidade, 0, false)
	if err != nil {
		log.Fatalf("[ERRO] - Erro ao configurar o tamanho da fila do consumidor: %s", err)
	}

	fila, err := canal.Consume(configs.rabbit.fila, "", false, false, false, false, nil)
	if err != nil {
		log.Fatalf("[ERRO] - Erro ao registrar o consumidor: %s", err)
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

		err := http.ListenAndServe(":8001", nil)
		if err != nil {
			log.Fatalf("[ERRO] - Erro ao inicializar servidor de metricas")
		}

		log.Printf("[INFO] - Servidor de metricas inicializado com sucesso")
	}()

	go func() {
		bufferFila := []amqp.Delivery{}
		timeout := time.NewTicker(configs.timeout)
		enviar := novoEnviar(cache, &configs.remetente, metricas)

		for {
			select {
			case mensagen := <-fila:
				bufferFila = append(bufferFila, mensagen)
				timeout.Reset(configs.timeout)

				if len(bufferFila) >= configs.buffer.tamanho {
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
