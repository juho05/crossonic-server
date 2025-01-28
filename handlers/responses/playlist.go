package responses

import (
	"time"

	"github.com/juho05/crossonic-server/repos"
	"github.com/juho05/crossonic-server/util"
)

type Playlists struct {
	Playlists []*Playlist `xml:"playlist" json:"playlist"`
}

type Playlist struct {
	ID        string    `xml:"id,attr" json:"id"`
	Name      string    `xml:"name,attr" json:"name"`
	Comment   *string   `xml:"comment,attr,omitempty" json:"comment,omitempty"`
	Owner     string    `xml:"owner,attr" json:"owner,omitempty"`
	Public    bool      `xml:"public,attr" json:"public"`
	SongCount int       `xml:"songCount,attr" json:"songCount"`
	Duration  int       `xml:"duration,attr" json:"duration"`
	Created   time.Time `xml:"created,attr"  json:"created"`
	Changed   time.Time `xml:"changed,attr"  json:"changed"`
	CoverArt  *string   `xml:"coverArt,attr,omitempty" json:"coverArt,omitempty"`
	Entry     []*Song   `xml:"entry,omitempty" json:"entry,omitempty"`
}

func NewPlaylist(p *repos.CompletePlaylist) *Playlist {
	playlist := &Playlist{
		ID:      p.ID,
		Name:    p.Name,
		Comment: p.Comment,
		Owner:   p.Owner,
		Public:  p.Public,
		Created: p.Created,
		Changed: p.Updated,
	}

	if p.PlaylistTrackInfo != nil {
		playlist.SongCount = p.TrackCount
		playlist.Duration = p.Duration.Seconds()
	}

	if HasCoverArt(p.ID) {
		playlist.CoverArt = &p.ID
	}

	return playlist
}

func NewPlaylists(p []*repos.CompletePlaylist) []*Playlist {
	return util.Map(p, NewPlaylist)
}
