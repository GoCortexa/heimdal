package systray

import (
	"fmt"
	"time"
)

// NotificationManager handles desktop notifications across platforms
type NotificationManager struct {
	systray SystemTray
	enabled bool
}

// NewNotificationManager creates a new notification manager
func NewNotificationManager(systray SystemTray) *NotificationManager {
	return &NotificationManager{
		systray: systray,
		enabled: true,
	}
}

// SetEnabled enables or disables notifications
func (nm *NotificationManager) SetEnabled(enabled bool) {
	nm.enabled = enabled
}

// IsEnabled returns whether notifications are enabled
func (nm *NotificationManager) IsEnabled() bool {
	return nm.enabled
}

// NotifyNewDevice sends a notification when a new device is detected
func (nm *NotificationManager) NotifyNewDevice(deviceName, deviceIP, deviceMAC string) {
	if !nm.enabled {
		return
	}

	title := "New Device Detected"
	message := fmt.Sprintf("Device: %s\nIP: %s\nMAC: %s", deviceName, deviceIP, deviceMAC)

	nm.systray.ShowNotification(title, message, NotificationInfo)
}

// NotifyAnomaly sends a notification when an anomaly is detected
func (nm *NotificationManager) NotifyAnomaly(anomalyType, deviceName, description string, severity string) {
	if !nm.enabled {
		return
	}

	title := fmt.Sprintf("Security Alert: %s", anomalyType)
	message := fmt.Sprintf("Device: %s\n%s", deviceName, description)

	// Map severity to notification severity
	var notifSeverity NotificationSeverity
	switch severity {
	case "critical", "high":
		notifSeverity = NotificationError
	case "medium":
		notifSeverity = NotificationWarning
	default:
		notifSeverity = NotificationInfo
	}

	nm.systray.ShowNotification(title, message, notifSeverity)
}

// NotifyStatusChange sends a notification when the monitoring status changes
func (nm *NotificationManager) NotifyStatusChange(oldStatus, newStatus Status) {
	if !nm.enabled {
		return
	}

	title := "Heimdal Status Changed"
	message := fmt.Sprintf("Status changed from %s to %s", oldStatus, newStatus)

	var severity NotificationSeverity
	if newStatus == StatusError {
		severity = NotificationError
	} else if newStatus == StatusPaused {
		severity = NotificationWarning
	} else {
		severity = NotificationInfo
	}

	nm.systray.ShowNotification(title, message, severity)
}

// NotifyError sends a notification for general errors
func (nm *NotificationManager) NotifyError(title, message string) {
	if !nm.enabled {
		return
	}

	nm.systray.ShowNotification(title, message, NotificationError)
}

// NotifyInfo sends a notification for general information
func (nm *NotificationManager) NotifyInfo(title, message string) {
	if !nm.enabled {
		return
	}

	nm.systray.ShowNotification(title, message, NotificationInfo)
}

// EventNotifier handles event-based notifications
type EventNotifier struct {
	manager           *NotificationManager
	deviceCache       map[string]time.Time // MAC -> last notification time
	anomalyCache      map[string]time.Time // anomaly key -> last notification time
	notificationDelay time.Duration        // minimum time between duplicate notifications
}

// NewEventNotifier creates a new event notifier
func NewEventNotifier(manager *NotificationManager) *EventNotifier {
	return &EventNotifier{
		manager:           manager,
		deviceCache:       make(map[string]time.Time),
		anomalyCache:      make(map[string]time.Time),
		notificationDelay: 5 * time.Minute, // default: don't repeat same notification within 5 minutes
	}
}

// SetNotificationDelay sets the minimum time between duplicate notifications
func (en *EventNotifier) SetNotificationDelay(delay time.Duration) {
	en.notificationDelay = delay
}

// OnNewDevice handles new device detection events
func (en *EventNotifier) OnNewDevice(deviceName, deviceIP, deviceMAC string) {
	// Check if we recently notified about this device
	if lastNotification, exists := en.deviceCache[deviceMAC]; exists {
		if time.Since(lastNotification) < en.notificationDelay {
			return // Skip duplicate notification
		}
	}

	// Send notification
	en.manager.NotifyNewDevice(deviceName, deviceIP, deviceMAC)

	// Update cache
	en.deviceCache[deviceMAC] = time.Now()
}

// OnAnomaly handles anomaly detection events
func (en *EventNotifier) OnAnomaly(anomalyType, deviceName, deviceMAC, description, severity string) {
	// Create a unique key for this anomaly
	anomalyKey := fmt.Sprintf("%s:%s:%s", deviceMAC, anomalyType, severity)

	// Check if we recently notified about this anomaly
	if lastNotification, exists := en.anomalyCache[anomalyKey]; exists {
		if time.Since(lastNotification) < en.notificationDelay {
			return // Skip duplicate notification
		}
	}

	// Send notification
	en.manager.NotifyAnomaly(anomalyType, deviceName, description, severity)

	// Update cache
	en.anomalyCache[anomalyKey] = time.Now()
}

// CleanupCache removes old entries from the notification cache
func (en *EventNotifier) CleanupCache() {
	now := time.Now()
	cleanupThreshold := en.notificationDelay * 2

	// Clean device cache
	for mac, lastTime := range en.deviceCache {
		if now.Sub(lastTime) > cleanupThreshold {
			delete(en.deviceCache, mac)
		}
	}

	// Clean anomaly cache
	for key, lastTime := range en.anomalyCache {
		if now.Sub(lastTime) > cleanupThreshold {
			delete(en.anomalyCache, key)
		}
	}
}
