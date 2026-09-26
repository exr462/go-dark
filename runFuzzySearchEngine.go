package main

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/exr462/go-dark/model"
)

//goland:noinspection GoMixedReceiverTypes
func (m *appModel) runFuzzySearchEngine() {
	query := strings.ToLower(strings.TrimSpace(m.state.FuzzyQueryInput.Value()))
	m.state.FuzzyResults = []model.FuzzyResult{}
	if query == "" {
		return
	}

	if len(m.state.Config.Projects) == 0 || m.state.SelectedProject >= len(m.state.Config.Projects) {
		return
	}

	proj := m.state.Config.Projects[m.state.SelectedProject]
	rootPath := proj.Path

	if _, err := os.Stat(rootPath); os.IsNotExist(err) {
		return
	}

	if m.state.FuzzyMode == model.FuzzyModeFiles {
		var traverse func(string)
		traverse = func(p string) {
			files, err := os.ReadDir(p)
			if err != nil {
				return
			}
			for _, f := range files {
				name := f.Name()
				if f.IsDir() && (name == "target" || name == "build" || name == "node_modules" || name == "bin" || name == ".git") {
					continue
				}
				if strings.HasPrefix(name, ".") && name != ".gitignore" {
					continue
				}

				fullP := filepath.Join(p, name)
				ext := strings.ToLower(filepath.Ext(name))

				if ext == ".jar" || ext == ".war" || ext == ".zip" || ext == ".class" || ext == ".exe" || ext == ".png" || ext == ".jpg" {
					continue
				}

				if strings.Contains(strings.ToLower(name), query) {
					m.state.FuzzyResults = append(m.state.FuzzyResults, model.FuzzyResult{
						FileName: name,
						FullPath: fullP,
					})
					if len(m.state.FuzzyResults) > 100 {
						return
					}
				}
				if f.IsDir() {
					traverse(fullP)
				}
			}
		}
		traverse(rootPath)
	} else {
		var deepScan func(string)
		deepScan = func(p string) {
			files, err := os.ReadDir(p)
			if err != nil {
				return
			}
			for _, f := range files {
				name := f.Name()
				if f.IsDir() && (name == "target" || name == "build" || name == "node_modules" || name == "bin" || name == ".git") {
					continue
				}
				if strings.HasPrefix(name, ".") && name != ".gitignore" {
					continue
				}

				fullP := filepath.Join(p, name)

				if f.IsDir() {
					deepScan(fullP)
				} else {
					ext := strings.ToLower(filepath.Ext(name))
					if ext == ".xml" || ext == ".go" || ext == ".json" || ext == ".java" || ext == ".txt" || ext == ".md" || ext == ".properties" || ext == ".yml" || ext == ".yaml" || ext == ".kt" || name == "Dockerfile" {
						file, err := os.Open(fullP)
						if err != nil {
							continue
						}

						scanner := bufio.NewScanner(file)
						lineCount := 0
						for scanner.Scan() {
							lineCount++
							txt := scanner.Text()
							if strings.Contains(strings.ToLower(txt), query) {
								m.state.FuzzyResults = append(m.state.FuzzyResults, model.FuzzyResult{
									FileName: name,
									FullPath: fullP,
									LineNum:  lineCount,
									Snippet:  strings.TrimSpace(txt),
								})
								if len(m.state.FuzzyResults) > 100 {
									file.Close()
									return
								}
							}
						}
						file.Close()
					}
				}
			}
		}
		deepScan(rootPath)
	}
}
