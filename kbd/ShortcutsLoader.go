package kbd

import (
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/exr462/go-dark/action"
)

func LoadShortcutsOnInputs(shortCuts []action.Shortcut, inputs []textinput.Model) {
	for inputField, shortCutAction := range ShortcutsMap {
		if int(shortCutAction) >= 0 && int(shortCutAction) < len(shortCuts) {
			for i := range shortCuts {
				if shortCuts[i].Action == shortCutAction {
					inputs[inputField].SetValue(shortCuts[i].KeyBinding)
				}
			}
		}
	}
}
