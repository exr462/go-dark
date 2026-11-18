package kbd

import (
	"charm.land/bubbles/v2/textinput"
	"github.com/exr462/go-dark/action"
)

func LoadShortcutsOnInputs(shortCuts []action.Shortcut, inputs []textinput.Model) {
	for inputField, shortCutAction := range ShortcutsMap {
		for i := range shortCuts {
			if shortCuts[i].Action == shortCutAction {
				inputs[inputField].SetValue(shortCuts[i].KeyBinding)
			}
		}
	}
}
