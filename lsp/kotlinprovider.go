package lsp

import (
	"os"
	"os/exec"
	"path/filepath"
)

// KotlinProvider implements components.LanguageProvider for Kotlin source workspaces
type KotlinProvider struct{}

// Name returns the formal string identity of the engine
func (k KotlinProvider) Name() string {
	return "Kotlin"
}

// Extensions covers both typical standard compilation source structures and scripts
func (k KotlinProvider) Extensions() []string {
	return []string{".kt", ".kts"}
}

// GetLSPConfig wires up the standard open-source kotlin-language-server execution parameters
func (k KotlinProvider) GetLSPConfig() LSPConfig {
	return LSPConfig{
		ServerBinary: "kotlin-language-server",
		Args:         []string{}, // Add any custom telemetry flags your tool requires here
	}
}

// GetBuildCommand optimizes for Gradle build tools, falling back to basic CLI single file compilers
func (k KotlinProvider) GetBuildCommand(filePath string) *exec.Cmd {
	dir := filepath.Dir(filePath)

	// Check if the workspace is backed by a Gradle Wrapper script
	if _, err := os.Stat(filepath.Join(dir, "gradlew")); err == nil {
		return exec.Command("./gradlew", "build")
	}

	// Standalone fallback: compile the file to a standard runnable jar output
	outputJar := filepath.Join(dir, "app.jar")
	return exec.Command("kotlinc", filePath, "-include-runtime", "-d", outputJar)
}

// GetRunCommand runs the project using Gradle or boots up the compiled jar file fallback
func (k KotlinProvider) GetRunCommand(filePath string) *exec.Cmd {
	dir := filepath.Dir(filePath)

	// Check if running via Gradle task wrapper environment
	if _, err := os.Stat(filepath.Join(dir, "gradlew")); err == nil {
		return exec.Command("./gradlew", "run")
	}

	// Standalone fallback: execute the generated jar file via the Java runtime environment
	outputJar := filepath.Join(dir, "app.jar")
	return exec.Command("java", "-jar", outputJar)
}
