package main

import (
	"net/url"

	"github.com/caarlos0/env/v11"
	"github.com/go-playground/validator/v10"
)

type config struct {
	BotToken     string   `env:"BOT_TOKEN" validate:"required"`
	APIURL       *url.URL `env:"API_URL" envDefault:"http://localhost:8080"`
	DatabasePath string   `env:"DATABASE_PATH" envDefault:"goracles-bot.sqlite"`
}

func (c *config) validate() error {
	validate := validator.New(validator.WithRequiredStructEnabled())

	return validate.Struct(c)
}

func loadConfig() (*config, error) {
	cfg := &config{}

	// load the config
	err := env.Parse(cfg)
	if err != nil {
		return &config{}, err
	}

	err = cfg.validate()
	if err != nil {
		return &config{}, err
	}

	return cfg, nil
}
