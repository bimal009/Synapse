package configs

// configs
type ModelConfig struct {
	Model     string
	URL       string
	APIKey    string
	Streaming bool
	Thinking  bool
}
type Config struct {
	Env    string
	Models map[string]ModelConfig
}

func (c *Config) ForRole(role string) ModelConfig {
	if m, ok := c.Models[role]; ok {
		return m
	}
	return c.Models["default"]
}
