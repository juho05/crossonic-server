package cache

import (
	"io"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testCache returns a cache rooted in a temporary directory that is
// automatically closed when the test finishes.
func testCache(t *testing.T, maxSize int64, maxUnused time.Duration) *Cache {
	t.Helper()
	c, err := New(t.TempDir(), maxSize, maxUnused)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = c.Close()
	})
	return c
}

// makeComplete creates an object, writes data to it and marks it complete.
func makeComplete(t *testing.T, c *Cache, key, data string) *Object {
	t.Helper()
	o, err := c.CreateObject(key)
	require.NoError(t, err)
	_, err = o.Write([]byte(data))
	require.NoError(t, err)
	require.NoError(t, o.SetComplete())
	return o
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func objectSize(o *Object) int64 {
	_, size, _ := o.stats()
	return size
}

// --- Cache ---------------------------------------------------------------

func TestNewCreatesDir(t *testing.T) {
	dir := t.TempDir() + "/nested/cache"
	c, err := New(dir, 1<<30, time.Hour)
	require.NoError(t, err)
	defer c.Close()
	info, err := os.Stat(dir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestCreateObject(t *testing.T) {
	c := testCache(t, 1<<30, time.Hour)

	o, err := c.CreateObject("a")
	require.NoError(t, err)
	assert.Equal(t, "a", o.Key())
	assert.True(t, fileExists(c.keyToPath("a")))

	_, err = c.CreateObject("a")
	assert.ErrorIs(t, err, os.ErrExist)
}

func TestGetObject(t *testing.T) {
	c := testCache(t, 1<<30, time.Hour)

	_, ok := c.GetObject("missing")
	assert.False(t, ok)

	makeComplete(t, c, "a", "data")
	o, ok := c.GetObject("a")
	assert.True(t, ok)
	assert.Equal(t, "a", o.Key())
}

func TestDeleteObject(t *testing.T) {
	c := testCache(t, 1<<30, time.Hour)
	makeComplete(t, c, "a", "data")
	path := c.keyToPath("a")

	require.NoError(t, c.DeleteObject("a"))
	_, ok := c.GetObject("a")
	assert.False(t, ok)
	assert.False(t, fileExists(path))
	assert.False(t, fileExists(path+"-complete"))

	// deleting a missing key is a no-op, not an error
	assert.NoError(t, c.DeleteObject("missing"))
}

func TestKeys(t *testing.T) {
	c := testCache(t, 1<<30, time.Hour)
	makeComplete(t, c, "a", "x")
	makeComplete(t, c, "b", "y")
	assert.ElementsMatch(t, []string{"a", "b"}, c.Keys())
}

func TestClear(t *testing.T) {
	c := testCache(t, 1<<30, time.Hour)
	makeComplete(t, c, "a", "x")
	makeComplete(t, c, "b", "y")

	require.NoError(t, c.Clear())
	assert.Empty(t, c.Keys())

	info, err := os.Stat(c.dir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

// TestLoadObjects verifies that a freshly opened cache picks up complete
// objects from disk and discards unfinished ones (those without a -complete
// marker).
func TestLoadObjects(t *testing.T) {
	dir := t.TempDir()

	c, err := New(dir, 1<<30, time.Hour)
	require.NoError(t, err)
	makeComplete(t, c, "keep", "data")

	incomplete, err := c.CreateObject("incomplete")
	require.NoError(t, err)
	_, err = incomplete.Write([]byte("partial"))
	require.NoError(t, err)
	require.NoError(t, c.Close())

	c2, err := New(dir, 1<<30, time.Hour)
	require.NoError(t, err)
	defer c2.Close()

	keep, ok := c2.GetObject("keep")
	require.True(t, ok)
	assert.True(t, keep.IsComplete())
	assert.Equal(t, int64(4), objectSize(keep))

	_, ok = c2.GetObject("incomplete")
	assert.False(t, ok)
	assert.False(t, fileExists(c2.keyToPath("incomplete")))
}

// TestKeyEscaping ensures keys with characters that are unsafe for file names
// survive a round trip through the filesystem.
func TestKeyEscaping(t *testing.T) {
	dir := t.TempDir()
	const key = "ab/cd ef-100"

	c, err := New(dir, 1<<30, time.Hour)
	require.NoError(t, err)
	makeComplete(t, c, key, "data")
	require.NoError(t, c.Close())

	c2, err := New(dir, 1<<30, time.Hour)
	require.NoError(t, err)
	defer c2.Close()

	o, ok := c2.GetObject(key)
	require.True(t, ok)
	assert.Equal(t, key, o.Key())
}

// --- clean ---------------------------------------------------------------

func TestCleanEvictsUnused(t *testing.T) {
	c := testCache(t, 1<<30, time.Millisecond)
	makeComplete(t, c, "a", "hello")
	path := c.keyToPath("a")

	time.Sleep(5 * time.Millisecond)
	c.clean()

	_, ok := c.GetObject("a")
	assert.False(t, ok)
	assert.False(t, fileExists(path))
	assert.False(t, fileExists(path+"-complete"))
}

func TestCleanKeepsObjectsWithActiveReaders(t *testing.T) {
	c := testCache(t, 1<<30, time.Millisecond)
	o := makeComplete(t, c, "a", "hello")

	r, err := o.Reader()
	require.NoError(t, err)

	time.Sleep(5 * time.Millisecond)
	c.clean()
	_, ok := c.GetObject("a")
	assert.True(t, ok, "object with an active reader must not be evicted")

	require.NoError(t, r.Close())
	time.Sleep(5 * time.Millisecond)
	c.clean()
	_, ok = c.GetObject("a")
	assert.False(t, ok, "object must be evicted once the reader is released")
}

// TestCleanSizeEviction verifies that, when the cache exceeds maxSize, the
// least recently accessed objects are evicted first.
func TestCleanSizeEviction(t *testing.T) {
	// maxUnused is huge so only the size-based eviction path runs.
	c := testCache(t, 30, time.Hour)
	const payload = "01234567890123456789" // 20 bytes

	makeComplete(t, c, "a", payload)
	time.Sleep(2 * time.Millisecond)
	makeComplete(t, c, "b", payload)
	time.Sleep(2 * time.Millisecond)
	makeComplete(t, c, "c", payload)

	c.clean()

	// total 60 bytes > 30: evict oldest (a, then b) until at/under maxSize.
	_, okA := c.GetObject("a")
	_, okB := c.GetObject("b")
	_, okC := c.GetObject("c")
	assert.False(t, okA, "oldest object should be evicted")
	assert.False(t, okB, "second oldest object should be evicted")
	assert.True(t, okC, "newest object should be kept")
}

// --- Object: read/write --------------------------------------------------

// TestReaderIncremental exercises reading from an object while it is still
// being written: reads return only the bytes available so far (0, nil when
// nothing new) and io.EOF once the object is complete and drained.
func TestReaderIncremental(t *testing.T) {
	c := testCache(t, 1<<30, time.Hour)
	o, err := c.CreateObject("a")
	require.NoError(t, err)
	_, err = o.Write([]byte("hello"))
	require.NoError(t, err)

	r, err := o.Reader()
	require.NoError(t, err)
	defer r.Close()

	buf := make([]byte, 16)

	n, err := r.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, "hello", string(buf[:n]))

	// nothing new and not complete -> (0, nil)
	n, err = r.Read(buf)
	assert.NoError(t, err)
	assert.Equal(t, 0, n)

	_, err = o.Write([]byte("!"))
	require.NoError(t, err)
	n, err = r.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, "!", string(buf[:n]))

	require.NoError(t, o.SetComplete())
	n, err = r.Read(buf)
	assert.Equal(t, 0, n)
	assert.ErrorIs(t, err, io.EOF)
}

func TestReaderComplete(t *testing.T) {
	c := testCache(t, 1<<30, time.Hour)
	o := makeComplete(t, c, "a", "hello world")

	r, err := o.Reader()
	require.NoError(t, err)
	defer r.Close()

	data, err := io.ReadAll(r)
	require.NoError(t, err)
	assert.Equal(t, "hello world", string(data))
}

func TestWriteAfterComplete(t *testing.T) {
	c := testCache(t, 1<<30, time.Hour)
	o := makeComplete(t, c, "a", "x")

	_, err := o.Write([]byte("y"))
	assert.ErrorIs(t, err, ErrComplete)
}

func TestReadAt(t *testing.T) {
	c := testCache(t, 1<<30, time.Hour)
	o := makeComplete(t, c, "a", "0123456789")

	r, err := o.Reader()
	require.NoError(t, err)
	defer r.Close()

	ra, ok := r.(io.ReaderAt)
	require.True(t, ok)

	buf := make([]byte, 4)
	n, err := ra.ReadAt(buf, 3)
	require.NoError(t, err)
	assert.Equal(t, 4, n)
	assert.Equal(t, "3456", string(buf))
}

// --- Object: seek --------------------------------------------------------

func TestSeek(t *testing.T) {
	c := testCache(t, 1<<30, time.Hour)
	o := makeComplete(t, c, "a", "0123456789")

	r, err := o.Reader()
	require.NoError(t, err)
	defer r.Close()

	pos, err := r.Seek(2, io.SeekStart)
	require.NoError(t, err)
	assert.Equal(t, int64(2), pos)
	data, err := io.ReadAll(r)
	require.NoError(t, err)
	assert.Equal(t, "23456789", string(data))

	pos, err = r.Seek(-2, io.SeekEnd)
	require.NoError(t, err)
	assert.Equal(t, int64(8), pos)
	buf := make([]byte, 4)
	n, err := r.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, "89", string(buf[:n]))

	_, err = r.Seek(0, io.SeekStart)
	require.NoError(t, err)
	pos, err = r.Seek(3, io.SeekCurrent)
	require.NoError(t, err)
	assert.Equal(t, int64(3), pos)

	_, err = r.Seek(-1, io.SeekStart)
	assert.Error(t, err)

	_, err = r.Seek(0, 99)
	assert.Error(t, err)
}

func TestSeekNotComplete(t *testing.T) {
	c := testCache(t, 1<<30, time.Hour)
	o, err := c.CreateObject("a")
	require.NoError(t, err)
	_, err = o.Write([]byte("data"))
	require.NoError(t, err)

	r, err := o.Reader()
	require.NoError(t, err)
	defer r.Close()

	_, err = r.Seek(0, io.SeekStart)
	assert.ErrorIs(t, err, ErrNotComplete)
}

// --- Object: reference counting -----------------------------------------

func TestUseReleaseFileRefCount(t *testing.T) {
	c := testCache(t, 1<<30, time.Hour)
	o := makeComplete(t, c, "a", "data")

	require.NoError(t, o.UseFile())
	require.NoError(t, o.UseFile())

	require.NoError(t, o.ReleaseFile())
	// still one reader -> the file may not be evicted yet
	closed, err := o.closeForEviction()
	require.NoError(t, err)
	assert.False(t, closed)

	require.NoError(t, o.ReleaseFile())
	// no readers left -> eviction is allowed
	closed, err = o.closeForEviction()
	require.NoError(t, err)
	assert.True(t, closed)

	// releasing more than was used is an error
	assert.Error(t, o.ReleaseFile())
}

// TestReaderAbortsWhenFileClosed covers the fix that prevents a nil-pointer
// panic when an object's file is closed (e.g. a non-graceful delete) while a
// reader is still active.
func TestReaderAbortsWhenFileClosed(t *testing.T) {
	c := testCache(t, 1<<30, time.Hour)
	o := makeComplete(t, c, "a", "hello world")

	r, err := o.Reader()
	require.NoError(t, err)
	defer r.Close()

	// close the underlying file out from under the active reader
	require.NoError(t, o.Close())

	buf := make([]byte, 4)
	_, err = r.Read(buf)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "closed")

	_, err = r.(io.ReaderAt).ReadAt(buf, 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "closed")
}

// --- Object: misc accessors ---------------------------------------------

func TestObjectAccessors(t *testing.T) {
	c := testCache(t, 1<<30, time.Hour)
	o := makeComplete(t, c, "mykey", "data")

	assert.Equal(t, "mykey", o.Key())
	assert.Equal(t, c.keyToPath("mykey"), o.Path())
	assert.True(t, o.IsComplete())
	assert.False(t, o.Modified().IsZero())
	assert.True(t, fileExists(c.keyToPath("mykey")+"-complete"))
}

// --- Concurrency ---------------------------------------------------------

// TestConcurrentReadWriteClean stresses the cache with many concurrent
// readers, a writer and the cleaner running in parallel. Run with -race to
// detect data races and use-after-close.
func TestConcurrentReadWriteClean(t *testing.T) {
	// small maxSize + tiny maxUnused so the cleaner aggressively tries to
	// evict while readers/writers are active.
	c := testCache(t, 64, time.Nanosecond)

	const (
		key     = "key"
		payload = "concurrent payload"
		workers = 20
	)

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			obj, err := c.CreateObject(key)
			if err != nil {
				// someone else owns the object; just try to read it
				if obj, ok := c.GetObject(key); ok {
					if r, err := obj.Reader(); err == nil {
						_, _ = io.Copy(io.Discard, r)
						_ = r.Close()
					}
				}
				return
			}

			// writer
			go func() {
				for _, b := range []byte(payload) {
					_, _ = obj.Write([]byte{b})
					time.Sleep(time.Millisecond)
				}
				_ = obj.SetComplete()
			}()

			// concurrent readers on the same object
			var rwg sync.WaitGroup
			for j := 0; j < 5; j++ {
				rwg.Add(1)
				go func() {
					defer rwg.Done()
					r, err := obj.Reader()
					if err != nil {
						return
					}
					defer r.Close()
					_, _ = io.Copy(io.Discard, r)
				}()
			}
			rwg.Wait()
		}()
	}

	// dedicated cleaner, mirroring the production ticker (one clean at a time)
	stop := make(chan struct{})
	go func() {
		for {
			select {
			case <-stop:
				return
			default:
				c.clean()
				time.Sleep(time.Millisecond)
			}
		}
	}()

	wg.Wait()
	close(stop)
}
