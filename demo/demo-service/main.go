package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	config, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("starting demo service on :%s in %s mode", config.Port, config.HealthMode)
	if err := http.ListenAndServe(":"+config.Port, newHandler(config)); err != nil {
		log.Fatal(err)
	}
}

func hostname() string {
	value, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return value
}
