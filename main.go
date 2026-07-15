package main

import (
	"embed"
	"html/template"
	"log"
	"os"

	"github.com/pjpangilinan/muse-journ/internal/app"
	"github.com/pjpangilinan/muse-journ/internal/config"
)

//go:embed cmd/dashboard/templates/*
var templateFS embed.FS

func main() {
	cfg := config.Load()

	if len(os.Args) < 2 {
		log.Fatal("usage: muse-journ <collector|dashboard|build-site>")
	}

	tmpl := template.Must(template.New("").Funcs(app.TemplateFuncs).ParseFS(templateFS, "cmd/dashboard/templates/*.html"))

	var err error
	switch os.Args[1] {
	case "collector":
		err = app.RunCollector(cfg)
	case "dashboard":
		err = app.RunDashboard(cfg, tmpl)
	case "build-site":
		err = app.BuildSite(cfg, tmpl)
	default:
		log.Fatalf("unknown command: %s", os.Args[1])
	}
	if err != nil {
		log.Fatal(err)
	}
}
