package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	ApplicationName = "godark"
	BaseGitURL      = "git@bitbucket.org:belgiantrain/"
)

var (
	AvailableProjects = []AvailableProject{
		{"ypto-maven-parent", false, true, []string{}},
		{"contract-first-api-parent", false, true, []string{"ypto-maven-parent"}},
		{"til-purchase", true, true, []string{"contract-first-api-parent"}},
		{"til-product", true, true, []string{"contract-first-api-parent"}},
		{"til-smm", true, true, []string{"contract-first-api-parent"}},
		{"til-fdm", true, true, []string{"contract-first-api-parent"}},
		{"til-b2b-traveler", true, true, []string{"contract-first-api-parent"}},
		{"til-stop-place", true, true, []string{"contract-first-api-parent"}},
		{"til-channel", true, true, []string{"contract-first-api-parent"}},
		{"til-b2c-voucher", true, true, []string{"contract-first-api-parent"}},
		{"til-fulfillment", true, true, []string{"contract-first-api-parent"}},
		{"til-document", true, true, []string{"contract-first-api-parent"}},
		{"til-dtm", true, true, []string{"contract-first-api-parent"}},
		{"til-notification", true, true, []string{"contract-first-api-parent"}},
		{"til-order", true, true, []string{"contract-first-api-parent"}},
		{"til-b2b-voucher", true, true, []string{"contract-first-api-parent"}},
		{"til-authorization", true, true, []string{"contract-first-api-parent"}},
		{"til-mms", true, true, []string{"contract-first-api-parent"}},
		{"til-report", true, true, []string{"contract-first-api-parent"}},
		{"til-travel-time", true, true, []string{"contract-first-api-parent"}},
		{"til-payment", true, true, []string{"contract-first-api-parent"}},
		{"til-global-aks", true, false, []string{"contract-first-api-parent"}},
		{"til-communication", true, true, []string{"contract-first-api-parent"}},
		{"til-cash-device", true, true, []string{"contract-first-api-parent"}},
		{"til-voucher", true, true, []string{"contract-first-api-parent"}},
	}
)

type Config struct {
	BasePath    string    `json:"base_path"`
	GitUsername string    `json:"git_username"`
	GitEmail    string    `json:"git_email"`
	Projects    []Project `json:"projects"`
	JDKs        []Profile `json:"jdks"`
	Mavens      []Profile `json:"mavens"`
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

	// Note: ioutil.ReadFile is deprecated since Go 1.16; os.ReadFile is preferred
	bytes, err := os.ReadFile(configPath)
	if err != nil {
		return Config{}, false
	}
	var cfg Config
	_ = json.Unmarshal(bytes, &cfg)

	// 1. Create a map of projects already present in the loaded config
	existingProjects := make(map[string]bool)
	for _, p := range cfg.Projects {
		existingProjects[p.Name] = true // Assuming Project struct has a '.Name' field
	}

	// 2. Add defaults from AvailableProjects if they aren't already there
	for _, ap := range AvailableProjects {
		if !existingProjects[ap.Name] {
			cfg.Projects = append(cfg.Projects, Project{
				Name:       ap.Name,
				Type:       "java",
				Path:       ResolvePath(cfg.BasePath, ap.Name),
				GitURL:     fmt.Sprintf("%s%s.git", BaseGitURL, ap.Name),
				Deployable: ap.Deployable,
			})
		}
	}

	// 3. Dynamically sync the Fetched status by checking if .git exists locally
	for i := range cfg.Projects {
		if cfg.Projects[i].Path == "" && cfg.BasePath != "" {
			cfg.Projects[i].Path = ResolvePath(cfg.BasePath, cfg.Projects[i].Name)
		}
		if cfg.Projects[i].Path != "" {
			gitDir := filepath.Join(cfg.Projects[i].Path, ".git")
			if _, err := os.Stat(gitDir); err == nil {
				cfg.Projects[i].Fetched = true
			} else {
				cfg.Projects[i].Fetched = false
			}
		}
	}

	isFirstRun := cfg.BasePath == ""
	return cfg, isFirstRun
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
	return os.WriteFile(configPath, bytes, 0o644)
}

func ResolvePath(base string, projectPath string) string {
	if filepath.IsAbs(projectPath) || strings.HasPrefix(projectPath, "~") {
		return projectPath
	}
	return filepath.Join(base, projectPath)
}
