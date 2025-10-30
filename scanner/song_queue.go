package scanner

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/juho05/crossonic-server"
	"github.com/juho05/crossonic-server/audiotags"
	"github.com/juho05/crossonic-server/repos"
	"github.com/juho05/crossonic-server/util"
)

type mediaFile struct {
	id           *string
	path         string
	size         int64
	contentType  string
	lastModified time.Time

	cover *string

	bitrate    int
	channels   int
	lengthMS   int
	sampleRate int

	title               string
	albumName           *string
	albumMBID           *string
	albumReleaseMBID    *string
	artistNames         []string
	artistMBIDs         []string
	albumArtistNames    []string
	albumArtistMBIDs    []string
	bpm                 *int
	originalDate        *repos.Date
	releaseDate         *repos.Date
	track               *int
	disc                *int
	discTitle           *string
	genres              []string
	musicBrainzID       *string
	replayGain          *float64
	replayGainPeak      *float64
	albumReplayGain     *float64
	albumReplayGainPeak *float64
	lyrics              *string

	recordLabels  []string
	releaseTypes  []string
	isCompilation bool

	albumVersion *string
}

type song struct {
	id           *string
	hasIDTag     bool
	path         string
	size         int64
	contentType  string
	lastModified time.Time

	bitrate    int
	channels   int
	lengthMS   int
	sampleRate int

	title                     string
	albumID                   *string
	albumName                 *string
	artistNames               []string
	artistIDs                 []string
	bpm                       *int
	originalDate              *repos.Date
	releaseDate               *repos.Date
	track                     *int
	disc                      *int
	genres                    []string
	musicBrainzID             *string
	albumMusicBrainzID        *string
	albumReleaseMusicBrainzID *string
	replayGain                *float64
	replayGainPeak            *float64
	lyrics                    *string
}

func (s *Scanner) runSaveSongsLoop(ctx context.Context) error {
	songs := make([]*mediaFile, 0, songQueueBatchSize)

	updateSongFiles := make(chan *song, updateSongFilesWorkerCount)
	var updateSongFilesWait sync.WaitGroup
	var updateSongFilesErr error
	for range updateSongFilesWorkerCount {
		updateSongFilesWait.Add(1)
		go func() {
			defer updateSongFilesWait.Done()
			for song := range updateSongFiles {
				// store id
				err := s.setCrossonicID(song.path, *song.id)
				if err != nil {
					if updateSongFilesErr == nil {
						updateSongFilesErr = fmt.Errorf("write crossonic id to file: %w", err)
					}
					return
				}

				// clear cache
				if song.lastModified.After(s.lastScan) {
					for _, key := range s.transcodeCache.Keys() {
						if strings.HasPrefix(key, *song.id) {
							err = s.transcodeCache.DeleteObject(key)
							if err != nil {
								updateSongFilesErr = fmt.Errorf("clear cache for song %s: %w", *song.id, err)
								return
							}
						}
					}
				}
			}
		}()
	}

	for song := range s.songQueue {
		if updateSongFilesErr != nil {
			break
		}
		songs = append(songs, song)
		if len(songs) == songQueueBatchSize {
			err := s.createOrUpdateSongs(ctx, songs, updateSongFiles)
			if err != nil {
				return fmt.Errorf("create or update songs: %w", err)
			}
			songs = songs[:0]
		}
	}

	if updateSongFilesErr == nil {
		err := s.createOrUpdateSongs(ctx, songs, updateSongFiles)
		if err != nil {
			return fmt.Errorf("create or update songs: %w", err)
		}
	}

	close(updateSongFiles)
	updateSongFilesWait.Wait()
	return updateSongFilesErr
}

func (s *Scanner) createOrUpdateSongs(ctx context.Context, mediaFiles []*mediaFile, updateSongFiles chan<- *song) error {
	create := make([]*song, 0, len(mediaFiles))
	var update []*song
	if !s.firstScan {
		update = make([]*song, 0, len(mediaFiles))
	}

	var changedCount int
	for _, media := range mediaFiles {
		song := &song{
			id:                        media.id,
			hasIDTag:                  media.id != nil,
			path:                      media.path,
			size:                      media.size,
			contentType:               media.contentType,
			lastModified:              media.lastModified,
			bitrate:                   media.bitrate,
			channels:                  media.channels,
			sampleRate:                media.sampleRate,
			lengthMS:                  media.lengthMS,
			title:                     media.title,
			bpm:                       media.bpm,
			releaseDate:               media.releaseDate,
			originalDate:              media.originalDate,
			track:                     media.track,
			disc:                      media.disc,
			genres:                    media.genres,
			musicBrainzID:             media.musicBrainzID,
			albumMusicBrainzID:        media.albumMBID,
			albumReleaseMusicBrainzID: media.albumReleaseMBID,
			replayGain:                media.replayGain,
			replayGainPeak:            media.replayGainPeak,
			lyrics:                    media.lyrics,
		}
		song.artistNames = media.artistNames
		song.artistIDs = make([]string, 0, len(media.artistNames))
		for i, a := range media.artistNames {
			var mbid *string
			if i < len(media.artistMBIDs) {
				mbid = &media.artistMBIDs[i]
			}
			id, err := s.artists.findOrCreate(ctx, s, a, mbid)
			if err != nil {
				return fmt.Errorf("find or create artist: %s", err)
			}
			song.artistIDs = append(song.artistIDs, id)
		}

		albumArtists := make([]findOrCreateAlbumParamsArtist, 0, len(media.albumArtistNames))
		for i, a := range media.albumArtistNames {
			var mbid *string
			if i < len(media.albumArtistMBIDs) {
				mbid = &media.albumArtistMBIDs[i]
			}
			id, err := s.artists.findOrCreate(ctx, s, a, mbid)
			if err != nil {
				return fmt.Errorf("find or create album artist: %s", err)
			}
			albumArtists = append(albumArtists, findOrCreateAlbumParamsArtist{
				id:   id,
				name: a,
			})
		}

		if media.albumName != nil {
			alb, err := s.albums.findOrCreate(ctx, s, *media.albumName, findOrCreateAlbumParams{
				mbid:           media.albumMBID,
				releaseMBID:    media.albumReleaseMBID,
				originalDate:   media.originalDate,
				releaseDate:    media.releaseDate,
				recordLabels:   media.recordLabels,
				releaseTypes:   media.releaseTypes,
				isCompilation:  &media.isCompilation,
				replayGain:     media.albumReplayGain,
				replayGainPeak: media.albumReplayGainPeak,
				artists:        albumArtists,
				cover:          media.cover,
				songPath:       media.path,
				version:        media.albumVersion,
			})
			if err != nil {
				return fmt.Errorf("find or create album: %w", err)
			}

			if media.disc != nil {
				err := s.albums.updateDiscTitle(ctx, s, alb, *media.disc, media.discTitle)
				if err != nil {
					return fmt.Errorf("update disc title: %w", err)
				}
			}

			song.albumName = media.albumName
			song.albumID = &alb.id
		}

		if song.id == nil {
			create = append(create, song)
		} else {
			update = append(update, song)
		}
		if song.lastModified.After(s.lastScan) {
			changedCount++
		}
	}

	failed, err := s.updateSongs(ctx, update)
	if err != nil {
		return fmt.Errorf("update songs: %w", err)
	}

	err = s.createSongs(ctx, slices.Concat(create, failed))
	if err != nil {
		return fmt.Errorf("create songs: %w", err)
	}

	for _, s := range update {
		if !s.hasIDTag {
			updateSongFiles <- s
			s.hasIDTag = true
		}
	}

	for _, s := range create {
		if !s.hasIDTag {
			updateSongFiles <- s
			s.hasIDTag = true
		}
	}

	if changedCount > 0 {
		changedSongs := make([]*song, 0, changedCount)
		changedSongIDs := make([]string, 0, changedCount)
		for _, song := range create {
			if song.lastModified.After(s.lastScan) {
				changedSongs = append(changedSongs, song)
				changedSongIDs = append(changedSongIDs, *song.id)
			}
		}
		for _, song := range update {
			if song.lastModified.After(s.lastScan) {
				changedSongs = append(changedSongs, song)
				changedSongIDs = append(changedSongIDs, *song.id)
			}
		}

		err = s.tx.Song().DeleteArtistConnections(ctx, changedSongIDs)
		if err != nil {
			return fmt.Errorf("delete artist connections: %w", err)
		}

		songArtistConns := make([]repos.SongArtistConnection, 0, len(changedSongs))
		for _, s := range changedSongs {
			for i, aID := range s.artistIDs {
				songArtistConns = append(songArtistConns, repos.SongArtistConnection{
					SongID:   *s.id,
					ArtistID: aID,
					Index:    i,
				})
			}
		}
		err = s.tx.Song().CreateArtistConnections(ctx, songArtistConns)
		if err != nil {
			return fmt.Errorf("create artist connections: %w", err)
		}

		genres := make(map[string][]string, len(changedSongs)/2)
		for _, song := range changedSongs {
			for _, g := range song.genres {
				genres[g] = append(genres[g], *song.id)
			}
		}

		err = s.tx.Genre().CreateIfNotExists(ctx, util.MapKeys(genres))
		if err != nil {
			return fmt.Errorf("create genres: %w", err)
		}

		err = s.tx.Song().DeleteGenreConnections(ctx, changedSongIDs)
		if err != nil {
			return fmt.Errorf("delete genre connections: %w", err)
		}

		songGenreConns := make([]repos.SongGenreConnection, 0, len(changedSongIDs))
		for g, songIDs := range genres {
			for _, sID := range songIDs {
				songGenreConns = append(songGenreConns, repos.SongGenreConnection{
					SongID: sID,
					Genre:  g,
				})
			}
		}
		err = s.tx.Song().CreateGenreConnections(ctx, songGenreConns)
		if err != nil {
			return fmt.Errorf("create genre connections: %w", err)
		}
	}

	return nil
}

func (s *Scanner) updateSongs(ctx context.Context, songs []*song) ([]*song, error) {
	count, err := s.tx.Song().TryUpdateAll(ctx, util.Map(songs, func(s *song) repos.UpdateSongAllParams {
		return repos.UpdateSongAllParams{
			ID:             *s.id,
			Path:           s.path,
			AlbumID:        s.albumID,
			Title:          s.title,
			Track:          s.track,
			OriginalDate:   s.originalDate,
			ReleaseDate:    s.releaseDate,
			Size:           s.size,
			ContentType:    s.contentType,
			Duration:       repos.NewDurationMS(int64(s.lengthMS)),
			BitRate:        s.bitrate,
			SamplingRate:   s.sampleRate,
			ChannelCount:   s.channels,
			Disc:           s.disc,
			BPM:            s.bpm,
			MusicBrainzID:  s.musicBrainzID,
			ReplayGain:     s.replayGain,
			ReplayGainPeak: s.replayGainPeak,
			Lyrics:         s.lyrics,
			AlbumName:      s.albumName,
			ArtistNames:    s.artistNames,
		}
	}))
	if err != nil {
		return nil, fmt.Errorf("try update all: %w", err)
	}
	if count == len(songs) {
		return nil, nil
	}

	failedIDs, err := s.tx.Song().FindNonExistentIDs(ctx, util.Map(songs, func(s *song) string {
		return *s.id
	}))
	if err != nil {
		return nil, fmt.Errorf("find non existent ids: %w", err)
	}

	failedIDsMap := make(map[string]struct{}, len(failedIDs))
	for _, f := range failedIDs {
		failedIDsMap[f] = struct{}{}
	}

	failed := make([]*song, 0, len(failedIDs))
	for _, song := range songs {
		if _, ok := failedIDsMap[*song.id]; ok {
			failed = append(failed, song)
		}
	}

	return failed, nil
}

func (s *Scanner) createSongs(ctx context.Context, songs []*song) error {
	if s.firstScan {
		err := s.createSongsInDB(ctx, songs)
		if err != nil {
			return fmt.Errorf("create songs in db: %w", err)
		}
		return nil
	}

	paths := make([]string, 0, len(songs))
	mbids := make([]string, 0, len(songs))
	for _, s := range songs {
		paths = append(paths, s.path)
		if s.musicBrainzID != nil {
			mbids = append(mbids, *s.musicBrainzID)
		}
	}

	matches, err := s.tx.Song().FindAllByPathOrMBID(ctx, paths, mbids, repos.IncludeSongInfo{
		Album: true,
	})
	if err != nil {
		return fmt.Errorf("find all by path or mbid: %w", err)
	}

	type mbidMatch struct {
		id          string
		releaseMBID *string
		albumMBID   *string
	}

	pathMatches := make(map[string]string, len(matches))
	mbidMatches := make(map[string][]mbidMatch, len(matches))

	for _, m := range matches {
		pathMatches[m.Path] = m.ID
		if m.MusicBrainzID != nil {
			mbidMatches[*m.MusicBrainzID] = append(mbidMatches[*m.MusicBrainzID], mbidMatch{
				id:          m.ID,
				releaseMBID: m.AlbumReleaseMBID,
				albumMBID:   m.AlbumMusicBrainzID,
			})
		}
	}

	create := make([]*song, 0, len(songs))
	update := make([]*song, 0)

songLoop:
	for _, s := range songs {
		if id, ok := pathMatches[s.path]; ok {
			s.id = &id
			update = append(update, s)
			continue
		}
		if s.musicBrainzID != nil {
			if matches, ok := mbidMatches[*s.musicBrainzID]; ok {
				// match by album release MBID
				if s.albumReleaseMusicBrainzID != nil {
					for _, m := range matches {
						if util.EqPtrVals(m.releaseMBID, s.albumReleaseMusicBrainzID) {
							s.id = &m.id
							update = append(update, s)
							continue songLoop
						}
					}
				}
				// match by album MBID
				for _, m := range matches {
					if m.releaseMBID == nil && util.EqPtrVals(m.albumMBID, s.albumMusicBrainzID) {
						s.id = &m.id
						update = append(update, s)
						continue songLoop
					}
				}
			}
		}
		create = append(create, s)
	}

	failed, err := s.updateSongs(ctx, update)
	if err != nil {
		return fmt.Errorf("update songs: %w", err)
	}
	create = append(create, failed...)

	err = s.createSongsInDB(ctx, create)
	if err != nil {
		return fmt.Errorf("create songs in db: %w", err)
	}

	return nil
}

func (s *Scanner) createSongsInDB(ctx context.Context, songs []*song) error {
	for _, s := range songs {
		id := crossonic.GenIDSong()
		s.id = &id
		s.hasIDTag = false
	}
	err := s.tx.Song().CreateAll(ctx, util.Map(songs, func(s *song) repos.CreateSongParams {
		return repos.CreateSongParams{
			ID:             s.id,
			Path:           s.path,
			AlbumID:        s.albumID,
			Title:          s.title,
			Track:          s.track,
			OriginalDate:   s.originalDate,
			ReleaseDate:    s.releaseDate,
			Size:           s.size,
			ContentType:    s.contentType,
			Duration:       repos.NewDurationMS(int64(s.lengthMS)),
			BitRate:        s.bitrate,
			SamplingRate:   s.sampleRate,
			ChannelCount:   s.channels,
			Disc:           s.disc,
			BPM:            s.bpm,
			MusicBrainzID:  s.musicBrainzID,
			ReplayGain:     s.replayGain,
			ReplayGainPeak: s.replayGainPeak,
			Lyrics:         s.lyrics,
			AlbumName:      s.albumName,
			ArtistNames:    s.artistNames,
		}
	}))
	if err != nil {
		return fmt.Errorf("create all: %w", err)
	}
	return nil
}

func (s *Scanner) setCrossonicID(path, id string) error {
	success := audiotags.WriteTag(path, "crossonic_id_"+s.instanceID, id)
	if !success {
		return fmt.Errorf("write failed")
	}
	return nil
}
