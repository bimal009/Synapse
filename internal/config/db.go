package config

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bimal009/Synapse/internal/models"
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
		CREATE TABLE IF NOT EXISTS chats (
			id           TEXT PRIMARY KEY,
			title        TEXT,
			project_path TEXT,              -- NULL if no project attached
			created_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
			last_opened  DATETIME
		);

		CREATE TABLE IF NOT EXISTS chat_messages (
			id         TEXT PRIMARY KEY,
			chat_id    TEXT NOT NULL,
			role       TEXT NOT NULL,       -- "user" | "assistant"
			content    TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (chat_id) REFERENCES chats(id)
		);

		CREATE TABLE IF NOT EXISTS sessions (
			id         TEXT PRIMARY KEY,
			convo_id   TEXT NOT NULL,
			goal       TEXT,
			dag        TEXT,
			status     TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (convo_id) REFERENCES chats(id)
		);

		CREATE TABLE IF NOT EXISTS task_runs (
			id         TEXT PRIMARY KEY,
			convo_id   TEXT NOT NULL,
			session_id TEXT NOT NULL,
			task_id    TEXT NOT NULL,
			agent_role TEXT,
			status     TEXT,
			summary    TEXT,
			error      TEXT,
			priority   INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (convo_id)   REFERENCES chats(id),
			FOREIGN KEY (session_id) REFERENCES sessions(id)
		);

		CREATE TABLE IF NOT EXISTS models (
			id         TEXT PRIMARY KEY,
			name       TEXT NOT NULL,       -- user label e.g. "My Qwen Coder"
			role       TEXT NOT NULL,       -- "planner" | "coder" | "default" etc
			model      TEXT NOT NULL,       -- "qwen2.5:3b"
			url        TEXT NOT NULL,
			api_key    TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS convo_models (
			convo_id   TEXT NOT NULL,
			role       TEXT NOT NULL,
			model_id   TEXT NOT NULL,
			PRIMARY KEY (convo_id, role),
			FOREIGN KEY (convo_id)  REFERENCES chats(id),
			FOREIGN KEY (model_id)  REFERENCES models(id)
		);

		CREATE TABLE IF NOT EXISTS permissions (
			id         TEXT PRIMARY KEY,
			convo_id   TEXT NOT NULL,
			action     TEXT NOT NULL,
			rule       TEXT NOT NULL CHECK(rule IN ('allow', 'ask', 'deny', 'always')),
			config     TEXT,                -- JSON metadata: {"reason": "...", "expires_at": null}
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(convo_id, action),
			FOREIGN KEY (convo_id) REFERENCES chats(id)
		);

		CREATE TABLE IF NOT EXISTS permission_events (
			id         TEXT PRIMARY KEY,
			convo_id   TEXT NOT NULL,
			action     TEXT NOT NULL,
			decision   TEXT NOT NULL CHECK(decision IN ('allowed', 'denied', 'asked:allowed', 'asked:denied')),
			source     TEXT CHECK(source IN ('rule', 'user', 'default')),
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (convo_id) REFERENCES chats(id)
		);

		CREATE TABLE IF NOT EXISTS roles (
			name        TEXT PRIMARY KEY,
			description TEXT,
			created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
		);

	`)
	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	if err := d.migrateChatsSchema(); err != nil {
		return fmt.Errorf("chats migration failed: %w", err)
	}

	if err := d.migrateTaskRunsSchema(); err != nil {
		return fmt.Errorf("task_runs migration failed: %w", err)
	}

	if err := d.migrateDagsSchema(); err != nil {
		return fmt.Errorf("dags migration failed: %w", err)
	}

	if err := d.seedRoles(); err != nil {
		return fmt.Errorf("seed roles failed: %w", err)
	}

	return nil
}

func (d *databaseInitilization) seedRoles() error {
	roles := models.AllRoles()
	for _, role := range roles {
		if _, err := d.db.Exec(
			`INSERT OR IGNORE INTO roles (name, description) VALUES (?, ?)`,
			role, models.RoleDescriptions[role],
		); err != nil {
			return fmt.Errorf("seed role %q: %w", role, err)
		}
	}

	// Remove stale roles left over from earlier seedings (e.g. the old tier
	// names "coding"/"fast"/"planning") so the table matches AllRoles().
	placeholders := make([]string, len(roles))
	args := make([]any, len(roles))
	for i, role := range roles {
		placeholders[i] = "?"
		args[i] = role
	}
	q := "DELETE FROM roles WHERE name NOT IN (" + strings.Join(placeholders, ", ") + ")"
	if _, err := d.db.Exec(q, args...); err != nil {
		return fmt.Errorf("prune stale roles: %w", err)
	}
	return nil
}
func (d *databaseInitilization) migrateChatsSchema() error {
	if d.hasColumn("chats", "project_path") {
		return nil
	}

	_, err := d.db.Exec(`
		CREATE TABLE chats_new (
			id           TEXT PRIMARY KEY,
			title        TEXT,
			project_path TEXT,
			created_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
			last_opened  DATETIME
		);

		INSERT INTO chats_new (id, title, project_path, created_at, last_opened)
			SELECT c.id, c.title, p.path, c.created_at, c.created_at
			FROM chats c
			LEFT JOIN projects p ON p.id = c.project_id;

		DROP TABLE chats;
		ALTER TABLE chats_new RENAME TO chats;
	`)
	return err
}

func (d *databaseInitilization) migrateDagsSchema() error {
	// Earlier versions stored the whole DAG as a JSON blob in dags.dag. Replace
	// that with a structured header + per-task rows. The DAG is regenerable, so
	// dropping the old blob table is safe.
	if d.hasColumn("dags", "dag") {
		if _, err := d.db.Exec(`DROP TABLE IF EXISTS dag_tasks; DROP TABLE dags;`); err != nil {
			return err
		}
	}

	_, err := d.db.Exec(`
		CREATE TABLE IF NOT EXISTS dags (
			convo_id       TEXT PRIMARY KEY,
			id             TEXT NOT NULL,
			objective      TEXT,
			failure_policy TEXT,
			updated_at     DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (convo_id) REFERENCES chats(id)
		);

		CREATE TABLE IF NOT EXISTS dag_tasks (
			convo_id              TEXT NOT NULL,
			task_id               TEXT NOT NULL,
			position              INTEGER NOT NULL DEFAULT 0,
			title                 TEXT,
			description           TEXT,
			objective             TEXT,
			status                TEXT,
			model_role            TEXT,
			owner                 TEXT,
			priority              INTEGER DEFAULT 0,
			timeout               INTEGER DEFAULT 0,
			retry_max_attempts    INTEGER DEFAULT 0,
			retry_backoff         TEXT,
			retry_backoff_seconds INTEGER DEFAULT 0,
			inputs                TEXT,   -- JSON array
			outputs               TEXT,   -- JSON array
			dependencies          TEXT,   -- JSON array
			validation            TEXT,   -- JSON array
			tags                  TEXT,   -- JSON array
			PRIMARY KEY (convo_id, task_id),
			FOREIGN KEY (convo_id) REFERENCES chats(id)
		);
	`)
	return err
}

func (d *databaseInitilization) migrateTaskRunsSchema() error {
	if d.hasColumn("task_runs", "priority") {
		return nil
	}
	_, err := d.db.Exec(`ALTER TABLE task_runs ADD COLUMN priority INTEGER DEFAULT 0`)
	return err
}

func (d *databaseInitilization) hasColumn(table, col string) bool {
	rows, err := d.db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return false
	}
	defer rows.Close()
	for rows.Next() {
		var (
			cid, notnull, pk int
			name, ctype      string
			dflt             sql.NullString
		)
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return false
		}
		if name == col {
			return true
		}
	}
	return false
}

func (d *databaseInitilization) DB() *sql.DB {
	return d.db
}
