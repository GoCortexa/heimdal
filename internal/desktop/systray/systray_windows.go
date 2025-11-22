//go:build windows

package systray

import (
	"fmt"
	"sync"

	"github.com/getlantern/systray"
)

// WindowsSystemTray implements SystemTray for Windows
type WindowsSystemTray struct {
	config       *Config
	currentStatus Status
	menuItems    []*MenuItem
	mu           sync.RWMutex
	quitChan     chan struct{}
	ready        chan struct{}
}

// NewWindowsSystemTray creates a new Windows system tray instance
func NewWindowsSystemTray(config *Config) *WindowsSystemTray {
	return &WindowsSystemTray{
		config:   config,
		quitChan: make(chan struct{}),
		ready:    make(chan struct{}),
	}
}

// Initialize sets up the system tray
func (w *WindowsSystemTray) Initialize() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.config == nil {
		return fmt.Errorf("config is required")
	}

	return nil
}

// UpdateStatus changes the icon to reflect current status
func (w *WindowsSystemTray) UpdateStatus(status Status) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.currentStatus = status

	// Update tooltip based on status
	tooltip := w.config.TooltipText
	switch status {
	case StatusActive:
		tooltip = fmt.Sprintf("%s - Active", w.config.AppName)
	case StatusPaused:
		tooltip = fmt.Sprintf("%s - Paused", w.config.AppName)
	case StatusError:
		tooltip = fmt.Sprintf("%s - Error", w.config.AppName)
	}

	systray.SetTooltip(tooltip)

	// In a real implementation, we would change the icon here
	// based on the status (e.g., different colored icons)
}

// ShowNotification displays a desktop notification
func (w *WindowsSystemTray) ShowNotification(title, message string, severity NotificationSeverity) {
	// Windows notifications can be implemented using:
	// - toast notifications (Windows 10+)
	// - balloon tips (older Windows)
	// For now, this is a placeholder that would integrate with
	// a Windows notification library

	// Example implementation would use:
	// github.com/go-toast/toast for Windows 10+ toast notifications
	fmt.Printf("[Windows Notification] %s: %s (severity: %s)\n", title, message, severity)
}

// SetMenu updates the system tray menu
func (w *WindowsSystemTray) SetMenu(items []*MenuItem) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.menuItems = items
}

// Run starts the system tray event loop (blocking)
func (w *WindowsSystemTray) Run() {
	systray.Run(w.onReady, w.onExit)
}

// onReady is called when the system tray is ready
func (w *WindowsSystemTray) onReady() {
	w.mu.RLock()
	config := w.config
	w.mu.RUnlock()

	// Set initial icon and tooltip
	// In a real implementation, we would load the icon from config.IconPath
	systray.SetTitle(config.AppName)
	systray.SetTooltip(config.TooltipText)

	// Build the menu
	w.buildMenu()

	close(w.ready)
}

// onExit is called when the system tray is exiting
func (w *WindowsSystemTray) onExit() {
	// Cleanup
	close(w.quitChan)
}

// buildMenu constructs the system tray menu from menu items
func (w *WindowsSystemTray) buildMenu() {
	w.mu.RLock()
	items := w.menuItems
	w.mu.RUnlock()

	// Clear existing menu
	// Note: systray library doesn't have a clear method,
	// so we need to rebuild on each SetMenu call

	for _, item := range items {
		w.addMenuItem(item)
	}
}

// addMenuItem adds a menu item to the system tray
func (w *WindowsSystemTray) addMenuItem(item *MenuItem) {
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
				case <-w.quitChan:
					return
				}
			}
		}(item.OnClick)
	}

	// Handle children (submenus)
	if len(item.Children) > 0 {
		for _, child := range item.Children {
			w.addSubMenuItem(menuItem, child)
		}
	}
}

// addSubMenuItem adds a submenu item
func (w *WindowsSystemTray) addSubMenuItem(parent *systray.MenuItem, item *MenuItem) {
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
				case <-w.quitChan:
					return
				}
			}
		}(item.OnClick)
	}
}

// Quit stops the system tray and cleans up resources
func (w *WindowsSystemTray) Quit() {
	systray.Quit()
}
