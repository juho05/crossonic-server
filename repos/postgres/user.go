package postgres

import (
	"context"
	"github.com/juho05/crossonic-server/config"

	"github.com/juho05/crossonic-server/repos"
	"github.com/nullism/bqb"
)

type userRepository struct {
	db   executer
	tx   func(ctx context.Context, fn func(u userRepository) error) error
	conf config.Config
}

func (u userRepository) Create(ctx context.Context, name, password string) error {
	encryptedPassword, err := repos.EncryptPassword(password, u.conf.EncryptionKey)
	if err != nil {
		return repos.NewError("encrypt password", repos.ErrGeneral, err)
	}
	q := bqb.New("INSERT INTO users (name, encrypted_password) VALUES (?, ?)", name, encryptedPassword)
	return executeQuery(ctx, u.db, q)
}

func (u userRepository) UpdateListenBrainzConnection(ctx context.Context, user string, lbUsername, lbToken *string) error {
	return u.tx(ctx, func(u userRepository) error {
		var encryptedToken []byte
		var err error
		if lbToken != nil {
			encryptedToken, err = repos.EncryptPassword(*lbToken, u.conf.EncryptionKey)
			if err != nil {
				return repos.NewError("encrypt ListenBrainz token", repos.ErrGeneral, err)
			}
		}
		q := bqb.New("UPDATE users SET encrypted_listenbrainz_token = ?, listenbrainz_username = ? WHERE name = ?", encryptedToken, lbUsername, user)
		err = executeQueryExpectAffectedRows(ctx, u.db, q)
		if err != nil {
			return err
		}
		q = bqb.New("DELETE FROM lb_feedback_status WHERE user_name = ?", user)
		err = executeQuery(ctx, u.db, q)
		if err != nil {
			return err
		}
		return nil
	})
}

func (u userRepository) FindAll(ctx context.Context) ([]*repos.User, error) {
	return selectQuery[*repos.User](ctx, u.db, bqb.New("SELECT users.* FROM users"))
}

func (u userRepository) FindByName(ctx context.Context, name string) (*repos.User, error) {
	us, err := getQuery[*repos.User](ctx, u.db, bqb.New("SELECT users.* FROM users WHERE name = ?", name))
	return us, err
}

func (u userRepository) DeleteByName(ctx context.Context, name string) error {
	return executeQueryExpectAffectedRows(ctx, u.db, bqb.New("DELETE FROM users WHERE name = ?", name))
}
