package api

import (
	"context"
	"fmt"
	"os"

	"stremio-addon-douban/internal/model"
	"stremio-addon-douban/internal/version"
)

type TraktAPI struct {
	*BaseAPI
}

func NewTraktAPI() *TraktAPI {
	return &TraktAPI{
		BaseAPI: NewBaseAPI("https://api.trakt.tv", map[string]string{
			"trakt-api-version": "2",
			"trakt-api-key":     os.Getenv("TRAKT_CLIENT_ID"),
			"User-Agent":        "stremio-addon-douban/" + version.Get(),
		}),
	}
}

func (t *TraktAPI) Search(ctx context.Context, searchType, query string) ([]TraktSearchResult, error) {
	var result []TraktSearchResult
	err := t.RequestJSON(ctx, "GET", "/search/"+searchType,
		map[string]string{"query": query},
		nil, nil,
		&CacheConfig{Key: fmt.Sprintf("trakt:search:%s:%s", searchType, query), TTL: model.SecondsPerDay * 2},
		&result)
	return result, err
}

func (t *TraktAPI) SearchByImdbID(ctx context.Context, imdbID string) ([]TraktSearchResult, error) {
	var result []TraktSearchResult
	err := t.RequestJSON(ctx, "GET", "/search/imdb/"+imdbID,
		nil, nil, nil,
		&CacheConfig{Key: "trakt:search:imdb:" + imdbID, TTL: model.SecondsPerDay * 2},
		&result)
	return result, err
}

func (t *TraktAPI) SearchByTmdbID(ctx context.Context, tmdbID string) ([]TraktSearchResult, error) {
	var result []TraktSearchResult
	err := t.RequestJSON(ctx, "GET", "/search/tmdb/"+tmdbID,
		nil, nil, nil,
		&CacheConfig{Key: "trakt:search:tmdb:" + tmdbID, TTL: model.SecondsPerDay * 2},
		&result)
	return result, err
}

func (t *TraktAPI) GetSearchResultField(result *TraktSearchResult, field string) any {
	switch field {
	case "ids":
		if result.Type == "show" || result.Type == "episode" {
			if result.Show != nil {
				return result.Show.IDs
			}
		}
		if result.Type == "movie" && result.Movie != nil {
			return result.Movie.IDs
		}
	case "title":
		if result.Type == "show" || result.Type == "episode" {
			if result.Show != nil {
				return result.Show.Title
			}
		}
		if result.Type == "movie" && result.Movie != nil {
			return result.Movie.Title
		}
	case "original_title":
		if result.Type == "show" || result.Type == "episode" {
			if result.Show != nil {
				return result.Show.OriginalTitle
			}
		}
		if result.Type == "movie" && result.Movie != nil {
			return result.Movie.OriginalTitle
		}
	case "year":
		if result.Type == "show" || result.Type == "episode" {
			if result.Show != nil {
				return result.Show.Year
			}
		}
		if result.Type == "movie" && result.Movie != nil {
			return result.Movie.Year
		}
	}
	return nil
}

type IDMapping struct {
	TraktID *int
	TmdbID  *int
	ImdbID  *string
}

func (t *TraktAPI) FormatIDsToIDMapping(ids *TraktIDs) *IDMapping {
	if ids == nil {
		return nil
	}
	return &IDMapping{
		TraktID: ids.Trakt,
		TmdbID:  ids.Tmdb,
		ImdbID:  ids.Imdb,
	}
}
