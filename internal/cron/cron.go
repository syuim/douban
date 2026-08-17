package cron

import (
	"context"
	"database/sql"
	"log"
	"math/rand"
	"sync/atomic"
	"time"

	"stremio-addon-douban/internal/api"
	"stremio-addon-douban/internal/collection"
	"stremio-addon-douban/internal/db"
)

var isRunning atomic.Bool

func RunScheduledTask() {
	if !isRunning.CompareAndSwap(false, true) {
		return
	}
	defer isRunning.Store(false)

	ctx := context.Background()
	log.Println("[cron] 每日缓存更新开始")

	database, err := db.GetDB()
	if err != nil {
		log.Printf("[cron] db error: %v", err)
		return
	}

	svc := api.GetService()

	// 豆瓣榜单（跳过 TMDB 与年度占位符，年度有单独的 YearlyRankingConfigs）
	for _, c := range collection.CollectionConfigs {
		if c.IsTmdb || collection.IsYearlyRankingID(c.ID) {
			continue
		}
		refreshCollection(ctx, database, svc, c.ID)
		randomSleep(5, 18)
	}

	// 年度榜单（每个具体年份）
	for _, c := range collection.YearlyRankingConfigs {
		refreshCollection(ctx, database, svc, c.ID)
		randomSleep(5, 18)
	}

	// 剧场 doulist
	for _, t := range collection.TheaterConfigs {
		refreshDoulist(ctx, database, svc, t.ID)
		randomSleep(5, 18)
	}

	// 删过期缓存（保底，刚全量写完理论上无过期）
	api.CleanExpiredCache()

	log.Println("[cron] 每日缓存更新完成")
}

// refreshCollection 删缓存 → 拉第 1 页 → 最多重试 2 次，失败跳过
func refreshCollection(ctx context.Context, database *sql.DB, svc *api.Service, catalogID string) {
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(1<<attempt) * time.Second)
		}

		database.ExecContext(ctx, "DELETE FROM api_cache WHERE key = ?", "subject_collection:"+catalogID+":0")

		_, err := svc.DoubanAPI.GetSubjectCollectionItems(ctx, catalogID, 0)
		if err == nil {
			return
		}
		log.Printf("[cron] %s 拉取失败(尝试%d): %v", catalogID, attempt+1, err)
	}
}

// refreshDoulist 删缓存 → 拉第 1 页 → 最多重试 2 次，失败跳过
func refreshDoulist(ctx context.Context, database *sql.DB, svc *api.Service, theaterID string) {
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(1<<attempt) * time.Second)
		}

		database.ExecContext(ctx, "DELETE FROM api_cache WHERE key = ?", "doulist:"+theaterID+":0")

		_, err := svc.DoubanAPI.GetDoulistItems(ctx, theaterID, 1)
		if err == nil {
			return
		}
		log.Printf("[cron] 剧场 %s 拉取失败(尝试%d): %v", theaterID, attempt+1, err)
	}
}

func randomSleep(minSec, maxSec int) {
	d := minSec + rand.Intn(maxSec-minSec+1)
	time.Sleep(time.Duration(d) * time.Second)
}