package config

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type DatabaseInitilization interface {
	Init() error
	DB() *sql.DB
}

type databaseInitilization struct {
	dbPath string
	db     *sql.DB
}

func NewDBInitialize() (DatabaseInitilization, error) {
	path, err := resolveDBPath()
	if err != nil {
		return nil, err
	}
	return &databaseInitilization{dbPath: path}, nil
}

func resolveDBPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	appDir := filepath.Join(configDir, "Synapse")

	if err := os.MkdirAll(appDir, 0755); err != nil {
		return "", err
	}

	return filepath.Join(appDir, "synapse.db"), nil
}

func (d *databaseInitilization) Init() error {
	dsn := fmt.Sprintf("file:%s?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=ON", d.dbPath)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return fmt.Errorf("failed to open db: %w", err)
	}

	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(time.Hour)
	db.SetConnMaxIdleTime(30 * time.Minute)

	if err := db.Ping(); err != nil {
		return fmt.Errorf("db unreachable: %w", err)
	}

	d.db = db

	return d.migrate()
}

func (d *databaseInitilization) migrate() error {
	_, err := d.db.Exec(`
		CREATE TABLE IF NOT EXISTS projects (
			id          TEXT PRIMARY KEY,
			name        TEXT NOT NULL,
			path        TEXT NOT NULL UNIQUE,
			trusted     BOOLEAN DEFAULT FALSE,
			created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
			last_opened DATETIME
		);

		CREATE TABLE IF NOT EXISTS sessions (
			id          TEXT PRIMARY KEY,
			project_id  TEXT NOT NULL,
			goal        TEXT,
			dag         TEXT,
			status      TEXT,
			created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (project_id) REFERENCES projects(id)
		);

		CREATE TABLE IF NOT EXISTS task_runs (
			id          TEXT PRIMARY KEY,
			project_id  TEXT NOT NULL,
			session_id  TEXT NOT NULL,
			task_id     TEXT NOT NULL,
			agent_role  TEXT,
			status      TEXT,
			summary     TEXT,
			error       TEXT,
			created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (project_id) REFERENCES projects(id),
			FOREIGN KEY (session_id) REFERENCES sessions(id)
		);
	`)
	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}
	return nil
}

func (d *databaseInitilization) DB() *sql.DB {
	return d.db
}
