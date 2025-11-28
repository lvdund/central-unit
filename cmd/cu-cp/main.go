package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"central-unit/internal/app"
)

func main() {
	cfgPath := flag.String("config", "config/config.yml", "path to YAML configuration")
	flag.Parse()

	// Create app - it will load config internally
	svc, err := app.New(*cfgPath)
	if err != nil {
		exit(fmt.Errorf("create app: %w", err))
	}

	// Start the application
	if err := svc.Start(); err != nil {
		exit(fmt.Errorf("start service: %w", err))
	}

	waitForShutdown(svc)
}

func waitForShutdown(svc *app.App) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	<-stop
	fmt.Println("shutdown signal received")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := svc.Stop(ctx); err != nil {
		fmt.Printf("graceful stop failed: %v\n", err)
		return
	}
	fmt.Println("central-unit stopped")
}

func exit(err error) {
	fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
	os.Exit(1)
}
