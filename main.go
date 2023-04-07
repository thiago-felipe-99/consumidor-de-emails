package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/allegro/bigcache/v3"
	"github.com/joho/godotenv"
	"github.com/minio/minio-go/v7"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/wneessen/go-mail"
)

type cache struct {
	data   *bigcache.BigCache
	bucket string
	minio  *minio.Client
}

func novoCache() (*cache, error) {
	return nil, nil
}

func (cache *cache) salvarArquivo(nome string) ([]byte, error) {
	objeto, err := cache.minio.GetObject(
		context.Background(),
		cache.bucket,
		nome,
		minio.GetObjectOptions{},
	)
	if err != nil {
		return nil, err
	}

	objetoInfo, err := objeto.Stat()
	if err != nil {
		return nil, err
	}

	arquivo := make([]byte, objetoInfo.Size)

	_, err = objeto.Read(arquivo)
	if err != nil {
		return nil, err
	}

	return arquivo, cache.data.Set(nome, arquivo)
}

func (cache *cache) pegarArqivo(nome string) ([]byte, error) {
	arquivo, err := cache.data.Get(nome)
	if err != nil {
		if errors.Is(err, bigcache.ErrEntryNotFound) {
			return cache.salvarArquivo(nome)
		}

		return nil, err
	}

	return arquivo, nil
}

type remetente struct {
	nome, email, senha, host string
	porta                    int
}

type rabbit struct {
	user, senha, host, porta, vhost, fila string
}

type buffer struct {
	tamanho, quantidade int
}

type configuracoes struct {
	remetente
	rabbit
	buffer
	timeoutSegundos time.Duration
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

	bufferSize, err := strconv.Atoi(os.Getenv("BUFFER_SIZE"))
	if err != nil {
		return nil, err
	}

	bufferQT, err := strconv.Atoi(os.Getenv("BUFFER_QT"))
	if err != nil {
		return nil, err
	}

	timeoutSegundos, err := strconv.Atoi(os.Getenv(("TIMEOUT_SECONDS")))
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
		buffer: buffer{
			tamanho:    bufferSize,
			quantidade: bufferQT,
		},
		timeoutSegundos: time.Duration(timeoutSegundos) * time.Second,
	}

	return config, nil
}

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

type metricas struct {
	emailsRecebidos            prometheus.Counter
	emailsRecebidosBytes       prometheus.Counter
	emailsEnviados             prometheus.Counter
	emailsEnviadosBytes        prometheus.Counter
	emailsAnexosEnviados       prometheus.Counter
	emailsAnexosEnviadosBytes  prometheus.Counter
	emailsEnviadosComAnexo     prometheus.Counter
	emailsReenviados           prometheus.Counter
	emailsTempoDeEnvioSegundos prometheus.Histogram
	emailsCacheAnexos          prometheus.Gauge
	emailsCacheAnexosBytes     prometheus.Gauge
}

func criarMetricas() *metricas {
	return &metricas{
		emailsRecebidos: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "emails_recebidos",
			Help: "A quantidade de emails recebidos pela fila do rabbit",
		}),
		emailsRecebidosBytes: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "emails_recebidos_bytes",
			Help: "A quantidade em bytes de emails recebidos pela fila do rebbit",
		}),
		emailsEnviados: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "emails_enviados",
			Help: "A quantidade de emails enviados com sucesso",
		}),
		emailsEnviadosBytes: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "emails_enviados_bytes",
			Help: "A quantidade em bytes de emails enviados com sucesso",
		}),
		emailsAnexosEnviados: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "emails_anexos_enviados",
			Help: "A quantidade de anexos enviados",
		}),
		emailsAnexosEnviadosBytes: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "emails_anexos_enviados_bytes",
			Help: "A quantidade em bytes de anexos enviados com sucesso",
		}),
		emailsEnviadosComAnexo: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "emails_enviados_com_anexo",
			Help: "A quantidade de emails enviados com sucesso e com anexo",
		}),
		emailsReenviados: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "emails_reenviados",
			Help: "A quantidade de emails reeenviados para a fila do rabbit",
		}),
		emailsTempoDeEnvioSegundos: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name: "emails_tempo_de_envio_segundos",
			Help: "O tempo de envio de lotes de emails em segundos",
		}),
		emailsCacheAnexos: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "emails_cache_anexo",
			Help: "A quantidade de anexos no cache",
		}),
		emailsCacheAnexosBytes: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "emails_cache_anexo_bytes",
			Help: "A quantidade em bytes de anexos no cache",
		}),
	}
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

func main() {
	configs, err := pegarConfiguracoes()
	if err != nil {
		log.Fatalf("[ERRO] - Erro ao ler as configurações: %s", err)
	}

	cache, err := novoCache()
	if err != nil {
		log.Fatalf("[ERRO] - Erro ao criar o cache de arquivos: %s", err)
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
		timeout := time.NewTicker(configs.timeoutSegundos)
		enviar := novoEnviar(cache, &configs.remetente, metricas)

		for {
			select {
			case mensagen := <-fila:
				bufferFila = append(bufferFila, mensagen)
				timeout.Reset(configs.timeoutSegundos)

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
