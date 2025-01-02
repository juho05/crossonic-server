package cache

import (
	"cmp"
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

	objects     map[string]*CacheObject
	objectsLock sync.RWMutex

	cleaning bool

	close chan<- struct{}
}

func New(cacheDir string, maxSize int64, maxUnusedTime time.Duration) (*Cache, error) {
	close := make(chan struct{})
	c := &Cache{
		dir:           cacheDir,
		objects:       make(map[string]*CacheObject),
		maxSize:       maxSize,
		maxUnusedTime: maxUnusedTime,
		close:         close,
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
			case <-close:
				break loop
			case <-ticker.C:
				c.clean()
			}
		}
		ticker.Stop()
	}()
	return c, nil
}

func (c *Cache) GetObject(key string) (*CacheObject, bool, error) {
	c.objectsLock.RLock()
	if o, ok := c.objects[key]; ok {
		c.objectsLock.RUnlock()
		return o, true, nil
	}
	c.objectsLock.RUnlock()
	object, err := c.newCacheObject(key)
	if err != nil {
		return nil, false, fmt.Errorf("cache new object: %w", err)
	}
	c.objectsLock.Lock()
	c.objects[key] = object
	c.objectsLock.Unlock()
	return object, false, nil
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
	objects := make([]*CacheObject, 0, len(c.objects))
	var largest *CacheObject
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
				os.Remove(c.keyToPath(key) + "-complete")
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
		os.Remove(c.keyToPath(largest.key) + "-complete")
		delete(c.objects, largest.key)
		size -= largest.size
	}
	slices.SortFunc(objects, func(a, b *CacheObject) int {
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
			log.Errorf("cache: load objects: %w", err)
			continue
		}
		if _, err := os.Stat(c.keyToPath(key) + "-complete"); err != nil {
			os.Remove(c.keyToPath(key))
			os.Remove(c.keyToPath(key) + "-complete")
			continue
		}
		i, err := f.Info()
		if err != nil {
			log.Errorf("cache: load objects: %w", err)
			continue
		}
		c.objects[key] = &CacheObject{
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
