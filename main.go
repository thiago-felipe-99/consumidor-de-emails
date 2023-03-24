package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/wneessen/go-mail"
)

const (
	tamanhoBuffer  = 1000
	duracaoTimeout = 2 * time.Second
)

type remetente struct {
	nome, email, senha, host string
	porta                    int
}

type rabbit struct {
	user, senha, host, porta, vhost, fila string
}

type configuracoes struct {
	remetente
	rabbit
}

func pegarConfiguracoes() (*configuracoes, error) {
	err := godotenv.Load()
	if err != nil {
		return nil, err
	}

	porta, err := strconv.Atoi(os.Getenv("SMTP_PORT"))
	if err != nil {
		return nil, err
	}

	config := &configuracoes{
		remetente: remetente{
			nome:  os.Getenv("SMTP_USERNAME"),
			email: os.Getenv("SMTP_USER"),
			senha: os.Getenv("SMTP_PASSWORD"),
			host:  os.Getenv("SMTP_HOST"),
			porta: porta,
		},
		rabbit: rabbit{
			user:  os.Getenv("RABBIT_USER"),
			senha: os.Getenv("RABBIT_PASSWORD"),
			host:  os.Getenv("RABBIT_HOST"),
			porta: os.Getenv("RABBIT_PORT"),
			vhost: os.Getenv("RABBIT_VHOST"),
			fila:  os.Getenv("RABBIT_QUEUE"),
		},
	}

	return config, nil
}

type destinatario struct {
	Nome, Email string
}

type email struct {
	Destinatario                destinatario
	Assunto, Mensagem, Template string
	Anexos                      []string
}

func enviarEmails(remetente remetente, emails []email) error {
	opcoesCliente := []mail.Option{
		mail.WithPort(remetente.porta),
		mail.WithSMTPAuth(mail.SMTPAuthPlain),
		mail.WithUsername(remetente.email),
		mail.WithPassword(remetente.senha),
		mail.WithTLSPolicy(mail.TLSMandatory),
	}

	cliente, err := mail.NewClient(remetente.host, opcoesCliente...)
	if err != nil {
		return err
	}

	mensagens := []*mail.Msg{}
	for _, email := range emails {
		mensagem := mail.NewMsg()
		err = mensagem.EnvelopeFromFormat(remetente.nome, remetente.email)
		if err != nil {
			return err
		}

		err = mensagem.AddToFormat(email.Destinatario.Nome, email.Destinatario.Email)
		if err != nil {
			return err
		}

		mensagem.Subject(email.Assunto)
		mensagem.SetBodyString(mail.TypeTextPlain, email.Mensagem)

		mensagens = append(mensagens, mensagem)
	}

	return cliente.DialAndSend(mensagens...)
}

func converterMensagemParaEmail(fila []amqp.Delivery) []email {
  emails := []email{}
  
	for _, mensagem := range fila {
		email := email{}
		err := json.Unmarshal(mensagem.Body, &email)
		if err != nil {
			log.Printf("Erro ao desserializar o email: %s", err)
			err = mensagem.Nack(false, true)
			if err != nil {
				log.Printf("Erro ao enviar sinal de finalização para o Rabbit: %s", err)
			}

			continue
		}

    emails = append(emails, email)

		err = mensagem.Ack(false)
		if err != nil {
			log.Printf("Erro ao enviar sinal de finalização para o Rabbit: %s", err)
		}
	}
	log.Printf("Enviando %d mensagens", len(fila))

  return emails
}

func main() {
	configuracoes, err := pegarConfiguracoes()
	if err != nil {
		log.Fatalf("Erro ao ler as configurações: %v", err)
	}

	rabbitURL := fmt.Sprintf(
		"amqp://%s:%s@%s:%s/%s",
		configuracoes.rabbit.user,
		configuracoes.rabbit.senha,
		configuracoes.rabbit.host,
		configuracoes.rabbit.porta,
		configuracoes.rabbit.vhost,
	)

	rabbit, err := amqp.Dial(rabbitURL)
	if err != nil {
		log.Fatalf("Erro ao conenectar com o Rabbit: %s", err)
	}
	defer rabbit.Close()

	fila, err := rabbit.Channel()
	if err != nil {
		log.Fatalf("Erro ao abrir o canal do Rabbit: %s", err)
	}
	defer fila.Close()

	err = fila.Qos(tamanhoBuffer, 0, false)
	if err != nil {
		log.Fatalf("Erro ao configurar o tamanho da fila do consumidor: %s", err)
	}

	mensagens, err := fila.Consume(configuracoes.rabbit.fila, "", false, false, false, false, nil)
	if err != nil {
		log.Fatalf("Erro ao registrar o consumidor: %s", err)
	}

	var esperar chan struct{}

	go func() {
		filaDeMensagens := []amqp.Delivery{}
		timeout := time.NewTicker(duracaoTimeout)

		for {
			select {
			case mensagen := <-mensagens:
				filaDeMensagens = append(filaDeMensagens, mensagen)
				timeout.Reset(duracaoTimeout)

				if len(filaDeMensagens) >= tamanhoBuffer {
          emails := converterMensagemParaEmail(filaDeMensagens)
          log.Println(emails)
					filaDeMensagens = filaDeMensagens[:0]
				}

			case <-timeout.C:
				if len(filaDeMensagens) > 0 {
          emails := converterMensagemParaEmail(filaDeMensagens)
          log.Println(emails)
					filaDeMensagens = filaDeMensagens[:0]
				}
			}
		}
	}()

	log.Printf(" [*] Esperando por mensagens. Para sair aperte CTRL+C")
	<-esperar
}
