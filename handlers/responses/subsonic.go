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
	AlbumCount    *int       `xml:"albumCount,attr,omitempty"         json:"albumCount,omitempty"`
	Starred       *time.Time `xml:"starred,attr,omitempty"       json:"starred,omitempty"`
	MusicBrainzID *string    `xml:"musicBrainzID,omitempty" json:"musicBrainzID,omitempty"`
	UserRating    *int       `xml:"userRating,attr,omitempty"    json:"userRating,omitempty"`
	AverageRating *float64   `xml:"averageRating,attr,omitempty" json:"averageRating,omitempty"`
	Albums        []*Album   `xml:"album,omitempty" json:"album,omitempty"`
}

type AlbumList2 struct {
	Albums []*Album `xml:"album" json:"album"`
}

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
	Genre         *string        `xml:"genre,attr,omitempty"   json:"genre,omitempty"`
	Genres        []*GenreRef    `xml:"genres,omitempty"       json:"genres,omitempty"`
	Year          *int           `xml:"year,attr,omitempty"    json:"year,omitempty"`
	Starred       *time.Time     `xml:"starred,attr,omitempty"         json:"starred,omitempty"`
	UserRating    *int           `xml:"userRating,attr,omitempty"      json:"userRating,omitempty"`
	AverageRating *float64       `xml:"averageRating,attr,omitempty"   json:"averageRating,omitempty"`
	IsDir         bool           `xml:"isDir,attr" json:"isDir"`
	Type          string         `xml:"type,attr" json:"type"`
	MediaType     string         `xml:"mediaType,attr" json:"mediaType"`
	MusicBrainzID *string        `xml:"musicBrainzId,attr,omitempty" json:"musicBrainzId,omitempty"`
	RecordLabels  []*RecordLabel `xml:"recordLabels,omitempty" json:"recordLabels,omitempty"`
	ReleaseTypes  []string       `xml:"releaseTypes,omitempty" json:"releaseTypes,omitempty"`
	IsCompilation *bool          `xml:"isCompilation,attr,omitempty" json:"isCompilation,omitempty"`
}

type ArtistRef struct {
	ID   string `xml:"id,attr" json:"id"`
	Name string `xml:"name,attr" json:"name"`
}

type GenreRef struct {
	Name string `xml:"name,attr" json:"name"`
}

type RandomSongs struct {
	Songs []*Song `xml:"song" json:"song"`
}

type Song struct {
	ID            string       `xml:"id,attr" json:"id"`
	IsDir         bool         `xml:"isDir,attr" json:"isDir"`
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

type ReplayGain struct {
	TrackGain    *float64 `xml:"trackGain,attr,omitempty" json:"trackGain,omitempty"`
	AlbumGain    *float64 `xml:"albumGain,attr,omitempty" json:"albumGain,omitempty"`
	TrackPeak    *float64 `xml:"trackPeak,attr,omitempty" json:"trackPeak,omitempty"`
	AlbumPeak    *float64 `xml:"albumPeak,attr,omitempty" json:"albumPeak,omitempty"`
	BaseGain     *float64 `xml:"baseGain,attr,omitempty" json:"baseGain,omitempty"`
	FallbackGain *float64 `xml:"fallbackGain,attr,omitempty" json:"fallbackGain,omitempty"`
}

type AlbumWithSongs struct {
	*Album
	Songs []*Song `xml:"song" json:"song"`
}

type RecordLabel struct {
	Name string `xml:"name,attr" json:"name"`
}

type NowPlaying struct {
	Entries []*NowPlayingEntry `xml:"entry" json:"entry"`
}

type NowPlayingEntry struct {
	*Song
	Username   string `xml:"username,attr" json:"username"`
	MinutesAgo int    `xml:"minutesAgo,attr" json:"minutesAgo"`
}

type SearchResult3 struct {
	Artists []*Artist `xml:"artist" json:"artist"`
	Albums  []*Album  `xml:"album" json:"album"`
	Songs   []*Song   `xml:"song" json:"song"`
}

type LyricsList struct {
	StructuredLyrics []*StructuredLyrics `xml:"structuredLyrics" json:"structuredLyrics"`
}

type StructuredLyrics struct {
	Lang          string  `xml:"lang" json:"lang"`
	Synced        bool    `xml:"synced" json:"synced"`
	DisplayArtist string  `xml:"displayArtist,omitempty" json:"displayArtist,omitempty"`
	DisplayTitle  string  `xml:"displayTitle,omitempty" json:"displayTitle,omitempty"`
	Offset        int     `xml:"offset,omitempty" json:"offset,omitempty"`
	Line          []*Line `xml:"line" json:"line"`
}

type Line struct {
	Value string `xml:"value" json:"value"`
	Start *int   `xml:"start,omitempty" json:"start,omitempty"`
}

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
	Entry     *[]*Song  `xml:"entry,omitempty" json:"entry,omitempty"`
}
