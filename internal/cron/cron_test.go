package cron

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"stremio-addon-douban/internal/db"
)

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "cron-test")
	if err != nil {
		panic(err)
	}
	_ = os.Setenv("DATABASE_PATH", dir+"/test.db")
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

func mustDB(t *testing.T) *sql.DB {
	t.Helper()
	database, err := db.GetDB()
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	return database
}

func setLastFailDate(t *testing.T, catalogID, date string) {
	t.Helper()
	if _, err := mustDB(t).Exec(
		"UPDATE collection_failures SET last_fail_date = ? WHERE collection_id = ?", date, catalogID); err != nil {
		t.Fatalf("set last_fail_date: %v", err)
	}
}

func failureRow(t *testing.T, catalogID string) (int, string) {
	t.Helper()
	var count int
	var date string
	err := mustDB(t).QueryRow(
		"SELECT fail_count, last_fail_date FROM collection_failures WHERE collection_id = ?", catalogID,
	).Scan(&count, &date)
	if err == sql.ErrNoRows {
		return -1, ""
	}
	if err != nil {
		t.Fatalf("query failure row: %v", err)
	}
	return count, date
}

func TestRecordCollectionFailureSameDay(t *testing.T) {
	ctx := context.Background()
	database, err := db.GetDB()
	if err != nil {
		t.Fatalf("db: %v", err)
	}

	recordCollectionFailure(ctx, database, "same-day-cat")
	recordCollectionFailure(ctx, database, "same-day-cat")

	count, date := failureRow(t, "same-day-cat")
	if count != 1 {
		t.Fatalf("same-day repeat should not increment count, got %d", count)
	}
	if date != time.Now().Format("2006-01-02") {
		t.Fatalf("unexpected last_fail_date %q", date)
	}
}

func TestRecordCollectionFailureResetsAfterGap(t *testing.T) {
	ctx := context.Background()
	database, err := db.GetDB()
	if err != nil {
		t.Fatalf("db: %v", err)
	}

	recordCollectionFailure(ctx, database, "gap-cat")
	setLastFailDate(t, "gap-cat", time.Now().AddDate(0, 0, -3).Format("2006-01-02"))
	recordCollectionFailure(ctx, database, "gap-cat")

	count, _ := failureRow(t, "gap-cat")
	if count != 1 {
		t.Fatalf("gap should reset count to 1, got %d", count)
	}
}

func TestRecordCollectionFailureStopsAfterThreeDays(t *testing.T) {
	ctx := context.Background()
	database, err := db.GetDB()
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	dead := "dead_cat"
	today := time.Now().Format("2006-01-02")
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	// 第 1 天失败
	recordCollectionFailure(ctx, database, dead)
	if count, _ := failureRow(t, dead); count != 1 {
		t.Fatalf("day1: expected count 1, got %d", count)
	}

	// 第 2 天失败
	setLastFailDate(t, dead, yesterday)
	recordCollectionFailure(ctx, database, dead)
	if count, _ := failureRow(t, dead); count != 2 {
		t.Fatalf("day2: expected count 2, got %d", count)
	}

	// 第 3 天失败 → 达到上限，失败记录保留并标记当天
	setLastFailDate(t, dead, yesterday)
	recordCollectionFailure(ctx, database, dead)
	count, date := failureRow(t, dead)
	if count != 3 || date != today {
		t.Fatalf("failure row should stay marked today, got count=%d date=%q", count, date)
	}

	// 同一天内再次失败：提前返回，不重复统计
	recordCollectionFailure(ctx, database, dead)
	if count, date := failureRow(t, dead); count != 3 || date != today {
		t.Fatalf("same-day repeat should not re-count, got count=%d date=%q", count, date)
	}
}
