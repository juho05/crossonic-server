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
	"path/filepath"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/juho05/crossonic-server/cache"
	"github.com/juho05/crossonic-server/config"
	"github.com/juho05/crossonic-server/ffmpeg"
	"github.com/juho05/crossonic-server/handlers"
	"github.com/juho05/crossonic-server/lastfm"
	"github.com/juho05/crossonic-server/listenbrainz"
	"github.com/juho05/crossonic-server/repos/postgres"
	"github.com/juho05/crossonic-server/scanner"
	"github.com/juho05/log"
)

func init() {
	_ = mime.AddExtensionType(".aac", "audio/aac")
	_ = mime.AddExtensionType(".mp3", "audio/mpeg")
	_ = mime.AddExtensionType(".oga", "audio/ogg")
	_ = mime.AddExtensionType(".ogg", "audio/ogg")
	_ = mime.AddExtensionType(".opus", "audio/opus")
	_ = mime.AddExtensionType(".wav", "audio/wav")
	_ = mime.AddExtensionType(".flac", "audio/flac")
	_ = mime.AddExtensionType(".wasm", "application/wasm")
}

func run() error {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", config.DBUser(), config.DBPassword(), config.DBHost(), config.DBPort(), config.DBName())
	db, err := postgres.NewDB(dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	transcoder, err := ffmpeg.NewTranscoder()
	if err != nil {
		return err
	}

	// 5 GB
	transcodeCache, err := cache.New(filepath.Join(config.CacheDir(), "transcode"), 5e9, 7*24*time.Hour)
	if err != nil {
		return err
	}

	// 1 GB
	coverCache, err := cache.New(filepath.Join(config.CacheDir(), "covers"), 1e9, 30*24*time.Hour)
	if err != nil {
		return err
	}

	scanner, err := scanner.New(config.MusicDir(), db, coverCache, transcodeCache)
	if err != nil {
		return err
	}

	lBrainz := listenbrainz.New(db)

	if !config.DisableStartupScan() {
		go func() {
			err = scanner.Scan(db, false)
			if err != nil {
				log.Errorf("scan media: %s", err)
			}
			lBrainz.StartPeriodicSync(24 * time.Hour)
		}()
	}

	lfm := lastfm.New(config.LastFMApiKey())

	handler := handlers.New(db, scanner, lBrainz, lfm, transcoder, transcodeCache, coverCache)

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
		lBrainz.Close()
		err = server.Shutdown(timeout)
		if err != nil {
			log.Errorf("shutdown: %s", err)
		}
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
	_ = godotenv.Load()
	config.LoadAll()

	log.SetSeverity(config.LogLevel())
	log.SetOutput(config.LogFile())

	err := run()
	if err != nil {
		log.Fatalf("ERROR: %s", err)
	}
	log.Info("Shutdown complete.")
}
