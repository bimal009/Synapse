package main

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"sync"

	"github.com/bimal009/Synapse/configs"
	"github.com/bimal009/Synapse/internal/agent"
	"github.com/bimal009/Synapse/internal/config"
	"github.com/bimal009/Synapse/internal/logger"
	"github.com/bimal009/Synapse/internal/models"
	"github.com/bimal009/Synapse/internal/project"
	"github.com/bimal009/Synapse/internal/tools"
	"github.com/ollama/ollama/api"
)

//go:embed prompts/system.md
var systemPrompt string

type App struct {
	ctx          context.Context
	cfg          configs.Config
	logger       *slog.Logger
	db           *sql.DB
	chat         project.Chat
	terminal     tools.Terminal
	dag          tools.Dag
	activeChatID string
	builtPrompt  string
}

func NewApp() *App {
	cfg := configs.Config{
		Env:    "dev",
		Models: map[string]configs.ModelConfig{},
	}
	return &App{
		cfg:    cfg,
		logger: logger.New(cfg.Env),
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	db, err := config.NewDBInitialize()
	if err != nil {
		a.logger.Error("failed to initialize db", "error", err)
		return
	}
	if err := db.Init(); err != nil {
		a.logger.Error("failed to run db migrations", "error", err)
		return
	}

	a.db = db.DB()
	a.chat = project.NewInit(a.db)
	a.terminal = tools.NewTerminal(a.db)
	a.dag = tools.NewDag()

	a.builtPrompt = strings.TrimSpace(systemPrompt) + fmt.Sprintf(
		"\n\n<env>\nOS: %s\nArch: %s\nShell: %s\n</env>",
		goruntime.GOOS,
		goruntime.GOARCH,
		shellName(),
	)

	a.logger.Info("startup complete", "os", goruntime.GOOS, "arch", goruntime.GOARCH)
}
func shellName() string {
	if goruntime.GOOS == "windows" {
		return `Windows Command Prompt (cmd.exe).
Execute commands using cmd.exe syntax only.

Available commands:
- dir        (list directory)
- type       (read a file)
- copy       (copy)
- move       (move)
- del        (delete file)
- rmdir      (remove directory)
- ren        (rename)
- mkdir      (create directory)
- cd         (change directory)
- echo       (print)
- findstr    (search in files)

Do NOT use:
- PowerShell cmdlets (Get-ChildItem, Get-Content, Copy-Item, Remove-Item, Select-String, Where-Object).
- Unix/Linux commands (ls, cat, grep, tail, sed, awk, rm, cp, mv).`
	}

	return `POSIX shell (sh/bash).
Execute commands using POSIX shell syntax.

Available commands:
- ls
- cat
- cp
- mv
- rm
- mkdir
- rmdir
- find
- grep
- tail
- head
- sed
- awk`
}
func (a *App) CreateChat(title string) (string, error) {
	if a.chat == nil {
		return "", fmt.Errorf("app not initialized")
	}
	id, err := a.chat.CreateChat(a.ctx, title)
	if err != nil {
		return "", err
	}
	a.activeChatID = id
	return id, nil
}

func (a *App) SetActiveChat(chatID string) {
	a.activeChatID = chatID
	a.cfg.Models = map[string]configs.ModelConfig{}
	if models, err := a.chat.LoadConfig(a.ctx, chatID); err == nil && len(models) > 0 {
		a.cfg.Models = models
	}
}

func (a *App) ListChats() ([]models.Chat, error) {
	if a.chat == nil {
		return nil, fmt.Errorf("app not initialized")
	}
	return a.chat.ListChats(a.ctx)
}

func (a *App) DeleteChat(chatID string) error {
	if a.chat == nil {
		return fmt.Errorf("app not initialized")
	}
	if a.activeChatID == chatID {
		a.activeChatID = ""
		a.cfg.Models = map[string]configs.ModelConfig{}
	}
	return a.chat.DeleteChat(a.ctx, chatID)
}

// ── Folder / project attachment ───────────────────────────────────────────────

func (a *App) SelectFolder() (string, error) {
	if a.chat == nil {
		return "", fmt.Errorf("app not initialized")
	}
	if a.activeChatID == "" {
		return "", fmt.Errorf("no active chat")
	}
	return a.chat.SelectFolder(a.ctx, a.activeChatID)
}

func (a *App) TrustFolder() error {
	if a.chat == nil {
		return fmt.Errorf("app not initialized")
	}
	if a.activeChatID == "" {
		return fmt.Errorf("no active chat")
	}
	if err := a.chat.TrustFolder(a.ctx, a.activeChatID); err != nil {
		return err
	}
	if chatModels, err := a.chat.LoadConfig(a.ctx, a.activeChatID); err == nil && len(chatModels) > 0 {
		a.cfg.Models = chatModels
	}
	return nil
}

func (a *App) IsTrusted() (bool, error) {
	if a.chat == nil {
		return false, fmt.Errorf("app not initialized")
	}
	if a.activeChatID == "" {
		return false, nil
	}
	return a.chat.IsTrusted(a.ctx, a.activeChatID)
}

func (a *App) GetPath() (string, error) {
	if a.chat == nil {
		return "", fmt.Errorf("app not initialized")
	}
	if a.activeChatID == "" {
		return "", nil
	}
	return a.chat.Path(a.ctx, a.activeChatID)
}

// ── Models ────────────────────────────────────────────────────────────────────

func (a *App) AddModel(m models.Model) error {
	if a.chat == nil {
		return fmt.Errorf("app not initialized")
	}
	return a.chat.AddModel(a.ctx, m)
}

func (a *App) UpdateModel(m models.Model) error {
	if a.chat == nil {
		return fmt.Errorf("app not initialized")
	}
	return a.chat.UpdateModel(a.ctx, m)
}

func (a *App) ListModels() ([]models.Model, error) {
	if a.chat == nil {
		return nil, fmt.Errorf("app not initialized")
	}
	return a.chat.ListModels(a.ctx)
}

func (a *App) DeleteModel(modelID string) error {
	if a.chat == nil {
		return fmt.Errorf("app not initialized")
	}
	return a.chat.DeleteModel(a.ctx, modelID)
}

func (a *App) SetActiveModel(role string, modelID string) error {
	if a.chat == nil {
		return fmt.Errorf("app not initialized")
	}
	if a.activeChatID == "" {
		return fmt.Errorf("no active chat")
	}
	if err := a.chat.SetActiveModel(a.ctx, a.activeChatID, role, modelID); err != nil {
		return err
	}
	if chatModels, err := a.chat.LoadConfig(a.ctx, a.activeChatID); err == nil {
		a.cfg.Models = chatModels
	}
	return nil
}

func (a *App) DeactivateModel(modelID string) error {
	if a.chat == nil {
		return fmt.Errorf("app not initialized")
	}
	if a.activeChatID == "" {
		return fmt.Errorf("no active chat")
	}
	if err := a.chat.DeactivateModel(a.ctx, a.activeChatID, modelID); err != nil {
		return err
	}
	if chatModels, err := a.chat.LoadConfig(a.ctx, a.activeChatID); err == nil {
		a.cfg.Models = chatModels
	}
	return nil
}

func (a *App) ActiveModelIDs() ([]string, error) {
	if a.chat == nil {
		return nil, fmt.Errorf("app not initialized")
	}
	if a.activeChatID == "" {
		return nil, nil
	}
	return a.chat.ActiveModelIDs(a.ctx, a.activeChatID)
}

func (a *App) ActiveModels() ([]models.Model, error) {
	if a.chat == nil {
		return nil, fmt.Errorf("app not initialized")
	}
	if a.activeChatID == "" {
		return nil, nil
	}
	return a.chat.ActiveModels(a.ctx, a.activeChatID)
}

// ── Terminal ──────────────────────────────────────────────────────────────────

func (a *App) ExecuteCommand(command string) (string, error) {
	if a.terminal == nil || a.chat == nil {
		return "", fmt.Errorf("app not initialized")
	}
	if a.activeChatID == "" {
		return "", fmt.Errorf("no active chat")
	}
	path, err := a.chat.Path(a.ctx, a.activeChatID)
	if err != nil {
		return "", err
	}
	if path == "" {
		return "", fmt.Errorf("no project attached to this chat")
	}
	return a.terminal.Execute(a.ctx, a.activeChatID, strings.Fields(command), path)
}

func (a *App) projectPath() (string, error) {
	if a.terminal == nil || a.chat == nil {
		return "", fmt.Errorf("app not initialized")
	}
	if a.activeChatID == "" {
		return "", fmt.Errorf("no active chat")
	}
	path, err := a.chat.Path(a.ctx, a.activeChatID)
	if err != nil {
		return "", err
	}
	if path == "" {
		return "", fmt.Errorf("no project attached to this chat")
	}
	return path, nil
}

func (a *App) FsCreate(path, content string) (string, error) {
	projectPath, err := a.projectPath()
	if err != nil {
		return "", err
	}
	return a.terminal.FsCreate(a.ctx, a.activeChatID, projectPath, path, content)
}

func (a *App) FsRead(path string) (string, error) {
	projectPath, err := a.projectPath()
	if err != nil {
		return "", err
	}
	return a.terminal.FsRead(a.ctx, a.activeChatID, projectPath, path)
}

func (a *App) FsUpdate(path, content string) (string, error) {
	projectPath, err := a.projectPath()
	if err != nil {
		return "", err
	}
	return a.terminal.FsUpdate(a.ctx, a.activeChatID, projectPath, path, content)
}

func (a *App) FsReplace(path, oldText, newText string) (string, error) {
	projectPath, err := a.projectPath()
	if err != nil {
		return "", err
	}
	return a.terminal.FsReplace(a.ctx, a.activeChatID, projectPath, path, oldText, newText)
}

func (a *App) FsDelete(path string) (string, error) {
	projectPath, err := a.projectPath()
	if err != nil {
		return "", err
	}
	return a.terminal.FsDelete(a.ctx, a.activeChatID, projectPath, path)
}

func (a *App) SetPermission(action, rule string) error {
	if a.chat == nil {
		return fmt.Errorf("app not initialized")
	}
	if a.activeChatID == "" {
		return fmt.Errorf("no active chat")
	}
	return a.chat.SetPermission(a.ctx, a.activeChatID, action, rule)
}

func (a *App) ListPermissions() ([]models.Permission, error) {
	if a.chat == nil {
		return nil, fmt.Errorf("app not initialized")
	}
	if a.activeChatID == "" {
		return nil, nil
	}
	return a.chat.ListPermissions(a.ctx, a.activeChatID)
}

// ── Agent ─────────────────────────────────────────────────────────────────────

func (a *App) Greet(name string) string {
	return "Hello " + name + ", Synapse is ready!"
}

func (a *App) buildToolFuncs(chatID, projectPath string) map[string]agent.ToolFunc {
	return map[string]agent.ToolFunc{
		"current_time": func(ctx context.Context, args map[string]any) (string, error) {
			tz, _ := args["timezone"].(string)
			return tools.CurrentTime(ctx, tz)
		},
		"execute": func(ctx context.Context, args map[string]any) (string, error) {
			command, _ := args["command"].(string)
			return a.terminal.Execute(ctx, chatID, strings.Fields(command), projectPath)
		},
		"fs": func(ctx context.Context, args map[string]any) (string, error) {
			op, _ := args["operation"].(string)
			path, _ := args["path"].(string)
			content, _ := args["content"].(string)
			oldText, _ := args["old"].(string)
			newText, _ := args["new"].(string)
			switch strings.ToLower(strings.TrimSpace(op)) {
			case "create":
				return a.terminal.FsCreate(ctx, chatID, projectPath, path, content)
			case "read":
				return a.terminal.FsRead(ctx, chatID, projectPath, path)
			case "update":
				return a.terminal.FsUpdate(ctx, chatID, projectPath, path, content)
			case "replace":
				return a.terminal.FsReplace(ctx, chatID, projectPath, path, oldText, newText)
			case "delete":
				return a.terminal.FsDelete(ctx, chatID, projectPath, path)
			default:
				return "", fmt.Errorf("unknown fs operation %q", op)
			}
		},
		"ask_permission": func(ctx context.Context, args map[string]any) (string, error) {
			action, _ := args["action"].(string)
			req, err := a.terminal.Ask(ctx, chatID, action)
			if err != nil {
				return "", err
			}
			if req.Allow {
				return "The user approved the action.", nil
			}
			return "The user denied the action.", nil
		},
		"create_dag": func(ctx context.Context, args map[string]any) (string, error) {
			data, err := dagArgBytes(args["dag"])
			if err != nil {
				return "", err
			}
			var g models.Dag
			if err := json.Unmarshal(data, &g); err != nil {
				return "", fmt.Errorf("invalid dag json: %w", err)
			}
			// Default missing task statuses so validation can focus on structure.
			for i := range g.Tasks {
				if strings.TrimSpace(g.Tasks[i].Status) == "" {
					g.Tasks[i].Status = "pending"
				}
			}
			if err := a.dag.Validates(ctx, g); err != nil {
				return "", err
			}
			if projectPath == "" {
				return "", fmt.Errorf("no project attached to this chat")
			}
			dir := filepath.Join(projectPath, ".synapse", "dag")
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return "", fmt.Errorf("create .synapse/dag dir: %w", err)
			}
			path, err := a.dag.CreateJson(ctx, g, filepath.Join(dir, "dag.json"))
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("DAG validated and saved to %s (%d tasks).", path, len(g.Tasks)), nil
		},
	}
}

// dagArgBytes normalizes the "dag" tool argument, which models may emit either
// as a JSON string or as an already-parsed object, into raw JSON bytes.
func dagArgBytes(raw any) ([]byte, error) {
	switch v := raw.(type) {
	case nil:
		return nil, fmt.Errorf("dag is required")
	case string:
		if strings.TrimSpace(v) == "" {
			return nil, fmt.Errorf("dag is required")
		}
		return []byte(v), nil
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("invalid dag: %w", err)
		}
		return b, nil
	}
}

func (a *App) RunAgents(prompt string) (string, error) {
	if a.activeChatID == "" {
		return "", fmt.Errorf("no active chat")
	}
	trusted, err := a.chat.IsTrusted(a.ctx, a.activeChatID)
	if err != nil {
		return "", err
	}
	if !trusted {
		return "", fmt.Errorf("no project attached to this chat")
	}
	if len(a.cfg.Models) == 0 {
		return "", fmt.Errorf("no models configured")
	}

	messages := make([]api.Message, 0, 2)
	if sp := strings.TrimSpace(a.builtPrompt); sp != "" {
		messages = append(messages, api.Message{Role: "system", Content: sp})
	}
	messages = append(messages, api.Message{Role: "user", Content: prompt})

	projectPath, _ := a.chat.Path(a.ctx, a.activeChatID)
	toolFuncs := a.buildToolFuncs(a.activeChatID, projectPath)

	a.logger.Info("run agents",
		"chat", a.activeChatID,
		"roles", len(a.cfg.Models),
		"prompt", prompt,
	)

	var (
		mu      sync.Mutex
		results = make(map[string]string)
		wg      sync.WaitGroup
	)

	for role := range a.cfg.Models {
		wg.Add(1)
		go func(role string) {
			defer wg.Done()

			out := ""
			ag, err := agent.New(a.cfg, role, a.logger, toolFuncs)
			if err == nil {
				out, err = ag.Chat(a.ctx, messages)
			}
			if err != nil {
				a.logger.Error("agent error", "role", role, "error", err)
				out = "Error: " + err.Error()
			} else {
				a.logger.Info("agent result", "role", role, "chars", len(out))
			}

			mu.Lock()
			results[role] = out
			mu.Unlock()
		}(role)
	}

	wg.Wait()

	if len(results) == 1 {
		for _, out := range results {
			return out, nil
		}
	}

	var sb strings.Builder
	for role, out := range results {
		fmt.Fprintf(&sb, "[%s]\n%s\n\n", role, out)
	}
	return strings.TrimSpace(sb.String()), nil
}
