package repos

import "context"

type User struct {
	Name              string `db:"name"`
	EncryptedPassword []byte `db:"encrypted_password"`

	ListenBrainzUsername       *string `db:"listenbrainz_username"`
	EncryptedListenBrainzToken []byte  `db:"encrypted_listenbrainz_token"`
}

// UserRepository is an interface to manipulate user data in a database.
type UserRepository interface {
	// Create creates a new user in the database. The password parameter is automatically encrypted before
	// storing it in the db.
	// Returns an error if a user with the name already exists.
	Create(ctx context.Context, name, password string) error

	// UpdateListenBrainzConnection updates the ListenBrainz username and token of the user.
	// The token is automatically encrypted.
	// Returns ErrNotFound if the user could not be found.
	UpdateListenBrainzConnection(ctx context.Context, user string, lbUsername, lbToken *string) error

	// FindAll returns all users.
	FindAll(ctx context.Context) ([]*User, error)

	// FindByName returns the user with the provided name.
	// Returns an error if no user was found.
	FindByName(ctx context.Context, name string) (*User, error)

	// DeleteByName deletes the user with the provided name.
	// Returns an error if no user was found.
	DeleteByName(ctx context.Context, name string) error
}
