package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds the resolved configuration for oclaw.
type Config struct {
	GatewayURL string
	Token      string
	AgentID    string // default agent to connect to
	Agents     []AgentConfig
}

// AgentConfig represents a configured agent from openclaw.json.
type AgentConfig struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Model string `json:"model"`
	Emoji string `json:"emoji"`
}

// openclawConfig is the partial structure of ~/.openclaw/openclaw.json we need.
type openclawConfig struct {
	Gateway struct {
		Port   int    `json:"port"`
		Remote struct {
			URL string `json:"url"`
		} `json:"remote"`
		Auth struct {
			Mode  string `json:"mode"`
			Token string `json:"token"`
		} `json:"auth"`
	} `json:"gateway"`
	Agents struct {
		Default string `json:"default"`
		List    []struct {
			ID    string `json:"id"`
			Name  string `json:"name"`
			Emoji string `json:"emoji"`
			Model struct {
				Primary string `json:"primary"`
			} `json:"model"`
		} `json:"list"`
	} `json:"agents"`
}

// Load reads configuration with precedence: flags > env > config file > defaults.
func Load(flagURL, flagToken, flagAgent string) (*Config, error) {
	cfg := &Config{
		GatewayURL: "ws://127.0.0.1:39421",
		Token:      "ollama",
		AgentID:    "",
	}

	// Load from openclaw.json
	if err := cfg.loadFromFile(); err != nil {
		// Not fatal — may not exist
		fmt.Fprintf(os.Stderr, "warning: could not load openclaw.json: %v\n", err)
	}

	// Env vars override
	if v := os.Getenv("OPENCLAW_GATEWAY_URL"); v != "" {
		cfg.GatewayURL = v
	}
	if v := os.Getenv("OPENCLAW_GATEWAY_TOKEN"); v != "" {
		cfg.Token = v
	}
	if v := os.Getenv("OPENCLAW_AGENT"); v != "" {
		cfg.AgentID = v
	}

	// CLI flags override
	if flagURL != "" {
		cfg.GatewayURL = flagURL
	}
	if flagToken != "" {
		cfg.Token = flagToken
	}
	if flagAgent != "" {
		cfg.AgentID = flagAgent
	}

	return cfg, nil
}

func (cfg *Config) loadFromFile() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	path := filepath.Join(home, ".openclaw", "openclaw.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var oc openclawConfig
	if err := json.Unmarshal(data, &oc); err != nil {
		return fmt.Errorf("parse openclaw.json: %w", err)
	}

	// Gateway URL
	if oc.Gateway.Remote.URL != "" {
		cfg.GatewayURL = oc.Gateway.Remote.URL
	} else if oc.Gateway.Port != 0 {
		cfg.GatewayURL = fmt.Sprintf("ws://127.0.0.1:%d", oc.Gateway.Port)
	}

	// Auth token
	if oc.Gateway.Auth.Token != "" {
		cfg.Token = oc.Gateway.Auth.Token
	}

	// Default agent
	if oc.Agents.Default != "" && cfg.AgentID == "" {
		cfg.AgentID = oc.Agents.Default
	}

	// Agent list
	for _, a := range oc.Agents.List {
		cfg.Agents = append(cfg.Agents, AgentConfig{
			ID:    a.ID,
			Name:  a.Name,
			Model: a.Model.Primary,
			Emoji: a.Emoji,
		})
	}

	// If still no agent, use first in list
	if cfg.AgentID == "" && len(cfg.Agents) > 0 {
		cfg.AgentID = cfg.Agents[0].ID
	}

	return nil
}

// FindAgent returns the agent config for the given ID, or nil if not found.
func (cfg *Config) FindAgent(id string) *AgentConfig {
	for i := range cfg.Agents {
		if cfg.Agents[i].ID == id {
			return &cfg.Agents[i]
		}
	}
	return nil
}
