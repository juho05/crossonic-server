package cache

import (
	"context"
	"errors"
	"fmt"
	"io"
)

type cacheReader struct {
	obj    *Object
	offset int64
	ctx    context.Context
}

func (c *Object) newCacheReader(ctx context.Context) (*cacheReader, error) {
	err := c.UseFile()
	if err != nil {
		return nil, fmt.Errorf("new cache reader: %w", err)
	}
	return &cacheReader{
		obj:    c,
		offset: 0,
		ctx:    ctx,
	}, nil
}

func (c *cacheReader) Read(p []byte) (n int, err error) {
	c.obj.lock.RLock()
	if err := c.waitForData(c.offset); err != nil {
		c.obj.lock.RUnlock()
		return 0, err
	}
	size := min(int64(len(p)), c.obj.size-c.offset)
	n, err = c.obj.file.ReadAt(p[:size], c.offset)
	c.offset += int64(n)
	c.obj.lock.RUnlock()
	if errors.Is(err, io.EOF) {
		err = nil
	}
	return n, err
}

func (c *cacheReader) ReadAt(p []byte, off int64) (n int, err error) {
	c.obj.lock.RLock()
	if err := c.waitForData(off); err != nil {
		c.obj.lock.RUnlock()
		return 0, err
	}
	size := min(int64(len(p)), c.obj.size-off)
	n, err = c.obj.file.ReadAt(p[:size], off)
	c.obj.lock.RUnlock()
	if errors.Is(err, io.EOF) {
		err = nil
	}
	return n, err
}

// waitForData blocks until at least one byte is readable at off. It must be
// called with the read lock held and returns with the lock still held; on a
// nil error the caller may read (c.obj.file is non-nil and c.obj.size > off).
// It returns io.EOF once a complete object has been fully consumed, an error
// if the underlying file was closed while the object was still incomplete, or
// the reader's context error if it is cancelled while waiting (e.g. the client
// disconnected mid-stream).
//
// While waiting it releases the read lock and blocks on the object's
// dataChanged channel, which Write/SetComplete/Close close on every relevant
// state change. This replaces the previous (0, nil) return that busy-spun the
// io.Copy loop for incomplete objects.
func (c *cacheReader) waitForData(off int64) error {
	for c.obj.size-off <= 0 {
		if c.obj.complete {
			return io.EOF
		}
		if c.obj.file == nil {
			// file was deleted non-gracefully -> abort
			return fmt.Errorf("cache reader is closed")
		}
		// Capture the channel under the lock: signalDataChanged closes the
		// current channel and installs a new one, so reading the field after
		// unlocking could miss a wakeup (and would race the writer's reassign).
		wait := c.obj.dataChanged
		c.obj.lock.RUnlock()
		select {
		case <-wait:
		case <-c.ctx.Done():
			c.obj.lock.RLock()
			return c.ctx.Err()
		}
		c.obj.lock.RLock()
	}
	if c.obj.file == nil {
		return fmt.Errorf("cache reader is closed")
	}
	return nil
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
