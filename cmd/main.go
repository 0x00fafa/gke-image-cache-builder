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

	// Configuration file support
	configFile := flag.String("config", "", "Path to YAML configuration file")
	flag.StringVar(configFile, "c", "", "Path to YAML configuration file (short form)")

	// Config generation and validation
	generateConfig := flag.String("generate-config", "", "Generate configuration template (basic|advanced|ci-cd|ml)")
	generateOutput := flag.String("output", "", "Output path for generated config (default: stdout)")
	validateConfig := flag.String("validate-config", "", "Validate YAML configuration file")

	// Define execution mode flags (mutually exclusive)
	localMode := flag.Bool("L", false, "Execute on current GCP VM (local mode)")
	flag.BoolVar(localMode, "local-mode", false, "Execute on current GCP VM (local mode)")

	remoteMode := flag.Bool("R", false, "Create temporary GCP VM for execution (remote mode)")
	flag.BoolVar(remoteMode, "remote-mode", false, "Create temporary GCP VM for execution (remote mode)")

	// Required parameters
	flag.StringVar(&cfg.ProjectName, "project-name", "", "GCP project name")
	flag.StringVar(&cfg.DiskImageName, "disk-image-name", "", "Name for the disk image")

	// Container images (repeatable)
	var containerImages stringSlice
	flag.Var(&containerImages, "container-image", "Container image to cache (repeatable)")

	// Zone and location
	flag.StringVar(&cfg.Zone, "z", "", "GCP zone (required for -R mode)")
	flag.StringVar(&cfg.Zone, "zone", "", "GCP zone (required for -R mode)")
	flag.StringVar(&cfg.Network, "n", cfg.Network, "VPC network for build VM (remote mode only)")
	flag.StringVar(&cfg.Network, "network", cfg.Network, "VPC network for build VM (remote mode only)")
	flag.StringVar(&cfg.Subnet, "u", cfg.Subnet, "Subnet for build VM (remote mode only)")
	flag.StringVar(&cfg.Subnet, "subnet", cfg.Subnet, "Subnet for build VM (remote mode only)")

	// Cache configuration
	flag.IntVar(&cfg.DiskSizeGB, "s", cfg.DiskSizeGB, "Disk size in GB")         // 改为 DiskSizeGB
	flag.IntVar(&cfg.DiskSizeGB, "disk-size", cfg.DiskSizeGB, "Disk size in GB") // 改为 DiskSizeGB
	flag.DurationVar(&cfg.Timeout, "t", cfg.Timeout, "Build timeout")
	flag.DurationVar(&cfg.Timeout, "timeout", cfg.Timeout, "Build timeout")

	// Image management
	flag.StringVar(&cfg.DiskFamilyName, "disk-family", cfg.DiskFamilyName, "Image family name") // 改为 DiskFamilyName
	var diskLabels stringMap                                                                    // 改为 diskLabels
	flag.Var(&diskLabels, "disk-labels", "Disk labels (key=value, repeatable)")                 // 改为 disk-labels

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
	helpConfig := flag.Bool("help-config", false, "Show configuration file help")
	showVersion := flag.Bool("version", false, "Show version information")

	flag.Parse()

	// Handle special commands first
	if *generateConfig != "" {
		if err := handleGenerateConfig(*generateConfig, *generateOutput); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to generate config: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *validateConfig != "" {
		if err := config.ValidateYAMLFile(*validateConfig); err != nil {
			fmt.Fprintf(os.Stderr, "Configuration validation failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✅ Configuration file '%s' is valid\n", *validateConfig)
		return
	}

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

	if *helpConfig {
		ui.ShowHelp("config", version)
		return
	}

	// Load configuration from YAML file first (if specified)
	if *configFile != "" {
		if err := cfg.LoadFromYAML(*configFile); err != nil {
			errorHandler.HandleConfigError(err)
			os.Exit(1)
		}
	}

	// Validate execution mode (command line takes precedence)
	if *localMode || *remoteMode {
		mode, err := validateExecutionMode(*localMode, *remoteMode)
		if err != nil {
			errorHandler.HandleConfigError(err)
			os.Exit(1)
		}
		cfg.Mode = mode
	}

	// Set parsed values (command line takes precedence over config file)
	if len(containerImages) > 0 {
		cfg.ContainerImages = []string(containerImages)
	}
	if len(diskLabels) > 0 { // 改为 diskLabels
		if cfg.DiskLabels == nil { // 改为 DiskLabels
			cfg.DiskLabels = make(map[string]string) // 改为 DiskLabels
		}
		for k, v := range diskLabels { // 改为 diskLabels
			cfg.DiskLabels[k] = v // Command line labels override config file labels  // 改为 DiskLabels
		}
	}

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
	fmt.Printf("Disk image '%s' is ready for use with GKE nodes.\n", cfg.DiskImageName)
}

// handleGenerateConfig handles configuration template generation
func handleGenerateConfig(templateType, outputPath string) error {
	if outputPath == "" {
		outputPath = fmt.Sprintf("gke-cache-%s.yaml", templateType)
	}

	if err := config.GenerateYAMLTemplate(outputPath, templateType); err != nil {
		return err
	}

	fmt.Printf("✅ Generated %s configuration template: %s\n", templateType, outputPath)
	fmt.Printf("📝 Edit the template and use it with: --config=%s\n", outputPath)
	return nil
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
