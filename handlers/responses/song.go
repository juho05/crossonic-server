package responses

import (
	"path/filepath"
	"time"

	"github.com/juho05/crossonic-server/config"

	"github.com/juho05/crossonic-server/repos"
	"github.com/juho05/crossonic-server/util"
)

type Song struct {
	ID            string       `xml:"id,attr" json:"id"`
	IsDir         bool         `xml:"isDir,attr" json:"isDir"`
	Parent        *string      `xml:"parent,attr" json:"parent"`
	Title         string       `xml:"title,attr" json:"title"`
	Album         *string      `xml:"album,attr,omitempty" json:"album,omitempty"`
	Artist        *string      `xml:"artist,attr,omitempty" json:"artist,omitempty"`
	Track         *int         `xml:"track,attr,omitempty" json:"track,omitempty"`
	Year          *int         `xml:"year,attr,omitempty" json:"year,omitempty"`
	Genre         *string      `xml:"genre,attr,omitempty" json:"genre,omitempty"`
	CoverArt      *string      `xml:"coverArt,attr,omitempty" json:"coverArt,omitempty"`
	Size          int64        `xml:"size,attr" json:"size"`
	ContentType   string       `xml:"contentType,attr" json:"contentType"`
	Suffix        string       `xml:"suffix,attr" json:"suffix"`
	Duration      int          `xml:"duration,attr" json:"duration"`
	BitRate       int          `xml:"bitRate,attr" json:"bitRate"`
	SamplingRate  int          `xml:"samplingRate,attr" json:"samplingRate"`
	ChannelCount  int          `xml:"channelCount,attr" json:"channelCount"`
	UserRating    *int         `xml:"userRating,attr,omitempty" json:"userRating,omitempty"`
	AverageRating *float64     `xml:"averageRating,attr,omitempty" json:"averageRating,omitempty"`
	PlayCount     *int         `xml:"playCount,attr,omitempty" json:"playCount,omitempty"`
	DiscNumber    *int         `xml:"discNumber,attr,omitempty" json:"discNumber,omitempty"`
	Created       time.Time    `xml:"created,attr" json:"created"`
	Starred       *time.Time   `xml:"starred,attr,omitempty" json:"starred,omitempty"`
	AlbumID       *string      `xml:"albumId,attr,omitempty" json:"albumId,omitempty"`
	ArtistID      *string      `xml:"artistId,attr,omitempty" json:"artistId,omitempty"`
	Type          string       `xml:"type,attr" json:"type"`
	MediaType     string       `xml:"mediaType,attr" json:"mediaType"`
	Played        *time.Time   `xml:"played,attr,omitempty" json:"played,omitempty"`
	BPM           *int         `xml:"bpm,attr,omitempty" json:"bpm,omitempty"`
	MusicBrainzID *string      `xml:"musicBrainzId,attr,omitempty" json:"musicBrainzId,omitempty"`
	Genres        []*GenreRef  `xml:"genres,omitempty" json:"genres,omitempty"`
	Artists       []*ArtistRef `xml:"artists,omitempty" json:"artists,omitempty"`
	AlbumArtists  []*ArtistRef `xml:"albumArtists,omitempty" json:"albumArtists,omitempty"`
	ReplayGain    *ReplayGain  `xml:"replayGain,omitempty" json:"replayGain,omitempty"`
}

func NewSong(s *repos.CompleteSong, conf config.Config) *Song {
	if s == nil {
		return nil
	}
	var coverArt *string
	if s.AlbumID != nil && HasCoverArt(*s.AlbumID, conf) {
		coverArt = s.AlbumID
	}

	var year *int
	if s.OriginalDate != nil {
		year = util.ToPtr(s.OriginalDate.Year())
	}

	song := &Song{
		ID:            s.ID,
		IsDir:         false,
		Title:         s.Title,
		Track:         s.Track,
		Year:          year,
		CoverArt:      coverArt,
		Size:          s.Size,
		ContentType:   s.ContentType,
		Suffix:        filepath.Ext(s.Path),
		Duration:      s.Duration.Seconds(),
		BitRate:       s.BitRate,
		SamplingRate:  s.SamplingRate,
		ChannelCount:  s.ChannelCount,
		DiscNumber:    s.Disc,
		Created:       s.Created,
		BPM:           s.BPM,
		MusicBrainzID: s.MusicBrainzID,
		ReplayGain: &ReplayGain{
			TrackGain: s.ReplayGain,
			TrackPeak: s.ReplayGainPeak,
		},
		Type:      "music",
		MediaType: "song",
	}

	if s.SongAlbumInfo != nil {
		song.Album = s.AlbumName
		song.AlbumID = s.AlbumID
		song.Parent = song.AlbumID
		song.ReplayGain.AlbumGain = s.AlbumReplayGain
		song.ReplayGain.AlbumPeak = s.AlbumReplayGainPeak
	}

	if s.SongAnnotations != nil {
		song.Starred = s.Starred
		song.UserRating = s.UserRating
		song.AverageRating = s.AverageRating
	}

	if s.SongPlayInfo != nil {
		song.PlayCount = &s.PlayCount
		song.Played = s.LastPlayed
	}

	if s.SongLists != nil {
		song.Genres = newGenreRefs(s.Genres)
		song.Genre = util.FirstOrNil(s.Genres)

		song.Artists = newArtistRefs(s.Artists)
		song.Artist = util.FirstOrNilMap(s.Artists, func(a repos.ArtistRef) string {
			return a.Name
		})
		song.ArtistID = util.FirstOrNilMap(s.Artists, func(a repos.ArtistRef) string {
			return a.ID
		})

		song.AlbumArtists = newArtistRefs(s.AlbumArtists)
		if len(song.AlbumArtists) > 0 && song.ArtistID == nil && song.Artist == nil {
			song.ArtistID = &song.AlbumArtists[0].ID
			song.Artist = &song.AlbumArtists[0].Name
		}
	}

	fallbackGain := repos.FallbackGain()
	song.ReplayGain.FallbackGain = &fallbackGain
	return song
}

func NewSongs(songs []*repos.CompleteSong, conf config.Config) []*Song {
	return util.Map(songs, func(s *repos.CompleteSong) *Song {
		return NewSong(s, conf)
	})
}
