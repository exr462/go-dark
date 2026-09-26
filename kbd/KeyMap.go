package kbd

import (
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type KeyMap struct {
	Up    key.Binding
	Down  key.Binding
	Left  key.Binding
	Right key.Binding
	Help  key.Binding
	Quit  key.Binding
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit}
}

// FullHelp returns keybindings for the expanded help view. It's part of the
// key.Map interface.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right}, // first column
		{k.Help, k.Quit},                // second column
	}
}

var Keys = KeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "move up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "move down"),
	),
	Left: key.NewBinding(
		key.WithKeys("left", "h"),
		key.WithHelp("←/h", "move left"),
	),
	Right: key.NewBinding(
		key.WithKeys("right", "l"),
		key.WithHelp("→/l", "move right"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "toggle help"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "esc", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}

type HelpModel struct {
	Keys       KeyMap
	Help       help.Model
	InputStyle lipgloss.Style
	LastKey    string
	Quitting   bool
}

func (m HelpModel) Init() tea.Cmd {
	return nil
}

func (m HelpModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// If we set a width on the help menu it can gracefully truncate
		// its view as needed.
		m.Help.SetWidth(msg.Width)

	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.Keys.Up):
			m.LastKey = "↑"
		case key.Matches(msg, m.Keys.Down):
			m.LastKey = "↓"
		case key.Matches(msg, m.Keys.Left):
			m.LastKey = "←"
		case key.Matches(msg, m.Keys.Right):
			m.LastKey = "→"
		case key.Matches(msg, m.Keys.Help):
			m.Help.ShowAll = !m.Help.ShowAll
		case key.Matches(msg, m.Keys.Quit):
			m.Quitting = true
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m HelpModel) View() tea.View {
	if m.Quitting {
		return tea.NewView("Bye!\n")
	}

	var status string
	if m.LastKey == "" {
		status = "Waiting for input..."
	} else {
		status = "You chose: " + m.InputStyle.Render(m.LastKey)
	}

	helpView := m.Help.View(m.Keys)
	height := 8 - strings.Count(status, "\n") - strings.Count(helpView, "\n")

	return tea.NewView(status + strings.Repeat("\n", height) + helpView)
}
