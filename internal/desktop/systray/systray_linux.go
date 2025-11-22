//go:build linux

package systray

import (
	"fmt"
	"sync"

	"github.com/getlantern/systray"
)

// LinuxSystemTray implements SystemTray for Linux
type LinuxSystemTray struct {
	config        *Config
	currentStatus Status
	menuItems     []*MenuItem
	mu            sync.RWMutex
	quitChan      chan struct{}
	ready         chan struct{}
}

// NewLinuxSystemTray creates a new Linux system tray instance
func NewLinuxSystemTray(config *Config) *LinuxSystemTray {
	return &LinuxSystemTray{
		config:   config,
		quitChan: make(chan struct{}),
		ready:    make(chan struct{}),
	}
}

// Initialize sets up the system tray
func (l *LinuxSystemTray) Initialize() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.config == nil {
		return fmt.Errorf("config is required")
	}

	return nil
}

// UpdateStatus changes the icon to reflect current status
func (l *LinuxSystemTray) UpdateStatus(status Status) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.currentStatus = status

	// Update tooltip based on status
	tooltip := l.config.TooltipText
	switch status {
	case StatusActive:
		tooltip = fmt.Sprintf("%s - Active", l.config.AppName)
	case StatusPaused:
		tooltip = fmt.Sprintf("%s - Paused", l.config.AppName)
	case StatusError:
		tooltip = fmt.Sprintf("%s - Error", l.config.AppName)
	}

	systray.SetTooltip(tooltip)

	// In a real implementation, we would change the icon here
	// based on the status (e.g., different colored icons)
	// Linux system tray icons work with various desktop environments
	// (GNOME, KDE, XFCE, etc.) through the StatusNotifierItem spec
}

// ShowNotification displays a desktop notification
func (l *LinuxSystemTray) ShowNotification(title, message string, severity NotificationSeverity) {
	// Linux notifications can be implemented using:
	// - libnotify (D-Bus notifications)
	// - Desktop Notifications Specification
	// For now, this is a placeholder that would integrate with
	// a Linux notification library

	// Example implementation would use:
	// github.com/gen2brain/beeep for cross-platform notifications
	// or direct D-Bus calls for more control
	fmt.Printf("[Linux Notification] %s: %s (severity: %s)\n", title, message, severity)
}

// SetMenu updates the system tray menu
func (l *LinuxSystemTray) SetMenu(items []*MenuItem) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.menuItems = items
}

// Run starts the system tray event loop (blocking)
func (l *LinuxSystemTray) Run() {
	systray.Run(l.onReady, l.onExit)
}

// onReady is called when the system tray is ready
func (l *LinuxSystemTray) onReady() {
	l.mu.RLock()
	config := l.config
	l.mu.RUnlock()

	// Set initial icon and tooltip
	// In a real implementation, we would load the icon from config.IconPath
	systray.SetTitle(config.AppName)
	systray.SetTooltip(config.TooltipText)

	// Build the menu
	l.buildMenu()

	close(l.ready)
}

// onExit is called when the system tray is exiting
func (l *LinuxSystemTray) onExit() {
	// Cleanup
	close(l.quitChan)
}

// buildMenu constructs the system tray menu from menu items
func (l *LinuxSystemTray) buildMenu() {
	l.mu.RLock()
	items := l.menuItems
	l.mu.RUnlock()

	// Clear existing menu
	// Note: systray library doesn't have a clear method,
	// so we need to rebuild on each SetMenu call

	for _, item := range items {
		l.addMenuItem(item)
	}
}

// addMenuItem adds a menu item to the system tray
func (l *LinuxSystemTray) addMenuItem(item *MenuItem) {
	if item == nil {
		return
	}

	menuItem := systray.AddMenuItem(item.Label, "")

	if !item.Enabled {
		menuItem.Disable()
	} else {
		menuItem.Enable()
	}

	if item.Checked {
		menuItem.Check()
	} else {
		menuItem.Uncheck()
	}

	// Handle click events
	if item.OnClick != nil {
		go func(onClick func()) {
			for {
				select {
				case <-menuItem.ClickedCh:
					onClick()
				case <-l.quitChan:
					return
				}
			}
		}(item.OnClick)
	}

	// Handle children (submenus)
	if len(item.Children) > 0 {
		for _, child := range item.Children {
			l.addSubMenuItem(menuItem, child)
		}
	}
}

// addSubMenuItem adds a submenu item
func (l *LinuxSystemTray) addSubMenuItem(parent *systray.MenuItem, item *MenuItem) {
	if item == nil {
		return
	}

	subItem := parent.AddSubMenuItem(item.Label, "")

	if !item.Enabled {
		subItem.Disable()
	} else {
		subItem.Enable()
	}

	if item.Checked {
		subItem.Check()
	} else {
		subItem.Uncheck()
	}

	// Handle click events
	if item.OnClick != nil {
		go func(onClick func()) {
			for {
				select {
				case <-subItem.ClickedCh:
					onClick()
				case <-l.quitChan:
					return
				}
			}
		}(item.OnClick)
	}
}

// Quit stops the system tray and cleans up resources
func (l *LinuxSystemTray) Quit() {
	systray.Quit()
}
