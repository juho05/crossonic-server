package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/juho05/crossonic-server/repos"
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

func run(conf config.Config) error {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", conf.DBUser, conf.DBPassword, conf.DBHost, conf.DBPort, conf.DBName)
	db, err := postgres.NewDB(dsn, conf)
	if err != nil {
		return err
	}
	defer db.Close()

	transcoder, err := ffmpeg.NewTranscoder()
	if err != nil {
		return err
	}

	// 5 GB
	transcodeCache, err := cache.New(filepath.Join(conf.CacheDir, "transcode"), 5e9, 7*24*time.Hour)
	if err != nil {
		return err
	}

	// 1 GB
	coverCache, err := cache.New(filepath.Join(conf.CacheDir, "covers"), 1e9, 30*24*time.Hour)
	if err != nil {
		return err
	}

	mediaScanner, err := scanner.New(conf.MusicDir, db, conf, coverCache, transcodeCache)
	if err != nil {
		return err
	}

	lBrainz := listenbrainz.New(db)

	if conf.StartupScan != config.StartupScanDisabled {
		go func() {
			err = mediaScanner.Scan(db, conf.StartupScan == config.StartupScanFull)
			if err != nil {
				log.Errorf("scan media: %s", err)
			}
			lBrainz.StartPeriodicSync(3 * time.Hour)
		}()
	} else {
		log.Tracef("calculating fallback gain...")
		fallbackGain, err := db.Song().GetMedianReplayGain(context.Background())
		if err != nil {
			return fmt.Errorf("get median replay gain: %w", err)
		}
		if fallbackGain > 0 {
			repos.SetFallbackGain(fallbackGain)
		}
		lBrainz.StartPeriodicSync(3 * time.Hour)
	}

	lfm := lastfm.New(conf.LastFMApiKey)

	handler := handlers.New(conf, db, mediaScanner, lBrainz, lfm, transcoder, transcodeCache, coverCache)

	addr := conf.ListenAddr

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
		_ = lBrainz.Close()
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
	conf, errs := config.Load(os.Environ())
	if len(errs) > 0 {
		for _, e := range errs {
			log.Errorf("ERROR: %s", e)
		}
		log.Fatalf("ERROR: failed to load config")
	}

	log.SetSeverity(conf.LogLevel)
	log.SetOutput(conf.LogFile)

	err := run(conf)
	if err != nil {
		log.Fatalf("ERROR: %s", err)
	}
	log.Info("Shutdown complete.")
}
