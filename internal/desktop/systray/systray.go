package systray

import (
	"time"
)

// Status represents the monitoring status of the application
type Status string

const (
	StatusActive Status = "active"
	StatusPaused Status = "paused"
	StatusError  Status = "error"
)

// NotificationSeverity represents the severity level of a notification
type NotificationSeverity string

const (
	NotificationInfo    NotificationSeverity = "info"
	NotificationWarning NotificationSeverity = "warning"
	NotificationError   NotificationSeverity = "error"
)

// MenuItem represents a menu item in the system tray
type MenuItem struct {
	Label    string
	Enabled  bool
	Checked  bool
	OnClick  func()
	Children []*MenuItem
}

// SystemTray manages the system tray icon and menu
type SystemTray interface {
	// Initialize sets up the system tray
	Initialize() error

	// UpdateStatus changes the icon to reflect current status
	UpdateStatus(status Status)

	// ShowNotification displays a desktop notification
	ShowNotification(title, message string, severity NotificationSeverity)

	// SetMenu updates the system tray menu
	SetMenu(items []*MenuItem)

	// Run starts the system tray event loop (blocking)
	Run()

	// Quit stops the system tray and cleans up resources
	Quit()
}

// EventType represents different types of system tray events
type EventType string

const (
	EventMenuItemClicked EventType = "menu_item_clicked"
	EventIconClicked     EventType = "icon_clicked"
	EventQuit            EventType = "quit"
)

// Event represents a system tray event
type Event struct {
	Type      EventType
	Timestamp time.Time
	Data      interface{}
}

// Config contains configuration for the system tray
type Config struct {
	AppName     string
	IconPath    string
	TooltipText string
}
