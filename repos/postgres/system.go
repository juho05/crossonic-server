package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jaevor/go-nanoid"
	"github.com/nullism/bqb"
)

type systemRepository struct {
	db executer
	tx func(ctx context.Context, fn func(s systemRepository) error) error
}

func (s systemRepository) InstanceID(ctx context.Context) (string, error) {
	genInstanceID, err := nanoid.CustomUnicode("abcdefghijklmnopqrstuvwxyz0123456789", 16)
	if err != nil {
		return "", fmt.Errorf("generate instance ID: %w", err)
	}
	instanceID, err := s.getOrCreate(ctx, "instance_id", genInstanceID())
	if err != nil {
		return "", fmt.Errorf("load instance ID: %s", err)
	}
	return instanceID, nil
}

func (s systemRepository) LastScan(ctx context.Context) (time.Time, error) {
	var lastScan time.Time
	lastScanStr, err := s.get(ctx, "last-scan")
	if err != nil {
		return time.Time{}, fmt.Errorf("get system value: %w", err)
	}

	lastScan, err = time.Parse(time.RFC3339, lastScanStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse time: %w", err)
	}

	return lastScan, nil
}

func (s systemRepository) SetLastScan(ctx context.Context, lastScan time.Time) error {
	return s.set(ctx, "last-scan", lastScan.Format(time.RFC3339))
}

func (s systemRepository) NeedsFullScan(ctx context.Context) (bool, error) {
	return getQuery[bool](ctx, s.db, bqb.New(`SELECT EXISTS (
		SELECT 1 FROM system WHERE key = ?
	)`, "needs-full-scan"))
}

func (s systemRepository) ResetNeedsFullScan(ctx context.Context) error {
	return executeQuery(ctx, s.db, bqb.New("DELETE FROM system WHERE key = ?", "needs-full-scan"))
}

func (s systemRepository) get(ctx context.Context, key string) (string, error) {
	q := bqb.New("SELECT value FROM system WHERE key = ?", key)
	return getQuery[string](ctx, s.db, q)
}

func (s systemRepository) getOrCreate(ctx context.Context, key, value string) (string, error) {
	q := bqb.New("INSERT INTO system (key, value) VALUES (?, ?) ON CONFLICT (key) DO UPDATE SET key = ? RETURNING value", key, value, key)
	return getQuery[string](ctx, s.db, q)
}

func (s systemRepository) set(ctx context.Context, key, value string) error {
	q := bqb.New("INSERT INTO system (key, value) VALUES (?, ?) ON CONFLICT (key) DO UPDATE SET value = ?", key, value, value)
	return executeQuery(ctx, s.db, q)
}
