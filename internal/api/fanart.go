package api

import (
	"context"
	"errors"
	"os"
	"strconv"
	"sync"
	"time"

	"stremio-addon-douban/internal/model"
)

type FanartAPI struct {
	*BaseAPI
	tmdbAPI *TmdbAPI
}

var fanartInstances sync.Map

type fanartInstanceEntry struct {
	mu       sync.Mutex
	instance *FanartAPI
}

func GetFanartAPI(clientKey ...string) *FanartAPI {
	key := "__default__"
	if len(clientKey) > 0 && clientKey[0] != "" {
		key = clientKey[0]
	}
	entry, _ := fanartInstances.LoadOrStore(key, &fanartInstanceEntry{})
	e := entry.(*fanartInstanceEntry)
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.instance == nil {
		k := ""
		if len(clientKey) > 0 {
			k = clientKey[0]
		}
		e.instance = NewFanartAPI(k)
	}
	return e.instance
}

func NewFanartAPI(clientKey string) *FanartAPI {
	headers := map[string]string{}
	return &FanartAPI{
		BaseAPI: NewBaseAPI("https://webservice.fanart.tv/v3.2", headers),
		tmdbAPI: GetTmdbAPI(),
	}
}

func (f *FanartAPI) fanartParams(clientKey string) map[string]string {
	p := map[string]string{"api_key": os.Getenv("FANART_API_KEY")}
	if clientKey != "" {
		p["client_key"] = clientKey
	}
	return p
}

func (f *FanartAPI) requestWithRetry(ctx context.Context, path string, params map[string]string, cacheCfg *CacheConfig, target any) error {
	maxRetries := 3
	for attempt := 0; attempt <= maxRetries; attempt++ {
		err := f.RequestJSON(ctx, "GET", path, params, nil, nil, cacheCfg, target)
		if err == nil {
			return nil
		}
		var apiErr *APIError
		if !errors.As(err, &apiErr) || apiErr.Status != 429 || attempt == maxRetries {
			return err
		}
		wait := time.Duration(1<<attempt) * time.Second
		if apiErr.RetryAfterSeconds > 0 {
			wait = time.Duration(apiErr.RetryAfterSeconds) * time.Second
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
	}
	return nil
}

func (f *FanartAPI) GetMovieImages(ctx context.Context, movieID string, clientKey string) (*FanartMovieResponse, error) {
	var result FanartMovieResponse
	err := f.requestWithRetry(ctx, "/movies/"+movieID, f.fanartParams(clientKey),
		&CacheConfig{Key: "fanart:movie:" + movieID, TTL: model.SecondsDayPlusBuffer},
		&result)
	return &result, err
}

func (f *FanartAPI) GetShowImages(ctx context.Context, tvID string, clientKey string) (*FanartTVResponse, error) {
	var result FanartTVResponse
	err := f.requestWithRetry(ctx, "/tv/"+tvID, f.fanartParams(clientKey),
		&CacheConfig{Key: "fanart:tv:" + tvID, TTL: model.SecondsDayPlusBuffer},
		&result)
	return &result, err
}

type SubjectImages struct {
	Poster     string
	Background string
	Logo       string
}

func (f *FanartAPI) GetSubjectImages(ctx context.Context, mediaType, id string, clientKey string) (*SubjectImages, error) {
	if id == "" {
		return nil, nil
	}

	if mediaType == "movie" {
		resp, err := f.GetMovieImages(ctx, id, clientKey)
		if err != nil {
			return nil, err
		}
		return &SubjectImages{
			Poster:     firstURL(resp.MoviePoster),
			Background: firstURL(resp.MovieBackground, resp.MovieThumb),
			Logo:       firstURL(resp.HdMovieLogo, resp.MovieLogo),
		}, nil
	}

	// TV: need to resolve TVDB ID first
	if !isNumeric(id) {
		return nil, nil
	}
	tmdbID, _ := strconv.Atoi(id)
	extIDs, err := f.tmdbAPI.GetExternalID(ctx, "tv", tmdbID)
	if err != nil || extIDs.TvdbID == "" {
		return nil, nil
	}
	resp, err := f.GetShowImages(ctx, extIDs.TvdbID, clientKey)
	if err != nil {
		return nil, err
	}
	return &SubjectImages{
		Poster:     firstURL(resp.TVPoster),
		Background: firstURL(resp.ShowBackground),
		Logo:       firstURL(resp.HdTVLogo),
	}, nil
}

func firstURL(slices ...[]FanartImage) string {
	for _, s := range slices {
		if len(s) > 0 && s[0].URL != "" {
			return s[0].URL
		}
	}
	return ""
}

func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	_, err := strconv.Atoi(s)
	return err == nil
}
