package scanner

import (
	"context"
	"errors"
	"fmt"

	"github.com/juho05/crossonic-server/repos"
	"github.com/juho05/crossonic-server/util"
)

type artist struct {
	id      string
	mbid    *string
	updated bool
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

	for _, a := range artists {
		alb := &artist{
			id:   a.ID,
			mbid: a.MusicBrainzID,
		}
		artistMap.artists[a.Name] = append(artistMap.artists[a.Name], alb)
	}

	return artistMap, nil
}

func (a *artistMap) findOrCreate(ctx context.Context, s *Scanner, name string, mbid *string) (string, error) {
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
			if changed {
				err := a.updateArtist(ctx, s, name, found)
				if err != nil {
					return "", fmt.Errorf("update artist: %w", err)
				}
			}
		}
		return found.id, nil
	}

	art := &artist{
		mbid:    mbid,
		updated: true,
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
