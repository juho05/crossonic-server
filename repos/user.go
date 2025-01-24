package repos

import "context"

type User struct {
	Name              string `db:"name"`
	EncryptedPassword []byte `db:"encrypted_password"`

	ListenBrainzUsername       *string `db:"listenbrainz_username"`
	EncryptedListenBrainzToken []byte  `db:"encrypted_listenbrainz_token"`
}

type UserRepository interface {
	Create(ctx context.Context, name, password string) error
	UpdateListenBrainzConnection(ctx context.Context, user string, lbUsername, lbToken *string) error
	FindAll(ctx context.Context) ([]*User, error)
	FindByName(ctx context.Context, name string) (*User, error)
	DeleteByName(ctx context.Context, name string) error
}
