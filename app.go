package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sync"

	"github.com/bimal009/Synapse/configs"
	"github.com/bimal009/Synapse/internal/agent"
	"github.com/bimal009/Synapse/internal/config"
	"github.com/bimal009/Synapse/internal/logger"
	"github.com/bimal009/Synapse/internal/project"
	"github.com/ollama/ollama/api"
)

type App struct {
	ctx     context.Context
	cfg     configs.Config
	logger  *slog.Logger
	db      *sql.DB
	project project.Project
}

func NewApp() *App {
	cfg := configs.Config{
		Env: "dev",
		Models: map[string]configs.ModelConfig{
			"planner": {Model: "llama3.2:3b", URL: "http://localhost:11434", Streaming: true, Thinking: "on"},
			"coder":   {Model: "qwen2.5:3b", URL: "http://localhost:11434", Streaming: true, Thinking: "on"},
		},
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
	a.project = project.NewInit(a.db)

	if path, err := a.project.Load(ctx); err != nil {
		a.logger.Error("failed to load last project", "error", err)
	} else if path != "" {
		a.logger.Info("restored project", "path", path)
	}

	a.logger.Info("startup complete", "db", a.db != nil, "project", a.project != nil)
}

func (a *App) SelectFolder() (string, error) {
	if a.project == nil {
		return "", fmt.Errorf("app not initialized")
	}
	return a.project.SelectFolder(a.ctx)
}

func (a *App) TrustFolder() error {
	return a.project.TrustFolder(a.ctx)
}

func (a *App) IsTrusted() bool {
	return a.project.IsTrusted()
}

// GetProject returns the path of the currently loaded (trusted) project,
// restored from the database. Empty string means no project is selected yet.
func (a *App) GetProject() (string, error) {
	if a.project == nil {
		return "", fmt.Errorf("app not initialized")
	}
	return a.project.Path(), nil
}

func (a *App) Greet(name string) string {
	return "Hello " + name + ", Synapse is ready!"
}

func (a *App) RunAgents(prompt string) {
	if !a.project.IsTrusted() {
		a.logger.Error("project not trusted, refusing to run agents")
		return
	}

	messages := []api.Message{
		{Role: "user", Content: prompt},
	}

	var wg sync.WaitGroup

	for role := range a.cfg.Models {
		wg.Add(1)
		go func(role string) {
			defer wg.Done()

			ag, err := agent.New(a.cfg, role, a.logger)
			if err != nil {
				a.logger.Error("failed to create agent", "role", role, "error", err)
				return
			}

			if err := ag.Chat(a.ctx, messages); err != nil {
				a.logger.Error("chat failed", "role", role, "error", err)
			}
		}(role)
	}

	wg.Wait()
}
