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

	seen := make(map[string]string) // tool-call signature -> result, to break loops
	lastText := ""                  // last real (non-JSON) assistant text, for fallback

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

		if raw := strings.TrimSpace(content.String()); raw != "" && !looksLikeToolCallJSON(raw) {
			lastText = raw
		}

		// qwen2.5-coder emits tool calls as plain JSON text even with stream:false,
		// sometimes several concatenated in one message. Parse all of them inline
		// if no structured tool calls came back.
		if len(toolCalls) == 0 {
			for _, tc := range parseInlineToolCalls(content.String()) {
				if _, known := a.toolFuncs[tc.Function.Name]; known {
					toolCalls = append(toolCalls, tc)
				}
			}
			if len(toolCalls) > 0 {
				// Content was just the raw JSON calls, not a real reply.
				content.Reset()
			}
		}

		if len(toolCalls) == 0 {
			text := strings.TrimSpace(content.String())
			// Don't leak a malformed/unknown tool-call JSON to the user. Nudge the
			// model to answer in plain text and try again.
			if looksLikeToolCallJSON(text) {
				messages = append(messages,
					api.Message{Role: "assistant", Content: text},
					api.Message{Role: "user", Content: "That was not a valid tool call. Using the results you already have, answer my question directly in plain text — do not output JSON or call more tools."},
				)
				continue
			}
			return text, nil
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

	if lastText != "" {
		return lastText, nil
	}
	return "", fmt.Errorf("tool loop exceeded %d turns", maxToolTurns)
}

func looksLikeToolCallJSON(s string) bool {
	s = strings.TrimSpace(s)
	if i := strings.Index(s, "```"); i != -1 {
		s = strings.TrimSpace(s[i+3:])
		s = strings.TrimPrefix(s, "json")
		s = strings.TrimSpace(s)
	}
	if !strings.HasPrefix(s, "{") {
		return false
	}
	return strings.Contains(s, `"name"`) && strings.Contains(s, `"arguments"`)
}

// parseInlineToolCalls extracts every JSON tool-call object the model emitted as
// plain text. Weak models sometimes concatenate several objects in one message
// (e.g. two `fs` reads back to back), so we decode them one after another rather
// than grabbing the span from the first "{" to the last "}".
func parseInlineToolCalls(content string) []api.ToolCall {
	s := strings.TrimSpace(content)
	// Strip a markdown fence if present.
	if i := strings.Index(s, "```"); i != -1 {
		s = s[i+3:]
		s = strings.TrimPrefix(s, "json")
		if j := strings.LastIndex(s, "```"); j != -1 {
			s = s[:j]
		}
	}
	start := strings.Index(s, "{")
	if start < 0 {
		return nil
	}

	dec := json.NewDecoder(strings.NewReader(s[start:]))
	var calls []api.ToolCall
	for {
		var tc struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments"`
		}
		if err := dec.Decode(&tc); err != nil {
			break
		}
		if tc.Name == "" {
			continue
		}
		calls = append(calls, api.ToolCall{
			Function: api.ToolCallFunction{
				Name:      tc.Name,
				Arguments: mustMarshalArgs(tc.Arguments),
			},
		})
	}
	return calls
}

func mustMarshalArgs(args map[string]any) api.ToolCallFunctionArguments {
	var out api.ToolCallFunctionArguments
	if b, err := json.Marshal(args); err == nil {
		_ = json.Unmarshal(b, &out)
	}
	return out
}
