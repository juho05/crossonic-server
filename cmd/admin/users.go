package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"syscall"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/juho05/crossonic-server/db/sqlc"
	"golang.org/x/crypto/ssh/terminal"
)

func usersList(store sqlc.Store) error {
	users, err := store.FindUsers(context.Background())
	if err != nil {
		return err
	}
	fmt.Printf("Users (%d):\n", len(users))
	for _, u := range users {
		fmt.Println("  -", u.Name)
	}
	return nil
}

func usersCreate(args []string, store sqlc.Store) error {
	if len(args) < 4 {
		fmt.Println("USAGE:", args[0], "users create <name>")
		os.Exit(1)
	}

	var password string
	for password == "" {
		fmt.Print("Enter password: ")
		p1, err := terminal.ReadPassword(syscall.Stdin)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Print("\nRepeat password: ")
		p2, err := terminal.ReadPassword(syscall.Stdin)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println()
		if bytes.Equal(p1, p2) {
			password = string(p1)
		} else {
			fmt.Println("Passwords don't match. Try again.")
		}
	}

	encryptedPassword, err := sqlc.EncryptPassword(password)
	if err != nil {
		return err
	}
	err = store.CreateUser(context.Background(), sqlc.CreateUserParams{
		Name:              args[3],
		EncryptedPassword: encryptedPassword,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if strings.Contains(pgErr.ConstraintName, "pkey") {
				return errors.New("user already exists")
			} else {
				return err
			}
		} else {
			return err
		}
	}
	return nil
}

func usersDelete(args []string, store sqlc.Store) error {
	if len(args) < 4 {
		fmt.Println("USAGE:", args[0], "users delete <name>")
		os.Exit(1)
	}
	_, err := store.DeleteUser(context.Background(), args[3])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("user '%s' does not exist", args[3])
		}
		return err
	}
	return nil
}

func users(args []string, store sqlc.Store) error {
	if len(args) < 3 {
		fmt.Println("USAGE:", args[0], "users <command>\n\nCOMMANDS:\n  list\n  create\n  delete")
		os.Exit(1)
	}
	var err error
	switch args[2] {
	case "list":
		err = usersList(store)
	case "create":
		err = usersCreate(args, store)
	case "delete":
		err = usersDelete(args, store)
	default:
		fmt.Println("Unknown command")
		fmt.Println("USAGE:", args[0], "users <command>\n\nCOMMANDS:\n  list\n  create\n  delete")
		os.Exit(1)
	}
	return err
}
