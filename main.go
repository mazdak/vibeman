package main

//go:generate swag init -g main.go -o docs

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"vibeman/internal/app"
)

func main() {
	// Create context that cancels on interrupt
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	// Create and run app
	application := app.New()
	if err := application.RunWithContext(ctx, os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
