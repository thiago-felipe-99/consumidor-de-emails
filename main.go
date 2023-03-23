package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type remetente struct {
	email, senha, host, porta string
}

type configuracoes struct {
	remetente
}

func pegarConfiguracoes() (*configuracoes, error) {
	err := godotenv.Load()

	config := &configuracoes{
		remetente: remetente{
			email: os.Getenv("SMTP_USER"),
			senha: os.Getenv("SMTP_PASSWORD"),
			host:  os.Getenv("SMTP_HOST"),
			porta: os.Getenv("SMTP_PORT"),
		},
	}

	return config, err
}

func main() {
	config, err := pegarConfiguracoes()
	if err != nil {
		log.Fatalf("Erro ao ler as configurações: %v", err)
	}

	log.Println(config)
}
