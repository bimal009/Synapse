package e2e

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"testing"

	_ "modernc.org/sqlite"
)

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", "file::memory:?cache=shared&_foreign_keys=ON")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })

	const schema = `
	CREATE TABLE chats (
		id           TEXT PRIMARY KEY,
		title        TEXT,
		project_path TEXT,
		created_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
		last_opened  DATETIME
	);
	CREATE TABLE permissions (
		id         TEXT PRIMARY KEY,
		convo_id   TEXT NOT NULL,
		action     TEXT NOT NULL,
		rule       TEXT NOT NULL CHECK(rule IN ('allow','ask','deny','always')),
		config     TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(convo_id, action),
		FOREIGN KEY (convo_id) REFERENCES chats(id)
	);`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestExecute(t *testing.T) {
	ctx := context.Background()
	dir, _ := os.Getwd()

	cmd := exec.CommandContext(ctx, "cmd.exe", "/C", "mkdir", "test")
	cmd.Dir = dir

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(output))
}
