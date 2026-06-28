package agent

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/bimal009/Synapse/configs"
	"github.com/ollama/ollama/api"
)

type Agent interface {
	Chat(ctx context.Context, messages []api.Message) error
}

type agent struct {
	model  string
	role   string
	client *api.Client
	logger *slog.Logger
}

type authTransport struct {
	key  string
	base http.RoundTripper
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.key != "" {
		req.Header.Set("Authorization", "Bearer "+t.key)
	}
	return t.base.RoundTrip(req)
}

func New(cfg configs.Config, role string, logger *slog.Logger) (Agent, error) {
	modelCfg := cfg.ForRole(role)

	parsedURL, err := url.Parse(modelCfg.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid url for role %s: %w", role, err)
	}

	httpClient := &http.Client{
		Timeout: 120 * time.Second,
		Transport: &authTransport{
			key:  modelCfg.APIKey,
			base: http.DefaultTransport,
		},
	}

	client := api.NewClient(parsedURL, httpClient)

	return &agent{
		model:  modelCfg.Model,
		role:   role,
		client: client,
		logger: logger,
	}, nil
}

func (a *agent) Chat(ctx context.Context, messages []api.Message) error {
	stream := true

	req := &api.ChatRequest{
		Model:    a.model,
		Messages: messages,
		Stream:   &stream,
	}

	return a.client.Chat(ctx, req, func(resp api.ChatResponse) error {
		fmt.Print(resp.Message.Content)

		if resp.Done {
			a.logger.Info("response complete", "role", a.role, "content", resp.Message.Content)
		}
		return nil
	})
}
