// Package failwatch 观察豆瓣 collection/doulist 的上游 404：失败次数达到
// maxFailCount 即停用直通（请求不再回源）。记录量小，复用 api_cache 表
// （key 带 failwatch: 前缀），不建新表；expires_at 设远避免被每日缓存清理。
package failwatch

import (
	"context"
	"database/sql"
	"log"
	"time"

	"stremio-addon-douban/internal/db"
)

const (
	KindCollection = "c"
	KindDoulist    = "d"
)

const maxFailCount = 3

// 10 年：仅防被 CleanExpiredCache 清理；人工删除该行即恢复
const ttlDecade = int64(10 * 365 * 24 * 3600 * 1000)

func cacheKey(kind, id string) string { return "failwatch:" + kind + ":" + id }

// Record 记录一次 404（SQL 原子累加）。同一失败源连续 maxFailCount 次
// 无成功即停用（Disabled 返回 true）；Clear 在成功请求后调用，归零重计。
func Record(ctx context.Context, kind, id string) {
	database, err := db.GetDB()
	if err != nil {
		log.Printf("[failwatch] db error: %v", err)
		return
	}
	k := cacheKey(kind, id)
	nowMs := time.Now().UnixMilli()
	if _, err := database.ExecContext(ctx, `
		INSERT INTO api_cache (key, value, expires_at, created_at) VALUES (?, '1', ?, ?)
		ON CONFLICT(key) DO UPDATE SET value = CAST(value AS INTEGER) + 1`,
		k, nowMs+ttlDecade, nowMs); err != nil {
		log.Printf("[failwatch] %s 记录失败: %v", k, err)
		return
	}
	var count int
	if err := database.QueryRowContext(ctx,
		"SELECT CAST(value AS INTEGER) FROM api_cache WHERE key = ?", k).Scan(&count); err == nil && count == maxFailCount {
		log.Printf("[failwatch] %s 连续 %d 次 404，已停用（删 api_cache 中该行可恢复）", k, count)
	}
}

// Clear 成功后清除失败记录（失败计数归零即恢复观察）。
func Clear(ctx context.Context, kind, id string) {
	database, err := db.GetDB()
	if err != nil {
		return
	}
	if _, err := database.ExecContext(ctx, "DELETE FROM api_cache WHERE key = ?", cacheKey(kind, id)); err != nil {
		log.Printf("[failwatch] %s 清除失败: %v", cacheKey(kind, id), err)
	}
}

// Disabled 返回该 id 是否已停用（连续失败次数 ≥ maxFailCount）。
func Disabled(ctx context.Context, kind, id string) bool {
	database, err := db.GetDB()
	if err != nil {
		return false
	}
	var count int
	err = database.QueryRowContext(ctx,
		"SELECT CAST(value AS INTEGER) FROM api_cache WHERE key = ?", cacheKey(kind, id)).Scan(&count)
	if err == sql.ErrNoRows {
		return false
	}
	if err != nil {
		return false // 查询失败宁可回源验证
	}
	return count >= maxFailCount
}
