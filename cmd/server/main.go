package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/juho05/crossonic-server/config"
	db "github.com/juho05/crossonic-server/db/sqlc"
	"github.com/juho05/crossonic-server/handlers"
	"github.com/juho05/crossonic-server/scanner"
	"github.com/juho05/log"
)

func init() {
	mime.AddExtensionType(".aac", "audio/aac")
	mime.AddExtensionType(".mp3", "audio/mpeg")
	mime.AddExtensionType(".oga", "audio/ogg")
	mime.AddExtensionType(".ogg", "audio/ogg")
	mime.AddExtensionType(".opus", "audio/opus")
	mime.AddExtensionType(".wav", "audio/wav")
	mime.AddExtensionType(".flac", "audio/flac")
}

func run() error {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", config.DBUser(), config.DBPassword(), config.DBHost(), config.DBPort(), config.DBName())
	dbConn, err := db.Connect(dsn)
	if err != nil {
		return err
	}
	defer db.Close(dbConn)

	if config.AutoMigrate() {
		err = db.AutoMigrate(dsn)
		if err != nil {
			return err
		}
	}

	store, err := db.NewStore(dbConn)
	if err != nil {
		return err
	}

	scanner := scanner.New(config.MusicDir(), store)
	err = scanner.ScanMediaFull()
	if err != nil {
		return err
	}

	handler := handlers.New(store)

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
	err = server.ListenAndServe()
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
