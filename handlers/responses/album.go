package responses

import (
	"github.com/juho05/crossonic-server/config"
	"slices"
	"time"

	"github.com/juho05/crossonic-server/repos"
	"github.com/juho05/crossonic-server/util"
)

type Album struct {
	ID            string         `xml:"id,attr,omitempty"       json:"id"`
	Created       time.Time      `xml:"created,attr,omitempty"  json:"created,omitempty"`
	ArtistID      *string        `xml:"artistId,attr,omitempty" json:"artistId,omitempty"`
	Artist        *string        `xml:"artist,attr,omitempty"             json:"artist,omitempty"`
	Artists       []*ArtistRef   `xml:"artists,omitempty"           json:"artists,omitempty"`
	CoverArt      *string        `xml:"coverArt,attr,omitempty" json:"coverArt,omitempty"`
	Title         string         `xml:"title,attr"  json:"title"`
	Name          string         `xml:"name,attr"              json:"name"`
	SongCount     int            `xml:"songCount,attr"         json:"songCount"`
	Duration      int            `xml:"duration,attr"          json:"duration"`
	PlayCount     *int           `xml:"playCount,attr,omitempty" json:"playCount,omitempty"`
	Genre         *string        `xml:"genre,attr,omitempty"   json:"genre,omitempty"`
	Genres        []*GenreRef    `xml:"genres,omitempty"       json:"genres,omitempty"`
	Year          *int           `xml:"year,attr,omitempty"    json:"year,omitempty"`
	Starred       *time.Time     `xml:"starred,attr,omitempty"         json:"starred,omitempty"`
	UserRating    *int           `xml:"userRating,attr,omitempty"      json:"userRating,omitempty"`
	AverageRating *float64       `xml:"averageRating,attr,omitempty"   json:"averageRating,omitempty"`
	Parent        *string        `xml:"parent,attr" json:"parent"`
	IsDir         bool           `xml:"isDir,attr" json:"isDir"`
	Type          string         `xml:"type,attr" json:"type"`
	MediaType     string         `xml:"mediaType,attr" json:"mediaType"`
	Played        *time.Time     `xml:"played,attr,omitempty" json:"played,omitempty"`
	MusicBrainzID *string        `xml:"musicBrainzId,attr,omitempty" json:"musicBrainzId,omitempty"`
	ReleaseMBID   *string        `xml:"releaseMbid,attr,omitempty" json:"releaseMbid,omitempty"`
	RecordLabels  []*RecordLabel `xml:"recordLabels,omitempty" json:"recordLabels,omitempty"`
	ReleaseTypes  []string       `xml:"releaseTypes,omitempty" json:"releaseTypes,omitempty"`
	IsCompilation *bool          `xml:"isCompilation,attr,omitempty" json:"isCompilation,omitempty"`
	DiscTitles    []DiscTitle    `xml:"discTitles,omitempty" json:"discTitles,omitempty"`
}

type DiscTitle struct {
	Disc  int    `xml:"disc,attr" json:"disc"`
	Title string `xml:"title,attr" json:"title"`
}

func NewAlbum(a *repos.CompleteAlbum, conf config.Config) *Album {
	if a == nil {
		return nil
	}
	album := &Album{
		ID:            a.ID,
		Created:       a.Created,
		Title:         a.Name,
		Name:          a.Name,
		Year:          a.Year,
		MusicBrainzID: a.MusicBrainzID,
		ReleaseMBID:   a.ReleaseMBID,
		IsCompilation: a.IsCompilation,
		ReleaseTypes:  a.ReleaseTypes,
		RecordLabels: util.Map(a.RecordLabels, func(label string) *RecordLabel {
			return &RecordLabel{
				Name: label,
			}
		}),
		IsDir:     true,
		Type:      "music",
		MediaType: "album",
	}

	if a.AlbumTrackInfo != nil {
		album.SongCount = a.TrackCount
		album.Duration = a.Duration.Seconds()
	}

	if a.AlbumAnnotations != nil {
		album.Starred = a.Starred
		album.UserRating = a.UserRating
		album.AverageRating = a.AverageRating
	}

	if a.AlbumPlayInfo != nil {
		album.PlayCount = &a.PlayCount
		album.Played = a.LastPlayed
	}

	if a.AlbumLists != nil {
		album.Genres = newGenreRefs(a.Genres)
		album.Genre = util.FirstOrNil(a.Genres)

		album.Artists = newArtistRefs(a.Artists)
		album.Artist = util.FirstOrNilMap(a.Artists, func(a repos.ArtistRef) string {
			return a.Name
		})
		album.ArtistID = util.FirstOrNilMap(a.Artists, func(a repos.ArtistRef) string {
			return a.ID
		})
		album.Parent = album.ArtistID
	}

	if a.DiscTitles != nil {
		titles := make([]DiscTitle, 0, len(a.DiscTitles))
		for disc, title := range a.DiscTitles {
			titles = append(titles, DiscTitle{
				Disc:  disc,
				Title: title,
			})
		}
		slices.SortFunc(titles, func(a, b DiscTitle) int {
			if a.Disc < b.Disc {
				return -1
			}
			if a.Disc > b.Disc {
				return 1
			}
			return 0
		})
		album.DiscTitles = titles
	}

	if HasCoverArt(a.ID, conf) {
		album.CoverArt = &a.ID
	}

	return album
}

func NewAlbums(a []*repos.CompleteAlbum, conf config.Config) []*Album {
	return util.Map(a, func(a *repos.CompleteAlbum) *Album {
		return NewAlbum(a, conf)
	})
}
