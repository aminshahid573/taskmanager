package main

import (
	"flag"
	"fmt"
	"os"

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
	fmt.Printf("Loaded config: %+v\n", cfg)
}
