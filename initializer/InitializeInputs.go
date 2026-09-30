package initializer

import (
	"os"
	"path/filepath"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/exr462/go-dark/kbd"
)

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

	return inputs
}
