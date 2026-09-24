package config

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

const (
	ApplicationName = "repodeck"
)

type JDKProfile struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type Project struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Type    string `json:"type"`
	GitURL  string `json:"git_url"`
	JDKName string `json:"jdk_name"`
}

type GitConfig struct {
	GitUsername string `json:"git_username"`
	GitEmail    string `json:"git_email"`
}

type Config struct {
	BasePath  string       `json:"base_path"`
	GitConfig GitConfig    `json:"git_config"`
	Projects  []Project    `json:"projects"`
	JDKs      []JDKProfile `json:"jdks"`
}

func GetConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", ApplicationName, "config.json"), nil
}

func LoadConfig() (Config, bool) {
	configPath, err := GetConfigPath()
	if err != nil {
		return Config{}, false
	}
	_ = os.MkdirAll(filepath.Dir(configPath), 0o755)

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Return an empty config to signal that we need the installer view
		return Config{Projects: []Project{}}, true
	}

	bytes, err := ioutil.ReadFile(configPath)
	if err != nil {
		return Config{}, false
	}
	var cfg Config
	_ = json.Unmarshal(bytes, &cfg)
	return cfg, false
}

func SaveConfig(cfg Config) error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}
	bytes, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(configPath, bytes, 0o644)
}

func ResolvePath(base string, projectPath string) string {
	if filepath.IsAbs(projectPath) || strings.HasPrefix(projectPath, "~") {
		return projectPath
	}
	return filepath.Join(base, projectPath)
}
