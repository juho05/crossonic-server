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
	"github.com/juho05/crossonic-server/config"
	"github.com/juho05/crossonic-server/repos"
	"github.com/juho05/crossonic-server/util"
	"github.com/juho05/log"
)

const maxParallelDirs = 10
const updateSongFilesWorkerCount = 10
const setAlbumCoversWorkerCount = 10
const saveArtistCoversWorkerCount = 10
const songQueueBatchSize = 100
const deleteOrphanedSongsByPathWorkerCount = 10
const deleteOrphanedSongsByPathBatchSize = 300

func (s *Scanner) Scan(db repos.DB, fullScan bool) (err error) {
	if !s.lock.TryLock() {
		return ErrAlreadyScanning
	}
	s.scanning = true
	s.fullScan = fullScan
	defer s.lock.Unlock()
	defer func() {
		if !s.songQueueClosed {
			s.songQueueClosed = true
			close(s.songQueue)
		}
		if !s.setAlbumCoverClosed {
			s.setAlbumCoverClosed = true
			close(s.setAlbumCover)
		}
		s.scanning = false
		s.albums = nil
		s.artists = nil
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
	}
	err = s.tx.System().ResetNeedsFullScan(ctx)
	if err != nil {
		return fmt.Errorf("reset needs full scan: %w", err)
	}

	if s.fullScan || s.firstScan {
		s.lastScan = time.Time{}
	}

	log.Infof("Scanning %s (full scan: %t)...", s.mediaDir, s.fullScan)

	if s.fullScan {
		log.Tracef("clearing cover cache...")
		err = s.coverCache.Clear()
		if err != nil {
			return fmt.Errorf("clear cover cache: %w", err)
		}
	}

	log.Tracef("finding artist images...")
	var waitFindArtistImages sync.WaitGroup
	var findArtistImagesErr error
	var foundArtistImages map[string]string
	waitFindArtistImages.Add(1)
	go func() {
		defer waitFindArtistImages.Done()
		images, err := s.findArtistImages(ctx)
		if err != nil {
			findArtistImagesErr = err
			return
		}
		foundArtistImages = images
	}()

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
		if !s.setAlbumCoverClosed {
			s.setAlbumCoverClosed = true
			close(s.setAlbumCover)
		}
		if err != nil {
			log.Errorf("scan: run save songs loop: %s", err)
			cancelCtx()
		}
	}()

	setAlbumCovers := make(chan error, 1)
	log.Tracef("starting set album covers loop with %d workers...", setAlbumCoversWorkerCount)
	go func() {
		err := s.runSetAlbumCoverLoop(ctx)
		if !s.setAlbumCoverClosed {
			s.setAlbumCoverClosed = true
			close(s.setAlbumCover)
		}
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

	log.Tracef("fixing scrobble metadata...")
	err = s.tx.Scrobble().FixMetadata(ctx)
	if err != nil {
		return fmt.Errorf("fix scrobble metadata: %w", err)
	}

	err = s.tx.System().SetLastScan(ctx, time.Now())
	if err != nil {
		return fmt.Errorf("update last scan: %w", err)
	}

	waitFindArtistImages.Wait()
	if findArtistImagesErr != nil {
		return fmt.Errorf("find artist images: %w", findArtistImagesErr)
	}
	log.Tracef("saving artist images with %d workers...", saveArtistCoversWorkerCount)
	err = s.saveArtistImages(ctx, foundArtistImages)
	if err != nil {
		return fmt.Errorf("save artist images: %w", err)
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
	prioritizeEmbedded := false
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
			coverImagePatterns := s.conf.CoverArtPriority
			for i := 0; i < len(coverImagePatterns); i++ {
				if coverImagePatterns[i] == config.CoverArtPriorityEmbedded {
					prioritizeEmbedded = true
					continue
				}
				match, err := filepath.Match(coverImagePatterns[i], e.Name())
				if err != nil {
					log.Errorf("invalid cover art priority pattern %s: %v", coverImagePatterns[i], err)
					continue
				}
				if !match {
					continue
				}
				c := filepath.Join(dir, e.Name())
				cover = &c
				break findCoverLoop
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

		err := s.processFile(filepath.Join(dir, e.Name()), cover, prioritizeEmbedded, changed)
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

func (s *Scanner) processFile(path string, cover *string, prioritizeEmbeddedCover, parentDirChanged bool) error {
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

	lyricsPath, lyricsModified := s.findLyricsSidecar(path)

	if !s.fullScan && !parentDirChanged && info.ModTime().Before(s.lastScan) && !lyricsModified {
		timeStat, err := times.Stat(path)
		if err != nil {
			return fmt.Errorf("times stat: %w", err)
		}
		if timeStat.HasChangeTime() && timeStat.ChangeTime().Before(s.lastScan) {
			return nil
		}
	}

	tags, props, hasImage, err := audiotags.Read(path, prioritizeEmbeddedCover || cover == nil)
	if err != nil {
		if errors.Is(err, audiotags.ErrNoMetadata) {
			return errNotAMediaFile
		}
		return fmt.Errorf("read tags: %w", err)
	}

	if props.IsEmpty() {
		return errNotAMediaFile
	}

	if prioritizeEmbeddedCover && cover != nil && hasImage {
		cover = nil
	}

	var songID *string
	idTag, ok := readSingleTag(tags, "CROSSONIC_ID_"+strings.ToUpper(s.instanceID))
	if ok && strings.HasPrefix(idTag, "tr_") {
		songID = &idTag
	}

	title, ok := readSingleTag(tags, "TITLE")
	if !ok {
		title = strings.TrimSuffix(filepath.Base(path), ext)
	}

	lyrics := s.scanLyrics(lyricsPath, tags)

	isCompilation := readSingleBoolTag(tags, "COMPILATION")

	artists := readStringTags(tags, "ARTISTS", "ARTIST")
	albumArtists := readStringTags(tags, "ALBUMARTISTS", "ALBUM_ARTISTS", "ALBUMARTIST", "ALBUM_ARTIST")
	if !isCompilation && len(albumArtists) == 0 && len(artists) > 0 {
		albumArtists = []string{artists[0]}
	}

	artistMBIDs := readStringTags(tags, "MUSICBRAINZ_ARTISTIDS", "MUSICBRAINZ_ARTISTID")
	albumArtistMBIDs := readStringTags(tags, "MUSICBRAINZ_ALBUMARTISTIDS", "MUSICBRAINZ_ALBUMARTISTID")

	album := readSingleTagOptional(tags, "ALBUM")

	originalDate := readDateTagFirstOptional(tags, "ORIGINALDATE", "ORIGINALYEAR", "DATE", "YEAR")
	releaseDate := readDateTagFirstOptional(tags, "RELEASEDATE", "RELEASEYEAR", "DATE", "YEAR", "ORIGINALDATE", "ORIGINALYEAR")
	if originalDate == nil {
		originalDate = releaseDate
	}
	if originalDate != nil && releaseDate != nil {
		if originalDate.Year() == releaseDate.Year() && originalDate.Month() != nil && releaseDate.Month() == nil {
			releaseDate = util.ToPtr(repos.NewDate(releaseDate.Year(), originalDate.Month(), originalDate.Day()))
		} else if originalDate.Year() == releaseDate.Year() && originalDate.Month() == releaseDate.Month() && originalDate.Day() != nil && releaseDate.Day() == nil {
			releaseDate = util.ToPtr(repos.NewDate(releaseDate.Year(), originalDate.Month(), originalDate.Day()))
		}
	}

	albumMBID := readSingleTagOptional(tags, "MUSICBRAINZ_RELEASEGROUPID")
	releaseMBID := readSingleTagOptional(tags, "MUSICBRAINZ_ALBUMID")

	albumVersion := readSingleTagFirstOptional(tags, "ALBUMVERSION", "VERSION")

	if !s.songQueueClosed {
		s.songQueue <- &mediaFile{
			id:                  songID,
			path:                path,
			size:                info.Size(),
			contentType:         contentType,
			lastModified:        info.ModTime(),
			cover:               cover,
			bitrate:             props.BitRate,
			channels:            props.Channels,
			lengthMS:            props.LengthMs,
			sampleRate:          props.SampleRate,
			title:               title,
			albumName:           album,
			albumMBID:           albumMBID,
			albumReleaseMBID:    releaseMBID,
			artistNames:         artists,
			artistMBIDs:         artistMBIDs,
			albumArtistNames:    albumArtists,
			albumArtistMBIDs:    albumArtistMBIDs,
			albumReplayGain:     readReplayGainTag(tags, "REPLAYGAIN_ALBUM_GAIN"),
			albumReplayGainPeak: readReplayGainTag(tags, "REPLAYGAIN_ALBUM_PEAK"),
			recordLabels:        readStringTags(tags, "LABELS", "LABEL"),
			releaseTypes:        readStringTags(tags, "RELEASETYPES", "RELEASETYPE", "RELEASE_TYPE"),
			isCompilation:       isCompilation,
			bpm:                 readSingleIntTagOptional(tags, "BPM"),
			originalDate:        originalDate,
			releaseDate:         releaseDate,
			track:               readSingleIntTagFirstOptional(tags, "/", "TRACKNUMBER"),
			disc:                readSingleIntTagFirstOptional(tags, "/", "DISCNUMBER"),
			discTitle:           readSingleTagOptional(tags, "DISCSUBTITLE"),
			genres:              readStringTags(tags, "GENRES", "GENRE"),
			musicBrainzID:       readSingleTagOptional(tags, "MUSICBRAINZ_TRACKID"),
			replayGain:          readReplayGainTag(tags, "REPLAYGAIN_TRACK_GAIN"),
			replayGainPeak:      readReplayGainTag(tags, "REPLAYGAIN_TRACK_PEAK"),
			lyrics:              lyrics,
			albumVersion:        albumVersion,
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

func readSingleTagFirstOptional(tags map[string][]string, keys ...string) *string {
	for _, k := range keys {
		v, ok := tags[k]
		if !ok || len(v) == 0 {
			continue
		}
		return &v[0]
	}
	return nil
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

func readDateTagFirstOptional(tags map[string][]string, keys ...string) *repos.Date {
	for _, k := range keys {
		v, ok := tags[k]
		if !ok || len(v) == 0 {
			continue
		}
		date, err := repos.ParseDate(v[0])
		if err != nil {
			log.Warnf("scan: invalid date value: %s", v[0])
			continue
		}
		return &date
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
