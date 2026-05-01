package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	BaseURL      string `json:"base_url"`
	ProjectID    string `json:"project_id"`
	AuthType     string `json:"auth_type,omitempty"`
	Cookie       string `json:"cookie,omitempty"`
	Email        string `json:"email,omitempty"`
	Password     string `json:"password,omitempty"`
	AuthCommand  string `json:"auth_command,omitempty"`
	UseDocker     bool   `json:"use_docker,omitempty"`
	RootFolderID  string `json:"root_folder_id,omitempty"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func Save(path string, cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "    ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
