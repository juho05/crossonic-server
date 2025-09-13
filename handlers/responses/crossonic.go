package responses

type ListenBrainzConfig struct {
	ListenBrainzUsername *string `xml:"listenBrainzUsername,attr" json:"listenBrainzUsername"`
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
