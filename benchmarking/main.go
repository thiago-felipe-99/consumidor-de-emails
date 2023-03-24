package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	amqp "github.com/rabbitmq/amqp091-go"
)

type rabbit struct {
	user, senha, host, porta, vhost, fila string
}

type configuracoes struct {
	rabbit
	bench           []int
	esperarConsumir bool
  contentType, body string
}

func pegarConfiguracoes() (*configuracoes, error) {
	err := godotenv.Load()
	if err != nil {
		return nil, err
	}

	ranges := strings.Split(os.Getenv("BENCH"), " ")
	log.Println(ranges)

	bench := []int{}
	for _, mark := range ranges {
		number, err := strconv.Atoi(mark)
		if err != nil {
			log.Fatalf("[ERROR] - Erro ao criar os testes: %s", err)
		}

		bench = append(bench, number)
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
		bench:           bench,
		esperarConsumir: os.Getenv("EXPECT_CONSUME") == "true",
    contentType: os.Getenv("CONTENT_TYPE"),
    body: os.Getenv("BODY"),
	}

	return config, nil
}

func logStatus(fila string, canal *amqp.Channel) (int, error) {
	status, err := canal.QueueDeclarePassive(
		fila,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Printf("[ERROR] - Erro ao pegar o status da fila: %s", err)

		return 0, err
	}

	log.Printf(
		"[INFO] - A fila '%s' tem %d mensagens e %d consumidores",
		status.Name,
		status.Messages,
		status.Consumers,
	)

	return status.Messages, nil
}

func main() {
	configs, err := pegarConfiguracoes()
	if err != nil {
		log.Fatalf("[ERRO] - Erro ao ler as configurações: %v", err)
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
		log.Fatalf("[ERRO] - Erro ao conectar com o Rabbit: %s", err)
	}
	defer rabbit.Close()

	canal, err := rabbit.Channel()
	if err != nil {
		log.Fatalf("[ERRO] - Erro ao abrir o canal do Rabbit: %s", err)
	}
	defer canal.Close()

	fila, err := canal.QueueDeclare(configs.rabbit.fila, false, false, false, false, nil)
	if err != nil {
		log.Fatalf("[ERRO] - Erro ao declarar a fila: %s", err)
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

	for _, quantidadeDeMensagens := range configs.bench {
		for i := 1; i <= quantidadeDeMensagens; i++ {
			err := canal.PublishWithContext(
				context.Background(),
				"",
				fila.Name,
				false,
				false,
				mensagem,
			)
			if err != nil {
				log.Fatalf("[ERROR] - Erro ao enviar mensagem para a fila: %s", err)
			}
		}

		log.Printf(
			"[INFO] - Foram enviados %d mensagens para a fila '%s'",
			quantidadeDeMensagens,
			fila.Name,
		)

		time.Sleep(1 * time.Second)

		qtMensagens, err := logStatus(configs.rabbit.fila, canal)
		if configs.esperarConsumir {
			for err != nil || qtMensagens > 0 {
        time.Sleep(1 * time.Second)
				qtMensagens, err = logStatus(configs.rabbit.fila, canal)
			}
		}
	}
}
