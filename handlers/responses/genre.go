package responses

import "github.com/juho05/crossonic-server/util"

type Genres []Genre

type Genre struct {
	SongCount  int    `xml:"songCount,attr" json:"songCount"`
	AlbumCount int    `xml:"albumCount,attr" json:"albumCount"`
	Value      string `xml:"value,attr" json:"value"`
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
