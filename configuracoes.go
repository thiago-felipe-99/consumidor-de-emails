package main

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type remetente struct {
	nome, email, senha, host string
	porta                    int
}

type rabbit struct {
	user, senha, host, vhost, fila string
	porta                          int
}

type buffer struct {
	tamanho, quantidade int
}

type cacheConfig struct {
	shards                   int
	lifeWindow, cleanWindow  time.Duration
	avgEntries, avgEntrySize int
	maxSize                  int
	statics, verbose         bool
}

type minioConfig struct {
	host                 string
	porta                int
	bucket               string
	accesKey, secrectKey string
	secure               bool
}

type configuracoes struct {
	remetente
	rabbit
	buffer
	timeout time.Duration
	cache   cacheConfig
	minio   minioConfig
}

func pegarConfiguracoes() (*configuracoes, error) {
	err := godotenv.Load()
	if err != nil {
		return nil, err
	}

	smtpPorta, err := strconv.Atoi(os.Getenv("SMTP_PORT"))
	if err != nil {
		return nil, err
	}

	rabbitPorta, err := strconv.Atoi(os.Getenv("RABBIT_PORT"))
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

	cacheShards, err := strconv.Atoi(os.Getenv(("CACHE_SHARDS")))
	if err != nil {
		return nil, err
	}

	cacheLifeWindowMinute, err := strconv.Atoi(os.Getenv(("CACHE_LIFE_WINDOW_MINUTE")))
	if err != nil {
		return nil, err
	}

	cacheCleanWindowMinute, err := strconv.Atoi(os.Getenv(("CACHE_CLEAN_WINDOW_MINUTE")))
	if err != nil {
		return nil, err
	}

	cacheAvgEntriesInWindow, err := strconv.Atoi(os.Getenv(("CACHE_AVG_ENTRIES_IN_WINDOW")))
	if err != nil {
		return nil, err
	}

	cacheAvgEntrySizeMB, err := strconv.Atoi(os.Getenv(("CACHE_AVG_ENTRY_SIZE_MB")))
	if err != nil {
		return nil, err
	}

	cacheMaxSizeMB, err := strconv.Atoi(os.Getenv(("CACHE_MAX_SIZE_MB")))
	if err != nil {
		return nil, err
	}

	minioPort, err := strconv.Atoi(os.Getenv(("MINIO_PORT")))
	if err != nil {
		return nil, err
	}

	config := &configuracoes{
		remetente: remetente{
			nome:  os.Getenv("SMTP_USERNAME"),
			email: os.Getenv("SMTP_USER"),
			senha: os.Getenv("SMTP_PASSWORD"),
			host:  os.Getenv("SMTP_HOST"),
			porta: smtpPorta,
		},
		rabbit: rabbit{
			user:  os.Getenv("RABBIT_USER"),
			senha: os.Getenv("RABBIT_PASSWORD"),
			host:  os.Getenv("RABBIT_HOST"),
			porta: rabbitPorta,
			vhost: os.Getenv("RABBIT_VHOST"),
			fila:  os.Getenv("RABBIT_QUEUE"),
		},
		buffer: buffer{
			tamanho:    bufferSize,
			quantidade: bufferQT,
		},
		timeout: time.Duration(timeoutSegundos) * time.Second,
		cache: cacheConfig{
			shards:       cacheShards,
			lifeWindow:   time.Duration(cacheLifeWindowMinute) * time.Minute,
			cleanWindow:  time.Duration(cacheCleanWindowMinute) * time.Minute,
			avgEntries:   cacheAvgEntriesInWindow,
			avgEntrySize: cacheAvgEntrySizeMB,
			maxSize:      cacheMaxSizeMB,
			statics:      os.Getenv("CACHE_STATICS_ENABLE") == "true",
			verbose:      os.Getenv("CACHE_VERBOSE") == "true",
		},
		minio: minioConfig{
			host:       os.Getenv("MINIO_HOSTNAME"),
			porta:      minioPort,
			bucket:     os.Getenv("MINIO_BUCKET"),
			accesKey:   os.Getenv("MINIO_ACCESS_KEY"),
			secrectKey: os.Getenv("MINIO_SECRETE_KEY"),
			secure:     os.Getenv("MINIO_USE_SSL") == "true",
		},
	}

	return config, nil
}
