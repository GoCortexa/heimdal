// Package classifier provides device type classification based on multiple signals
// including vendor, hostname patterns, mDNS services, and network behavior.
package classifier

// DeviceType represents the primary category of a device
type DeviceType string

const (
	DeviceTypeUnknown     DeviceType = "unknown"
	DeviceTypePhone       DeviceType = "phone"
	DeviceTypeTablet      DeviceType = "tablet"
	DeviceTypeComputer    DeviceType = "computer"
	DeviceTypeLaptop      DeviceType = "laptop"
	DeviceTypeServer      DeviceType = "server"
	DeviceTypeRouter      DeviceType = "router"
	DeviceTypeSwitch      DeviceType = "switch"
	DeviceTypePrinter     DeviceType = "printer"
	DeviceTypeScanner     DeviceType = "scanner"
	DeviceTypeTV          DeviceType = "tv"
	DeviceTypeStreaming   DeviceType = "streaming" // Chromecast, Apple TV, Roku
	DeviceTypeCamera      DeviceType = "camera"
	DeviceTypeSpeaker     DeviceType = "speaker"
	DeviceTypeIoT         DeviceType = "iot"       // Generic IoT device
	DeviceTypeSmartHome   DeviceType = "smarthome" // Smart lights, thermostats, etc.
	DeviceTypeNAS         DeviceType = "nas"       // Network Attached Storage
	DeviceTypeGameConsole DeviceType = "console"
	DeviceTypeWearable    DeviceType = "wearable"
)

// DeviceCategory represents a broader classification
type DeviceCategory string

const (
	CategoryUnknown        DeviceCategory = "unknown"
	CategoryEndpoint       DeviceCategory = "endpoint"       // User devices (phones, computers)
	CategoryInfrastructure DeviceCategory = "infrastructure" // Network equipment
	CategoryIoT            DeviceCategory = "iot"            // IoT and smart devices
	CategoryPeripheral     DeviceCategory = "peripheral"     // Printers, scanners
	CategoryEntertainment  DeviceCategory = "entertainment"  // TVs, streaming, gaming
)

// DeviceInfo contains all classification information for a device
type DeviceInfo struct {
	Type       DeviceType
	Category   DeviceCategory
	Confidence float64  // 0.0 to 1.0
	Signals    []string // List of signals that contributed to classification
}

// GetCategory returns the category for a given device type
func (dt DeviceType) GetCategory() DeviceCategory {
	switch dt {
	case DeviceTypePhone, DeviceTypeTablet, DeviceTypeComputer, DeviceTypeLaptop, DeviceTypeWearable:
		return CategoryEndpoint
	case DeviceTypeRouter, DeviceTypeSwitch, DeviceTypeNAS:
		return CategoryInfrastructure
	case DeviceTypeIoT, DeviceTypeSmartHome, DeviceTypeCamera, DeviceTypeSpeaker:
		return CategoryIoT
	case DeviceTypePrinter, DeviceTypeScanner:
		return CategoryPeripheral
	case DeviceTypeTV, DeviceTypeStreaming, DeviceTypeGameConsole:
		return CategoryEntertainment
	case DeviceTypeServer:
		return CategoryInfrastructure
	default:
		return CategoryUnknown
	}
}

// String returns the string representation of the device type
func (dt DeviceType) String() string {
	return string(dt)
}

// String returns the string representation of the device category
func (dc DeviceCategory) String() string {
	return string(dc)
}
