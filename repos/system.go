package repos

import (
	"context"
	"time"
)

type SystemRepository interface {
	// InstanceID returns an ID that uniquely identifies this instance of crossonic-server
	// and is preserved between application restarts.
	InstanceID(ctx context.Context) (string, error)
	// LastScan returns the time of the last media scan.
	LastScan(ctx context.Context) (time.Time, error)
	// SetLastScan updates the time of the last media scan.
	SetLastScan(ctx context.Context, lastScan time.Time) error
	// NeedsFullScan whether a full scan should be triggered. It is set by database migrations when
	// a new data layout requires all media to be re-scanned.
	NeedsFullScan(ctx context.Context) (bool, error)
	// ResetNeedsFullScan removes the needs-full-scan marker after the media was successfully re-scanned.
	ResetNeedsFullScan(ctx context.Context) error
}
