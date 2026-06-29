export interface ChatSession {
  id: string;
  title: string;
  createdAt: Date;
  updatedAt: Date;
  projectPath?: string; // empty if no project attached
}

export interface ChatMessage {
  id: string;
  role: "user" | "assistant";
  content: string;
  error?: boolean;
}

export interface ModelConfig {
  id?: string;
  name: string;
  role: string;
  model: string;
  url: string;
  api_key: string;
}
