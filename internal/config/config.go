package config

import (
	"encoding/json"
	"os"
	"path/filepath"
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

const (
	MetadataDir = ".overleaf"
	ConfigFile  = "config.json"
	LegacyConfigFile = "overleaf_config.json"
)

func GetConfigPath() string {
	return filepath.Join(MetadataDir, ConfigFile)
}

func Load(path string) (*Config, error) {
	// Migration logic
	if _, err := os.Stat(LegacyConfigFile); err == nil {
		if _, err := os.Stat(MetadataDir); os.IsNotExist(err) {
			_ = os.MkdirAll(MetadataDir, 0755)
		}
		newPath := GetConfigPath()
		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			_ = os.Rename(LegacyConfigFile, newPath)
			path = newPath
		}
	}

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
	dir := filepath.Dir(path)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	data, err := json.MarshalIndent(cfg, "", "    ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
