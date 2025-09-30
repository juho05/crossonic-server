package responses

import (
	"encoding/json"
	"encoding/xml"
	"io"
	"net/http"

	"github.com/juho05/crossonic-server"
	"github.com/juho05/log"
)

const (
	apiVersion = "1.16.1"
	xmlns      = "http://subsonic.org/restapi"
)

type status string

const (
	statusOK     = "ok"
	statusFailed = "failed"
)

type Response struct {
	Status           status `xml:"status,attr" json:"status"`
	Version          string `xml:"version,attr" json:"version"`
	XMLNS            string `xml:"xmlns,attr" json:"-"`
	Type             string `xml:"type,attr" json:"type"`
	ServerVersion    string `xml:"serverVersion,attr" json:"serverVersion"`
	OpenSubsonic     bool   `xml:"openSubsonic,attr" json:"openSubsonic"`
	Crossonic        bool   `xml:"crossonic,attr" json:"crossonic"`
	CrossonicVersion string `xml:"crossonicVersion,attr" json:"crossonicVersion"`

	// Subsonic
	Error                  *Error                  `xml:"error,omitempty" json:"error,omitempty"`
	OpenSubsonicExtensions *OpenSubsonicExtensions `xml:"openSubsonicExtensions,omitempty" json:"openSubsonicExtensions,omitempty"`
	License                *License                `xml:"license,omitempty" json:"license,omitempty"`
	ScanStatus             *ScanStatus             `xml:"scanStatus,omitempty" json:"scanStatus,omitempty"`
	Genres                 *Genres                 `xml:"genres,omitempty" json:"genres,omitempty"`
	Directory              *Directory              `xml:"directory,omitempty" json:"directory,omitempty"`
	Indexes                *ArtistIndexes          `xml:"indexes,omitempty" json:"indexes,omitempty"`
	Artists                *ArtistIndexes          `xml:"artists,omitempty" json:"artists,omitempty"`
	AlbumList              *AlbumList              `xml:"albumList,omitempty" json:"albumList,omitempty"`
	AlbumList2             *AlbumList2             `xml:"albumList2,omitempty" json:"albumList2,omitempty"`
	RandomSongs            *RandomSongs            `xml:"randomSongs,omitempty" json:"randomSongs,omitempty"`
	Album                  *AlbumWithSongs         `xml:"album,omitempty" json:"album,omitempty"`
	Artist                 *Artist                 `xml:"artist,omitempty" json:"artist,omitempty"`
	NowPlaying             *NowPlaying             `xml:"nowPlaying,omitempty" json:"nowPlaying,omitempty"`
	SearchResult3          *SearchResult3          `xml:"searchResult3,omitempty" json:"searchResult3,omitempty"`
	Lyrics                 *Lyrics                 `xml:"lyrics,omitempty" json:"lyrics,omitempty"`
	LyricsList             *LyricsList             `xml:"lyricsList,omitempty" json:"lyricsList,omitempty"`
	Playlists              *Playlists              `xml:"playlists,omitempty" json:"playlists,omitempty"`
	Playlist               *Playlist               `xml:"playlist,omitempty" json:"playlist,omitempty"`
	Song                   *Song                   `xml:"song,omitempty" json:"song,omitempty"`
	Starred                *Starred                `xml:"starred,omitempty" json:"starred,omitempty"`
	Starred2               *Starred2               `xml:"starred2,omitempty" json:"starred2,omitempty"`
	SongsByGenre           *SongsByGenre           `xml:"songsByGenre,omitempty" json:"songsByGenre,omitempty"`
	AlbumInfo              *AlbumInfo              `xml:"albumInfo,omitempty" json:"albumInfo,omitempty"`
	ArtistInfo             *ArtistInfo             `xml:"artistInfo,omitempty" json:"artistInfo,omitempty"`
	ArtistInfo2            *ArtistInfo             `xml:"artistInfo2,omitempty" json:"artistInfo2,omitempty"`
	InternetRadioStations  *InternetRadioStations  `xml:"internetRadioStations,omitempty" json:"internetRadioStations,omitempty"`

	// Crossonic
	ListenBrainzConfig *ListenBrainzConfig `xml:"listenBrainzConfig,omitempty" json:"listenBrainzConfig,omitempty"`
	Recap              *Recap              `xml:"recap,omitempty" json:"recap,omitempty"`
	TopSongsRecap      *TopSongsRecap      `xml:"topSongsRecap,omitempty" json:"topSongsRecap,omitempty"`
	AppearsOn          *AppearsOn          `xml:"appearsOn,omitempty" json:"appearsOn,omitempty"`
	Songs              *Songs              `xml:"songs,omitempty" json:"songs,omitempty"`
	AlbumVersions      *AlbumVersions      `xml:"albumVersions,omitempty" json:"albumVersions,omitempty"`
}

func New() Response {
	return Response{
		Status:           statusOK,
		Version:          apiVersion,
		XMLNS:            xmlns,
		Type:             crossonic.ServerName,
		ServerVersion:    crossonic.Version,
		OpenSubsonic:     true,
		Crossonic:        true,
		CrossonicVersion: crossonic.ProtocolVersion,
	}
}

func EncodeError(w io.Writer, format, msg string, code SubsonicError) {
	r := Response{
		Status:           statusFailed,
		Version:          apiVersion,
		XMLNS:            xmlns,
		Type:             crossonic.ServerName,
		ServerVersion:    crossonic.Version,
		OpenSubsonic:     true,
		Crossonic:        true,
		CrossonicVersion: crossonic.ProtocolVersion,
		Error: &Error{
			Code:    code,
			Message: msg,
		},
	}
	r.EncodeOrLog(w, format)
}

func (r Response) EncodeOrLog(w io.Writer, format string) {
	err := r.Encode(w, format)
	if err != nil {
		log.Error(err)
	}
}

func (r Response) Encode(w io.Writer, format string) error {
	type response struct {
		XMLName   xml.Name `xml:"subsonic-response" json:"-"`
		*Response `json:"subsonic-response"`
	}
	rw, isRW := w.(http.ResponseWriter)
	if format == "json" {
		if isRW {
			rw.Header().Set("Content-Type", "application/json; charset=utf-8")
		}
		return json.NewEncoder(w).Encode(response{
			Response: &r,
		})
	}
	if isRW {
		rw.Header().Set("Content-Type", "application/xml; charset=utf-8")
	}
	encoder := xml.NewEncoder(w)
	err := encoder.Encode(response{
		Response: &r,
	})
	if err != nil {
		return err
	}
	return encoder.Close()
}
