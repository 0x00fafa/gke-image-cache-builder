package gcp

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/option"
)

// Client wraps GCP API clients with enhanced functionality
type Client struct {
	compute     *compute.Service
	projectName string
	credentials *google.Credentials
}

// NewClient creates a new enhanced GCP client
func NewClient(projectName, credentialsPath string) (*Client, error) {
	ctx := context.Background()
	var opts []option.ClientOption
	var creds *google.Credentials
	var err error

	if credentialsPath != "" {
		// Read the credentials file
		credsData, readErr := ioutil.ReadFile(credentialsPath)
		if readErr != nil {
			return nil, fmt.Errorf("failed to read credentials file: %w", readErr)
		}

		opts = append(opts, option.WithCredentialsFile(credentialsPath))
		creds, err = google.CredentialsFromJSON(ctx, credsData, compute.ComputeScope)
	} else {
		creds, err = google.FindDefaultCredentials(ctx, compute.ComputeScope)
		if creds != nil {
			opts = append(opts, option.WithCredentials(creds))
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get credentials: %w", err)
	}

	computeService, err := compute.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create compute service: %w", err)
	}

	return &Client{
		compute:     computeService,
		projectName: projectName,
		credentials: creds,
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

// Credentials returns the credentials
func (c *Client) Credentials() *google.Credentials {
	return c.credentials
}

// WaitForOperation waits for a GCP operation to complete
func (c *Client) WaitForOperation(ctx context.Context, operation *compute.Operation, zone string) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		var op *compute.Operation
		var err error

		if zone != "" {
			// Zone operation
			op, err = c.compute.ZoneOperations.Get(c.projectName, zone, operation.Name).Context(ctx).Do()
		} else {
			// Global operation
			op, err = c.compute.GlobalOperations.Get(c.projectName, operation.Name).Context(ctx).Do()
		}

		if err != nil {
			return fmt.Errorf("failed to get operation status: %w", err)
		}

		if op.Status == "DONE" {
			if op.Error != nil {
				return fmt.Errorf("operation failed: %v", op.Error)
			}
			return nil
		}

		// Wait before checking again
		time.Sleep(2 * time.Second)
	}
}

// GetInstance retrieves information about a VM instance
func (c *Client) GetInstance(ctx context.Context, zone, instanceName string) (*compute.Instance, error) {
	instance, err := c.compute.Instances.Get(c.projectName, zone, instanceName).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get instance %s: %w", instanceName, err)
	}
	return instance, nil
}

// GetDisk retrieves information about a disk
func (c *Client) GetDisk(ctx context.Context, zone, diskName string) (*compute.Disk, error) {
	disk, err := c.compute.Disks.Get(c.projectName, zone, diskName).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get disk %s: %w", diskName, err)
	}
	return disk, nil
}

// GetImage retrieves information about an image
func (c *Client) GetImage(ctx context.Context, imageName string) (*compute.Image, error) {
	image, err := c.compute.Images.Get(c.projectName, imageName).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get image %s: %w", imageName, err)
	}
	return image, nil
}

// ListImages lists images in the project
func (c *Client) ListImages(ctx context.Context) ([]*compute.Image, error) {
	imageList, err := c.compute.Images.List(c.projectName).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list images: %w", err)
	}
	return imageList.Items, nil
}

// GetCurrentInstanceMetadata gets metadata for the current instance (local mode)
func (c *Client) GetCurrentInstanceMetadata(ctx context.Context) (*InstanceMetadata, error) {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// Get instance name
	nameReq, err := http.NewRequest("GET", "http://metadata.google.internal/computeMetadata/v1/instance/name", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create name request: %w", err)
	}
	nameReq.Header.Set("Metadata-Flavor", "Google")

	nameResp, err := client.Do(nameReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get instance name: %w", err)
	}
	defer nameResp.Body.Close()

	nameBody, err := ioutil.ReadAll(nameResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read instance name: %w", err)
	}

	// Get zone
	zoneReq, err := http.NewRequest("GET", "http://metadata.google.internal/computeMetadata/v1/instance/zone", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create zone request: %w", err)
	}
	zoneReq.Header.Set("Metadata-Flavor", "Google")

	zoneResp, err := client.Do(zoneReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get instance zone: %w", err)
	}
	defer zoneResp.Body.Close()

	zoneBody, err := ioutil.ReadAll(zoneResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read instance zone: %w", err)
	}

	// Zone format: projects/PROJECT_NUMBER/zones/ZONE_NAME
	zonePath := string(zoneBody)
	parts := strings.Split(zonePath, "/")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid zone format: %s", zonePath)
	}
	zone := parts[len(parts)-1]

	return &InstanceMetadata{
		Name: string(nameBody),
		Zone: zone,
	}, nil
}

// InstanceMetadata holds instance metadata
type InstanceMetadata struct {
	Name string
	Zone string
}
