package repos

import "context"

type User struct {
	Name              string `db:"name"`
	EncryptedPassword []byte `db:"encrypted_password"`

	ListenBrainzUsername       *string `db:"listenbrainz_username"`
	EncryptedListenBrainzToken []byte  `db:"encrypted_listenbrainz_token"`
	ListenBrainzScrobble       bool    `db:"listenbrainz_scrobble"`
	ListenBrainzSyncFeedback   bool    `db:"listenbrainz_sync_feedback"`
}

type UpdateListenBrainzSettingsParams struct {
	Scrobble     Optional[bool]
	SyncFeedback Optional[bool]
}

type UpdateUserParams struct {
	Name     Optional[string]
	Password Optional[string]
}

// UserRepository is an interface to manipulate user data in a database.
type UserRepository interface {
	// Create creates a new user in the database. The password parameter is automatically encrypted before
	// storing it in the db.
	// Returns an error if a user with the name already exists.
	Create(ctx context.Context, name, password string) error

	// UpdateListenBrainzConnection updates the ListenBrainz username and token of the user.
	// The token is automatically encrypted.
	// lbUsername and lbToken must either both be nil or not nil. If they are nil, the ListenBrainz settings
	// will be reset to their default values.
	// Returns ErrNotFound if the user could not be found.
	UpdateListenBrainzConnection(ctx context.Context, user string, lbUsername, lbToken *string) error

	// UpdateListenBrainzSettings is used to enable/disable specific ListenBrainz sync features such as scrobbling
	// and syncing love feedback.
	UpdateListenBrainzSettings(ctx context.Context, user string, params UpdateListenBrainzSettingsParams) error

	// FindAll returns all users.
	FindAll(ctx context.Context) ([]*User, error)

	// FindByName returns the user with the provided name.
	// Returns an error if no user was found.
	FindByName(ctx context.Context, name string) (*User, error)

	// Update updates the specified properties of the user.
	// If no user with the name is found, ErrNotFound will be returned.
	Update(ctx context.Context, name string, params UpdateUserParams) error

	// DeleteByName deletes the user with the provided name.
	// Returns an error if no user was found.
	DeleteByName(ctx context.Context, name string) error
}
