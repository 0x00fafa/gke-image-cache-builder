package auth

import (
	"context"
	"fmt"
	"os"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
)

// GCPAuth handles Google Cloud Platform authentication
type GCPAuth struct {
	credentialsPath string
}

// NewGCPAuth creates a new GCP authentication handler
func NewGCPAuth(credentialsPath string) *GCPAuth {
	return &GCPAuth{
		credentialsPath: credentialsPath,
	}
}

// GetCredentials returns GCP credentials for API access
func (g *GCPAuth) GetCredentials(ctx context.Context) (*google.Credentials, error) {
	var creds *google.Credentials
	var err error

	if g.credentialsPath != "" {
		// Use service account file
		creds, err = google.CredentialsFromJSON(ctx, g.readCredentialsFile(),
			"https://www.googleapis.com/auth/cloud-platform")
	} else {
		// Use default credentials (metadata server, gcloud, etc.)
		creds, err = google.FindDefaultCredentials(ctx,
			"https://www.googleapis.com/auth/cloud-platform")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get GCP credentials: %w", err)
	}

	return creds, nil
}

// GetClientOption returns a client option for GCP services
func (g *GCPAuth) GetClientOption(ctx context.Context) (option.ClientOption, error) {
	if g.credentialsPath != "" {
		return option.WithCredentialsFile(g.credentialsPath), nil
	}

	// Use default credentials
	creds, err := g.GetCredentials(ctx)
	if err != nil {
		return nil, err
	}

	return option.WithCredentials(creds), nil
}

func (g *GCPAuth) readCredentialsFile() []byte {
	data, err := os.ReadFile(g.credentialsPath)
	if err != nil {
		return nil
	}
	return data
}

// ValidateCredentials checks if the credentials are valid
func (g *GCPAuth) ValidateCredentials(ctx context.Context) error {
	_, err := g.GetCredentials(ctx)
	return err
}
