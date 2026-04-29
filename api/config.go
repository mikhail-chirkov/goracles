package main

import (
	"fmt"
	"net/url"

	"github.com/caarlos0/env/v11"
)

type config struct {
	BrightskyURL *url.URL `env:"BRIGHTSKY_API_URL"`
	Port         *uint16  `env:"PORT"`
}

func loadConfig() (*config, error) {
	cfg := &config{}

	// load the config
	err := env.Parse(cfg)
	if err != nil {
		return &config{}, err
	}

	// validate
	if cfg.BrightskyURL == nil {
		return &config{}, fmt.Errorf("BRIGHTSKY_API_URL is required to be set")
	}

	if cfg.Port == nil {
		return &config{}, fmt.Errorf("PORT is required to be set")
	}

	return cfg, nil
}
