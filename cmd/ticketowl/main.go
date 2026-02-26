package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	coreconfig "github.com/wisbric/core/pkg/config"

	"github.com/wisbric/ticketowl/internal/app"
	"github.com/wisbric/ticketowl/internal/config"
)

func main() {
	mode := flag.String("mode", "", "run mode: api, worker, seed, seed-demo, migrate (overrides APP_MODE)")
	flag.Parse()

	cfg, err := coreconfig.Load[config.Config]()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: loading config: %v\n", err)
		os.Exit(1)
	}

	// CLI flag overrides env var.
	if *mode != "" {
		cfg.Mode = *mode
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := app.Run(ctx, cfg); err != nil {
		slog.Error("fatal", "error", err)
		os.Exit(1)
	}
}
