//go:build darwin

package systray

import (
	"fmt"
	"sync"

	"github.com/getlantern/systray"
)

// DarwinSystemTray implements SystemTray for macOS
type DarwinSystemTray struct {
	config        *Config
	currentStatus Status
	menuItems     []*MenuItem
	mu            sync.RWMutex
	quitChan      chan struct{}
	ready         chan struct{}
}

// NewDarwinSystemTray creates a new macOS system tray instance
func NewDarwinSystemTray(config *Config) *DarwinSystemTray {
	return &DarwinSystemTray{
		config:   config,
		quitChan: make(chan struct{}),
		ready:    make(chan struct{}),
	}
}

// Initialize sets up the system tray
func (d *DarwinSystemTray) Initialize() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.config == nil {
		return fmt.Errorf("config is required")
	}

	return nil
}

// UpdateStatus changes the icon to reflect current status
func (d *DarwinSystemTray) UpdateStatus(status Status) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.currentStatus = status

	// Update tooltip based on status
	tooltip := d.config.TooltipText
	switch status {
	case StatusActive:
		tooltip = fmt.Sprintf("%s - Active", d.config.AppName)
		// On macOS, we could also update the menu bar icon
		// to show a green indicator
	case StatusPaused:
		tooltip = fmt.Sprintf("%s - Paused", d.config.AppName)
		// Show a yellow/orange indicator
	case StatusError:
		tooltip = fmt.Sprintf("%s - Error", d.config.AppName)
		// Show a red indicator
	}

	systray.SetTooltip(tooltip)

	// In a real implementation, we would change the icon here
	// based on the status (e.g., different colored icons)
	// macOS menu bar icons are typically monochrome with template images
}

// ShowNotification displays a desktop notification
func (d *DarwinSystemTray) ShowNotification(title, message string, severity NotificationSeverity) {
	// macOS notifications can be implemented using:
	// - NSUserNotificationCenter (older macOS)
	// - UNUserNotificationCenter (macOS 10.14+)
	// For now, this is a placeholder that would integrate with
	// a macOS notification library

	// Example implementation would use:
	// github.com/deckarep/gosx-notifier for macOS notifications
	fmt.Printf("[macOS Notification] %s: %s (severity: %s)\n", title, message, severity)
}

// SetMenu updates the system tray menu
func (d *DarwinSystemTray) SetMenu(items []*MenuItem) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.menuItems = items
}

// Run starts the system tray event loop (blocking)
func (d *DarwinSystemTray) Run() {
	systray.Run(d.onReady, d.onExit)
}

// onReady is called when the system tray is ready
func (d *DarwinSystemTray) onReady() {
	d.mu.RLock()
	config := d.config
	d.mu.RUnlock()

	// Set initial icon and tooltip
	// In a real implementation, we would load the icon from config.IconPath
	// macOS menu bar icons should be template images (monochrome)
	systray.SetTitle(config.AppName)
	systray.SetTooltip(config.TooltipText)

	// Build the menu
	d.buildMenu()

	close(d.ready)
}

// onExit is called when the system tray is exiting
func (d *DarwinSystemTray) onExit() {
	// Cleanup
	close(d.quitChan)
}

// buildMenu constructs the system tray menu from menu items
func (d *DarwinSystemTray) buildMenu() {
	d.mu.RLock()
	items := d.menuItems
	d.mu.RUnlock()

	// Clear existing menu
	// Note: systray library doesn't have a clear method,
	// so we need to rebuild on each SetMenu call

	for _, item := range items {
		d.addMenuItem(item)
	}
}

// addMenuItem adds a menu item to the system tray
func (d *DarwinSystemTray) addMenuItem(item *MenuItem) {
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
				case <-d.quitChan:
					return
				}
			}
		}(item.OnClick)
	}

	// Handle children (submenus)
	if len(item.Children) > 0 {
		for _, child := range item.Children {
			d.addSubMenuItem(menuItem, child)
		}
	}
}

// addSubMenuItem adds a submenu item
func (d *DarwinSystemTray) addSubMenuItem(parent *systray.MenuItem, item *MenuItem) {
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
				case <-d.quitChan:
					return
				}
			}
		}(item.OnClick)
	}
}

// Quit stops the system tray and cleans up resources
func (d *DarwinSystemTray) Quit() {
	systray.Quit()
}
