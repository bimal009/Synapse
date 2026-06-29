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

	"github.com/bimal009/Synapse/configs"
	"github.com/bimal009/Synapse/internal/models"
	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func dbTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}

type Chat interface {
	CreateChat(ctx context.Context, title string) (string, error)
	SelectFolder(ctx context.Context, chatID string) (string, error)
	TrustFolder(ctx context.Context, chatID string) error
	IsTrusted(ctx context.Context, chatID string) (bool, error)
	Path(ctx context.Context, chatID string) (string, error)
	ListChats(ctx context.Context) ([]models.Chat, error)
	DeleteChat(ctx context.Context, chatID string) error
	LoadConfig(ctx context.Context, chatID string) (map[string]configs.ModelConfig, error)
	AddModel(ctx context.Context, m models.Model) error
	UpdateModel(ctx context.Context, m models.Model) error
	SetActiveModel(ctx context.Context, chatID string, role string, modelID string) error
	DeactivateModel(ctx context.Context, chatID string, modelID string) error
	ActiveModelIDs(ctx context.Context, chatID string) ([]string, error)
	ActiveModels(ctx context.Context, chatID string) ([]models.Model, error)
	ListModels(ctx context.Context) ([]models.Model, error)
	DeleteModel(ctx context.Context, modelID string) error
	ListRoles(ctx context.Context) ([]models.Role, error)
	ListPermissions(ctx context.Context, chatID string) ([]models.Permission, error)
	SetPermission(ctx context.Context, chatID, action, rule string) error
	SaveDag(ctx context.Context, chatID string, dag models.Dag) error
	GetDag(ctx context.Context, chatID string) (models.Dag, bool, error)
	DeleteTask(ctx context.Context, chatID, taskID string) error
}

type chat struct {
	db          *sql.DB
	pendingPath map[string]string // chatID -> selected path, before trust
}

func NewInit(db *sql.DB) Chat {
	return &chat{
		db:          db,
		pendingPath: make(map[string]string),
	}
}

//go:embed all:templete
var templateFS embed.FS

func (c *chat) CreateChat(ctx context.Context, title string) (string, error) {
	id := uuid.New().String()
	now := dbTime(time.Now())

	_, err := c.db.ExecContext(ctx, `
		INSERT INTO chats (id, title, project_path, created_at, last_opened)
		VALUES (?, ?, NULL, ?, ?)
	`, id, title, now, now)
	if err != nil {
		return "", fmt.Errorf("failed to create chat: %w", err)
	}

	// Seed default read permission
	permID := uuid.New().String()
	_, err = c.db.ExecContext(ctx, `
		INSERT OR IGNORE INTO permissions (id, convo_id, action, rule)
		VALUES (?, ?, 'read', ?)
	`, permID, id, models.RuleAllow)
	if err != nil {
		return "", fmt.Errorf("failed to seed permissions: %w", err)
	}

	return id, nil
}

func (c *chat) SelectFolder(ctx context.Context, chatID string) (string, error) {
	path, err := runtime.OpenDirectoryDialog(ctx, runtime.OpenDialogOptions{
		Title: "Select Project Folder",
	})
	if err != nil {
		return "", fmt.Errorf("folder dialog failed: %w", err)
	}
	if path == "" {
		return "", nil
	}

	c.pendingPath[chatID] = path
	return path, nil
}

func (c *chat) TrustFolder(ctx context.Context, chatID string) error {
	path, ok := c.pendingPath[chatID]
	if !ok || path == "" {
		return fmt.Errorf("no folder selected for chat %s", chatID)
	}

	_, err := c.db.ExecContext(ctx, `
		UPDATE chats
		SET project_path = ?, last_opened = ?
		WHERE id = ?
	`, path, dbTime(time.Now()), chatID)
	if err != nil {
		return fmt.Errorf("failed to attach project path: %w", err)
	}

	delete(c.pendingPath, chatID)

	dst := filepath.Join(path, ".synapse")
	go func() {
		if _, err := os.Stat(dst); err == nil {
			runtime.EventsEmit(ctx, "chat:setup:done", dst)
			return
		}
		runtime.EventsEmit(ctx, "chat:setup:start", dst)
		if err := copyEmbeddedDir(templateFS, "templete/.synapse", dst); err != nil {
			runtime.EventsEmit(ctx, "chat:setup:error", err.Error())
			return
		}
		runtime.EventsEmit(ctx, "chat:setup:done", dst)
	}()

	return nil
}

func (c *chat) IsTrusted(ctx context.Context, chatID string) (bool, error) {
	var path sql.NullString
	err := c.db.QueryRowContext(ctx,
		`SELECT project_path FROM chats WHERE id = ?`, chatID,
	).Scan(&path)
	if errors.Is(err, sql.ErrNoRows) {
		return false, fmt.Errorf("chat not found: %s", chatID)
	}
	if err != nil {
		return false, err
	}
	return path.Valid && path.String != "", nil
}

func (c *chat) Path(ctx context.Context, chatID string) (string, error) {
	var path sql.NullString
	err := c.db.QueryRowContext(ctx,
		`SELECT project_path FROM chats WHERE id = ?`, chatID,
	).Scan(&path)
	if errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("chat not found: %s", chatID)
	}
	if err != nil {
		return "", err
	}
	if !path.Valid {
		return "", nil
	}
	return path.String, nil
}

func (c *chat) ListChats(ctx context.Context) ([]models.Chat, error) {
	rows, err := c.db.QueryContext(ctx, `
		SELECT id, COALESCE(title,''), COALESCE(project_path,''), 
		       COALESCE(created_at,''), COALESCE(last_opened,'')
		FROM chats
		ORDER BY last_opened DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list chats: %w", err)
	}
	defer rows.Close()

	var chats []models.Chat
	for rows.Next() {
		var ch models.Chat
		if err := rows.Scan(
			&ch.ID, &ch.Title, &ch.ProjectPath,
			&ch.CreatedAt, &ch.LastOpened,
		); err != nil {
			return nil, err
		}
		chats = append(chats, ch)
	}
	return chats, rows.Err()
}

func (c *chat) DeleteChat(ctx context.Context, chatID string) error {
	_, err := c.db.ExecContext(ctx, `DELETE FROM chats WHERE id = ?`, chatID)
	if err != nil {
		return fmt.Errorf("failed to delete chat: %w", err)
	}
	return nil
}

func (c *chat) ListPermissions(ctx context.Context, chatID string) ([]models.Permission, error) {
	if chatID == "" {
		return nil, fmt.Errorf("chatID is required")
	}

	rows, err := c.db.QueryContext(ctx, `
		SELECT id, convo_id, action, rule, COALESCE(config,''), COALESCE(created_at,'')
		FROM permissions
		WHERE convo_id = ?
		ORDER BY action
	`, chatID)
	if err != nil {
		return nil, fmt.Errorf("failed to list permissions: %w", err)
	}
	defer rows.Close()

	var perms []models.Permission
	for rows.Next() {
		var p models.Permission
		if err := rows.Scan(
			&p.ID, &p.ConvoID, &p.Action, &p.Rule, &p.Config, &p.CreatedAt,
		); err != nil {
			return nil, err
		}
		perms = append(perms, p)
	}
	return perms, rows.Err()
}

func (c *chat) SetPermission(ctx context.Context, chatID, action, rule string) error {
	if chatID == "" || action == "" {
		return fmt.Errorf("chatID and action are required")
	}
	switch rule {
	case models.RuleAllow, models.RuleAsk, models.RuleDeny, models.RuleAlways:
	default:
		return fmt.Errorf("invalid rule: %s", rule)
	}

	_, err := c.db.ExecContext(ctx, `
		INSERT INTO permissions (id, convo_id, action, rule, created_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(convo_id, action) DO UPDATE SET rule = excluded.rule
	`, uuid.New().String(), chatID, action, rule, dbTime(time.Now()))
	if err != nil {
		return fmt.Errorf("failed to set permission: %w", err)
	}
	return nil
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
