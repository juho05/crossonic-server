package listenbrainz

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/juho05/crossonic-server"
	"github.com/juho05/crossonic-server/config"
	db "github.com/juho05/crossonic-server/db/sqlc"
	"github.com/juho05/log"
)

var (
	ErrUnexpectedResponseCode = errors.New("unexpected response code")
	ErrUnexpectedResponseBody = errors.New("unexpected response body")
	ErrUnauthenticated        = errors.New("unauthenticated")
)

type ListenBrainz struct {
	Store  db.Store
	cancel context.CancelFunc
}

type Listen struct {
	ListenedAt  time.Time
	SongName    string
	AlbumName   *string
	ArtistName  *string
	SongMBID    *string
	AlbumMBID   *string
	ReleaseMBID *string
	ArtistMBIDs []string
	TrackNumber *int
	DurationMS  *int
}

type Connection struct {
	LBUsername string
	Token      string
}

func New(store db.Store) *ListenBrainz {
	return &ListenBrainz{
		Store: store,
	}
}

type additionalInfo struct {
	MediaPlayer             string   `json:"media_player,omitempty"`
	SubmissionClient        string   `json:"submission_client"`
	SubmissionClientVersion string   `json:"submission_client_version"`
	ArtistMBIDs             []string `json:"artist_mbids,omitempty"`
	ReleaseGroupMBID        *string  `json:"release_group_mbid,omitempty"`
	ReleaseMBID             *string  `json:"release_mbid,omitempty"`
	RecordingMBID           *string  `json:"recording_mbid,omitempty"`
	TrackNumber             *int     `json:"tracknumber,omitempty"`
	DurationMS              *int     `json:"duration_ms,omitempty"`
}

type trackMetadata struct {
	ArtistName     *string        `json:"artist_name,omitempty"`
	ReleaseName    *string        `json:"release_name,omitempty"`
	TrackName      string         `json:"track_name"`
	AdditionalInfo additionalInfo `json:"additional_info"`
}

type payload struct {
	ListenedAt    int64         `json:"listened_at,omitempty"`
	TrackMetadata trackMetadata `json:"track_metadata"`
}

type body struct {
	ListenType string    `json:"listen_type"`
	Payload    []payload `json:"payload"`
}

func (l *ListenBrainz) SubmitPlayingNow(ctx context.Context, con Connection, listen *Listen, mediaPlayer string) error {
	_, err := listenBrainzRequest[any](ctx, "/1/submit-listens", http.MethodPost, con.Token, body{
		ListenType: "playing_now",
		Payload: []payload{
			{
				TrackMetadata: trackMetadata{
					ArtistName:  listen.ArtistName,
					ReleaseName: listen.AlbumName,
					TrackName:   listen.SongName,
					AdditionalInfo: additionalInfo{
						MediaPlayer:             mediaPlayer,
						SubmissionClient:        crossonic.ServerName,
						SubmissionClientVersion: crossonic.Version,
						ArtistMBIDs:             listen.ArtistMBIDs,
						ReleaseGroupMBID:        listen.AlbumMBID,
						ReleaseMBID:             listen.ReleaseMBID,
						RecordingMBID:           listen.SongMBID,
						TrackNumber:             listen.TrackNumber,
						DurationMS:              listen.DurationMS,
					},
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("submit playing now: %w", err)
	}
	return nil
}

func (l *ListenBrainz) SubmitSingle(ctx context.Context, con Connection, listen *Listen, mediaPlayer string) error {
	_, err := listenBrainzRequest[any](ctx, "/1/submit-listens", http.MethodPost, con.Token, body{
		ListenType: "single",
		Payload: []payload{
			{
				ListenedAt: listen.ListenedAt.Unix(),
				TrackMetadata: trackMetadata{
					ArtistName:  listen.ArtistName,
					ReleaseName: listen.AlbumName,
					TrackName:   listen.SongName,
					AdditionalInfo: additionalInfo{
						MediaPlayer:             mediaPlayer,
						SubmissionClient:        crossonic.ServerName,
						SubmissionClientVersion: crossonic.Version,
						ArtistMBIDs:             listen.ArtistMBIDs,
						ReleaseGroupMBID:        listen.AlbumMBID,
						ReleaseMBID:             listen.ReleaseMBID,
						RecordingMBID:           listen.SongMBID,
						TrackNumber:             listen.TrackNumber,
						DurationMS:              listen.DurationMS,
					},
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("submit playing now: %w", err)
	}
	return nil
}

func (l *ListenBrainz) SubmitImport(ctx context.Context, con Connection, listens []*Listen, mediaPlayer string) error {
	payloads := make([]payload, 0, len(listens))
	for _, listen := range listens {
		payloads = append(payloads, payload{
			ListenedAt: listen.ListenedAt.Unix(),
			TrackMetadata: trackMetadata{
				ArtistName:  listen.ArtistName,
				ReleaseName: listen.AlbumName,
				TrackName:   listen.SongName,
				AdditionalInfo: additionalInfo{
					MediaPlayer:             mediaPlayer,
					SubmissionClient:        crossonic.ServerName,
					SubmissionClientVersion: crossonic.Version,
					ArtistMBIDs:             listen.ArtistMBIDs,
					ReleaseGroupMBID:        listen.AlbumMBID,
					ReleaseMBID:             listen.ReleaseMBID,
					RecordingMBID:           listen.SongMBID,
					TrackNumber:             listen.TrackNumber,
					DurationMS:              listen.DurationMS,
				},
			},
		})
	}
	for i := 0; i < len(payloads); {
		next := i + min(1000, len(payloads))
		_, err := listenBrainzRequest[any](ctx, "/1/submit-listens", http.MethodPost, con.Token, body{
			ListenType: "import",
			Payload:    payloads[i:next],
		})
		if err != nil {
			return fmt.Errorf("submit playing now: %w", err)
		}
		i = next
	}
	return nil
}

func (l *ListenBrainz) CheckToken(ctx context.Context, token string) (Connection, error) {
	type response struct {
		Code     int    `json:"code"`
		Message  string `json:"message"`
		Valid    bool   `json:"valid"`
		UserName string `json:"user_name"`
	}
	res, err := listenBrainzRequest[response](ctx, "/1/validate-token", http.MethodGet, token, nil)
	if err != nil {
		return Connection{}, fmt.Errorf("check listenbrainz token: %s", err)
	}
	if !res.Valid {
		return Connection{}, fmt.Errorf("check listenbrainz token: %w: %s", ErrUnauthenticated, res.Message)
	}
	return Connection{
		LBUsername: res.UserName,
		Token:      token,
	}, nil
}

func (l *ListenBrainz) GetListenbrainzConnection(ctx context.Context, user string) (Connection, error) {
	u, err := l.Store.FindUser(ctx, user)
	if err != nil {
		return Connection{}, fmt.Errorf("get listenbrainz token: %w", err)
	}
	if u.EncryptedListenbrainzToken == nil || u.ListenbrainzUsername == nil {
		return Connection{}, ErrUnauthenticated
	}
	token, err := db.DecryptPassword(u.EncryptedListenbrainzToken)
	if err != nil {
		return Connection{}, fmt.Errorf("get listenbrainz token: %w", err)
	}
	return Connection{
		LBUsername: *u.ListenbrainzUsername,
		Token:      token,
	}, nil
}

func (l *ListenBrainz) SubmitMissingListens(ctx context.Context) error {
	scrobbles, err := l.Store.FindUnsubmittedLBScrobbles(ctx)
	if err != nil {
		return fmt.Errorf("submit missing listenbrainz scrobbles: find unsubmitted scrobbles: %w", err)
	}
	if len(scrobbles) == 0 {
		log.Trace("no unsubmitted scrobbles")
		return nil
	}
	ids := make([]string, len(scrobbles))
	for i, s := range scrobbles {
		ids[i] = s.SongID
	}
	songs, err := l.Store.FindSongs(ctx, ids)
	if err != nil {
		return fmt.Errorf("submit missing listenbrainz scrobbles: find songs: %w", err)
	}
	type song struct {
		ID          string
		Name        string
		MBID        *string
		AlbumName   *string
		AlbumMBID   *string
		ReleaseMBID *string
		ArtistName  *string
		ArtistMBIDs []string
		TrackNumber *int
		DurationMS  int
	}
	songMap := make(map[string]*song, len(songs))
	for _, s := range songs {
		songMap[s.ID] = &song{
			ID:          s.ID,
			Name:        s.Title,
			MBID:        s.MusicBrainzID,
			AlbumName:   s.AlbumName,
			AlbumMBID:   s.AlbumMusicBrainzID,
			ReleaseMBID: s.AlbumReleaseMbid,
			TrackNumber: int32PtrToIntPtr(s.Track),
			DurationMS:  int(s.DurationMs),
		}
	}
	artists, err := l.Store.FindArtistRefsBySongs(ctx, ids)
	if err != nil {
		return fmt.Errorf("submit missing listenbrainz scrobbles: find artist refs: %s", err)
	}
	for _, a := range artists {
		song := songMap[a.SongID]
		if song.ArtistName == nil {
			song.ArtistName = &a.Name
		}
		if a.MusicBrainzID != nil {
			song.ArtistMBIDs = append(song.ArtistMBIDs, *a.MusicBrainzID)
		}
	}
	listensPerUser := make(map[string][]*Listen, len(scrobbles))
	for _, s := range scrobbles {
		if _, ok := listensPerUser[s.UserName]; !ok {
			listensPerUser[s.UserName] = make([]*Listen, 0, 10)
		}
		song := songMap[s.SongID]
		listensPerUser[s.UserName] = append(listensPerUser[s.UserName], &Listen{
			ListenedAt:  s.Time.Time,
			SongName:    song.Name,
			AlbumName:   song.AlbumName,
			ArtistName:  song.ArtistName,
			SongMBID:    song.MBID,
			AlbumMBID:   song.AlbumMBID,
			ReleaseMBID: song.ReleaseMBID,
			ArtistMBIDs: song.ArtistMBIDs,
			TrackNumber: song.TrackNumber,
			DurationMS:  &song.DurationMS,
		})
	}

	successfulUsernames := make([]string, 0, len(listensPerUser))
	var count int
	for user, listens := range listensPerUser {
		con, err := l.GetListenbrainzConnection(ctx, user)
		if err != nil {
			return fmt.Errorf("submit missing listenbrainz scrobbles: get listenbrainz connection for user %s: %w", user, err)
		}
		err = l.SubmitImport(ctx, con, listens, "")
		if err != nil {
			return fmt.Errorf("submit missing listenbrainz scrobbles: submit: %w", err)
		}
		successfulUsernames = append(successfulUsernames, user)
		count += len(listens)
	}
	err = l.Store.SetLBSubmittedByUsers(context.Background(), successfulUsernames)
	if err != nil {
		return fmt.Errorf("submit missing listenbrainz scrobbles: set submitted status in db: %w", err)
	}

	log.Tracef("Submitted %d/%d unsubmitted listens to ListenBrainz", count, len(scrobbles))
	return nil
}

func (l *ListenBrainz) StartPeriodicallySubmittingListens(period time.Duration) {
	go func() {
		ctx, cancel := context.WithCancel(context.Background())
		l.cancel = cancel
	loop:
		for {
			select {
			case <-ctx.Done():
				break loop
			default:
			}

			err := l.SubmitMissingListens(ctx)
			if err != nil {
				log.Error(err)
			}

			select {
			case <-ctx.Done():
				break loop
			default:
			}
			time.Sleep(period)
		}
	}()
}

func listenBrainzRequest[T any](ctx context.Context, endpoint, method, token string, body any) (T, error) {
	var obj T
	var data io.Reader
	if body != nil {
		d, err := json.Marshal(body)
		if err != nil {
			return obj, fmt.Errorf("listenbrainz request: %w", err)
		}
		data = bytes.NewBuffer(d)
	}
	url := config.ListenBrainzURL() + endpoint
	req, err := http.NewRequestWithContext(ctx, method, url, data)
	if err != nil {
		return obj, fmt.Errorf("listenbrainz request: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Token %s", token))
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return obj, fmt.Errorf("listenbrainz request: %w", err)
	}
	if res.StatusCode == http.StatusTooManyRequests {
		res.Body.Close()
		secondsStr := res.Header.Get("X-RateLimit-Reset-In")
		if secondsStr == "" {
			log.Error("missing X-RateLimit-Reset-In in 429 ListenBrainz response")
			secondsStr = "1"
		}
		seconds, err := strconv.Atoi(secondsStr)
		if err != nil {
			log.Errorf("invalid value of X-RateLimit-Reset-In in 429 ListenBrainz response: %s", secondsStr)
			seconds = 1
		}
		time.Sleep(time.Duration(seconds)*time.Second + 500*time.Millisecond)
		return listenBrainzRequest[T](ctx, endpoint, method, token, body)
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusUnauthorized {
		return obj, fmt.Errorf("listenbrainz request: %w", ErrUnauthenticated)
	}
	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusCreated {
		return obj, fmt.Errorf("listenbrainz request: %w: %d", ErrUnexpectedResponseCode, res.StatusCode)
	}
	err = json.NewDecoder(res.Body).Decode(&obj)
	if err != nil {
		return obj, fmt.Errorf("listenbrainz request: %w: %w", ErrUnexpectedResponseBody, err)
	}
	return obj, nil
}

func (l *ListenBrainz) Close() error {
	l.cancel()
	return nil
}

func int32PtrToIntPtr(ptr *int32) *int {
	if ptr == nil {
		return nil
	}
	v32 := *ptr
	v := int(v32)
	return &v
}
