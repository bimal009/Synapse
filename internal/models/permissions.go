package models

const (
	RuleAllow  = "allow"
	RuleAsk    = "ask"
	RuleDeny   = "deny"
	RuleAlways = "always"
)

const (
	DecisionAllowed      = "allowed"
	DecisionDenied       = "denied"
	DecisionAskedAllowed = "asked:allowed"
	DecisionAskedDenied  = "asked:denied"
)

const (
	SourceRule    = "rule"
	SourceUser    = "user"
	SourceDefault = "default"
)

type Permission struct {
	ID        string `json:"id"               db:"id"`
	ConvoID   string `json:"convo_id"         db:"convo_id"`
	Action    string `json:"action"           db:"action"`
	Rule      string `json:"rule"             db:"rule"`   // allow | ask | deny | always
	Config    string `json:"config,omitempty" db:"config"` // JSON metadata
	CreatedAt string `json:"created_at"       db:"created_at"`
}

type PermissionEvent struct {
	ID        string `json:"id"         db:"id"`
	ConvoID   string `json:"convo_id"   db:"convo_id"`
	Action    string `json:"action"     db:"action"`
	Decision  string `json:"decision"   db:"decision"` // allowed | denied | asked:allowed | asked:denied
	Source    string `json:"source"     db:"source"`   // rule | user | default
	CreatedAt string `json:"created_at" db:"created_at"`
}
