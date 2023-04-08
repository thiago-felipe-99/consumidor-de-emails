package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/allegro/bigcache/v3"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type cache struct {
	data   *bigcache.BigCache
	bucket string
	minio  *minio.Client
}

func novoCache(configuracoes *configuracoes) (*cache, error) {
	dataConfig := bigcache.Config{
		Shards:             configuracoes.Cache.Shards,
		LifeWindow:         time.Duration(configuracoes.Cache.LifeWindow) * time.Minute,
		CleanWindow:        time.Duration(configuracoes.Cache.CleanWindow) * time.Minute,
		MaxEntriesInWindow: configuracoes.Cache.AvgEntries,
		MaxEntrySize:       configuracoes.Cache.AvgEntrySize,
		HardMaxCacheSize:   configuracoes.Cache.MaxSize,
		StatsEnabled:       configuracoes.Cache.Statics,
		Verbose:            configuracoes.Cache.Verbose,
	}

	data, err := bigcache.New(context.Background(), dataConfig)
	if err != nil {
		return nil, err
	}

	host := fmt.Sprintf("%s:%d", configuracoes.Minio.Host, configuracoes.Minio.Port)
	minioOptions := &minio.Options{
		Creds: credentials.NewStaticV4(
			configuracoes.Minio.AccessKey,
			configuracoes.Minio.SecretKey,
			"",
		),
	}

	minio, err := minio.New(host, minioOptions)
	if err != nil {
		return nil, err
	}

	return &cache{
		data:   data,
		bucket: configuracoes.Minio.Bucket,
		minio:  minio,
	}, nil
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

func (cache *cache) PegarArqivo(nome string) ([]byte, error) {
	arquivo, err := cache.data.Get(nome)
	if err != nil {
		if errors.Is(err, bigcache.ErrEntryNotFound) {
			return cache.salvarArquivo(nome)
		}

		return nil, err
	}

	return arquivo, nil
}
