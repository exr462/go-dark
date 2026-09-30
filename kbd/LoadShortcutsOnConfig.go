package kbd

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/exr462/go-dark/action"
)

func LoadShortcutsOnConfig(shortCuts []action.Shortcut, inputs []textinput.Model) {
	// Save dynamic values safely across memory pointers
	for inputField, shortCutAction := range ShortcutsMap {
		// Safeguard check: ensure the InputField enum value exists within the active Inputs array index limits
		if int(inputField) >= 0 && int(inputField) < len(inputs) {
			newBindingValue := inputs[inputField].Value()

			// Only update if the user actually typed a new configuration binding string
			if trimmed := strings.TrimSpace(newBindingValue); trimmed != "" {
				for i := range shortCuts {
					if shortCuts[i].Action == shortCutAction {
						shortCuts[i].KeyBinding = trimmed
					}
				}
			}
		}
	}
}
