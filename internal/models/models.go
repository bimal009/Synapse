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
