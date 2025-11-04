package sqlite

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
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

	// open SQLite connection
	db, err := sqlx.Open("sqlite3", cfg.Path)
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

	// run migrations
	if err := runMigrations(db.DB); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return &DB{DB: db}, nil
}

// executes db schema
func runMigrations(db *sql.DB) error {
	schema := `
	-- Create tasks table
	CREATE TABLE IF NOT EXISTS tasks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		description TEXT,
		priority TEXT NOT NULL DEFAULT 'medium',
		status TEXT NOT NULL DEFAULT 'pending',
		tags TEXT,
		project TEXT,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		due_date DATETIME,

		CHECK(title != ''),
		CHECK(length(title) <= 200),
		CHECK(length(description) <= 1000),
		CHECK(priority IN ('low', 'medium', 'high', 'urgent')),
		CHECK(status IN ('pending', 'in_progress', 'completed', 'cancelled'))
	);

	CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
	CREATE INDEX IF NOT EXISTS idx_tasks_priority ON tasks(priority);
	CREATE INDEX IF NOT EXISTS idx_tasks_project ON tasks(project);
	CREATE INDEX IF NOT EXISTS idx_tasks_due_date ON tasks(due_date);
	CREATE INDEX IF NOT EXISTS idx_tasks_created_at ON tasks(created_at);

	CREATE TRIGGER IF NOT EXISTS update_tasks_updated_at
		AFTER UPDATE ON tasks
		FOR EACH ROW
	BEGIN
		UPDATE tasks SET updated_at = CURRENT_TIMESTAMP WHERE id = OLD.id;
	END;
	`

	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	return nil
}

func (db *DB) Close() error {
	return db.DB.Close()
}
