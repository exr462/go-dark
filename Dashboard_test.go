package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/model"
)

func bigDashboardUI(width, height int) *model.UI {
	ui := &model.UI{
		Config:       config.Config{ShortCuts: action.DefaultShortcuts},
		ViewState:    model.StateDashboard,
		WindowWidth:  width,
		WindowHeight: height,
	}
	for i := range 26 {
		ui.Config.Projects = append(ui.Config.Projects, config.Project{
			ID: fmt.Sprintf("belgiantrain/til-service-%02d", i), Name: fmt.Sprintf("til-service-with-a-long-name-%02d", i), Type: "maven", Fetched: i%2 == 0,
		})
	}
	for i := range 80 {
		ui.TreeNodes = append(ui.TreeNodes, model.FileNode{Name: fmt.Sprintf("file-%02d.java", i), Depth: i % 4})
	}
	return ui
}

// 26 projects used to make every row wrap and the list run past the bottom
// of the screen; the dashboard must fit any reasonable window.
func TestDashboard_FitsWindowWithManyProjects(t *testing.T) {
	for _, size := range [][2]int{{100, 30}, {140, 40}, {80, 24}} {
		ui := bigDashboardUI(size[0], size[1])
		ui.SelectedProject = 20
		ui.SelectedFile = 70
		m := &appModel{ui: ui}

		out := ansi.Strip(m.render())
		lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
		if len(lines) > size[1] {
			t.Fatalf("%dx%d: dashboard is %d lines tall", size[0], size[1], len(lines))
		}
		if !strings.Contains(out, "│> til-service") {
			t.Fatalf("%dx%d: the selected project must stay visible", size[0], size[1])
		}
		if !strings.Contains(out, "of 80") || !strings.Contains(out, "file-70.java") {
			t.Fatalf("%dx%d: expected a scroll position and the selected file:\n%s", size[0], size[1], out)
		}
	}
}
