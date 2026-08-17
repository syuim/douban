package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/url"
	"os"
	"strconv"

	"stremio-addon-douban/internal/api"
)

type ImageProvider struct {
	Provider string          `json:"provider"`
	Extra    json.RawMessage `json:"extra"`
}

type FanartExtra struct {
	APIKey string `json:"apiKey,omitempty"`
}

type TmdbExtra struct {
	APIKey         string   `json:"apiKey,omitempty"`
	ImageLanguages []string `json:"imageLanguages,omitempty"`
}

// defaultImageProviders 固定图片来源优先级：TMDB → 豆瓣 → Fanart
func defaultImageProviders() []ImageProvider {
	return []ImageProvider{
		{Provider: "tmdb", Extra: json.RawMessage(`{}`)},
		{Provider: "douban", Extra: json.RawMessage(`{}`)},
		{Provider: "fanart", Extra: json.RawMessage(`{}`)},
	}
}

type ImageURLs struct {
	Poster     string
	Background string
	Logo       string
}

// directProxyURL 生成直连统一代理的图片 URL（emby-proxy /url?url=...），
// 客户端不经 addon 节点直接访问 CF 边缘缓存。
func directProxyURL(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	proxyURL := os.Getenv("PROXY_URL")
	if proxyURL == "" {
		proxyURL = api.DefaultProxyURL
	}
	return proxyURL + "?url=" + url.QueryEscape(rawURL)
}

func generateImageURLs(ctx context.Context, doubanInfo api.DoubanSubjectCollectionItem, tmdbID int, imdbID string) ImageURLs {
	result := ImageURLs{}

	for _, provider := range defaultImageProviders() {
		var urls *ImageURLs

		switch provider.Provider {
		case "douban":
			cover := doubanInfo.CoverURL
			var bg string
			if len(doubanInfo.Photos) > 0 {
				bg = doubanInfo.Photos[0]
			}
			urls = &ImageURLs{
				Poster:     directProxyURL(cover),
				Background: directProxyURL(bg),
			}

		case "fanart":
			id := ""
			if tmdbID != 0 {
				id = itoa(tmdbID)
			} else if imdbID != "" {
				id = imdbID
			}
			if id != "" {
				var extra FanartExtra
				if len(provider.Extra) > 0 {
					if err := json.Unmarshal(provider.Extra, &extra); err != nil {
						log.Printf("invalid fanart provider extra: %v", err)
					}
				}
				fanartAPI := api.GetFanartAPI(extra.APIKey)
				images, err := fanartAPI.GetSubjectImages(ctx, doubanInfo.Type, id, extra.APIKey)
				if err == nil && images != nil {
					urls = &ImageURLs{
						Poster:     directProxyURL(images.Poster),
						Background: directProxyURL(images.Background),
						Logo:       directProxyURL(images.Logo),
					}
				}
			}

		case "tmdb":
			if tmdbID != 0 {
				var extra TmdbExtra
				if len(provider.Extra) > 0 {
					if err := json.Unmarshal(provider.Extra, &extra); err != nil {
						log.Printf("invalid tmdb provider extra: %v", err)
					}
				}
				tmdbAPI := api.GetTmdbAPI(extra.APIKey)
				images, err := tmdbAPI.GetSubjectImages(ctx, doubanInfo.Type, tmdbID, extra.ImageLanguages)
				if err == nil && images != nil {
					langs := extra.ImageLanguages
					if langs == nil {
						langs = api.TMDBImageLanguage
					}
					urls = &ImageURLs{}
					if sorted := api.SortTmdbImages(images.Posters, langs); len(sorted) > 0 {
						urls.Poster = directProxyURL("https://image.tmdb.org/t/p/original" + sorted[0].FilePath)
					}
					if sorted := api.SortTmdbImages(images.Backdrops, langs); len(sorted) > 0 {
						urls.Background = directProxyURL("https://image.tmdb.org/t/p/original" + sorted[0].FilePath)
					}
					if sorted := api.SortTmdbImages(images.Logos, langs); len(sorted) > 0 {
						urls.Logo = directProxyURL("https://image.tmdb.org/t/p/original" + sorted[0].FilePath)
					}
				}
			}
		}

		if urls != nil {
			if result.Poster == "" {
				result.Poster = urls.Poster
			}
			if result.Background == "" {
				result.Background = urls.Background
			}
			if result.Logo == "" {
				result.Logo = urls.Logo
			}
			if result.Poster != "" && result.Background != "" && result.Logo != "" {
				break
			}
		}
	}

	return result
}

func itoa(n int) string {
	return strconv.Itoa(n)
}
