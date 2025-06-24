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
	"time"

	"github.com/juho05/log"
)

type Cache struct {
	dir string

	maxSize       int64
	maxUnusedTime time.Duration

	objects     map[string]*Object
	objectsLock sync.RWMutex

	cleaning bool

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
	c.objectsLock.RLock()
	if _, ok := c.objects[key]; ok {
		c.objectsLock.RUnlock()
		return nil, os.ErrExist
	}
	c.objectsLock.RUnlock()
	object, err := c.newCacheObject(key)
	if err != nil {
		return nil, fmt.Errorf("cache new object: %w", err)
	}
	c.objectsLock.Lock()
	c.objects[key] = object
	c.objectsLock.Unlock()
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

func (c *Cache) InvalidateGracefully() error {
	c.objectsLock.Lock()
	defer c.objectsLock.Unlock()
	for key, o := range c.objects {
		if o.readerCount > 0 {
			o.delete = true
			continue
		}
		err := o.Close()
		if err != nil {
			return fmt.Errorf("cache: invalidate gracefully: delete object: %w", err)
		}
		err = os.Remove(c.keyToPath(key))
		if err != nil {
			return fmt.Errorf("cache: invalidate gracefully: delete object: %w", err)
		}
		delete(c.objects, key)
		if o.complete {
			err = os.Remove(c.keyToPath(key) + "-complete")
			if err != nil {
				return fmt.Errorf("cache: invalidate gracefully: delete object: %w", err)
			}
		}
	}
	return nil
}

func (c *Cache) clean() {
	if c.cleaning {
		return
	}
	c.cleaning = true
	defer func() {
		c.cleaning = false
	}()
	log.Tracef("cleaning cache in %s...", c.dir)
	c.objectsLock.Lock()
	defer c.objectsLock.Unlock()
	var size int64
	objects := make([]*Object, 0, len(c.objects))
	var largest *Object
	for key, o := range c.objects {
		if o.readerCount <= 0 {
			if o.delete || time.Since(o.accessed) > c.maxUnusedTime {
				err := o.Close()
				if err != nil {
					log.Errorf("cache: clean: %s", err)
				}
				err = os.Remove(c.keyToPath(key))
				if err != nil {
					log.Errorf("cache: clean: %s", err)
				}
				err = os.Remove(c.keyToPath(key) + "-complete")
				if err != nil && !errors.Is(err, os.ErrNotExist) {
					log.Warnf("failed to remove %s: %s", c.keyToPath(key)+"-complete", err)
				}
				delete(c.objects, key)
				continue
			}
			if time.Since(o.accessed) > 30*time.Minute && (largest == nil || o.size > largest.size) {
				largest = o
			}
		}
		size += o.size
		objects = append(objects, o)
	}
	if size <= c.maxSize {
		log.Tracef("max cache size not reached yet: %d kB of %d kB", size/1000, c.maxSize/1000)
		return
	}
	oldSize := size
	if largest != nil && float64(c.maxSize)/float64(largest.size) > 0.3 {
		err := largest.Close()
		if err != nil {
			log.Errorf("cache: clean: %s", err)
		}
		err = os.Remove(c.keyToPath(largest.key))
		if err != nil {
			log.Errorf("cache: clean: %s", err)
		}
		err = os.Remove(c.keyToPath(largest.key) + "-complete")
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			log.Warnf("failed to remove %s: %s", c.keyToPath(largest.key)+"-complete", err)
		}
		delete(c.objects, largest.key)
		size -= largest.size
	}
	slices.SortFunc(objects, func(a, b *Object) int {
		return cmp.Compare(time.Since(b.accessed), time.Since(a.accessed))
	})
	for _, o := range objects {
		if (largest != nil && o.key == largest.key) || o.readerCount > 0 {
			continue
		}
		if size <= c.maxSize {
			break
		}
		err := o.Close()
		if err != nil {
			log.Errorf("cache: clean: %s", err)
		}
		err = os.Remove(c.keyToPath(o.key))
		if err != nil {
			log.Errorf("cache: clean: %s", err)
		}
		delete(c.objects, o.key)
		size -= o.size
	}
	log.Tracef("cleaned %d kB; new size: %d kB of %d kB", oldSize/1000, size/1000, c.maxSize/1000)
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
			cache:    c,
			key:      key,
			size:     i.Size(),
			complete: true,
			modified: i.ModTime(),
			accessed: time.Now(),
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
