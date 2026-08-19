package handler

import (
	"context"
	"encoding/json"
	"log"
	"math/rand/v2"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"stremio-addon-douban/internal/api"
	"stremio-addon-douban/internal/collection"
	"stremio-addon-douban/internal/model"
)

func handleCatalogRequest(w http.ResponseWriter, r *http.Request, catalogID string, extra map[string]string) {
	collectionID := catalogID
	if collection.IsYearlyRankingID(catalogID) {
		latest := collection.GetLatestYearlyRanking(catalogID)
		if latest == nil {
			http.NotFound(w, r)
			return
		}
		collectionID = latest.ID
	}

	// TMDB catalog
	cc := collection.FindCollectionConfig(catalogID)
	if cc != nil && cc.IsTmdb {
		handleTmdbCatalog(w, r, cc, extra)
		return
	}

	svc := api.GetService()
	ctx := r.Context()

	// genre support
	if genre := extra["genre"]; genre != "" {
		category, err := svc.DoubanAPI.GetSubjectCollectionCategory(ctx, collectionID)
		if err == nil && category != nil {
			for _, item := range category.Items {
				if item.Name == genre {
					collectionID = item.ID
					break
				}
			}
		}
	}

	skip := 0
	if s := extra["skip"]; s != "" {
		skip, _ = strconv.Atoi(s)
	}

	data, err := svc.DoubanAPI.GetSubjectCollectionItems(ctx, collectionID, skip)
	if err != nil || data == nil {
		http.NotFound(w, r)
		return
	}

	items := data.SubjectCollectionItems
	if len(items) == 0 {
		writeJSON(w, model.CatalogResponse{Metas: []model.MetaDetail{}})
		return
	}

	metas := buildCatalogMetas(r, items)
	writeJSON(w, model.CatalogResponse{
		Metas:           metas,
		CacheMaxAge:     model.SecondsPerDay,
		StaleRevalidate: model.SecondsPerWeek,
		StaleError:      model.SecondsPerWeek,
	})
}

// 随机池固定为热门电影 + 热门剧集（原由配置 RandomCatalogIDs 控制，现已定死）
var randomCatalogIDs = []string{"movie_hot_gaia", "tv_hot"}

func handleRandomCatalog(w http.ResponseWriter, r *http.Request, extra map[string]string) {
	if s := extra["skip"]; s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			writeJSON(w, model.CatalogResponse{Metas: []model.MetaDetail{}})
			return
		}
	}

	svc := api.GetService()
	ctx := r.Context()
	uniqueMap := make(map[int]api.DoubanSubjectCollectionItem)
	for _, id := range randomCatalogIDs {
		data, err := svc.DoubanAPI.GetSubjectCollectionItems(ctx, id, 0)
		if err != nil {
			continue
		}
		for _, item := range data.SubjectCollectionItems {
			uniqueMap[int(item.ID)] = item
		}
	}

	var allItems []api.DoubanSubjectCollectionItem
	for _, item := range uniqueMap {
		allItems = append(allItems, item)
	}
	rand.Shuffle(len(allItems), func(i, j int) { allItems[i], allItems[j] = allItems[j], allItems[i] })
	if len(allItems) > 10 {
		allItems = allItems[:10]
	}

	if len(allItems) == 0 {
		writeJSON(w, model.CatalogResponse{Metas: []model.MetaDetail{}})
		return
	}

	metas := buildCatalogMetas(r, allItems)
	writeJSON(w, model.CatalogResponse{Metas: metas})
}

func handleTheaterCatalog(w http.ResponseWriter, r *http.Request, catalogID string, extra map[string]string) {
	svc := api.GetService()
	ctx := r.Context()

	items, err := svc.DoubanAPI.GetDoulistItems(ctx, catalogID, 0)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// 分页（每页 20，与其它 catalog 一致），保持 doulist 原始顺序
	skip := 0
	if s := extra["skip"]; s != "" {
		skip, _ = strconv.Atoi(s)
	}
	if skip >= len(items) {
		writeJSON(w, model.CatalogResponse{Metas: []model.MetaDetail{}})
		return
	}
	end := skip + 20
	if end > len(items) {
		end = len(items)
	}
	subjectItems := doulistItemsToSubject(items[skip:end])

	if len(subjectItems) == 0 {
		writeJSON(w, model.CatalogResponse{Metas: []model.MetaDetail{}})
		return
	}

	metas := buildCatalogMetas(r, subjectItems)
	writeJSON(w, model.CatalogResponse{
		Metas:           metas,
		CacheMaxAge:     model.SecondsPerDay,
		StaleRevalidate: model.SecondsPerWeek,
		StaleError:      model.SecondsPerWeek,
	})
}

// doulistItemsToSubject 把豆瓣片单条目转换为 subject collection 条目（年份从 subtitle 首段提取）
func doulistItemsToSubject(items []api.DoubanDoulistItem) []api.DoubanSubjectCollectionItem {
	subjectItems := make([]api.DoubanSubjectCollectionItem, 0, len(items))
	for _, item := range items {
		year := ""
		if item.Subtitle != "" {
			parts := strings.SplitN(item.Subtitle, "/", 2)
			year = strings.TrimSpace(parts[0])
		}
		subjectItems = append(subjectItems, api.DoubanSubjectCollectionItem{
			ID:           item.TargetID,
			Type:         item.Type,
			Title:        item.Title,
			CardSubtitle: item.Subtitle,
			Description:  item.Comment,
			CoverURL:     item.CoverURL,
			URL:          item.URL,
			Year:         year,
			Rating:       item.Rating,
		})
	}
	return subjectItems
}

func handleTmdbCatalog(w http.ResponseWriter, r *http.Request, catalog *collection.CollectionConfig, extra map[string]string) {
	tmdbAPI := api.GetTmdbAPI()
	ctx := r.Context()

	skip := 0
	if s := extra["skip"]; s != "" {
		skip, _ = strconv.Atoi(s)
	}
	page := skip/20 + 1

	if catalog.ID == collection.TmdbDiscoverAnimeID {
		movieGenreMap, _ := tmdbAPI.GetGenres(ctx, "movie")
		tvGenreMap, _ := tmdbAPI.GetGenres(ctx, "tv")
		movieRes, _ := tmdbAPI.Discover(ctx, "movie", map[string]string{
			"with_genres": "16", "sort_by": "popularity.desc", "vote_count_gte": "200",
		}, page)
		tvRes, _ := tmdbAPI.Discover(ctx, "tv", map[string]string{
			"with_genres": "16", "sort_by": "popularity.desc", "vote_count_gte": "200",
		}, page)

		var metas []model.MetaDetail
		if movieRes != nil {
			metas = append(metas, buildTmdbMetas(r, movieRes.Results, "movie", toGenreMap(movieGenreMap), "series")...)
		}
		if tvRes != nil {
			metas = append(metas, buildTmdbMetas(r, tvRes.Results, "series", toGenreMap(tvGenreMap), "series")...)
		}
		writeJSON(w, model.CatalogResponse{
			Metas:           metas,
			CacheMaxAge:     model.SecondsPerDay,
			StaleRevalidate: model.SecondsPerWeek,
			StaleError:      model.SecondsPerWeek,
		})
		return
	}

	tmdbType := "movie"
	if catalog.Type == "series" {
		tmdbType = "tv"
	}

	genreMap, _ := tmdbAPI.GetGenres(ctx, tmdbType)

	var data *api.TmdbTrendingResult
	switch catalog.ID {
	case collection.TmdbTrendingMovieID:
		data, _ = tmdbAPI.Trending(ctx, "movie", "week", page)
	case collection.TmdbTrendingTvID:
		data, _ = tmdbAPI.Trending(ctx, "tv", "week", page)
	case collection.TmdbDiscoverMovieID, collection.TmdbDiscoverTvID:
		data, _ = tmdbAPI.Discover(ctx, tmdbType, discoverParams(extra), page)
	}

	if data == nil || len(data.Results) == 0 {
		if data == nil {
			http.NotFound(w, r)
		} else {
			writeJSON(w, model.CatalogResponse{Metas: []model.MetaDetail{}})
		}
		return
	}

	metas := buildTmdbMetas(r, data.Results, catalog.Type, toGenreMap(genreMap), "")
	writeJSON(w, model.CatalogResponse{
		Metas:           metas,
		CacheMaxAge:     model.SecondsPerDay,
		StaleRevalidate: model.SecondsPerWeek,
		StaleError:      model.SecondsPerWeek,
	})
}

func buildCatalogMetas(r *http.Request, items []api.DoubanSubjectCollectionItem) []model.MetaDetail {
	svc := api.GetService()
	ctx := r.Context()

	doubanIDs := make([]int, len(items))
	for i, item := range items {
		doubanIDs[i] = int(item.ID)
	}
	mappingCache, missingIDs, _ := svc.FetchIDMapping(ctx, doubanIDs)

	// background fill missing mappings
	if len(missingIDs) > 0 {
		itemMap := make(map[int]api.DoubanSubjectCollectionItem)
		for _, item := range items {
			itemMap[int(item.ID)] = item
		}
		go func() {
			bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			for _, doubanID := range missingIDs {
				item, ok := itemMap[doubanID]
				if !ok {
					continue
				}
				mapping, err := svc.FindExternalID(bgCtx, api.FindIDParams{
					DoubanID: doubanID, Type: item.Type, Title: item.Title,
				})
				if err == nil && mapping != nil {
					if err := svc.PersistIDMapping(bgCtx, []api.DoubanIDMapping{*mapping}, false, "update"); err != nil {
						log.Printf("persist mapping douban:%d: %v", doubanID, err)
					}
				}
			}
		}()
	}

	// TMDB 标准分类（genre_ids）：对已映射 tmdbID 的条目并发查详情（25h 缓存），失败静默跳过
	tmdbAPI := api.GetTmdbAPI()
	genreIDs := make([][]int, len(items))
	var wg sync.WaitGroup
	for i, item := range items {
		mapping := mappingCache[int(item.ID)]
		if mapping == nil || mapping.TmdbID == nil {
			continue
		}
		tmdbType := "movie"
		if item.Type == "tv" {
			tmdbType = "tv"
		}
		wg.Add(1)
		go func(i int, tmdbType string, id int) {
			defer wg.Done()
			if detail, err := tmdbAPI.GetDetail(ctx, tmdbType, id); err == nil {
				genreIDs[i] = detail.GenreIDs
			}
		}(i, tmdbType, *mapping.TmdbID)
	}
	wg.Wait()

	metas := make([]model.MetaDetail, 0, len(items))
	for i, item := range items {
		mapping := mappingCache[int(item.ID)]
		var imdbID string
		var tmdbID int
		if mapping != nil {
			if mapping.ImdbID != nil {
				imdbID = *mapping.ImdbID
			}
			if mapping.TmdbID != nil {
				tmdbID = *mapping.TmdbID
			}
		}

		images := generateImageURLs(ctx, item, tmdbID, imdbID)

		metaType := "movie"
		if item.Type == "tv" {
			metaType = "series"
		}

		genres := []string{}
		if parts := strings.Split(item.CardSubtitle, "/"); len(parts) >= 3 {
			g := strings.TrimSpace(parts[2])
			if g != "" {
				genres = strings.Fields(g)
			}
		}

		meta := model.MetaDetail{
			ID:          "douban:" + strconv.Itoa(int(item.ID)),
			Type:        metaType,
			Name:        item.Title,
			Description: item.Description,
			Poster:      images.Poster,
			Background:  images.Background,
			Logo:        images.Logo,
			Year:        item.Year,
			Genres:      genres,
			GenreIDs:    genreIDs[i],
			Links: []model.MetaLink{
				{Name: "豆瓣评分：" + ratingStr(item.Rating), Category: "douban", URL: orDefault(item.URL, "#")},
			},
		}

		if imdbID != "" {
			meta.IMDBID = imdbID
		}
		if tmdbID != 0 {
			meta.TMDBIDStr = "tmdb:" + strconv.Itoa(tmdbID)
		}
		meta.ID = generateID(int(item.ID), imdbID, tmdbID)

		metas = append(metas, meta)
	}

	return metas
}

func buildTmdbMetas(r *http.Request, items []api.TmdbTrendingItem, metaType string, genreMap map[int]string, displayType string) []model.MetaDetail {
	tmdbType := "movie"
	if metaType == "series" {
		tmdbType = "tv"
	}

	metas := make([]model.MetaDetail, 0, len(items))
	for _, item := range items {
		genres := []string{}
		for _, gid := range item.GenreIDs {
			if name, ok := genreMap[gid]; ok {
				genres = append(genres, name)
			}
		}

		dt := displayType
		if dt == "" {
			dt = metaType
		}

		poster, background := "", ""
		if item.PosterPath != nil {
			poster = directProxyURL("https://image.tmdb.org/t/p/original" + *item.PosterPath)
		}
		if item.BackdropPath != nil {
			background = directProxyURL("https://image.tmdb.org/t/p/original" + *item.BackdropPath)
		}

		meta := model.MetaDetail{
			ID:          "tmdb:" + tmdbType + ":" + strconv.Itoa(int(item.ID)),
			Type:        dt,
			Name:        item.Title,
			Description: item.Overview,
			Poster:      poster,
			Background:  background,
			Year:        item.Year,
			Genres:      genres,
			GenreIDs:    item.GenreIDs,
			TMDBID:      item.ID,
			Links: []model.MetaLink{
				{
					Name:     "TMDB 评分：" + strconv.FormatFloat(item.VoteAverage, 'f', 1, 64),
					Category: "tmdb",
					URL:      "https://www.themoviedb.org/" + tmdbType + "/" + strconv.Itoa(int(item.ID)),
				},
			},
		}

		meta.TMDBIDStr = "tmdb:" + strconv.Itoa(int(item.ID))

		metas = append(metas, meta)
	}
	return metas
}

func parseExtra(extraStr string, r *http.Request) map[string]string {
	result := make(map[string]string)
	if extraStr != "" {
		for _, pair := range strings.Split(extraStr, "&") {
			if kv := strings.SplitN(pair, "=", 2); len(kv) == 2 {
				val := kv[1]
				if decoded, err := url.QueryUnescape(val); err == nil {
					val = decoded
				}
				result[kv[0]] = val
			}
		}
	}
	// also check query params
	for key := range result {
		if v := r.URL.Query().Get(key); v != "" {
			result[key] = v
		}
	}
	if skip := r.URL.Query().Get("skip"); skip != "" {
		result["skip"] = skip
	}
	if genre := r.URL.Query().Get("genre"); genre != "" {
		result["genre"] = genre
	}
	return result
}

func toGenreMap(gm *api.TmdbGenreMap) map[int]string {
	m := make(map[int]string)
	if gm != nil {
		for _, g := range gm.Genres {
			m[g.ID] = g.Name
		}
	}
	return m
}

// discoverParamAllowlist tmdb_discover catalog 可透传的 Discover 查询参数白名单，
// 平台（with_networks/with_watch_providers/watch_region）、类型、排序与日期过滤
var discoverParamAllowlist = map[string]bool{
	"with_networks":            true,
	"with_watch_providers":     true,
	"watch_region":             true,
	"with_genres":              true,
	"without_genres":           true,
	"sort_by":                  true,
	"vote_count.gte":           true,
	"primary_release_date.lte": true,
	"first_air_date.lte":       true,
}

func discoverParams(extra map[string]string) map[string]string {
	p := make(map[string]string)
	for k, v := range extra {
		if discoverParamAllowlist[k] && v != "" {
			p[k] = v
		}
	}
	return p
}

func ratingStr(r *api.DoubanRating) string {
	if r == nil {
		return "N/A"
	}
	return strconv.FormatFloat(r.Value, 'f', -1, 64)
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func generateID(doubanID int, imdbID string, tmdbID int) string {
	if tmdbID != 0 {
		return "tmdb:" + strconv.Itoa(tmdbID)
	}
	if imdbID != "" {
		return imdbID
	}
	return "douban:" + strconv.Itoa(doubanID)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
