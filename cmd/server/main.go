package main

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/juho05/crossonic-server/config"
	"github.com/juho05/crossonic-server/handlers"
	"github.com/juho05/log"
)

func run() error {
	handler := handlers.New()

	addr := config.ListenAddr()

	server := http.Server{
		Addr:     addr,
		Handler:  handler,
		ErrorLog: log.NewStdLogger(log.ERROR),
		TLSConfig: &tls.Config{
			MinVersion:       tls.VersionTLS13,
			CurvePreferences: []tls.CurveID{tls.CurveP256, tls.X25519},
			CipherSuites: []uint16{
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
				tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			},
		},
	}

	closed := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, syscall.SIGINT, syscall.SIGTERM)
		<-sigint
		timeout, cancelTimeout := context.WithTimeout(context.Background(), 5*time.Second)
		log.Info("Shutting down...")
		server.Shutdown(timeout)
		cancelTimeout()
		close(closed)
	}()

	log.Infof("Listening on http://%s...", addr)
	err := server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		err = nil
	}
	if err == nil {
		<-closed
	}
	return err
}

func main() {
	godotenv.Load()
	config.LoadAll()

	log.SetSeverity(config.LogLevel())
	log.SetOutput(config.LogFile())

	err := run()
	if err != nil {
		log.Fatalf("ERROR: %s", err)
	}
	log.Info("Shutdown complete.")
}
