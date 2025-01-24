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

	crossonic "github.com/juho05/crossonic-server"
	"github.com/juho05/crossonic-server/audiotags"
	"github.com/juho05/crossonic-server/config"
	"github.com/juho05/crossonic-server/repos"
	"github.com/juho05/log"
)

type mediaFile struct {
	id          *string
	path        string
	size        int64
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
	musicBrainzAlbumReleaseID *string
	musicBrainzArtistIDs      []string
	musicBrainzAlbumArtistIDs []string
	replayGainTrack           *float64
	replayGainTrackPeak       *float64
	replayGainAlbum           *float64
	replayGainAlbumPeak       *float64
	releaseTypes              []string
	lyrics                    *string
}

type song struct {
	id        string
	albumID   *string
	albumName *string
}

func (s *Scanner) ScanMediaFull() error {
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

	songCount, err := s.db.Song().Count(ctx)
	if err != nil {
		return fmt.Errorf("get song count: %w", err)
	}
	s.firstScan = songCount == 0

	c := make(chan mediaFile)
	s.waitGroup.Add(1)
	go s.scanMediaDir(ctx, s.mediaDir, c)

	processDone := make(chan bool)
	go s.processMediaFiles(ctx, c, processDone)
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
	log.Infof("Scanned %d files.", s.Count)
	return nil
}

var imagePrios = []string{"front", "folder", "cover"}

func (s *Scanner) processMediaFiles(ctx context.Context, c <-chan mediaFile, done chan<- bool) {
	startTime := time.Now()
	var err error
	s.tx, err = s.db.NewTransaction(ctx)
	if err != nil {
		log.Errorf("process media files: %s", err)
		return
	}
	defer close(done)
	defer func() {
		if s.tx == nil {
			return
		}
		err = s.tx.Rollback()
		if err != nil {
			log.Errorf("process media files: rollback: %s", err)
		}
	}()

	lastScan, err := s.tx.System().LastScan(ctx)
	if err != nil {
		log.Errorf("process media files: get last scan: %s", err)
	}

	cachedCoverKeys := make(map[string][]string)
	for _, k := range s.coverCache.Keys() {
		parts := strings.Split(k, "-")
		id := strings.Join(parts[:len(parts)-1], "-")
		cachedCoverKeys[id] = append(cachedCoverKeys[id], k)
	}

	updatedAlbums := make(map[string]struct{})
	updatedArtists := make(map[string]bool)

	albumCovers := make(map[string]struct{})
	songCovers := make(map[string]struct{})
	playlistCoverDir := filepath.Join(s.coverDir, "playlists")
	artistCoverDir := filepath.Join(s.coverDir, "artists")
	albumCoverDir := filepath.Join(s.coverDir, "albums")
	songCoverDir := filepath.Join(s.coverDir, "songs")

	err = os.MkdirAll(playlistCoverDir, 0755)
	if err != nil {
		log.Errorf("process media files: create playlist cover dir: %s", err)
		return
	}

	err = os.MkdirAll(artistCoverDir, 0755)
	if err != nil {
		log.Errorf("process media files: create artist cover dir: %s", err)
		return
	}

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

	err = s.tx.Genre().DeleteAll(ctx)
	if err != nil {
		log.Errorf("process media files: delete all genres: %s", err)
		return
	}

	for media := range c {
		err := s.tx.Genre().CreateIfNotExists(ctx, media.genres)
		if err != nil {
			log.Errorf("failed to create genres in db for %s: %s", media.path, err)
			return
		}
		song, err := s.findOrCreateSong(ctx, media)
		if err != nil {
			log.Errorf("failed to find/create song in db for %s: %s", media.path, err)
			return
		}
		var albumID *string
		if media.album != nil {
			artistIDs, err := s.tx.Artist().FindOrCreateIDsByNames(ctx, media.albumArtists)
			if err != nil && len(media.albumArtists) > 0 {
				log.Errorf("failed to find/create album artists for album %s: %s", *media.album, err)
				return
			}
			for i, aID := range artistIDs {
				if _, ok := updatedArtists[aID]; !ok {
					var musicBrainzID *string
					if len(media.musicBrainzAlbumArtistIDs) > i {
						musicBrainzID = &media.musicBrainzAlbumArtistIDs[i]
					}
					err = s.tx.Artist().Update(ctx, aID, repos.UpdateArtistParams{
						Name:          repos.NewOptionalFull(media.albumArtists[i]),
						MusicBrainzID: repos.NewOptionalFull(musicBrainzID),
					})
					if err != nil {
						log.Errorf("failed to update artist %s: %s", aID, err)
					} else {
						updatedArtists[aID] = len(media.musicBrainzAlbumArtistIDs) > i
					}
				}
			}
			album, err := s.findAlbumID(ctx, media.album, media.albumArtists, media.musicBrainzAlbumID)
			if err != nil {
				if errors.Is(err, repos.ErrNotFound) {
					a, err := s.tx.Album().Create(ctx, repos.CreateAlbumParams{
						Name:           *media.album,
						Year:           media.year,
						RecordLabels:   media.labels,
						MusicBrainzID:  media.musicBrainzAlbumID,
						ReleaseMBID:    media.musicBrainzAlbumReleaseID,
						ReleaseTypes:   media.releaseTypes,
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
				err := s.tx.Album().Update(ctx, album, repos.UpdateAlbumParams{
					Name:           repos.NewOptionalFull(*media.album),
					Year:           repos.NewOptionalFull(media.year),
					RecordLabels:   repos.NewOptionalFull(repos.StringList(media.labels)),
					ReleaseTypes:   repos.NewOptionalFull(repos.StringList(media.releaseTypes)),
					IsCompilation:  repos.NewOptionalFull(&media.compilation),
					ReplayGain:     repos.NewOptionalFull(media.replayGainAlbum),
					ReplayGainPeak: repos.NewOptionalFull(media.replayGainAlbumPeak),
					MusicBrainzID:  repos.NewOptionalFull(media.musicBrainzAlbumID),
					ReleaseMBID:    repos.NewOptionalFull(media.musicBrainzAlbumReleaseID),
				})
				if err != nil {
					log.Errorf("failed to update album of %s: %s", media.path, err)
					return
				}
				updatedAlbums[album] = struct{}{}
			}
			err = s.tx.Album().SetArtists(ctx, album, artistIDs)
			if err != nil {
				log.Errorf("failed to update artists of album %s: %s", album, err)
				return
			}
			err = s.tx.Album().SetGenres(ctx, album, media.genres)
			if err != nil {
				log.Errorf("failed to update genres of album %s: %s", album, err)
				return
			}
			albumID = &album
		}

		var coverID *string
		if media.coverPath != nil {
			originalFile, err := os.Open(*media.coverPath)
			if err != nil {
				log.Errorf("failed to open cover art file %s: %s", *media.coverPath, err)
			} else {
				func() {
					defer originalFile.Close()
					stat, err := originalFile.Stat()
					if err != nil {
						log.Errorf("failed to stat cover art file %s: %s", *media.coverPath, err)
					} else {
						var updateCover bool
						if stat.ModTime().After(lastScan) {
							updateCover = true
						} else {
							if albumID == nil {
								_, err = os.Stat(filepath.Join(config.DataDir(), "covers", "songs", song.id))
								updateCover = err != nil
							} else {
								_, err = os.Stat(filepath.Join(config.DataDir(), "covers", "albums", *albumID))
								updateCover = err != nil
							}
						}
						if updateCover {
							if keys, ok := cachedCoverKeys[song.id]; ok {
								for _, k := range keys {
									err := s.coverCache.DeleteObject(k)
									if err != nil {
										log.Errorf("scan: invalidate cover cache for %s: %s", *albumID)
									}
								}
							}
							var file *os.File
							if albumID != nil {
								if _, ok := albumCovers[*albumID]; !ok {
									if keys, ok := cachedCoverKeys[*albumID]; ok {
										for _, k := range keys {
											err := s.coverCache.DeleteObject(k)
											if err != nil {
												log.Errorf("scan: invalidate cover cache for %s: %s", *albumID)
											}
										}
									}
									file, err = os.Create(filepath.Join(albumCoverDir, *albumID))
									if err != nil {
										log.Errorf("failed to create cover art file for %s: %s", media.path, err)
									}
								}
								coverID = albumID
							} else {
								if _, ok := songCovers[song.id]; !ok {
									file, err = os.Create(filepath.Join(songCoverDir, song.id))
									if err != nil {
										log.Errorf("failed to create cover art file for %s: %s", media.path, err)
									}
								}
								coverID = &song.id
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
						} else {
							if albumID != nil {
								coverID = albumID
								albumCovers[*albumID] = struct{}{}
							} else {
								coverID = &song.id
								songCovers[song.id] = struct{}{}
							}
						}
					}
				}()
			}
		}

		artistIDs, err := s.tx.Artist().FindOrCreateIDsByNames(ctx, media.artists)
		if err != nil {
			log.Errorf("failed to find/create album artists for %s: %s", media.path, err)
			return
		}
		for i, aID := range artistIDs {
			if hasMBrainz, ok := updatedArtists[aID]; !ok || (!hasMBrainz && len(media.musicBrainzArtistIDs) > i) {
				var musicBrainzID *string
				if len(media.musicBrainzArtistIDs) > i {
					musicBrainzID = &media.musicBrainzArtistIDs[i]
				}
				err = s.tx.Artist().Update(ctx, aID, repos.UpdateArtistParams{
					Name:          repos.NewOptionalFull(media.artists[i]),
					MusicBrainzID: repos.NewOptionalFull(musicBrainzID),
				})
				if err != nil {
					log.Errorf("failed to update artist %s (%s): %s", media.artists[i], aID, err)
				} else {
					updatedArtists[aID] = len(media.musicBrainzArtistIDs) > i
				}
			}
		}

		err = s.tx.Song().Update(ctx, song.id, repos.UpdateSongParams{
			Path:           repos.NewOptionalFull(media.path),
			AlbumID:        repos.NewOptionalFull(albumID),
			Title:          repos.NewOptionalFull(media.title),
			Track:          repos.NewOptionalFull(media.track),
			Year:           repos.NewOptionalFull(media.year),
			Size:           repos.NewOptionalFull(media.size),
			ContentType:    repos.NewOptionalFull(media.contentType),
			Duration:       repos.NewOptionalFull(repos.NewDurationMS(int64(media.lengthMS))),
			BitRate:        repos.NewOptionalFull(media.bitrate),
			SamplingRate:   repos.NewOptionalFull(media.sampleRate),
			ChannelCount:   repos.NewOptionalFull(media.channels),
			Disc:           repos.NewOptionalFull(media.disc),
			BPM:            repos.NewOptionalFull(media.bpm),
			MusicBrainzID:  repos.NewOptionalFull(media.musicBrainzSongID),
			ReplayGain:     repos.NewOptionalFull(media.replayGainTrack),
			ReplayGainPeak: repos.NewOptionalFull(media.replayGainTrackPeak),
			Lyrics:         repos.NewOptionalFull(media.lyrics),
			CoverID:        repos.NewOptionalFull(coverID),
		})
		if err != nil {
			log.Errorf("failed to update song in db: %s", err)
			return
		}

		err = s.tx.Song().SetArtists(ctx, song.id, artistIDs)
		if err != nil {
			log.Errorf("failed to update song artists: %s", err)
			return
		}

		err = s.tx.Song().SetGenres(ctx, song.id, media.genres)
		if err != nil {
			log.Errorf("failed to update song genres: %s", err)
			return
		}
		s.Count++
	}

	err = s.clean(ctx, startTime)
	if err != nil {
		log.Errorf("process media files: %s", err)
		return
	}

	err = s.tx.System().SetLastScan(ctx, time.Now())
	if err != nil {
		log.Errorf("process media files: %s", err)
		return
	}

	err = s.tx.Commit()
	s.tx = nil
	if err != nil {
		log.Errorf("process media files: %s", err)
		return
	}

	err = s.cleanCovers(updatedArtists, albumCovers, songCovers)
	if err != nil {
		log.Errorf("process media files: %s", err)
		return
	}

	err = s.transcodeCache.InvalidateGracefully()
	if err != nil {
		log.Warnf("process media files: %w", err)
	}
	done <- true
}

func (s *Scanner) clean(ctx context.Context, startTime time.Time) error {
	err := s.tx.Song().DeleteLastUpdatedBefore(ctx, startTime)
	if err != nil {
		return fmt.Errorf("clean: %w", err)
	}
	err = s.tx.Album().DeleteLastUpdatedBefore(ctx, startTime)
	if err != nil {
		return fmt.Errorf("clean: %w", err)
	}
	err = s.tx.Artist().DeleteLastUpdatedBefore(ctx, startTime)
	if err != nil {
		return fmt.Errorf("clean: %w", err)
	}
	return nil
}

func (s *Scanner) cleanCovers(artists map[string]bool, albums, songs map[string]struct{}) error {
	artistCoverDir := filepath.Join(s.coverDir, "artists")
	entries, err := os.ReadDir(artistCoverDir)
	if err != nil {
		return fmt.Errorf("clean covers: %w", err)
	}
	for _, e := range entries {
		if _, ok := artists[e.Name()]; !ok {
			err = os.Remove(filepath.Join(artistCoverDir, e.Name()))
			if err != nil {
				return fmt.Errorf("clean covers: %w", err)
			}
		}
	}

	albumCoverDir := filepath.Join(s.coverDir, "albums")
	entries, err = os.ReadDir(albumCoverDir)
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
	tags["crossonic_id_"+s.instanceID] = []string{id}
	if !file.WriteTags(tags) {
		return fmt.Errorf("set crossonic id: write failed")
	}
	return nil
}

func (s *Scanner) findOrCreateSong(ctx context.Context, media mediaFile) (sng *song, err error) {
	if media.id != nil {
		s, err := s.tx.Song().FindByID(ctx, *media.id, repos.IncludeSongInfoAlbum())
		if err == nil {
			return &song{
				id:        s.ID,
				albumName: s.AlbumName,
				albumID:   s.AlbumID,
			}, nil
		} else if !errors.Is(err, repos.ErrNotFound) {
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
		ss, err := s.tx.Song().FindByMusicBrainzID(ctx, *media.musicBrainzSongID, repos.IncludeSongInfoAlbum())
		if err != nil {
			return nil, fmt.Errorf("find or create song by musicbrainz ID: %w", err)
		}
		var sn *song
		for _, s := range ss {
			if media.musicBrainzAlbumID != nil && s.AlbumMusicBrainzID == media.musicBrainzAlbumID {
				if sn == nil || (media.musicBrainzAlbumReleaseID != nil && s.AlbumReleaseMBID == media.musicBrainzAlbumReleaseID) {
					done := sn != nil
					sn = &song{
						id:        s.ID,
						albumID:   s.AlbumID,
						albumName: s.AlbumName,
					}
					if done {
						break
					}
				}
			} else if media.album != nil && s.AlbumName == media.album {
				sn = &song{
					id:        s.ID,
					albumID:   s.AlbumID,
					albumName: s.AlbumName,
				}
			} else {
				sn = &song{
					id:        s.ID,
					albumID:   s.AlbumID,
					albumName: s.AlbumName,
				}
				break
			}
		}
		if sn != nil {
			return sn, nil
		}
	}

	sn, err := s.tx.Song().FindByPath(ctx, media.path, repos.IncludeSongInfoBare())
	if err == nil {
		return &song{
			id:        sn.ID,
			albumName: sn.AlbumName,
			albumID:   sn.AlbumID,
		}, nil
	} else if !errors.Is(err, repos.ErrNotFound) {
		return nil, fmt.Errorf("find or create song by path: %w", err)
	}

	sc, err := s.tx.Song().Create(ctx, repos.CreateSongParams{
		Path:           media.path,
		AlbumID:        nil,
		Title:          media.title,
		Track:          media.track,
		Year:           media.year,
		Size:           int64(media.size),
		ContentType:    media.contentType,
		Duration:       repos.NewDurationMS(int64(media.lengthMS)),
		BitRate:        media.bitrate,
		SamplingRate:   media.sampleRate,
		ChannelCount:   media.channels,
		Disc:           media.disc,
		BPM:            media.bpm,
		MusicBrainzID:  media.musicBrainzSongID,
		ReplayGain:     media.replayGainTrack,
		ReplayGainPeak: media.replayGainTrackPeak,
		Lyrics:         media.lyrics,
		CoverID:        nil,
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
		return "", fmt.Errorf("find album id: %w", repos.ErrNotFound)
	}
	albums, err := s.tx.Album().FindAlbumsByNameWithArtistMatchCount(ctx, *albumName, albumArtists)
	if err != nil {
		return "", fmt.Errorf("find album id: %w", err)
	}
	if len(albums) == 0 {
		return "", fmt.Errorf("find album id: %w", repos.ErrNotFound)
	}
	if musicBrainzAlbumID != nil {
		for _, a := range albums {
			if a.AlbumMusicBrainzID == nil {
				continue
			}
			if *a.AlbumMusicBrainzID == *musicBrainzAlbumID {
				return a.AlbumID, nil
			}
		}
	}
	if len(albumArtists) == 0 {
		for _, a := range albums {
			if a.ArtistMatches == 0 {
				return a.AlbumID, nil
			}
		}
		return albums[0].AlbumID, nil
	}
	var bestMatch string
	var matches int
	for _, a := range albums {
		if int(a.ArtistMatches) > matches {
			matches = int(a.ArtistMatches)
			bestMatch = a.AlbumID
		}
	}
	if matches > 0 {
		return bestMatch, nil
	}
	return "", fmt.Errorf("find album id: %w", repos.ErrNotFound)
}

func (s *Scanner) scanMediaDir(ctx context.Context, path string, c chan<- mediaFile) {
	defer s.waitGroup.Done()
	if !config.ScanHidden() && filepath.Base(path)[0] == '.' {
		return
	}
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
			_ = os.MkdirAll(scanCoverDir, 0755)
			aID := crossonic.GenID()
			coverPath := filepath.Join(scanCoverDir, aID+".jpg")
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

	var aID *string
	idStr, ok := readSingleTag(tags, "crossonic_id_"+s.instanceID)
	if ok && strings.HasPrefix(idStr, "tr_") {
		aID = &idStr
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

	isCompilation := readSingleBoolTag(tags, "compilation")

	artists := readStringTags(tags, "artists", "artist")
	albumArtists := readStringTags(tags, "albumartists", "album_artists", "albumartist", "album_artist")
	if !isCompilation && len(albumArtists) == 0 && len(artists) > 0 {
		albumArtists = []string{artists[0]}
	}

	c <- mediaFile{
		id:                        aID,
		path:                      path,
		size:                      info.Size(),
		updated:                   info.ModTime(),
		contentType:               mime.TypeByExtension(ext),
		coverPath:                 coverPath,
		bitrate:                   props.Bitrate,
		channels:                  props.Channels,
		lengthMS:                  props.LengthMs,
		sampleRate:                props.Samplerate,
		title:                     title,
		album:                     readSingleTagOptional(tags, "album"),
		artists:                   artists,
		albumArtists:              albumArtists,
		bpm:                       readSingleIntTagOptional(tags, "bpm"),
		compilation:               isCompilation,
		year:                      readSingleIntTagFirstOptional(tags, "-", "originalyear", "year", "originaldate", "date"),
		track:                     readSingleIntTagFirstOptional(tags, "/", "tracknumber"),
		disc:                      readSingleIntTagFirstOptional(tags, "/", "discnumber"),
		genres:                    readStringTags(tags, "genres", "genre"),
		labels:                    readStringTags(tags, "labels", "label"),
		musicBrainzSongID:         readSingleTagOptional(tags, "musicbrainz_trackid"),
		musicBrainzAlbumID:        readSingleTagOptional(tags, "musicbrainz_releasegroupid"),
		musicBrainzAlbumReleaseID: readSingleTagOptional(tags, "musicbrainz_albumid"),
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
