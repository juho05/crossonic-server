package responses

type ListenBrainzConfig struct {
	ListenBrainzUsername *string `xml:"listenBrainzUsername,attr" json:"listenBrainzUsername"`
	SyncFeedback         *bool   `xml:"syncFeedback,attr" json:"syncFeedback"`
	Scrobble             *bool   `xml:"scrobble,attr" json:"scrobble"`
}

type Recap struct {
	TotalDurationMS int64 `xml:"totalDurationMs,attr" json:"totalDurationMs"`
	SongCount       int   `xml:"songCount,attr" json:"songCount"`
	AlbumCount      int   `xml:"albumCount,attr" json:"albumCount"`
	ArtistCount     int   `xml:"artistCount,attr" json:"artistCount"`
}

type TopSongsRecap struct {
	Songs []*TopSongsRecapSong `xml:"song" json:"song"`
}

type TopSongsRecapSong struct {
	*Song
	TotalDurationMS int64 `xml:"totalDurationMs,attr" json:"totalDurationMs"`
}

type AppearsOn struct {
	Albums []*Album `xml:"album" json:"album"`
}

type AlbumVersions struct {
	Albums []*Album `xml:"album" json:"album"`
}

type Songs struct {
	Songs []*Song `xml:"song" json:"song"`
}
