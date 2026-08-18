// Package db provides the SQLite storage layer, a port of the original
// AnimeGarden PostgreSQL schema (see apps/server/drizzle/*.sql).
package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// Open opens (creating if needed) the SQLite database at path and applies
// the schema. Relative paths resolve against the working directory (DATA_DIR
// env overrides the location).
func Open(path string) (*sql.DB, error) {
	if path == "" {
		path = filepath.Join("data", "animegarden.db")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path+"?_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1) // SQLite: single writer
	if err := db.Ping(); err != nil {
		return nil, err
	}
	if err := Migrate(db); err != nil {
		return nil, err
	}
	return db, nil
}

// Migrate applies the schema (idempotent).
func Migrate(db *sql.DB) error {
	ddl := []string{
		`CREATE TABLE IF NOT EXISTS providers (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			refreshed_at TEXT NOT NULL,
			is_active INTEGER NOT NULL DEFAULT 1
		)`,
		`CREATE TABLE IF NOT EXISTS resources (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			provider_name TEXT NOT NULL,
			provider_id TEXT NOT NULL,
			title TEXT NOT NULL,
			title_alt TEXT NOT NULL,
			title_search TEXT NOT NULL DEFAULT '',
			href TEXT NOT NULL,
			type TEXT NOT NULL,
			magnet TEXT NOT NULL,
			tracker TEXT NOT NULL,
			size INTEGER NOT NULL,
			created_at TEXT NOT NULL,
			fetched_at TEXT NOT NULL,
			indexed_at TEXT NOT NULL DEFAULT '',
			publisher_id INTEGER NOT NULL,
			fansub_id INTEGER,
			duplicated_id INTEGER,
			subject_id INTEGER,
			metadata TEXT,
			is_deleted INTEGER NOT NULL DEFAULT 0,
			UNIQUE(provider_name, provider_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_resources_created_at ON resources (created_at DESC, id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_resources_live_created_at ON resources (created_at DESC, id DESC) WHERE is_deleted = 0 AND duplicated_id IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_resources_subject_created_at ON resources (subject_id, created_at DESC, id DESC) WHERE is_deleted = 0 AND duplicated_id IS NULL AND subject_id IS NOT NULL`,
		`CREATE INDEX IF NOT EXISTS idx_resources_type_created_at ON resources (type, created_at DESC, id DESC) WHERE is_deleted = 0 AND duplicated_id IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_resources_fansub_created_at ON resources (fansub_id, created_at DESC, id DESC) WHERE is_deleted = 0 AND duplicated_id IS NULL AND fansub_id IS NOT NULL`,
		`CREATE INDEX IF NOT EXISTS idx_resources_publisher_created_at ON resources (publisher_id, created_at DESC, id DESC) WHERE is_deleted = 0 AND duplicated_id IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_resources_provider_created_at ON resources (provider_name, created_at DESC, id DESC) WHERE is_deleted = 0`,
		`CREATE INDEX IF NOT EXISTS idx_resources_subject ON resources (subject_id)`,
		`CREATE INDEX IF NOT EXISTS idx_resources_fansub ON resources (fansub_id)`,
		`CREATE INDEX IF NOT EXISTS idx_resources_publisher ON resources (publisher_id)`,
		`CREATE TABLE IF NOT EXISTS details (
			id INTEGER PRIMARY KEY,
			description TEXT NOT NULL DEFAULT '',
			magnets TEXT NOT NULL DEFAULT '[]',
			files TEXT NOT NULL DEFAULT '[]',
			has_more_files INTEGER NOT NULL DEFAULT 0,
			fetched_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS subjects (
			bangumi_id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			keywords TEXT NOT NULL,
			actived_at TEXT NOT NULL,
			is_archived INTEGER NOT NULL DEFAULT 1
		)`,
		`CREATE TABLE IF NOT EXISTS tags (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE
		)`,
		`CREATE TABLE IF NOT EXISTS teams (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			avatar TEXT,
			providers TEXT NOT NULL DEFAULT '{}'
		)`,
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			avatar TEXT,
			providers TEXT NOT NULL DEFAULT '{}'
		)`,
		`CREATE TABLE IF NOT EXISTS collections (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			hash TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL DEFAULT '',
			user TEXT NOT NULL,
			filters TEXT NOT NULL DEFAULT '[]',
			fetched_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS telegram_messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			resource_id INTEGER NOT NULL,
			publisher_id INTEGER NOT NULL,
			fansub_id INTEGER,
			subject_id INTEGER NOT NULL,
			episode TEXT NOT NULL,
			telegram_chat_id INTEGER,
			telegram_message_id INTEGER,
			status INTEGER NOT NULL,
			sent_at TEXT,
			edited_at TEXT,
			updated_at TEXT NOT NULL,
			UNIQUE(publisher_id, subject_id, episode)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_telegram_resource ON telegram_messages (resource_id)`,
		`CREATE INDEX IF NOT EXISTS idx_telegram_status ON telegram_messages (status)`,
	}

	for _, stmt := range ddl {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("migrate: %w (stmt: %s)", err, stmt)
		}
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	seeds := []struct {
		sql  string
		args []any
	}{
		{`INSERT OR IGNORE INTO providers (id, name, refreshed_at, is_active) VALUES ('dmhy', '动漫花园', ?, 1)`, []any{now}},
		{`INSERT OR IGNORE INTO providers (id, name, refreshed_at, is_active) VALUES ('moe', '萌番组', ?, 1)`, []any{now}},
		{`INSERT OR IGNORE INTO providers (id, name, refreshed_at, is_active) VALUES ('mikan', '蜜柑计划', ?, 1)`, []any{now}},
		{`INSERT OR IGNORE INTO providers (id, name, refreshed_at, is_active) VALUES ('ani', 'ANi', ?, 1)`, []any{now}},
		{`INSERT OR IGNORE INTO users (name, avatar, providers) VALUES ('anonymous', '', '{}')`, nil},
		{`INSERT OR IGNORE INTO users (name, avatar, providers) VALUES ('ANi', '', '{}')`, nil},
		{`INSERT OR IGNORE INTO teams (name, avatar, providers) VALUES ('ANi', '', '{}')`, nil},
	}
	for _, seed := range seeds {
		if _, err := db.Exec(seed.sql, seed.args...); err != nil {
			return fmt.Errorf("seed: %w (stmt: %s)", err, seed.sql)
		}
	}
	return nil
}
