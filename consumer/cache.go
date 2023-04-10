package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/allegro/bigcache/v3"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var errMaxEntrySize = fmt.Errorf("entry is to big")

type cache struct {
	data         *bigcache.BigCache
	bucket       string
	minio        *minio.Client
	maxEntrySize int64
}

func newCache(configs *configurations) (*cache, error) {
	const megabyte = 1000 * 1000

	dataConfig := bigcache.Config{
		Shards:             configs.Cache.Shards,
		LifeWindow:         time.Duration(configs.Cache.LifeWindow) * time.Minute,
		CleanWindow:        time.Duration(configs.Cache.CleanWindow) * time.Minute,
		MaxEntriesInWindow: configs.Cache.AvgEntries,
		MaxEntrySize:       configs.Cache.AvgEntrySize * megabyte,
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
		data:         data,
		bucket:       configs.Cache.Bucket,
		minio:        minio,
		maxEntrySize: int64(configs.Cache.MaxEntrySize) * megabyte,
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

	if objectInfo.Size > cache.maxEntrySize {
		return nil, errMaxEntrySize
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

type template struct {
	cache
}

func newTemplate(configs *configurations) (*template, error) {
	const megabyte = 1000 * 1000

	dataConfig := bigcache.Config{
		Shards:             configs.Template.Shards,
		LifeWindow:         0,
		CleanWindow:        0,
		MaxEntriesInWindow: configs.Template.AvgEntries,
		MaxEntrySize:       configs.Template.AvgEntrySize * megabyte,
		HardMaxCacheSize:   configs.Template.MaxSize,
		StatsEnabled:       configs.Template.Statics,
		Verbose:            configs.Template.Verbose,
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

	template := &template{
		cache: cache{
			data:         data,
			bucket:       configs.Template.Bucket,
			minio:        minio,
			maxEntrySize: int64(configs.Template.MaxEntrySize) * megabyte,
		},
	}

	return template, nil
}

func (template *template) setAll() {
	options := minio.ListObjectsOptions{
		WithVersions: false,
		WithMetadata: true,
		Prefix:       "",
		Recursive:    true,
	}

  templatesQuantity := 0

	for info := range template.cache.minio.ListObjects(context.Background(), template.bucket, options) {
		if info.Err != nil {
			log.Printf("[ERROR] - Error getting '%s' template info: %s", info.Key, info.Err)

			continue
		}

		object, err := template.cache.minio.GetObject(
			context.Background(),
			template.bucket,
			info.Key,
			minio.GetObjectOptions{},
		)
		if err != nil {
			log.Printf("[ERROR] - Error getting template: %s", err)

			continue
		}

    // Need to get a new status because ListObjects doesn't return ContentType
    // https://github.com/minio/minio-go/issues/1593
		status, err := object.Stat()
		if err != nil {
			log.Printf("[ERROR] - Error getting '%s' template status: %s", info.Key, err)

			continue
		}

		if status.Size > template.maxEntrySize {
			log.Printf("[ERROR] - '%s' template is to big: %dMB", info.Key, (info.Size)/1000/1000)

			continue
		}

		contentType := status.ContentType
		if contentType != "text/plain" && contentType != "text/html" {
			log.Printf(
				"[ERROR] - '%s' template has a invalid Content Type: %s",
				info.Key,
				info.ContentType,
			)

			continue
		}

		buffer := make([]byte, info.Size)

		_, err = object.Read(buffer)
		if err != nil && !errors.Is(err, io.EOF) {
			log.Printf("[ERROR] - Error reading template: %s", err)

			continue
		} else if err == nil {
			log.Printf("[ERROR] - Unable to get all template")

			continue
		}

		err = template.cache.data.Set(info.Key, buffer)
		if err != nil {
			log.Printf("[ERROR] - Error setting template: %s", err)

			continue
		}

    templatesQuantity++
	}

  log.Printf("[INFO] - %d templates on cache", templatesQuantity)
}

func (template *template) get(name string) ([]byte, error) {
	return template.cache.getFile(name)
}
