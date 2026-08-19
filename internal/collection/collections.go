package collection

import (
	"fmt"

	"stremio-addon-douban/internal/model"
)

type CollectionConfig struct {
	model.ManifestCatalog
	HasGenre  bool `json:"-"`
	IsDefault bool `json:"-"`
	IsTmdb    bool `json:"-"`
}

type YearlyRankingItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Year int    `json:"year"`
}

const (
	MovieYearlyRankingID = "__movie_yearly_ranking__"
	TVYearlyRankingID    = "__tv_yearly_ranking__"

	TmdbTrendingMovieID = "tmdb_trending_movie"
	TmdbTrendingTvID    = "tmdb_trending_tv"
	TmdbDiscoverAnimeID = "tmdb_discover_anime"
	TmdbDiscoverMovieID = "tmdb_discover_movie"
	TmdbDiscoverTvID    = "tmdb_discover_tv"
)

var MovieYearlyRanking = []YearlyRankingItem{
	{ID: "ECE472UNY", Year: 2025}, {ID: "ECBE7RX5A", Year: 2024},
	{ID: "ECQ46F7XI", Year: 2023}, {ID: "ECKA55LSA", Year: 2022},
	{ID: "ECWY6B2GQ", Year: 2021}, {ID: "EC2A5MRIY", Year: 2020},
	{ID: "ECFYHQBWQ", Year: 2019}, {ID: "2018_movie_1", Year: 2018},
	{ID: "2017_movie_chinese_score", Year: 2017}, {ID: "2016_movie_451", Year: 2016},
	{ID: "2015_movie_3", Year: 2015}, {ID: "2014_movie_2", Year: 2014},
}

var TVYearlyRanking = []YearlyRankingItem{
	{ID: "EC2FACYKQ", Year: 2025}, {ID: "ECYA7RAZQ", Year: 2024},
	{ID: "ECTE6EOZA", Year: 2023}, {ID: "ECWU56XUI", Year: 2022},
	{ID: "ECOY56I6Y", Year: 2021}, {ID: "ECCM5TXSI", Year: 2020},
	{ID: "ECR4HOW3I", Year: 2019}, {ID: "2018_tv_23", Year: 2018},
	{ID: "2017_tv_domestic_score", Year: 2017}, {ID: "2016_tv_478", Year: 2016},
	{ID: "2015_tv_6", Year: 2015}, {ID: "2014_tv_14", Year: 2014},
}

var YearlyRankings = map[string][]YearlyRankingItem{
	MovieYearlyRankingID: MovieYearlyRanking,
	TVYearlyRankingID:    TVYearlyRanking,
}

func init() {
	for i := range MovieYearlyRanking {
		MovieYearlyRanking[i].Name = fmt.Sprintf("豆瓣 %d 评分最高电影", MovieYearlyRanking[i].Year)
	}
	for i := range TVYearlyRanking {
		TVYearlyRanking[i].Name = fmt.Sprintf("豆瓣 %d 评分最高剧集", TVYearlyRanking[i].Year)
	}
}

var movieGenreConfigs = []struct{ ID, Name string }{
	{"film_genre_27", "剧情片榜"}, {"movie_comedy", "喜剧片榜"}, {"movie_love", "爱情片榜"},
	{"movie_action", "动作片榜"}, {"movie_scifi", "科幻片榜"}, {"film_genre_31", "动画片榜"},
	{"film_genre_32", "悬疑片榜"}, {"film_genre_46", "犯罪片榜"}, {"film_genre_33", "惊悚片榜"},
	{"film_genre_49", "冒险片榜"}, {"film_genre_41", "家庭片榜"}, {"film_genre_42", "儿童片榜"},
	{"film_genre_44", "历史片榜"}, {"film_genre_39", "音乐片榜"}, {"film_genre_48", "奇幻片榜"},
	{"film_genre_34", "恐怖片榜"}, {"film_genre_45", "战争片榜"}, {"film_genre_43", "传记片榜"},
	{"film_genre_40", "歌舞片榜"}, {"film_genre_50", "武侠片榜"}, {"film_genre_37", "情色片榜"},
	{"natural_disasters", "灾难片榜"}, {"film_genre_47", "西部片榜"}, {"film_genre_51", "古装片榜"},
	{"ECCEPGM4Y", "运动片榜"}, {"film_genre_36", "短片榜"},
}

var tvGenreConfigs = []struct{ ID, Name string }{
	{"EC74443FY", "大陆剧榜"}, {"ECFA5DI7Q", "美剧榜"}, {"ECVACXBWI", "英剧榜"},
	{"ECNA46YBA", "日剧榜"}, {"ECBE5CBEI", "韩剧榜"}, {"ECVM47WUA", "港剧榜"},
	{"ECBI5EL6A", "台剧榜"}, {"EC6I5FYHA", "欧洲剧榜"},
}

var CollectionConfigs []CollectionConfig

func init() {
	CollectionConfigs = []CollectionConfig{
		{ManifestCatalog: model.ManifestCatalog{ID: "movie_hot_gaia", Name: "豆瓣热门电影", Type: "movie"}, IsDefault: true},
		{ManifestCatalog: model.ManifestCatalog{ID: "movie_weekly_best", Name: "一周口碑电影榜", Type: "movie"}, IsDefault: true},
		{ManifestCatalog: model.ManifestCatalog{ID: "movie_real_time_hotest", Name: "实时热门电影", Type: "movie"}, IsDefault: true},
		{ManifestCatalog: model.ManifestCatalog{ID: "movie_top250", Name: "豆瓣电影 Top250", Type: "movie"}, IsDefault: true},
		{ManifestCatalog: model.ManifestCatalog{ID: "movie_showing", Name: "影院热映", Type: "movie"}, IsDefault: true},
	}

	for _, g := range movieGenreConfigs {
		CollectionConfigs = append(CollectionConfigs, CollectionConfig{
			ManifestCatalog: model.ManifestCatalog{ID: g.ID, Name: g.Name, Type: "movie"},
			HasGenre:        true,
		})
	}

	CollectionConfigs = append(CollectionConfigs, CollectionConfig{
		ManifestCatalog: model.ManifestCatalog{ID: MovieYearlyRankingID, Name: "豆瓣年度评分最高电影", Type: "movie"},
		HasGenre:        true,
	})

	tvConfigs := []CollectionConfig{
		{ManifestCatalog: model.ManifestCatalog{ID: "tv_hot", Name: "近期热门剧集", Type: "series"}, IsDefault: true},
		{ManifestCatalog: model.ManifestCatalog{ID: "tv_american", Name: "近期热门美剧", Type: "series"}},
		{ManifestCatalog: model.ManifestCatalog{ID: "tv_korean", Name: "近期热门韩剧", Type: "series"}},
		{ManifestCatalog: model.ManifestCatalog{ID: "tv_domestic", Name: "近期热门国产剧", Type: "series"}},
		{ManifestCatalog: model.ManifestCatalog{ID: "tv_japanese", Name: "近期热门日剧", Type: "series"}},
		{ManifestCatalog: model.ManifestCatalog{ID: "tv_animation", Name: "近期热门动画", Type: "series"}, IsDefault: true},
		{ManifestCatalog: model.ManifestCatalog{ID: "show_hot", Name: "近期热门综艺节目", Type: "series"}, IsDefault: true},
		{ManifestCatalog: model.ManifestCatalog{ID: "tv_documentary", Name: "近期热门纪录片", Type: "series"}},
		{ManifestCatalog: model.ManifestCatalog{ID: "tv_real_time_hotest", Name: "实时热门电视", Type: "series"}, IsDefault: true},
		{ManifestCatalog: model.ManifestCatalog{ID: "tv_chinese_best_weekly", Name: "华语口碑剧集榜", Type: "series"}, IsDefault: true},
		{ManifestCatalog: model.ManifestCatalog{ID: "tv_global_best_weekly", Name: "全球口碑剧集榜", Type: "series"}, IsDefault: true},
		{ManifestCatalog: model.ManifestCatalog{ID: "show_chinese_best_weekly", Name: "国内口碑综艺榜", Type: "series"}, IsDefault: true},
		{ManifestCatalog: model.ManifestCatalog{ID: "show_global_best_weekly", Name: "国外口碑综艺榜", Type: "series"}, IsDefault: true},
	}
	CollectionConfigs = append(CollectionConfigs, tvConfigs...)

	for _, g := range tvGenreConfigs {
		CollectionConfigs = append(CollectionConfigs, CollectionConfig{
			ManifestCatalog: model.ManifestCatalog{ID: g.ID, Name: g.Name, Type: "series"},
			HasGenre:        true,
		})
	}

	CollectionConfigs = append(CollectionConfigs, CollectionConfig{
		ManifestCatalog: model.ManifestCatalog{ID: TVYearlyRankingID, Name: "豆瓣年度评分最高剧集", Type: "series"},
		HasGenre:        true,
	})

	CollectionConfigs = append(CollectionConfigs,
		CollectionConfig{ManifestCatalog: model.ManifestCatalog{ID: TmdbTrendingMovieID, Name: "TMDB 本周热门电影", Type: "movie"}, IsTmdb: true, IsDefault: true},
		CollectionConfig{ManifestCatalog: model.ManifestCatalog{ID: TmdbTrendingTvID, Name: "TMDB 本周热门剧集", Type: "series"}, IsTmdb: true, IsDefault: true},
		CollectionConfig{ManifestCatalog: model.ManifestCatalog{ID: TmdbDiscoverAnimeID, Name: "TMDB 热门动漫", Type: "series"}, IsTmdb: true, IsDefault: true},
		CollectionConfig{ManifestCatalog: model.ManifestCatalog{ID: TmdbDiscoverMovieID, Name: "TMDB 热门电影", Type: "movie"}, IsTmdb: true},
		CollectionConfig{ManifestCatalog: model.ManifestCatalog{ID: TmdbDiscoverTvID, Name: "TMDB 热门剧集", Type: "series"}, IsTmdb: true},
	)
}

var YearlyRankingConfigs []CollectionConfig

func init() {
	for _, item := range MovieYearlyRanking {
		YearlyRankingConfigs = append(YearlyRankingConfigs, CollectionConfig{
			ManifestCatalog: model.ManifestCatalog{ID: item.ID, Name: item.Name, Type: "movie"},
		})
	}
	for _, item := range TVYearlyRanking {
		YearlyRankingConfigs = append(YearlyRankingConfigs, CollectionConfig{
			ManifestCatalog: model.ManifestCatalog{ID: item.ID, Name: item.Name, Type: "series"},
		})
	}
}

func IsYearlyRankingID(id string) bool {
	_, ok := YearlyRankings[id]
	return ok
}

func GetLatestYearlyRanking(id string) *YearlyRankingItem {
	items, ok := YearlyRankings[id]
	if !ok || len(items) == 0 {
		return nil
	}
	latest := &items[0]
	for i := range items {
		if items[i].Year > latest.Year {
			latest = &items[i]
		}
	}
	return latest
}

func FindCollectionConfig(id string) *CollectionConfig {
	for i := range CollectionConfigs {
		if CollectionConfigs[i].ID == id {
			return &CollectionConfigs[i]
		}
	}
	return nil
}
