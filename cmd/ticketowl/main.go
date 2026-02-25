package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/wisbric/ticketowl/internal/app"
	"github.com/wisbric/ticketowl/internal/config"
)

func main() {
	mode := flag.String("mode", "", "run mode: api, worker, seed, seed-demo, migrate (overrides TICKETOWL_MODE)")
	flag.Parse()

	cfg, err := config.Load()
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
