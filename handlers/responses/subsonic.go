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
	Scanning bool       `xml:"scanning,attr" json:"scanning"`
	Count    *int       `xml:"count,attr,omitempty" json:"count,omitempty"`
	LastScan *time.Time `xml:"lastScan,attr,omitempty" json:"lastScan,omitempty"`
	// full or quick
	FullScan  bool       `xml:"fullScan,attr" json:"fullScan"`
	StartTime *time.Time `xml:"startTime,attr,omitempty" json:"startTime,omitempty"`
}

type ArtistIndexes struct {
	IgnoredArticles string   `xml:"ignoredArticles,attr" json:"ignoredArticles"`
	LastModified    int64    `xml:"lastModified,attr" json:"lastModified"`
	Index           []*Index `xml:"index" json:"index"`
}

type Index struct {
	Name   string    `xml:"name,attr" json:"name"`
	Artist []*Artist `xml:"artist" json:"artist"`
}

type AlbumList struct {
	Albums []*Album `xml:"album" json:"album"`
}

type AlbumList2 struct {
	Albums []*Album `xml:"album" json:"album"`
}

type RandomSongs struct {
	Songs []*Song `xml:"song" json:"song"`
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

type Lyrics struct {
	Title  string  `xml:"title,attr" json:"title"`
	Artist *string `xml:"artist,attr,omitempty" json:"artist,omitempty"`
	Value  string  `xml:",chardata" json:"value"`
}

type LyricsList struct {
	StructuredLyrics []*StructuredLyrics `xml:"structuredLyrics" json:"structuredLyrics"`
}

type StructuredLyrics struct {
	Lang          string  `xml:"lang,attr" json:"lang"`
	Synced        bool    `xml:"synced,attr" json:"synced"`
	DisplayArtist string  `xml:"displayArtist,attr,omitempty" json:"displayArtist,omitempty"`
	DisplayTitle  string  `xml:"displayTitle,attr,omitempty" json:"displayTitle,omitempty"`
	Offset        int     `xml:"offset,attr,omitempty" json:"offset,omitempty"`
	Line          []*Line `xml:"line" json:"line"`
}

type Line struct {
	Value string `xml:",chardata" json:"value"`
	Start *int   `xml:"start,attr,omitempty" json:"start,omitempty"`
}

type Starred struct {
	Songs   []*Song   `xml:"song" json:"song"`
	Albums  []*Album  `xml:"album" json:"album"`
	Artists []*Artist `xml:"artist" json:"artist"`
}

type Starred2 struct {
	Songs   []*Song   `xml:"song" json:"song"`
	Albums  []*Album  `xml:"album" json:"album"`
	Artists []*Artist `xml:"artist" json:"artist"`
}

type SongsByGenre struct {
	Songs []*Song `xml:"song" json:"song"`
}

type AlbumInfo struct {
	Notes          *string `xml:"notes,omitempty" json:"notes,omitempty"`
	MusicBrainzID  *string `xml:"musicBrainzId,omitempty" json:"musicBrainzId,omitempty"`
	LastFMUrl      *string `xml:"lastFmUrl,omitempty" json:"lastFmUrl,omitempty"`
	SmallImageURL  *string `xml:"smallImageUrl,omitempty" json:"smallImageUrl,omitempty"`
	MediumImageURL *string `xml:"mediumImageUrl,omitempty" json:"mediumImageUrl,omitempty"`
	LargeImageURL  *string `xml:"largeImageUrl,omitempty" json:"largeImageUrl,omitempty"`
}

type ArtistInfo struct {
	Biography      *string `xml:"biography,omitempty" json:"biography,omitempty"`
	MusicBrainzID  *string `xml:"musicBrainzId,omitempty" json:"musicBrainzId,omitempty"`
	LastFMUrl      *string `xml:"lastFmUrl,omitempty" json:"lastFmUrl,omitempty"`
	SmallImageURL  *string `xml:"smallImageUrl,omitempty" json:"smallImageUrl,omitempty"`
	MediumImageURL *string `xml:"mediumImageUrl,omitempty" json:"mediumImageUrl,omitempty"`
	LargeImageURL  *string `xml:"largeImageUrl,omitempty" json:"largeImageUrl,omitempty"`
}

type Directory struct {
	ID            string     `xml:"id,attr" json:"id"`
	Name          string     `xml:"name,attr" json:"name"`
	Parent        *string    `xml:"parent,attr,omitempty" json:"parent,omitempty"`
	Starred       *time.Time `xml:"starred,attr,omitempty" json:"starred,omitempty"`
	UserRating    *int       `xml:"userRating,attr,omitempty" json:"userRating,omitempty"`
	AverageRating *float64   `xml:"averageRating,attr,omitempty" json:"averageRating,omitempty"`
	PlayCount     *int       `xml:"playCount,attr,omitempty" json:"playCount,omitempty"`
	Child         []any      `xml:"child" json:"child"`
}

type InternetRadioStations struct {
	Stations []InternetRadioStation `xml:"internetRadioStation" json:"internetRadioStation"`
}

type InternetRadioStation struct {
	ID          string  `xml:"id,attr" json:"id"`
	Name        string  `xml:"name,attr" json:"name"`
	StreamURL   string  `xml:"streamUrl,attr" json:"streamUrl"`
	HomepageURL *string `xml:"homepageUrl,attr,omitempty" json:"homepageUrl,omitempty"`
}
