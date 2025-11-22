//go:build windows

package systray

func newPlatformSystemTray(config *Config) SystemTray {
	return NewWindowsSystemTray(config)
}
