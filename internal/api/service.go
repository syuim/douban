package api

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"stremio-addon-douban/internal/db"
)

type DoubanIDMapping struct {
	DoubanID   int
	TmdbID     *int
	ImdbID     *string
	Calibrated bool
}

type Service struct {
	DoubanAPI *DoubanAPI
	TmdbAPI   *TmdbAPI
}

var serviceInstance *Service

func GetService() *Service {
	if serviceInstance == nil {
		serviceInstance = &Service{
			DoubanAPI: NewDoubanAPI(),
			TmdbAPI:   GetTmdbAPI(),
		}
	}
	return serviceInstance
}

func (s *Service) FetchIDMapping(ctx context.Context, doubanIDs []int) (map[int]*DoubanIDMapping, []int, error) {
	if len(doubanIDs) == 0 {
		return map[int]*DoubanIDMapping{}, nil, nil
	}

	database, err := db.GetDB()
	if err != nil {
		return nil, nil, err
	}

	mappingCache := make(map[int]*DoubanIDMapping)
	mappedIDs := make(map[int]bool)

	// batch query
	placeholders := make([]string, len(doubanIDs))
	args := make([]any, len(doubanIDs))
	for i, id := range doubanIDs {
		placeholders[i] = "?"
		args[i] = id
	}
	query := fmt.Sprintf("SELECT douban_id, tmdb_id, imdb_id FROM douban_mapping WHERE douban_id IN (%s)",
		strings.Join(placeholders, ","))

	rows, err := database.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var m DoubanIDMapping
		var tmdbID sql.NullInt64
		var imdbID sql.NullString
		if err := rows.Scan(&m.DoubanID, &tmdbID, &imdbID); err != nil {
			continue
		}
		if tmdbID.Valid {
			v := int(tmdbID.Int64)
			m.TmdbID = &v
		}
		if imdbID.Valid {
			m.ImdbID = &imdbID.String
		}
		if m.ImdbID != nil || m.TmdbID != nil {
			mappingCache[m.DoubanID] = &m
			mappedIDs[m.DoubanID] = true
		}
	}

	var missingIDs []int
	for _, id := range doubanIDs {
		if !mappedIDs[id] {
			missingIDs = append(missingIDs, id)
		}
	}

	return mappingCache, missingIDs, nil
}

func (s *Service) FetchMappingOne(ctx context.Context, doubanID int) (*DoubanIDMapping, error) {
	database, err := db.GetDB()
	if err != nil {
		return nil, err
	}

	var m DoubanIDMapping
	var tmdbID sql.NullInt64
	var imdbID sql.NullString
	err = database.QueryRowContext(ctx,
		"SELECT douban_id, tmdb_id, imdb_id FROM douban_mapping WHERE douban_id = ?", doubanID).
		Scan(&m.DoubanID, &tmdbID, &imdbID)
	if err != nil {
		return nil, err
	}
	if tmdbID.Valid {
		v := int(tmdbID.Int64)
		m.TmdbID = &v
	}
	if imdbID.Valid {
		m.ImdbID = &imdbID.String
	}
	return &m, nil
}

func (s *Service) PersistIDMapping(ctx context.Context, mappings []DoubanIDMapping, skipNil bool, mode string) error {
	database, err := db.GetDB()
	if err != nil {
		return err
	}

	for _, m := range mappings {
		if skipNil && m.ImdbID == nil && m.TmdbID == nil {
			continue
		}

		now := time.Now().UnixMilli()
		var execErr error
		if mode == "ignore" {
			_, execErr = database.ExecContext(ctx, `
				INSERT OR IGNORE INTO douban_mapping (douban_id, tmdb_id, imdb_id, calibrated, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?)`,
				m.DoubanID, m.TmdbID, m.ImdbID, m.Calibrated, now, now)
		} else {
			_, execErr = database.ExecContext(ctx, `
				INSERT INTO douban_mapping (douban_id, tmdb_id, imdb_id, calibrated, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?)
				ON CONFLICT(douban_id) DO UPDATE SET
					tmdb_id = COALESCE(excluded.tmdb_id, douban_mapping.tmdb_id),
					imdb_id = COALESCE(excluded.imdb_id, douban_mapping.imdb_id),
					updated_at = excluded.updated_at
				WHERE douban_mapping.calibrated IS NOT true`,
				m.DoubanID, m.TmdbID, m.ImdbID, m.Calibrated, now, now)
		}
		if execErr != nil {
			return fmt.Errorf("persist mapping %d: %w", m.DoubanID, execErr)
		}
	}
	return nil
}

type FindIDParams struct {
	DoubanID int
	Type     string
	Title    string
}

func (s *Service) FindExternalID(ctx context.Context, params FindIDParams) (*DoubanIDMapping, error) {
	result := &DoubanIDMapping{
		DoubanID: params.DoubanID,
	}

	// 1. 豆瓣详情页 IMDb → TMDB /find 精确解析
	desc, _ := s.DoubanAPI.GetSubjectDetailDesc(ctx, params.DoubanID)
	var imdbID string
	if desc != nil {
		imdbID = desc["IMDb"]
	}
	if imdbID != "" {
		result.ImdbID = &imdbID
		result.TmdbID = s.findTmdbByImdb(ctx, imdbID, params.Type)
	}

	// 2. 无 tmdb 时用 TMDB 标题搜索兜底
	if result.TmdbID == nil {
		title := params.Title
		if title == "" && desc != nil {
			title = desc["title"]
		}
		if title != "" {
			result.TmdbID = s.findIDWithTmdbSearchTitle(ctx, params.Type, title)
		}
	}

	return result, nil
}

// findTmdbByImdb 用 TMDB /find 按 IMDb ID 精确解析 tmdb id。
// 按条目类型取对应结果：剧集条目误用 TMDB movie id 会导致后续 tv/{id} 全量 404。
func (s *Service) findTmdbByImdb(ctx context.Context, imdbID, mediaType string) *int {
	find, err := s.TmdbAPI.FindByID(ctx, imdbID, "imdb_id")
	if err != nil || find == nil {
		return nil
	}

	switch mediaType {
	case "tv":
		if len(find.TVResults) == 1 {
			return intPtr(find.TVResults[0].ID)
		}
	case "movie":
		if len(find.MovieResults) == 1 {
			return intPtr(find.MovieResults[0].ID)
		}
	default:
		var tmdbID *int
		if len(find.MovieResults) == 1 {
			tmdbID = intPtr(find.MovieResults[0].ID)
		}
		if len(find.TVResults) == 1 {
			tmdbID = intPtr(find.TVResults[0].ID)
		}
		return tmdbID
	}
	return nil
}

func intPtr(n int) *int { return &n }

var seasonRe = regexp.MustCompile(`\s*（?第?[0-9一二三四五六七八九十百零]+季）?\s*`)
var partRe = regexp.MustCompile(`\s*（?第[0-9一二三四五六七八九十百零]+部）?\s*$`)

func cleanSearchTitle(title string) string {
	return strings.TrimSpace(seasonRe.ReplaceAllString(title, ""))
}

// cleanPartTitle 去掉结尾的「第X部」：豆瓣常把续作拆成独立条目，TMDB 合并在同一剧集下。
func cleanPartTitle(title string) string {
	return strings.TrimSpace(partRe.ReplaceAllString(title, ""))
}

// findIDWithTmdbSearchTitle 用 TMDB 标题搜索解析 tmdb id。
// 豆瓣标题常带季名/部数/副标题后缀（「第二季」「第二部」「：夏篇」「～早春养生篇～」），
// 完整标题在 TMDB 搜不到，按 原题 → 去季名 → 去部数 → 去副标题 的保序候选逐个尝试；
// 每条查询仅接受唯一结果或名称精确匹配的唯一项，避免误映射。
func (s *Service) findIDWithTmdbSearchTitle(ctx context.Context, mediaType, title string) *int {
	searchType := "movie"
	if mediaType == "tv" {
		searchType = "tv"
	}

	var candidates []string
	seen := map[string]bool{}
	addCandidate := func(t string) {
		t = strings.TrimSpace(t)
		if t == "" || seen[t] {
			return
		}
		seen[t] = true
		candidates = append(candidates, t)
	}
	addCandidate(title)
	cleaned := cleanSearchTitle(title)
	addCandidate(cleaned)
	part := cleanPartTitle(cleaned)
	addCandidate(part)
	for _, t := range []string{title, cleaned, part} {
		if i := strings.IndexAny(t, "：:～~"); i > 0 && utf8.RuneCountInString(t[:i]) >= 2 {
			addCandidate(t[:i])
		}
	}

	for _, q := range candidates {
		res, err := s.TmdbAPI.Search(ctx, searchType, map[string]string{
			"query":    q,
			"language": "zh-CN",
		})
		if err != nil || res == nil || len(res.Results) == 0 {
			continue
		}
		if len(res.Results) == 1 {
			return intPtr(res.Results[0].ID)
		}
		var matches []TmdbTrendingItem
		for i := range res.Results {
			r := &res.Results[i]
			if seen[r.Title] || seen[r.Name] {
				matches = append(matches, *r)
			}
		}
		if len(matches) == 1 {
			return intPtr(matches[0].ID)
		}
	}
	return nil
}
