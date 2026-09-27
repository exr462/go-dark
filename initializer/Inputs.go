package initializer

import (
	"os"
	"path/filepath"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/exr462/go-dark/model"
)

func MakeInputs() []textinput.Model {
	home, _ := os.UserHomeDir()
	inputs := make([]textinput.Model, model.Ceiling)
	for i := range inputs {
		inputs[i] = textinput.New()
	}

	inputs[model.GitWorkspace].Placeholder = "Global Workspace Base Path"
	inputs[model.GitWorkspace].SetValue(filepath.Join(home, "workspace"))
	inputs[model.GitUsername].Placeholder = "e.g. John Doe"
	inputs[model.GitEmail].Placeholder = "e.g. john@example.com"
	inputs[model.JdkName].Placeholder = "Profile Name (e.g. Java-17)"
	inputs[model.JdkPath].Placeholder = "JAVA_HOME path (e.g. /usr/lib/jvm/...)"
	inputs[model.MvnName].Placeholder = "Maven Profile Name (e.g. Maven-3.9)"
	inputs[model.MvnPath].Placeholder = "MAVEN_HOME directory path"
	inputs[model.EditContent].Placeholder = "Opening editor... please wait 😘"

	return inputs
}
