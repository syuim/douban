package db

import (
	"os"
	"sync"
	"testing"
)

func TestMigrateSchemaCleansOldSchema(t *testing.T) {
	dir, err := os.MkdirTemp("", "db_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	t.Setenv("DATABASE_PATH", dir+"/test.db")

	instance = nil
	once = sync.Once{}

	db, err := GetDB()
	if err != nil {
		t.Fatal(err)
	}

	// collection_failures should NOT exist
	var tableCount int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='collection_failures'").Scan(&tableCount)
	if err != nil {
		t.Fatal(err)
	}
	if tableCount != 0 {
		t.Error("collection_failures table still exists after migration")
	}

	// last_attempted_at column should NOT exist
	var colCount int
	err = db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('douban_mapping') WHERE name='last_attempted_at'").Scan(&colCount)
	if err != nil {
		t.Fatal(err)
	}
	if colCount != 0 {
		t.Error("last_attempted_at column still exists after migration")
	}

	// idx_mapping_pending should NOT exist
	var idxCount int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_mapping_pending'").Scan(&idxCount)
	if err != nil {
		t.Fatal(err)
	}
	if idxCount != 0 {
		t.Error("idx_mapping_pending index still exists after migration")
	}
}

func TestMigrateSchemaIdempotent(t *testing.T) {
	dir, err := os.MkdirTemp("", "db_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	t.Setenv("DATABASE_PATH", dir+"/test2.db")

	instance = nil
	once = sync.Once{}

	// First startup: create tables + migrate
	_, err = GetDB()
	if err != nil {
		t.Fatal(err)
	}

	instance = nil
	once = sync.Once{}

	// Second startup: migrate again on clean schema
	db2, err := GetDB()
	if err != nil {
		t.Fatal(err)
	}

	var ok int
	err = db2.QueryRow("SELECT 1").Scan(&ok)
	if err != nil || ok != 1 {
		t.Errorf("database not functional after second startup: %v", err)
	}
}

func TestActiveTablesCount(t *testing.T) {
	dir, err := os.MkdirTemp("", "db_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	t.Setenv("DATABASE_PATH", dir+"/test3.db")

	instance = nil
	once = sync.Once{}

	db, err := GetDB()
	if err != nil {
		t.Fatal(err)
	}

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'").Scan(&count)
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Errorf("expected 2 tables (douban_mapping, api_cache), got %d", count)
	}
}