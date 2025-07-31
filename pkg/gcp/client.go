package gcp

import (
	"context"
	"fmt"

	"google.golang.org/api/compute/v1"
	"google.golang.org/api/option"
)

// Client wraps GCP API clients (compute only, no storage)
type Client struct {
	compute     *compute.Service
	projectName string
}

// NewClient creates a new GCP client
func NewClient(projectName, credentialsPath string) (*Client, error) {
	ctx := context.Background()

	var opts []option.ClientOption
	if credentialsPath != "" {
		opts = append(opts, option.WithCredentialsFile(credentialsPath))
	}

	computeService, err := compute.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create compute service: %w", err)
	}

	return &Client{
		compute:     computeService,
		projectName: projectName,
	}, nil
}

// Compute returns the compute service
func (c *Client) Compute() *compute.Service {
	return c.compute
}

// ProjectName returns the project name
func (c *Client) ProjectName() string {
	return c.projectName
}
