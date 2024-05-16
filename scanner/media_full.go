package scanner

import (
	"context"
	"errors"
	"fmt"
	"image/jpeg"
	"io"
	"io/fs"
	"mime"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/juho05/crossonic-server"
	"github.com/juho05/crossonic-server/audiotags"
	"github.com/juho05/crossonic-server/config"
	db "github.com/juho05/crossonic-server/db/sqlc"
	"github.com/juho05/log"
)

type mediaFile struct {
	id          *string
	path        string
	size        int
	updated     time.Time
	contentType string
	coverPath   *string

	bitrate    int
	channels   int
	lengthMS   int
	sampleRate int

	title                     string
	album                     *string
	artists                   []string
	albumArtists              []string
	bpm                       *int
	compilation               bool
	year                      *int
	track                     *int
	disc                      *int
	genres                    []string
	labels                    []string
	musicBrainzSongID         *string
	musicBrainzAlbumID        *string
	musicBrainzArtistIDs      []string
	musicBrainzAlbumArtistIDs []string
	replayGainTrack           *float32
	replayGainTrackPeak       *float32
	replayGainAlbum           *float32
	replayGainAlbumPeak       *float32
	releaseTypes              []string
	lyrics                    *string
}

type song struct {
	id        string
	albumID   *string
	albumName *string
}

func (s *Scanner) ScanMediaFull(printStatus bool) error {
	if !s.lock.TryLock() {
		return ErrAlreadyScanning
	}
	s.Scanning = true
	defer func() {
		s.Scanning = false
	}()
	defer s.lock.Unlock()

	s.Count = 0

	log.Infof("Scanning %s...", s.mediaDir)

	ctx := context.Background()

	songCount, err := s.store.FindSongCount(ctx)
	if err != nil {
		return fmt.Errorf("get song count: %w", err)
	}
	s.firstScan = songCount == 0

	c := make(chan mediaFile)
	s.waitGroup.Add(1)
	go s.scanMediaDir(ctx, s.mediaDir, c)

	processDone := make(chan bool)
	go s.processMediaFiles(ctx, c, processDone, printStatus)
	s.waitGroup.Wait()
	close(c)
	success := <-processDone
	err = os.RemoveAll(filepath.Join(config.DataDir(), "covers", "scan"))
	if err != nil {
		log.Errorf("delete scan cover cache: %s", err)
	}
	if !success {
		log.Error("Scan failed.")
		return errors.New("scan error")
	}
	log.Info("Scan complete.")
	return nil
}

var imagePrios = []string{"front", "folder", "cover"}

func (s *Scanner) processMediaFiles(ctx context.Context, c <-chan mediaFile, done chan<- bool, printStatus bool) {
	startTime := time.Now()
	s.originalStore = s.store
	var err error
	s.store, err = s.store.BeginTransaction(ctx)
	if err != nil {
		s.store = s.originalStore
		log.Errorf("process media files: %s", err)
		return
	}
	defer close(done)
	defer func() {
		s.store.Rollback(ctx)
		s.store = s.originalStore
	}()

	updatedAlbums := make(map[string]struct{})
	updatedArtists := make(map[string]bool)

	albumCovers := make(map[string]struct{})
	songCovers := make(map[string]struct{})
	albumCoverDir := filepath.Join(s.coverDir, "albums")
	songCoverDir := filepath.Join(s.coverDir, "songs")

	err = os.MkdirAll(albumCoverDir, 0755)
	if err != nil {
		log.Errorf("process media files: create album cover dir: %s", err)
		return
	}

	err = os.MkdirAll(songCoverDir, 0755)
	if err != nil {
		log.Errorf("process media files: create album cover dir: %s", err)
		return
	}

	err = s.store.DeleteAllGenres(ctx)
	if err != nil {
		log.Errorf("process media files: delete all genres: %s", err)
		return
	}

	for media := range c {
		if printStatus && s.Count%5 == 0 {
			fmt.Print("\rScanned: ", s.Count)
		}
		song, err := s.findOrCreateSong(ctx, media)
		if err != nil {
			log.Errorf("failed to find/create song in db for %s: %s", media.path, err)
			return
		}
		var albumID *string
		if media.album != nil {
			artistIDs, err := s.store.FindOrCreateArtistIDs(ctx, media.albumArtists)
			if err != nil && len(media.albumArtists) > 0 {
				log.Errorf("failed to find/create album artists for album %s: %s", *media.album, err)
				return
			}
			for i, art := range artistIDs {
				if _, ok := updatedArtists[art]; !ok {
					var musicBrainzID *string
					if len(media.musicBrainzAlbumArtistIDs) > i {
						musicBrainzID = &media.musicBrainzAlbumArtistIDs[i]
					}
					err = s.store.UpdateArtist(ctx, db.UpdateArtistParams{
						ID:            art,
						Name:          media.albumArtists[i],
						MusicBrainzID: musicBrainzID,
					})
					if err != nil {
						log.Errorf("failed to update artist %s: %s", art, err)
					} else {
						updatedArtists[art] = len(media.musicBrainzAlbumArtistIDs) > i
					}
				}
			}
			var recordLabels *string
			if len(media.labels) > 0 {
				labels := strings.Join(media.labels, "\003")
				recordLabels = &labels
			}
			var releaseTypes *string
			if len(media.releaseTypes) > 0 {
				types := strings.Join(media.releaseTypes, "\003")
				releaseTypes = &types
			}
			album, err := s.findAlbumID(ctx, media.album, media.albumArtists, media.musicBrainzAlbumID)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					a, err := s.store.CreateAlbum(ctx, db.CreateAlbumParams{
						ID:             "al_" + crossonic.GenID(),
						Name:           *media.album,
						Year:           intPtrToInt32Ptr(media.year),
						RecordLabels:   recordLabels,
						MusicBrainzID:  media.musicBrainzAlbumID,
						ReleaseTypes:   releaseTypes,
						IsCompilation:  &media.compilation,
						ReplayGain:     media.replayGainAlbum,
						ReplayGainPeak: media.replayGainAlbumPeak,
					})
					if err != nil {
						log.Errorf("failed to find/create album of %s in db: %s", media.path, err)
						return
					}
					album = a.ID
				} else {
					log.Errorf("process file %s: %s", media.path, err)
					return
				}
			} else if _, ok := updatedAlbums[album]; !ok {
				err := s.store.UpdateAlbum(ctx, db.UpdateAlbumParams{
					ID:             album,
					Name:           *media.album,
					Year:           intPtrToInt32Ptr(media.year),
					RecordLabels:   recordLabels,
					ReleaseTypes:   releaseTypes,
					IsCompilation:  &media.compilation,
					ReplayGain:     media.replayGainAlbum,
					ReplayGainPeak: media.replayGainAlbumPeak,
				})
				if err != nil {
					log.Errorf("failed to update album of %s: %s", media.path, err)
					return
				}
				updatedAlbums[album] = struct{}{}
			}
			err = s.store.UpdateAlbumArtists(ctx, album, artistIDs)
			if err != nil {
				log.Errorf("failed to update artists of album %s: %s", album, err)
				return
			}
			err = s.store.UpdateAlbumGenres(ctx, album, media.genres)
			if err != nil {
				log.Errorf("failed to update genres of album %s: %s", album, err)
				return
			}
			albumID = &album
		}

		if media.coverPath != nil {
			var file *os.File
			if albumID != nil {
				if _, ok := albumCovers[*albumID]; !ok {
					file, err = os.Create(filepath.Join(albumCoverDir, *albumID))
					if err != nil {
						log.Errorf("failed to create cover art file for %s: %s", media.path, err)
					}
				}
			} else {
				if _, ok := songCovers[song.id]; !ok {
					file, err = os.Create(filepath.Join(songCoverDir, song.id))
					if err != nil {
						log.Errorf("failed to create cover art file for %s: %s", media.path, err)
					}
				}
			}
			if file != nil {
				originalFile, err := os.Open(*media.coverPath)
				if err != nil {
					log.Errorf("failed to open cover art file %s: %s", *media.coverPath, err)
				} else {
					_, err = io.Copy(file, originalFile)
					if err != nil {
						log.Errorf("failed to copy cover art file %s: %s", *media.coverPath, err)
					} else {
						if albumID != nil {
							albumCovers[*albumID] = struct{}{}
						} else {
							songCovers[song.id] = struct{}{}
						}
					}
					originalFile.Close()
				}
				file.Close()
			}
		}

		artistIDs, err := s.store.FindOrCreateArtistIDs(ctx, media.artists)
		if err != nil {
			log.Errorf("failed to find/create album artists for %s: %s", media.path, err)
			return
		}
		for i, art := range artistIDs {
			if hasMBrainz, ok := updatedArtists[art]; !ok || (!hasMBrainz && len(media.musicBrainzArtistIDs) > i) {
				var musicBrainzID *string
				if len(media.musicBrainzArtistIDs) > i {
					musicBrainzID = &media.musicBrainzArtistIDs[i]
				}
				err = s.store.UpdateArtist(ctx, db.UpdateArtistParams{
					ID:            art,
					Name:          media.artists[i],
					MusicBrainzID: musicBrainzID,
				})
				if err != nil {
					log.Errorf("failed to update artist %s (%s): %s", media.artists[i], art, err)
				} else {
					updatedArtists[art] = len(media.musicBrainzArtistIDs) > i
				}
			}
		}

		err = s.store.UpdateSong(ctx, db.UpdateSongParams{
			ID:             song.id,
			Path:           media.path,
			AlbumID:        albumID,
			Title:          media.title,
			Track:          intPtrToInt32Ptr(media.track),
			Year:           intPtrToInt32Ptr(media.year),
			Size:           int64(media.size),
			ContentType:    media.contentType,
			DurationMs:     int32(media.lengthMS),
			BitRate:        int32(media.bitrate),
			SamplingRate:   int32(media.sampleRate),
			ChannelCount:   int32(media.channels),
			DiscNumber:     intPtrToInt32Ptr(media.disc),
			Bpm:            intPtrToInt32Ptr(media.bpm),
			MusicBrainzID:  media.musicBrainzSongID,
			ReplayGain:     media.replayGainTrack,
			ReplayGainPeak: media.replayGainTrackPeak,
			Lyrics:         media.lyrics,
		})
		if err != nil {
			log.Errorf("failed to update song in db: %s", err)
			return
		}

		err = s.store.UpdateSongArtists(ctx, song.id, artistIDs)
		if err != nil {
			log.Errorf("failed to update song artists: %s", err)
			return
		}

		err = s.store.UpdateSongGenres(ctx, song.id, media.genres)
		if err != nil {
			log.Errorf("failed to update song genres: %s", err)
			return
		}
		s.Count++
	}
	if printStatus {
		fmt.Println("\rScanned:", s.Count)
	}

	err = s.clean(ctx, startTime)
	if err != nil {
		log.Errorf("process media files: %s", err)
		return
	}

	err = s.store.Commit(ctx)
	if err != nil {
		log.Errorf("process media files: %s", err)
		return
	}

	err = s.cleanCovers(albumCovers, songCovers)
	if err != nil {
		log.Errorf("process media files: %s", err)
		return
	}

	done <- true
}

func (s *Scanner) clean(ctx context.Context, startTime time.Time) error {
	pgTime := pgtype.Timestamptz{
		Time:  startTime,
		Valid: true,
	}
	err := s.store.DeleteSongsLastUpdatedBefore(ctx, pgTime)
	if err != nil {
		return fmt.Errorf("clean: %w", err)
	}
	err = s.store.DeleteAlbumsLastUpdatedBefore(ctx, pgTime)
	if err != nil {
		return fmt.Errorf("clean: %w", err)
	}
	err = s.store.DeleteArtistsLastUpdatedBefore(ctx, pgTime)
	if err != nil {
		return fmt.Errorf("clean: %w", err)
	}
	return nil
}

func (s *Scanner) cleanCovers(albums, songs map[string]struct{}) error {
	albumCoverDir := filepath.Join(s.coverDir, "albums")
	entries, err := os.ReadDir(albumCoverDir)
	if err != nil {
		return fmt.Errorf("clean covers: %w", err)
	}
	for _, e := range entries {
		if _, ok := albums[e.Name()]; !ok {
			err = os.Remove(filepath.Join(albumCoverDir, e.Name()))
			if err != nil {
				return fmt.Errorf("clean covers: %w", err)
			}
		}
	}

	songCoverDir := filepath.Join(s.coverDir, "songs")
	entries, err = os.ReadDir(songCoverDir)
	if err != nil {
		return fmt.Errorf("clean covers: %w", err)
	}
	for _, e := range entries {
		if _, ok := songs[e.Name()]; !ok {
			err = os.Remove(filepath.Join(songCoverDir, e.Name()))
			if err != nil {
				return fmt.Errorf("clean covers: %w", err)
			}
		}
	}
	return nil
}

func (s *Scanner) setCrossonicID(path, id string) error {
	file, err := audiotags.Open(path)
	if err != nil {
		return fmt.Errorf("set crossonic id: %w", err)
	}
	defer file.Close()
	if !file.HasMedia() {
		return fmt.Errorf("set crossonic id: unsupported format")
	}
	tags := file.ReadTags()
	tags["crossonic_id_"+s.store.InstanceID()] = []string{id}
	if !file.WriteTags(tags) {
		return fmt.Errorf("set crossonic id: write failed")
	}
	return nil
}

func (s *Scanner) findOrCreateSong(ctx context.Context, media mediaFile) (sng *song, err error) {
	if media.id != nil {
		s, err := s.store.FindSong(ctx, *media.id)
		if err == nil {
			return &song{
				id:        s.ID,
				albumName: s.AlbumName,
				albumID:   s.AlbumID,
			}, nil
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("find or create song by id: %w", err)
		}
	}
	defer func() {
		if err == nil && sng != nil {
			go func() {
				err2 := s.setCrossonicID(media.path, sng.id)
				if err2 != nil {
					log.Errorf("failed to write crossonic_id metadata into %s: %s", media.path, err)
				}
			}()
		}
	}()
	if media.musicBrainzSongID != nil {
		s, err := s.store.FindSongByMusicBrainzID(ctx, media.musicBrainzSongID)
		if err == nil {
			return &song{
				id:        s.ID,
				albumName: s.AlbumName,
				albumID:   s.AlbumID,
			}, nil
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("find or create song by musicbrainz ID: %w", err)
		}
	}

	sn, err := s.store.FindSongByPath(ctx, media.path)
	if err == nil {
		return &song{
			id:        sn.ID,
			albumName: sn.AlbumName,
			albumID:   sn.AlbumID,
		}, nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("find or create song by path: %w", err)
	}

	sc, err := s.store.CreateSong(ctx, db.CreateSongParams{
		ID:             "tr_" + crossonic.GenID(),
		Path:           media.path,
		AlbumID:        nil,
		Title:          media.title,
		Track:          intPtrToInt32Ptr(media.track),
		Year:           intPtrToInt32Ptr(media.year),
		Size:           int64(media.size),
		ContentType:    media.contentType,
		DurationMs:     int32(media.lengthMS),
		BitRate:        int32(media.bitrate),
		SamplingRate:   int32(media.sampleRate),
		ChannelCount:   int32(media.channels),
		DiscNumber:     intPtrToInt32Ptr(media.disc),
		Bpm:            intPtrToInt32Ptr(media.bpm),
		MusicBrainzID:  media.musicBrainzSongID,
		ReplayGain:     media.replayGainTrack,
		ReplayGainPeak: media.replayGainTrackPeak,
	})
	if err != nil {
		return nil, fmt.Errorf("find or create song: %w", err)
	}
	return &song{
		id:        sc.ID,
		albumName: media.album,
		albumID:   sc.AlbumID,
	}, nil
}

func (s *Scanner) findAlbumID(ctx context.Context, albumName *string, albumArtists []string, musicBrainzAlbumID *string) (string, error) {
	if albumName == nil {
		return "", fmt.Errorf("find album id: %w", pgx.ErrNoRows)
	}
	albums, err := s.store.FindAlbumsByNameWithArtistMatchCount(ctx, db.FindAlbumsByNameWithArtistMatchCountParams{
		Name:        *albumName,
		ArtistNames: albumArtists,
	})
	if err != nil {
		return "", fmt.Errorf("find album id: %w", err)
	}
	if len(albums) == 0 {
		return "", fmt.Errorf("find album id: %w", pgx.ErrNoRows)
	}
	if musicBrainzAlbumID != nil {
		for _, a := range albums {
			if a.MusicBrainzID == nil {
				continue
			}
			if *a.MusicBrainzID == *musicBrainzAlbumID {
				return a.ID, nil
			}
		}
	}
	if len(albumArtists) == 0 {
		for _, a := range albums {
			if a.ArtistMatches == 0 {
				return a.ID, nil
			}
		}
		return albums[0].ID, nil
	}
	var bestMatch string
	var matches int
	for _, a := range albums {
		if int(a.ArtistMatches) > matches {
			matches = int(a.ArtistMatches)
			bestMatch = a.ID
		}
	}
	if matches > 0 {
		return bestMatch, nil
	}
	return "", fmt.Errorf("find album id: %w", pgx.ErrNoRows)
}

func (s *Scanner) scanMediaDir(ctx context.Context, path string, c chan<- mediaFile) {
	defer s.waitGroup.Done()
	entries, err := os.ReadDir(path)
	if err != nil {
		log.Errorf("scan media dir: %s: %s", path, err)
	}
	imagePrio := -1
	var image string
	for _, e := range entries {
		if !e.Type().IsRegular() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		fileType := mime.TypeByExtension(ext)
		if fileType != "image/jpeg" && fileType != "image/png" {
			continue
		}
		base := strings.TrimSuffix(e.Name(), ext)
		for i := imagePrio + 1; i < len(imagePrios); i++ {
			if base == imagePrios[i] {
				image = filepath.Join(path, e.Name())
				break
			}
		}
	}
	for _, e := range entries {
		if e.Type() == fs.ModeSymlink {
			path, err := filepath.EvalSymlinks(filepath.Join(path, e.Name()))
			if err != nil {
				continue
			}
			info, err := os.Stat(path)
			if err != nil {
				continue
			}
			if info.IsDir() {
				s.waitGroup.Add(1)
				go s.scanMediaDir(ctx, path, c)
			} else {
				image = s.scanFile(path, image, c)
			}
			continue
		}
		if e.IsDir() {
			s.waitGroup.Add(1)
			go s.scanMediaDir(ctx, filepath.Join(path, e.Name()), c)
			continue
		}
		image = s.scanFile(filepath.Join(path, e.Name()), image, c)
	}
}

func (s *Scanner) scanFile(path, img string, c chan<- mediaFile) (newImg string) {
	newImg = img

	ext := filepath.Ext(path)
	info, err := os.Stat(path)
	if err != nil {
		log.Errorf("scan media dir: scan file: %s", err)
		return
	}
	if info.IsDir() {
		return
	}
	if !strings.HasPrefix(mime.TypeByExtension(ext), "audio/") {
		return
	}

	file, err := audiotags.Open(path)
	if err != nil {
		return
	}
	defer file.Close()
	if !file.HasMedia() {
		return
	}
	props := file.ReadAudioProperties()
	tags := file.ReadTags()
	if img == "" {
		cover, _ := file.ReadImage()
		if cover != nil {
			scanCoverDir := filepath.Join(s.coverDir, "scan")
			os.MkdirAll(scanCoverDir, 0755)
			id := crossonic.GenID()
			coverPath := filepath.Join(scanCoverDir, id+".jpg")
			imgFile, err := os.Create(coverPath)
			if err != nil {
				log.Errorf("scan media dir: scan file: save embedded cover to temp file: %s", err)
			} else {
				err := jpeg.Encode(imgFile, cover, nil)
				imgFile.Close()
				if err != nil {
					log.Errorf("scan media dir: scan file: save embedded cover to temp file: %s", err)
				} else {
					newImg = coverPath
				}
			}
		}
	}

	var id *string
	idStr, ok := readSingleTag(tags, "crossonic_id_"+s.store.InstanceID())
	if ok && strings.HasPrefix(idStr, "tr_") {
		id = &idStr
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

	var coverPath *string
	if newImg != "" {
		coverPath = &newImg
	}

	c <- mediaFile{
		id:                        id,
		path:                      path,
		size:                      int(info.Size()),
		updated:                   info.ModTime(),
		contentType:               mime.TypeByExtension(ext),
		coverPath:                 coverPath,
		bitrate:                   props.Bitrate,
		channels:                  props.Channels,
		lengthMS:                  props.LengthMs,
		sampleRate:                props.Samplerate,
		title:                     title,
		album:                     readSingleTagOptional(tags, "album"),
		artists:                   readStringTags(tags, "artists", "artist"),
		albumArtists:              readStringTags(tags, "albumartists", "album_artists", "albumartist", "album_artist"),
		bpm:                       readSingleIntTagOptional(tags, "bpm"),
		compilation:               readSingleBoolTag(tags, "compilation"),
		year:                      readSingleIntTagFirstOptional(tags, "-", "originalyear", "year", "originaldate", "date"),
		track:                     readSingleIntTagFirstOptional(tags, "/", "tracknumber"),
		disc:                      readSingleIntTagFirstOptional(tags, "/", "discnumber"),
		genres:                    readStringTags(tags, "genres", "genre"),
		labels:                    readStringTags(tags, "labels", "label"),
		musicBrainzSongID:         readSingleTagOptional(tags, "musicbrainz_trackid"),
		musicBrainzAlbumID:        readSingleTagOptional(tags, "musicbrainz_releasegroupid"),
		musicBrainzArtistIDs:      readStringTags(tags, "musicbrainz_artistids", "musicbrainz_artistid"),
		musicBrainzAlbumArtistIDs: readStringTags(tags, "musicbrainz_albumartistids", "musicbrainz_albumartistid"),
		replayGainTrack:           readReplayGainTag(tags, "replaygain_track_gain"),
		replayGainTrackPeak:       readReplayGainTag(tags, "replaygain_track_peak"),
		replayGainAlbum:           readReplayGainTag(tags, "replaygain_album_gain"),
		replayGainAlbumPeak:       readReplayGainTag(tags, "replaygain_album_peak"),
		releaseTypes:              readStringTags(tags, "releasetypes", "releasetype"),
		lyrics:                    lyrics,
	}

	return newImg
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

func readReplayGainTag(tags map[string][]string, key string) *float32 {
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
	f64, err := strconv.ParseFloat(str, 32)
	if err != nil {
		return nil
	}
	f32 := float32(f64)
	return &f32
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

func intPtrToInt32Ptr(ptr *int) *int32 {
	if ptr == nil {
		return nil
	}
	v := *ptr
	v32 := int32(v)
	return &v32
}
