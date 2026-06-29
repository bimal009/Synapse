package tools

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"sort"
	"strings"
	"time"

	"github.com/bimal009/Synapse/internal/models"
	"github.com/google/uuid"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type Request struct {
	Allow  bool
	Deny   bool
	Action string
}

const (
	respDeny   = "deny"
	respOnce   = "once"
	respAlways = "always"
)

type Terminal interface {
	Execute(ctx context.Context, chatID string, args []string, projectPath string) (string, error)
	Authorize(ctx context.Context, chatID, action string) (Request, error)
	CheckPermission(ctx context.Context, chatID, action string) (string, error)
	Ask(ctx context.Context, chatID, action string) (Request, error)
	FsCreate(ctx context.Context, chatID, projectPath, path, content string) (string, error)
	FsRead(ctx context.Context, chatID, projectPath, path string) (string, error)
	FsUpdate(ctx context.Context, chatID, projectPath, path, content string) (string, error)
	FsReplace(ctx context.Context, chatID, projectPath, path, oldText, newText string) (string, error)
	FsDelete(ctx context.Context, chatID, projectPath, path string) (string, error)
}

type terminal struct {
	db *sql.DB
}

func NewTerminal(db *sql.DB) Terminal {
	return &terminal{db: db}
}

func (t *terminal) Execute(ctx context.Context, chatID string, args []string, projectPath string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("no command provided")
	}
	action := strings.Join(args, " ")

	req, err := t.Authorize(ctx, chatID, action)
	if err != nil {
		return "", err
	}
	if !req.Allow {
		return "", fmt.Errorf("action %q not permitted", action)
	}

	dir, err := filepath.Abs(projectPath)
	if err != nil {
		return "", fmt.Errorf("invalid project path: %w", err)
	}
	if info, statErr := os.Stat(dir); statErr != nil || !info.IsDir() {
		return "", fmt.Errorf("project path is not a directory: %s", dir)
	}

	var cmd *exec.Cmd
	if goruntime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd.exe", append([]string{"/C"}, args...)...)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", action)
	}
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "TERM=dumb")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("command failed: %w", err)
	}
	return string(output), nil
}

func (t *terminal) Authorize(ctx context.Context, chatID, action string) (Request, error) {
	rule, err := t.CheckPermission(ctx, chatID, action)
	if err != nil {
		return Request{Deny: true, Action: action}, err
	}
	switch rule {
	case models.RuleAllow, models.RuleAlways:
		t.logEvent(ctx, chatID, action, models.DecisionAllowed, models.SourceRule)
		return Request{Allow: true, Action: action}, nil
	case models.RuleDeny:
		t.logEvent(ctx, chatID, action, models.DecisionDenied, models.SourceRule)
		return Request{Deny: true, Action: action}, fmt.Errorf("action %q denied by rule", action)
	case models.RuleAsk:
		return t.Ask(ctx, chatID, action)
	default:
		t.logEvent(ctx, chatID, action, models.DecisionDenied, models.SourceDefault)
		return Request{Deny: true, Action: action}, fmt.Errorf("no rule for action %q", action)
	}
}

func (t *terminal) CheckPermission(ctx context.Context, chatID, action string) (string, error) {
	if chatID == "" || action == "" {
		return "", fmt.Errorf("chatID and action are required")
	}
	var rule string
	err := t.db.QueryRowContext(ctx, `
		SELECT rule FROM permissions WHERE convo_id = ? AND action = ?
	`, chatID, action).Scan(&rule)
	if errors.Is(err, sql.ErrNoRows) {
		return models.RuleAsk, nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to check permission: %w", err)
	}
	return rule, nil
}

func (t *terminal) Ask(ctx context.Context, chatID, action string) (Request, error) {
	if action == "" {
		return Request{Deny: true}, fmt.Errorf("action is empty")
	}
	responseCh := make(chan string, 1)

	wailsruntime.EventsOnce(ctx, "ask:permission:response", func(optionalData ...interface{}) {
		if len(optionalData) == 0 {
			responseCh <- respDeny
			return
		}
		switch v := optionalData[0].(type) {
		case string:
			responseCh <- v
		case bool:
			if v {
				responseCh <- respOnce
			} else {
				responseCh <- respDeny
			}
		default:
			responseCh <- respDeny
		}
	})

	wailsruntime.EventsEmit(ctx, "ask:permission", map[string]string{"action": action})

	select {
	case resp := <-responseCh:
		switch resp {
		case respAlways:
			if err := t.savePermission(ctx, chatID, action, models.RuleAlways); err != nil {
				log.Printf("save permission failed: %v", err)
			}
			t.logEvent(ctx, chatID, action, models.DecisionAskedAllowed, models.SourceUser)
			return Request{Allow: true, Action: action}, nil
		case respOnce:
			t.logEvent(ctx, chatID, action, models.DecisionAskedAllowed, models.SourceUser)
			return Request{Allow: true, Action: action}, nil
		default:
			t.logEvent(ctx, chatID, action, models.DecisionAskedDenied, models.SourceUser)
			return Request{Deny: true, Action: action}, fmt.Errorf("user denied action %q", action)
		}
	case <-ctx.Done():
		t.logEvent(ctx, chatID, action, models.DecisionAskedDenied, models.SourceUser)
		return Request{Deny: true, Action: action}, fmt.Errorf("permission request cancelled: %w", ctx.Err())
	}
}

func (t *terminal) savePermission(ctx context.Context, chatID, action, rule string) error {
	const q = `
		INSERT INTO permissions (id, convo_id, action, rule, created_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(convo_id, action) DO UPDATE SET rule = excluded.rule`
	_, err := t.db.ExecContext(ctx, q,
		uuid.NewString(), chatID, action, rule, time.Now().UTC().Format(time.RFC3339))
	return err
}

func (t *terminal) logEvent(ctx context.Context, chatID, action, decision, source string) {
	ev := models.PermissionEvent{
		ID:        uuid.NewString(),
		ConvoID:   chatID,
		Action:    action,
		Decision:  decision,
		Source:    source,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	const q = `
		INSERT INTO permission_events
			(id, convo_id, action, decision, source, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`
	if _, err := t.db.ExecContext(ctx, q,
		ev.ID, ev.ConvoID, ev.Action, ev.Decision, ev.Source, ev.CreatedAt,
	); err != nil {
		log.Printf("permission_events insert failed: %v", err)
	}
}

func resolveInProject(projectPath, path string) (string, error) {
	if projectPath == "" {
		return "", fmt.Errorf("no project attached to this chat")
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("path is required")
	}
	root, err := filepath.Abs(projectPath)
	if err != nil {
		return "", fmt.Errorf("invalid project path: %w", err)
	}
	abs := path
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(root, abs)
	}
	abs = filepath.Clean(abs)
	rel, err := filepath.Rel(root, abs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("path %q escapes the project directory", path)
	}
	return abs, nil
}

func (t *terminal) authorizeFS(ctx context.Context, chatID, projectPath, action, path string) (string, error) {
	abs, err := resolveInProject(projectPath, path)
	if err != nil {
		return "", err
	}
	req, err := t.Authorize(ctx, chatID, action)
	if err != nil {
		return "", err
	}
	if !req.Allow {
		return "", fmt.Errorf("action %q not permitted", action)
	}
	return abs, nil
}

func (t *terminal) FsCreate(ctx context.Context, chatID, projectPath, path, content string) (string, error) {
	abs, err := t.authorizeFS(ctx, chatID, projectPath, "create file "+path, path)
	if err != nil {
		return "", err
	}
	if _, statErr := os.Stat(abs); statErr == nil {
		return "", fmt.Errorf("file already exists: %s (use the update operation to overwrite)", path)
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return "", fmt.Errorf("create parent dir: %w", err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("write %s: %w", path, err)
	}
	return fmt.Sprintf("Created %s (%d bytes).", path, len(content)), nil
}

func (t *terminal) FsRead(ctx context.Context, chatID, projectPath, path string) (string, error) {
	abs, err := t.authorizeFS(ctx, chatID, projectPath, "read file "+path, path)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return "", suggestNotFound(projectPath, path, "read")
		}
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	return string(data), nil
}

func (t *terminal) FsUpdate(ctx context.Context, chatID, projectPath, path, content string) (string, error) {
	abs, err := t.authorizeFS(ctx, chatID, projectPath, "update file "+path, path)
	if err != nil {
		return "", err
	}
	if _, statErr := os.Stat(abs); statErr != nil {
		return "", suggestNotFound(projectPath, path, "update")
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return "", fmt.Errorf("create parent dir: %w", err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("write %s: %w", path, err)
	}
	return fmt.Sprintf("Updated %s (%d bytes).", path, len(content)), nil
}

func (t *terminal) FsReplace(ctx context.Context, chatID, projectPath, path, oldText, newText string) (string, error) {
	if oldText == "" {
		return "", fmt.Errorf("old text is required for replace")
	}
	abs, err := t.authorizeFS(ctx, chatID, projectPath, "replace in file "+path, path)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return "", suggestNotFound(projectPath, path, "replace in")
		}
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	content := string(data)
	count := strings.Count(content, oldText)
	if count == 0 {
		return "", fmt.Errorf("text not found in %s", path)
	}
	content = strings.ReplaceAll(content, oldText, newText)
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("write %s: %w", path, err)
	}
	return fmt.Sprintf("Replaced %d occurrence(s) in %s.", count, path), nil
}

func (t *terminal) FsDelete(ctx context.Context, chatID, projectPath, path string) (string, error) {
	abs, err := t.authorizeFS(ctx, chatID, projectPath, "delete file "+path, path)
	if err != nil {
		return "", err
	}
	if _, statErr := os.Stat(abs); statErr != nil {
		return "", suggestNotFound(projectPath, path, "delete")
	}
	if err := os.RemoveAll(abs); err != nil {
		return "", fmt.Errorf("delete %s: %w", path, err)
	}
	return fmt.Sprintf("Deleted %s.", path), nil
}

func suggestNotFound(projectPath, path, verb string) error {
	root, err := filepath.Abs(projectPath)
	if err != nil {
		return fmt.Errorf("cannot %s %q: not found", verb, path)
	}
	matches := findSimilar(root, path)
	if len(matches) == 0 {
		return fmt.Errorf("cannot %s %q: not found. Explore the project first with execute (e.g. \"dir /s /b\" or \"findstr /s /i\") to find the correct path, then retry with an exact path", verb, path)
	}
	return fmt.Errorf("cannot %s %q: not found. Closest existing paths:\n  %s\nRetry with one of these exact paths", verb, path, strings.Join(matches, "\n  "))
}

func findSimilar(root, query string) []string {
	tokens := pathTokens(query)
	if len(tokens) == 0 {
		return nil
	}
	type hit struct {
		path  string
		score int
	}
	var hits []hit
	visited := 0
	filepath.WalkDir(root, func(p string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if d.IsDir() {
			switch strings.ToLower(d.Name()) {
			case ".git", "node_modules", "dist", "build", "vendor", ".next":
				return filepath.SkipDir
			}
			return nil
		}
		visited++
		if visited > 20000 {
			return filepath.SkipAll
		}
		rel, relErr := filepath.Rel(root, p)
		if relErr != nil {
			return nil
		}
		lower := strings.ToLower(filepath.ToSlash(rel))
		score := 0
		for _, tok := range tokens {
			if strings.Contains(lower, tok) {
				score++
			}
		}
		if score > 0 {
			hits = append(hits, hit{filepath.ToSlash(rel), score})
		}
		return nil
	})
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].score != hits[j].score {
			return hits[i].score > hits[j].score
		}
		return hits[i].path < hits[j].path
	})
	limit := 10
	if len(hits) < limit {
		limit = len(hits)
	}
	out := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		out = append(out, hits[i].path)
	}
	return out
}

func pathTokens(s string) []string {
	lower := strings.ToLower(s)
	fields := strings.FieldsFunc(lower, func(r rune) bool {
		return !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'))
	})
	seen := make(map[string]bool, len(fields))
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		if len(f) < 3 || seen[f] {
			continue
		}
		seen[f] = true
		out = append(out, f)
	}
	return out
}
