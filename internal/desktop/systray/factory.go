package systray

// NewPlatformSystemTray creates a platform-specific system tray implementation
// This function is implemented in platform-specific files with build tags
func NewPlatformSystemTray(config *Config) SystemTray {
	return newPlatformSystemTray(config)
}
