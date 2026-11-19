package config

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/exr462/go-dark/action"
)

const ApplicationName = "godark"

type Config struct {
	BasePath    string            `json:"base_path"`
	GitUsername string            `json:"git_username"`
	GitEmail    string            `json:"git_email"`
	Projects    []Project         `json:"projects"`
	JDKs        []Profile         `json:"jdks"`
	Mavens      []Profile         `json:"mavens"`
	MaxListTag  int               `json:"max_list_tag"`
	ShortCuts   []action.Shortcut `json:"short_cuts"`
	Teams       []Team            `json:"teams"`
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

	migrated := cfg.RefreshProjects()

	// Drop any shortcut whose Action didn't survive JSON decoding as a known
	// action (action.invalidAction) - these are either corrupted entries or
	// leftovers from before Action gained stable string names, when the
	// const block's iota ordering could silently drift and remap an old
	// shortcut onto today's wrong action (e.g. a legacy "Enter" shortcut
	// reinterpreting itself as "Save", making Enter save-and-quit the
	// editor). The backfill loop right below re-adds the correct current
	// default for whatever gets dropped here.
	validShortCuts := cfg.ShortCuts[:0]
	for _, sc := range cfg.ShortCuts {
		if sc.Action.IsValid() {
			validShortCuts = append(validShortCuts, sc)
		}
	}
	droppedInvalid := len(cfg.ShortCuts) != len(validShortCuts)
	cfg.ShortCuts = validShortCuts

	if cfg.ShortCuts == nil || len(cfg.ShortCuts) == 0 {
		for _, ap := range action.DefaultShortcuts {
			cfg.ShortCuts = append(cfg.ShortCuts, ap)
		}
		err := SaveConfig(cfg)
		if err != nil {
			// can never happen
			return Config{}, false
		}
	} else {
		// we are going to add the missing shortcuts
		added := false
		for _, ap := range action.DefaultShortcuts {
			toAdd := true
			for _, key := range cfg.ShortCuts {
				if key.Action == ap.Action {
					toAdd = false
				}
			}

			if toAdd {
				cfg.ShortCuts = append(cfg.ShortCuts, ap)
				added = true
			}
		}

		// Persist immediately if anything was dropped (invalid/legacy
		// entries) or backfilled, so a corrupted config.json self-heals on
		// disk right away instead of silently re-breaking on every run until
		// something else happens to trigger a save.
		if added || droppedInvalid || migrated {
			_ = SaveConfig(cfg)
		}
	}

	isFirstRun := cfg.BasePath == ""
	return cfg, isFirstRun
}

// RefreshProjects rebuilds the effective project list: it migrates legacy
// entries, merges the cached team manifests with the local projects and
// re-detects which projects are cloned. It reports whether a legacy
// migration happened (and the config should be saved).
func (cfg *Config) RefreshProjects() bool {
	// 1. Upgrade projects saved before IDs/team manifests existed.
	migrated := migrateLegacyProjects(cfg.Projects)

	// 2. Merge the cached team manifests (synced at startup, see the team
	//    package) with the user's local projects. An empty profile starts
	//    with the built-in team, whose manifest ships inside go-dark.
	cfg.Teams = withBuiltin(cfg.Teams, cfg.Projects)
	for _, t := range cfg.Teams {
		if t.Builtin {
			if err := writeBuiltin(t); err != nil {
				log.Printf("team manifest: %v", err)
			}
		}
	}
	manifests, unavailable, manifestErrs := LoadCachedManifests(cfg.Teams)
	for _, e := range manifestErrs {
		log.Printf("team manifest: %v", e)
	}
	cfg.Projects = MergeManifests(cfg.Projects, manifests, unavailable, cfg.BasePath)

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

	if cfg.Projects == nil {
		cfg.Projects = []Project{}
	}
	return migrated
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
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
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
