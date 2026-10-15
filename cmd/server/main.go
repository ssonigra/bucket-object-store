package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"bucket-object-store/internal/httpapi"
	"bucket-object-store/internal/store"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		fmt.Print("What port do you want to use? (e.g. 8080): ")
		if _, err := fmt.Fscan(os.Stdin, &port); err != nil {
			log.Fatalf("could not read port: %v", err)
		}
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		log.Fatalf("PORT must be a number between 1 and 65535; got %q", port)
	}
	fmt.Printf("You selected port %d.\n", portNumber)

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", portNumber),
		Handler:           httpapi.NewHandler(store.New()),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("object service listening on port %d", portNumber)
		serverErrors <- server.ListenAndServe()
	}()

	shutdownSignal, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-serverErrors:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP server stopped: %v", err)
		}
	case <-shutdownSignal.Done():
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("graceful shutdown failed: %v", err)
			_ = server.Close()
		}
	}
}
