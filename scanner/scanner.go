package scanner

import (
	"errors"
	"path/filepath"
	"sync"

	"github.com/juho05/crossonic-server/config"
	db "github.com/juho05/crossonic-server/db/sqlc"
)

var (
	ErrAlreadyScanning = errors.New("scan already in progress")
)

type Scanner struct {
	lock          sync.Mutex
	waitGroup     sync.WaitGroup
	mediaDir      string
	originalStore db.Store
	store         db.Store
	coverDir      string
	firstScan     bool
}

func New(mediaDir string, store db.Store) *Scanner {
	return &Scanner{
		mediaDir:      mediaDir,
		store:         store,
		originalStore: store,
		coverDir:      filepath.Join(config.DataDir(), "covers"),
	}
}
