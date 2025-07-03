package scanner

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/juho05/crossonic-server/cache"
	"github.com/juho05/crossonic-server/config"
	"github.com/juho05/crossonic-server/repos"
)

var (
	ErrAlreadyScanning = errors.New("scan already in progress")
)

type Scanner struct {
	lock sync.Mutex

	mediaDir string
	conf     config.Config

	tx repos.Transaction

	coverDir string

	coverCache     *cache.Cache
	transcodeCache *cache.Cache

	scanning  bool
	counter   atomic.Uint32
	scanStart time.Time
	fullScan  bool

	instanceID string
	firstScan  bool
	lastScan   time.Time

	artists *artistMap
	albums  *albumMap

	songQueue           chan *mediaFile
	songQueueClosed     bool
	setAlbumCover       chan albumCover
	setAlbumCoverClosed bool
}

func New(mediaDir string, db repos.DB, conf config.Config, coverCache *cache.Cache, transcodeCache *cache.Cache) (*Scanner, error) {
	instanceID, err := db.System().InstanceID(context.Background())
	if err != nil {
		return nil, fmt.Errorf("get instance id: %w", err)
	}
	return &Scanner{
		mediaDir:       mediaDir,
		coverDir:       filepath.Join(conf.DataDir, "covers"),
		coverCache:     coverCache,
		transcodeCache: transcodeCache,
		instanceID:     instanceID,
		conf:           conf,
	}, nil
}

func (s *Scanner) Scanning() bool {
	return s.scanning
}

func (s *Scanner) Count() int {
	return int(s.counter.Load())
}
