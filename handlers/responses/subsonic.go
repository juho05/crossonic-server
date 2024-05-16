package responses

import "time"

type Error struct {
	Code    SubsonicError `xml:"code,attr" json:"code"`
	Message string        `xml:"message,attr" json:"message"`
}

type OpenSubsonicExtensions []OpenSubsonicExtension

type OpenSubsonicExtension struct {
	Name     string `xml:"name,attr" json:"name"`
	Versions []int  `xml:"versions" json:"versions"`
}

type License struct {
	Valid bool `xml:"valid,attr" json:"valid"`
}

type ScanStatus struct {
	Scanning bool `xml:"scanning,attr" json:"scanning"`
	Count    *int `xml:"count,attr,omitempty" json:"count,omitempty"`
}

type Genres []Genre

type Genre struct {
	SongCount  int    `xml:"songCount,attr" json:"songCount"`
	AlbumCount int    `xml:"albumCount,attr" json:"albumCount"`
	Value      string `xml:"value,attr" json:"value"`
}

type Artists struct {
	IgnoredArticles string   `xml:"ignoredArticles,attr" json:"ignoredArticles"`
	Index           []*Index `xml:"index" json:"index"`
}

type Index struct {
	Name   string    `xml:"name,attr" json:"name"`
	Artist []*Artist `xml:"artist" json:"artist"`
}

type Artist struct {
	ID            string     `xml:"id,attr,omitempty"       json:"id"`
	Name          string     `xml:"name,attr"               json:"name"`
	CoverArt      *string    `xml:"coverArt,attr,omitempty" json:"coverArt,omitempty"`
	AlbumCount    *int       `xml:"albumCount,attr"         json:"albumCount,omitempty"`
	Starred       *time.Time `xml:"starred,attr,omitempty"       json:"starred,omitempty"`
	MusicBrainzID *string    `xml:"musicBrainzID,omitempty" json:"musicBrainzID,omitempty"`
	UserRating    *int       `xml:"userRating,attr,omitempty"    json:"userRating,omitempty"`
	AverageRating *float64   `xml:"averageRating,attr,omitempty" json:"averageRating,omitempty"`
}

type AlbumList2 struct {
	Albums []*Album `xml:"album" json:"album"`
}

type Album struct {
	ID            string       `xml:"id,attr,omitempty"       json:"id"`
	Created       time.Time    `xml:"created,attr,omitempty"  json:"created,omitempty"`
	ArtistID      *string      `xml:"artistId,attr,omitempty" json:"artistId,omitempty"`
	Artist        *string      `xml:"artist,attr"             json:"artist"`
	Artists       []*ArtistRef `xml:"artists"           json:"artists"`
	DisplayArtist *string      `xml:"diplayArtist,attr,omitempty" json:"displayArtist,omitempty"`
	CoverID       *string      `xml:"coverArt,attr,omitempty" json:"coverArt,omitempty"`
	Title         string       `xml:"title,attr"  json:"title"`
	Name          string       `xml:"name,attr"              json:"name"`
	TrackCount    int          `xml:"songCount,attr"         json:"songCount"`
	Duration      int          `xml:"duration,attr"          json:"duration"`
	PlayCount     *int         `xml:"playCount,attr,omitempty"          json:"playCount,omitempty"`
	Genre         *string      `xml:"genre,attr,omitempty"   json:"genre,omitempty"`
	Genres        []*GenreRef  `xml:"genres,omitempty"       json:"genres,omitempty"`
	Year          *int         `xml:"year,attr,omitempty"    json:"year,omitempty"`
	Starred       *time.Time   `xml:"starred,attr,omitempty"         json:"starred,omitempty"`
	UserRating    *int         `xml:"userRating,attr,omitempty"      json:"userRating,omitempty"`
	AverageRating *float64     `xml:"averageRating,attr,omitempty"   json:"averageRating,omitempty"`
}

type ArtistRef struct {
	ID   string `xml:"id,attr" json:"id"`
	Name string `xml:"name,attr" json:"name"`
}

type GenreRef struct {
	Name string `xml:"name,attr" json:"name"`
}
