package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/juho05/crossonic-server/repos"
	"github.com/juho05/crossonic-server/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserRepository(t *testing.T) {
	db, encKey := thSetupDatabase(t)

	ctx := context.Background()

	require.Equal(t, 0, thCount(t, db, "users"), "there should be no users at beginning of test")

	repo := db.User()

	getUser := func(user string, allowNil bool) *repos.User {
		u, err := repo.FindByName(ctx, user)
		if errors.Is(err, repos.ErrNotFound) && allowNil {
			return nil
		}
		require.NoErrorf(t, err, "get test user: %v", err)
		return u
	}

	t.Run("Create", func(t *testing.T) {
		t.Run("create user", func(t *testing.T) {
			user := "testuser"
			err := repo.Create(ctx, user, "testpassword")
			require.NoErrorf(t, err, "create user: %v", err)

			var u repos.User
			err = db.db.GetContext(ctx, &u, "SELECT * FROM users WHERE name = $1", user)
			require.NoErrorf(t, err, "get created user: %v", err)

			password, err := repos.DecryptPassword(u.EncryptedPassword, encKey)
			assert.NoErrorf(t, err, "decrypt user password: %v", err)
			assert.Equal(t, "testpassword", password)

			assert.Nil(t, u.ListenBrainzUsername, "listenbrainz username should be nil")
			assert.Nil(t, u.EncryptedListenBrainzToken, "listenbrainz token should be nil")
			assert.True(t, u.ListenBrainzScrobble, "listenbrainz scrobble should be true")
			assert.False(t, u.ListenBrainzSyncFeedback, "listenbrainz sync feedback should be false")
		})

		t.Run("trying to create existing user should return error", func(t *testing.T) {
			user := thCreateUser(t, db)
			err := repo.Create(ctx, user, "asdf")
			assert.Error(t, err, "expected error creating already existing user")
		})
	})

	t.Run("UpdateListenBrainzConnection", func(t *testing.T) {
		user := thCreateUser(t, db)
		user2 := thCreateUser(t, db)

		t.Run("add listenbrainz connection", func(t *testing.T) {
			err := repo.UpdateListenBrainzConnection(ctx, user, util.ToPtr("lbtestuser"), util.ToPtr("lbtesttoken"))
			assert.NoErrorf(t, err, "update listenbrainz connection: %v", err)

			u := getUser(user, false)

			assert.NotNil(t, u.ListenBrainzUsername)
			assert.NotNil(t, u.EncryptedListenBrainzToken)

			assert.Equal(t, "lbtestuser", *u.ListenBrainzUsername)

			token, err := repos.DecryptPassword(u.EncryptedListenBrainzToken, encKey)
			assert.NoErrorf(t, err, "decrypt listenbrainz token: %v", err)
			assert.Equal(t, "lbtesttoken", token)

			assert.True(t, u.ListenBrainzScrobble)
			assert.False(t, u.ListenBrainzSyncFeedback)

			u2 := getUser(user2, false)
			assert.Nil(t, u2.ListenBrainzUsername, "updating one user should not affect another")
			assert.Nil(t, u2.EncryptedListenBrainzToken, "updating one user should not affect another")
		})

		t.Run("partial listenbrainz connection", func(t *testing.T) {
			err := repo.UpdateListenBrainzConnection(ctx, user, util.ToPtr("lbtestuser"), nil)
			assert.Error(t, err, "expected error setting lbToken to nil but not lbUsername updating listenbrainz connection")

			u := getUser(user, false)

			assert.NotNil(t, u.ListenBrainzUsername)
			assert.NotNil(t, u.EncryptedListenBrainzToken)
			assert.Equal(t, "lbtestuser", *u.ListenBrainzUsername)
			token, err := repos.DecryptPassword(u.EncryptedListenBrainzToken, encKey)
			assert.NoErrorf(t, err, "decrypt listenbrainz token: %v", err)
			assert.Equal(t, "lbtesttoken", token)

			err = repo.UpdateListenBrainzConnection(ctx, user, nil, util.ToPtr("lbtesttoken"))
			assert.Error(t, err, "expected error setting lbUsername to nil but not lbToken updating listenbrainz connection")

			assert.NotNil(t, u.ListenBrainzUsername)
			assert.NotNil(t, u.EncryptedListenBrainzToken)
			assert.Equal(t, "lbtestuser", *u.ListenBrainzUsername)
			token, err = repos.DecryptPassword(u.EncryptedListenBrainzToken, encKey)
			assert.NoErrorf(t, err, "decrypt listenbrainz token: %v", err)
			assert.Equal(t, "lbtesttoken", token)
		})

		t.Run("remove listenbrainz connection", func(t *testing.T) {
			err := repo.UpdateListenBrainzConnection(ctx, user, util.ToPtr("lbtestuser"), util.ToPtr("lbtesttoken"))
			require.NoErrorf(t, err, "update listenbrainz connection: %v", err)

			err = repo.UpdateListenBrainzSettings(ctx, user, repos.UpdateListenBrainzSettingsParams{
				Scrobble:     repos.NewOptionalFull(false),
				SyncFeedback: repos.NewOptionalFull(true),
			})
			require.NoErrorf(t, err, "update listenbrainz settings: %v", err)

			err = repo.UpdateListenBrainzConnection(ctx, user, nil, nil)
			assert.NoErrorf(t, err, "update listenbrainz connection: %v", err)

			u := getUser(user, false)

			assert.Nil(t, u.ListenBrainzUsername)
			assert.Nil(t, u.EncryptedListenBrainzToken)
			assert.True(t, u.ListenBrainzScrobble)
			assert.False(t, u.ListenBrainzSyncFeedback)
		})

		t.Run("user does not exist", func(t *testing.T) {
			err := repo.UpdateListenBrainzConnection(ctx, "does not exist", util.ToPtr("lbtestuser"), util.ToPtr("lbtesttoken"))
			assert.Truef(t, errors.Is(err, repos.ErrNotFound), "expected error %v, got %v", repos.ErrNotFound, err)
		})

		// TODO test that lb_feedback_status is reset after changing ListenBrainz connection
	})

	t.Run("UpdateListenBrainzSettings", func(t *testing.T) {
		user := thCreateUser(t, db)

		user2 := thCreateUser(t, db)

		err := repo.UpdateListenBrainzConnection(ctx, user, util.ToPtr("lbtestuser"), util.ToPtr("lbtesttoken"))
		require.NoErrorf(t, err, "update listenbrainz settings: %v", err)

		t.Run("user with listenbrainz connection exists", func(t *testing.T) {
			err := repo.UpdateListenBrainzSettings(ctx, user, repos.UpdateListenBrainzSettingsParams{
				Scrobble:     repos.NewOptionalFull(false),
				SyncFeedback: repos.NewOptionalFull(true),
			})
			assert.NoErrorf(t, err, "update listenbrainz settings: %v", err)

			u := getUser(user, false)

			assert.False(t, u.ListenBrainzScrobble)
			assert.True(t, u.ListenBrainzSyncFeedback)

			u2 := getUser(user2, false)

			assert.True(t, u2.ListenBrainzScrobble, "updating one user should affect another one")
			assert.False(t, u2.ListenBrainzSyncFeedback, "updating one user should not affect another one")
		})

		t.Run("user does not exist", func(t *testing.T) {
			err := repo.UpdateListenBrainzSettings(ctx, "doesnotexist", repos.UpdateListenBrainzSettingsParams{
				Scrobble:     repos.NewOptionalFull(false),
				SyncFeedback: repos.NewOptionalFull(true),
			})
			assert.ErrorIs(t, err, repos.ErrNotFound, "expected error %v, got %v", repos.ErrNotFound, err)
		})

		t.Run("user does not have a listenbrainz connection", func(t *testing.T) {
			err := repo.UpdateListenBrainzSettings(ctx, user2, repos.UpdateListenBrainzSettingsParams{
				Scrobble:     repos.NewOptionalFull(false),
				SyncFeedback: repos.NewOptionalFull(true),
			})
			assert.ErrorIs(t, err, repos.ErrNotFound, "expected error %v, got %v", repos.ErrNotFound, err)
		})

		t.Run("empty update", func(t *testing.T) {
			err := repo.UpdateListenBrainzSettings(ctx, user, repos.UpdateListenBrainzSettingsParams{
				Scrobble:     repos.NewOptionalFull(false),
				SyncFeedback: repos.NewOptionalFull(true),
			})
			require.NoErrorf(t, err, "update listenbrainz settings: %v", err)

			err = repo.UpdateListenBrainzSettings(ctx, user, repos.UpdateListenBrainzSettingsParams{
				Scrobble:     repos.NewOptionalEmpty[bool](),
				SyncFeedback: repos.NewOptionalEmpty[bool](),
			})
			assert.NoErrorf(t, err, "update listenbrainz settings: %v", err)

			u := getUser(user, false)

			assert.False(t, u.ListenBrainzScrobble)
			assert.True(t, u.ListenBrainzSyncFeedback)
		})
	})

	t.Run("FindAll", func(t *testing.T) {
		thDeleteAll(t, db, "users")

		t.Run("zero users", func(t *testing.T) {
			users, err := repo.FindAll(ctx)
			assert.NoErrorf(t, err, "find all users: %v", err)
			assert.NotNil(t, users)
			assert.Equal(t, 0, len(users))
		})

		user1 := thCreateUser(t, db)

		t.Run("one user", func(t *testing.T) {
			users, err := repo.FindAll(ctx)
			assert.NoErrorf(t, err, "find all users: %v", err)
			assert.NotNil(t, users)
			assert.Equal(t, 1, len(users))
			assert.Equal(t, user1, users[0].Name)
		})

		user2 := thCreateUser(t, db)

		t.Run("two users", func(t *testing.T) {
			users, err := repo.FindAll(ctx)
			assert.NoErrorf(t, err, "find all users: %v", err)
			assert.NotNil(t, users)
			assert.Equal(t, 2, len(users))
			assert.True(t,
				(user1 == users[0].Name && user2 == users[1].Name) ||
					user1 == users[1].Name && user2 == users[0].Name,
				"the two correct users are returned",
			)
		})
	})

	t.Run("FindByName", func(t *testing.T) {
		// ensure multiple users exist
		_ = thCreateUser(t, db)
		user := thCreateUser(t, db)

		t.Run("user does not exist", func(t *testing.T) {
			_, err := repo.FindByName(ctx, "does not exist")
			assert.True(t, errors.Is(err, repos.ErrNotFound))
		})

		t.Run("user does exist", func(t *testing.T) {
			u, err := repo.FindByName(ctx, user)
			assert.NoErrorf(t, err, "find user: %v", err)
			assert.NotNil(t, u)
			assert.Equal(t, user, u.Name)
		})
	})

	t.Run("DeleteByName", func(t *testing.T) {
		t.Run("user does not exist", func(t *testing.T) {
			thDeleteAll(t, db, "users")
			_ = thCreateUser(t, db)
			_ = thCreateUser(t, db)
			require.Equal(t, 2, thCount(t, db, "users"))

			err := repo.DeleteByName(ctx, "does not exist")
			assert.True(t, errors.Is(err, repos.ErrNotFound))
			require.Equal(t, 2, thCount(t, db, "users"))
		})

		t.Run("user does exist", func(t *testing.T) {
			thDeleteAll(t, db, "users")
			_ = thCreateUser(t, db)
			user := thCreateUser(t, db)
			require.Equal(t, 2, thCount(t, db, "users"))

			err := repo.DeleteByName(ctx, user)
			assert.NoErrorf(t, err, "delete test user: %v", err)

			u := getUser(user, true)
			assert.Nil(t, u, "test user should not exist after delete")

			assert.Equal(t, 1, thCount(t, db, "users"), "only the user with matching name should be deleted")
		})
	})
}

// test helpers

func thCreateUser(t *testing.T, db *DB) string {
	t.Helper()
	user := "testuser-" + uuid.NewString()
	err := db.User().Create(context.Background(), user, "testpassword")
	require.NoErrorf(t, err, "create test user: %v", err)
	return user
}
