package main

import (
	"errors"
	"os"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/knadh/koanf/parsers/dotenv"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/structs"
	"github.com/knadh/koanf/v2"
)

type sender struct {
	Name  string `config:"name"  validate:"required"`
	Email string `config:"email" validate:"required"`
}

type smtp struct {
	User     string `config:"user"     validate:"required"`
	Password string `config:"password" validate:"required"`
	Host     string `config:"host"     validate:"required"`
	Port     int    `config:"port"     validate:"required"`
}

type rabbit struct {
	User     string `config:"user"      validate:"required"`
	Password string `config:"password"  validate:"required"`
	Host     string `config:"host"      validate:"required"`
	Port     int    `config:"port"      validate:"required"`
	Vhost    string `config:"vhost"     validate:"required"`
	Queue    string `config:"queue"     validate:"required"`
	MaxRetry int    `config:"max_retry" validate:"required"`
}

type buffer struct {
	Size     int `config:"size"     validate:"required"`
	Quantity int `config:"quantity" validate:"required"`
}

type cacheConfig struct {
	Shards       int  `config:"shards"         validate:"required"`
	LifeWindow   int  `config:"life_window"    validate:"required"`
	CleanWindow  int  `config:"clean_window"   validate:"required"`
	AvgEntries   int  `config:"avg_entries"    validate:"required"`
	AvgEntrySize int  `config:"avg_entry_size" validate:"required"`
	MaxEntrySize int  `config:"max_entry_size" validate:"required"`
	MaxSize      int  `config:"max_size"       validate:"required"`
	Statics      bool `config:"statics"`
	Verbose      bool `config:"verbose"`
}

type minioConfig struct {
	Host      string `config:"host"       validate:"required"`
	Port      int    `config:"port"       validate:"required"`
	Bucket    string `config:"bucket"     validate:"required"`
	AccessKey string `config:"access_key" validate:"required"`
	SecretKey string `config:"secret_key" validate:"required"`
	Secure    bool   `config:"secure"`
}

type configurations struct {
	Sender  sender      `config:"sender"  validate:"required"`
	SMTP    smtp        `config:"smtp"    validate:"required"`
	Rabbit  rabbit      `config:"rabbit"  validate:"required"`
	Buffer  buffer      `config:"buffer"  validate:"required"`
	Timeout int         `config:"timeout" validate:"required"`
	Cache   cacheConfig `config:"cache"   validate:"required"`
	Minio   minioConfig `config:"minio"   validate:"required"`
}

//nolint:gomnd
func defaultConfigurations() configurations {
	return configurations{
		SMTP: smtp{
			Port: 587,
		},
		Rabbit: rabbit{
			Port:     5672,
			Vhost:    "/",
			MaxRetry: 4,
		},
		Buffer: buffer{
			Size:     100,
			Quantity: 10,
		},
		Cache: cacheConfig{
			Shards:       64,
			LifeWindow:   60,
			CleanWindow:  5,
			AvgEntries:   10,
			AvgEntrySize: 10,
			MaxEntrySize: 25,
			MaxSize:      1000,
			Statics:      false,
			Verbose:      false,
		},
		Minio: minioConfig{
			Port:   9000,
			Secure: true,
		},
		Timeout: 2,
	}
}

func parseEnv(env string) string {
	keys := strings.SplitN(env, "_", 2) //nolint:gomnd
	size := len(keys)

	var key string

	switch size {
	case 0:
		return ""
	case 1:
		key = keys[0]
	default:
		key = keys[0] + "__" + strings.Join(keys[1:], "_")
	}

	return strings.ToLower(key)
}

func getConfigurations() (*configurations, error) {
	koanfConfig := koanf.Conf{
		Delim:       "__",
		StrictMerge: false,
	}

	configRaw := koanf.NewWithConf(koanfConfig)

	err := configRaw.Load(structs.Provider(defaultConfigurations(), "config"), nil)
	if err != nil {
		return nil, err
	}

	err = configRaw.Load(file.Provider(".env"), dotenv.ParserEnv("", "__", parseEnv))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	err = configRaw.Load(env.Provider("", "__", parseEnv), nil)
	if err != nil {
		return nil, err
	}

	config := &configurations{}

	err = configRaw.UnmarshalWithConf("", config, koanf.UnmarshalConf{
		Tag:       "config",
		FlatPaths: false,
	})
	if err != nil {
		return nil, err
	}

	validate := validator.New()

	return config, validate.Struct(config)
}
