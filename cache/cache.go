package cache

import (
	"cmp"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/juho05/log"
)

type Cache struct {
	dir string

	maxSize       int64
	maxUnusedTime time.Duration

	objects     map[string]*Object
	objectsLock sync.RWMutex

	cleaning atomic.Bool

	close chan<- struct{}
}

func New(cacheDir string, maxSize int64, maxUnusedTime time.Duration) (*Cache, error) {
	closeFn := make(chan struct{})
	c := &Cache{
		dir:           cacheDir,
		objects:       make(map[string]*Object),
		maxSize:       maxSize,
		maxUnusedTime: maxUnusedTime,
		close:         closeFn,
	}
	err := os.MkdirAll(cacheDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("new cache: %w", err)
	}
	err = c.loadObjects()
	if err != nil {
		return nil, fmt.Errorf("new cache: %w", err)
	}
	ticker := time.NewTicker(5 * time.Minute)
	go func() {
		c.clean()
	loop:
		for {
			select {
			case <-closeFn:
				break loop
			case <-ticker.C:
				c.clean()
			}
		}
		ticker.Stop()
	}()
	return c, nil
}

func (c *Cache) GetObject(key string) (*Object, bool) {
	c.objectsLock.RLock()
	defer c.objectsLock.RUnlock()
	o, ok := c.objects[key]
	return o, ok
}

func (c *Cache) CreateObject(key string) (*Object, error) {
	c.objectsLock.Lock()
	defer c.objectsLock.Unlock()
	if _, ok := c.objects[key]; ok {
		return nil, os.ErrExist
	}
	object, err := c.newCacheObject(key)
	if err != nil {
		return nil, fmt.Errorf("cache new object: %w", err)
	}
	c.objects[key] = object
	return object, nil
}

func (c *Cache) DeleteObject(key string) error {
	c.objectsLock.Lock()
	defer c.objectsLock.Unlock()
	o, ok := c.objects[key]
	if !ok {
		return nil
	}
	err := o.Close()
	if err != nil {
		return fmt.Errorf("cache: delete object: %w", err)
	}
	err = os.Remove(c.keyToPath(key))
	if err != nil {
		return fmt.Errorf("cache: delete object: %w", err)
	}
	delete(c.objects, key)
	if o.complete {
		err = os.Remove(c.keyToPath(key) + "-complete")
		if err != nil {
			return fmt.Errorf("cache: delete object: %w", err)
		}
	}
	return nil
}

func (c *Cache) Clear() error {
	c.objectsLock.Lock()
	defer c.objectsLock.Unlock()
	for key := range c.objects {
		o := c.objects[key]
		err := o.Close()
		if err != nil {
			log.Errorf("cache: clear: close object: %v", err)
			continue
		}
	}
	clear(c.objects)

	err := os.RemoveAll(c.dir)
	if err != nil {
		return fmt.Errorf("cache: clean cache: remove directory: %w", err)
	}
	err = os.MkdirAll(c.dir, 0755)
	if err != nil {
		return fmt.Errorf("cache: clean cache: re-create directory: %w", err)
	}
	return nil
}

// evict closes the object and removes it from the cache (memory and disk) if
// it has no active readers, reporting whether it was removed. It must be
// called while holding objectsLock.
func (c *Cache) evict(o *Object) bool {
	closed, err := o.closeForEviction()
	if err != nil {
		log.Errorf("cache: clean: close object: %s", err)
	}
	if !closed {
		return false
	}
	key := o.key
	if err := os.Remove(c.keyToPath(key)); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Errorf("cache: clean: remove object: %s", err)
	}
	if err := os.Remove(c.keyToPath(key) + "-complete"); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Warnf("failed to remove %s: %s", c.keyToPath(key)+"-complete", err)
	}
	delete(c.objects, key)
	return true
}

func (c *Cache) clean() {
	if !c.cleaning.CompareAndSwap(false, true) {
		return
	}
	defer c.cleaning.Store(false)
	log.Tracef("cleaning cache in %s...", c.dir)
	c.objectsLock.Lock()
	defer c.objectsLock.Unlock()

	type entry struct {
		obj      *Object
		size     int64
		accessed time.Time
	}

	var size int64
	entries := make([]entry, 0, len(c.objects))
	var largest *Object
	var largestSize int64
	for _, o := range c.objects {
		readerCount, objSize, accessed := o.stats()
		if readerCount <= 0 {
			if time.Since(accessed) > c.maxUnusedTime {
				c.evict(o)
				continue
			}
			if time.Since(accessed) > 30*time.Minute && (largest == nil || objSize > largestSize) {
				largest = o
				largestSize = objSize
			}
		}
		size += objSize
		entries = append(entries, entry{obj: o, size: objSize, accessed: accessed})
	}
	if size <= c.maxSize {
		log.Tracef("max cache size not reached yet: %d kB of %d kB", size/1000, c.maxSize/1000)
		return
	}
	oldSize := size
	if largest != nil && float64(c.maxSize)/float64(largestSize) > 0.3 {
		if c.evict(largest) {
			size -= largestSize
		}
	}
	slices.SortFunc(entries, func(a, b entry) int {
		return cmp.Compare(time.Since(b.accessed), time.Since(a.accessed))
	})
	for _, e := range entries {
		if largest != nil && e.obj.key == largest.key {
			continue
		}
		if size <= c.maxSize {
			break
		}
		if c.evict(e.obj) {
			size -= e.size
		}
	}
	log.Tracef("cleaned %d kB; new size: %d kB of %d kB", oldSize/1000-size/1000, size/1000, c.maxSize/1000)
}

func (c *Cache) loadObjects() error {
	c.objectsLock.Lock()
	defer c.objectsLock.Unlock()
	clear(c.objects)

	files, err := os.ReadDir(c.dir)
	if err != nil {
		return fmt.Errorf("load objects: %w", err)
	}
	for _, f := range files {
		if f.IsDir() || strings.HasSuffix(f.Name(), "-complete") {
			continue
		}
		key, err := c.keyFromPath(f.Name())
		if err != nil {
			log.Errorf("cache: load objects: %s", err)
			continue
		}
		if _, err := os.Stat(c.keyToPath(key) + "-complete"); err != nil {
			err := os.Remove(c.keyToPath(key))
			if err != nil {
				log.Warnf("failed to remove unfinished cache object %s: %s", c.keyToPath(key), err)
			}
			continue
		}
		i, err := f.Info()
		if err != nil {
			log.Errorf("cache: load objects: %s", err)
			continue
		}
		c.objects[key] = &Object{
			cache:       c,
			key:         key,
			size:        i.Size(),
			complete:    true,
			modified:    i.ModTime(),
			accessed:    time.Now(),
			dataChanged: make(chan struct{}),
		}
	}

	return nil
}

func (c *Cache) keyToPath(key string) string {
	return filepath.Join(c.dir, url.PathEscape(key))
}

func (c *Cache) keyFromPath(path string) (string, error) {
	key, err := url.PathUnescape(filepath.Base(path))
	if err != nil {
		return "", fmt.Errorf("key from path: %w", err)
	}
	return key, nil
}

func (c *Cache) Close() error {
	close(c.close)
	c.objectsLock.Lock()
	defer c.objectsLock.Unlock()
	for _, o := range c.objects {
		err := o.Close()
		if err != nil {
			log.Errorf("cache: close: %s", err)
		}
	}
	clear(c.objects)
	return nil
}

func (c *Cache) Keys() []string {
	c.objectsLock.RLock()
	defer c.objectsLock.RUnlock()
	keys := make([]string, 0, len(c.objects))
	for k := range c.objects {
		keys = append(keys, k)
	}
	return keys
}
