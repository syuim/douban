package cron

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"stremio-addon-douban/internal/api"
	"stremio-addon-douban/internal/collection"
	"stremio-addon-douban/internal/db"
)

const (
	warmupItemsPerCatalog = 30
	calibrateCooldownMs   = 60 * 60 * 1000
	calibrateBatchSize    = 50
)

var isRunning atomic.Bool

func RunScheduledTask() {
	if !isRunning.CompareAndSwap(false, true) {
		return
	}
	defer isRunning.Store(false)

	ctx := context.Background()

	log.Println("[cron] task start")
	api.CleanExpiredCache()

	warmupCatalogs(ctx)
	calibrateMappings(ctx)
	log.Println("[cron] task done")
}

func warmupCatalogs(ctx context.Context) {
	database, err := db.GetDB()
	if err != nil {
		return
	}

	// 固定预热全部豆瓣 catalog（含年度榜单具体年份）与全部剧场，不再依赖配置
	catalogIDSet := make(map[string]bool)
	for _, c := range collection.CollectionConfigs {
		if !c.IsTmdb {
			catalogIDSet[c.ID] = true
		}
	}
	for _, c := range collection.YearlyRankingConfigs {
		catalogIDSet[c.ID] = true
	}

	theaterIDSet := make(map[string]bool)
	for _, t := range collection.TheaterConfigs {
		theaterIDSet[t.ID] = true
	}

	svc := api.GetService()

	if len(catalogIDSet) > 0 {
		for catalogID := range catalogIDSet {
			if cc := collection.FindCollectionConfig(catalogID); cc != nil && cc.IsTmdb {
				continue
			}

			collectionID := catalogID
			if collection.IsYearlyRankingID(catalogID) {
				latest := collection.GetLatestYearlyRanking(catalogID)
				if latest == nil {
					continue
				}
				collectionID = latest.ID
			}

			page1, err := svc.DoubanAPI.GetSubjectCollectionItems(ctx, collectionID, 0)
			if err != nil {
				var apiErr *api.APIError
				if errors.As(err, &apiErr) && apiErr.Status == 404 {
					recordCollectionFailure(ctx, database, catalogID)
				}
				continue
			}
			clearCollectionFailure(ctx, database, catalogID)

			var items []api.DoubanSubjectCollectionItem
			items = append(items, page1.SubjectCollectionItems...)

			if warmupItemsPerCatalog > api.DoubanPageSize {
				page2, err := svc.DoubanAPI.GetSubjectCollectionItems(ctx, collectionID, api.DoubanPageSize)
				if err == nil {
					items = append(items, page2.SubjectCollectionItems...)
				}
			}

			if len(items) > warmupItemsPerCatalog {
				items = items[:warmupItemsPerCatalog]
			}
			if len(items) == 0 {
				continue
			}

			fillIDMappings(ctx, svc, items)
		}
		log.Printf("[cron] warmup: done, %d catalogs", len(catalogIDSet))
	}

	if len(theaterIDSet) > 0 {
		for theaterID := range theaterIDSet {
			items, err := svc.DoubanAPI.GetDoulistItems(ctx, theaterID, 1)
			if err != nil || len(items) == 0 {
				continue
			}
			subjectItems := make([]api.DoubanSubjectCollectionItem, 0, len(items))
			for _, item := range items {
				subjectItems = append(subjectItems, api.DoubanSubjectCollectionItem{
					ID:    item.TargetID,
					Type:  item.Type,
					Title: item.Title,
				})
			}
			fillIDMappings(ctx, svc, subjectItems)
		}
		log.Printf("[cron] warmup: done, %d theaters", len(theaterIDSet))
	}
}

// fillIDMappings 为缺失映射的条目补齐 douban→tmdb/imdb 映射
func fillIDMappings(ctx context.Context, svc *api.Service, items []api.DoubanSubjectCollectionItem) {
	doubanIDs := make([]int, len(items))
	for i, item := range items {
		doubanIDs[i] = int(item.ID)
	}
	_, missingIDs, err := svc.FetchIDMapping(ctx, doubanIDs)
	if err != nil {
		return
	}

	var newMappings []api.DoubanIDMapping
	for _, doubanID := range missingIDs {
		var item *api.DoubanSubjectCollectionItem
		for i := range items {
			if int(items[i].ID) == doubanID {
				item = &items[i]
				break
			}
		}
		if item == nil {
			continue
		}
		mapping, err := svc.FindExternalID(ctx, api.FindIDParams{
			DoubanID: doubanID, Type: item.Type, Title: item.Title,
		})
		if err == nil && mapping != nil {
			newMappings = append(newMappings, *mapping)
		}
	}

	if len(newMappings) > 0 {
		svc.PersistIDMapping(ctx, newMappings, false, "update")
	}
}

func calibrateMappings(ctx context.Context) {
	database, err := db.GetDB()
	if err != nil {
		log.Printf("[cron] calibrate: db error: %v", err)
		return
	}

	retryThreshold := time.Now().UnixMilli() - calibrateCooldownMs
	rows, err := database.QueryContext(ctx, `
		SELECT douban_id, imdb_id FROM douban_mapping
		WHERE tmdb_id IS NULL
			AND (calibrated IS NOT true OR calibrated IS NULL)
			AND (last_attempted_at IS NULL OR last_attempted_at < ?)
		LIMIT ?`, retryThreshold, calibrateBatchSize)
	if err != nil {
		log.Printf("[cron] calibrate: query error: %v", err)
		return
	}
	defer rows.Close()

	type mappingItem struct {
		DoubanID int
		ImdbID   sql.NullString
	}
	var allData []mappingItem
	for rows.Next() {
		var item mappingItem
		if err := rows.Scan(&item.DoubanID, &item.ImdbID); err != nil {
			log.Printf("[cron] calibrate: scan error: %v", err)
			continue
		}
		allData = append(allData, item)
	}
	if err := rows.Err(); err != nil {
		log.Printf("[cron] calibrate: rows error: %v", err)
	}

	log.Printf("[cron] calibrate: %d candidates", len(allData))
	if len(allData) == 0 {
		return
	}

	svc := api.GetService()

	var results []api.DoubanIDMapping
	var processedIDs []int
	var updateErrs int

	for _, item := range allData {
		processedIDs = append(processedIDs, item.DoubanID)
		result := calibrateOne(ctx, svc, item.DoubanID, item.ImdbID)
		if result.mapping != nil {
			results = append(results, *result.mapping)
		} else {
			log.Printf("[cron] calibrate: douban %d unmapped, reason=%s candidates=%s", item.DoubanID, result.reason, result.candidates)
		}
		if err := updateCalibrateAttempt(ctx, database, item.DoubanID); err != nil && updateErrs == 0 {
			log.Printf("[cron] calibrate: update error detail: %v", err)
		}
	}

	if len(results) > 0 {
		if err := svc.PersistIDMapping(ctx, results, true, "update"); err != nil {
			log.Printf("[cron] calibrate: persist error: %v", err)
		}
	}

	log.Printf("[cron] calibrate: done, matched %d/%d, update errors %d", len(results), len(processedIDs), updateErrs)
}

type calibrateResult struct {
	mapping    *api.DoubanIDMapping
	reason     string
	candidates string
}

func updateCalibrateAttempt(ctx context.Context, database *sql.DB, doubanID int) error {
	now := time.Now().UnixMilli()
	_, err := database.ExecContext(ctx,
		"UPDATE douban_mapping SET last_attempted_at = ? WHERE douban_id = ?",
		now, doubanID)
	return err
}

func calibrateOne(ctx context.Context, svc *api.Service, doubanID int, imdbID sql.NullString) calibrateResult {
	var doubanDetail *api.DoubanSubjectDetail

	if imdbID.Valid && imdbID.String != "" {
		if m := svc.FindByImdbID(ctx, imdbID.String); m != nil {
			m.DoubanID = doubanID
			m.Calibrated = true
			return calibrateResult{mapping: m}
		}
	}

	if doubanDetail == nil {
		doubanDetail, _ = svc.DoubanAPI.GetSubjectDetail(ctx, doubanID)
	}
	if doubanDetail == nil {
		return calibrateResult{reason: "豆瓣详情获取失败"}
	}

	traktType := "movie"
	if doubanDetail.Type == "tv" {
		traktType = "show"
	}

	if m := svc.FindByTitle(ctx, doubanDetail.Type, doubanDetail.Title, doubanDetail.OriginalTitle, doubanDetail.Year); m != nil {
		m.DoubanID = doubanID
		m.Calibrated = true
		return calibrateResult{mapping: m}
	}

	results, err := svc.TraktAPI.Search(ctx, traktType, doubanDetail.Title)
	if err != nil {
		return calibrateResult{reason: "Trakt 搜索失败: " + err.Error()}
	}
	return calibrateResult{
		reason:     "标题匹配无唯一结果",
		candidates: traktCandidatesJSON(results, doubanDetail.Title, doubanDetail.OriginalTitle, doubanDetail.Year),
	}
}

func traktCandidatesJSON(results []api.TraktSearchResult, title, originalTitle, year string) string {
	type candidate struct {
		Type    string  `json:"type"`
		TraktID *int    `json:"trakt_id,omitempty"`
		TmdbID  *int    `json:"tmdb_id,omitempty"`
		ImdbID  string  `json:"imdb_id,omitempty"`
		Title   string  `json:"title,omitempty"`
		Year    int     `json:"year,omitempty"`
		Score   float64 `json:"score"`
	}
	out := make([]candidate, 0, len(results))
	for i := range results {
		r := &results[i]
		if !candidateRelevant(r, title, originalTitle, year) {
			continue
		}
		ids, _ := traktIDs(r)
		if ids == nil {
			continue
		}
		c := candidate{
			Type:    r.Type,
			TraktID: ids.Trakt,
			TmdbID:  ids.Tmdb,
			Score:   r.Score,
		}
		if ids.Imdb != nil {
			c.ImdbID = *ids.Imdb
		}
		if title, _ := traktTitle(r); title != "" {
			c.Title = title
		}
		if year, _ := traktYear(r); year != 0 {
			c.Year = year
		}
		out = append(out, c)
	}
	if len(out) == 0 {
		return ""
	}
	data, _ := json.Marshal(out)
	return string(data)
}

func candidateRelevant(r *api.TraktSearchResult, title, originalTitle, year string) bool {
	candTitle, _ := traktTitle(r)
	candOrig, _ := traktOriginalTitle(r)
	if candOrig == "" {
		candOrig = candTitle
	}
	if title != "" && (candTitle == title || candOrig == title) {
		return true
	}
	if originalTitle != "" && candOrig == originalTitle {
		return true
	}
	if year != "" {
		if y, _ := traktYear(r); fmt.Sprintf("%d", y) == year {
			return true
		}
	}
	return false
}

func traktIDs(r *api.TraktSearchResult) (*api.TraktIDs, bool) {
	svc := api.GetService()
	ids, ok := svc.TraktAPI.GetSearchResultField(r, "ids").(api.TraktIDs)
	if !ok {
		return nil, false
	}
	return &ids, true
}

func traktTitle(r *api.TraktSearchResult) (string, bool) {
	svc := api.GetService()
	title, ok := svc.TraktAPI.GetSearchResultField(r, "title").(string)
	return title, ok
}

func traktOriginalTitle(r *api.TraktSearchResult) (string, bool) {
	svc := api.GetService()
	title, ok := svc.TraktAPI.GetSearchResultField(r, "original_title").(string)
	return title, ok
}

func traktYear(r *api.TraktSearchResult) (int, bool) {
	svc := api.GetService()
	year, ok := svc.TraktAPI.GetSearchResultField(r, "year").(int)
	return year, ok
}

const maxFailDays = 3

func recordCollectionFailure(ctx context.Context, database *sql.DB, catalogID string) {
	today := time.Now().Format("2006-01-02")

	var failCount int
	var lastDate string
	err := database.QueryRowContext(ctx,
		"SELECT fail_count, last_fail_date FROM collection_failures WHERE collection_id = ?", catalogID,
	).Scan(&failCount, &lastDate)

	if err == sql.ErrNoRows {
		_, _ = database.ExecContext(ctx,
			"INSERT INTO collection_failures (collection_id, fail_count, last_fail_date) VALUES (?, 1, ?)",
			catalogID, today)
		return
	}
	if err != nil {
		return
	}

	if lastDate == today {
		return
	}

	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	if lastDate == yesterday {
		failCount++
	} else {
		failCount = 1
	}

	if failCount >= maxFailDays {
		// 无配置系统可移除，仅保留失败记录并标记当天，避免每小时重复统计
		if _, err := database.ExecContext(ctx,
			"UPDATE collection_failures SET fail_count = ?, last_fail_date = ? WHERE collection_id = ?",
			failCount, today, catalogID); err != nil {
			log.Printf("[cron] warmup: mark dead collection %s failed: %v", catalogID, err)
		}
		log.Printf("[cron] warmup: collection %s failed %d consecutive days", catalogID, failCount)
		return
	}

	_, _ = database.ExecContext(ctx,
		"UPDATE collection_failures SET fail_count = ?, last_fail_date = ? WHERE collection_id = ?",
		failCount, today, catalogID)
}

func clearCollectionFailure(ctx context.Context, database *sql.DB, catalogID string) {
	_, _ = database.ExecContext(ctx, "DELETE FROM collection_failures WHERE collection_id = ?", catalogID)
}
