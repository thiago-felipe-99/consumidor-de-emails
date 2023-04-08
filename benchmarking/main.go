package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	amqp "github.com/rabbitmq/amqp091-go"
)

type rabbit struct {
	user, senha, host, porta, vhost, fila string
}

type configuracoes struct {
	rabbit
	quantidadeDeMensagens int
	contentType, body     string
}

func pegarConfiguracoes() (*configuracoes, error) {
	err := godotenv.Load()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	quantidadeDeMensagens, err := strconv.Atoi(os.Getenv("MESSAGES_QUANTITY"))
	if err != nil {
		return nil, err
	}

	config := &configuracoes{
		rabbit: rabbit{
			user:  os.Getenv("RABBIT_USER"),
			senha: os.Getenv("RABBIT_PASSWORD"),
			host:  os.Getenv("RABBIT_HOST"),
			porta: os.Getenv("RABBIT_PORT"),
			vhost: os.Getenv("RABBIT_VHOST"),
			fila:  os.Getenv("RABBIT_QUEUE"),
		},
		quantidadeDeMensagens: quantidadeDeMensagens,
		contentType:           os.Getenv("CONTENT_TYPE"),
		body:                  os.Getenv("BODY"),
	}

	return config, nil
}

func main() {
	configs, err := pegarConfiguracoes()
	if err != nil {
		log.Printf("[ERRO] - Erro ao ler as configurações: %v", err)

		return
	}

	rabbitURL := fmt.Sprintf(
		"amqp://%s:%s@%s:%s/%s",
		configs.rabbit.user,
		configs.rabbit.senha,
		configs.rabbit.host,
		configs.rabbit.porta,
		configs.rabbit.vhost,
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

	fila, err := canal.QueueDeclare(configs.rabbit.fila, false, false, false, false, nil)
	if err != nil {
		log.Printf("[ERRO] - Erro ao declarar a fila: %s", err)

		return
	}

	log.Printf(
		"[INFO] - A fila '%s' tem %d mensagens e %d consumidores",
		fila.Name,
		fila.Messages,
		fila.Consumers,
	)

	mensagem := amqp.Publishing{
		ContentType: configs.contentType,
		Body:        []byte(configs.body),
	}

	for i := 1; i <= configs.quantidadeDeMensagens; i++ {
		err := canal.PublishWithContext(
			context.Background(),
			"",
			fila.Name,
			false,
			false,
			mensagem,
		)
		if err != nil {
			log.Printf("[ERRO] - Erro ao enviar mensagem para a fila: %s", err)

			return
		}
	}

	log.Printf(
		"[INFO] - Foram enviados %d mensagens para a fila '%s'",
		configs.quantidadeDeMensagens,
		fila.Name,
	)
}
