package cache

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

var (
	ErrComplete    = errors.New("complete")
	ErrNotComplete = errors.New("not complete")
)

type Object struct {
	cache *Cache
	key   string

	lock        sync.RWMutex
	size        int64
	file        *os.File
	readerCount int
	complete    bool

	// dataChanged is closed (and replaced with a fresh channel) whenever the
	// object's readable state changes: more data is written, it is marked
	// complete, or the underlying file is closed. Readers waiting for more data
	// of an incomplete object block on it instead of busy-spinning.
	dataChanged chan struct{}

	modified time.Time
	accessed time.Time
}

// signalDataChanged wakes any readers blocked waiting for more data and arms a
// fresh channel for future waiters. The caller must hold the write lock.
func (c *Object) signalDataChanged() {
	close(c.dataChanged)
	c.dataChanged = make(chan struct{})
}

func (c *Cache) newCacheObject(key string) (*Object, error) {
	file, err := os.Create(c.keyToPath(key))
	if err != nil {
		return nil, fmt.Errorf("new cache object: %w", err)
	}
	_ = os.Remove(c.keyToPath(key) + "-complete")
	return &Object{
		cache:       c,
		key:         key,
		modified:    time.Now(),
		accessed:    time.Now(),
		file:        file,
		size:        0,
		dataChanged: make(chan struct{}),
	}, nil
}

func (c *Object) Write(p []byte) (n int, err error) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.complete {
		return 0, fmt.Errorf("cache object: write: %w", ErrComplete)
	}
	n, err = c.file.Write(p)
	c.size += int64(n)
	c.modified = time.Now()
	if n > 0 {
		c.signalDataChanged()
	}
	return n, err
}

func (c *Object) SetComplete() error {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.complete = true
	c.signalDataChanged()
	f, err := os.Create(c.cache.keyToPath(c.key) + "-complete")
	if err != nil {
		return fmt.Errorf("set complete: %w", err)
	}
	f.Close()
	return nil
}

func (c *Object) IsComplete() bool {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.complete
}

func (c *Object) Modified() time.Time {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.modified
}

// Reader returns a reader over the object. For an incomplete object, Read
// blocks until more data is written; ctx cancellation (e.g. the client
// disconnecting) unblocks it with the context error.
func (c *Object) Reader(ctx context.Context) (io.ReadSeekCloser, error) {
	return c.newCacheReader(ctx)
}

func (c *Object) Key() string {
	return c.key
}

func (c *Object) Path() string {
	return c.cache.keyToPath(c.key)
}

func (c *Object) Close() error {
	c.lock.Lock()
	defer c.lock.Unlock()
	var err error
	if c.file != nil {
		err = c.file.Close()
		c.file = nil
		// Wake readers blocked on an incomplete object so they observe the
		// closed file and return instead of waiting forever.
		c.signalDataChanged()
	}
	return err
}

func (c *Object) UseFile() error {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.file == nil {
		f, err := os.Open(c.cache.keyToPath(c.key))
		if err != nil {
			return fmt.Errorf("cache object: use file: %w", err)
		}
		c.file = f
	}
	c.readerCount++
	c.accessed = time.Now()
	return nil
}

func (c *Object) ReleaseFile() error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.readerCount <= 0 {
		return fmt.Errorf("cache object: release file: file not in use")
	}

	c.readerCount--

	if c.readerCount <= 0 && c.complete && c.file != nil {
		err := c.file.Close()
		c.file = nil
		if err != nil {
			return fmt.Errorf("cache object: release file: %w", err)
		}
	}
	return nil
}

// stats returns a consistent snapshot of the bookkeeping fields read by the
// cache cleaner. Using this instead of touching the fields directly keeps the
// reads synchronized with Write/UseFile/ReleaseFile.
func (c *Object) stats() (readerCount int, size int64, accessed time.Time) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.readerCount, c.size, c.accessed
}

// closeForEviction closes the underlying file if (and only if) the object has
// no active readers, reporting whether it did. The readerCount check and the
// close happen under the same lock as UseFile/ReleaseFile, so an object can
// never be evicted while a reader is using (or about to use) it.
func (c *Object) closeForEviction() (closed bool, err error) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.readerCount > 0 {
		return false, nil
	}
	if c.file != nil {
		err = c.file.Close()
		c.file = nil
		c.signalDataChanged()
	}
	return true, err
}
