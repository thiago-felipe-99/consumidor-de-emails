package main

import (
	"errors"
	"os"
	"strings"

	"github.com/knadh/koanf/parsers/dotenv"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/structs"
	"github.com/knadh/koanf/v2"
)

type sender struct {
	Name  string `config:"name"`
	Email string `config:"email"`
}

type smtp struct {
	User     string `config:"user"`
	Password string `config:"password"`
	Host     string `config:"host"`
	Port     int    `config:"port"`
}

type rabbit struct {
	User     string `config:"user"`
	Password string `config:"password"`
	Host     string `config:"host"`
	Port     int    `config:"port"`
	Vhost    string `config:"vhost"`
	Queue    string `config:"queue"`
}

type buffer struct {
	Size     int `config:"size"`
	Quantity int `config:"quantity"`
}

type cacheConfig struct {
	Shards       int  `config:"shards"`
	LifeWindow   int  `config:"life_window"`
	CleanWindow  int  `config:"clean_window"`
	AvgEntries   int  `config:"avg_entries"`
	AvgEntrySize int  `config:"avg_entry_size"`
	MaxSize      int  `config:"maxsize"`
	Statics      bool `config:"statics"`
	Verbose      bool `config:"verbose"`
}

type minioConfig struct {
	Host       string `config:"host"`
	Port       int    `config:"port"`
	Bucket     string `config:"bucket"`
	AccessKey  string `config:"access_key"`
	SecrectKey string `config:"secrect_key"`
	Secure     bool   `config:"secure"`
}

type configuracoes struct {
	Sender  sender      `config:"sender"`
	SMTP    smtp        `config:"smtp"`
	Rabbit  rabbit      `config:"rabbit"`
	Buffer  buffer      `config:"buffer"`
	Timeout int         `config:"timeout"`
	Cache   cacheConfig `config:"cache"`
	Minio   minioConfig `config:"minio"`
}

//nolint:gomnd
func configuracoesPadroes() configuracoes {
	return configuracoes{
		SMTP: smtp{
			Port: 587,
		},
		Rabbit: rabbit{
			Port:  5672,
			Vhost: "/",
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
			AvgEntrySize: 25,
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
	key := strings.SplitN(env, "_", 2) //nolint:gomnd
	size := len(key)

	var final string

	switch size {
	case 0:
		return ""
	case 1:
		final = key[0]
	default:
		final = key[0] + "__" + strings.Join(key[1:], "_")
	}

	return strings.ToLower(final)
}

func pegarConfiguracoes() (*configuracoes, error) {
	koanfConfig := koanf.Conf{
		Delim:       "__",
		StrictMerge: false,
	}

	configRaw := koanf.NewWithConf(koanfConfig)

	err := configRaw.Load(structs.Provider(configuracoesPadroes(), "config"), nil)
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

	config := &configuracoes{}

	err = configRaw.UnmarshalWithConf("", config, koanf.UnmarshalConf{
		Tag:       "config",
		FlatPaths: false,
	})
	if err != nil {
		return nil, err
	}

	return config, nil
}
