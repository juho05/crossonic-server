package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"mime"
	"os"

	"github.com/joho/godotenv"
	"github.com/juho05/crossonic-server/config"
	"github.com/juho05/crossonic-server/repos/postgres"
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
}

func genEncryptionKey() error {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		return fmt.Errorf("generate encryption key: %w", err)
	}
	fmt.Println("Key:", base64.RawStdEncoding.EncodeToString(key))
	return nil
}

func run(args []string, conf config.Config) error {
	if len(args) < 2 {
		fmt.Println("USAGE:", args[0], "<command>\n\nCOMMANDS:\n  gen-encryption-key\n  users\n  remove-crossonic-metadata")
		os.Exit(1)
	}
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", conf.DBUser, conf.DBPassword, conf.DBHost, conf.DBPort, conf.DBName)
	db, err := postgres.NewDB(dsn, conf)
	if err != nil {
		return err
	}
	defer db.Close()

	switch args[1] {
	case "users":
		err = users(args, db)
	case "remove-crossonic-metadata":
		err = removeCrossonicMetadata(args, db, conf)
	default:
		fmt.Println("Unknown command")
		fmt.Println("USAGE:", args[0], "<command>\n\nCOMMANDS:\n  gen-encryption-key\n  users\n  remove-crossonic-metadata")
		os.Exit(1)
	}

	return err
}

func main() {
	_ = godotenv.Load()

	// gen-encryption-key does not depend on config and should therefore work without a valid config
	if len(os.Args) > 1 && os.Args[1] == "gen-encryption-key" {
		err := genEncryptionKey()
		if err != nil {
			log.Fatalf("%s", err)
		}
		return
	}

	conf, errs := config.Load(os.Environ())
	if len(errs) > 0 {
		for _, e := range errs {
			log.Errorf("ERROR: %s", e)
		}
		log.Fatalf("ERROR: failed to load config")
	}

	log.SetSeverity(conf.LogLevel)
	log.SetOutput(conf.LogFile)

	err := run(os.Args, conf)
	if err != nil {
		log.Fatalf("%s", err)
	}
}
