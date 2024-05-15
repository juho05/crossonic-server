package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"mime"
	"os"

	"github.com/joho/godotenv"
	"github.com/juho05/crossonic-server/config"
	db "github.com/juho05/crossonic-server/db/sqlc"
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

	switch args[1] {
	case "users":
		err = users(args, store)
	case "remove-crossonic-metadata":
		err = removeCrossonicMetadata(args, store)
	default:
		fmt.Println("Unknown command")
		fmt.Println("USAGE:", args[0], "<command>\n\nCOMMANDS:\n  gen-encryption-key\n  users\n  remove-crossonic-metadata")
		os.Exit(1)
	}

	return err
}

func main() {
	godotenv.Load()

	log.SetSeverity(config.LogLevel())
	log.SetOutput(config.LogFile())

	err := run(os.Args)
	if err != nil {
		log.Fatalf("%s", err)
	}
}
