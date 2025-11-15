package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime/debug"

	"github.com/mosiko1234/heimdal/sensor/internal/config"
	"github.com/mosiko1234/heimdal/sensor/internal/logger"
	"github.com/mosiko1234/heimdal/sensor/internal/orchestrator"
)

const (
	defaultConfigPath = "/etc/heimdal/config.json"
	version           = "2.0.0"
)

var (
	configPath = flag.String("config", defaultConfigPath, "Path to configuration file")
	showVersion = flag.Bool("version", false, "Show version information")
	showHelp    = flag.Bool("help", false, "Show help information")
)

func main() {
	// Parse command-line flags
	flag.Parse()

	// Show version if requested
	if *showVersion {
		fmt.Printf("Heimdal Sensor v%s\n", version)
		os.Exit(0)
	}

	// Show help if requested
	if *showHelp {
		printHelp()
		os.Exit(0)
	}

	// Set up panic recovery
	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC: %v", r)
			log.Printf("Stack trace:\n%s", debug.Stack())
			os.Exit(1)
		}
	}()

	// Load configuration
	log.Printf("Loading configuration from: %s", *configPath)
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize structured logging
	if err := logger.Initialize(cfg.Logging.File, cfg.Logging.Level); err != nil {
		log.Fatalf("Failed to initialize logging: %v", err)
	}

	logger.Info("=== Heimdal Sensor v%s ===", version)
	logger.Info("Configuration loaded successfully")
	logger.Info("Log level: %s", cfg.Logging.Level)
	logger.Info("Log file: %s", cfg.Logging.File)

	// Create orchestrator
	logger.Info("Creating orchestrator...")
	orch, err := orchestrator.NewOrchestrator(cfg)
	if err != nil {
		logger.Error("Failed to create orchestrator: %v", err)
		os.Exit(1)
	}

	// Run orchestrator (blocks until shutdown signal)
	logger.Info("Starting orchestrator...")
	if err := orch.Run(); err != nil {
		logger.Error("Orchestrator error: %v", err)
		os.Exit(1)
	}

	logger.Info("Heimdal Sensor exited cleanly")
}



// printHelp displays usage information
func printHelp() {
	fmt.Printf("Heimdal Sensor v%s\n\n", version)
	fmt.Println("Usage:")
	fmt.Printf("  %s [options]\n\n", os.Args[0])
	fmt.Println("Options:")
	flag.PrintDefaults()
	fmt.Println("\nDescription:")
	fmt.Println("  Heimdal is a network security sensor for zero-touch deployment on")
	fmt.Println("  Raspberry Pi hardware. It performs automated network discovery,")
	fmt.Println("  traffic interception, and behavioral profiling of network devices.")
	fmt.Println("\nExamples:")
	fmt.Printf("  %s\n", os.Args[0])
	fmt.Printf("  %s --config /path/to/config.json\n", os.Args[0])
	fmt.Printf("  %s --version\n", os.Args[0])
}
