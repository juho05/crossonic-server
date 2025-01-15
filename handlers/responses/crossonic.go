package responses

type ListenBrainzConfig struct {
	ListenBrainzUsername *string `xml:"listenBrainzUsername,attr" json:"listenBrainzUsername"`
}

type Recap struct {
	TotalDurationMS int64 `xml:"totalDurationMs" json:"totalDurationMs"`
	SongCount       int64 `xml:"songCount" json:"songCount"`
	AlbumCount      int64 `xml:"albumCount" json:"albumCount"`
	ArtistCount     int64 `xml:"artistCount" json:"artistCount"`
}

type TopSongsRecap struct {
	Songs []*TopSongsRecapSong `xml:"song" json:"song"`
}

type TopSongsRecapSong struct {
	*Song
	TotalDurationMS int64 `xml:"totalDurationMs" json:"totalDurationMs"`
}
