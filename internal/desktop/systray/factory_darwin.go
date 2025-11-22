//go:build darwin

package systray

func newPlatformSystemTray(config *Config) SystemTray {
	return NewDarwinSystemTray(config)
}
