package cache

import (
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

type CacheObject struct {
	cache *Cache
	key   string

	lock        sync.RWMutex
	size        int64
	file        *os.File
	readerCount int
	complete    bool

	modified time.Time
	accessed time.Time
}

func (c *Cache) newCacheObject(key string) (*CacheObject, error) {
	file, err := os.Create(c.keyToPath(key))
	if err != nil {
		return nil, fmt.Errorf("new cache object: %w", err)
	}
	os.Remove(c.keyToPath(key) + "-complete")
	return &CacheObject{
		cache:    c,
		key:      key,
		modified: time.Now(),
		accessed: time.Now(),
		file:     file,
		size:     0,
	}, nil
}

func (c *CacheObject) Write(p []byte) (n int, err error) {
	if c.complete {
		return 0, fmt.Errorf("cache object: write: %w", ErrComplete)
	}
	c.lock.Lock()
	defer c.lock.Unlock()
	n, err = c.file.Write(p)
	c.size += int64(n)
	c.modified = time.Now()
	return n, err
}

func (c *CacheObject) SetComplete() error {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.complete = true
	f, err := os.Create(c.cache.keyToPath(c.key) + "-complete")
	if err != nil {
		return fmt.Errorf("set complete: %w", err)
	}
	f.Close()
	go c.cache.clean()
	return nil
}

func (c *CacheObject) IsComplete() bool {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.complete
}

func (c *CacheObject) Modified() time.Time {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.modified
}

func (c *CacheObject) Reader() (io.ReadSeekCloser, error) {
	return c.newCacheReader()
}

func (c *CacheObject) Key() string {
	return c.key
}

func (c *CacheObject) Path() string {
	return c.cache.keyToPath(c.key)
}

func (c *CacheObject) Close() error {
	c.lock.Lock()
	defer c.lock.Unlock()
	var err error
	if c.file != nil {
		err = c.file.Close()
		c.file = nil
	}
	return err
}

func (c *CacheObject) UseFile() error {
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

func (c *CacheObject) ReleaseFile() error {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.readerCount--
	if c.readerCount <= 0 && c.complete {
		err := c.file.Close()
		c.file = nil
		if err != nil {
			return fmt.Errorf("cache object: release file: %w", err)
		}
	}
	if c.readerCount < 0 {
		return fmt.Errorf("cache object: release file: file not in use")
	}
	return nil
}
