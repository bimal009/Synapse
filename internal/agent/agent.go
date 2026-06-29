package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bimal009/Synapse/configs"
	"github.com/bimal009/Synapse/internal/tools"
	"github.com/ollama/ollama/api"
)

type ToolFunc func(ctx context.Context, args map[string]any) (string, error)

const maxToolTurns = 15

// Ollama defaults num_ctx to ~4k, which truncates a large create_dag argument
// (e.g. a multi-task DAG) mid-generation. Give the model a roomy window.
const numCtx = 16384

type Agent interface {
	Chat(ctx context.Context, messages []api.Message) (string, error)
}

type agent struct {
	model     string
	role      string
	client    *api.Client
	logger    *slog.Logger
	toolFuncs map[string]ToolFunc
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

func New(cfg configs.Config, role string, logger *slog.Logger, toolFuncs map[string]ToolFunc) (Agent, error) {
	modelCfg := cfg.ForRole(role)

	parsedURL, err := url.Parse(modelCfg.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid url for role %s: %w", role, err)
	}
	parsedURL.Path = ""
	parsedURL.RawQuery = ""
	parsedURL.Fragment = ""

	client := api.NewClient(parsedURL, &http.Client{
		Timeout: 60 * time.Minute,
		Transport: &authTransport{
			key:  modelCfg.APIKey,
			base: http.DefaultTransport,
		},
	})

	return &agent{
		model:     strings.TrimSpace(modelCfg.Model),
		role:      role,
		client:    client,
		logger:    logger,
		toolFuncs: toolFuncs,
	}, nil
}

func (a *agent) runTool(ctx context.Context, name string, args map[string]any) string {
	fn, ok := a.toolFuncs[name]
	if !ok {
		return fmt.Sprintf("error: unknown tool %q", name)
	}
	out, err := fn(ctx, args)
	if err != nil {
		return "error: " + err.Error()
	}
	return out
}

func toArgMap(a api.ToolCallFunctionArguments) map[string]any {
	var m map[string]any
	if b, err := json.Marshal(a); err == nil {
		_ = json.Unmarshal(b, &m)
	}
	return m
}

func (a *agent) Chat(ctx context.Context, messages []api.Message) (string, error) {
	stream := false
	toolset := tools.DefaultTools()

	a.logger.Info("agent request",
		"role", a.role,
		"model", a.model,
		"messages", len(messages),
		"tools", len(toolset),
	)

	seen := make(map[string]string) // tool-call signature -> result, to break repeat loops

	for turn := 0; turn < maxToolTurns; turn++ {
		var (
			content   strings.Builder
			toolCalls []api.ToolCall
		)

		err := a.client.Chat(ctx, &api.ChatRequest{
			Model:    a.model,
			Messages: messages,
			Stream:   &stream,
			Tools:    toolset,
			Options:  map[string]any{"num_ctx": numCtx},
		}, func(resp api.ChatResponse) error {
			content.WriteString(resp.Message.Content)
			toolCalls = append(toolCalls, resp.Message.ToolCalls...)
			if resp.Done {
				a.logger.Info("agent response",
					"role", a.role,
					"done_reason", resp.DoneReason,
					"tool_calls", len(toolCalls),
					"content", content.String(),
				)
			}
			return nil
		})
		if err != nil {
			return "", fmt.Errorf("agent chat [turn %d]: %w", turn, err)
		}

		if len(toolCalls) == 0 {
			return strings.TrimSpace(content.String()), nil
		}

		messages = append(messages, api.Message{
			Role:      "assistant",
			Content:   content.String(),
			ToolCalls: toolCalls,
		})

		for _, tc := range toolCalls {
			name := tc.Function.Name
			args := toArgMap(tc.Function.Arguments)

			sigBytes, _ := json.Marshal(args)
			sig := name + "|" + string(sigBytes)

			var result string
			if prev, dup := seen[sig]; dup {
				result = fmt.Sprintf("You already called %q with the same arguments; its result was: %s\nDo not call it again — use this result to write your final plain-text answer to the user now.", name, prev)
				a.logger.Info("tool call deduped", "role", a.role, "name", name)
			} else {
				a.logger.Info("tool call", "role", a.role, "name", name, "args", args)
				result = a.runTool(ctx, name, args)
				a.logger.Info("tool result", "role", a.role, "name", name, "result", result)
				seen[sig] = result
			}

			messages = append(messages, api.Message{
				Role:     "tool",
				ToolName: name,
				Content:  result,
			})
		}
	}

	return "", fmt.Errorf("tool loop exceeded %d turns", maxToolTurns)
}
