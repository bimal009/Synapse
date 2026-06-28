package project

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Project is the public surface of a project. The concrete implementation
// stays unexported so callers depend only on this interface.
type Project interface {
	SelectFolder(ctx context.Context) (string, error)
	TrustFolder(ctx context.Context) error
	IsTrusted() bool
	Path() string
	Load(ctx context.Context) (string, error)
}

type project struct {
	path    string
	trusted bool
	db      *sql.DB
}

func NewInit(db *sql.DB) Project {
	return &project{db: db}
}

func (p *project) SelectFolder(ctx context.Context) (string, error) {
	path, err := runtime.OpenDirectoryDialog(ctx, runtime.OpenDialogOptions{
		Title: "Select Project Folder",
	})
	if err != nil {
		return "", err
	}
	p.path = path
	p.trusted = false
	return path, nil
}

//go:embed all:templete
var templateFS embed.FS

func (p *project) TrustFolder(ctx context.Context) error {
	if p.path == "" {
		return fmt.Errorf("no folder selected")
	}

	name := filepath.Base(p.path)
	id := uuid.New().String()
	now := time.Now()

	_, err := p.db.ExecContext(ctx, `
		INSERT INTO projects (id, name, path, trusted, created_at, last_opened)
		VALUES (?, ?, ?, TRUE, ?, ?)
		ON CONFLICT(path) DO UPDATE SET
			trusted = TRUE,
			last_opened = excluded.last_opened
	`, id, name, p.path, now, now)
	if err != nil {
		return fmt.Errorf("failed to trust folder: %w", err)
	}

	p.trusted = true

	dst := filepath.Join(p.path, ".synapse")
	go func() {
		if _, err := os.Stat(dst); err == nil {
			return
		}
		runtime.EventsEmit(ctx, "project:setup:start", dst)
		if err := copyEmbeddedDir(templateFS, "templete/.synapse", dst); err != nil {
			runtime.EventsEmit(ctx, "project:setup:error", err.Error())
			return
		}
		runtime.EventsEmit(ctx, "project:setup:done", dst)
	}()

	return nil
}

func (p *project) IsTrusted() bool {
	return p.trusted
}

func (p *project) Path() string {
	return p.path
}

func (p *project) Load(ctx context.Context) (string, error) {
	var path string
	err := p.db.QueryRowContext(ctx, `
		SELECT path FROM projects
		WHERE trusted = TRUE
		ORDER BY last_opened DESC
		LIMIT 1
	`).Scan(&path)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to load project: %w", err)
	}

	p.path = path
	p.trusted = true
	return path, nil
}

func copyEmbeddedDir(src fs.FS, srcDir, dstDir string) error {
	return fs.WalkDir(src, srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		target := filepath.Join(dstDir, rel)

		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}

		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}

		in, err := src.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()

		out, err := os.Create(target)
		if err != nil {
			return err
		}
		defer out.Close()

		_, err = io.Copy(out, in)
		return err
	})
}
