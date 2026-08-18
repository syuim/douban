package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	_ "modernc.org/sqlite"
)

var (
	instance *sql.DB
	once     sync.Once
)

func GetDB() (*sql.DB, error) {
	var initErr error
	once.Do(func() {
		dbPath := os.Getenv("DATABASE_PATH")
		if dbPath == "" {
			dbPath = "./data/stremio.db"
		}
		if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
			initErr = fmt.Errorf("create data dir: %w", err)
			return
		}

		// _pragma in DSN applies to every pooled connection;
		// db.Exec("PRAGMA ...") would only affect one connection
		dsn := "file:" + dbPath + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
		db, err := sql.Open("sqlite", dsn)
		if err != nil {
			initErr = fmt.Errorf("open sqlite: %w", err)
			return
		}

		if err := createTables(db); err != nil {
			initErr = fmt.Errorf("create tables: %w", err)
			return
		}
		migrateSchema(db)

		instance = db
	})
	return instance, initErr
}

func createTables(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS douban_mapping (
			douban_id integer PRIMARY KEY NOT NULL,
			tmdb_id integer,
			imdb_id text,
			trakt_id integer,
			calibrated integer DEFAULT false,
			created_at integer,
			updated_at integer
		);

		CREATE TABLE IF NOT EXISTS api_cache (
			key text PRIMARY KEY NOT NULL,
			value text NOT NULL,
			expires_at integer NOT NULL,
			created_at integer
		);
		CREATE INDEX IF NOT EXISTS idx_api_cache_expires_at ON api_cache (expires_at);
	`)
	return err
}

// migrateSchema 清理死表/死字段/死索引，幂等
func migrateSchema(db *sql.DB) {
	db.Exec("DROP TABLE IF EXISTS collection_failures")
	db.Exec("DROP INDEX IF EXISTS idx_mapping_pending")
	if _, err := db.Exec("ALTER TABLE douban_mapping DROP COLUMN last_attempted_at"); err != nil {
		// 新数据库无该列，忽略
	}
}
