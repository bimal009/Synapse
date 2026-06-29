package project

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bimal009/Synapse/internal/models"
)

func marshalList(s []string) string {
	if len(s) == 0 {
		return "[]"
	}
	b, err := json.Marshal(s)
	if err != nil {
		return "[]"
	}
	return string(b)
}

func unmarshalList(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" || s == "[]" {
		return nil
	}
	var out []string
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return nil
	}
	return out
}

// SaveDag replaces this chat's DAG with dag, as a header row plus one row per
// task. Tasks already in progress (status changed from pending) keep their
// status across the rewrite.
func (c *chat) SaveDag(ctx context.Context, chatID string, dag models.Dag) error {
	if chatID == "" {
		return fmt.Errorf("chatID is required")
	}

	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	prev := map[string]string{}
	rows, err := tx.QueryContext(ctx,
		`SELECT task_id, COALESCE(status, '') FROM dag_tasks WHERE convo_id = ?`, chatID)
	if err == nil {
		for rows.Next() {
			var id, status string
			if err := rows.Scan(&id, &status); err == nil {
				prev[id] = status
			}
		}
		rows.Close()
	}

	now := dbTime(time.Now())
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO dags (convo_id, id, objective, failure_policy, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(convo_id) DO UPDATE SET
			id = excluded.id,
			objective = excluded.objective,
			failure_policy = excluded.failure_policy,
			updated_at = excluded.updated_at
	`, chatID, dag.ID, dag.Objective, dag.FailurePolicy, now); err != nil {
		return fmt.Errorf("save dag header: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM dag_tasks WHERE convo_id = ?`, chatID); err != nil {
		return fmt.Errorf("clear dag tasks: %w", err)
	}

	for i, t := range dag.Tasks {
		status := strings.TrimSpace(t.Status)
		if status == "" {
			status = "pending"
		}
		if ps, ok := prev[t.ID]; ok && ps != "" && ps != "pending" {
			status = ps // preserve progressed work
		}

		var rmax, rsec int
		var rback string
		if t.RetryPolicy != nil {
			rmax = t.RetryPolicy.MaxAttempts
			rback = t.RetryPolicy.Backoff
			rsec = t.RetryPolicy.BackoffSeconds
		}

		if _, err := tx.ExecContext(ctx, `
			INSERT INTO dag_tasks (
				convo_id, task_id, position, title, description, objective, status,
				model_role, owner, priority, timeout,
				retry_max_attempts, retry_backoff, retry_backoff_seconds,
				inputs, outputs, dependencies, validation, tags
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, chatID, t.ID, i, t.Title, t.Description, t.Objective, status,
			t.ModelRole, t.Owner, t.Priority, t.Timeout,
			rmax, rback, rsec,
			marshalList(t.Inputs), marshalList(t.Outputs), marshalList(t.Dependencies),
			marshalList(t.Validation), marshalList(t.Tags),
		); err != nil {
			return fmt.Errorf("insert task %q: %w", t.ID, err)
		}
	}

	return tx.Commit()
}

// GetDag reconstructs this chat's DAG from its header and task rows. The bool is
// false when no DAG exists for the chat.
func (c *chat) GetDag(ctx context.Context, chatID string) (models.Dag, bool, error) {
	if chatID == "" {
		return models.Dag{}, false, fmt.Errorf("chatID is required")
	}

	var dag models.Dag
	err := c.db.QueryRowContext(ctx, `
		SELECT id, COALESCE(objective, ''), COALESCE(failure_policy, '')
		FROM dags WHERE convo_id = ?
	`, chatID).Scan(&dag.ID, &dag.Objective, &dag.FailurePolicy)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Dag{}, false, nil
	}
	if err != nil {
		return models.Dag{}, false, fmt.Errorf("load dag header: %w", err)
	}

	rows, err := c.db.QueryContext(ctx, `
		SELECT task_id, COALESCE(title,''), COALESCE(description,''), COALESCE(objective,''),
		       COALESCE(status,''), COALESCE(model_role,''), COALESCE(owner,''),
		       priority, timeout, retry_max_attempts, COALESCE(retry_backoff,''), retry_backoff_seconds,
		       COALESCE(inputs,''), COALESCE(outputs,''), COALESCE(dependencies,''),
		       COALESCE(validation,''), COALESCE(tags,'')
		FROM dag_tasks WHERE convo_id = ? ORDER BY position
	`, chatID)
	if err != nil {
		return models.Dag{}, false, fmt.Errorf("load dag tasks: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var t models.Task
		var rmax, rsec int
		var rback string
		var inputs, outputs, deps, valid, tags string
		if err := rows.Scan(
			&t.ID, &t.Title, &t.Description, &t.Objective,
			&t.Status, &t.ModelRole, &t.Owner,
			&t.Priority, &t.Timeout, &rmax, &rback, &rsec,
			&inputs, &outputs, &deps, &valid, &tags,
		); err != nil {
			return models.Dag{}, false, err
		}
		t.Inputs = unmarshalList(inputs)
		t.Outputs = unmarshalList(outputs)
		t.Dependencies = unmarshalList(deps)
		t.Validation = unmarshalList(valid)
		t.Tags = unmarshalList(tags)
		if rmax != 0 || rback != "" || rsec != 0 {
			t.RetryPolicy = &models.RetryPolicy{MaxAttempts: rmax, Backoff: rback, BackoffSeconds: rsec}
		}
		dag.Tasks = append(dag.Tasks, t)
	}
	return dag, true, rows.Err()
}

// DeleteTask removes a single task from the chat's DAG. It refuses if the task's
// status has changed from pending (work already started).
func (c *chat) DeleteTask(ctx context.Context, chatID, taskID string) error {
	if chatID == "" || taskID == "" {
		return fmt.Errorf("chatID and task id are required")
	}

	var status string
	err := c.db.QueryRowContext(ctx,
		`SELECT COALESCE(status, '') FROM dag_tasks WHERE convo_id = ? AND task_id = ?`,
		chatID, taskID).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("no task %q in the plan", taskID)
	}
	if err != nil {
		return err
	}
	if status != "" && status != "pending" {
		return fmt.Errorf("refusing to delete task %q: its status is %q (already in progress)", taskID, status)
	}

	_, err = c.db.ExecContext(ctx,
		`DELETE FROM dag_tasks WHERE convo_id = ? AND task_id = ?`, chatID, taskID)
	return err
}
