// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.26.0

package db

type User struct {
	Name              string
	EncryptedPassword []byte
}