package failwatch

import (
	"context"
	"os"
	"testing"

	"stremio-addon-douban/internal/db"
)

func setup(t *testing.T) {
	t.Helper()
	oldPath := os.Getenv("DATABASE_PATH")
	if err := os.Setenv("DATABASE_PATH", t.TempDir()+"/test.db"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if oldPath == "" {
			os.Unsetenv("DATABASE_PATH")
		} else {
			os.Setenv("DATABASE_PATH", oldPath)
		}
	})
}

// resetTable 清空本包写入的观察记录，保证测试间隔离
func resetTable(t *testing.T) {
	t.Helper()
	database, err := db.GetDB()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec("DELETE FROM api_cache WHERE key LIKE 'failwatch:%'"); err != nil {
		t.Fatal(err)
	}
}

func TestRecordAccumulatesAndDisables(t *testing.T) {
	setup(t)
	resetTable(t)
	ctx := context.Background()

	Record(ctx, KindCollection, "top250")
	if Disabled(ctx, KindCollection, "top250") {
		t.Fatal("1 次失败不应停用")
	}
	Record(ctx, KindCollection, "top250")
	Record(ctx, KindCollection, "top250")
	if !Disabled(ctx, KindCollection, "top250") {
		t.Fatal("连续 3 次 404 后应停用")
	}
}

func TestClearRestores(t *testing.T) {
	setup(t)
	resetTable(t)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		Record(ctx, KindCollection, "movie_hot_gaia")
	}
	if !Disabled(ctx, KindCollection, "movie_hot_gaia") {
		t.Fatal("前置：应已停用")
	}

	Clear(ctx, KindCollection, "movie_hot_gaia")
	if Disabled(ctx, KindCollection, "movie_hot_gaia") {
		t.Fatal("Clear 后应恢复")
	}

	// 恢复后重新起算
	Record(ctx, KindCollection, "movie_hot_gaia")
	if Disabled(ctx, KindCollection, "movie_hot_gaia") {
		t.Fatal("恢复后首次失败不应停用")
	}
}

func TestKindIsolation(t *testing.T) {
	setup(t)
	resetTable(t)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		Record(ctx, KindCollection, "1292052")
	}
	if !Disabled(ctx, KindCollection, "1292052") {
		t.Fatal("前置：c:1292052 应停用")
	}
	if Disabled(ctx, KindDoulist, "1292052") {
		t.Fatal("c 与 d 命名空间应互不影响")
	}
}

func TestPersistenceInTable(t *testing.T) {
	setup(t)
	resetTable(t)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		Record(ctx, KindDoulist, "155026800")
	}

	// 观察记录落在 api_cache 表中：直查库验证，重启后 Disabled 读库天然恢复
	database, err := db.GetDB()
	if err != nil {
		t.Fatal(err)
	}
	var count int
	if err := database.QueryRowContext(ctx,
		"SELECT CAST(value AS INTEGER) FROM api_cache WHERE key = ?", cacheKey(KindDoulist, "155026800")).Scan(&count); err != nil {
		t.Fatalf("表中应有记录: %v", err)
	}
	if count != 3 {
		t.Fatalf("count 应为 3，got %d", count)
	}
}
