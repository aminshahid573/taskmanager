package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/aminshahid573/taskmanager/internal/app"
	"github.com/aminshahid573/taskmanager/internal/config"
)

func main() {
	// Load configuration
	configPath := flag.String("config", "config/local.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}
	
	//setup structured looging
	logger := app.NewLogger(cfg.Log.Level, cfg.Log.Format) 
	slog.SetDefault(logger)

	slog.Info("Starting application",
		"env", cfg.App.Environment,
		"version",cfg.App.Version,
		)

}
