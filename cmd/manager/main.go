package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/seedspirit/nano-backend.ai/internal/manager"
)

const (
	defaultAddr   = ":8090"
	defaultDBPath = "manager.db"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	ctx := context.Background()
	if err := run(ctx); err != nil {
		slog.Error("manager failed", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	app, err := manager.NewApp(ctx, manager.Args{
		Addr:   defaultAddr,
		DBPath: defaultDBPath,
	})
	if err != nil {
		return err
	}
	defer app.Close()

	slog.Info("manager starting", "addr", defaultAddr)
	return app.Start(ctx)
}
