package api

type TraktSearchResult struct {
	Score   float64        `json:"score"`
	Type    string         `json:"type"`
	Movie   *TraktMovie    `json:"movie"`
	Show    *TraktShow     `json:"show"`
	Episode *TraktEpisode  `json:"episode"`
}

type TraktMovie struct {
	Title         string   `json:"title"`
	OriginalTitle string   `json:"original_title"`
	Year          int      `json:"year"`
	IDs           TraktIDs `json:"ids"`
}

type TraktShow struct {
	Title         string   `json:"title"`
	OriginalTitle string   `json:"original_title"`
	Year          int      `json:"year"`
	IDs           TraktIDs `json:"ids"`
}

type TraktEpisode struct {
	Title string   `json:"title"`
	IDs   TraktIDs `json:"ids"`
}

type TraktIDs struct {
	Trakt *int    `json:"trakt"`
	Tmdb  *int    `json:"tmdb"`
	Imdb  *string `json:"imdb"`
}
