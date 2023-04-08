package main

import (
	"encoding/json"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/wneessen/go-mail"
)

type destinatario struct {
	Nome, Email string
}

type email struct {
	Destinatario      destinatario
	Assunto, Mensagem string
	Tipo              mail.ContentType
	Anexos            []string
	mensagemRabbit    amqp.Delivery
}

type enviar struct {
	*cache
	*remetente
	*metricas
}

func novoEnviar(cache *cache, remetente *remetente, metricas *metricas) *enviar {
	return &enviar{
		cache:     cache,
		remetente: remetente,
		metricas:  metricas,
	}
}

func (enviar *enviar) emailsParaFila(descricao string, err error, emails []email) {
	enviar.metricas.emailsReenviados.Add(float64(len(emails)))

	log.Printf("[ERRO] - Erro ao processar um lote de emails, reenviando eles para a fila")
	log.Printf("[ERRO] - %s: %s", descricao, err)

	for _, email := range emails {
		err = email.mensagemRabbit.Nack(false, true)
		if err != nil {
			log.Printf("[ERRO] Erro ao reenviar mensagem para fila: %s", err)
		}
	}
}

func (enviar *enviar) mensagemParaFila(descricao string, err error, mensagem amqp.Delivery) {
	enviar.metricas.emailsReenviados.Inc()

	log.Printf("[ERRO] - Erro ao processar a mensagem, reenviando ela para a fila")
	log.Printf("[ERRO] - %s: %s", descricao, err)

	err = mensagem.Nack(false, true)
	if err != nil {
		log.Printf("[ERRO] Erro ao reenviar mensagem para fila: %s", err)
	}
}

func (enviar *enviar) emails(fila []amqp.Delivery) {
	tempoInicial := time.Now()
	enviar.metricas.emailsRecebidos.Add(float64(len(fila)))

	emails := []email{}
	bytesRecebidos := 0

	for _, mensagem := range fila {
		bytesRecebidos += len(mensagem.Body)

		email := email{}

		err := json.Unmarshal(mensagem.Body, &email)
		if err != nil {
			descricao := "Erro ao converter a mensagem para um email"
			enviar.mensagemParaFila(descricao, err, mensagem)
		} else {
			email.mensagemRabbit = mensagem
			emails = append(emails, email)
		}
	}

	enviar.metricas.emailsRecebidosBytes.Add(float64(bytesRecebidos))

	opcoesCliente := []mail.Option{
		mail.WithPort(enviar.remetente.porta),
		mail.WithSMTPAuth(mail.SMTPAuthPlain),
		mail.WithUsername(enviar.remetente.email),
		mail.WithPassword(enviar.remetente.senha),
		mail.WithTLSPolicy(mail.TLSMandatory),
	}

	cliente, err := mail.NewClient(enviar.remetente.host, opcoesCliente...)
	if err != nil {
		descricao := "Erro ao criar um cliente de email"
		enviar.emailsParaFila(descricao, err, emails)
		return
	}

	mensagens := []*mail.Msg{}
	emailsProcessados := []email{}

	for _, email := range emails {
		mensagem := mail.NewMsg()
		err = mensagem.EnvelopeFromFormat(enviar.remetente.nome, enviar.remetente.email)
		if err != nil {
			descricao := "Erro ao colocar remetente no email"
			enviar.mensagemParaFila(descricao, err, email.mensagemRabbit)
			continue
		}

		err = mensagem.AddToFormat(email.Destinatario.Nome, email.Destinatario.Email)
		if err != nil {
			descricao := "Erro ao colocar destinatario no email"
			enviar.mensagemParaFila(descricao, err, email.mensagemRabbit)
			continue
		}

		mensagem.Subject(email.Assunto)
		mensagem.SetBodyString(email.Tipo, email.Mensagem)

		mensagens = append(mensagens, mensagem)
		emailsProcessados = append(emailsProcessados, email)
	}

	err = cliente.DialAndSend(mensagens...)
	if err != nil {
		descricao := "Erro ao enviar os emails"
		enviar.emailsParaFila(descricao, err, emailsProcessados)
		return
	}

	emailsEnviados := 0
	bytesEnviados := 0

	for _, email := range emailsProcessados {
		err := email.mensagemRabbit.Ack(false)
		if err != nil {
			log.Printf("[ERRO] - Erro ao enviar mensagem de finalização para o rabbit: %s", err)
		} else {
			emailsEnviados += 1
			bytesEnviados += len(email.Mensagem)
		}
	}

	tempoDecorrido := time.Since(tempoInicial).Seconds()

	enviar.metricas.emailsEnviados.Add(float64(emailsEnviados))
	enviar.metricas.emailsTempoDeEnvioSegundos.Observe(tempoDecorrido)
	enviar.metricas.emailsEnviadosBytes.Add(float64(bytesEnviados))

	log.Printf("[INFO] - Foram enviado %d emails", emailsEnviados)
}
