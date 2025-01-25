package repos

import (
	"context"
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
	CoverID        *string    `db:"cover_id"`
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
	*SongLists
}

type SongStreamInfo struct {
	Path         string     `db:"path"`
	BitRate      int        `db:"bit_rate"`
	ContentType  string     `db:"content_type"`
	Duration     DurationMS `db:"duration_ms"`
	ChannelCount int        `db:"channel_count"`
}

// params

type IncludeSongInfo struct {
	Album bool

	Annotations    bool
	AnnotationUser string

	Lists bool
}

func IncludeSongInfoBare() IncludeSongInfo {
	return IncludeSongInfo{}
}

func IncludeSongInfoAlbum() IncludeSongInfo {
	return IncludeSongInfo{
		Album: true,
	}
}

func IncludeSongInfoFull(user string) IncludeSongInfo {
	return IncludeSongInfo{
		Album:          true,
		Annotations:    true,
		AnnotationUser: user,
		Lists:          true,
	}
}

type CreateSongParams struct {
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
	CoverID        *string
}

type UpdateSongParams struct {
	Path           Optional[string]
	AlbumID        Optional[*string]
	Title          Optional[string]
	Track          Optional[*int]
	Year           Optional[*int]
	Size           Optional[int64]
	ContentType    Optional[string]
	Duration       Optional[DurationMS]
	BitRate        Optional[int]
	SamplingRate   Optional[int]
	ChannelCount   Optional[int]
	Disc           Optional[*int]
	BPM            Optional[*int]
	MusicBrainzID  Optional[*string]
	ReplayGain     Optional[*float64]
	ReplayGainPeak Optional[*float64]
	Lyrics         Optional[*string]
	CoverID        Optional[*string]
}

type SongFindRandomParams struct {
	FromYear *int
	ToYear   *int
	Genres   []string
	Limit    int
}

type SongFindBySearchParams struct {
	Query  string
	Offset int
	Limit  int
}

type SongSetLBFeedbackUpdatedParams struct {
	SongID string
	MBID   string
}

// repo

type SongRepository interface {
	FindByID(ctx context.Context, id string, include IncludeSongInfo) (*CompleteSong, error)
	FindByIDs(ctx context.Context, ids []string, include IncludeSongInfo) ([]*CompleteSong, error)
	FindByMusicBrainzID(ctx context.Context, mbid string, include IncludeSongInfo) ([]*CompleteSong, error)
	FindByPath(ctx context.Context, path string, include IncludeSongInfo) (*CompleteSong, error)
	FindRandom(ctx context.Context, params SongFindRandomParams, include IncludeSongInfo) ([]*CompleteSong, error)
	FindBySearchQuery(ctx context.Context, params SongFindBySearchParams, include IncludeSongInfo) ([]*CompleteSong, error)

	GetStreamInfo(ctx context.Context, id string) (*SongStreamInfo, error)

	Create(ctx context.Context, params CreateSongParams) (*Song, error)
	Update(ctx context.Context, id string, params UpdateSongParams) error

	DeleteLastUpdatedBefore(ctx context.Context, before time.Time) error

	SetArtists(ctx context.Context, songID string, artistIDs []string) error
	AddArtists(ctx context.Context, songID string, artistIDs []string) error
	RemoveArtists(ctx context.Context, songID string) error

	SetGenres(ctx context.Context, songID string, genres []string) error
	AddGenres(ctx context.Context, songID string, genres []string) error
	RemoveGenres(ctx context.Context, songID string) error

	Star(ctx context.Context, user, songID string) error
	StarMultiple(ctx context.Context, user string, songID []string) (int, error)
	UnStar(ctx context.Context, user, songID string) error

	SetRating(ctx context.Context, user, songID string, rating int) error
	RemoveRating(ctx context.Context, user, songID string) error

	SetLBFeedbackUpdated(ctx context.Context, user string, params []SongSetLBFeedbackUpdatedParams) error
	RemoveLBFeedbackUpdated(ctx context.Context, user string, songIDs []string) error
	FindLBFeedbackUpdatedSongIDsInMBIDListNotStarred(ctx context.Context, user string, mbids []string) ([]string, error)
	DeleteLBFeedbackUpdatedStarsNotInMBIDList(ctx context.Context, user string, mbids []string) (int, error)
	FindNotLBUpdatedSongs(ctx context.Context, user string, include IncludeSongInfo) ([]*CompleteSong, error)

	Count(ctx context.Context) (int, error)
	GetMedianReplayGain(ctx context.Context) (float64, error)
}
