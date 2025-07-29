package scanner

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"mime"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/djherbis/times"
	"github.com/juho05/crossonic-server/audiotags"
	"github.com/juho05/crossonic-server/repos"
	"github.com/juho05/log"
)

const maxParallelDirs = 10
const updateSongFilesWorkerCount = 10
const setAlbumCoversWorkerCount = 10
const songQueueBatchSize = 100
const deleteOrphanedSongsByPathWorkerCount = 10
const deleteOrphanedSongsByPathBatchSize = 300

var coverImageNames = []string{"front", "folder", "cover"}

func (s *Scanner) Scan(db repos.DB, fullScan bool) (err error) {
	if !s.lock.TryLock() {
		return ErrAlreadyScanning
	}
	s.scanning = true
	s.fullScan = fullScan
	defer s.lock.Unlock()
	defer func() {
		if !s.songQueueClosed {
			close(s.songQueue)
		}
		if !s.setAlbumCoverClosed {
			close(s.setAlbumCover)
		}
		s.scanning = false
		s.albums = nil
		s.artists = nil
		s.songQueueClosed = true
		s.setAlbumCoverClosed = true
	}()

	s.scanStart = time.Now()
	previousCount := s.counter.Load()
	defer func() {
		if err != nil {
			s.counter.Store(previousCount)
		}
	}()
	s.counter.Store(0)

	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()

	s.tx, err = db.NewTransaction(ctx)
	if err != nil {
		return fmt.Errorf("new transaction: %w", err)
	}
	defer func() {
		err := s.tx.Rollback()
		if err != nil && !errors.Is(err, sql.ErrTxDone) {
			log.Errorf("rollback full scan tx: %s", err)
		}
		s.tx = nil
	}()

	s.lastScan, err = db.System().LastScan(ctx)
	if err != nil && !errors.Is(err, repos.ErrNotFound) {
		return fmt.Errorf("get last scan: %w", err)
	}
	if errors.Is(err, repos.ErrNotFound) {
		s.fullScan = true
		s.firstScan = true
		log.Infof("No last scan time found: enabling full scan and marking as first scan")
	} else {
		songCount, err := s.tx.Song().Count(ctx)
		if err != nil {
			return fmt.Errorf("get song count: %w", err)
		}
		s.firstScan = songCount == 0
		if s.firstScan {
			log.Tracef("detected first scan")
		}
	}

	if !s.fullScan {
		needsFullScan, err := s.tx.System().NeedsFullScan(ctx)
		if err != nil {
			return fmt.Errorf("check if full scan is needed: %w", err)
		}
		s.fullScan = needsFullScan
		err = s.tx.System().ResetNeedsFullScan(ctx)
		if err != nil {
			return fmt.Errorf("reset needs full scan: %w", err)
		}
	}

	if s.fullScan || s.firstScan {
		s.lastScan = time.Time{}
	}

	log.Infof("Scanning %s (full scan: %t)...", s.mediaDir, s.fullScan)

	log.Tracef("loading artist map from db...")
	s.artists, err = newArtistMapFromDB(ctx, s)
	if err != nil {
		return fmt.Errorf("new artist map from db: %w", err)
	}

	log.Tracef("loading album map from db...")
	s.albums, err = newAlbumMapFromDB(ctx, s)
	if err != nil {
		return fmt.Errorf("new album map from db: %w", err)
	}

	s.songQueue = make(chan *mediaFile, songQueueBatchSize)
	s.songQueueClosed = false
	s.setAlbumCover = make(chan albumCover, setAlbumCoversWorkerCount)
	s.setAlbumCoverClosed = false

	saveSongsDone := make(chan error, 1)
	log.Tracef("starting save songs loop with batch size %d...", songQueueBatchSize)
	go func() {
		err := s.runSaveSongsLoop(ctx)
		saveSongsDone <- err
		close(s.setAlbumCover)
		s.setAlbumCoverClosed = true
		if err != nil {
			log.Errorf("scan: run save songs loop: %s", err)
			cancelCtx()
		}
	}()

	setAlbumCovers := make(chan error, 1)
	log.Tracef("starting set album covers loop with %d workers...", setAlbumCoversWorkerCount)
	go func() {
		err := s.runSetAlbumCoverLoop(ctx)
		setAlbumCovers <- err
		if err != nil {
			log.Errorf("scan: set album covers loop: %s", err)
			cancelCtx()
		}
	}()

	log.Tracef("scanning media dir with %d workers...", maxParallelDirs)
	err = s.scanMediaDir(ctx)
	close(s.songQueue)
	s.songQueueClosed = true
	if err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("scan dir: %w", err)
	}

	log.Tracef("waiting until save songs worker is done...")
	err = <-saveSongsDone
	if err != nil {
		return fmt.Errorf("run save songs loop: %w", err)
	}

	log.Tracef("waiting until set album covers worker is done...")
	err = <-setAlbumCovers
	if err != nil {
		return fmt.Errorf("run set album covers loop: %w", err)
	}

	log.Tracef("updating album artists...")
	err = s.albums.updateArtists(ctx, s)
	if err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("update album artists: %w", err)
	}

	log.Tracef("deleting orphaned songs/albums/artists...")
	err = s.deleteOrphaned(ctx)
	if err != nil {
		return fmt.Errorf("delete orphaned: %w", err)
	}

	select {
	case <-ctx.Done():
		return context.Canceled
	default:
	}

	log.Tracef("fixing track numbers in playlists...")
	err = s.tx.Playlist().FixTrackNumbers(ctx)
	if err != nil {
		return fmt.Errorf("fix playlist track numbers: %w", err)
	}

	log.Tracef("calculating fallback gain...")
	fallbackGain, err := s.tx.Song().GetMedianReplayGain(ctx)
	if err != nil {
		return fmt.Errorf("get median replay gain: %w", err)
	}
	if fallbackGain != 0 {
		repos.SetFallbackGain(fallbackGain)
	}

	err = s.tx.System().SetLastScan(ctx, time.Now())
	if err != nil {
		return fmt.Errorf("update last scan: %w", err)
	}

	log.Tracef("committing changes...")
	err = s.tx.Commit()
	if err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	log.Infof("Scanned %d files in %s.", s.counter.Load(), time.Since(s.scanStart).Round(time.Millisecond))
	return nil
}

func (s *Scanner) scanMediaDir(ctx context.Context) error {
	type dir struct {
		changed bool
		path    string
	}
	dirs := make(chan dir, maxParallelDirs)

	var waitGroup sync.WaitGroup
	var scanDirError error
	for range maxParallelDirs {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			for dir := range dirs {
				err := s.scanMediaFilesInDir(ctx, dir.path, dir.changed)
				if err != nil {
					log.Errorf("scan media dir: %s", err)
					if scanDirError == nil {
						scanDirError = err
					}
					return
				}
			}
		}()
	}

	info, err := os.Lstat(s.mediaDir)
	if err != nil {
		return fmt.Errorf("stat media dir: %w", err)
	}
	err = s.walkDir(s.mediaDir, fs.FileInfoToDirEntry(info), s.checkIfChanged(s.mediaDir, info), func(path string, d fs.DirEntry, parentChanged bool, err error) error {
		if err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return filepath.SkipAll
		default:
		}
		if !d.IsDir() {
			return nil
		}
		if scanDirError != nil {
			return filepath.SkipAll
		}
		if !s.conf.ScanHidden && d.Name()[0] == '.' {
			return filepath.SkipDir
		}
		dirs <- dir{
			changed: parentChanged || s.checkIfChangedByPath(path),
			path:    path,
		}
		return nil
	})
	if err != nil {
		if !errors.Is(err, filepath.SkipDir) && !errors.Is(err, filepath.SkipAll) {
			return fmt.Errorf("walk media dir: %w", err)
		}
	}

	close(dirs)
	waitGroup.Wait()
	if err != nil {
		return fmt.Errorf("walk dir: %w", err)
	}

	if scanDirError != nil {
		return fmt.Errorf("scan media files in dir: %w", scanDirError)
	}

	return nil
}

// modified version of filepath.WalkDir
func (s *Scanner) walkDir(path string, d fs.DirEntry, parentChanged bool, walkDirFn func(path string, d fs.DirEntry, parentChanged bool, err error) error) error {
	changed := parentChanged
	if !changed {
		stat, err := times.Stat(path)
		if err != nil {
			return err
		}
		changed = stat.ModTime().After(s.lastScan) || !stat.HasChangeTime() || stat.ChangeTime().After(s.lastScan)
	}
	if err := walkDirFn(path, d, changed, nil); err != nil || !d.IsDir() {
		if errors.Is(err, filepath.SkipDir) && d.IsDir() {
			// Successfully skipped directory.
			err = nil
		}
		return err
	}

	dirs, err := os.ReadDir(path)
	if err != nil {
		// Second call, to report ReadDir error.
		err = walkDirFn(path, d, changed, err)
		if err != nil {
			if errors.Is(err, filepath.SkipDir) && d.IsDir() {
				err = nil
			}
			return err
		}
	}

	for _, d1 := range dirs {
		path1 := filepath.Join(path, d1.Name())
		if err := s.walkDir(path1, d1, changed, walkDirFn); err != nil {
			if errors.Is(err, filepath.SkipDir) {
				break
			}
			return err
		}
	}
	return nil
}

func (s *Scanner) scanMediaFilesInDir(ctx context.Context, dir string, changed bool) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read dir: %w", err)
	}

	var cover *string
findCoverLoop:
	for _, e := range entries {
		if !e.Type().IsRegular() {
			continue
		}

		if !s.conf.ScanHidden && e.Name()[0] == '.' {
			continue
		}

		ext := filepath.Ext(e.Name())
		fileType := mime.TypeByExtension(ext)
		if fileType == "image/jpeg" || fileType == "image/png" {
			base := strings.ToLower(strings.TrimSuffix(e.Name(), ext))
			for i := 0; i < len(coverImageNames); i++ {
				if base == coverImageNames[i] {
					c := filepath.Join(dir, e.Name())
					cover = &c
					break findCoverLoop
				}
			}
		}
	}

	for _, e := range entries {
		if !e.Type().IsRegular() {
			continue
		}

		select {
		case <-ctx.Done():
			return context.Canceled
		default:
		}

		err := s.processFile(filepath.Join(dir, e.Name()), cover, changed)
		if errors.Is(err, errNotAMediaFile) {
			continue
		}
		if err != nil {
			return fmt.Errorf("process file: %w", err)
		}
	}
	return nil
}

var errNotAMediaFile = errors.New("not a media file")

func (s *Scanner) processFile(path string, cover *string, parentDirChanged bool) error {
	ext := filepath.Ext(path)
	if !strings.HasPrefix(mime.TypeByExtension(ext), "audio/") {
		return errNotAMediaFile
	}

	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat: %w", err)
	}
	if info.IsDir() {
		return errNotAMediaFile
	}
	contentType := mime.TypeByExtension(ext)
	if !strings.HasPrefix(contentType, "audio/") {
		return errNotAMediaFile
	}

	s.counter.Add(1)

	if !s.fullScan && !parentDirChanged && info.ModTime().Before(s.lastScan) {
		timeStat, err := times.Stat(path)
		if err != nil {
			return fmt.Errorf("times stat: %w", err)
		}
		if timeStat.HasChangeTime() && timeStat.ChangeTime().Before(s.lastScan) {
			return nil
		}
	}

	file, err := audiotags.Open(path)
	if err != nil {
		return fmt.Errorf("read tags: %w", err)
	}
	defer file.Close()
	if !file.HasMedia() {
		return errNotAMediaFile
	}

	props := file.ReadAudioProperties()
	tags := file.ReadTags()

	var songID *string
	idTag, ok := readSingleTag(tags, "crossonic_id_"+s.instanceID)
	if ok && strings.HasPrefix(idTag, "tr_") {
		songID = &idTag
	}

	title, ok := readSingleTag(tags, "title")
	if !ok {
		title = strings.TrimSuffix(filepath.Base(path), ext)
	}

	var lyrics *string
	if l, ok := tags["lyrics"]; ok {
		ly := strings.Join(l, "\n")
		lyrics = &ly
	} else if l, ok := tags["unsyncedlyrics"]; ok {
		ly := strings.Join(l, "\n")
		lyrics = &ly
	}

	isCompilation := readSingleBoolTag(tags, "compilation")

	artists := readStringTags(tags, "artists", "artist")
	albumArtists := readStringTags(tags, "albumartists", "album_artists", "albumartist", "album_artist")
	if !isCompilation && len(albumArtists) == 0 && len(artists) > 0 {
		albumArtists = []string{artists[0]}
	}

	artistMBIDs := readStringTags(tags, "musicbrainz_artistids", "musicbrainz_artistid")
	albumArtistMBIDs := readStringTags(tags, "musicbrainz_albumartistids", "musicbrainz_albumartistid")

	album := readSingleTagOptional(tags, "album")

	year := readSingleIntTagFirstOptional(tags, "-", "originalyear", "year", "originaldate", "date")

	albumMBID := readSingleTagOptional(tags, "musicbrainz_releasegroupid")
	releaseMBID := readSingleTagOptional(tags, "musicbrainz_albumid")

	if !s.songQueueClosed {
		s.songQueue <- &mediaFile{
			id:                  songID,
			path:                path,
			size:                info.Size(),
			contentType:         contentType,
			lastModified:        info.ModTime(),
			cover:               cover,
			bitrate:             props.Bitrate,
			channels:            props.Channels,
			lengthMS:            props.LengthMs,
			sampleRate:          props.Samplerate,
			title:               title,
			albumName:           album,
			albumMBID:           albumMBID,
			albumReleaseMBID:    releaseMBID,
			artistNames:         artists,
			artistMBIDs:         artistMBIDs,
			albumArtistNames:    albumArtists,
			albumArtistMBIDs:    albumArtistMBIDs,
			albumReplayGain:     readReplayGainTag(tags, "replaygain_album_gain"),
			albumReplayGainPeak: readReplayGainTag(tags, "replaygain_album_peak"),
			recordLabels:        readStringTags(tags, "labels", "label"),
			releaseTypes:        readStringTags(tags, "releasetypes", "releasetype"),
			isCompilation:       isCompilation,
			bpm:                 readSingleIntTagOptional(tags, "bpm"),
			year:                year,
			track:               readSingleIntTagFirstOptional(tags, "/", "tracknumber"),
			disc:                readSingleIntTagFirstOptional(tags, "/", "discnumber"),
			discTitle:           readSingleTagOptional(tags, "discsubtitle"),
			genres:              readStringTags(tags, "genres", "genre"),
			musicBrainzID:       readSingleTagOptional(tags, "musicbrainz_trackid"),
			replayGain:          readReplayGainTag(tags, "replaygain_track_gain"),
			replayGainPeak:      readReplayGainTag(tags, "replaygain_track_peak"),
			lyrics:              lyrics,
		}
	}

	return nil
}

func (s *Scanner) checkIfChanged(path string, info fs.FileInfo) bool {
	if info.ModTime().After(s.lastScan) {
		return true
	}
	return s.checkIfChangedByPath(path)
}

func (s *Scanner) checkIfChangedByPath(path string) bool {
	stat, err := times.Stat(path)
	if err != nil {
		log.Errorf("check if file changed: %s", err)
		return true
	}
	return stat.ModTime().After(s.lastScan) || !stat.HasChangeTime() || stat.ChangeTime().After(s.lastScan)
}

func readSingleTag(tags map[string][]string, key string) (string, bool) {
	v, ok := tags[key]
	if !ok {
		return "", false
	}
	if len(v) == 0 {
		return "", false
	}
	return v[0], true
}

func readSingleTagOptional(tags map[string][]string, key string) *string {
	v, ok := tags[key]
	if !ok {
		return nil
	}
	if len(v) == 0 {
		return nil
	}
	return &v[0]
}

func readSingleIntTagOptional(tags map[string][]string, key string) *int {
	v, ok := tags[key]
	if !ok {
		return nil
	}
	if len(v) == 0 {
		return nil
	}
	i, err := strconv.Atoi(v[0])
	if err != nil {
		return nil
	}
	return &i
}

func readSingleIntTagFirstOptional(tags map[string][]string, sep string, keys ...string) *int {
	for _, k := range keys {
		v, ok := tags[k]
		if !ok || len(v) == 0 {
			continue
		}
		parts := strings.Split(v[0], sep)
		i, err := strconv.Atoi(parts[0])
		if err != nil {
			return nil
		}
		return &i
	}
	return nil
}

func readReplayGainTag(tags map[string][]string, key string) *float64 {
	v, ok := tags[key]
	if !ok {
		return nil
	}
	if len(v) == 0 {
		return nil
	}
	str := strings.ToLower(v[0])
	str = strings.ReplaceAll(str, "db", "")
	str = strings.ReplaceAll(str, "+", "")
	str = strings.TrimSpace(str)
	gain, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return nil
	}
	return &gain
}

func readSingleBoolTag(tags map[string][]string, key string) bool {
	v, ok := tags[key]
	if !ok {
		return false
	}
	if len(v) == 0 {
		return false
	}
	b, _ := strconv.ParseBool(v[0])
	return b
}

func readStringTags(tags map[string][]string, keys ...string) []string {
	for _, k := range keys {
		v, ok := tags[k]
		if !ok || len(v) == 0 {
			continue
		}
		return v
	}
	return []string{}
}
