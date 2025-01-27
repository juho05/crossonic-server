package responses

import "github.com/juho05/crossonic-server/util"

type Genres struct {
	Genres []*Genre `xml:"genre" json:"genre"`
}

type Genre struct {
	SongCount  int    `xml:"songCount,attr" json:"songCount"`
	AlbumCount int    `xml:"albumCount,attr" json:"albumCount"`
	Value      string `xml:",chardata" json:"value"`
}

type GenreRef struct {
	Name string `xml:"name,attr" json:"name"`
}

func newGenreRefs(genres []string) []*GenreRef {
	return util.Map(genres, func(g string) *GenreRef {
		return &GenreRef{
			Name: g,
		}
	})
}
