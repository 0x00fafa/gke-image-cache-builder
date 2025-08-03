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
	return ExecuteSetupScriptWithArgs("setup")
}

// ExecuteSetupScriptWithArgs executes the setup script with specific arguments
func ExecuteSetupScriptWithArgs(args ...string) error {
	// Create temporary file
	tmpDir := os.TempDir()
	scriptPath := filepath.Join(tmpDir, "gke-setup-and-verify.sh")

	// Write embedded script to temporary file
	if err := os.WriteFile(scriptPath, []byte(setupScript), 0755); err != nil {
		return fmt.Errorf("failed to write setup script: %w", err)
	}

	// Ensure cleanup
	defer os.Remove(scriptPath)

	// Prepare command with arguments
	cmdArgs := append([]string{scriptPath}, args...)
	cmd := exec.Command("/bin/bash", cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("setup script execution failed: %w", err)
	}

	return nil
}

// ExecuteImageProcessing executes the image processing workflow
func ExecuteImageProcessing(deviceName, authMechanism string, storeChecksums bool, images []string) error {
	args := []string{
		"full-workflow",
		deviceName,
		authMechanism,
		fmt.Sprintf("%t", storeChecksums),
	}
	args = append(args, images...)

	return ExecuteSetupScriptWithArgs(args...)
}

// ExecuteDiskPreparation executes disk preparation
func ExecuteDiskPreparation(deviceName, mountPoint string) error {
	return ExecuteSetupScriptWithArgs("prepare-disk", deviceName, mountPoint)
}

// ExecuteImageVerification executes image verification
func ExecuteImageVerification(mountPoint string) error {
	return ExecuteSetupScriptWithArgs("verify-image", mountPoint)
}

// GetSetupScript returns the embedded setup script content
func GetSetupScript() string {
	return setupScript
}

// WriteSetupScriptToFile writes the embedded script to a specified file path
func WriteSetupScriptToFile(filePath string) error {
	return os.WriteFile(filePath, []byte(setupScript), 0755)
}
