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

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/juho05/crossonic-server"
	"github.com/juho05/crossonic-server/config"
	"github.com/juho05/crossonic-server/db/sqlc"
	"github.com/juho05/log"
)

var (
	ErrUnexpectedResponseCode = errors.New("unexpected response code")
	ErrUnexpectedResponseBody = errors.New("unexpected response body")
	ErrUnauthenticated        = errors.New("unauthenticated")
	ErrNotEnoughMetadata      = errors.New("now enough metadata")
	ErrNotFound               = errors.New("not found")
)

type ListenBrainz struct {
	Store  sqlc.Store
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

type FeedbackScore int

const (
	FeedbackScoreLove FeedbackScore = 1
	FeedbackScoreNone FeedbackScore = 0
	FeedbackScoreHate FeedbackScore = -1
)

type Feedback struct {
	SongID     string
	SongName   string
	SongMBID   *string
	AlbumName  *string
	ArtistName *string

	Score FeedbackScore
}

type Connection struct {
	User       string
	LBUsername string
	Token      string
}

func New(store sqlc.Store) *ListenBrainz {
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
	token, err := sqlc.DecryptPassword(u.EncryptedListenbrainzToken)
	if err != nil {
		return Connection{}, fmt.Errorf("get listenbrainz token: %w", err)
	}
	return Connection{
		User:       user,
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

func (l *ListenBrainz) StartPeriodicSync(period time.Duration) {
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

			err = l.SyncSongFeedback(ctx)
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

func (l *ListenBrainz) UpdateSongFeedback(ctx context.Context, con Connection, feedback []*Feedback) (int, error) {
	var missingMBID []*Feedback
	var missingMBIDIndices []int
	for i, f := range feedback {
		if f.SongMBID == nil {
			if f.ArtistName == nil {
				log.Warnf("listenbrainz: update song feedback: not enough metadata for: %s (%s)", f.SongName, f.SongID)
				continue
			}
			missingMBID = append(missingMBID, f)
			missingMBIDIndices = append(missingMBIDIndices, i)
		}
	}

	if len(missingMBID) > 0 {
		type response struct {
			Index            int     `json:"index"`
			RecordingMBID    *string `json:"recording_mbid"`
			RecordingNameArg string  `json:"recording_name_arg"`
			ArtistNameArg    string  `json:"artist_name_arg"`
		}
		type recording struct {
			RecordingName string  `json:"recording_name"`
			ArtistName    string  `json:"artist_name"`
			ReleaseName   *string `json:"release_name,omitempty"`
		}
		type request struct {
			Recordings []recording `json:"recordings"`
		}
		req := request{
			Recordings: make([]recording, 0, len(missingMBID)),
		}
		for _, f := range missingMBID {
			req.Recordings = append(req.Recordings, recording{
				RecordingName: f.SongName,
				ArtistName:    *f.ArtistName,
				ReleaseName:   f.AlbumName,
			})
		}
		res, err := listenBrainzRequest[[]response](ctx, "/1/metadata/lookup", http.MethodPost, con.Token, req)
		if err != nil {
			return 0, fmt.Errorf("update song favorites: %w", err)
		}
		for _, r := range res {
			if r.RecordingMBID != nil {
				feedback[missingMBIDIndices[r.Index]].SongMBID = r.RecordingMBID
			} else {
				log.Warnf("listenbrainz: update song feedback: not found: %s by %s", r.RecordingNameArg, r.ArtistNameArg)
			}
		}
	}

	type request struct {
		RecordingMBID string        `json:"recording_mbid"`
		Score         FeedbackScore `json:"score"`
	}
	successSongs := make([]sqlc.SetLBFeedbackUpdatedParams, 0, len(feedback))
	for _, f := range feedback {
		if f.SongMBID == nil {
			continue
		}
		_, err := listenBrainzRequest[any](ctx, "/1/feedback/recording-feedback", http.MethodPost, con.Token, request{
			RecordingMBID: *f.SongMBID,
			Score:         f.Score,
		})
		if err != nil {
			log.Errorf("listenbrainz: update song feedback: %s", err)
			continue
		}
		successSongs = append(successSongs, sqlc.SetLBFeedbackUpdatedParams{
			SongID:   f.SongID,
			UserName: con.User,
			Mbid:     *f.SongMBID,
		})
	}

	_, err := l.Store.SetLBFeedbackUpdated(ctx, successSongs)
	if err != nil {
		return 0, fmt.Errorf("listenbrainz: update song feedback: update lb_feedback_updated: %w", err)
	}
	return len(successSongs), nil
}

func (l *ListenBrainz) SyncSongFeedback(ctx context.Context) error {
	users, err := l.Store.FindUsers(ctx)
	if err != nil {
		return fmt.Errorf("sync song feedback: %w", err)
	}
	var deletedLocal int
	var createdLocal int
	var uploadedLocal int
	for _, u := range users {
		if u.ListenbrainzUsername == nil {
			continue
		}
		token, err := sqlc.DecryptPassword(u.EncryptedListenbrainzToken)
		if err != nil {
			log.Errorf("listenbrainz: sync song feedback: decrypt listenbrainz token: %s", err)
			continue
		}

		// upload non-uploaded feedback to ListenBrainz
		songs, err := l.Store.FindNotLBUpdatedSongs(ctx, u.Name)
		if err != nil {
			log.Errorf("listenbrainz: sync song feedback: find not lb updated songs: %s", err)
			continue
		}
		notUpdatedSongIDs := make([]string, 0, len(songs))
		songMap := make(map[string]*Feedback, len(songs))
		feedbackList := make([]*Feedback, 0, len(songs))
		for _, s := range songs {
			score := FeedbackScoreNone
			if s.Starred.Valid {
				score = FeedbackScoreLove
			}
			feedback := &Feedback{
				SongID:    s.ID,
				SongName:  s.Title,
				SongMBID:  s.MusicBrainzID,
				AlbumName: s.AlbumName,
				Score:     score,
			}
			feedbackList = append(feedbackList, feedback)
			notUpdatedSongIDs = append(notUpdatedSongIDs, s.ID)
		}

		if len(songMap) > 0 {
			artists, err := l.Store.FindArtistRefsBySongs(ctx, notUpdatedSongIDs)
			if err != nil {
				log.Errorf("listenbrainz: sync song feedback: find artists for not lb updated songs: %s", err)
				continue
			}
			for _, a := range artists {
				song := songMap[a.SongID]
				if song.ArtistName == nil {
					song.ArtistName = &a.Name
				}
			}
		}

		uploadCount, err := l.UpdateSongFeedback(ctx, Connection{
			User:       u.Name,
			LBUsername: *u.ListenbrainzUsername,
			Token:      token,
		}, feedbackList)
		if err != nil {
			log.Errorf("listenbrainz: sync song feedback: upload not updated feedback: %s", err)
			continue
		}
		uploadedLocal += uploadCount

		lbFeedback, err := l.collectListenbrainzLoveFeedback(ctx, *u.ListenbrainzUsername, token)
		if err != nil {
			log.Errorf("listenbrainz: sync song feedback: collect listenbrainz love feedback: %s", err)
			continue
		}
		lbFeedbackMBIDs := make([]string, 0, len(lbFeedback))
		for _, f := range lbFeedback {
			lbFeedbackMBIDs = append(lbFeedbackMBIDs, f.RecordingMBID)
		}

		tx, err := l.Store.BeginTransaction(ctx)
		if err != nil {
			log.Errorf("listenbrainz: sync song feedback: begin transaction: %s", err)
			continue
		}

		// delete local uploaded feedback that is not present on ListenBrainz
		result, err := tx.DeleteLBFeedbackUpdatedStarsNotInMBIDList(ctx, sqlc.DeleteLBFeedbackUpdatedStarsNotInMBIDListParams{
			UserName:  u.Name,
			SongMbids: lbFeedbackMBIDs,
		})
		if err != nil {
			log.Errorf("listenbrainz: sync song feedback: delete stars when not in list of love feedback: %s", err)
			tx.Rollback(ctx)
			continue
		}
		deletedLocal += int(result.RowsAffected())

		// update local uploaded feedback that is present on ListenBrainz
		songIDs, err := tx.FindLBFeedbackUpdatedSongIDsInMBIDListNotStarred(ctx, sqlc.FindLBFeedbackUpdatedSongIDsInMBIDListNotStarredParams{
			UserName:  u.Name,
			SongMbids: lbFeedbackMBIDs,
		})
		if err != nil {
			log.Errorf("listenbrainz: sync song feedback: find non-starred songs in list of love feedback: %s", err)
			tx.Rollback(ctx)
			continue
		}
		stars := make([]sqlc.StarSongsParams, 0, len(songIDs))
		for _, s := range songIDs {
			stars = append(stars, sqlc.StarSongsParams{
				SongID:   s,
				UserName: u.Name,
				Created: pgtype.Timestamptz{
					Time:  time.Now(),
					Valid: true,
				},
			})
		}
		newStarCount, err := tx.StarSongs(ctx, stars)
		if err != nil {
			log.Errorf("listenbrainz: sync song feedback: star non-starred songs in list of love feedback: %s", err)
			tx.Rollback(ctx)
			continue
		}
		createdLocal += int(newStarCount)

		err = tx.Commit(ctx)
		if err != nil {
			log.Errorf("listenbrainz: sync song feedback: commit transaction: %s", err)
			tx.Rollback(ctx)
			continue
		}
	}

	log.Tracef("Synced listenbrainz stars (uploaded: %d; created: %d; deleted: %d)", uploadedLocal, createdLocal, deletedLocal)
	return nil
}

type loveFeedback struct {
	RecordingMBID string `json:"recording_mbid"`
	TrackMetadata struct {
		ArtistName  *string `json:"artist_name"`
		ReleaseName *string `json:"release_name"`
		TrackName   *string `json:"track_name"`
		MBIDMapping struct {
			ArtistMBIDs []string `json:"artist_mbids"`
			ReleaseMBID *string  `json:"release_mbid"`
		} `json:"mbid_mapping"`
	} `json:"track_metadata"`
}

func (l *ListenBrainz) collectListenbrainzLoveFeedback(ctx context.Context, lbUserName, token string) (map[string]loveFeedback, error) {
	feedback := make(map[string]loveFeedback)
	totalCount := -1
	pageSize := 1000
	type response struct {
		Feedback   []loveFeedback `json:"feedback"`
		TotalCount int            `json:"total_count"`
	}
	for page := 0; totalCount == -1 || page*pageSize < totalCount; page++ {
		res, err := listenBrainzRequest[response](ctx, fmt.Sprintf("/1/feedback/user/%s/get-feedback?metadata=true&score=1&count=%d&offset=%d", lbUserName, pageSize, page*pageSize), http.MethodGet, token, nil)
		if err != nil {
			return nil, fmt.Errorf("collect listenbrainz love feedback: %w", err)
		}
		totalCount = res.TotalCount
		for _, f := range res.Feedback {
			if f.RecordingMBID != "" {
				feedback[f.RecordingMBID] = f
			}
		}
	}
	return feedback, nil
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
