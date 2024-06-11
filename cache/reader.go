package cache

import (
	"errors"
	"fmt"
	"io"
)

type cacheReader struct {
	obj    *CacheObject
	offset int64
}

func (c *CacheObject) newCacheReader() (*cacheReader, error) {
	err := c.UseFile()
	if err != nil {
		return nil, fmt.Errorf("new cache reader: %w", err)
	}
	return &cacheReader{
		obj:    c,
		offset: 0,
	}, nil
}

func (c *cacheReader) Read(p []byte) (n int, err error) {
	c.obj.lock.RLock()
	defer c.obj.lock.RUnlock()
	if c.obj.size-c.offset <= 0 {
		if c.obj.complete {
			return 0, io.EOF
		}
		return 0, nil
	}
	size := min(int64(len(p)), c.obj.size-c.offset)
	n, err = c.obj.file.ReadAt(p[:size], c.offset)
	c.offset += int64(n)
	if errors.Is(err, io.EOF) {
		err = nil
	}
	return n, err
}

func (c *cacheReader) ReadAt(p []byte, off int64) (n int, err error) {
	c.obj.lock.RLock()
	defer c.obj.lock.RUnlock()
	if c.obj.size-off <= 0 {
		if c.obj.complete {
			return 0, io.EOF
		}
		return 0, nil
	}
	size := min(int64(len(p)), c.obj.size-off)
	n, err = c.obj.file.ReadAt(p[:size], off)
	if errors.Is(err, io.EOF) {
		err = nil
	}
	return n, err
}

func (c *cacheReader) Seek(offset int64, whence int) (int64, error) {
	c.obj.lock.RLock()
	defer c.obj.lock.RUnlock()
	if !c.obj.complete {
		return 0, fmt.Errorf("cacheReader: seek: not available: %w", ErrNotComplete)
	}
	switch whence {
	case io.SeekCurrent:
		offset = c.offset + offset
	case io.SeekStart:
		break
	case io.SeekEnd:
		offset = c.obj.size + offset
	default:
		return 0, fmt.Errorf("cacheReader: seek: invalid whence")
	}
	if offset < 0 {
		return 0, fmt.Errorf("cacheReader: seek: negative offset")
	}
	c.offset = offset
	return c.offset, nil
}

func (c *cacheReader) Close() error {
	err := c.obj.ReleaseFile()
	if err != nil {
		return fmt.Errorf("cache reader close: %w", err)
	}
	return nil
}
