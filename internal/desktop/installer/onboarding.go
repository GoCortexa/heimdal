// Package installer provides installation and onboarding functionality for the Heimdal Desktop agent.
//
// The onboarding wizard guides users through first-time setup, including:
//   - Permission explanation and verification
//   - Dependency verification (Npcap on Windows, libpcap on macOS/Linux)
//   - Network interface selection and auto-detection
//   - Initial configuration
//
// The wizard provides a console-based interface for headless operation and can be
// extended with GUI support in the future.
package installer

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"runtime"
	"strings"

	"github.com/mosiko1234/heimdal/sensor/internal/desktop/config"
	"github.com/mosiko1234/heimdal/sensor/internal/logger"
	"github.com/mosiko1234/heimdal/sensor/internal/platform"
)

// OnboardingWizard manages the first-run onboarding process
type OnboardingWizard struct {
	config           *config.DesktopConfig
	packetCapture    platform.PacketCaptureProvider
	systemIntegrator platform.SystemIntegrator
	logger           *logger.Logger
	reader           *bufio.Reader
}

// NewOnboardingWizard creates a new onboarding wizard instance
func NewOnboardingWizard(
	cfg *config.DesktopConfig,
	packetCapture platform.PacketCaptureProvider,
	systemIntegrator platform.SystemIntegrator,
) *OnboardingWizard {
	return &OnboardingWizard{
		config:           cfg,
		packetCapture:    packetCapture,
		systemIntegrator: systemIntegrator,
		logger:           logger.NewComponentLogger("OnboardingWizard"),
		reader:           bufio.NewReader(os.Stdin),
	}
}

// Run executes the onboarding wizard
func (w *OnboardingWizard) Run() error {
	w.printWelcome()

	// Step 1: Explain permissions
	if err := w.explainPermissions(); err != nil {
		return fmt.Errorf("permission explanation failed: %w", err)
	}

	// Step 2: Verify dependencies
	if err := w.verifyDependencies(); err != nil {
		return fmt.Errorf("dependency verification failed: %w", err)
	}

	// Step 3: Select network interface
	if err := w.selectNetworkInterface(); err != nil {
		return fmt.Errorf("network interface selection failed: %w", err)
	}

	// Step 4: Configure auto-start (optional)
	if err := w.configureAutoStart(); err != nil {
		w.logger.Warn("Auto-start configuration failed: %v", err)
		// Non-fatal, continue
	}

	w.printCompletion()

	return nil
}

// printWelcome displays the welcome message
func (w *OnboardingWizard) printWelcome() {
	fmt.Println()
	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                                                               ║")
	fmt.Println("║           Welcome to Heimdal Desktop Agent!                  ║")
	fmt.Println("║                                                               ║")
	fmt.Println("║   Network Monitoring and Security for Your Computer          ║")
	fmt.Println("║                                                               ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("This wizard will guide you through the initial setup.")
	fmt.Println()
}

// explainPermissions explains required permissions to the user
func (w *OnboardingWizard) explainPermissions() error {
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println("Step 1: Understanding Required Permissions")
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()

	switch runtime.GOOS {
	case "windows":
		fmt.Println("Heimdal Desktop requires Administrator privileges to:")
		fmt.Println("  • Capture network packets using Npcap")
		fmt.Println("  • Monitor network traffic on your computer")
		fmt.Println("  • Install as a Windows Service (optional)")
		fmt.Println()
		fmt.Println("These permissions are necessary for network monitoring.")
		fmt.Println("Heimdal will NEVER:")
		fmt.Println("  • Modify your system files")
		fmt.Println("  • Access personal data without your consent")
		fmt.Println("  • Send data to external servers (unless cloud sync is enabled)")

	case "darwin":
		fmt.Println("Heimdal Desktop requires elevated privileges to:")
		fmt.Println("  • Capture network packets using libpcap")
		fmt.Println("  • Monitor network traffic on your computer")
		fmt.Println("  • Install as a LaunchAgent (optional)")
		fmt.Println()
		fmt.Println("You may be prompted for your password to grant these permissions.")
		fmt.Println()
		fmt.Println("Heimdal will NEVER:")
		fmt.Println("  • Modify your system files")
		fmt.Println("  • Access personal data without your consent")
		fmt.Println("  • Send data to external servers (unless cloud sync is enabled)")

	case "linux":
		fmt.Println("Heimdal Desktop requires elevated privileges to:")
		fmt.Println("  • Capture network packets using libpcap")
		fmt.Println("  • Monitor network traffic on your computer")
		fmt.Println("  • Install as a systemd user service (optional)")
		fmt.Println()
		fmt.Println("You may need to run with sudo or grant CAP_NET_RAW capability.")
		fmt.Println()
		fmt.Println("Heimdal will NEVER:")
		fmt.Println("  • Modify your system files")
		fmt.Println("  • Access personal data without your consent")
		fmt.Println("  • Send data to external servers (unless cloud sync is enabled)")
	}

	fmt.Println()
	fmt.Print("Do you understand and accept these requirements? (yes/no): ")

	response, err := w.reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read user input: %w", err)
	}

	response = strings.ToLower(strings.TrimSpace(response))
	if response != "yes" && response != "y" {
		return fmt.Errorf("user declined permission requirements")
	}

	fmt.Println()
	return nil
}

// verifyDependencies checks for required dependencies
func (w *OnboardingWizard) verifyDependencies() error {
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println("Step 2: Verifying Dependencies")
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()

	switch runtime.GOOS {
	case "windows":
		return w.verifyNpcap()
	case "darwin":
		return w.verifyLibpcap()
	case "linux":
		return w.verifyLibpcap()
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// verifyNpcap checks if Npcap is installed on Windows
func (w *OnboardingWizard) verifyNpcap() error {
	fmt.Println("Checking for Npcap installation...")

	// Try to open a test packet capture to verify Npcap is installed
	// We'll use a dummy interface name - if Npcap is not installed, this will fail
	testErr := w.packetCapture.Open("", false, "")
	if testErr != nil {
		// Close if it somehow opened
		w.packetCapture.Close()
	}

	// Check if the error indicates Npcap is missing
	if testErr != nil && strings.Contains(strings.ToLower(testErr.Error()), "npcap") {
		fmt.Println()
		fmt.Println("⚠️  Npcap is NOT installed!")
		fmt.Println()
		fmt.Println("Npcap is required for packet capture on Windows.")
		fmt.Println("Please download and install Npcap from:")
		fmt.Println("  https://npcap.com/#download")
		fmt.Println()
		fmt.Println("Installation instructions:")
		fmt.Println("  1. Download the Npcap installer")
		fmt.Println("  2. Run the installer with Administrator privileges")
		fmt.Println("  3. Accept the license agreement")
		fmt.Println("  4. Install with default options")
		fmt.Println("  5. Restart this setup wizard")
		fmt.Println()
		return fmt.Errorf("Npcap is not installed")
	}

	fmt.Println("✓ Npcap is installed")
	fmt.Println()
	return nil
}

// verifyLibpcap checks if libpcap is available on macOS/Linux
func (w *OnboardingWizard) verifyLibpcap() error {
	fmt.Println("Checking for libpcap...")

	// Try to open a test packet capture
	testErr := w.packetCapture.Open("", false, "")
	if testErr != nil {
		w.packetCapture.Close()
	}

	// Check if the error indicates libpcap is missing
	if testErr != nil && (strings.Contains(strings.ToLower(testErr.Error()), "libpcap") ||
		strings.Contains(strings.ToLower(testErr.Error()), "permission")) {

		fmt.Println()
		fmt.Println("⚠️  libpcap is not available or permissions are insufficient!")
		fmt.Println()

		if runtime.GOOS == "darwin" {
			fmt.Println("On macOS, you may need to:")
			fmt.Println("  1. Grant Full Disk Access to Terminal or this application")
			fmt.Println("  2. Run with sudo: sudo ./heimdal-desktop")
			fmt.Println()
			fmt.Println("To grant Full Disk Access:")
			fmt.Println("  System Preferences → Security & Privacy → Privacy → Full Disk Access")
			fmt.Println("  Add Terminal or this application to the list")
		} else {
			fmt.Println("On Linux, you may need to:")
			fmt.Println("  1. Install libpcap-dev: sudo apt-get install libpcap-dev")
			fmt.Println("  2. Grant capabilities: sudo setcap cap_net_raw,cap_net_admin=eip ./heimdal-desktop")
			fmt.Println("  3. Or run with sudo: sudo ./heimdal-desktop")
		}

		fmt.Println()
		fmt.Print("Continue anyway? (yes/no): ")

		response, err := w.reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read user input: %w", err)
		}

		response = strings.ToLower(strings.TrimSpace(response))
		if response != "yes" && response != "y" {
			return fmt.Errorf("user declined to continue without libpcap")
		}

		fmt.Println()
		return nil
	}

	fmt.Println("✓ libpcap is available")
	fmt.Println()
	return nil
}

// selectNetworkInterface allows the user to select a network interface
func (w *OnboardingWizard) selectNetworkInterface() error {
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println("Step 3: Network Interface Selection")
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()

	// Auto-detect network interfaces
	interfaces, err := DetectNetworkInterfaces()
	if err != nil {
		return fmt.Errorf("failed to detect network interfaces: %w", err)
	}

	if len(interfaces) == 0 {
		return fmt.Errorf("no network interfaces found")
	}

	// Display available interfaces
	fmt.Println("Available network interfaces:")
	fmt.Println()

	for i, iface := range interfaces {
		fmt.Printf("  [%d] %s\n", i+1, iface.Name)
		fmt.Printf("      MAC: %s\n", iface.HardwareAddr)
		if len(iface.Addrs) > 0 {
			fmt.Printf("      IP:  %s\n", iface.Addrs[0])
		}
		if iface.IsDefault {
			fmt.Printf("      (Recommended - Default Gateway)\n")
		}
		fmt.Println()
	}

	// Find default interface
	defaultIndex := 0
	for i, iface := range interfaces {
		if iface.IsDefault {
			defaultIndex = i
			break
		}
	}

	fmt.Printf("Select interface [1-%d] (default: %d): ", len(interfaces), defaultIndex+1)

	response, err := w.reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read user input: %w", err)
	}

	response = strings.TrimSpace(response)

	// Use default if empty
	selectedIndex := defaultIndex
	if response != "" {
		var selection int
		_, err := fmt.Sscanf(response, "%d", &selection)
		if err != nil || selection < 1 || selection > len(interfaces) {
			fmt.Printf("Invalid selection, using default: %s\n", interfaces[defaultIndex].Name)
		} else {
			selectedIndex = selection - 1
		}
	}

	// Update configuration
	w.config.Network.Interface = interfaces[selectedIndex].Name
	w.config.Network.AutoDetect = false

	fmt.Printf("\n✓ Selected interface: %s\n", interfaces[selectedIndex].Name)
	fmt.Println()

	return nil
}

// configureAutoStart asks if the user wants to enable auto-start
func (w *OnboardingWizard) configureAutoStart() error {
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println("Step 4: Auto-Start Configuration (Optional)")
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()

	fmt.Println("Would you like Heimdal Desktop to start automatically when you log in?")
	fmt.Print("Enable auto-start? (yes/no, default: no): ")

	response, err := w.reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read user input: %w", err)
	}

	response = strings.ToLower(strings.TrimSpace(response))

	if response == "yes" || response == "y" {
		w.config.SystemTray.AutoStart = true
		fmt.Println("\n✓ Auto-start enabled")
	} else {
		w.config.SystemTray.AutoStart = false
		fmt.Println("\n✓ Auto-start disabled")
	}

	fmt.Println()
	return nil
}

// printCompletion displays the completion message
func (w *OnboardingWizard) printCompletion() {
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println("Setup Complete!")
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()
	fmt.Println("Heimdal Desktop is now configured and ready to use.")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  • The agent will start monitoring your network")
	fmt.Println("  • Access the dashboard at: http://localhost:8080")
	fmt.Println("  • Check the system tray for status and controls")
	fmt.Println()
	fmt.Println("For help and documentation, visit:")
	fmt.Println("  https://heimdal.io/docs")
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()
}

// NetworkInterfaceInfo contains information about a network interface
type NetworkInterfaceInfo struct {
	Name         string
	HardwareAddr string
	Addrs        []string
	IsUp         bool
	IsDefault    bool
}

// DetectNetworkInterfaces detects available network interfaces
func DetectNetworkInterfaces() ([]NetworkInterfaceInfo, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to get network interfaces: %w", err)
	}

	var result []NetworkInterfaceInfo

	// Get default gateway interface
	defaultIface := getDefaultInterface()

	for _, iface := range interfaces {
		// Skip loopback and down interfaces
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if iface.Flags&net.FlagUp == 0 {
			continue
		}

		// Get addresses
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		// Skip interfaces without addresses
		if len(addrs) == 0 {
			continue
		}

		info := NetworkInterfaceInfo{
			Name:         iface.Name,
			HardwareAddr: iface.HardwareAddr.String(),
			Addrs:        make([]string, 0, len(addrs)),
			IsUp:         iface.Flags&net.FlagUp != 0,
			IsDefault:    iface.Name == defaultIface,
		}

		// Extract IP addresses
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok {
				// Skip IPv6 link-local addresses
				if ipnet.IP.IsLinkLocalUnicast() {
					continue
				}
				info.Addrs = append(info.Addrs, ipnet.IP.String())
			}
		}

		// Only include interfaces with valid addresses
		if len(info.Addrs) > 0 {
			result = append(result, info)
		}
	}

	return result, nil
}

// getDefaultInterface attempts to determine the default network interface
func getDefaultInterface() string {
	// Try to connect to a public IP to determine which interface is used
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return ""
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	// Find the interface with this IP
	interfaces, err := net.Interfaces()
	if err != nil {
		return ""
	}

	for _, iface := range interfaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok {
				if ipnet.IP.Equal(localAddr.IP) {
					return iface.Name
				}
			}
		}
	}

	return ""
}
