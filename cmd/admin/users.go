package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

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
		if errors.Is(err, repos.ErrExists) {
			return errors.New("user already exists")
		}
		return err
	}
	fmt.Printf("Created user '%s'.\n", args[3])
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
	fmt.Printf("Deleted user '%s'.\n", args[3])
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

	fmt.Printf("Changed name from '%s' to '%s'.\n", user, name)
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

	fmt.Printf("Changed password of user '%s'.\n", user)
	return nil
}

func usersApiKeys(args []string, db repos.DB) error {
	if len(args) < 5 {
		fmt.Println("USAGE:", args[0], "users api-keys <command> <user_name>\n\nCOMMANDS:\n  create\n  list\n  delete")
		os.Exit(1)
	}
	switch args[3] {
	case "create":
		return usersApiKeysCreate(args[4], db)
	case "list":
		return usersApiKeysList(args[4], db)
	case "delete":
		return usersApiKeysDelete(args, db)
	default:
		fmt.Println("USAGE:", args[0], "users api-keys <command> <user_name>\n\nCOMMANDS:\n  create\n  list\n  delete")
		os.Exit(1)
	}
	return nil
}

func usersApiKeysCreate(user string, db repos.DB) error {
	name := input("Display name")

	key, err := db.User().CreateAPIKey(context.Background(), user, name)
	if err != nil {
		if errors.Is(err, repos.ErrNotFound) {
			return fmt.Errorf("user '%s' does not exist", user)
		}
		if errors.Is(err, repos.ErrExists) {
			return fmt.Errorf("key '%s' already exists for user '%s'", key, user)
		}
		return fmt.Errorf("create api key in db: %w", err)
	}
	fmt.Printf("Created new API key with name '%s' for user '%s':\n", name, user)
	fmt.Println(key)
	return nil
}

func usersApiKeysList(user string, db repos.DB) error {
	keys, err := db.User().FindAPIKeys(context.Background(), user)
	if err != nil {
		return fmt.Errorf("list api keys in db: %w", err)
	}
	if len(keys) == 0 {
		fmt.Printf("No API keys found for user '%s'.\n", user)
		return nil
	}
	for _, key := range keys {
		fmt.Printf("- %s (created: %s)\n", key.Name, key.Created.Format(time.DateTime))
	}
	return nil
}

func usersApiKeysDelete(args []string, db repos.DB) error {
	if len(args) < 6 {
		fmt.Println("USAGE:", args[0], "users api-keys <command> <user_name>\n\nCOMMANDS:\n  create\n  list\n  delete")
		os.Exit(1)
	}
	user := args[4]
	key := args[5]
	err := db.User().DeleteAPIKey(context.Background(), user, key)
	if err != nil {
		if errors.Is(err, repos.ErrNotFound) {
			return fmt.Errorf("user '%s' does not have an API key with name '%s'", user, key)
		}
		return fmt.Errorf("delete api key in db: %w", err)
	}
	fmt.Printf("Deleted API key '%s'.\n", key)
	return nil
}

func users(args []string, db repos.DB) error {
	if len(args) < 3 {
		fmt.Println("USAGE:", args[0], "users <command>\n\nCOMMANDS:\n  create\n  list\n  update\n  api-keys\n  delete")
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
	case "api-keys":
		err = usersApiKeys(args, db)
	case "delete":
		err = usersDelete(args, db)
	default:
		fmt.Println("Unknown command")
		fmt.Println("USAGE:", args[0], "users <command>\n\nCOMMANDS:\n  list\n  create\n  update\n  api-keys\n  delete")
		os.Exit(1)
	}
	return err
}
