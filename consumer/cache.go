package main

import (
	"context"
	"errors"
	"fmt"
	"io"
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

func newCache(configs *configurations) (*cache, error) {
	dataConfig := bigcache.Config{
		Shards:             configs.Cache.Shards,
		LifeWindow:         time.Duration(configs.Cache.LifeWindow) * time.Minute,
		CleanWindow:        time.Duration(configs.Cache.CleanWindow) * time.Minute,
		MaxEntriesInWindow: configs.Cache.AvgEntries,
		MaxEntrySize:       configs.Cache.AvgEntrySize,
		HardMaxCacheSize:   configs.Cache.MaxSize,
		StatsEnabled:       configs.Cache.Statics,
		Verbose:            configs.Cache.Verbose,
	}

	data, err := bigcache.New(context.Background(), dataConfig)
	if err != nil {
		return nil, err
	}

	host := fmt.Sprintf("%s:%d", configs.Minio.Host, configs.Minio.Port)
	minioOptions := &minio.Options{
		Creds: credentials.NewStaticV4(
			configs.Minio.AccessKey,
			configs.Minio.SecretKey,
			"",
		),
	}

	minio, err := minio.New(host, minioOptions)
	if err != nil {
		return nil, err
	}

	return &cache{
		data:   data,
		bucket: configs.Minio.Bucket,
		minio:  minio,
	}, nil
}

func (cache *cache) getFileFromMinio(name string) ([]byte, error) {
	object, err := cache.minio.GetObject(
		context.Background(),
		cache.bucket,
		name,
		minio.GetObjectOptions{},
	)
	if err != nil {
		return nil, err
	}

	objectInfo, err := object.Stat()
	if err != nil {
		return nil, err
	}

	file := make([]byte, objectInfo.Size)

	_, err = object.Read(file)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}

	return file, cache.data.Set(name, file)
}

func (cache *cache) getFile(name string) ([]byte, error) {
	file, err := cache.data.Get(name)
	if err != nil {
		if errors.Is(err, bigcache.ErrEntryNotFound) {
			return cache.getFileFromMinio(name)
		}

		return nil, err
	}

	return file, nil
}
