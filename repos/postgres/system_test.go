package postgres

import (
	"context"
	"errors"
	"github.com/juho05/crossonic-server/repos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestSystemRepository(t *testing.T) {
	db, _ := thSetupDatabase(t)

	ctx := context.Background()

	repo := db.System()

	t.Run("InstanceID", func(t *testing.T) {
		thDeleteAll(t, db, "system")

		instanceID, err := repo.InstanceID(ctx)
		require.NoErrorf(t, err, "getting instance ID for first time: %v", err)
		assert.NotEmpty(t, instanceID)

		instanceID2, err := repo.InstanceID(ctx)
		require.NoErrorf(t, err, "getting instance ID for second time: %v", err)
		assert.Equalf(t, instanceID, instanceID2, "instance ID should stay the same when calling multiple times")
	})

	t.Run("(Set)LastScan", func(t *testing.T) {
		thDeleteAll(t, db, "system")

		_, err := repo.LastScan(ctx)
		assert.True(t, errors.Is(err, repos.ErrNotFound), "first call to LastScan should return ErrNotFound")

		lastScanTime := time.Date(2222, 1, 2, 3, 4, 5, 0, time.UTC)

		err = repo.SetLastScan(ctx, lastScanTime)
		require.NoErrorf(t, err, "set last scan time: %v", err)

		lastScan, err := repo.LastScan(ctx)
		require.NoErrorf(t, err, "get last scan time: %v", err)
		assert.Equalf(t, lastScanTime, lastScan, "last scan time from the repo should match the provided time")
	})

	t.Run("(Reset)NeedsFullScan", func(t *testing.T) {
		thDeleteAll(t, db, "system")

		needsFullScan, err := repo.NeedsFullScan(ctx)
		require.NoErrorf(t, err, "needs full scan: %v", err)
		assert.Falsef(t, needsFullScan, "first call to needs full scan should return false")

		_, err = db.db.ExecContext(ctx, "INSERT INTO system (key, value) VALUES ($1, $2)", "needs-full-scan", "1")
		require.NoErrorf(t, err, "set needs-full-scan to true: %v", err)

		needsFullScan, err = repo.NeedsFullScan(ctx)
		require.NoErrorf(t, err, "needs full scan: %v", err)
		assert.Truef(t, needsFullScan, "needs full scan after updating db should return true")

		err = repo.ResetNeedsFullScan(ctx)
		require.NoErrorf(t, err, "reset needs full scan: %v", err)

		needsFullScan, err = repo.NeedsFullScan(ctx)
		require.NoErrorf(t, err, "needs full scan: %v", err)
		assert.Falsef(t, needsFullScan, "needs full scan should return false after resetting")
	})
}
