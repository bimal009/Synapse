package models

type Model struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Role      string `json:"role"`
	Model     string `json:"model"`
	URL       string `json:"url"`
	APIKey    string `json:"api_key,omitempty"`
	CreatedAt string `json:"created_at"` // string, not time.Time
}
type ConvoModel struct {
	ConvoID string `json:"convo_id" db:"convo_id"`
	Role    string `json:"role"     db:"role"`
	ModelID string `json:"model_id" db:"model_id"`
}

type Role struct {
	Name        string `json:"name"        db:"name"`
	Description string `json:"description" db:"description"`
}

const (
	RolePlanner    = "planner"
	RoleReasoning  = "reasoning"
	RoleCoder      = "coder"
	RoleTester     = "tester"
	RoleQA         = "qa"
	RoleAnalyst    = "analyst"
	RoleReviewer   = "reviewer"
	RoleResearcher = "researcher"
	RoleVision     = "vision"
	RoleArchitect  = "architect"
	RoleDesigner   = "designer"
	RoleDevOps     = "devops"
	RoleSecurity   = "security"
	RoleWriter     = "writer"
	RoleEditor     = "editor"
	RoleData       = "data"
	RoleSummarizer = "summarizer"
	RoleGeneral    = "general"
)

func AllRoles() []string {
	return []string{
		RolePlanner,
		RoleReasoning,
		RoleCoder,
		RoleTester,
		RoleQA,
		RoleAnalyst,
		RoleReviewer,
		RoleResearcher,
		RoleVision,
		RoleArchitect,
		RoleDesigner,
		RoleDevOps,
		RoleSecurity,
		RoleWriter,
		RoleEditor,
		RoleData,
		RoleSummarizer,
		RoleGeneral,
	}
}

var RoleDescriptions = map[string]string{
	RolePlanner:    "Decomposes objectives and builds the task DAG.",
	RoleReasoning:  "Complex multi-step reasoning and analysis.",
	RoleCoder:      "Writes and edits code.",
	RoleTester:     "Writes and runs tests.",
	RoleQA:         "Verifies correctness and overall quality.",
	RoleAnalyst:    "Analyzes data and synthesizes findings.",
	RoleReviewer:   "Reviews code and designs for issues.",
	RoleResearcher: "Gathers and synthesizes information.",
	RoleVision:     "Understands images and visual content.",
	RoleArchitect:  "Designs system and software architecture.",
	RoleDesigner:   "UI/UX and visual design.",
	RoleDevOps:     "Infrastructure, CI/CD, and deployment.",
	RoleSecurity:   "Security review and threat analysis.",
	RoleWriter:     "Documentation and content writing.",
	RoleEditor:     "Proofreads and refines text.",
	RoleData:       "Data engineering and pipelines.",
	RoleSummarizer: "Condenses long content into summaries.",
	RoleGeneral:    "General-purpose default agent.",
}

func IsValidRole(role string) bool {
	for _, r := range AllRoles() {
		if r == role {
			return true
		}
	}
	return false
}
