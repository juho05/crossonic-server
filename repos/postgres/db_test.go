package postgres

import (
	"context"
	"fmt"
	"github.com/juho05/crossonic-server"
	"github.com/juho05/crossonic-server/config"
	"github.com/juho05/log"
	"github.com/nullism/bqb"
	migrate "github.com/rubenv/sql-migrate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"net/http"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	log.SetSeverity(log.NONE)
	os.Exit(m.Run())
}

func TestMigrations(t *testing.T) {
	db, _ := thSetupDatabase(t)

	migrations := &migrate.HttpFileSystemMigrationSource{
		FileSystem: http.FS(crossonic.MigrationsFS),
	}
	nDown, err := migrate.Exec(db.db.DB, "postgres", migrations, migrate.Down)
	assert.NoErrorf(t, err, "migrate down: %v", err)
	assert.Greaterf(t, nDown, 2, "migrate down resulted in %d migrations", nDown)

	nUp, err := migrate.Exec(db.db.DB, "postgres", migrations, migrate.Up)
	assert.NoErrorf(t, err, "migrate up: %v", err)
	assert.Equalf(t, nDown, nUp, "down migration count (%d) does not match up migration count (%d)", nDown, nUp)
}

// test helpers

// the datbase is automatically closed on test cleanup
func thSetupDatabase(t *testing.T) (db *DB, encryptionKey []byte) {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping db tests in short mode")
	}

	ctx := context.Background()

	dbName := "crossonic"
	dbUser := "user"
	dbPassword := "password"

	encryptionKey = []byte{0xdd, 0xd5, 0xc1, 0xd3, 0x0c, 0xf8, 0x99, 0x1f, 0xdf, 0x7f, 0xe2,
		0x58, 0x13, 0x8e, 0xda, 0xb0, 0xc0, 0x37, 0xa1, 0x4a, 0xa2, 0x54, 0x5b, 0x86, 0xe6, 0xe4, 0x86, 0x7f, 0x68, 0x27, 0xf4, 0xad}

	postgresContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase(dbName),
		postgres.WithUsername(dbUser),
		postgres.WithPassword(dbPassword),
		postgres.BasicWaitStrategies(),
	)
	require.NoError(t, err, "setup test db: %v", err)

	dsn, err := postgresContainer.ConnectionString(ctx)
	require.NoError(t, err, "get connection string for test db: %v", err)

	db, err = NewDB(dsn, config.Config{
		AutoMigrate:   true,
		EncryptionKey: encryptionKey,
	})
	require.NoError(t, err, "new test db: %v", err)

	t.Cleanup(func() {
		err = db.Close()
		assert.NoError(t, err, "close db: %v", err)
		err = postgresContainer.Terminate(ctx)
		assert.NoError(t, err, "terminate test db container: %v", err)
	})

	return db, encryptionKey
}

func thCount(t *testing.T, db *DB, table string) int {
	t.Helper()
	var count int
	err := db.db.Get(&count, fmt.Sprintf("SELECT COALESCE(COUNT(*), 0) FROM %s", table))
	require.NoErrorf(t, err, "count %s: %v", table, err)
	return count
}

func thDeleteAll(t *testing.T, db *DB, table string) {
	t.Helper()
	_, err := db.db.Exec(fmt.Sprintf("DELETE FROM %s", table))
	require.NoErrorf(t, err, "delete all from %s: %v", table, err)
}

func thExists(t *testing.T, db *DB, table string, fields map[string]any) bool {
	t.Helper()
	var exists bool
	where := bqb.Optional("WHERE")
	where.And("(1 = 1)")
	for name, expectedValue := range fields {
		if expectedValue == nil {
			where.And(fmt.Sprintf("%s IS NULL", name))
			continue
		}
		where.And(fmt.Sprintf("(%s = ?)", name), expectedValue)
	}
	sql, args, err := bqb.New(fmt.Sprintf("SELECT EXISTS(SELECT * FROM %s ?)", table), where).ToPgsql()
	require.NoErrorf(t, err, "create exists query: %v", err)
	err = db.db.Get(&exists, sql, args...)
	require.NoErrorf(t, err, "check exists for table %s: %v", table, err)
	return exists
}
