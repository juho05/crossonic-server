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

	"github.com/juho05/crossonic-server/audiotags"
	"github.com/juho05/crossonic-server/repos"
	"github.com/juho05/log"
)

const maxParallelDirs = 10
const updateSongFilesWorkerCount = 10
const setAlbumCoversWorkerCount = 10
const songQueueBatchSize = 100

var coverImageNames = []string{"front", "folder", "cover"}

func (s *Scanner) Scan(db repos.DB) error {
	if !s.lock.TryLock() {
		return ErrAlreadyScanning
	}
	s.scanning = true
	defer func() {
		if s.songQueue != nil {
			close(s.songQueue)
		}
		if s.setAlbumCover != nil {
			close(s.setAlbumCover)
		}
		s.scanning = false
		s.albums = nil
		s.artists = nil
		s.songQueue = nil
		s.setAlbumCover = nil
	}()
	defer s.lock.Unlock()

	s.scanStart = time.Now()

	s.counter.Store(0)

	log.Infof("Scanning %s...", s.mediaDir)

	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()

	var err error
	s.tx, err = db.NewTransaction(ctx)
	if err != nil {
		return fmt.Errorf("new transaction: %w", err)
	}
	defer func() {
		err := s.tx.Rollback()
		if err != nil && !errors.Is(err, sql.ErrTxDone) {
			log.Error("rollback full scan tx: %w", err)
		}
		s.tx = nil
	}()

	s.lastScan, err = db.System().LastScan(ctx)
	if err != nil && !errors.Is(err, repos.ErrNotFound) {
		return fmt.Errorf("get last scan: %w", err)
	}

	songCount, err := s.tx.Song().Count(ctx)
	if err != nil {
		return fmt.Errorf("get song count: %w", err)
	}
	s.firstScan = songCount == 0
	if s.firstScan {
		log.Tracef("detected first scan")
	}

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
	s.setAlbumCover = make(chan albumCover, setAlbumCoversWorkerCount)

	saveSongsDone := make(chan error, 1)
	log.Tracef("starting save songs loop with batch size %d...", songQueueBatchSize)
	go func() {
		err := s.runSaveSongsLoop(ctx)
		saveSongsDone <- err
		close(s.setAlbumCover)
		s.setAlbumCover = nil
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
	s.songQueue = nil
	if err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("scan dir: %w", err)
	}

	log.Tracef("updating album artists...")
	err = s.albums.updateArtists(ctx, s)
	if err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("update album artists: %w", err)
	}

	err = <-saveSongsDone
	if err != nil {
		return fmt.Errorf("run save songs loop: %w", err)
	}

	err = <-setAlbumCovers
	if err != nil {
		return fmt.Errorf("run set album covers loop: %w", err)
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

	log.Tracef("committing changes...")
	err = s.tx.Commit()
	if err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	log.Infof("Scanned %d files in %s.", s.counter.Load(), time.Since(s.scanStart).Round(time.Millisecond))
	return nil
}

func (s *Scanner) scanMediaDir(ctx context.Context) error {
	dirs := make(chan string, maxParallelDirs)

	var waitGroup sync.WaitGroup
	var scanDirError error
	for range maxParallelDirs {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			for dir := range dirs {
				err := s.scanMediaFilesInDir(ctx, dir)
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

	err := filepath.WalkDir(s.mediaDir, func(path string, d fs.DirEntry, err error) error {
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
		dirs <- path
		return nil
	})
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

func (s *Scanner) scanMediaFilesInDir(ctx context.Context, dir string) error {
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

		ext := filepath.Ext(e.Name())
		fileType := mime.TypeByExtension(ext)
		if fileType == "image/jpeg" || fileType == "image/png" {
			base := strings.TrimSuffix(e.Name(), ext)
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

		err := s.processFile(filepath.Join(dir, e.Name()), cover)
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

func (s *Scanner) processFile(path string, cover *string) error {
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
		genres:              readStringTags(tags, "genres", "genre"),
		musicBrainzID:       readSingleTagOptional(tags, "musicbrainz_trackid"),
		replayGain:          readReplayGainTag(tags, "replaygain_track_gain"),
		replayGainPeak:      readReplayGainTag(tags, "replaygain_track_peak"),
		lyrics:              lyrics,
	}

	return nil
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
