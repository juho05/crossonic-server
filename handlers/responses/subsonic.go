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
