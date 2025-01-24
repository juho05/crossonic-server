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

func run(args []string) error {
	if len(args) < 2 {
		fmt.Println("USAGE:", args[0], "<command>\n\nCOMMANDS:\n  gen-encryption-key\n  users\n  remove-crossonic-metadata")
		os.Exit(1)
	}
	if args[1] == "gen-encryption-key" {
		return genEncryptionKey()
	}
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", config.DBUser(), config.DBPassword(), config.DBHost(), config.DBPort(), config.DBName())
	db, err := postgres.NewDB(dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	switch args[1] {
	case "users":
		err = users(args, db)
	case "remove-crossonic-metadata":
		err = removeCrossonicMetadata(args, db)
	default:
		fmt.Println("Unknown command")
		fmt.Println("USAGE:", args[0], "<command>\n\nCOMMANDS:\n  gen-encryption-key\n  users\n  remove-crossonic-metadata")
		os.Exit(1)
	}

	return err
}

func main() {
	_ = godotenv.Load()

	log.SetSeverity(config.LogLevel())
	log.SetOutput(config.LogFile())

	err := run(os.Args)
	if err != nil {
		log.Fatalf("%s", err)
	}
}
