package repos

import (
	"context"
	"time"
)

type SystemRepository interface {
	InstanceID(ctx context.Context) (string, error)
	LastScan(ctx context.Context) (time.Time, error)
	SetLastScan(ctx context.Context, lastScan time.Time) error
}
