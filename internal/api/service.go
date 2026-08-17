package api

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"

	"stremio-addon-douban/internal/db"
)

type DoubanIDMapping struct {
	DoubanID   int
	TmdbID     *int
	ImdbID     *string
	TraktID    *int
	Calibrated bool
}

type Service struct {
	DoubanAPI *DoubanAPI
	TraktAPI  *TraktAPI
	TmdbAPI   *TmdbAPI
}

var serviceInstance *Service

func GetService() *Service {
	if serviceInstance == nil {
		serviceInstance = &Service{
			DoubanAPI: NewDoubanAPI(),
			TraktAPI:  NewTraktAPI(),
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
	query := fmt.Sprintf("SELECT douban_id, tmdb_id, imdb_id, trakt_id FROM douban_mapping WHERE douban_id IN (%s)",
		strings.Join(placeholders, ","))

	rows, err := database.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var m DoubanIDMapping
		var tmdbID, traktID sql.NullInt64
		var imdbID sql.NullString
		if err := rows.Scan(&m.DoubanID, &tmdbID, &imdbID, &traktID); err != nil {
			continue
		}
		if tmdbID.Valid {
			v := int(tmdbID.Int64)
			m.TmdbID = &v
		}
		if imdbID.Valid {
			m.ImdbID = &imdbID.String
		}
		if traktID.Valid {
			v := int(traktID.Int64)
			m.TraktID = &v
		}
		if m.ImdbID != nil || m.TmdbID != nil || m.TraktID != nil {
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
	var tmdbID, traktID sql.NullInt64
	var imdbID sql.NullString
	err = database.QueryRowContext(ctx,
		"SELECT douban_id, tmdb_id, imdb_id, trakt_id FROM douban_mapping WHERE douban_id = ?", doubanID).
		Scan(&m.DoubanID, &tmdbID, &imdbID, &traktID)
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
	if traktID.Valid {
		v := int(traktID.Int64)
		m.TraktID = &v
	}
	return &m, nil
}

func (s *Service) PersistIDMapping(ctx context.Context, mappings []DoubanIDMapping, skipNil bool, mode string) error {
	database, err := db.GetDB()
	if err != nil {
		return err
	}

	for _, m := range mappings {
		if skipNil && m.ImdbID == nil && m.TmdbID == nil && m.TraktID == nil {
			continue
		}

		now := time.Now().UnixMilli()
		if mode == "ignore" {
			database.ExecContext(ctx, `
				INSERT OR IGNORE INTO douban_mapping (douban_id, tmdb_id, imdb_id, trakt_id, calibrated, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?)`,
				m.DoubanID, m.TmdbID, m.ImdbID, m.TraktID, m.Calibrated, now, now)
		} else {
			database.ExecContext(ctx, `
				INSERT INTO douban_mapping (douban_id, tmdb_id, imdb_id, trakt_id, calibrated, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?)
				ON CONFLICT(douban_id) DO UPDATE SET
					tmdb_id = COALESCE(excluded.tmdb_id, douban_mapping.tmdb_id),
					imdb_id = COALESCE(excluded.imdb_id, douban_mapping.imdb_id),
					trakt_id = COALESCE(excluded.trakt_id, douban_mapping.trakt_id),
					updated_at = excluded.updated_at
				WHERE douban_mapping.calibrated IS NOT true`,
				m.DoubanID, m.TmdbID, m.ImdbID, m.TraktID, m.Calibrated, now, now)
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

	// 1. Try Douban detail desc for IMDb ID, then TMDB /find as the primary direct lookup.
	desc, _ := s.DoubanAPI.GetSubjectDetailDesc(ctx, params.DoubanID)
	var imdbID string
	if desc != nil {
		imdbID = desc["IMDb"]
	}
	if imdbID != "" {
		result.ImdbID = &imdbID
		if m := s.findWithImdb(ctx, imdbID); m != nil {
			result.TraktID = m.TraktID
			result.TmdbID = m.TmdbID
			result.ImdbID = m.ImdbID
		}
	}

	// 2. If no IMDb, search Trakt by title
	if result.ImdbID == nil {
		title := params.Title
		if title == "" && desc != nil {
			title = desc["title"]
		}
		if title != "" {
			traktIDs, err := s.findIDWithTraktSearchTitle(ctx, params.Type, title, "", "")
			if err == nil && traktIDs != nil {
				result.TraktID = traktIDs.TraktID
				result.TmdbID = traktIDs.TmdbID
				result.ImdbID = traktIDs.ImdbID
			}
		}
	}

	return result, nil
}

// FindByImdbID resolves a known IMDb ID through TMDB first and Trakt as a
// secondary source, preferring to keep Trakt ID when both are available.
func (s *Service) FindByImdbID(ctx context.Context, imdbID string) *DoubanIDMapping {
	if imdbID == "" {
		return nil
	}
	m := s.findWithImdb(ctx, imdbID)
	if m == nil {
		return nil
	}
	return &DoubanIDMapping{
		TmdbID:  m.TmdbID,
		ImdbID:  m.ImdbID,
		TraktID: m.TraktID,
	}
}

// FindByTitle resolves an ID by Trakt title search with conservative
// disambiguation: unique match, unique cleaned name, unique original title,
// or unique year match for movies.
func (s *Service) FindByTitle(ctx context.Context, mediaType, title, originalTitle, year string) *DoubanIDMapping {
	m, err := s.findIDWithTraktSearchTitle(ctx, mediaType, title, originalTitle, year)
	if err != nil || m == nil {
		return nil
	}
	return &DoubanIDMapping{
		TmdbID:  m.TmdbID,
		ImdbID:  m.ImdbID,
		TraktID: m.TraktID,
	}
}

func (s *Service) findWithImdb(ctx context.Context, imdbID string) *IDMapping {
	m := &IDMapping{ImdbID: &imdbID}

	// Direct TMDB lookup by IMDb ID is the most reliable fallback.
	if find, err := s.TmdbAPI.FindByID(ctx, imdbID, "imdb_id"); err == nil && find != nil {
		if len(find.MovieResults) == 1 {
			m.TmdbID = intPtr(find.MovieResults[0].ID)
		}
		if len(find.TVResults) == 1 {
			m.TmdbID = intPtr(find.TVResults[0].ID)
		}
	}

	// Trakt lookup can provide Trakt ID even when TMDB does not.
	data, err := s.TraktAPI.SearchByImdbID(ctx, imdbID)
	if err == nil && len(data) > 0 {
		var trakt *IDMapping
		if m.TmdbID != nil {
			trakt = findTraktResultByTmdb(s, data, *m.TmdbID, &imdbID)
		}
		if trakt == nil {
			trakt = mappingFromTraktResult(s, &data[0], &imdbID)
		}
		if trakt != nil {
			if m.TmdbID == nil {
				m.TmdbID = trakt.TmdbID
			}
			m.TraktID = trakt.TraktID
			m.ImdbID = trakt.ImdbID
		}
	}

	if m.TmdbID == nil && m.TraktID == nil {
		return nil
	}
	return m
}

func findTraktResultByTmdb(s *Service, results []TraktSearchResult, tmdbID int, fallbackImdb *string) *IDMapping {
	for i := range results {
		ids, ok := s.TraktAPI.GetSearchResultField(&results[i], "ids").(TraktIDs)
		if !ok || ids.Tmdb == nil || *ids.Tmdb != tmdbID {
			continue
		}
		return mappingFromTraktResult(s, &results[i], fallbackImdb)
	}
	return nil
}

func mappingFromTraktResult(s *Service, result *TraktSearchResult, fallbackImdb *string) *IDMapping {
	ids, ok := s.TraktAPI.GetSearchResultField(result, "ids").(TraktIDs)
	if !ok {
		return nil
	}
	m := s.TraktAPI.FormatIDsToIDMapping(&ids)
	if m == nil {
		return nil
	}
	if m.ImdbID == nil && fallbackImdb != nil {
		m.ImdbID = fallbackImdb
	}
	return m
}

func intPtr(n int) *int { return &n }

var seasonRe = regexp.MustCompile(`\s*（?第?[0-9一二三四五六七八九十百零]+季）?\s*`)

func cleanSearchTitle(title string) string {
	return strings.TrimSpace(seasonRe.ReplaceAllString(title, ""))
}

func (s *Service) findIDWithTraktSearchTitle(ctx context.Context, mediaType, title, originalTitle, year string) (*IDMapping, error) {
	traktType := "movie"
	if mediaType == "tv" {
		traktType = "show"
	}

	data, err := s.TraktAPI.Search(ctx, traktType, title)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil
	}
	if len(data) == 1 {
		return mappingFromTraktResult(s, &data[0], nil), nil
	}

	// match by title or cleaned title
	titleSet := map[string]bool{title: true}
	cleaned := cleanSearchTitle(title)
	if cleaned != "" && cleaned != title {
		titleSet[cleaned] = true
	}

	var nameMatches []TraktSearchResult
	for _, r := range data {
		traktTitle, _ := s.TraktAPI.GetSearchResultField(&r, "title").(string)
		traktOrigTitle, _ := s.TraktAPI.GetSearchResultField(&r, "original_title").(string)
		if titleSet[traktTitle] || titleSet[traktOrigTitle] {
			nameMatches = append(nameMatches, r)
		}
	}
	if len(nameMatches) == 1 {
		return mappingFromTraktResult(s, &nameMatches[0], nil), nil
	}

	// fallback: unique original title match
	if originalTitle != "" {
		var origMatches []TraktSearchResult
		for i := range data {
			ot, _ := s.TraktAPI.GetSearchResultField(&data[i], "original_title").(string)
			if ot == originalTitle {
				origMatches = append(origMatches, data[i])
			}
		}
		if len(origMatches) == 1 {
			return mappingFromTraktResult(s, &origMatches[0], nil), nil
		}
	}

	// fallback: unique year match for movies
	if mediaType == "movie" && year != "" {
		var yearMatches []TraktSearchResult
		for i := range data {
			y, _ := s.TraktAPI.GetSearchResultField(&data[i], "year").(int)
			if fmt.Sprintf("%d", y) == year {
				yearMatches = append(yearMatches, data[i])
			}
		}
		if len(yearMatches) == 1 {
			return mappingFromTraktResult(s, &yearMatches[0], nil), nil
		}
	}

	return nil, nil
}
