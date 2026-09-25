package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

const (
	ApplicationName = "godark"
)

type Project struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	Type       string `json:"type"`
	GitURL     string `json:"git_url"`
	JDKName    string `json:"jdk_name"`
	MavenName  string `json:"maven_name"`
	Deployable bool   `json:"deployable"`
	Fetched    bool   `json:"fetched"`
}

type Profile struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type AvailableProject struct {
	Name       string `json:"name"`
	Deployable bool   `json:"deployable"`
}

const BaseGitURL = "git@bitbucket.org:belgiantrain/"

var (
	AvailableProjects = []AvailableProject{
		{"ypto-maven-parent", false},
		{"contract-first-api-parent", false},
		{"til-purchase", true},
		{"til-product", true},
		{"til-smm", true},
		{"til-fdm", true},
		{"til-b2b-traveler", true},
		{"til-stop-place", true},
		{"til-channel", true},
		{"til-b2c-voucher", true},
		{"til-fulfillment", true},
		{"til-document", true},
		{"til-dtm", true},
		{"til-notification", true},
		{"til-order", true},
		{"til-b2b-voucher", true},
		{"til-authorization", true},
		{"til-mms", true},
		{"til-report", true},
		{"til-travel-time", true},
		{"til-payment", true},
		{"til-global-aks", true},
		{"til-communication", true},
		{"til-cash-device", true},
		{"til-voucher", true},
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
		if !existingProjects[ap.Name] { // Assuming AvailableProject struct has '.Name'
			cfg.Projects = append(cfg.Projects, Project{
				Name:       ap.Name,
				Type:       "java",
				Path:       ResolvePath(cfg.BasePath, ap.Name),
				GitURL:     fmt.Sprintf("%s%s.git", BaseGitURL, ap.Name),
				Deployable: ap.Deployable, // Or whatever the boolean represents in your struct
			})
		}
	}

	// Note: Changed return to true/false depending on your business logic
	// for successfully loaded configs
	return cfg, true
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
