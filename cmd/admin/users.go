package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/juho05/crossonic-server/repos"
)

func usersList(db repos.DB) error {
	users, err := db.User().FindAll(context.Background())
	if err != nil {
		return err
	}
	fmt.Printf("Users (%d):\n", len(users))
	for _, u := range users {
		fmt.Println("  -", u.Name)
	}
	return nil
}

func usersCreate(args []string, db repos.DB) error {
	if len(args) < 4 {
		fmt.Println("USAGE:", args[0], "users create <name>")
		os.Exit(1)
	}

	var password string
	for password == "" {
		p1 := inputPassword("Enter password")
		p2 := inputPassword("Repeat password")
		if p1 == p2 {
			password = p1
		} else {
			fmt.Println("Passwords don't match. Try again.")
		}
	}

	err := db.User().Create(context.Background(), args[3], password)
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
	fmt.Printf("Created user %s.\n", args[3])
	return nil
}

func usersDelete(args []string, db repos.DB) error {
	if len(args) < 4 {
		fmt.Println("USAGE:", args[0], "users delete <name>")
		os.Exit(1)
	}
	err := db.User().DeleteByName(context.Background(), args[3])
	if err != nil {
		if errors.Is(err, repos.ErrNotFound) {
			return fmt.Errorf("user '%s' does not exist", args[3])
		}
		return err
	}
	fmt.Printf("Deleted user %s.\n", args[3])
	return nil
}

func usersUpdate(args []string, db repos.DB) error {
	if len(args) < 5 {
		fmt.Println("USAGE:", args[0], "users update <name/password> <name>")
		os.Exit(1)
	}
	switch args[3] {
	case "name":
		return usersChangeName(args[4], db)
	case "password":
		return usersChangePassword(args[4], db)
	default:
		fmt.Println("USAGE:", args[0], "users update <name/password> <name>")
		os.Exit(1)
	}
	return nil
}

func usersChangeName(user string, db repos.DB) error {
	name := input("Enter new name")

	err := db.User().Update(context.Background(), user, repos.UpdateUserParams{
		Name: repos.NewOptionalFull(name),
	})
	if err != nil {
		return fmt.Errorf("update user name in db: %w", err)
	}

	fmt.Printf("Changed name from %s to %s.\n", user, name)
	return nil
}

func usersChangePassword(user string, db repos.DB) error {
	var password string
	for password == "" {
		p1 := inputPassword("Enter new password")
		p2 := inputPassword("Repeat password")
		if p1 == p2 {
			password = p1
		} else {
			fmt.Println("Passwords don't match. Try again.")
		}
	}

	err := db.User().Update(context.Background(), user, repos.UpdateUserParams{
		Password: repos.NewOptionalFull(password),
	})
	if err != nil {
		return fmt.Errorf("update user name in db: %w", err)
	}

	fmt.Printf("Changed password of user %s.\n", user)
	return nil
}

func users(args []string, db repos.DB) error {
	if len(args) < 3 {
		fmt.Println("USAGE:", args[0], "users <command>\n\nCOMMANDS:\n  list\n  create\n  update\n  delete")
		os.Exit(1)
	}
	var err error
	switch args[2] {
	case "list":
		err = usersList(db)
	case "create":
		err = usersCreate(args, db)
	case "update":
		err = usersUpdate(args, db)
	case "delete":
		err = usersDelete(args, db)
	default:
		fmt.Println("Unknown command")
		fmt.Println("USAGE:", args[0], "users <command>\n\nCOMMANDS:\n  list\n  create\n  update\n  delete")
		os.Exit(1)
	}
	return err
}
