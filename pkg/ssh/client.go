package ssh

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"

	"golang.org/x/crypto/ssh"

	"github.com/0x00fafa/gke-image-cache-builder/pkg/log"
)

// generateSSHKey generates a new SSH key pair
func generateSSHKey(privateKeyPath string) error {
	// Generate private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate private key: %w", err)
	}

	// Marshal private key to PEM format
	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}

	// Write private key to file
	privateKeyFile, err := os.Create(privateKeyPath)
	if err != nil {
		return fmt.Errorf("failed to create private key file: %w", err)
	}
	defer privateKeyFile.Close()

	if err := pem.Encode(privateKeyFile, privateKeyPEM); err != nil {
		return fmt.Errorf("failed to encode private key: %w", err)
	}

	// Set proper permissions for private key
	if err := os.Chmod(privateKeyPath, 0600); err != nil {
		return fmt.Errorf("failed to set private key permissions: %w", err)
	}

	// Generate public key
	publicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return fmt.Errorf("failed to generate public key: %w", err)
	}

	// Marshal public key to authorized_keys format
	publicKeyBytes := ssh.MarshalAuthorizedKey(publicKey)

	// Write public key to file
	publicKeyPath := privateKeyPath + ".pub"
	publicKeyFile, err := os.Create(publicKeyPath)
	if err != nil {
		return fmt.Errorf("failed to create public key file: %w", err)
	}
	defer publicKeyFile.Close()

	if _, err := publicKeyFile.Write(publicKeyBytes); err != nil {
		return fmt.Errorf("failed to write public key: %w", err)
	}

	return nil
}

// Client handles SSH connections to remote instances
type Client struct {
	logger *log.Logger
	config *ssh.ClientConfig
}

// NewClient creates a new SSH client
func NewClient(logger *log.Logger) (*Client, error) {
	// Find SSH key path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	sshDir := filepath.Join(homeDir, ".ssh")

	// Check for private key files in order of preference
	keyPaths := []string{
		filepath.Join(sshDir, "id_rsa"),
		filepath.Join(sshDir, "id_ecdsa"),
		filepath.Join(sshDir, "id_ed25519"),
	}

	var keyPath string
	for _, path := range keyPaths {
		if _, err := os.Stat(path); err == nil {
			keyPath = path
			break
		}
	}

	// If no key found, generate a new one
	if keyPath == "" {
		logger.Warn("No SSH private key found, generating a new one...")

		// Ensure .ssh directory exists
		if err := os.MkdirAll(sshDir, 0700); err != nil {
			return nil, fmt.Errorf("failed to create .ssh directory: %w", err)
		}

		// Generate new SSH key pair
		keyPath = filepath.Join(sshDir, "id_rsa")
		if err := generateSSHKey(keyPath); err != nil {
			return nil, fmt.Errorf("failed to generate SSH key: %w", err)
		}

		logger.Info("Generated new SSH key pair")
	}

	// Read private key
	key, err := ioutil.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read SSH private key: %w", err)
	}

	// Parse private key
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SSH private key: %w", err)
	}

	// Create SSH client config
	config := &ssh.ClientConfig{
		User: "abc",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // Insecure but acceptable for this use case
		Timeout:         30 * time.Second,
	}

	return &Client{
		logger: logger,
		config: config,
	}, nil
}

// ExecuteCommand executes a command on a remote host
func (c *Client) ExecuteCommand(ctx context.Context, host, command string) (string, error) {
	c.logger.Infof("Executing SSH command on %s: %s", host, command)

	// Connect to the remote host
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:22", host), c.config)
	if err != nil {
		return "", fmt.Errorf("failed to connect to %s: %w", host, err)
	}
	defer client.Close()

	// Create a session
	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	// Capture output
	var output strings.Builder
	session.Stdout = &output
	session.Stderr = &output

	// Execute the command
	if err := session.Run(command); err != nil {
		return output.String(), fmt.Errorf("command failed: %w, output: %s", err, output.String())
	}

	c.logger.Success("SSH command executed successfully")
	return output.String(), nil
}

// ExecuteCommandWithProgress executes a command on a remote host with progress monitoring
func (c *Client) ExecuteCommandWithProgress(ctx context.Context, host, command string, progressCallback func(string)) (string, error) {
	c.logger.Infof("Executing SSH command on %s with progress monitoring", host)

	// Connect to the remote host
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:22", host), c.config)
	if err != nil {
		return "", fmt.Errorf("failed to connect to %s: %w", host, err)
	}
	defer client.Close()

	// Create a session
	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	// Create pipes for stdout and stderr
	stdout, err := session.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := session.Start(command); err != nil {
		return "", fmt.Errorf("failed to start command: %w", err)
	}

	// Read output in real-time
	output := &strings.Builder{}
	done := make(chan error, 1)

	go func() {
		// Read from both stdout and stderr
		multi := io.MultiReader(stdout, stderr)
		buf := make([]byte, 1024)

		for {
			n, err := multi.Read(buf)
			if n > 0 {
				chunk := string(buf[:n])
				output.WriteString(chunk)
				if progressCallback != nil {
					progressCallback(chunk)
				}
			}
			if err != nil {
				if err != io.EOF {
					done <- err
				}
				break
			}
		}
		done <- session.Wait()
	}()

	// Wait for completion or context cancellation
	select {
	case err := <-done:
		if err != nil {
			return output.String(), fmt.Errorf("command failed: %w, output: %s", err, output.String())
		}
	case <-ctx.Done():
		// Try to terminate the session
		session.Signal(ssh.SIGINT)
		time.Sleep(2 * time.Second)
		session.Signal(ssh.SIGKILL)
		return output.String(), fmt.Errorf("command cancelled: %w", ctx.Err())
	}

	c.logger.Success("SSH command executed successfully")
	return output.String(), nil
}

// WaitForSSHReady waits for SSH to be ready on a host
func (c *Client) WaitForSSHReady(ctx context.Context, host string) error {
	c.logger.Infof("Waiting for SSH to be ready on %s...", host)

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	for {
		select {
		case <-timeoutCtx.Done():
			return fmt.Errorf("timeout waiting for SSH to be ready on %s", host)
		case <-ticker.C:
			// Try to connect
			client, err := ssh.Dial("tcp", fmt.Sprintf("%s:22", host), c.config)
			if err != nil {
				c.logger.Debugf("SSH not ready yet: %v", err)
				continue
			}
			client.Close()
			c.logger.Success("SSH is ready")
			return nil
		}
	}
}
