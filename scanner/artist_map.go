package scanner

import (
	"context"
	"errors"
	"fmt"

	"github.com/juho05/crossonic-server/repos"
	"github.com/juho05/crossonic-server/util"
)

type artist struct {
	id             string
	mbid           *string
	musicFolderIDs map[int]struct{}
	updated        bool
}

type artistMap struct {
	// artist name -> artists
	artists map[string][]*artist

	// artist name -> image path
	artistImages map[string]string
}

func newArtistMapFromDB(ctx context.Context, s *Scanner) (*artistMap, error) {
	artists, err := s.tx.Artist().FindAll(ctx, repos.FindArtistsParams{}, repos.IncludeArtistInfoBare())
	if err != nil {
		return nil, fmt.Errorf("find all artists: %w", err)
	}

	artistMap := &artistMap{
		artists: make(map[string][]*artist, int(float64(len(artists))*0.8)),
	}

	artistIDMap := make(map[string]*artist, len(artists))

	for _, a := range artists {
		art := &artist{
			id:             a.ID,
			mbid:           a.MusicBrainzID,
			musicFolderIDs: map[int]struct{}{},
		}
		artistMap.artists[a.Name] = append(artistMap.artists[a.Name], art)
		artistIDMap[art.id] = art
	}

	if s.fullScan {
		artistMusicFolderAssociations, err := s.tx.MusicFolder().GetAllArtistAsssociations(ctx)
		if err != nil {
			return nil, fmt.Errorf("get all artist music folder associations: %w", err)
		}

		for _, a := range artistMusicFolderAssociations {
			artistIDMap[a.ArtistID].musicFolderIDs[a.MusicFolderID] = struct{}{}
		}
	}

	return artistMap, nil
}

func (a *artistMap) findOrCreate(ctx context.Context, s *Scanner, name string, mbid *string, musicFolderID int) (string, error) {
	var found *artist
	for _, a := range a.artists[name] {
		// match mbid
		if a.mbid != nil && mbid != nil {
			if *a.mbid == *mbid {
				found = a
				break
			}
			continue
		}

		// not enough data -> artists equal
		found = a
		break
	}

	if found != nil {
		if !found.updated {
			var changed bool
			if util.EqPtrVals(found.mbid, mbid) {
				found.mbid = mbid
				changed = true
			}
			found.updated = true
			if changed || s.fullScan {
				err := a.updateArtist(ctx, s, name, found)
				if err != nil {
					return "", fmt.Errorf("update artist: %w", err)
				}
			}
		}
		if _, ok := found.musicFolderIDs[musicFolderID]; !ok {
			found.musicFolderIDs[musicFolderID] = struct{}{}
		}
		return found.id, nil
	}

	art := &artist{
		mbid:    mbid,
		updated: true,
		musicFolderIDs: map[int]struct{}{
			musicFolderID: {},
		},
	}
	a.artists[name] = append(a.artists[name], art)

	err := a.createArtist(ctx, s, name, art)
	if err != nil {
		return "", fmt.Errorf("create artist: %w", err)
	}

	return art.id, nil
}

func (a *artistMap) updateArtist(ctx context.Context, s *Scanner, name string, artist *artist) error {
	err := s.tx.Artist().Update(ctx, artist.id, repos.UpdateArtistParams{
		Name:          repos.NewOptionalFull(name),
		MusicBrainzID: repos.NewOptionalFull(artist.mbid),
	})
	if err != nil {
		if errors.Is(err, repos.ErrNotFound) {
			err = a.createArtist(ctx, s, name, artist)
			if err != nil {
				return fmt.Errorf("update: create artist: %w", err)
			}
		}
		return fmt.Errorf("update: %w", err)
	}
	return nil
}

func (a *artistMap) createArtist(ctx context.Context, s *Scanner, name string, artist *artist) error {
	artID, err := s.tx.Artist().Create(ctx, repos.CreateArtistParams{
		Name:          name,
		MusicBrainzID: artist.mbid,
	})
	if err != nil {
		return fmt.Errorf("create: %w", err)
	}
	artist.id = artID
	return nil
}

func (a *artistMap) updateMusicFolderAssociations(ctx context.Context, s *Scanner) error {
	err := s.tx.MusicFolder().DeleteAllArtistAssociations(ctx)
	if err != nil {
		return fmt.Errorf("delete all artist associations: %w", err)
	}

	associations := make([]repos.ArtistMusicFolderAssociation, 0, len(a.artists))
	for _, artists := range a.artists {
		for _, artist := range artists {
			for musicFolderID := range artist.musicFolderIDs {
				associations = append(associations, repos.ArtistMusicFolderAssociation{
					MusicFolderID: musicFolderID,
					ArtistID:      artist.id,
				})
			}
		}
	}

	err = s.tx.MusicFolder().CreateArtistAssociations(ctx, associations)
	if err != nil {
		return fmt.Errorf("create artist associations: %w", err)
	}

	err = s.tx.MusicFolder().DeleteArtistAssociationsWithoutSongs(ctx)
	if err != nil {
		return fmt.Errorf("delete artist associations without songs: %w", err)
	}

	return nil
}
