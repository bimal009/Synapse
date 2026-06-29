package project

import (
	"context"
	"fmt"
	"time"

	"github.com/bimal009/Synapse/configs"
	"github.com/bimal009/Synapse/internal/models"
	"github.com/google/uuid"
)

func (c *chat) LoadConfig(ctx context.Context, chatID string) (map[string]configs.ModelConfig, error) {
	rows, err := c.db.QueryContext(ctx, `
		SELECT m.role, m.model, m.url, COALESCE(m.api_key, '')
		FROM convo_models cm
		JOIN models m ON cm.model_id = m.id
		WHERE cm.convo_id = ?
	`, chatID)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	defer rows.Close()

	result := make(map[string]configs.ModelConfig)
	for rows.Next() {
		var role, model, url, apiKey string
		if err := rows.Scan(&role, &model, &url, &apiKey); err != nil {
			return nil, err
		}
		result[role] = configs.ModelConfig{
			Model:     model,
			URL:       url,
			APIKey:    apiKey,
			Streaming: true,
			Thinking:  true,
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// No fallback: if no model is activated for this chat, return an empty map
	// so RunAgents reports "no models configured" instead of silently using a
	// hardcoded default.
	return result, nil
}

func (c *chat) AddModel(ctx context.Context, m models.Model) error {
	if m.Name == "" || m.Model == "" || m.URL == "" || m.Role == "" {
		return fmt.Errorf("name, model, url and role are required")
	}

	m.ID = uuid.New().String()
	m.CreatedAt = dbTime(time.Now()) // store as string directly

	_, err := c.db.ExecContext(ctx, `
		INSERT INTO models (id, name, role, model, url, api_key, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, m.ID, m.Name, m.Role, m.Model, m.URL, m.APIKey, m.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to add model: %w", err)
	}
	return nil
}

func (c *chat) ListModels(ctx context.Context) ([]models.Model, error) {
	rows, err := c.db.QueryContext(ctx, `
		SELECT id, name, role, model, url, COALESCE(api_key,''), COALESCE(created_at,'')
		FROM models
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list models: %w", err)
	}
	defer rows.Close()

	var list []models.Model
	for rows.Next() {
		var m models.Model
		if err := rows.Scan(
			&m.ID, &m.Name, &m.Role, &m.Model,
			&m.URL, &m.APIKey, &m.CreatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

func (c *chat) UpdateModel(ctx context.Context, m models.Model) error {
	if m.ID == "" {
		return fmt.Errorf("model id is required")
	}
	if m.Name == "" || m.Model == "" || m.URL == "" || m.Role == "" {
		return fmt.Errorf("name, model, url and role are required")
	}

	res, err := c.db.ExecContext(ctx, `
		UPDATE models
		SET name = ?, role = ?, model = ?, url = ?, api_key = ?
		WHERE id = ?
	`, m.Name, m.Role, m.Model, m.URL, m.APIKey, m.ID)
	if err != nil {
		return fmt.Errorf("failed to update model: %w", err)
	}

	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("model not found: %s", m.ID)
	}
	return nil
}

func (c *chat) DeleteModel(ctx context.Context, modelID string) error {
	_, err := c.db.ExecContext(ctx,
		`DELETE FROM convo_models WHERE model_id = ?`, modelID)
	if err != nil {
		return fmt.Errorf("failed to remove model assignments: %w", err)
	}

	_, err = c.db.ExecContext(ctx,
		`DELETE FROM models WHERE id = ?`, modelID)
	if err != nil {
		return fmt.Errorf("failed to delete model: %w", err)
	}
	return nil
}

func (c *chat) SetActiveModel(ctx context.Context, chatID string, role string, modelID string) error {
	if chatID == "" || role == "" || modelID == "" {
		return fmt.Errorf("chatID, role and modelID are required")
	}

	_, err := c.db.ExecContext(ctx, `
		INSERT INTO convo_models (convo_id, role, model_id)
		VALUES (?, ?, ?)
		ON CONFLICT(convo_id, role) DO UPDATE SET model_id = excluded.model_id
	`, chatID, role, modelID)
	if err != nil {
		return fmt.Errorf("failed to set active model: %w", err)
	}
	return nil
}

func (c *chat) DeactivateModel(ctx context.Context, chatID string, modelID string) error {
	if chatID == "" || modelID == "" {
		return fmt.Errorf("chatID and modelID are required")
	}

	_, err := c.db.ExecContext(ctx, `
		DELETE FROM convo_models
		WHERE convo_id = ? AND model_id = ?
	`, chatID, modelID)
	if err != nil {
		return fmt.Errorf("failed to deactivate model: %w", err)
	}
	return nil
}

func (c *chat) ActiveModelIDs(ctx context.Context, chatID string) ([]string, error) {
	if chatID == "" {
		return nil, nil
	}

	rows, err := c.db.QueryContext(ctx, `
		SELECT model_id FROM convo_models WHERE convo_id = ?
	`, chatID)
	if err != nil {
		return nil, fmt.Errorf("failed to list active models: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// ActiveModels returns the full model rows activated for a chat.
func (c *chat) ActiveModels(ctx context.Context, chatID string) ([]models.Model, error) {
	if chatID == "" {
		return nil, nil
	}

	rows, err := c.db.QueryContext(ctx, `
		SELECT m.id, m.name, m.role, m.model, m.url, COALESCE(m.api_key,''), COALESCE(m.created_at,'')
		FROM convo_models cm
		JOIN models m ON cm.model_id = m.id
		WHERE cm.convo_id = ?
		ORDER BY m.role
	`, chatID)
	if err != nil {
		return nil, fmt.Errorf("failed to list active models: %w", err)
	}
	defer rows.Close()

	var list []models.Model
	for rows.Next() {
		var m models.Model
		if err := rows.Scan(
			&m.ID, &m.Name, &m.Role, &m.Model, &m.URL, &m.APIKey, &m.CreatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	return list, rows.Err()
}
