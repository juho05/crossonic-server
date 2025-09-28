package repos

import (
	"context"
	"slices"
	"time"
)

// models

type Song struct {
	ID             string     `db:"id"`
	Path           string     `db:"path"`
	AlbumID        *string    `db:"album_id"`
	Title          string     `db:"title"`
	Track          *int       `db:"track"`
	Year           *int       `db:"year"`
	Size           int64      `db:"size"`
	ContentType    string     `db:"content_type"`
	Duration       DurationMS `db:"duration_ms"`
	BitRate        int        `db:"bit_rate"`
	SamplingRate   int        `db:"sampling_rate"`
	ChannelCount   int        `db:"channel_count"`
	Disc           *int       `db:"disc_number"`
	Created        time.Time  `db:"created"`
	Updated        time.Time  `db:"updated"`
	BPM            *int       `db:"bpm"`
	MusicBrainzID  *string    `db:"music_brainz_id"`
	ReplayGain     *float64   `db:"replay_gain"`
	ReplayGainPeak *float64   `db:"replay_gain_peak"`
	Lyrics         *string    `db:"lyrics"`
}

type SongAlbumInfo struct {
	AlbumName           *string  `db:"album_name"`
	AlbumReplayGain     *float64 `db:"album_replay_gain"`
	AlbumReplayGainPeak *float64 `db:"album_replay_gain_peak"`
	AlbumMusicBrainzID  *string  `db:"album_music_brainz_id"`
	AlbumReleaseMBID    *string  `db:"album_release_mbid"`
}

type SongAnnotations struct {
	Starred       *time.Time `db:"starred"`
	UserRating    *int       `db:"user_rating"`
	AverageRating *float64   `db:"avg_rating"`
}

type SongPlayInfo struct {
	PlayCount  int        `db:"play_count"`
	LastPlayed *time.Time `db:"last_played"`
}

type SongLists struct {
	Genres       []string    `db:"-"`
	Artists      []ArtistRef `db:"-"`
	AlbumArtists []ArtistRef `db:"-"`
}

type ArtistRef struct {
	ID            string  `db:"id"`
	Name          string  `db:"name"`
	MusicBrainzID *string `db:"music_brainz_id"`
}

type CompleteSong struct {
	Song
	*SongAlbumInfo
	*SongAnnotations
	*SongPlayInfo
	*SongLists
}

type SongStreamInfo struct {
	Path         string     `db:"path"`
	BitRate      int        `db:"bit_rate"`
	ContentType  string     `db:"content_type"`
	Duration     DurationMS `db:"duration_ms"`
	ChannelCount int        `db:"channel_count"`
}

type SongArtistConnection struct {
	SongID   string `db:"song_id"`
	ArtistID string `db:"artist_id"`
	Index    int    `db:"index"`
}

type SongGenreConnection struct {
	SongID string `db:"song_id"`
	Genre  string `db:"genre_name"`
}

// params

type IncludeSongInfo struct {
	Album bool

	User        string
	Annotations bool
	PlayInfo    bool

	Lists bool
}

func IncludeSongInfoBare() IncludeSongInfo {
	return IncludeSongInfo{}
}

func IncludeSongInfoFull(user string) IncludeSongInfo {
	return IncludeSongInfo{
		Album:       true,
		User:        user,
		Annotations: true,
		PlayInfo:    true,
		Lists:       true,
	}
}

type CreateSongParams struct {
	ID             *string
	Path           string
	AlbumID        *string
	Title          string
	Track          *int
	Year           *int
	Size           int64
	ContentType    string
	Duration       DurationMS
	BitRate        int
	SamplingRate   int
	ChannelCount   int
	Disc           *int
	BPM            *int
	MusicBrainzID  *string
	ReplayGain     *float64
	ReplayGainPeak *float64
	Lyrics         *string
	AlbumName      *string
	ArtistNames    []string
}

type UpdateSongAllParams struct {
	ID             string
	Path           string
	AlbumID        *string
	Title          string
	Track          *int
	Year           *int
	Size           int64
	ContentType    string
	Duration       DurationMS
	BitRate        int
	SamplingRate   int
	ChannelCount   int
	Disc           *int
	BPM            *int
	MusicBrainzID  *string
	ReplayGain     *float64
	ReplayGainPeak *float64
	Lyrics         *string
	AlbumName      *string
	ArtistNames    []string
}

type SongOrder string

const (
	SongOrderTitle       SongOrder = "title"
	SongOrderRandom      SongOrder = "random"
	SongOrderReleaseDate SongOrder = "release"
	SongOrderAdded       SongOrder = "added"
	SongOrderLastPlayed  SongOrder = "lastPlayed"
	SongOrderPlayCount   SongOrder = "playCount"
	SongOrderStarred     SongOrder = "starred"
	SongOrderBPM         SongOrder = "bpm"
)

func (s SongOrder) Valid() bool {
	return slices.Contains([]SongOrder{
		SongOrderTitle,
		SongOrderRandom,
		SongOrderReleaseDate,
		SongOrderAdded,
		SongOrderLastPlayed,
		SongOrderPlayCount,
		SongOrderStarred,
		SongOrderBPM,
	}, s)
}

type SongFindAllFilter struct {
	Search string

	OnlyStarred bool

	MinBPM *int
	MaxBPM *int

	FromYear *int
	ToYear   *int

	Genres []string

	ArtistIDs []string
	AlbumIDs  []string

	Order      *SongOrder
	OrderDesc  bool
	RandomSeed *string

	Paginate Paginate
}

type SongSetLBFeedbackUploadedParams struct {
	SongID     string
	RemoteMBID *string
	Uploaded   bool
}

// repo

type SongRepository interface {
	FindByID(ctx context.Context, id string, include IncludeSongInfo) (*CompleteSong, error)
	FindByIDs(ctx context.Context, ids []string, include IncludeSongInfo) ([]*CompleteSong, error)
	FindAllFiltered(ctx context.Context, filter SongFindAllFilter, include IncludeSongInfo) ([]*CompleteSong, error)
	FindByMusicBrainzID(ctx context.Context, mbid string, include IncludeSongInfo) ([]*CompleteSong, error)
	FindByPath(ctx context.Context, path string, include IncludeSongInfo) (*CompleteSong, error)
	FindByTitle(ctx context.Context, title string, include IncludeSongInfo) ([]*CompleteSong, error)
	FindAllByPathOrMBID(ctx context.Context, paths []string, mbids []string, include IncludeSongInfo) ([]*CompleteSong, error)
	FindNonExistentIDs(ctx context.Context, ids []string) ([]string, error)

	FindPaths(ctx context.Context, updatedBefore time.Time, paginate Paginate) ([]string, error)
	DeleteByPaths(ctx context.Context, paths []string) error

	GetStreamInfo(ctx context.Context, id string) (*SongStreamInfo, error)

	CreateAll(ctx context.Context, params []CreateSongParams) error
	TryUpdateAll(ctx context.Context, params []UpdateSongAllParams) (int, error)

	DeleteLastUpdatedBefore(ctx context.Context, before time.Time) error

	DeleteArtistConnections(ctx context.Context, songIDs []string) error
	CreateArtistConnections(ctx context.Context, connections []SongArtistConnection) error

	DeleteGenreConnections(ctx context.Context, songIDs []string) error
	CreateGenreConnections(ctx context.Context, connections []SongGenreConnection) error

	Star(ctx context.Context, user, songID string) error
	StarMultiple(ctx context.Context, user string, songID []string) (int, error)
	UnStar(ctx context.Context, user, songID string) error
	UnStarMultiple(ctx context.Context, user string, songID []string) (int, error)

	SetRating(ctx context.Context, user, songID string, rating int) error
	RemoveRating(ctx context.Context, user, songID string) error

	FindNotUploadedLBFeedback(ctx context.Context, user string, lbLovedMBIDs []string, include IncludeSongInfo) ([]*CompleteSong, error)
	FindLocalOutdatedFeedbackByLB(ctx context.Context, user string, lbLovedMBIDs []string, include IncludeSongInfo) ([]*CompleteSong, error)
	SetLBFeedbackUploaded(ctx context.Context, user string, params []SongSetLBFeedbackUploadedParams, updateRemoteMBIDs bool) error
	SetLBFeedbackUploadedForAllMatchingStarredSongs(ctx context.Context, user string, lbLovedMBIDs []string) error

	Count(ctx context.Context) (int, error)
	GetMedianReplayGain(ctx context.Context) (float64, error)
}
