package main

import "strings"

func (m *appModel) loadShortcutsOnInputs() {
	for inputField, shortCutAction := range shortcutsMap {
		if int(shortCutAction) >= 0 && int(shortCutAction) < len(m.ui.Config.ShortCuts) {
			for i := 0; i < len(m.ui.Config.ShortCuts); i++ {
				if m.ui.Config.ShortCuts[i].Action == shortCutAction {
					m.ui.Inputs[inputField].SetValue(m.ui.Config.ShortCuts[i].KeyBinding)
				}
			}
		}
	}
}

func (m *appModel) loadShortcutsOnConfig() {
	// Save dynamic values safely across memory pointers
	for inputField, shortCutAction := range shortcutsMap {
		// Safeguard check: ensure the InputField enum value exists within the active Inputs array index limits
		if int(inputField) >= 0 && int(inputField) < len(m.ui.Inputs) {
			newBindingValue := m.ui.Inputs[inputField].Value()

			// Only update if the user actually typed a new configuration binding string
			if trimmed := strings.TrimSpace(newBindingValue); trimmed != "" {
				for i := 0; i < len(m.ui.Config.ShortCuts); i++ {
					if m.ui.Config.ShortCuts[i].Action == shortCutAction {
						m.ui.Config.ShortCuts[i].KeyBinding = trimmed
					}
				}
			}
		}
	}
}
