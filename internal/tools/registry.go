package tools

import "github.com/ollama/ollama/api"

func DefaultTools() []api.Tool {
	return []api.Tool{
		executeTool(),
		fsTool(),
		askTool(),
		dagTool(),
		getDagTool(),
		deleteDagTool(),
		currentTimeTool(),
	}
}

func currentTimeTool() api.Tool {
	props := api.NewToolPropertiesMap()
	props.Set("timezone", api.ToolProperty{
		Type:        api.PropertyType{"string"},
		Description: `Optional IANA timezone, e.g. "UTC" or "America/New_York". Defaults to UTC.`,
	})

	return api.Tool{
		Type: "function",
		Function: api.ToolFunction{
			Name:        "current_time",
			Description: "Get the current date and time as an RFC3339 string.",
			Parameters: api.ToolFunctionParameters{
				Type:       "object",
				Properties: props,
			},
		},
	}
}

func executeTool() api.Tool {
	props := api.NewToolPropertiesMap()
	props.Set("command", api.ToolProperty{
		Type:        api.PropertyType{"string"},
		Description: `The shell command to run in the project directory, e.g. "go test ./...".`,
	})

	return api.Tool{
		Type: "function",
		Function: api.ToolFunction{
			Name:        "execute",
			Description: "Execute a shell command inside the trusted project directory and return its output. Subject to the user's permission rules.",
			Parameters: api.ToolFunctionParameters{
				Type:       "object",
				Required:   []string{"command"},
				Properties: props,
			},
		},
	}
}

func askTool() api.Tool {
	props := api.NewToolPropertiesMap()
	props.Set("action", api.ToolProperty{
		Type:        api.PropertyType{"string"},
		Description: `A short identifier of the sensitive action you need to perform, e.g. "write file main.go" or "delete .env".`,
	})

	return api.Tool{
		Type: "function",
		Function: api.ToolFunction{
			Name: "ask_permission",
			Description: "Request the user's permission before performing a sensitive or irreversible action (writing/deleting files, running risky commands). " +
				"Returns whether the user approved or denied. " +
				"This is NOT for asking the user questions — ask any clarifying questions directly in your reply instead.",
			Parameters: api.ToolFunctionParameters{
				Type:       "object",
				Required:   []string{"action"},
				Properties: props,
			},
		},
	}
}

func fsTool() api.Tool {
	props := api.NewToolPropertiesMap()
	props.Set("operation", api.ToolProperty{
		Type:        api.PropertyType{"string"},
		Enum:        []any{"create", "read", "update", "replace", "delete"},
		Description: "The filesystem operation to perform.",
	})
	props.Set("path", api.ToolProperty{
		Type:        api.PropertyType{"string"},
		Description: `Project-relative file path, e.g. ".synapse/dag/prompts/build.prompt".`,
	})
	props.Set("content", api.ToolProperty{
		Type:        api.PropertyType{"string"},
		Description: "File contents. Required for create and update.",
	})
	props.Set("old", api.ToolProperty{
		Type:        api.PropertyType{"string"},
		Description: "For replace: the existing text to find.",
	})
	props.Set("new", api.ToolProperty{
		Type:        api.PropertyType{"string"},
		Description: "For replace: the text to substitute in.",
	})

	return api.Tool{
		Type: "function",
		Function: api.ToolFunction{
			Name:        "fs",
			Description: "Create, read, update, replace text in, or delete a file inside the project directory. Every operation is gated by the same permission rules as execute.",
			Parameters: api.ToolFunctionParameters{
				Type:       "object",
				Required:   []string{"operation", "path"},
				Properties: props,
			},
		},
	}
}

func dagTool() api.Tool {
	strItems := map[string]any{"type": "string"}
	taskSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id":           map[string]any{"type": "string", "description": "Unique snake_case task id (e.g. validate_schema)."},
			"title":        map[string]any{"type": "string", "description": "Short task title."},
			"description":  map[string]any{"type": "string", "description": "What the task does (recommended)."},
			"dependencies": map[string]any{"type": "array", "items": strItems, "description": "Ids of tasks that must finish before this one."},
			"inputs":       map[string]any{"type": "array", "items": strItems, "description": "Artifact names this task consumes."},
			"outputs":      map[string]any{"type": "array", "items": strItems, "description": "Artifact names this task produces."},
			"model_role":   map[string]any{"type": "string", "description": "Agent role that runs this task (one of the available roles)."},
			"priority":     map[string]any{"type": "integer", "description": "Higher = scheduled first among ready tasks. Optional, defaults to 0."},
		},
		"required": []any{"id", "title"},
	}

	props := api.NewToolPropertiesMap()
	props.Set("id", api.ToolProperty{
		Type:        api.PropertyType{"string"},
		Description: "Short identifier for the plan, e.g. chat-app-dag.",
	})
	props.Set("objective", api.ToolProperty{
		Type:        api.PropertyType{"string"},
		Description: "One-line statement of what the plan achieves.",
	})
	props.Set("failure_policy", api.ToolProperty{
		Type:        api.PropertyType{"string"},
		Enum:        []any{"block", "skip", "recover"},
		Description: "How the graph reacts to a task failure. Defaults to block.",
	})
	props.Set("tasks", api.ToolProperty{
		Type:        api.PropertyType{"array"},
		Items:       taskSchema,
		Description: "All tasks in the plan, as structured objects (not a JSON string).",
	})

	return api.Tool{
		Type: "function",
		Function: api.ToolFunction{
			Name:        "create_dag",
			Description: "Create or replace this chat's task DAG and validate it. Pass the plan as structured fields (id, objective, failure_policy, tasks[]) in one call — not a JSON string. Task ids must be unique and the graph must be acyclic. Tasks already in progress keep their status. On validation error, fix the plan and call create_dag again.",
			Parameters: api.ToolFunctionParameters{
				Type:       "object",
				Required:   []string{"id", "objective", "tasks"},
				Properties: props,
			},
		},
	}
}

func getDagTool() api.Tool {
	return api.Tool{
		Type: "function",
		Function: api.ToolFunction{
			Name:        "get_dag",
			Description: "Read the current task DAG for this chat and return it as JSON. Takes no arguments. Use this instead of reading any file to see the plan.",
			Parameters: api.ToolFunctionParameters{
				Type:       "object",
				Properties: api.NewToolPropertiesMap(),
			},
		},
	}
}

func deleteDagTool() api.Tool {
	props := api.NewToolPropertiesMap()
	props.Set("id", api.ToolProperty{
		Type:        api.PropertyType{"string"},
		Description: "Id of the task to remove from the plan.",
	})

	return api.Tool{
		Type: "function",
		Function: api.ToolFunction{
			Name:        "delete_dag",
			Description: "Delete a single task (by id) from this chat's DAG. Refuses if the task's status has changed (already in progress or done). Use it to drop a wrong task, then add a corrected one with create_dag.",
			Parameters: api.ToolFunctionParameters{
				Type:       "object",
				Required:   []string{"id"},
				Properties: props,
			},
		},
	}
}
