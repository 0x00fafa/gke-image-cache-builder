package auth

import (
	"context"
	"fmt"
	"strings"
)

// RegistryAuth handles container registry authentication
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

func isGCPRegistry(registry string) bool {
	gcpRegistries := []string{
		"gcr.io",
		"us.gcr.io",
		"eu.gcr.io",
		"asia.gcr.io",
		"pkg.dev",
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
