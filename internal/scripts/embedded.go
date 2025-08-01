package scripts

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

//go:embed setup-and-verify.sh
var setupScript string

// GetSetupScript returns the embedded setup script
func GetSetupScript() string {
	return setupScript
}

// ExecuteSetupScript executes the setup script locally (for local mode)
func ExecuteSetupScript() error {
	// Create a temporary script file
	tmpFile, err := os.CreateTemp("", "gke-setup-*.sh")
	if err != nil {
		return fmt.Errorf("failed to create temporary script file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Write the script content to the temporary file
	if _, err := tmpFile.WriteString(setupScript); err != nil {
		return fmt.Errorf("failed to write script content: %w", err)
	}

	// Make the script executable
	if err := os.Chmod(tmpFile.Name(), 0755); err != nil {
		return fmt.Errorf("failed to make script executable: %w", err)
	}

	// Close the file before executing
	tmpFile.Close()

	// Execute the script with bash
	cmd := exec.Command("bash", tmpFile.Name())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("script execution failed: %w", err)
	}

	return nil
}

// ValidateScriptContent validates that the embedded script contains required components
func ValidateScriptContent() error {
	requiredComponents := []string{
		"install_containerd",
		"configure_containerd",
		"verify_installation",
		"setup_cache_environment",
	}

	for _, component := range requiredComponents {
		if !strings.Contains(setupScript, component) {
			return fmt.Errorf("script missing required component: %s", component)
		}
	}

	return nil
}

// GetScriptInfo returns information about the embedded script
func GetScriptInfo() map[string]interface{} {
	lines := strings.Split(setupScript, "\n")
	return map[string]interface{}{
		"total_lines":    len(lines),
		"size_bytes":     len(setupScript),
		"has_shebang":    strings.HasPrefix(setupScript, "#!/bin/bash"),
		"contains_setup": strings.Contains(setupScript, "main()"),
	}
}
