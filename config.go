package main

import (
	"errors"
	"os"

	"github.com/goccy/go-yaml"
)

type Config struct {
	Key     string `yaml:"key"`
	Account string `yaml:"account"`
	Zone    string `yaml:"zone"`
	Record  string `yaml:"record"`
}

func LoadConfig() (*Config, error) {
	file, err := os.OpenFile("config.yml", os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	var cfg Config

	err = yaml.NewDecoder(file).Decode(&cfg)
	if err != nil {
		return nil, err
	}

	if cfg.Key == "" {
		return nil, errors.New("missing key")
	} else if cfg.Account == "" {
		return nil, errors.New("missing account")
	} else if cfg.Zone == "" {
		return nil, errors.New("missing zone")
	} else if cfg.Record == "" {
		return nil, errors.New("missing record")
	}

	return &cfg, nil
}
