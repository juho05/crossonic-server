package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/juho05/crossonic-server/config"
	db "github.com/juho05/crossonic-server/db/sqlc"
	"github.com/juho05/log"
)

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
		fmt.Println("USAGE:", args[0], "<command>\n\nCOMMANDS:\n  gen-encryption-key\n  users")
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

	store := db.NewStore(dbConn)
	if config.AutoMigrate() {
		return err
	}

	switch args[1] {
	case "users":
		err = users(args, store)
	default:
		fmt.Println("Unknown command")
		fmt.Println("USAGE:", args[0], "<command>\n\nCOMMANDS:\n  gen-encryption-key")
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
