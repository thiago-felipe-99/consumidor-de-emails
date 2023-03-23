package main

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/wneessen/go-mail"
)

type remetente struct {
	nome, email, senha, host string
	porta              int
}

type configuracoes struct {
	remetente
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
      nome: os.Getenv("SMTP_USERNAME"),
			email: os.Getenv("SMTP_USER"),
			senha: os.Getenv("SMTP_PASSWORD"),
			host:  os.Getenv("SMTP_HOST"),
			porta: porta,
		},
	}

	return config, nil
}

type email struct {
	destinatario, descricao, mensagem string
	caminhoAnexos                     []string
}

func enviarEmail(remetente remetente, email email) error {
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

	mensagem := mail.NewMsg()
	err = mensagem.EnvelopeFromFormat(remetente.nome, remetente.email)
	if err != nil {
		return err
	}

	err = mensagem.AddTo(email.destinatario)
	if err != nil {
		return err
	}

	mensagem.Subject(email.descricao)
	mensagem.SetBodyString(mail.TypeTextPlain, email.mensagem)

	return cliente.DialAndSend(mensagem)
}

func main() {
	config, err := pegarConfiguracoes()
	if err != nil {
		log.Fatalf("Erro ao ler as configurações: %v", err)
	}

	email := email{
		destinatario:  os.Getenv("EMAIL_TEST"),
		descricao:     "Testando o serviço de email",
		mensagem:      "Uma mensagem bem bonita",
		caminhoAnexos: []string{},
	}

	err = enviarEmail(config.remetente, email)
	if err != nil {
		log.Fatalf("Erro ao enviar o email: %v", err)
	}

  log.Println("Email enviado com sucesso")
}
