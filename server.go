package main

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// handleSignals is responsible for handling the Linux OS termination signals.
// This is necessary to gracefully exit from the server. This is done by
// passing the cancel callback function which signals services to close.
func handleSignals(cancel context.CancelFunc, signalChan chan os.Signal) {
	signal.Notify(
		signalChan,
		syscall.SIGHUP,  // kill -SIGHUP XXXX
		syscall.SIGINT,  // kill -SIGINT XXXX or Ctrl+c
		syscall.SIGQUIT, // kill -SIGQUIT XXXX
		syscall.SIGTERM, // kill -SIGTERM XXXX
	)
	defer signal.Reset()

	// Block until signal is received.
	<-signalChan
	//log.Print("signal: os.Interrupt - shutting down...\n")

	// Notify the server to shutdown.
	cancel()
}

// Starts an http server on the given address [HOST][PORT]. Can be shut down
// with a context with Cancel safely.
func StartHTTPServer(addr string, ctx context.Context) error {
	shutdownChan := make(chan bool, 1)

	server := &http.Server{
		Addr:              addr,
		BaseContext:       func(_ net.Listener) context.Context { return ctx },
		Handler:           http.DefaultServeMux,
		ReadTimeout:       1 * time.Second,
		WriteTimeout:      1 * time.Second,
		IdleTimeout:       30 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
	}

	go func() {
		if err := server.ListenAndServeTLS(CertFile, KeyFile); !errors.Is(err, http.ErrServerClosed) {
			log.Println("http:", err)
		}

		shutdownChan <- true
	}()

	<-ctx.Done()

	ctxShutdown, shutdownRelease := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownRelease()

	err := server.Shutdown(ctxShutdown)
	if errors.Is(err, http.ErrServerClosed) {
		err = nil
	}

	<-shutdownChan
	//log.Println("http:", "Server gracefully shut down.")

	return err
}
