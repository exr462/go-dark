package initializer

import (
	"os"
	"path/filepath"

	"charm.land/bubbles/v2/textinput"
	"github.com/exr462/go-dark/kbd"
)

// placeholderInputWidth fits the longest placeholder and typical paths
// inside the 85-column configuration modals.
const placeholderInputWidth = 60

func MakeInputs() []textinput.Model {
	home, _ := os.UserHomeDir()
	inputs := make([]textinput.Model, kbd.Ceiling)
	for i := range inputs {
		inputs[i] = textinput.New()
	}

	inputs[kbd.GitWorkspace].Placeholder = "Global Workspace Base Path"
	inputs[kbd.GitWorkspace].SetValue(filepath.Join(home, "workspace"))
	inputs[kbd.GitUsername].Placeholder = "e.g. John Doe"
	inputs[kbd.GitEmail].Placeholder = "e.g. john@example.com"
	inputs[kbd.JdkName].Placeholder = "Profile Name (e.g. Java-17)"
	inputs[kbd.JdkPath].Placeholder = "JAVA_HOME path (e.g. /usr/lib/jvm/...)"
	inputs[kbd.MvnName].Placeholder = "Maven Profile Name (e.g. Maven-3.9)"
	inputs[kbd.MvnPath].Placeholder = "MAVEN_HOME directory path"
	inputs[kbd.EditContent].Placeholder = "Opening editor... please wait 😘"

	// bubbles v2 renders only the first rune of a placeholder when the
	// input has no width, so every input that shows a placeholder gets one.
	for i := range inputs {
		if inputs[i].Placeholder != "" {
			inputs[i].SetWidth(placeholderInputWidth)
		}
	}

	return inputs
}
