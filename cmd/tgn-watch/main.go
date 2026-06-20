package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/m0zgen/tgn-watch/internal/config"
	"github.com/m0zgen/tgn-watch/internal/runner"
	"github.com/m0zgen/tgn-watch/internal/version"
)

func main() {
	configPath := flag.String("config", "configs/config.example.yml", "Path to config file")
	showVersion := flag.Bool("version", false, "Print version")
	flag.Parse()

	if *showVersion {
		fmt.Printf("tgn-watch %s commit=%s date=%s\n", version.Version, version.Commit, version.Date)
		return
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("config load failed: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	r := runner.New(cfg)
	if err := r.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatalf("runner stopped with error: %v", err)
	}
}
