// Package orchestrator provides the main coordination and lifecycle management
// for the Heimdal Desktop agent components. It handles component initialization,
// startup sequencing, health monitoring, and graceful shutdown.
//
// This orchestrator is specifically for the desktop product (Windows, macOS, Linux)
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

	"github.com/mosiko1234/heimdal/sensor/internal/cloud"
	"github.com/mosiko1234/heimdal/sensor/internal/core/detection"
	"github.com/mosiko1234/heimdal/sensor/internal/core/packet"
	"github.com/mosiko1234/heimdal/sensor/internal/core/profiler"
	"github.com/mosiko1234/heimdal/sensor/internal/database"
	"github.com/mosiko1234/heimdal/sensor/internal/desktop/config"
	"github.com/mosiko1234/heimdal/sensor/internal/desktop/featuregate"
	"github.com/mosiko1234/heimdal/sensor/internal/desktop/interceptor"
	"github.com/mosiko1234/heimdal/sensor/internal/desktop/systray"
	"github.com/mosiko1234/heimdal/sensor/internal/desktop/visualizer"
	"github.com/mosiko1234/heimdal/sensor/internal/discovery"
	"github.com/mosiko1234/heimdal/sensor/internal/errors"
	"github.com/mosiko1234/heimdal/sensor/internal/logger"
	"github.com/mosiko1234/heimdal/sensor/internal/platform"
)

// Component interface defines the lifecycle methods for all components
type Component interface {
	Start() error
	Stop() error
	Name() string
}

// DesktopOrchestrator manages the lifecycle of all Heimdal desktop agent components
// using platform abstraction interfaces for packet capture, system integration, and storage.
type DesktopOrchestrator struct {
	config *config.DesktopConfig
	logger *logger.Logger

	// Platform interfaces
	packetCapture    platform.PacketCaptureProvider
	systemIntegrator platform.SystemIntegrator
	storage          platform.StorageProvider

	// Component instances
	featureGate         *featuregate.FeatureGate
	deviceScanner       *discovery.Scanner
	deviceStore         database.DeviceStore
	trafficInterceptor  *interceptor.DesktopTrafficInterceptor
	analyzer            *packet.Analyzer
	profilerComp        *profiler.Profiler
	detector            *detection.Detector
	visualizerComp      *visualizer.Visualizer
	systemTray          *systray.SystemTray
	cloudOrch           *cloud.Orchestrator
	discoveryStatusCh   chan discovery.StatusUpdate
	lastDiscoveryStatus string
	lastDiscoveryLevel  discovery.StatusLevel

	// Communication channels
	packetChan  chan packet.PacketInfo
	anomalyChan chan *detection.Anomaly

	// Lifecycle management
	ctx        context.Context
	cancel     context.CancelFunc
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

// NewDesktopOrchestrator creates a new desktop orchestrator instance with platform interfaces
func NewDesktopOrchestrator(
	cfg *config.DesktopConfig,
	packetCapture platform.PacketCaptureProvider,
	systemIntegrator platform.SystemIntegrator,
	storage platform.StorageProvider,
) (*DesktopOrchestrator, error) {
	if cfg == nil {
		return nil, fmt.Errorf("configuration is required")
	}
	if packetCapture == nil {
		return nil, fmt.Errorf("packet capture provider is required")
	}
	if storage == nil {
		return nil, fmt.Errorf("storage provider is required")
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &DesktopOrchestrator{
		config:            cfg,
		packetCapture:     packetCapture,
		systemIntegrator:  systemIntegrator,
		storage:           storage,
		logger:            logger.NewComponentLogger("DesktopOrchestrator"),
		ctx:               ctx,
		cancel:            cancel,
		packetChan:        make(chan packet.PacketInfo, 1000),
		anomalyChan:       make(chan *detection.Anomaly, 100),
		shutdownCh:        make(chan struct{}),
		componentHealth:   make(map[string]*componentHealthInfo),
		discoveryStatusCh: make(chan discovery.StatusUpdate, 16),
	}, nil
}

// Run starts all components and blocks until shutdown signal is received
func (o *DesktopOrchestrator) Run() error {
	o.logger.Info("=== Heimdal Desktop Agent Starting ===")

	// Initialize all components
	if err := o.initializeComponents(); err != nil {
		return errors.Wrap(err, "failed to initialize components")
	}

	// Start all components in correct order
	if err := o.startComponents(); err != nil {
		return errors.Wrap(err, "failed to start components")
	}

	o.logger.Info("=== Heimdal Desktop Agent Running ===")

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
func (o *DesktopOrchestrator) initializeComponents() error {
	o.logger.Info("Initializing components...")

	// 1. Initialize Storage
	o.logger.Info("Initializing storage at %s", o.config.Database.Path)
	storageOpts := &platform.StorageOptions{
		ReadOnly:   false,
		SyncWrites: true,
		CacheSize:  100 * 1024 * 1024, // 100MB cache
	}
	if err := o.storage.Open(o.config.Database.Path, storageOpts); err != nil {
		return errors.Wrap(err, "failed to open storage")
	}

	// 2. Initialize Feature Gate
	o.logger.Info("Initializing feature gate with tier: %s", o.config.FeatureGate.Tier)
	// Create a local license validator (can be nil for free tier)
	var validator featuregate.LicenseValidator
	if o.config.FeatureGate.Tier != string(featuregate.TierFree) {
		// Use a default secret for local validation
		validator = featuregate.NewLocalLicenseValidator("heimdal-secret-key")
	}
	fg := featuregate.New(featuregate.Tier(o.config.FeatureGate.Tier), o.config.FeatureGate.LicenseKey, validator)
	o.featureGate = fg

	// 3. Initialize Device Discovery (network + scanner)
	if err := o.initializeDeviceDiscovery(); err != nil {
		return errors.Wrap(err, "failed to initialize device discovery")
	}

	// 4. Initialize Packet Analyzer
	o.logger.Info("Initializing packet analyzer with platform interface...")
	analyzer, err := packet.NewAnalyzer(o.packetCapture, o.packetChan, nil)
	if err != nil {
		return errors.Wrap(err, "failed to initialize packet analyzer")
	}
	o.analyzer = analyzer

	// 5. Initialize Behavioral Profiler
	o.logger.Info("Initializing behavioral profiler...")
	profilerCfg := profiler.DefaultConfig()
	profilerComp, err := profiler.NewProfiler(o.storage, o.packetChan, profilerCfg)
	if err != nil {
		return errors.Wrap(err, "failed to initialize profiler")
	}
	o.profilerComp = profilerComp
	o.initComponentHealth("Profiler")

	// 6. Initialize Anomaly Detector
	o.logger.Info("Initializing anomaly detector...")
	detectorCfg := &detection.Config{
		Sensitivity:       o.config.Detection.Sensitivity,
		BaselineThreshold: 100,
	}
	detector, err := detection.NewDetector(detectorCfg)
	if err != nil {
		return errors.Wrap(err, "failed to initialize detector")
	}
	o.detector = detector
	o.initComponentHealth("Detector")

	// 7. Initialize Traffic Interceptor (if enabled and tier allows)
	if o.config.Interceptor.Enabled {
		if o.featureGate.CanAccess(featuregate.FeatureTrafficBlocking) {
			o.logger.Info("Initializing traffic interceptor...")
			interceptorCfg := &interceptor.Config{
				InterfaceName: o.config.Network.Interface,
				GatewayIP:     nil, // Will be auto-detected
				SpoofInterval: 2 * time.Second,
				MaxTargets:    50,
			}
			trafficInterceptor, err := interceptor.NewDesktopTrafficInterceptor(interceptorCfg)
			if err != nil {
				o.logger.Warn("Failed to initialize traffic interceptor: %v", err)
				o.logger.Info("Continuing without traffic interception")
			} else {
				o.trafficInterceptor = trafficInterceptor
				o.initComponentHealth("TrafficInterceptor")
			}
		} else {
			o.logger.Info("Traffic interceptor requires Pro tier or higher")
		}
	} else {
		o.logger.Info("Traffic interceptor is disabled in configuration")
	}

	// 8. Initialize Local Visualizer
	o.logger.Info("Initializing local visualizer on port %d", o.config.Visualizer.Port)
	visualizerCfg := &visualizer.Config{
		Port:        o.config.Visualizer.Port,
		Storage:     o.storage,
		FeatureGate: o.featureGate,
	}
	visualizerComp, err := visualizer.NewVisualizer(visualizerCfg)
	if err != nil {
		return errors.Wrap(err, "failed to initialize visualizer")
	}
	o.visualizerComp = visualizerComp
	o.initComponentHealth("Visualizer")

	// 9. Initialize System Tray
	o.logger.Info("Initializing system tray...")
	systemTray := NewPlatformSystemTray(o.visualizerComp, o.config.SystemTray.AutoStart)
	o.systemTray = systemTray
	o.initComponentHealth("SystemTray")

	// 10. Initialize Cloud Connector (if enabled)
	if o.config.Cloud.Enabled {
		o.logger.Info("Initializing cloud connector...")
		if err := o.initializeCloudConnector(); err != nil {
			o.logger.Warn("Failed to initialize cloud connector: %v", err)
			o.logger.Info("Local operations will continue without cloud connectivity")
		}
	} else {
		o.logger.Info("Cloud connector is disabled in configuration")
	}

	o.logger.Info("Initialized components successfully")
	return nil
}

// startComponents launches all components as goroutines in the correct order
func (o *DesktopOrchestrator) startComponents() error {
	o.logger.Info("Starting components...")

	// Start device discovery scanner
	if o.deviceScanner != nil {
		o.logger.Info("Starting device discovery scanner...")
		if err := o.deviceScanner.Start(); err != nil {
			return errors.Wrap(err, "failed to start device discovery scanner")
		}
		o.markComponentRunning(o.deviceScanner.Name(), true)
	}

	// 1. Start Packet Analyzer
	o.logger.Info("Starting packet analyzer on interface: %s", o.config.Network.Interface)
	if err := o.analyzer.Start(o.config.Network.Interface, true, ""); err != nil {
		return errors.Wrap(err, "failed to start packet analyzer")
	}
	o.initComponentHealth("PacketAnalyzer")
	o.markComponentRunning("PacketAnalyzer", true)

	// 2. Start Profiler
	o.logger.Info("Starting profiler...")
	if err := o.profilerComp.Start(); err != nil {
		return errors.Wrap(err, "failed to start profiler")
	}
	o.markComponentRunning("Profiler", true)

	// 3. Start Detector (runs in background)
	o.logger.Info("Starting anomaly detector...")
	o.wg.Add(1)
	go o.detectorLoop()
	o.markComponentRunning("Detector", true)

	// 4. Start Traffic Interceptor (if initialized)
	if o.trafficInterceptor != nil {
		o.logger.Info("Starting traffic interceptor...")
		if err := o.trafficInterceptor.Start(); err != nil {
			o.logger.Warn("Failed to start traffic interceptor: %v", err)
		} else {
			o.markComponentRunning("TrafficInterceptor", true)
		}
	}

	// 5. Start Visualizer
	o.logger.Info("Starting visualizer...")
	if err := o.visualizerComp.Start(); err != nil {
		o.logger.Warn("Failed to start visualizer: %v", err)
	} else {
		o.markComponentRunning("Visualizer", true)
	}

	// 6. Start System Tray (if initialized)
	if o.systemTray != nil {
		o.logger.Info("Starting system tray...")
		if err := (*o.systemTray).Initialize(); err != nil {
			o.logger.Warn("Failed to start system tray: %v", err)
		} else {
			o.markComponentRunning("SystemTray", true)
		}
	}

	// 7. Start Cloud Connector (if initialized)
	if o.cloudOrch != nil {
		o.logger.Info("Starting cloud connector...")
		if err := o.cloudOrch.Start(); err != nil {
			o.logger.Warn("Failed to start cloud connector: %v", err)
		} else {
			o.markComponentRunning(o.cloudOrch.Name(), true)
		}
	}

	// 8. Start event notification handler
	o.wg.Add(1)
	go o.eventNotificationLoop()

	// 9. Start component health monitoring
	o.wg.Add(1)
	go o.healthMonitorLoop()

	o.logger.Info("All components started successfully")
	return nil
}

// detectorLoop runs the anomaly detector in the background
func (o *DesktopOrchestrator) detectorLoop() {
	defer o.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-o.shutdownCh:
			return
		case <-ticker.C:
			// Get all profiles from storage and analyze
			profiles := o.profilerComp.GetAllProfiles()

			// Analyze each profile for anomalies
			for _, profile := range profiles {
				anomalies, err := o.detector.Analyze(profile)
				if err != nil {
					o.logger.Error("Failed to analyze profile %s: %v", profile.MAC, err)
					continue
				}

				// Send anomalies to notification channel
				for _, anomaly := range anomalies {
					select {
					case o.anomalyChan <- anomaly:
					default:
						// Channel full, drop anomaly
					}
				}
			}
		}
	}
}

// eventNotificationLoop handles anomaly events for notifications
func (o *DesktopOrchestrator) eventNotificationLoop() {
	defer o.wg.Done()

	for {
		select {
		case <-o.shutdownCh:
			return
		case anomaly := <-o.anomalyChan:
			// Anomaly detected
			if o.systemTray != nil {
				severity := systray.NotificationWarning
				if anomaly.Severity == detection.SeverityHigh || anomaly.Severity == detection.SeverityCritical {
					severity = systray.NotificationError
				}
				(*o.systemTray).ShowNotification(
					"Anomaly Detected",
					fmt.Sprintf("%s: %s", anomaly.Type, anomaly.Description),
					severity,
				)
			}
			// Send to visualizer for real-time update
			if o.visualizerComp != nil {
				o.visualizerComp.BroadcastUpdate("anomaly", anomaly)
			}
		}
	}
}

// healthMonitorLoop periodically checks component health
func (o *DesktopOrchestrator) healthMonitorLoop() {
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

// checkComponentHealth verifies all components are running
func (o *DesktopOrchestrator) checkComponentHealth() {
	o.healthMu.RLock()
	defer o.healthMu.RUnlock()

	allHealthy := true
	for name, health := range o.componentHealth {
		if !health.isRunning {
			o.logger.Warn("Component %s is not running", name)
			allHealthy = false
		}
	}

	// Update system tray status
	if o.systemTray != nil {
		if allHealthy {
			(*o.systemTray).UpdateStatus(systray.StatusActive)
		} else {
			(*o.systemTray).UpdateStatus(systray.StatusError)
		}
	}
}

func (o *DesktopOrchestrator) initializeDeviceDiscovery() error {
	if o.deviceStore == nil {
		store, err := newStorageDeviceStore(o.storage)
		if err != nil {
			return err
		}
		o.deviceStore = store
	}

	provider, err := o.buildNetworkConfigProvider()
	if err != nil {
		return err
	}

	scanInterval := time.Duration(o.config.Discovery.ScanInterval) * time.Second
	inactiveTimeout := time.Duration(o.config.Discovery.InactiveTimeout) * time.Minute

	statusSink := o.discoveryStatusSink()
	o.deviceScanner = discovery.NewScanner(provider, o.deviceStore, nil, scanInterval, o.config.Discovery.MDNSEnabled, inactiveTimeout, nil, statusSink)
	o.initComponentHealth(o.deviceScanner.Name())
	o.startDiscoveryStatusMonitor()

	o.logger.Info("Device discovery initialized (interface=%s, scan_interval=%v, mdns=%v)",
		o.config.Network.Interface, scanInterval, o.config.Discovery.MDNSEnabled)
	return nil
}

func (o *DesktopOrchestrator) initializeCloudConnector() error {
	// For now, cloud connector is a stub for desktop
	// The desktop agent will send telemetry data when cloud is enabled
	// Full implementation requires adapting the cloud.Orchestrator to work with
	// desktop config types or creating a unified config interface
	
	o.logger.Info("Cloud connector initialization deferred (provider: %s)", o.config.Cloud.Provider)
	o.logger.Info("Telemetry will be sent via stub connector for free tier")
	
	// TODO: Implement full cloud connector for desktop
	// This requires either:
	// 1. Creating adapter types to convert desktop.config types to config types
	// 2. Refactoring cloud package to use interface-based config
	// 3. Creating a separate desktop cloud connector package
	
	return nil
}

func (o *DesktopOrchestrator) discoveryStatusSink() discovery.StatusSink {
	if o.discoveryStatusCh == nil {
		return nil
	}

	return func(update discovery.StatusUpdate) {
		select {
		case o.discoveryStatusCh <- update:
		default:
			<-o.discoveryStatusCh
			o.discoveryStatusCh <- update
		}
	}
}

func (o *DesktopOrchestrator) startDiscoveryStatusMonitor() {
	if o.discoveryStatusCh == nil {
		return
	}

	o.wg.Add(1)
	go func() {
		defer o.wg.Done()
		for {
			select {
			case <-o.shutdownCh:
				return
			case update := <-o.discoveryStatusCh:
				o.handleDiscoveryStatus(update)
			}
		}
	}()
}

func (o *DesktopOrchestrator) handleDiscoveryStatus(update discovery.StatusUpdate) {
	if update.Message == "" {
		return
	}

	if update.Message == o.lastDiscoveryStatus && update.Level == o.lastDiscoveryLevel {
		return
	}

	o.lastDiscoveryStatus = update.Message
	o.lastDiscoveryLevel = update.Level

	o.logger.Info("Discovery status update: %s", update.Message)

	if o.systemTray != nil {
		label := fmt.Sprintf("Discovery: %s", update.Message)
		menu := []*systray.MenuItem{
			{Label: label, Enabled: false},
		}
		(*o.systemTray).SetMenu(menu)

		switch update.Level {
		case discovery.StatusLevelWarning:
			(*o.systemTray).ShowNotification("Discovery warning", update.Message, systray.NotificationWarning)
		case discovery.StatusLevelError:
			(*o.systemTray).ShowNotification("Discovery error", update.Message, systray.NotificationError)
		}
	}
}

// initComponentHealth initializes health tracking for a component
func (o *DesktopOrchestrator) initComponentHealth(name string) {
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
func (o *DesktopOrchestrator) markComponentRunning(name string, running bool) {
	o.healthMu.Lock()
	defer o.healthMu.Unlock()

	if health, exists := o.componentHealth[name]; exists {
		health.isRunning = running
	}
}

// shutdown performs graceful shutdown of all components
func (o *DesktopOrchestrator) shutdown() error {
	o.logger.Info("=== Heimdal Desktop Agent Shutting Down ===")

	o.mu.Lock()
	defer o.mu.Unlock()

	// Signal all goroutines to stop
	close(o.shutdownCh)
	o.cancel()

	// Create a timeout context for shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Stop components in reverse order
	o.logger.Info("Stopping components in reverse order...")

	// 1. Stop System Tray
	if o.systemTray != nil {
		o.logger.Info("Stopping system tray...")
		(*o.systemTray).UpdateStatus(systray.StatusPaused)
		o.markComponentRunning("SystemTray", false)
	}

	// 2. Stop Visualizer
	if o.visualizerComp != nil {
		o.logger.Info("Stopping visualizer...")
		if err := o.visualizerComp.Stop(); err != nil {
			o.logger.Warn("Error stopping visualizer: %v", err)
		}
		o.markComponentRunning("Visualizer", false)
	}

	// 3. Stop Traffic Interceptor
	if o.trafficInterceptor != nil {
		o.logger.Info("Stopping traffic interceptor...")
		if err := o.trafficInterceptor.Stop(); err != nil {
			o.logger.Warn("Error stopping traffic interceptor: %v", err)
		}
		o.markComponentRunning("TrafficInterceptor", false)
	}

	// 4. Stop Detector (will stop via shutdownCh)
	o.markComponentRunning("Detector", false)

	// 5. Stop Profiler
	if o.profilerComp != nil {
		o.logger.Info("Stopping profiler...")
		if err := o.profilerComp.Stop(); err != nil {
			o.logger.Warn("Error stopping profiler: %v", err)
		}
		o.markComponentRunning("Profiler", false)
	}

	// 6. Stop Packet Analyzer
	if o.analyzer != nil {
		o.logger.Info("Stopping packet analyzer...")
		if err := o.analyzer.Stop(); err != nil {
			o.logger.Warn("Error stopping packet analyzer: %v", err)
		}
		o.markComponentRunning("PacketAnalyzer", false)
	}

	// 7. Stop Cloud Connector
	if o.cloudOrch != nil {
		o.logger.Info("Stopping cloud connector...")
		if err := o.cloudOrch.Stop(); err != nil {
			o.logger.Warn("Error stopping cloud connector: %v", err)
		}
		o.markComponentRunning(o.cloudOrch.Name(), false)
	}

	// 8. Stop Device Discovery
	if o.deviceScanner != nil {
		o.logger.Info("Stopping device discovery scanner...")
		if err := o.deviceScanner.Stop(); err != nil {
			o.logger.Warn("Error stopping device discovery scanner: %v", err)
		}
		o.markComponentRunning(o.deviceScanner.Name(), false)
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
	close(o.packetChan)
	close(o.anomalyChan)

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

	// Close storage
	if o.storage != nil {
		o.logger.Info("Closing storage...")
		if err := o.storage.Close(); err != nil {
			o.logger.Warn("Error closing storage: %v", err)
		}
	}

	o.logger.Info("=== Heimdal Desktop Agent Stopped ===")
	return nil
}

// GetComponentStatus returns the current status of all components
func (o *DesktopOrchestrator) GetComponentStatus() map[string]bool {
	o.healthMu.RLock()
	defer o.healthMu.RUnlock()

	status := make(map[string]bool)
	for name, health := range o.componentHealth {
		status[name] = health.isRunning
	}

	return status
}

// PauseMonitoring pauses packet capture and analysis
func (o *DesktopOrchestrator) PauseMonitoring() error {
	o.logger.Info("Pausing monitoring...")

	if o.analyzer != nil {
		if err := o.analyzer.Stop(); err != nil {
			return errors.Wrap(err, "failed to stop analyzer")
		}
		o.markComponentRunning("PacketAnalyzer", false)
	}

	if o.systemTray != nil {
		(*o.systemTray).UpdateStatus(systray.StatusPaused)
	}

	return nil
}

// ResumeMonitoring resumes packet capture and analysis
func (o *DesktopOrchestrator) ResumeMonitoring() error {
	o.logger.Info("Resuming monitoring...")

	if o.analyzer != nil {
		if err := o.analyzer.Start(o.config.Network.Interface, true, ""); err != nil {
			return errors.Wrap(err, "failed to start analyzer")
		}
		o.markComponentRunning("PacketAnalyzer", true)
	}

	if o.systemTray != nil {
		(*o.systemTray).UpdateStatus(systray.StatusActive)
	}

	return nil
}

// NewPlatformSystemTray creates a platform-specific system tray implementation
func NewPlatformSystemTray(visualizer *visualizer.Visualizer, autoStart bool) *systray.SystemTray {
	config := &systray.Config{
		AppName:     "Heimdal Desktop",
		IconPath:    "",
		TooltipText: "Heimdal Network Monitor",
	}

	st := systray.NewPlatformSystemTray(config)
	return &st
}
