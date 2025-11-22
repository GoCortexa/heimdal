//go:build linux

package systray

func newPlatformSystemTray(config *Config) SystemTray {
	return NewLinuxSystemTray(config)
}
