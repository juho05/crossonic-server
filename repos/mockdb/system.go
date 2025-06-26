package mockdb

import (
	"context"
	"time"
)

type SystemRepository struct {
	InstanceIDMock         func(ctx context.Context) (string, error)
	LastScanMock           func(ctx context.Context) (time.Time, error)
	SetLastScanMock        func(ctx context.Context, t time.Time) error
	NeedsFullScanMock      func(ctx context.Context) (bool, error)
	ResetNeedsFullScanMock func(ctx context.Context) error
}

func (s SystemRepository) InstanceID(ctx context.Context) (string, error) {
	if s.InstanceIDMock != nil {
		return s.InstanceIDMock(ctx)
	}
	panic("not implemented")
}

func (s SystemRepository) LastScan(ctx context.Context) (time.Time, error) {
	if s.LastScanMock != nil {
		return s.LastScanMock(ctx)
	}
	panic("not implemented")
}

func (s SystemRepository) SetLastScan(ctx context.Context, lastScan time.Time) error {
	if s.SetLastScanMock != nil {
		return s.SetLastScanMock(ctx, lastScan)
	}
	panic("not implemented")
}

func (s SystemRepository) NeedsFullScan(ctx context.Context) (bool, error) {
	if s.NeedsFullScanMock != nil {
		return s.NeedsFullScanMock(ctx)
	}
	panic("not implemented")
}

func (s SystemRepository) ResetNeedsFullScan(ctx context.Context) error {
	if s.ResetNeedsFullScanMock != nil {
		return s.ResetNeedsFullScanMock(ctx)
	}
	panic("not implemented")
}
