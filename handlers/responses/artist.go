package responses

import (
	"time"

	"github.com/juho05/crossonic-server/repos"
	"github.com/juho05/crossonic-server/util"
)

type Artist struct {
	ID            string     `xml:"id,attr,omitempty"       json:"id"`
	Name          string     `xml:"name,attr"               json:"name"`
	CoverArt      *string    `xml:"coverArt,attr,omitempty" json:"coverArt,omitempty"`
	AlbumCount    *int       `xml:"albumCount,attr,omitempty"         json:"albumCount,omitempty"`
	Starred       *time.Time `xml:"starred,attr,omitempty"       json:"starred,omitempty"`
	MusicBrainzID *string    `xml:"musicBrainzID,omitempty" json:"musicBrainzID,omitempty"`
	UserRating    *int       `xml:"userRating,attr,omitempty"    json:"userRating,omitempty"`
	AverageRating *float64   `xml:"averageRating,attr,omitempty" json:"averageRating,omitempty"`
	Albums        []*Album   `xml:"album,omitempty" json:"album,omitempty"`
}

type ArtistRef struct {
	ID   string `xml:"id,attr" json:"id"`
	Name string `xml:"name,attr" json:"name"`
}

func NewArtist(a *repos.CompleteArtist) *Artist {
	if a == nil {
		return nil
	}

	artist := &Artist{
		ID:            a.ID,
		Name:          a.Name,
		MusicBrainzID: a.MusicBrainzID,
	}

	if a.ArtistAlbumInfo != nil {
		artist.AlbumCount = &a.AlbumCount
	}

	if a.ArtistAnnotations != nil {
		artist.Starred = a.Starred
		artist.UserRating = a.UserRating
		artist.AverageRating = a.AverageRating
	}

	if HasCoverArt(a.ID) {
		artist.CoverArt = &a.ID
	}
	return artist
}

func NewArtists(a []*repos.CompleteArtist) []*Artist {
	return util.Map(a, NewArtist)
}

func newArtistRef(a repos.ArtistRef) *ArtistRef {
	return &ArtistRef{
		ID:   a.ID,
		Name: a.Name,
	}
}

func newArtistRefs(a []repos.ArtistRef) []*ArtistRef {
	return util.Map(a, newArtistRef)
}
