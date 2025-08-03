package auth

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
)

// RegistryAuth handles container registry authentication with extended support
type RegistryAuth struct {
	authType string
	gcpAuth  *GCPAuth
}

// NewRegistryAuth creates a new registry authentication handler
func NewRegistryAuth(authType string, gcpAuth *GCPAuth) *RegistryAuth {
	return &RegistryAuth{
		authType: authType,
		gcpAuth:  gcpAuth,
	}
}

// GetAuthConfig returns authentication configuration for a registry
func (r *RegistryAuth) GetAuthConfig(ctx context.Context, registry string) (*AuthConfig, error) {
	switch r.authType {
	case "None":
		return &AuthConfig{Type: "none"}, nil
	case "ServiceAccountToken":
		return r.getServiceAccountAuth(ctx, registry)
	case "DockerConfig":
		return r.getDockerConfigAuth(registry)
	case "BasicAuth":
		return r.getBasicAuth(registry)
	default:
		return nil, fmt.Errorf("unsupported auth type: %s", r.authType)
	}
}

func (r *RegistryAuth) getServiceAccountAuth(ctx context.Context, registry string) (*AuthConfig, error) {
	// Only apply service account auth for GCP registries
	if !isGCPRegistry(registry) {
		return &AuthConfig{Type: "none"}, nil
	}

	creds, err := r.gcpAuth.GetCredentials(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get GCP credentials for registry auth: %w", err)
	}

	token, err := creds.TokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	return &AuthConfig{
		Type:     "bearer",
		Token:    token.AccessToken,
		Username: "_token",
		Password: token.AccessToken,
		Registry: registry,
	}, nil
}

func (r *RegistryAuth) getDockerConfigAuth(registry string) (*AuthConfig, error) {
	// Read Docker config from standard locations
	dockerConfigPath := os.Getenv("DOCKER_CONFIG")
	if dockerConfigPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		dockerConfigPath = homeDir + "/.docker"
	}

	configFile := dockerConfigPath + "/config.json"
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return &AuthConfig{Type: "none"}, nil
	}

	// Parse Docker config file (simplified implementation)
	// In a real implementation, this would parse the JSON config file
	return &AuthConfig{
		Type:     "docker-config",
		Registry: registry,
	}, nil
}

func (r *RegistryAuth) getBasicAuth(registry string) (*AuthConfig, error) {
	// Get credentials from environment variables
	username := os.Getenv("REGISTRY_USERNAME")
	password := os.Getenv("REGISTRY_PASSWORD")

	if username == "" || password == "" {
		return &AuthConfig{Type: "none"}, nil
	}

	// Create basic auth token
	auth := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))

	return &AuthConfig{
		Type:     "basic",
		Token:    auth,
		Username: username,
		Password: password,
		Registry: registry,
	}, nil
}

func isGCPRegistry(registry string) bool {
	gcpRegistries := []string{
		"gcr.io",
		"us.gcr.io",
		"eu.gcr.io",
		"asia.gcr.io",
		"pkg.dev",
		"us-docker.pkg.dev",
		"eu-docker.pkg.dev",
		"asia-docker.pkg.dev",
	}
	for _, gcpReg := range gcpRegistries {
		if strings.Contains(registry, gcpReg) {
			return true
		}
	}
	return false
}

// AuthConfig holds registry authentication configuration
type AuthConfig struct {
	Type     string
	Token    string
	Username string
	Password string
	Registry string
}
