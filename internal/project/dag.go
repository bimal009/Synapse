package project

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

func (c *chat) SaveDag(ctx context.Context, chatID, dagJSON string) error {
	if chatID == "" {
		return fmt.Errorf("chatID is required")
	}
	if strings.TrimSpace(dagJSON) == "" {
		return fmt.Errorf("dag is empty")
	}
	_, err := c.db.ExecContext(ctx, `
		INSERT INTO dags (convo_id, dag, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(convo_id) DO UPDATE SET dag = excluded.dag, updated_at = excluded.updated_at
	`, chatID, dagJSON, dbTime(time.Now()))
	if err != nil {
		return fmt.Errorf("failed to save dag: %w", err)
	}
	return nil
}

func (c *chat) GetDag(ctx context.Context, chatID string) (string, error) {
	if chatID == "" {
		return "", fmt.Errorf("chatID is required")
	}
	var dag string
	err := c.db.QueryRowContext(ctx,
		`SELECT dag FROM dags WHERE convo_id = ?`, chatID,
	).Scan(&dag)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to load dag: %w", err)
	}
	return dag, nil
}
