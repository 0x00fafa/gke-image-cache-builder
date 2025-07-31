package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/ai-on-gke/tools/gke-image-cache-builder/pkg/builder"
	"github.com/ai-on-gke/tools/gke-image-cache-builder/pkg/config"
	"github.com/ai-on-gke/tools/gke-image-cache-builder/pkg/ui"
)

var (
	version   = "2.0.0"
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
	flag.StringVar(&cfg.ProjectName, "project-name", "", "GCP project name")
	flag.StringVar(&cfg.DiskImageName, "disk-image-name", "", "Name for the disk image") // 修改参数名

	// Container images (repeatable)
	var containerImages stringSlice
	flag.Var(&containerImages, "container-image", "Container image to cache (repeatable)")

	// Zone and location
	flag.StringVar(&cfg.Zone, "z", "", "GCP zone (required for -R mode)")
	flag.StringVar(&cfg.Zone, "zone", "", "GCP zone (required for -R mode)")
	flag.StringVar(&cfg.Network, "n", cfg.Network, "VPC network")
	flag.StringVar(&cfg.Network, "network", cfg.Network, "VPC network")
	flag.StringVar(&cfg.Subnet, "u", cfg.Subnet, "Subnet")
	flag.StringVar(&cfg.Subnet, "subnet", cfg.Subnet, "Subnet")

	// Cache configuration
	flag.IntVar(&cfg.CacheSizeGB, "s", cfg.CacheSizeGB, "Cache disk size in GB")
	flag.IntVar(&cfg.CacheSizeGB, "cache-size", cfg.CacheSizeGB, "Cache disk size in GB")
	flag.DurationVar(&cfg.Timeout, "t", cfg.Timeout, "Build timeout")
	flag.DurationVar(&cfg.Timeout, "timeout", cfg.Timeout, "Build timeout")

	// Image management
	flag.StringVar(&cfg.CacheFamilyName, "cache-family", cfg.CacheFamilyName, "Image family name")
	var cacheLabels stringMap
	flag.Var(&cacheLabels, "cache-labels", "Cache disk labels (key=value, repeatable)")

	// Authentication
	flag.StringVar(&cfg.GCPOAuth, "gcp-oauth", "", "Path to GCP service account credential file")
	flag.StringVar(&cfg.ServiceAccount, "service-account", cfg.ServiceAccount, "Service account email")
	flag.StringVar(&cfg.ImagePullAuth, "image-pull-auth", cfg.ImagePullAuth, "Image pull authentication")

	// Logging (console only, no GCS)
	verbose := flag.Bool("v", false, "Enable verbose logging")
	flag.BoolVar(verbose, "verbose", false, "Enable verbose logging")
	quiet := flag.Bool("q", false, "Suppress non-error output")
	flag.BoolVar(quiet, "quiet", false, "Suppress non-error output")

	// Advanced options
	flag.StringVar(&cfg.JobName, "job-name", cfg.JobName, "Build job name")
	machineType := flag.String("machine-type", "e2-standard-2", "VM machine type for -R mode")
	preemptible := flag.Bool("preemptible", false, "Use preemptible VM for -R mode")
	diskType := flag.String("disk-type", "pd-standard", "Cache disk type")

	// Help options
	helpFull := flag.Bool("help-full", false, "Show complete help")
	helpExamples := flag.Bool("help-examples", false, "Show usage examples")
	showVersion := flag.Bool("version", false, "Show version information")

	flag.Parse()

	// Handle help and version flags
	if *showVersion {
		ui.ShowVersionInfo(version, buildTime, gitCommit)
		return
	}

	if *helpFull {
		ui.ShowHelp("full", version)
		return
	}

	if *helpExamples {
		ui.ShowHelp("examples", version)
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
	cfg.CacheLabels = map[string]string(cacheLabels)
	cfg.Verbose = *verbose
	cfg.Quiet = *quiet
	cfg.MachineType = *machineType
	cfg.Preemptible = *preemptible
	cfg.DiskType = *diskType

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

	if err := builder.BuildImageCache(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Build failed: %v\n", err)
		os.Exit(1)
	}

	toolInfo := ui.GetToolInfo()
	fmt.Printf("✅ %s completed successfully!\n", toolInfo.ShortDesc)
	fmt.Printf("Disk image '%s' is ready for use with GKE nodes.\n", cfg.DiskImageName) // 修改字段名
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
