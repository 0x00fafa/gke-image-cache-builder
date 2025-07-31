package scripts

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

//go:embed setup-and-verify.sh
var setupScript string

// ExecuteSetupScript writes the embedded script to a temporary file and executes it
func ExecuteSetupScript() error {
	// Create temporary file
	tmpDir := os.TempDir()
	scriptPath := filepath.Join(tmpDir, "gke-setup-and-verify.sh")

	// Write embedded script to temporary file
	if err := os.WriteFile(scriptPath, []byte(setupScript), 0755); err != nil {
		return fmt.Errorf("failed to write setup script: %w", err)
	}

	// Ensure cleanup
	defer os.Remove(scriptPath)

	// Execute the script
	cmd := exec.Command("/bin/bash", scriptPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("setup script execution failed: %w", err)
	}

	return nil
}

// GetSetupScript returns the embedded setup script content
func GetSetupScript() string {
	return setupScript
}

// WriteSetupScriptToFile writes the embedded script to a specified file path
func WriteSetupScriptToFile(filePath string) error {
	return os.WriteFile(filePath, []byte(setupScript), 0755)
}
