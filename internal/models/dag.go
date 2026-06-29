package models

type RetryPolicy struct {
	MaxAttempts    int    `json:"max_attempts"`
	Backoff        string `json:"backoff"`
	BackoffSeconds int    `json:"backoff_seconds"`
}

type TaskState struct {
	ID        string `json:"id"         db:"id"`
	RunID     string `json:"run_id"     db:"run_id"`
	TaskID    string `json:"task_id"    db:"task_id"`
	Status    string `json:"status"     db:"status"`
	Attempts  int    `json:"attempts"   db:"attempts"`
	OutputRef string `json:"output_ref" db:"output_ref"`
	UpdatedAt string `json:"updated_at" db:"updated_at"`
}

type Dag struct {
	ID            string `json:"id"             validate:"required"`
	Objective     string `json:"objective"      validate:"required"`
	FailurePolicy string `json:"failure_policy" validate:"required,oneof=block skip recover"`
	Tasks         []Task `json:"tasks"          validate:"required,min=1,dive"`
}

type Task struct {
	ID           string       `json:"id"           validate:"required"`
	Title        string       `json:"title"        validate:"required"`
	Description  string       `json:"description"  validate:"required"`
	Objective    string       `json:"objective"`
	Inputs       []string     `json:"inputs,omitempty"`
	Outputs      []string     `json:"outputs,omitempty"`
	Dependencies []string     `json:"dependencies"`
	Status       string       `json:"status"       validate:"required,oneof=pending blocked ready waiting running validating completed failed retrying skipped cancelled"`
	Validation   []string     `json:"validation,omitempty"`
	RetryPolicy  *RetryPolicy `json:"retry_policy,omitempty"`
	Timeout      int          `json:"timeout,omitempty"      validate:"omitempty,min=1"`
	Priority     int          `json:"priority,omitempty"`
	Tags         []string     `json:"tags,omitempty"`
	Owner        string       `json:"owner,omitempty"`
	ModelRole    string       `json:"model_role,omitempty" validate:"omitempty,oneof=fast coding reasoning planning"`
}
