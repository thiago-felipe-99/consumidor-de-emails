package main

import (
	"context"
	"errors"
	"fmt"

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
		Shards:             configuracoes.cache.shards,
		LifeWindow:         configuracoes.cache.lifeWindow,
		CleanWindow:        configuracoes.cache.cleanWindow,
		MaxEntriesInWindow: configuracoes.cache.avgEntries,
		MaxEntrySize:       configuracoes.cache.avgEntrySize,
		HardMaxCacheSize:   configuracoes.cache.maxSize,
		StatsEnabled:       configuracoes.cache.statics,
		Verbose:            configuracoes.cache.verbose,
	}

	data, err := bigcache.New(context.Background(), dataConfig)
	if err != nil {
		return nil, err
	}

	host := fmt.Sprintf("%s:%d", configuracoes.minio.host, configuracoes.minio.porta)
	minioOptions := &minio.Options{
		Creds: credentials.NewStaticV4(
			configuracoes.minio.accesKey,
			configuracoes.minio.secrectKey,
			"",
		),
	}

	minio, err := minio.New(host, minioOptions)
	if err != nil {
		return nil, err
	}

	return &cache{
		data:   data,
		bucket: configuracoes.minio.bucket,
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
