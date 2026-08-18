package model

type ManifestCatalog struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	Genres  []string `json:"genres,omitempty"`
}

type MetaDetail struct {
	ID          string     `json:"id"`
	Type        string     `json:"type"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	Poster      string     `json:"poster,omitempty"`
	Background  string     `json:"background,omitempty"`
	Logo        string     `json:"logo,omitempty"`
	Year        string     `json:"year,omitempty"`
	Genres      []string   `json:"genres"`
	Links       []MetaLink `json:"links,omitempty"`
	IMDBID      string     `json:"imdb_id,omitempty"`
	TMDBID      int        `json:"tmdbId,omitempty"`
	TMDBIDStr   string     `json:"tmdb_id,omitempty"`
}

type MetaLink struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	URL      string `json:"url"`
}

type CatalogResponse struct {
	Metas           []MetaDetail `json:"metas"`
	CacheMaxAge     int          `json:"cacheMaxAge,omitempty"`
	StaleRevalidate int          `json:"staleRevalidate,omitempty"`
	StaleError      int          `json:"staleError,omitempty"`
}

const (
	SecondsPerDay        = 86400
	SecondsPerWeek       = 604800
	SecondsDayPlusBuffer = 90000 // 25h：每日 2:00 全量拉取时旧缓存仍有 1h 余量
)
