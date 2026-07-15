package main

import (
	"log"

	"github.com/pjpangilinan/muse-journ/internal/app"
	"github.com/pjpangilinan/muse-journ/internal/config"
)

func main() {
	cfg := config.Load()
	if err := app.RunCollector(cfg); err != nil {
		log.Fatal(err)
	}
}
