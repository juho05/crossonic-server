package scanner

import (
	"errors"
	"path/filepath"
	"sync"

	"github.com/juho05/crossonic-server/cache"
	"github.com/juho05/crossonic-server/config"
	sqlc "github.com/juho05/crossonic-server/db/sqlc"
)

var (
	ErrAlreadyScanning = errors.New("scan already in progress")
)

type Scanner struct {
	lock          sync.Mutex
	waitGroup     sync.WaitGroup
	mediaDir      string
	originalStore sqlc.Store
	store         sqlc.Store
	coverDir      string
	firstScan     bool

	coverCache     *cache.Cache
	transcodeCache *cache.Cache

	Scanning bool
	Count    int
}

func New(mediaDir string, store sqlc.Store, coverCache *cache.Cache, transcodeCache *cache.Cache) *Scanner {
	return &Scanner{
		mediaDir:       mediaDir,
		store:          store,
		originalStore:  store,
		coverDir:       filepath.Join(config.DataDir(), "covers"),
		coverCache:     coverCache,
		transcodeCache: transcodeCache,
	}
}
