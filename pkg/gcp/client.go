package gcp

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"google.golang.org/api/compute/v1"
	"google.golang.org/api/option"
)

// Client wraps the GCP compute service
type Client struct {
	service     *compute.Service
	projectID   string
	credentials string
}

// NewClient creates a new GCP client
func NewClient(projectID, credentials string) (*Client, error) {
	ctx := context.Background()

	var opts []option.ClientOption
	if credentials != "" {
		opts = append(opts, option.WithCredentialsFile(credentials))
	}

	service, err := compute.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create compute service: %w", err)
	}

	return &Client{
		service:     service,
		projectID:   projectID,
		credentials: credentials,
	}, nil
}

// CreateInstanceWithStartupScript creates a new GCP VM instance with startup script
func (c *Client) CreateInstanceWithStartupScript(ctx context.Context, zone, name, machineType, startupScript string) error {
	instance := &compute.Instance{
		Name:        name,
		MachineType: fmt.Sprintf("zones/%s/machineTypes/%s", zone, machineType),
		Disks: []*compute.AttachedDisk{
			{
				Boot:       true,
				AutoDelete: true,
				InitializeParams: &compute.AttachedDiskInitializeParams{
					SourceImage: "projects/ubuntu-os-cloud/global/images/family/ubuntu-2004-lts",
					DiskSizeGb:  20,
					DiskType:    fmt.Sprintf("zones/%s/diskTypes/pd-standard", zone),
				},
			},
		},
		NetworkInterfaces: []*compute.NetworkInterface{
			{
				AccessConfigs: []*compute.AccessConfig{
					{
						Type: "ONE_TO_ONE_NAT",
						Name: "External NAT",
					},
				},
			},
		},
		ServiceAccounts: []*compute.ServiceAccount{
			{
				Email: "default",
				Scopes: []string{
					"https://www.googleapis.com/auth/cloud-platform",
				},
			},
		},
		Metadata: &compute.Metadata{
			Items: []*compute.MetadataItems{
				{
					Key:   "startup-script",
					Value: &startupScript,
				},
			},
		},
	}

	op, err := c.service.Instances.Insert(c.projectID, zone, instance).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to create instance: %w", err)
	}

	return c.WaitForOperation(ctx, zone, op.Name)
}

// DeleteInstance deletes a GCP VM instance
func (c *Client) DeleteInstance(ctx context.Context, zone, name string) error {
	op, err := c.service.Instances.Delete(c.projectID, zone, name).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to delete instance: %w", err)
	}

	return c.WaitForOperation(ctx, zone, op.Name)
}

// GetInstance gets information about a GCP VM instance
func (c *Client) GetInstance(ctx context.Context, zone, name string) (*compute.Instance, error) {
	instance, err := c.service.Instances.Get(c.projectID, zone, name).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get instance: %w", err)
	}

	return instance, nil
}

// WaitForOperation waits for a GCP operation to complete
func (c *Client) WaitForOperation(ctx context.Context, zone, operationName string) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		op, err := c.service.ZoneOperations.Get(c.projectID, zone, operationName).Context(ctx).Do()
		if err != nil {
			return fmt.Errorf("failed to get operation status: %w", err)
		}

		if op.Status == "DONE" {
			if op.Error != nil {
				return fmt.Errorf("operation failed: %v", op.Error)
			}
			return nil
		}

		time.Sleep(5 * time.Second)
	}
}

// IsRunningOnGCP checks if the current environment is a GCP VM
func IsRunningOnGCP() bool {
	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	req, err := http.NewRequest("GET", "http://metadata.google.internal/computeMetadata/v1/instance/zone", nil)
	if err != nil {
		return false
	}

	req.Header.Set("Metadata-Flavor", "Google")

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == 200 && resp.Header.Get("Metadata-Flavor") == "Google"
}

// GetCurrentZone gets the zone of the current GCP VM
func GetCurrentZone() (string, error) {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	req, err := http.NewRequest("GET", "http://metadata.google.internal/computeMetadata/v1/instance/zone", nil)
	if err != nil {
		return "", fmt.Errorf("failed to create metadata request: %w", err)
	}

	req.Header.Set("Metadata-Flavor", "Google")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to access GCP metadata server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("metadata server returned status %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read metadata response: %w", err)
	}

	// Response format: projects/PROJECT_NUMBER/zones/ZONE_NAME
	zonePath := string(body)
	parts := strings.Split(zonePath, "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("unexpected zone format: %s", zonePath)
	}

	return parts[len(parts)-1], nil
}
