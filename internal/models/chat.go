package models

import "time"

type Chat struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	ProjectPath string `json:"project_path"`
	CreatedAt   string `json:"created_at"`
	LastOpened  string `json:"last_opened"` // empty string if null
}

type ChatMessage struct {
	ID        string    `json:"id"         db:"id"`
	ChatID    string    `json:"chat_id"    db:"chat_id"`
	Role      string    `json:"role"       db:"role"` // "user" | "assistant"
	Content   string    `json:"content"    db:"content"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type ChatWithMessages struct {
	Chat
	Messages []ChatMessage `json:"messages"`
}
