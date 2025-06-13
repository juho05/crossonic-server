package scanner

import (
	"context"
	"errors"
	"fmt"

	"slices"

	"github.com/juho05/crossonic-server/repos"
	"github.com/juho05/crossonic-server/util"
)

type album struct {
	id             string
	mbid           *string
	releaseMBID    *string
	year           *int
	recordLabels   []string
	releaseTypes   []string
	isCompilation  *bool
	replayGain     *float64
	replayGainPeak *float64
	albumArtistIDs map[string]int
	updated        bool
}

type albumMap struct {
	albums map[string][]*album
}

type findOrCreateAlbumParams struct {
	mbid           *string
	releaseMBID    *string
	year           *int
	recordLabels   []string
	releaseTypes   []string
	isCompilation  *bool
	replayGain     *float64
	replayGainPeak *float64
	albumArtistIDs []string
	cover          *string
	songPath       string
}

func newAlbumMapFromDB(ctx context.Context, s *Scanner) (*albumMap, error) {
	albums, err := s.tx.Album().FindAll(ctx, repos.FindAlbumParams{}, repos.IncludeAlbumInfoBare())
	if err != nil {
		return nil, fmt.Errorf("find all albums: %w", err)
	}

	connections := make(map[string]map[string]int, len(albums))

	artistConnections, err := s.tx.Album().GetAllArtistConnections(ctx)
	if err != nil {
		return nil, fmt.Errorf("find all album artist connections: %w", err)
	}
	for _, c := range artistConnections {
		if connections[c.AlbumID] == nil {
			connections[c.AlbumID] = make(map[string]int)
		}
		connections[c.AlbumID][c.ArtistID] = len(connections[c.AlbumID])
	}

	albumMap := &albumMap{
		albums: make(map[string][]*album, int(float64(len(albums))*0.8)),
	}

	for _, a := range albums {
		albumArtistIDs := connections[a.ID]
		if albumArtistIDs == nil {
			albumArtistIDs = make(map[string]int, 0)
		}
		alb := &album{
			id:             a.ID,
			mbid:           a.MusicBrainzID,
			releaseMBID:    a.ReleaseMBID,
			year:           a.Year,
			albumArtistIDs: albumArtistIDs,
		}
		albumMap.albums[a.Name] = append(albumMap.albums[a.Name], alb)
	}

	return albumMap, nil
}

func (a *albumMap) findOrCreate(ctx context.Context, s *Scanner, name string, params findOrCreateAlbumParams) (string, error) {
	var found *album
	for _, a := range a.albums[name] {
		// match release mbid
		if a.releaseMBID != nil && params.releaseMBID != nil {
			if *a.releaseMBID == *params.releaseMBID {
				found = a
				break
			}
			continue
		}

		// match mbid
		if a.mbid != nil && params.mbid != nil {
			if *a.mbid == *params.mbid {
				found = a
				break
			}
			continue
		}

		// match album artists
		if len(a.albumArtistIDs) > 0 && len(params.albumArtistIDs) > 0 {
			if len(a.albumArtistIDs) == len(params.albumArtistIDs) {
				match := true
				for _, artist := range params.albumArtistIDs {
					if _, ok := a.albumArtistIDs[artist]; !ok {
						match = false
						break
					}
				}
				if match {
					found = a
					break
				}
			}
			continue
		}

		// match year
		if a.year != nil && params.year != nil {
			if *a.year == *params.year {
				found = a
				break
			}
			continue
		}

		// not enough data -> albums equal
		found = a
		break
	}

	if found != nil {
		if !found.updated {
			var changed bool
			if !util.EqPtrVals(found.mbid, params.mbid) {
				found.mbid = params.mbid
				changed = true
			}
			if !util.EqPtrVals(found.releaseMBID, params.releaseMBID) {
				found.releaseMBID = params.releaseMBID
				changed = true
			}
			if !util.EqPtrVals(found.year, params.year) {
				found.year = params.year
				changed = true
			}
			if !slices.Equal(found.recordLabels, params.recordLabels) {
				found.recordLabels = params.recordLabels
				changed = true
			}
			if !slices.Equal(found.releaseTypes, params.releaseTypes) {
				found.releaseTypes = params.releaseTypes
				changed = true
			}
			if !util.EqPtrVals(found.isCompilation, params.isCompilation) {
				found.isCompilation = params.isCompilation
				changed = true
			}
			if !util.EqPtrVals(found.replayGain, params.replayGain) {
				found.replayGain = params.replayGain
				changed = true
			}
			if !util.EqPtrVals(found.replayGainPeak, params.replayGainPeak) {
				found.replayGainPeak = params.replayGainPeak
				changed = true
			}
			if len(found.albumArtistIDs) != len(params.albumArtistIDs) {
				found.albumArtistIDs = make(map[string]int, len(params.albumArtistIDs))
				for i, art := range params.albumArtistIDs {
					found.albumArtistIDs[art] = i
				}
			} else {
				equal := true
				for i, art := range params.albumArtistIDs {
					if i2, ok := found.albumArtistIDs[art]; !ok || i != i2 {
						equal = false
						break
					}
				}
				if !equal {
					found.albumArtistIDs = make(map[string]int, len(params.albumArtistIDs))
					for i, art := range params.albumArtistIDs {
						found.albumArtistIDs[art] = i
					}
				}
			}
			found.updated = true

			if changed {
				err := a.updateAlbum(ctx, s, name, found)
				if err != nil {
					return "", fmt.Errorf("update album: %w", err)
				}
			}

			s.setAlbumCover <- albumCover{
				id:       found.id,
				cover:    params.cover,
				songPath: params.songPath,
			}
		}
		return found.id, nil
	}

	albumArtists := make(map[string]int, len(params.albumArtistIDs))
	for i, art := range params.albumArtistIDs {
		albumArtists[art] = i
	}

	alb := &album{
		mbid:           params.mbid,
		releaseMBID:    params.releaseMBID,
		year:           params.year,
		recordLabels:   params.recordLabels,
		releaseTypes:   params.releaseTypes,
		isCompilation:  params.isCompilation,
		replayGain:     params.replayGain,
		replayGainPeak: params.replayGainPeak,
		albumArtistIDs: albumArtists,
		updated:        true,
	}
	a.albums[name] = append(a.albums[name], alb)

	// sets alb.id
	err := a.createAlbum(ctx, s, name, alb)
	if err != nil {
		return "", fmt.Errorf("create album: %w", err)
	}

	s.setAlbumCover <- albumCover{
		id:       alb.id,
		cover:    params.cover,
		songPath: params.songPath,
	}

	return alb.id, nil
}

func (a *albumMap) updateAlbum(ctx context.Context, s *Scanner, name string, album *album) error {
	err := s.tx.Album().Update(ctx, album.id, repos.UpdateAlbumParams{
		Year:           repos.NewOptionalFull(album.year),
		RecordLabels:   repos.NewOptionalFull(repos.StringList(album.recordLabels)),
		MusicBrainzID:  repos.NewOptionalFull(album.mbid),
		ReleaseMBID:    repos.NewOptionalFull(album.releaseMBID),
		ReleaseTypes:   repos.NewOptionalFull(repos.StringList(album.releaseTypes)),
		IsCompilation:  repos.NewOptionalFull(album.isCompilation),
		ReplayGain:     repos.NewOptionalFull(album.replayGain),
		ReplayGainPeak: repos.NewOptionalFull(album.replayGainPeak),
	})
	if err != nil {
		if errors.Is(err, repos.ErrNotFound) {
			err = a.createAlbum(ctx, s, name, album)
			if err != nil {
				return fmt.Errorf("update: create album: %w", err)
			}
		}
		return fmt.Errorf("update: %w", err)
	}
	return nil
}

func (a *albumMap) createAlbum(ctx context.Context, s *Scanner, name string, album *album) error {
	albID, err := s.tx.Album().Create(ctx, repos.CreateAlbumParams{
		Name:           name,
		Year:           album.year,
		RecordLabels:   album.recordLabels,
		MusicBrainzID:  album.mbid,
		ReleaseMBID:    album.releaseMBID,
		ReleaseTypes:   album.releaseTypes,
		IsCompilation:  album.isCompilation,
		ReplayGain:     album.replayGain,
		ReplayGainPeak: album.replayGainPeak,
	})
	if err != nil {
		return fmt.Errorf("create: %w", err)
	}
	album.id = albID
	return nil
}

func (a *albumMap) updateArtists(ctx context.Context, s *Scanner) error {
	err := s.tx.Album().RemoveAllArtistConnections(ctx)
	if err != nil {
		return fmt.Errorf("remove all artist connections: %w", err)
	}

	connections := make([]repos.AlbumArtistConnection, 0, len(a.albums))
	for _, albs := range a.albums {
		for _, alb := range albs {
			for artistID, i := range alb.albumArtistIDs {
				connections = append(connections, repos.AlbumArtistConnection{
					AlbumID:  alb.id,
					ArtistID: artistID,
					Index:    i,
				})
			}
		}
	}

	err = s.tx.Album().CreateArtistConnections(ctx, connections)
	if err != nil {
		return fmt.Errorf("create new artist connections: %w", err)
	}

	return nil
}
