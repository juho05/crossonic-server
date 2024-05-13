package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	db "github.com/juho05/crossonic-server/db/sqlc"
)

func usersList(store db.Store) error {
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

func usersCreate(args []string, store db.Store) error {
	if len(args) < 5 {
		fmt.Println("USAGE:", args[0], "users create <name> <password>")
		os.Exit(1)
	}
	encryptedPassword, err := db.EncryptPassword(args[4])
	if err != nil {
		return err
	}
	err = store.CreateUser(context.Background(), db.CreateUserParams{
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

func usersDelete(args []string, store db.Store) error {
	if len(args) < 4 {
		fmt.Println("USAGE:", args[0], "users delete <name>")
		os.Exit(1)
	}
	_, err := store.DeleteUser(context.Background(), args[3])
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("user '%s' does not exist", args[3])
		} else if errors.As(err, &pgErr) {
			fmt.Println(pgErr.Message)
			fmt.Println(pgErr.Code)
			fmt.Println(pgErr.ConstraintName)
		} else {
			return err
		}
	}
	return nil
}

func users(args []string, store db.Store) error {
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
