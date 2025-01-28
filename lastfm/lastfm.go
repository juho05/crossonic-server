package lastfm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/andybalholm/cascadia"
	"github.com/juho05/crossonic-server"
	"github.com/juho05/log"
	"golang.org/x/net/html"
)

var (
	ErrUnexpectedResponseCode = errors.New("unexpected response code")
	ErrUnexpectedResponseBody = errors.New("unexpected response body")
	ErrUnauthenticated        = errors.New("unauthenticated")
	ErrNotFound               = errors.New("not found")
)

const baseURL = "https://ws.audioscrobbler.com/2.0"

type LastFm struct {
	apiKey string
}

func New(apiKey string) *LastFm {
	return &LastFm{
		apiKey: apiKey,
	}
}

type ArtistInfo struct {
	Name string  `json:"name"`
	MBID *string `json:"mbid"`
	URL  string  `json:"url"`
	Bio  struct {
		Summary *string `json:"summary"`
		Content *string `json:"content"`
	} `json:"bio"`
}

func (l *LastFm) GetArtistInfo(ctx context.Context, name string, mbid *string) (ArtistInfo, error) {
	params := make(map[string][]string, 3)
	if mbid != nil {
		params["mbid"] = []string{*mbid}
	} else {
		params["artist"] = []string{name}
	}
	params["autocorrect"] = []string{"1"}

	log.Tracef("fetching artist info for %s from last.fm...", name)
	res, err := lastFMRequest[ArtistInfo](l, ctx, "artist.getinfo", "artist", params)
	if mbid != nil && errors.Is(err, ErrNotFound) {
		log.Tracef("artist %s not found on last.fm with mbid, trying by name...", name)
		delete(params, "mbid")
		params["artist"] = []string{name}
		res, err = lastFMRequest[ArtistInfo](l, ctx, "artist.getinfo", "artist", params)
	}
	if err != nil {
		return ArtistInfo{}, fmt.Errorf("get artist info: %w", err)
	}
	if res.Bio.Content != nil {
		c := cleanUpLastFmDescription(*res.Bio.Content)
		res.Bio.Content = &c
	}
	return res, nil
}

type AlbumInfo struct {
	Name string  `json:"name"`
	MBID *string `json:"mbid"`
	URL  string  `json:"url"`
	Wiki struct {
		Summary *string `json:"summary"`
		Content *string `json:"content"`
	} `json:"wiki"`
}

func (l *LastFm) GetAlbumInfo(ctx context.Context, name, artistName string, mbid *string) (AlbumInfo, error) {
	params := make(map[string][]string, 3)
	if mbid != nil {
		params["mbid"] = []string{*mbid}
	} else {
		params["artist"] = []string{artistName}
		params["album"] = []string{name}
	}
	params["autocorrect"] = []string{"1"}

	log.Tracef("fetching album info for %s from last.fm...", name)
	res, err := lastFMRequest[AlbumInfo](l, ctx, "album.getinfo", "album", params)
	if errors.Is(err, ErrNotFound) {
		log.Tracef("album %s not found on last.fm with mbid, trying by name...", name)
		delete(params, "mbid")
		params["artist"] = []string{artistName}
		params["album"] = []string{name}
		res, err = lastFMRequest[AlbumInfo](l, ctx, "album.getinfo", "album", params)
	}
	if err != nil {
		return AlbumInfo{}, fmt.Errorf("get album info: %w", err)
	}
	if res.Wiki.Content != nil {
		c := cleanUpLastFmDescription(*res.Wiki.Content)
		res.Wiki.Content = &c
	}
	return res, nil
}

var artistOpenGraphQuery = cascadia.MustCompile(`html > head > meta[property="og:image"]`)

// from https://github.com/sentriz/gonic/blob/0e45f5e84cd650211351179edf3eed89a54c6c75/lastfm/client.go#L182
func (l *LastFm) GetArtistImageURL(ctx context.Context, artistURL string) (string, error) {
	resp, err := http.Get(artistURL)
	if err != nil {
		return "", fmt.Errorf("get artist image url: get artist page: %w", err)
	}
	defer resp.Body.Close()

	node, err := html.Parse(resp.Body)
	if err != nil {
		return "", fmt.Errorf("get artist image url: parse html: %w", err)
	}

	n := cascadia.Query(node, artistOpenGraphQuery)
	if n == nil {
		return "", nil
	}

	var imageURL string
	for _, attr := range n.Attr {
		if attr.Key == "content" {
			imageURL = attr.Val
			break
		}
	}

	return imageURL, nil
}

func lastFMRequest[T any](l *LastFm, ctx context.Context, method, responseKey string, params map[string][]string) (T, error) {
	var obj T
	query := make(url.Values, len(params)+3)
	query.Set("method", method)
	query.Set("api_key", l.apiKey)
	query.Set("format", "json")
	for k, v := range params {
		query[k] = v
	}
	url := fmt.Sprintf("%s?%s", baseURL, query.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return obj, fmt.Errorf("last.fm new request: %w", err)
	}
	req.Header.Set("User-Agent", fmt.Sprintf("%s/%s", crossonic.ServerName, crossonic.Version))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return obj, fmt.Errorf("last.fm do request: %w", err)
	}
	if res.StatusCode == http.StatusTooManyRequests {
		res.Body.Close()
		secondsStr := res.Header.Get("X-RateLimit-Reset-In")
		if secondsStr == "" {
			secondsStr = res.Header.Get("Retry-After")
			if secondsStr == "" {
				log.Error("missing X-RateLimit-Reset-In or Retry-After in 429 last.fm response")
				secondsStr = "1"
			}
		}
		seconds, err := strconv.Atoi(secondsStr)
		if err != nil {
			t, err := time.Parse(http.TimeFormat, secondsStr)
			if err != nil {
				log.Errorf("invalid value of X-RateLimit-Reset-In/Retry-After in 429 last.fm response: %s", secondsStr)
				seconds = 1
			} else {
				seconds = int(math.Ceil(time.Until(t).Seconds()))
			}
		}
		time.Sleep(time.Duration(seconds) * time.Second)
		return lastFMRequest[T](l, ctx, method, responseKey, params)
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusUnauthorized {
		return obj, fmt.Errorf("last.fm request: %w", ErrUnauthenticated)
	}
	if res.StatusCode != http.StatusOK {
		if res.StatusCode == http.StatusNotFound {
			return obj, fmt.Errorf("last.fm request: %w", ErrNotFound)
		}
		return obj, fmt.Errorf("last.fm request: %w: %d", ErrUnexpectedResponseCode, res.StatusCode)
	}

	var body map[string]json.RawMessage
	err = json.NewDecoder(res.Body).Decode(&body)
	if err != nil {
		return obj, fmt.Errorf("last.fm request: decode: %w: %w", ErrUnexpectedResponseBody, err)
	}
	if e, ok := body["error"]; ok {
		var code int
		err = json.Unmarshal(e, &code)
		if err != nil {
			return obj, fmt.Errorf("last.fm request: decode error code: %w: %w", ErrUnexpectedResponseBody, err)
		}
		// rate limit exceeded
		if code == 29 {
			time.Sleep(1 * time.Second)
			return lastFMRequest[T](l, ctx, method, responseKey, params)
		}
		// not found
		if code == 6 {
			return obj, fmt.Errorf("last.fm request: %w", ErrNotFound)
		}
		return obj, fmt.Errorf("last.fm request: error code %d: %w", code, ErrUnexpectedResponseCode)
	}
	if data, ok := body[responseKey]; ok {
		err = json.Unmarshal(data, &obj)
		if err != nil {
			return obj, fmt.Errorf("last.fm request: decode response: %w: %w", ErrUnexpectedResponseBody, err)
		}
		return obj, nil
	} else {
		return obj, fmt.Errorf("last.fm request: response key %s does not exist in response: %w", responseKey, ErrUnexpectedResponseBody)
	}
}

func cleanUpLastFmDescription(description string) string {
	runes := []rune(description)
	var builder strings.Builder
	for i := 0; i < len(runes); i++ {
		// replace newlines with space
		if runes[i] == '\n' {
			runes[i] = ' '
		}

		// skip double space
		if runes[i] == ' ' && i > 0 && runes[i-1] == ' ' {
			continue
		}

		if i < len(description)-1 {
			// remove <a> tags
			if runes[i] == '<' && (runes[i+1] == 'a' || (runes[i+1] == '/' && i < len(runes)-2 && runes[i+2] == 'a')) {
				for i < len(runes) && runes[i] != '>' {
					i++
				}
				continue
			}
		}

		builder.WriteRune(runes[i])
	}
	str := builder.String()
	str = strings.TrimSuffix(str, "Read more on Last.fm. User-contributed text is available under the Creative Commons By-SA License; additional terms may apply.")
	return strings.TrimSpace(str)
}
