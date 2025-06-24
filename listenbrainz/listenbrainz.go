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
	"github.com/juho05/crossonic-server/repos"
	"github.com/juho05/crossonic-server/util"
	"github.com/juho05/log"
)

var (
	ErrUnexpectedResponseCode = errors.New("unexpected response code")
	ErrUnexpectedResponseBody = errors.New("unexpected response body")
	ErrUnauthenticated        = errors.New("unauthenticated")
)

type ListenBrainz struct {
	db     repos.DB
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
	Duration    *repos.DurationMS
}

type FeedbackScore int

//goland:noinspection GoUnusedConst
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

func New(db repos.DB) *ListenBrainz {
	return &ListenBrainz{
		db: db,
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
	DurationMS              *int64   `json:"duration_ms,omitempty"`
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
	durationMs := listen.Duration.ToStd().Milliseconds()
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
						DurationMS:              &durationMs,
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
	durationMs := listen.Duration.ToStd().Milliseconds()
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
						DurationMS:              &durationMs,
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
		durationMs := listen.Duration.ToStd().Milliseconds()
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
					DurationMS:              &durationMs,
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
	u, err := l.db.User().FindByName(ctx, user)
	if err != nil {
		return Connection{}, fmt.Errorf("get listenbrainz token: %w", err)
	}
	if u.EncryptedListenBrainzToken == nil || u.ListenBrainzUsername == nil {
		return Connection{}, ErrUnauthenticated
	}
	token, err := repos.DecryptPassword(u.EncryptedListenBrainzToken)
	if err != nil {
		return Connection{}, fmt.Errorf("get listenbrainz token: %w", err)
	}
	return Connection{
		User:       user,
		LBUsername: *u.ListenBrainzUsername,
		Token:      token,
	}, nil
}

func (l *ListenBrainz) SubmitMissingListens(ctx context.Context) error {
	scrobbles, err := l.db.Scrobble().FindUnsubmittedLBScrobbles(ctx)
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
	songs, err := l.db.Song().FindByIDs(ctx, ids, repos.IncludeSongInfo{
		Album: true,
		Lists: true,
	})
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
		DurationMS  int64
	}
	songMap := make(map[string]*song, len(songs))
	for _, s := range songs {
		var artistName *string
		if len(s.Artists) > 0 {
			artistName = &s.Artists[0].Name
		}
		artistMBIDs := make([]string, 0, len(s.Artists))
		for _, a := range s.Artists {
			if a.MusicBrainzID != nil {
				artistMBIDs = append(artistMBIDs, *a.MusicBrainzID)
			}
		}
		songMap[s.ID] = &song{
			ID:          s.ID,
			Name:        s.Title,
			MBID:        s.MusicBrainzID,
			AlbumName:   s.AlbumName,
			AlbumMBID:   s.AlbumMusicBrainzID,
			ReleaseMBID: s.AlbumReleaseMBID,
			TrackNumber: s.Track,
			DurationMS:  s.Duration.ToStd().Milliseconds(),
			ArtistName:  artistName,
			ArtistMBIDs: artistMBIDs,
		}
	}
	listensPerUser := make(map[string][]*Listen, len(scrobbles))
	for _, s := range scrobbles {
		if _, ok := listensPerUser[s.User]; !ok {
			listensPerUser[s.User] = make([]*Listen, 0, 10)
		}
		song := songMap[s.SongID]
		duration := repos.NewDurationMS(song.DurationMS)
		listensPerUser[s.User] = append(listensPerUser[s.User], &Listen{
			ListenedAt:  s.Time,
			SongName:    song.Name,
			AlbumName:   song.AlbumName,
			ArtistName:  song.ArtistName,
			SongMBID:    song.MBID,
			AlbumMBID:   song.AlbumMBID,
			ReleaseMBID: song.ReleaseMBID,
			ArtistMBIDs: song.ArtistMBIDs,
			TrackNumber: song.TrackNumber,
			Duration:    &duration,
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
	err = l.db.Scrobble().SetLBSubmittedByUsers(context.Background(), successfulUsernames)
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

// https://github.com/metabrainz/listenbrainz-server/blob/2dafb0da41c327d2831e5130086243b4dc5035c9/listenbrainz/webserver/views/metadata_api.py#L25
const maxLookupsPerPost = 50

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
		for i := 0; i < len(missingMBID); i += maxLookupsPerPost {
			mbids := missingMBID[i:min(len(missingMBID), i+maxLookupsPerPost)]
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
				Recordings: make([]recording, 0, len(mbids)),
			}
			for _, f := range mbids {
				req.Recordings = append(req.Recordings, recording{
					RecordingName: f.SongName,
					ArtistName:    *f.ArtistName,
					ReleaseName:   f.AlbumName,
				})
			}
			log.Tracef("listenbrainz: looking up %d recording mbids", len(req.Recordings))
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
	}

	type request struct {
		RecordingMBID string        `json:"recording_mbid"`
		Score         FeedbackScore `json:"score"`
	}
	successSongs := make([]repos.SongSetLBFeedbackUploadedParams, 0, len(feedback))
	for _, f := range feedback {
		if f.SongMBID == nil {
			continue
		}
		log.Tracef("listenbrainz: uploading feedback for %s (%s, %v): %v", f.SongName, f.SongID, f.SongMBID, f.Score)
		_, err := listenBrainzRequest[any](ctx, "/1/feedback/recording-feedback", http.MethodPost, con.Token, request{
			RecordingMBID: *f.SongMBID,
			Score:         f.Score,
		})
		if err != nil {
			log.Errorf("listenbrainz: update song feedback: %s", err)
			continue
		}
		successSongs = append(successSongs, repos.SongSetLBFeedbackUploadedParams{
			SongID:     f.SongID,
			RemoteMBID: f.SongMBID,
			Uploaded:   true,
		})
	}

	err := l.db.Song().SetLBFeedbackUploaded(ctx, con.User, successSongs, true)
	if err != nil {
		return 0, fmt.Errorf("listenbrainz: update song feedback: update lb_feedback_status: %w", err)
	}
	return len(successSongs), nil
}

// SyncSongFeedback syncs according to the following logic:
// if upload status is unknown:      global favorite <- local is favorite OR remote is favorite
// if upload status is uploaded:     global favorite <- remote is favorite
// if upload status is not uploaded: global favorite <- local is favorite
func (l *ListenBrainz) SyncSongFeedback(ctx context.Context) error {
	users, err := l.db.User().FindAll(ctx)
	if err != nil {
		return fmt.Errorf("sync song feedback: %w", err)
	}
	var deletedLocal int
	var createdLocal int
	var uploadedLocal int
	for _, u := range users {
		if u.ListenBrainzUsername == nil {
			continue
		}
		token, err := repos.DecryptPassword(u.EncryptedListenBrainzToken)
		if err != nil {
			log.Errorf("listenbrainz: sync song feedback: decrypt listenbrainz token: %s", err)
			continue
		}

		lbFeedback, err := l.collectListenbrainzLoveFeedback(ctx, *u.ListenBrainzUsername, token)
		if err != nil {
			log.Errorf("listenbrainz: sync song feedback: collect listenbrainz love feedback: %s", err)
			continue
		}
		lbLovedMBIDs := make([]string, 0, len(lbFeedback))
		for _, f := range lbFeedback {
			lbLovedMBIDs = append(lbLovedMBIDs, f.RecordingMBID)
		}

		err = l.db.Song().SetLBFeedbackUploadedForAllMatchingStarredSongs(ctx, u.Name, lbLovedMBIDs)
		if err != nil {
			log.Errorf("listenbrainz: sync song feedback: set all already matching loved songs to uploaded: %s", err)
			continue
		}

		songs, err := l.db.Song().FindNotUploadedLBFeedback(ctx, u.Name, lbLovedMBIDs, repos.IncludeSongInfo{
			Album:       true,
			User:        u.Name,
			Annotations: true,
			Lists:       true,
		})
		if err != nil {
			log.Errorf("listenbrainz: sync song feedback: find not uploaded songs: %s", err)
			continue
		}
		feedbackList := make([]*Feedback, 0, len(songs))
		for _, s := range songs {
			score := FeedbackScoreNone
			if s.Starred != nil {
				score = FeedbackScoreLove
			}
			var artistName *string
			if len(s.Artists) > 0 {
				artistName = &s.Artists[0].Name
			}
			feedback := &Feedback{
				SongID:     s.ID,
				SongName:   s.Title,
				SongMBID:   s.MusicBrainzID,
				AlbumName:  s.AlbumName,
				Score:      score,
				ArtistName: artistName,
			}
			feedbackList = append(feedbackList, feedback)
		}

		songs, err = l.db.Song().FindLocalOutdatedFeedbackByLB(ctx, u.Name, lbLovedMBIDs, repos.IncludeSongInfo{
			User:        u.Name,
			Annotations: true,
		})
		if err != nil {
			log.Errorf("listenbrainz: sync song feedback: find local outdated feedback: %s", err)
			continue
		}

		unstarSongIDs := make([]string, 0, 5)
		starSongIDs := make([]string, 0, 5)
		for _, s := range songs {
			if s.Starred != nil {
				unstarSongIDs = append(unstarSongIDs, s.ID)
			} else {
				starSongIDs = append(starSongIDs, s.ID)
			}
		}

		uploadCount, err := l.UpdateSongFeedback(ctx, Connection{
			User:       u.Name,
			LBUsername: *u.ListenBrainzUsername,
			Token:      token,
		}, feedbackList)
		if err != nil {
			log.Errorf("listenbrainz: sync song feedback: upload not updated feedback: %s", err)
			continue
		}
		uploadedLocal += uploadCount

		err = l.db.Transaction(ctx, func(tx repos.Tx) error {
			deletedCount, err := tx.Song().UnStarMultiple(ctx, u.Name, unstarSongIDs)
			if err != nil {
				return fmt.Errorf("unstar starred songs not in list of love feedback: %w", err)
			}
			deletedLocal += deletedCount

			newStarCount, err := tx.Song().StarMultiple(ctx, u.Name, starSongIDs)
			if err != nil {
				return fmt.Errorf("star non-starred songs in list of love feedback: %w", err)
			}
			createdLocal += newStarCount

			err = tx.Song().SetLBFeedbackUploaded(ctx, u.Name, util.Map(songs, func(s *repos.CompleteSong) repos.SongSetLBFeedbackUploadedParams {
				return repos.SongSetLBFeedbackUploadedParams{
					SongID:     s.ID,
					RemoteMBID: nil,
					Uploaded:   true,
				}
			}), false)
			if err != nil {
				return fmt.Errorf("update lb_feedback_status: %w", err)
			}
			return nil
		})
		if err != nil {
			log.Errorf("listenbrainz: sync song feedback: update local feedback: %s", err)
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
