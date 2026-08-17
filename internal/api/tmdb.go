package api

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"

	"stremio-addon-douban/internal/model"
)

var TMDBImageLanguage = []string{"zh", "en", "ja", "ko", "null"}

type TmdbAPI struct {
	*BaseAPI
	apiKey string
}

var (
	tmdbInstances sync.Map
)

type tmdbInstanceEntry struct {
	mu       sync.Mutex
	instance *TmdbAPI
}

func GetTmdbAPI(apiKey ...string) *TmdbAPI {
	key := "__default__"
	if len(apiKey) > 0 && apiKey[0] != "" {
		key = apiKey[0]
	}
	entry, _ := tmdbInstances.LoadOrStore(key, &tmdbInstanceEntry{})
	e := entry.(*tmdbInstanceEntry)
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.instance == nil {
		k := ""
		if len(apiKey) > 0 {
			k = apiKey[0]
		}
		e.instance = NewTmdbAPI(k)
	}
	return e.instance
}

func NewTmdbAPI(apiKey string) *TmdbAPI {
	if apiKey == "" {
		apiKey = os.Getenv("TMDB_API_KEY")
	}
	return &TmdbAPI{
		BaseAPI: NewBaseAPI("https://api.themoviedb.org/3", map[string]string{
			"Authorization": "Bearer " + apiKey,
		}),
		apiKey: apiKey,
	}
}

func (t *TmdbAPI) Search(ctx context.Context, searchType string, params map[string]string) (*TmdbSearchResult, error) {
	var result TmdbSearchResult
	err := t.RequestJSON(ctx, "GET", "/search/"+searchType, params, nil, nil, nil, &result)
	return &result, err
}

func (t *TmdbAPI) FindByID(ctx context.Context, externalID, externalSource string) (*TmdbFindResult, error) {
	var result TmdbFindResult
	err := t.RequestJSON(ctx, "GET", "/find/"+externalID,
		map[string]string{"external_source": externalSource, "language": "zh-CN"},
		nil, nil, nil, &result)
	return &result, err
}

func (t *TmdbAPI) GetExternalID(ctx context.Context, mediaType string, id int) (*TmdbExternalIDs, error) {
	var result TmdbExternalIDs
	err := t.RequestJSON(ctx, "GET", fmt.Sprintf("/%s/%d/external_ids", mediaType, id),
		nil, nil, nil,
		&CacheConfig{Key: fmt.Sprintf("tmdb:%s:%d:external_ids", mediaType, id), TTL: model.SecondsPerWeek},
		&result)
	return &result, err
}

func (t *TmdbAPI) GetSubjectImages(ctx context.Context, mediaType string, id int, imageLanguages []string) (*TmdbSubjectImages, error) {
	if imageLanguages == nil {
		imageLanguages = TMDBImageLanguage
	}
	var result TmdbSubjectImages
	err := t.RequestJSON(ctx, "GET", fmt.Sprintf("/%s/%d/images", mediaType, id),
		map[string]string{"include_image_language": strings.Join(imageLanguages, ",")},
		nil, nil,
		&CacheConfig{Key: fmt.Sprintf("tmdb:%s:%d:images:%s", mediaType, id, strings.Join(imageLanguages, ",")), TTL: model.SecondsPerWeek},
		&result)
	return &result, err
}

func (t *TmdbAPI) Trending(ctx context.Context, mediaType, timeWindow string, page int) (*TmdbTrendingResult, error) {
	var result TmdbTrendingResult
	err := t.RequestJSON(ctx, "GET", fmt.Sprintf("/trending/%s/%s", mediaType, timeWindow),
		map[string]string{"language": "zh-CN", "page": fmt.Sprintf("%d", page)},
		nil, nil,
		&CacheConfig{Key: fmt.Sprintf("tmdb:trending:%s:%s:%d", mediaType, timeWindow, page), TTL: model.SecondsPerDay * 2},
		&result)
	if err != nil {
		return nil, err
	}
	for i := range result.Results {
		result.Results[i].Year = extractYear(result.Results[i].ReleaseDate, result.Results[i].FirstAirDate)
		if result.Results[i].Title == "" {
			result.Results[i].Title = result.Results[i].Name
		}
	}
	return &result, nil
}

func (t *TmdbAPI) Discover(ctx context.Context, mediaType string, params map[string]string, page int) (*TmdbTrendingResult, error) {
	p := map[string]string{"language": "zh-CN", "page": fmt.Sprintf("%d", page)}
	for k, v := range params {
		p[k] = v
	}
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	pairs := make([]string, 0, len(keys))
	for _, k := range keys {
		pairs = append(pairs, k+"="+params[k])
	}
	paramsKey := strings.Join(pairs, ",")
	var result TmdbTrendingResult
	err := t.RequestJSON(ctx, "GET", "/discover/"+mediaType, p, nil, nil,
		&CacheConfig{Key: fmt.Sprintf("tmdb:discover:%s:%s:%d", mediaType, paramsKey, page), TTL: model.SecondsPerDay * 2},
		&result)
	if err != nil {
		return nil, err
	}
	for i := range result.Results {
		result.Results[i].Year = extractYear(result.Results[i].ReleaseDate, result.Results[i].FirstAirDate)
		if result.Results[i].Title == "" {
			result.Results[i].Title = result.Results[i].Name
		}
	}
	return &result, nil
}

func (t *TmdbAPI) GetDetail(ctx context.Context, mediaType string, id int) (*TmdbDetail, error) {
	var result TmdbDetail
	err := t.RequestJSON(ctx, "GET", fmt.Sprintf("/%s/%d", mediaType, id),
		map[string]string{"language": "zh-CN", "append_to_response": "credits,external_ids"},
		nil, nil,
		&CacheConfig{Key: fmt.Sprintf("tmdb:detail:%s:%d", mediaType, id), TTL: model.SecondsPerWeek},
		&result)
	if err != nil {
		return nil, err
	}
	// post-process
	if result.Title == "" {
		result.Title = result.Name
	}
	result.Year = extractYear(result.ReleaseDate, result.FirstAirDate)
	result.ImdbID = result.ExternalIDs.ImdbID
	for _, c := range result.Credits.Crew {
		if c.Job == "Director" {
			result.Directors = append(result.Directors, c.Name)
		}
	}
	for i, c := range result.Credits.Cast {
		if i >= 10 {
			break
		}
		result.CastNames = append(result.CastNames, c.Name)
	}
	for _, g := range result.Genres {
		result.GenreNames = append(result.GenreNames, g.Name)
	}
	return &result, nil
}

func (t *TmdbAPI) GetGenres(ctx context.Context, mediaType string) (*TmdbGenreMap, error) {
	var result TmdbGenreMap
	err := t.RequestJSON(ctx, "GET", "/genre/"+mediaType+"/list",
		map[string]string{"language": "zh-CN"},
		nil, nil,
		&CacheConfig{Key: "tmdb:genres:" + mediaType, TTL: model.SecondsPerWeek},
		&result)
	return &result, err
}

func extractYear(releaseDate, firstAirDate string) string {
	d := releaseDate
	if d == "" {
		d = firstAirDate
	}
	if len(d) >= 4 {
		return d[:4]
	}
	return ""
}

func SortTmdbImages(images []TmdbImageData, imageLanguages []string) []TmdbImageData {
	sorted := make([]TmdbImageData, len(images))
	copy(sorted, images)
	sort.SliceStable(sorted, func(i, j int) bool {
		li := tmdbImageLangIndex(sorted[i], imageLanguages)
		lj := tmdbImageLangIndex(sorted[j], imageLanguages)
		if li != lj {
			return li < lj
		}
		if sorted[i].VoteAverage != sorted[j].VoteAverage {
			return sorted[i].VoteAverage > sorted[j].VoteAverage
		}
		return sorted[i].VoteCount > sorted[j].VoteCount
	})
	return sorted
}

func tmdbImageLangIndex(img TmdbImageData, langs []string) int {
	langCode := "null"
	if img.Iso639_1 != nil {
		langCode = *img.Iso639_1
	}
	fullLang := langCode
	if img.Iso3166_1 != "" {
		fullLang = langCode + "-" + img.Iso3166_1
	}
	for i, l := range langs {
		if l == fullLang || l == langCode {
			return i
		}
	}
	return len(langs)
}
