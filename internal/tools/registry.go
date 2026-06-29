package tools

import "github.com/ollama/ollama/api"

func DefaultTools() []api.Tool {
	return []api.Tool{
		executeTool(),
		fsTool(),
		askTool(),
		dagTool(),
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
	props := api.NewToolPropertiesMap()
	props.Set("dag", api.ToolProperty{
		Type: api.PropertyType{"string"},
		Description: `The execution plan as a JSON object: ` +
			`{"id","objective","failure_policy":"block|skip|recover",` +
			`"tasks":[{"id","title","description","dependencies":[...],` +
			`"inputs":[...],"outputs":[...],"status":"pending"}]}. ` +
			`Task ids must be unique, dependencies must reference existing tasks, ` +
			`and the graph must be acyclic.`,
	})

	return api.Tool{
		Type: "function",
		Function: api.ToolFunction{
			Name:        "create_dag",
			Description: "Validate a task DAG (unique ids, resolvable dependencies, no cycles, consistent inputs/outputs) and persist it to .synapse/dag.json.",
			Parameters: api.ToolFunctionParameters{
				Type:       "object",
				Required:   []string{"dag"},
				Properties: props,
			},
		},
	}
}
