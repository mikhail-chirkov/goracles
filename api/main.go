package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	cfg, cfgError := loadConfig()
	if cfgError != nil {
		log.Fatalf("Failed to load the configuration with error %v", cfgError)
	}

	weatherClient := newWeatherClient(cfg.BrightskyURL)

	server := newServer(weatherClient)
	http.Handle("/predict", handlerMiddleware(server.predictHandler))

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *cfg.Port), nil))
}
