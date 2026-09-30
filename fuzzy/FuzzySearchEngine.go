package fuzzy

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/exr462/go-dark/model"
)

//goland:noinspection GoMixedReceiverTypes
func FuzzySearchEngine(ui *model.UI) {
	query := strings.ToLower(strings.TrimSpace(ui.FuzzyQueryInput.Value()))
	ui.FuzzyResults = []model.FuzzyResult{}
	if query == "" {
		return
	}

	if len(ui.Config.Projects) == 0 || ui.SelectedProject >= len(ui.Config.Projects) {
		return
	}

	proj := ui.Config.Projects[ui.SelectedProject]
	rootPath := proj.Path

	if _, err := os.Stat(rootPath); os.IsNotExist(err) {
		return
	}

	if ui.FuzzyMode == model.FuzzyModeFiles {
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
					ui.FuzzyResults = append(ui.FuzzyResults, model.FuzzyResult{
						FileName: name,
						FullPath: fullP,
					})
					if len(ui.FuzzyResults) > 100 {
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
								ui.FuzzyResults = append(ui.FuzzyResults, model.FuzzyResult{
									FileName: name,
									FullPath: fullP,
									LineNum:  lineCount,
									Snippet:  strings.TrimSpace(txt),
								})
								if len(ui.FuzzyResults) > 100 {
									_ = file.Close()
									return
								}
							}
						}
						_ = file.Close()
					}
				}
			}
		}
		deepScan(rootPath)
	}
}
