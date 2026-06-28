package models

type TaskStatus string

const (
	StatusPending TaskStatus = "pending"
	StatusRunning TaskStatus = "running"
	StatusDone    TaskStatus = "done"
	StatusFailed  TaskStatus = "failed"
)

type Task struct {
	ID        string     `json:"id" db:"id"`
	Name      string     `json:"name" db:"name"`
	AgentRole string     `json:"agent_role" db:"agent_role"`
	Prompt    string     `json:"prompt" db:"prompt"`
	DependsOn []string   `json:"depends_on" db:"depends_on"`
	SkillPath string     `json:"skill_path,omitempty" db:"skill_path"`
	Status    TaskStatus `json:"status" db:"status"`
	Result    string     `json:"result,omitempty" db:"result"`
	Err       string     `json:"error,omitempty" db:"error"`
}

type DAG struct {
	Goal  string           `json:"goal" db:"goal"`
	Tasks map[string]*Task `json:"tasks" db:"tasks"`
}
