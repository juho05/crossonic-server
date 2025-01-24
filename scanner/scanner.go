package scanner

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/juho05/crossonic-server/cache"
	"github.com/juho05/crossonic-server/config"
	"github.com/juho05/crossonic-server/repos"
)

var (
	ErrAlreadyScanning = errors.New("scan already in progress")
)

type Scanner struct {
	lock      sync.Mutex
	waitGroup sync.WaitGroup
	mediaDir  string

	db repos.DB
	tx repos.Transaction

	coverDir  string
	firstScan bool

	coverCache     *cache.Cache
	transcodeCache *cache.Cache

	Scanning bool
	Count    int

	instanceID string
}

func New(mediaDir string, db repos.DB, coverCache *cache.Cache, transcodeCache *cache.Cache) (*Scanner, error) {
	instanceID, err := db.System().InstanceID(context.Background())
	if err != nil {
		return nil, fmt.Errorf("new scanner: %w", err)
	}
	return &Scanner{
		mediaDir:       mediaDir,
		db:             db,
		coverDir:       filepath.Join(config.DataDir(), "covers"),
		coverCache:     coverCache,
		transcodeCache: transcodeCache,
		instanceID:     instanceID,
	}, nil
}
