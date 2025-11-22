// +build property

package property

import (
	"sync"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/mosiko1234/heimdal/sensor/internal/desktop/systray"
)

// MockSystemTray implements SystemTray for testing
type MockSystemTray struct {
	notifications []NotificationRecord
	mu            sync.Mutex
}

type NotificationRecord struct {
	Title    string
	Message  string
	Severity systray.NotificationSeverity
	Time     time.Time
}

func NewMockSystemTray() *MockSystemTray {
	return &MockSystemTray{
		notifications: make([]NotificationRecord, 0),
	}
}

func (m *MockSystemTray) Initialize() error {
	return nil
}

func (m *MockSystemTray) UpdateStatus(status systray.Status) {
	// No-op for testing
}

func (m *MockSystemTray) ShowNotification(title, message string, severity systray.NotificationSeverity) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.notifications = append(m.notifications, NotificationRecord{
		Title:    title,
		Message:  message,
		Severity: severity,
		Time:     time.Now(),
	})
}

func (m *MockSystemTray) SetMenu(items []*systray.MenuItem) {
	// No-op for testing
}

func (m *MockSystemTray) Run() {
	// No-op for testing
}

func (m *MockSystemTray) Quit() {
	// No-op for testing
}

func (m *MockSystemTray) GetNotifications() []NotificationRecord {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]NotificationRecord{}, m.notifications...)
}

func (m *MockSystemTray) ClearNotifications() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.notifications = make([]NotificationRecord, 0)
}

// Feature: monorepo-architecture, Property 16: Event Notification Triggering
// Validates: Requirements 12.4
func TestProperty_EventNotificationTriggering(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("New device events trigger notifications with correct content",
		prop.ForAll(
			func(deviceName, deviceIP, deviceMAC string) bool {
				// Create mock system tray
				mockTray := NewMockSystemTray()
				manager := systray.NewNotificationManager(mockTray)

				// Trigger new device notification
				manager.NotifyNewDevice(deviceName, deviceIP, deviceMAC)

				// Verify notification was sent
				notifications := mockTray.GetNotifications()
				if len(notifications) != 1 {
					t.Logf("Expected 1 notification, got %d", len(notifications))
					return false
				}

				notification := notifications[0]

				// Verify notification contains device information
				if notification.Title == "" {
					t.Log("Notification title is empty")
					return false
				}

				// Message should contain device name, IP, and MAC
				if deviceName != "" && !contains(notification.Message, deviceName) {
					t.Logf("Notification message missing device name: %s", deviceName)
					return false
				}

				if deviceIP != "" && !contains(notification.Message, deviceIP) {
					t.Logf("Notification message missing device IP: %s", deviceIP)
					return false
				}

				if deviceMAC != "" && !contains(notification.Message, deviceMAC) {
					t.Logf("Notification message missing device MAC: %s", deviceMAC)
					return false
				}

				// Verify severity is Info for new devices
				if notification.Severity != systray.NotificationInfo {
					t.Logf("Expected Info severity, got %s", notification.Severity)
					return false
				}

				return true
			},
			genDeviceName(),
			genIPString(),
			genMACString(),
		))

	properties.Property("Anomaly events trigger notifications with appropriate severity",
		prop.ForAll(
			func(anomalyType, deviceName, description, severity string) bool {
				// Create mock system tray
				mockTray := NewMockSystemTray()
				manager := systray.NewNotificationManager(mockTray)

				// Trigger anomaly notification
				manager.NotifyAnomaly(anomalyType, deviceName, description, severity)

				// Verify notification was sent
				notifications := mockTray.GetNotifications()
				if len(notifications) != 1 {
					t.Logf("Expected 1 notification, got %d", len(notifications))
					return false
				}

				notification := notifications[0]

				// Verify notification contains anomaly information
				if notification.Title == "" {
					t.Log("Notification title is empty")
					return false
				}

				// Title should contain anomaly type
				if anomalyType != "" && !contains(notification.Title, anomalyType) {
					t.Logf("Notification title missing anomaly type: %s", anomalyType)
					return false
				}

				// Message should contain device name and description
				if deviceName != "" && !contains(notification.Message, deviceName) {
					t.Logf("Notification message missing device name: %s", deviceName)
					return false
				}

				if description != "" && !contains(notification.Message, description) {
					t.Logf("Notification message missing description: %s", description)
					return false
				}

				// Verify severity mapping
				expectedSeverity := mapSeverity(severity)
				if notification.Severity != expectedSeverity {
					t.Logf("Expected %s severity, got %s for input severity %s",
						expectedSeverity, notification.Severity, severity)
					return false
				}

				return true
			},
			genAnomalyType(),
			genDeviceName(),
			genDescription(),
			genSeverity(),
		))

	properties.Property("Disabled notifications do not trigger",
		prop.ForAll(
			func(deviceName, deviceIP, deviceMAC string) bool {
				// Create mock system tray
				mockTray := NewMockSystemTray()
				manager := systray.NewNotificationManager(mockTray)

				// Disable notifications
				manager.SetEnabled(false)

				// Trigger new device notification
				manager.NotifyNewDevice(deviceName, deviceIP, deviceMAC)

				// Verify no notification was sent
				notifications := mockTray.GetNotifications()
				if len(notifications) != 0 {
					t.Logf("Expected 0 notifications when disabled, got %d", len(notifications))
					return false
				}

				return true
			},
			genDeviceName(),
			genIPString(),
			genMACString(),
		))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	if substr == "" {
		return true
	}
	return len(s) >= len(substr) && (s == substr || containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// mapSeverity maps severity string to NotificationSeverity
func mapSeverity(severity string) systray.NotificationSeverity {
	switch severity {
	case "critical", "high":
		return systray.NotificationError
	case "medium":
		return systray.NotificationWarning
	default:
		return systray.NotificationInfo
	}
}

// Generators

func genDeviceName() gopter.Gen {
	return gen.OneConstOf(
		"Smart TV",
		"iPhone",
		"Laptop",
		"IoT Camera",
		"Smart Speaker",
		"Thermostat",
		"",
	)
}

func genIPString() gopter.Gen {
	return gen.SliceOfN(4, gen.IntRange(0, 255)).Map(func(octets []int) string {
		if len(octets) != 4 {
			return ""
		}
		return formatIP(octets[0], octets[1], octets[2], octets[3])
	})
}

func formatIP(a, b, c, d int) string {
	return string([]byte{
		byte('0' + a/100), byte('0' + (a/10)%10), byte('0' + a%10), '.',
		byte('0' + b/100), byte('0' + (b/10)%10), byte('0' + b%10), '.',
		byte('0' + c/100), byte('0' + (c/10)%10), byte('0' + c%10), '.',
		byte('0' + d/100), byte('0' + (d/10)%10), byte('0' + d%10),
	})
}

func genMACString() gopter.Gen {
	return gen.SliceOfN(6, gen.IntRange(0, 255)).Map(func(bytes []int) string {
		if len(bytes) != 6 {
			return ""
		}
		return formatMAC(bytes)
	})
}

func formatMAC(bytes []int) string {
	result := ""
	for i, b := range bytes {
		if i > 0 {
			result += ":"
		}
		result += formatHex(b)
	}
	return result
}

func formatHex(n int) string {
	const hexChars = "0123456789ABCDEF"
	return string([]byte{hexChars[n/16], hexChars[n%16]})
}

func genAnomalyType() gopter.Gen {
	return gen.OneConstOf(
		"Unexpected Destination",
		"Unusual Port",
		"Traffic Spike",
		"Suspicious Activity",
		"Port Scan",
	)
}

func genDescription() gopter.Gen {
	return gen.OneConstOf(
		"Device contacted unknown server",
		"Unusual traffic pattern detected",
		"High volume of connections",
		"Suspicious port activity",
		"",
	)
}

func genSeverity() gopter.Gen {
	return gen.OneConstOf(
		"critical",
		"high",
		"medium",
		"low",
		"info",
	)
}
