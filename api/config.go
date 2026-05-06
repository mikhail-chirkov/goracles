package main

import (
	"net/url"

	"github.com/caarlos0/env/v11"
)

type config struct {
	BrightskyURL *url.URL `env:"BRIGHTSKY_API_URL" envDefault:"https://api.brightsky.dev"`
	Port         *uint16  `env:"PORT" envDefault:"8080"`
}

func loadConfig() (*config, error) {
	cfg := &config{}

	// load the config
	err := env.Parse(cfg)
	if err != nil {
		return &config{}, err
	}

	return cfg, nil
}
