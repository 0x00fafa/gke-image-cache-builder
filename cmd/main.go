package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/0x00fafa/gke-image-cache-builder/pkg/builder"
	"github.com/0x00fafa/gke-image-cache-builder/pkg/config"
	"github.com/0x00fafa/gke-image-cache-builder/pkg/ui"
)

var (
	version   = "1.0.0"
	buildTime = "unknown"
	gitCommit = "unknown"
)

func main() {
	// Handle no arguments case
	if len(os.Args) == 1 {
		ui.ShowNoArgsHelp()
		os.Exit(1)
	}

	cfg := config.NewConfig()
	errorHandler := ui.NewErrorHandler()

	// Define execution mode flags (mutually exclusive)
	localMode := flag.Bool("L", false, "Execute on current GCP VM (local mode)")
	flag.BoolVar(localMode, "local-mode", false, "Execute on current GCP VM (local mode)")

	remoteMode := flag.Bool("R", false, "Create temporary GCP VM for execution (remote mode)")
	flag.BoolVar(remoteMode, "remote-mode", false, "Create temporary GCP VM for execution (remote mode)")

	// Required parameters
	flag.StringVar(&cfg.ProjectName, "project-name", "", "Name of a GCP project where the script will be run")
	flag.StringVar(&cfg.ImageName, "image-name", "", "Name of the image that will be generated")
	flag.StringVar(&cfg.Zone, "zone", "", "Zone where the resources will be used")
	flag.StringVar(&cfg.GCSPath, "gcs-path", "", "GCS path prefix to dump the logs")

	// Container images (repeatable)
	var containerImages stringSlice
	flag.Var(&containerImages, "container-image", "Container image to include (can be specified multiple times)")

	// Optional parameters
	flag.StringVar(&cfg.ImageFamilyName, "image-family-name", cfg.ImageFamilyName, "Name of the image family")
	flag.StringVar(&cfg.JobName, "job-name", cfg.JobName, "Name of the workflow")
	flag.StringVar(&cfg.GCPOAuth, "gcp-oauth", "", "Path to GCP service account credential file")
	flag.IntVar(&cfg.DiskSizeGB, "disk-size-gb", cfg.DiskSizeGB, "Size of disk in GB")
	flag.StringVar(&cfg.ImagePullAuth, "image-pull-auth", cfg.ImagePullAuth, "Auth mechanism for pulling images")
	flag.DurationVar(&cfg.Timeout, "timeout", cfg.Timeout, "Timeout for each step")
	flag.StringVar(&cfg.Network, "network", cfg.Network, "VPC network")
	flag.StringVar(&cfg.Subnet, "subnet", cfg.Subnet, "Subnet")
	flag.StringVar(&cfg.ServiceAccount, "service-account", cfg.ServiceAccount, "Service account email")

	// Custom flag for multiple labels
	var imageLabels stringMap
	flag.Var(&imageLabels, "image-labels", "Image labels in key=value format (can be specified multiple times)")

	// Help options
	helpExamples := flag.Bool("help-examples", false, "Show usage examples")
	helpFull := flag.Bool("help-full", false, "Show complete help with all options")
	showVersion := flag.Bool("version", false, "Show version information")
	envInfo := flag.Bool("env-info", false, "Show environment information")

	flag.Parse()

	// Handle special flags
	if *showVersion {
		ui.ShowVersionInfo(version, buildTime, gitCommit)
		return
	}

	if *helpExamples {
		ui.ShowHelp("examples", version)
		return
	}

	if *helpFull {
		ui.ShowHelp("full", version)
		return
	}

	if *envInfo {
		envInfo, err := config.ValidateEnvironment(config.ModeUnspecified)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to detect environment: %v\n", err)
			os.Exit(1)
		}
		ui.ShowEnvironmentInfo(envInfo)
		return
	}

	// Validate execution mode
	mode, err := validateExecutionMode(*localMode, *remoteMode)
	if err != nil {
		errorHandler.HandleConfigError(err)
		os.Exit(1)
	}
	cfg.Mode = mode

	// Set parsed values
	cfg.ContainerImages = []string(containerImages)
	cfg.ImageLabels = map[string]string(imageLabels)

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		errorHandler.HandleConfigError(err)
		os.Exit(1)
	}

	// Create and run builder
	builder, err := builder.NewBuilder(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create builder: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	if err := builder.BuildDiskImage(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Build failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Disk image build completed successfully!")
}

// validateExecutionMode ensures exactly one execution mode is specified
func validateExecutionMode(local, remote bool) (config.ExecutionMode, error) {
	if local && remote {
		return config.ModeUnspecified, fmt.Errorf("cannot specify both -L (local) and -R (remote) modes")
	}
	if !local && !remote {
		return config.ModeUnspecified, fmt.Errorf("execution mode required: use -L (local) or -R (remote)")
	}
	if local {
		return config.ModeLocal, nil
	}
	return config.ModeRemote, nil
}

// stringSlice implements flag.Value for multiple string values
type stringSlice []string

func (s *stringSlice) String() string {
	return strings.Join(*s, ",")
}

func (s *stringSlice) Set(value string) error {
	*s = append(*s, value)
	return nil
}

// stringMap implements flag.Value for key=value pairs
type stringMap map[string]string

func (m *stringMap) String() string {
	var pairs []string
	for k, v := range *m {
		pairs = append(pairs, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(pairs, ",")
}

func (m *stringMap) Set(value string) error {
	if *m == nil {
		*m = make(map[string]string)
	}

	parts := strings.SplitN(value, "=", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid format, expected key=value")
	}

	(*m)[parts[0]] = parts[1]
	return nil
}
