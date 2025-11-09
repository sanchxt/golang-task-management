package sqlite

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	sqlite3 "github.com/mattn/go-sqlite3"
)

var (
	registerOnce sync.Once
)

type DB struct {
	*sqlx.DB
}

type Config struct {
	Path string
}

// creates a new db conn & runs migrations
func NewDB(cfg Config) (*DB, error) {
	dir := filepath.Dir(cfg.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// register REGEXP function once
	registerOnce.Do(func() {
		sql.Register("sqlite3_with_regexp",
			&sqlite3.SQLiteDriver{
				ConnectHook: func(conn *sqlite3.SQLiteConn) error {
					return conn.RegisterFunc("regexp", regexpFunc, true)
				},
			})
	})

	// open SQLite connection
	db, err := sqlx.Open("sqlite3_with_regexp", cfg.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// enable foreign keys and WAL
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode = WAL"); err != nil {
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	if err := runMigrations(db.DB); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return &DB{DB: db}, nil
}

// REGEXP function for SQLite
func regexpFunc(pattern, text string) (bool, error) {
	matched, err := regexp.MatchString(pattern, text)
	if err != nil {
		return false, err
	}
	return matched, nil
}

// executes db schema
func runMigrations(db *sql.DB) error {
	statements := []string{
		// create projects table
		`CREATE TABLE IF NOT EXISTS projects (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			description TEXT,
			parent_id INTEGER,
			color TEXT,
			icon TEXT,
			status TEXT NOT NULL DEFAULT 'active',
			is_favorite BOOLEAN NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

			CHECK(name != ''),
			CHECK(length(name) <= 100),
			CHECK(length(description) <= 500),
			CHECK(status IN ('active', 'archived', 'completed')),
			CHECK(parent_id != id),
			FOREIGN KEY (parent_id) REFERENCES projects(id) ON DELETE CASCADE
		)`,

		`CREATE INDEX IF NOT EXISTS idx_projects_name ON projects(name)`,
		`CREATE INDEX IF NOT EXISTS idx_projects_parent_id ON projects(parent_id)`,
		`CREATE INDEX IF NOT EXISTS idx_projects_status ON projects(status)`,
		`CREATE INDEX IF NOT EXISTS idx_projects_is_favorite ON projects(is_favorite)`,
		`CREATE INDEX IF NOT EXISTS idx_projects_created_at ON projects(created_at)`,

		`CREATE TRIGGER IF NOT EXISTS update_projects_updated_at
			AFTER UPDATE ON projects
			FOR EACH ROW
		BEGIN
			UPDATE projects SET updated_at = CURRENT_TIMESTAMP WHERE id = OLD.id;
		END`,

		// create tasks table
		`CREATE TABLE IF NOT EXISTS tasks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			description TEXT,
			priority TEXT NOT NULL DEFAULT 'medium',
			status TEXT NOT NULL DEFAULT 'pending',
			tags TEXT,
			project_id INTEGER,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			due_date DATETIME,

			CHECK(title != ''),
			CHECK(length(title) <= 200),
			CHECK(length(description) <= 1000),
			CHECK(priority IN ('low', 'medium', 'high', 'urgent')),
			CHECK(status IN ('pending', 'in_progress', 'completed', 'cancelled')),
			FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE SET NULL
		)`,

		`CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status)`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_priority ON tasks(priority)`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_project_id ON tasks(project_id)`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_due_date ON tasks(due_date)`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_created_at ON tasks(created_at)`,

		`CREATE TRIGGER IF NOT EXISTS update_tasks_updated_at
			AFTER UPDATE ON tasks
			FOR EACH ROW
		BEGIN
			UPDATE tasks SET updated_at = CURRENT_TIMESTAMP WHERE id = OLD.id;
		END`,

		// create project_templates table
		`CREATE TABLE IF NOT EXISTS project_templates (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			description TEXT,
			task_definitions TEXT NOT NULL,
			project_defaults TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

			CHECK(name != ''),
			CHECK(length(name) <= 100),
			CHECK(length(description) <= 500)
		)`,

		`CREATE INDEX IF NOT EXISTS idx_templates_name ON project_templates(name)`,
		`CREATE INDEX IF NOT EXISTS idx_templates_created_at ON project_templates(created_at)`,

		`CREATE TRIGGER IF NOT EXISTS update_templates_updated_at
			AFTER UPDATE ON project_templates
			FOR EACH ROW
		BEGIN
			UPDATE project_templates SET updated_at = CURRENT_TIMESTAMP WHERE id = OLD.id;
		END`,

		// create saved_views table
		`CREATE TABLE IF NOT EXISTS saved_views (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			description TEXT,
			filter_config TEXT NOT NULL,
			is_favorite BOOLEAN NOT NULL DEFAULT 0,
			hot_key INTEGER,
			last_accessed DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

			CHECK(name != ''),
			CHECK(length(name) <= 100),
			CHECK(length(description) <= 500),
			CHECK(hot_key IS NULL OR (hot_key >= 1 AND hot_key <= 9)),
			UNIQUE(hot_key)
		)`,

		`CREATE INDEX IF NOT EXISTS idx_saved_views_name ON saved_views(name)`,
		`CREATE INDEX IF NOT EXISTS idx_saved_views_is_favorite ON saved_views(is_favorite)`,
		`CREATE INDEX IF NOT EXISTS idx_saved_views_hot_key ON saved_views(hot_key)`,
		`CREATE INDEX IF NOT EXISTS idx_saved_views_last_accessed ON saved_views(last_accessed DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_saved_views_created_at ON saved_views(created_at)`,

		`CREATE TRIGGER IF NOT EXISTS update_saved_views_updated_at
			AFTER UPDATE ON saved_views
			FOR EACH ROW
		BEGIN
			UPDATE saved_views SET updated_at = CURRENT_TIMESTAMP WHERE id = OLD.id;
		END`,

		// create search_history table
		`CREATE TABLE IF NOT EXISTS search_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			query_text TEXT NOT NULL,
			search_mode TEXT NOT NULL,
			fuzzy_threshold INTEGER,
			query_type TEXT NOT NULL,
			project_filter TEXT,
			result_count INTEGER DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

			CHECK(query_text != ''),
			CHECK(search_mode IN ('text', 'regex', 'fuzzy')),
			CHECK(query_type IN ('simple', 'query_language', 'project_mention')),
			CHECK(fuzzy_threshold IS NULL OR (fuzzy_threshold >= 0 AND fuzzy_threshold <= 100))
		)`,

		`CREATE INDEX IF NOT EXISTS idx_search_history_updated_at ON search_history(updated_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_search_history_query_text ON search_history(query_text)`,

		`CREATE TRIGGER IF NOT EXISTS update_search_history_updated_at
			AFTER UPDATE ON search_history
			FOR EACH ROW
		BEGIN
			UPDATE search_history SET updated_at = CURRENT_TIMESTAMP WHERE id = OLD.id;
		END`,
	}

	migrations := []string{
		`ALTER TABLE projects ADD COLUMN aliases TEXT DEFAULT '[]'`,

		`ALTER TABLE projects ADD COLUMN notes TEXT DEFAULT ''`,
	}

	for i, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("failed to execute statement %d: %w", i+1, err)
		}
	}

	for _, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			if !isDuplicateColumnError(err) {
				return fmt.Errorf("failed to execute migration: %w", err)
			}
		}
	}

	aliasIndexStmt := `CREATE INDEX IF NOT EXISTS idx_projects_aliases ON projects(aliases)`
	if _, err := db.Exec(aliasIndexStmt); err != nil {
		return fmt.Errorf("failed to create aliases index: %w", err)
	}

	return nil
}

func isDuplicateColumnError(err error) bool {
	if err == nil {
		return false
	}
	return regexp.MustCompile(`duplicate column`).MatchString(err.Error())
}

func (db *DB) Close() error {
	return db.DB.Close()
}
