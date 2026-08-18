package api

type TmdbGenre struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type TmdbGenreMap struct {
	Genres []TmdbGenre `json:"genres"`
}

type TmdbTrendingItem struct {
	ID           int     `json:"id"`
	Title        string  `json:"title"`
	Name         string  `json:"name"`
	Overview     string  `json:"overview"`
	PosterPath   *string `json:"poster_path"`
	BackdropPath *string `json:"backdrop_path"`
	VoteAverage  float64 `json:"vote_average"`
	ReleaseDate  string  `json:"release_date"`
	FirstAirDate string  `json:"first_air_date"`
	GenreIDs     []int   `json:"genre_ids"`
	MediaType    string  `json:"media_type"`
	Year         string  `json:"-"`
}

type TmdbTrendingResult struct {
	Results []TmdbTrendingItem `json:"results"`
}

type TmdbImageData struct {
	FilePath     string  `json:"file_path"`
	Iso639_1     *string `json:"iso_639_1"`
	Iso3166_1    string  `json:"iso_3166_1"`
	VoteAverage  float64 `json:"vote_average"`
	VoteCount    int     `json:"vote_count"`
}

type TmdbSubjectImages struct {
	Backdrops []TmdbImageData `json:"backdrops"`
	Posters   []TmdbImageData `json:"posters"`
	Logos     []TmdbImageData `json:"logos"`
}

type TmdbFindResult struct {
	MovieResults       []TmdbFindItem `json:"movie_results"`
	TVResults          []TmdbFindItem `json:"tv_results"`
	TVEpisodeResults   []TmdbFindEpisodeItem `json:"tv_episode_results"`
}

type TmdbFindItem struct {
	ID int `json:"id"`
}

type TmdbFindEpisodeItem struct {
	ShowID int `json:"show_id"`
}

type TmdbExternalIDs struct {
	ID     int    `json:"id"`
	ImdbID string `json:"imdb_id"`
	TvdbID string `json:"tvdb_id"`
}

type TmdbCredit struct {
	Name       string `json:"name"`
	Job        string `json:"job"`
	Department string `json:"department"`
}

type TmdbCast struct {
	Name string `json:"name"`
}

type TmdbCredits struct {
	Crew []TmdbCredit `json:"crew"`
	Cast []TmdbCast   `json:"cast"`
}

type TmdbDetailGenre struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type TmdbExternalIDsNested struct {
	ImdbID string `json:"imdb_id"`
}

type TmdbDetail struct {
	ID           int                    `json:"id"`
	Title        string                 `json:"title"`
	Name         string                 `json:"name"`
	Overview     string                 `json:"overview"`
	PosterPath   *string                `json:"poster_path"`
	BackdropPath *string                `json:"backdrop_path"`
	ReleaseDate  string                 `json:"release_date"`
	FirstAirDate string                 `json:"first_air_date"`
	VoteAverage  float64                `json:"vote_average"`
	Genres       []TmdbDetailGenre      `json:"genres"`
	Credits      TmdbCredits            `json:"credits"`
	ExternalIDs  TmdbExternalIDsNested  `json:"external_ids"`
	Year         string                 `json:"-"`
	ImdbID       string                 `json:"-"`
	Directors    []string               `json:"-"`
	CastNames    []string               `json:"-"`
	GenreNames   []string               `json:"-"`
	GenreIDs     []int                  `json:"-"`
}

type TmdbSearchResult struct {
	Results []TmdbTrendingItem `json:"results"`
}
