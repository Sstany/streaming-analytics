package config

import (
	srvconf "streaming/api/internal/server/config"
)

type Config struct {
	Server   *srvconf.ServerConfig `yaml:"server"`
	Postgres *postgres.Config      `yaml:"postgres"`
}

func MustNew(path string) *Config {
	cfg := &Config{}
	if err := cleanenv.ReadConfig(path, cfg); err != nil {
		panic(err)
	}
	return cfg
}
