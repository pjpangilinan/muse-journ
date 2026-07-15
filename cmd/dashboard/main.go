package main

import (
	"embed"
	"html/template"
	"log"

	"github.com/pjpangilinan/muse-journ/internal/app"
	"github.com/pjpangilinan/muse-journ/internal/config"
)

//go:embed templates/*
var templateFS embed.FS

func main() {
	cfg := config.Load()

	tmpl := template.Must(template.New("").Funcs(app.TemplateFuncs).ParseFS(templateFS, "templates/*.html"))

	if err := app.RunDashboard(cfg, tmpl); err != nil {
		log.Fatal(err)
	}
}
