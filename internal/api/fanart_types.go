package api

type FanartImage struct {
	URL string `json:"url"`
}

type FanartMovieResponse struct {
	HdMovieLogo   []FanartImage `json:"hdmovielogo"`
	MovieLogo     []FanartImage `json:"movielogo"`
	MoviePoster   []FanartImage `json:"movieposter"`
	MovieBackground []FanartImage `json:"moviebackground"`
	MovieThumb    []FanartImage `json:"moviethumb"`
}

type FanartTVResponse struct {
	HdTVLogo     []FanartImage `json:"hdtvlogo"`
	TVPoster     []FanartImage `json:"tvposter"`
	ShowBackground []FanartImage `json:"showbackground"`
}
