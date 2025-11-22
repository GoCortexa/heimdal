// +build windows

package desktop_windows

import (
	"github.com/mosiko1234/heimdal/sensor/internal/platform"
)

// Compile-time interface checks
var (
	_ platform.PacketCaptureProvider = (*WindowsPacketCapture)(nil)
	_ platform.SystemIntegrator      = (*WindowsSystemIntegrator)(nil)
	_ platform.StorageProvider       = (*WindowsStorage)(nil)
)
