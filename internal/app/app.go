package app

import (
	"context"
	"log/slog"

	"github.com/aminshahid573/taskmanager/internal/config"
)

func Run(cfg *config.Config, logger *slog.Logger) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	//track cleanup functions (LIFO order)
	cleanupFuncs := make([]func() error,0)
	defer func() {
		//execute cleanup in reverse order
		for i := len(cleanupFuncs) - 1; i > 0; i-- {
			if err := cleanupFuncs[i](); err != nil {
				slog.Error("Cleanup failed", "error", err)
			}
		}
	}()

	return nil
}
