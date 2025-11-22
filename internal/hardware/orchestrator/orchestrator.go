// Package orchestrator provides the main coordination and lifecycle management
// for the Heimdal Hardware sensor components. It handles component initialization,
// startup sequencing, health monitoring, automatic recovery, and graceful shutdown.
//
// This orchestrator is specifically for the hardware product (Raspberry Pi sensor)
// and uses platform-specific implementations through interfaces for maximum flexibility.
package orchestrator

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/mosiko1234/heimdal/sensor/internal/analyzer"
	"github.com/mosiko1234/heimdal/sensor/internal/api"
	"github.com/mosiko1234/heimdal/sensor/internal/cloud"
	"github.com/mosiko1234/heimdal/sensor/internal/cloud/aws"
	"github.com/mosiko1234/heimdal/sensor/internal/cloud/gcp"
	"github.com/mosiko1234/heimdal/sensor/internal/config"
	"github.com/mosiko1234/heimdal/sensor/internal/core/packet"
	"github.com/mosiko1234/heimdal/sensor/internal/database"
	"github.com/mosiko1234/heimdal/sensor/internal/discovery"
	"github.com/mosiko1234/heimdal/sensor/internal/errors"
	"github.com/mosiko1234/heimdal/sensor/internal/interceptor"
	"github.com/mosiko1234/heimdal/sensor/internal/logger"
	"github.com/mosiko1234/heimdal/sensor/internal/netconfig"
	"github.com/mosiko1234/heimdal/sensor/internal/platform"
	"github.com/mosiko1234/heimdal/sensor/internal/profiler"
)

// Component interface defines the lifecycle methods for all components
type Component interface {
	Start() error
	Stop() error
	Name() string
}

// HardwareOrchestrator manages the lifecycle of all Heimdal hardware sensor components
// using platform abstraction interfaces for packet capture and system integration.
type HardwareOrchestrator struct {
	config     *config.Config
	db         *database.DatabaseManager
	components []Component
	logger     *logger.Logger

	// Platform interfaces
	packetCapture   platform.PacketCaptureProvider
	systemIntegrator platform.SystemIntegrator

	// Component instances
	netConfig    *netconfig.AutoConfig
	scanner      *discovery.Scanner
	arpSpoofer   *interceptor.ARPSpoofer
	analyzer     *packet.Analyzer
	profilerComp *profiler.Profiler
	apiServer    *api.APIServer
	cloudOrch    *cloud.Orchestrator

	// Communication channels
	deviceChan       chan *database.Device
	packetChan       chan packet.PacketInfo       // For analyzer output
	profilerChan     chan analyzer.PacketInfo     // For profiler input (old format)

	// Lifecycle management
	shutdownCh chan struct{}
	wg         sync.WaitGroup
	mu         sync.Mutex

	// Component health tracking
	componentHealth map[string]*componentHealthInfo
	healthMu        sync.RWMutex
}

// componentHealthInfo tracks health and restart information for a component
type componentHealthInfo struct {
	name          string
	restartCount  int
	lastRestart   time.Time
	restartWindow time.Time // Start of current 1-hour window
	isRunning     bool
}

// NewHardwareOrchestrator creates a new hardware orchestrator instance with platform interfaces
func NewHardwareOrchestrator(
	cfg *config.Config,
	packetCapture platform.PacketCaptureProvider,
	systemIntegrator platform.SystemIntegrator,
) (*HardwareOrchestrator, error) {
	if cfg == nil {
		return nil, fmt.Errorf("configuration is required")
	}
	if packetCapture == nil {
		return nil, fmt.Errorf("packet capture provider is required")
	}

	return &HardwareOrchestrator{
		config:           cfg,
		packetCapture:    packetCapture,
		systemIntegrator: systemIntegrator,
		logger:           logger.NewComponentLogger("HardwareOrchestrator"),
		components:       make([]Component, 0),
		deviceChan:       make(chan *database.Device, 100),
		packetChan:       make(chan packet.PacketInfo, 1000),
		profilerChan:     make(chan analyzer.PacketInfo, 1000),
		shutdownCh:       make(chan struct{}),
		componentHealth:  make(map[string]*componentHealthInfo),
	}, nil
}

// Run starts all components and blocks until shutdown signal is received
func (o *HardwareOrchestrator) Run() error {
	o.logger.Info("=== Heimdal Hardware Sensor Starting ===")

	// Initialize all components
	if err := o.initializeComponents(); err != nil {
		return errors.Wrap(err, "failed to initialize components")
	}

	// Start all components in correct order
	if err := o.startComponents(); err != nil {
		return errors.Wrap(err, "failed to start components")
	}

	o.logger.Info("=== Heimdal Hardware Sensor Running ===")

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	// Wait for shutdown signal
	sig := <-sigChan
	o.logger.Info("Received signal: %v", sig)

	// Perform graceful shutdown
	return o.shutdown()
}

// initializeComponents creates all component instances with proper dependencies
func (o *HardwareOrchestrator) initializeComponents() error {
	o.logger.Info("Initializing components...")

	// 1. Initialize Database
	o.logger.Info("Initializing database at %s", o.config.Database.Path)
	var db *database.DatabaseManager
	err := errors.RetryWithBackoff("database initialization", errors.DefaultRetryConfig(), func() error {
		var err error
		db, err = database.NewDatabaseManager(o.config.Database.Path)
		return err
	})
	if err != nil {
		return errors.Wrap(err, "failed to initialize database")
	}
	o.db = db

	// 2. Initialize Network Auto-Config (blocking until network detected)
	o.logger.Info("Initializing network auto-configuration...")
	o.netConfig = netconfig.NewAutoConfig()
	if err := o.netConfig.DetectNetwork(); err != nil {
		return errors.Wrap(err, "failed to detect network")
	}

	netCfg := o.netConfig.GetConfig()
	if netCfg == nil {
		return fmt.Errorf("network configuration is nil after detection")
	}
	o.logger.Info("Network detected: interface=%s, ip=%s, gateway=%s, cidr=%s",
		netCfg.Interface, netCfg.LocalIP, netCfg.Gateway, netCfg.CIDR)

	// 3. Initialize Device Discovery Scanner
	o.logger.Info("Initializing device discovery scanner...")
	scanInterval := time.Duration(o.config.Discovery.ARPScanInterval) * time.Second
	inactiveTimeout := time.Duration(o.config.Discovery.InactiveTimeout) * time.Minute
	o.scanner = discovery.NewScanner(
		o.netConfig,
		o.db,
		o.deviceChan,
		scanInterval,
		o.config.Discovery.MDNSEnabled,
		inactiveTimeout,
	)
	o.components = append(o.components, o.scanner)
	o.initComponentHealth(o.scanner.Name())

	// 4. Initialize Traffic Interceptor (ARP Spoofer)
	if o.config.Interceptor.Enabled {
		o.logger.Info("Initializing traffic interceptor (ARP spoofer)...")
		spoofInterval := time.Duration(o.config.Interceptor.SpoofInterval) * time.Second
		o.arpSpoofer = interceptor.NewARPSpoofer(
			o.netConfig,
			o.deviceChan,
			spoofInterval,
			o.config.Interceptor.TargetMACs,
		)
		o.components = append(o.components, o.arpSpoofer)
		o.initComponentHealth(o.arpSpoofer.Name())
	} else {
		o.logger.Info("Traffic interceptor is disabled in configuration")
	}

	// 5. Initialize Packet Analyzer using platform interface
	o.logger.Info("Initializing packet analyzer with platform interface...")
	analyzer, err := packet.NewAnalyzer(o.packetCapture, o.packetChan, nil)
	if err != nil {
		return errors.Wrap(err, "failed to initialize packet analyzer")
	}
	o.analyzer = analyzer
	
	// Create an adapter component that wraps the analyzer
	analyzerComponent := &analyzerComponent{
		analyzer:   analyzer,
		netConfig:  netCfg,
		bpfFilter:  "not broadcast and not multicast",
	}
	o.components = append(o.components, analyzerComponent)
	o.initComponentHealth(analyzerComponent.Name())

	// 6. Initialize Behavioral Profiler
	// Note: For now, we're using the old profiler from internal/profiler
	// which uses the database manager directly. In the future, this should
	// be migrated to use the core profiler with a StorageProvider implementation.
	o.logger.Info("Initializing behavioral profiler...")
	persistInterval := time.Duration(o.config.Profiler.PersistInterval) * time.Second
	profilerComp, err := profiler.NewProfiler(o.db, o.profilerChan, persistInterval)
	if err != nil {
		return errors.Wrap(err, "failed to initialize profiler")
	}
	o.profilerComp = profilerComp
	o.components = append(o.components, o.profilerComp)
	o.initComponentHealth(o.profilerComp.Name())
	
	// Start adapter goroutine to convert packet.PacketInfo to analyzer.PacketInfo
	o.wg.Add(1)
	go o.packetInfoAdapter()

	// 7. Initialize Web API Server
	o.logger.Info("Initializing web API server...")
	o.apiServer = api.NewAPIServer(
		o.db,
		o.config.API.Host,
		o.config.API.Port,
		o.config.API.RateLimitPerMinute,
	)
	o.initComponentHealth(o.apiServer.Name())

	// 8. Initialize Cloud Connector (if enabled)
	if o.config.Cloud.Enabled {
		o.logger.Info("Initializing cloud connector...")
		cloudOrch, err := cloud.NewOrchestrator(&o.config.Cloud, o.db)
		if err != nil {
			o.logger.Warn("Failed to initialize cloud orchestrator: %v", err)
			o.logger.Info("Local operations will continue without cloud connectivity")
		} else {
			// Create the appropriate connector based on provider
			var connector cloud.CloudConnector
			var err error
			switch o.config.Cloud.Provider {
			case "aws":
				connector, err = aws.NewAWSIoTConnector(&o.config.Cloud.AWS, o.db)
				if err != nil {
					o.logger.Warn("Failed to create AWS connector: %v", err)
				}
			case "gcp":
				connector, err = gcp.NewGoogleCloudConnector(&o.config.Cloud.GCP, o.db)
				if err != nil {
					o.logger.Warn("Failed to create GCP connector: %v", err)
				}
			default:
				o.logger.Warn("Unknown cloud provider: %s", o.config.Cloud.Provider)
			}

			if connector != nil {
				cloudOrch.SetConnector(connector)
				o.cloudOrch = cloudOrch
				o.components = append(o.components, o.cloudOrch)
				o.initComponentHealth(o.cloudOrch.Name())
			}
		}
	} else {
		o.logger.Info("Cloud connector is disabled in configuration")
	}

	o.logger.Info("Initialized %d components successfully", len(o.components))
	return nil
}

// startComponents launches all components as goroutines in the correct order
func (o *HardwareOrchestrator) startComponents() error {
	o.logger.Info("Starting components...")

	// Start components in order (excluding API server which needs special handling)
	for _, component := range o.components {
		// Skip API server - we'll start it separately
		if component.Name() == "APIServer" {
			continue
		}

		o.logger.Info("Starting component: %s", component.Name())
		if err := o.startComponentWithRecovery(component); err != nil {
			return errors.Wrap(err, "failed to start component %s", component.Name())
		}

		// Mark as running
		o.markComponentRunning(component.Name(), true)
	}

	// Start API server with context (special case)
	if o.apiServer != nil {
		o.logger.Info("Starting component: %s", o.apiServer.Name())
		ctx, cancel := context.WithCancel(context.Background())

		// Store cancel function for shutdown
		o.wg.Add(1)
		go func() {
			defer o.wg.Done()
			if err := o.apiServer.Start(ctx); err != nil {
				o.logger.Error("API server error: %v", err)
			}
		}()

		// Store cancel function for later
		go func() {
			<-o.shutdownCh
			cancel()
		}()

		o.markComponentRunning(o.apiServer.Name(), true)
	}

	// Start component health monitoring
	o.wg.Add(1)
	go o.healthMonitorLoop()

	o.logger.Info("All %d components started successfully", len(o.components)+1) // +1 for API server
	return nil
}

// startComponentWithRecovery starts a component with automatic restart on failure
func (o *HardwareOrchestrator) startComponentWithRecovery(component Component) error {
	if err := component.Start(); err != nil {
		return err
	}

	// Launch recovery goroutine
	o.wg.Add(1)
	go func() {
		defer o.wg.Done()
		o.componentRecoveryLoop(component)
	}()

	return nil
}

// componentRecoveryLoop monitors a component and restarts it on failure
func (o *HardwareOrchestrator) componentRecoveryLoop(component Component) {
	// Wait for shutdown signal
	<-o.shutdownCh
}

// healthMonitorLoop periodically checks component health and restarts failed components
func (o *HardwareOrchestrator) healthMonitorLoop() {
	defer o.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-o.shutdownCh:
			return
		case <-ticker.C:
			o.checkComponentHealth()
		}
	}
}

// checkComponentHealth verifies all components are running and restarts if needed
func (o *HardwareOrchestrator) checkComponentHealth() {
	o.healthMu.RLock()
	defer o.healthMu.RUnlock()

	now := time.Now()
	for name, health := range o.componentHealth {
		if !health.isRunning {
			o.logger.Warn("Component %s is not running", name)
			continue
		}

		// Check if we're in a new hour window
		if now.Sub(health.restartWindow) > time.Hour {
			// Reset restart count for new window
			health.restartCount = 0
			health.restartWindow = now
		}

		// Log health status
		if health.restartCount > 0 {
			o.logger.Debug("Component %s health: running (restarts in last hour: %d/5)",
				name, health.restartCount)
		}
	}
}

// initComponentHealth initializes health tracking for a component
func (o *HardwareOrchestrator) initComponentHealth(name string) {
	o.healthMu.Lock()
	defer o.healthMu.Unlock()

	o.componentHealth[name] = &componentHealthInfo{
		name:          name,
		restartCount:  0,
		restartWindow: time.Now(),
		isRunning:     false,
	}
}

// markComponentRunning updates the running status of a component
func (o *HardwareOrchestrator) markComponentRunning(name string, running bool) {
	o.healthMu.Lock()
	defer o.healthMu.Unlock()

	if health, exists := o.componentHealth[name]; exists {
		health.isRunning = running
	}
}

// shutdown performs graceful shutdown of all components
func (o *HardwareOrchestrator) shutdown() error {
	o.logger.Info("=== Heimdal Hardware Sensor Shutting Down ===")

	o.mu.Lock()
	defer o.mu.Unlock()

	// Signal all goroutines to stop
	close(o.shutdownCh)

	// Create a timeout context for shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Stop components in reverse order
	o.logger.Info("Stopping components in reverse order...")

	// Stop API server first
	if o.apiServer != nil {
		o.logger.Info("Stopping component: %s", o.apiServer.Name())
		if err := o.apiServer.Stop(); err != nil {
			o.logger.Warn("Error stopping %s: %v", o.apiServer.Name(), err)
		}
		o.markComponentRunning(o.apiServer.Name(), false)
	}

	// Stop other components in reverse order
	for i := len(o.components) - 1; i >= 0; i-- {
		component := o.components[i]
		o.logger.Info("Stopping component: %s", component.Name())
		if err := component.Stop(); err != nil {
			o.logger.Warn("Error stopping %s: %v", component.Name(), err)
		}
		o.markComponentRunning(component.Name(), false)
	}

	// Close packet capture
	if o.packetCapture != nil {
		o.logger.Info("Closing packet capture...")
		if err := o.packetCapture.Close(); err != nil {
			o.logger.Warn("Error closing packet capture: %v", err)
		}
	}

	// Close communication channels
	o.logger.Info("Closing communication channels...")
	close(o.deviceChan)
	close(o.packetChan)
	close(o.profilerChan)

	// Wait for all goroutines to finish with timeout
	o.logger.Info("Waiting for goroutines to finish...")
	done := make(chan struct{})
	go func() {
		o.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		o.logger.Info("All goroutines finished")
	case <-shutdownCtx.Done():
		o.logger.Warn("Shutdown timeout reached, forcing exit")
	}

	// Close database
	if o.db != nil {
		o.logger.Info("Closing database...")
		errors.SafeClose(o.db, "database")
	}

	o.logger.Info("=== Heimdal Hardware Sensor Stopped ===")
	return nil
}

// GetComponentStatus returns the current status of all components
func (o *HardwareOrchestrator) GetComponentStatus() map[string]bool {
	o.healthMu.RLock()
	defer o.healthMu.RUnlock()

	status := make(map[string]bool)
	for name, health := range o.componentHealth {
		status[name] = health.isRunning
	}

	return status
}


// packetInfoAdapter converts packet.PacketInfo to analyzer.PacketInfo
// This is a temporary adapter until the profiler is migrated to use the core types
func (o *HardwareOrchestrator) packetInfoAdapter() {
	defer o.wg.Done()

	for {
		select {
		case <-o.shutdownCh:
			return
		case info := <-o.packetChan:
			// Convert packet.PacketInfo to analyzer.PacketInfo
			analyzerInfo := analyzer.PacketInfo{
				Timestamp: info.Timestamp,
				SrcMAC:    info.SrcMAC,
				DstIP:     info.DstIP,
				DstPort:   info.DstPort,
				Protocol:  info.Protocol,
				Size:      info.Size,
			}

			// Send to profiler channel (non-blocking)
			select {
			case o.profilerChan <- analyzerInfo:
				// Successfully sent
			default:
				// Channel full, drop packet
			}
		}
	}
}

// analyzerComponent wraps the packet.Analyzer to implement the Component interface
type analyzerComponent struct {
	analyzer  *packet.Analyzer
	netConfig *netconfig.NetworkConfig
	bpfFilter string
}

func (a *analyzerComponent) Start() error {
	return a.analyzer.Start(a.netConfig.Interface, true, a.bpfFilter)
}

func (a *analyzerComponent) Stop() error {
	return a.analyzer.Stop()
}

func (a *analyzerComponent) Name() string {
	return "PacketAnalyzer"
}
