package auth

import (
	"context"
)

// Manager coordinates authentication across different services
type Manager struct {
	gcpAuth      *GCPAuth
	registryAuth *RegistryAuth
}

// NewManager creates a new authentication manager
func NewManager(gcpCredentialsPath, registryAuthType string) *Manager {
	gcpAuth := NewGCPAuth(gcpCredentialsPath)
	registryAuth := NewRegistryAuth(registryAuthType, gcpAuth)

	return &Manager{
		gcpAuth:      gcpAuth,
		registryAuth: registryAuth,
	}
}

// GetGCPAuth returns the GCP authentication handler
func (m *Manager) GetGCPAuth() *GCPAuth {
	return m.gcpAuth
}

// GetRegistryAuth returns the registry authentication handler
func (m *Manager) GetRegistryAuth() *RegistryAuth {
	return m.registryAuth
}

// ValidateAll validates all authentication configurations
func (m *Manager) ValidateAll(ctx context.Context) error {
	// Validate GCP credentials
	if err := m.gcpAuth.ValidateCredentials(ctx); err != nil {
		return err
	}

	// Registry auth validation is done per-registry basis
	return nil
}
