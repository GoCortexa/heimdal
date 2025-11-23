package classifier

import (
	"strings"
)

// VendorRule maps vendor patterns to device types
type VendorRule struct {
	Pattern    string
	DeviceType DeviceType
	Confidence float64
}

// HostnameRule maps hostname patterns to device types
type HostnameRule struct {
	Pattern    string
	DeviceType DeviceType
	Confidence float64
}

// ServiceRule maps mDNS service types to device types
type ServiceRule struct {
	ServiceType string
	DeviceType  DeviceType
	Confidence  float64
}

// vendorRules defines classification rules based on vendor/manufacturer
var vendorRules = []VendorRule{
	// Mobile Devices
	{"Apple", DeviceTypePhone, 0.6}, // Could be phone, tablet, or computer
	{"Samsung", DeviceTypePhone, 0.6},
	{"Google", DeviceTypePhone, 0.5},
	{"Huawei", DeviceTypePhone, 0.7},
	{"Xiaomi", DeviceTypePhone, 0.7},
	{"OnePlus", DeviceTypePhone, 0.9},
	{"Motorola Mobility", DeviceTypePhone, 0.8},
	{"LG Electronics", DeviceTypePhone, 0.6},
	{"Sony Mobile", DeviceTypePhone, 0.9},
	{"HTC", DeviceTypePhone, 0.9},

	// Computers
	{"Dell", DeviceTypeComputer, 0.7},
	{"HP Inc", DeviceTypeComputer, 0.6}, // HP makes both computers and printers
	{"Hewlett Packard", DeviceTypeComputer, 0.6},
	{"Lenovo", DeviceTypeComputer, 0.7},
	{"Asus", DeviceTypeComputer, 0.7},
	{"Acer", DeviceTypeComputer, 0.7},
	{"Microsoft", DeviceTypeComputer, 0.6},
	{"Intel", DeviceTypeComputer, 0.5},

	// Network Equipment
	{"Cisco", DeviceTypeRouter, 0.7},
	{"Netgear", DeviceTypeRouter, 0.8},
	{"TP-Link", DeviceTypeRouter, 0.8},
	{"D-Link", DeviceTypeRouter, 0.8},
	{"Ubiquiti", DeviceTypeRouter, 0.8},
	{"MikroTik", DeviceTypeRouter, 0.9},
	{"Aruba", DeviceTypeRouter, 0.7},
	{"Juniper", DeviceTypeRouter, 0.8},

	// IoT & Smart Home
	{"Raspberry Pi", DeviceTypeIoT, 0.8},
	{"Amazon", DeviceTypeIoT, 0.6}, // Echo, Fire TV
	{"Ring", DeviceTypeCamera, 0.9},
	{"Nest", DeviceTypeSmartHome, 0.9},
	{"Philips Lighting", DeviceTypeSmartHome, 0.9},
	{"Belkin", DeviceTypeIoT, 0.6},
	{"Sonos", DeviceTypeSpeaker, 0.9},
	{"Bose", DeviceTypeSpeaker, 0.8},

	// Streaming & Entertainment
	{"Roku", DeviceTypeStreaming, 0.9},
	{"Chromecast", DeviceTypeStreaming, 0.9},
	{"Sony", DeviceTypeTV, 0.5},
	{"Samsung Electronics", DeviceTypeTV, 0.5},
	{"LG", DeviceTypeTV, 0.5},
	{"Nintendo", DeviceTypeGameConsole, 0.9},
	{"Sony Computer Entertainment", DeviceTypeGameConsole, 0.9},
	{"Microsoft Xbox", DeviceTypeGameConsole, 0.9},

	// Printers & Scanners
	{"HP", DeviceTypePrinter, 0.7}, // HP printers are common
	{"Hewlett Packard", DeviceTypePrinter, 0.7},
	{"Canon", DeviceTypePrinter, 0.7},
	{"Epson", DeviceTypePrinter, 0.8},
	{"Brother", DeviceTypePrinter, 0.8},
	{"Xerox", DeviceTypePrinter, 0.9},
	{"Ricoh", DeviceTypePrinter, 0.8},

	// NAS & Storage
	{"Synology", DeviceTypeNAS, 0.9},
	{"QNAP", DeviceTypeNAS, 0.9},
	{"Western Digital", DeviceTypeNAS, 0.7},
	{"Seagate", DeviceTypeNAS, 0.7},
}

// hostnameRules defines classification rules based on hostname patterns
var hostnameRules = []HostnameRule{
	// Mobile
	{"iphone", DeviceTypePhone, 0.95},
	{"ipad", DeviceTypeTablet, 0.95},
	{"android", DeviceTypePhone, 0.9},
	{"galaxy", DeviceTypePhone, 0.8},
	{"pixel", DeviceTypePhone, 0.9},

	// Computers
	{"macbook", DeviceTypeLaptop, 0.95},
	{"imac", DeviceTypeComputer, 0.95},
	{"mac-mini", DeviceTypeComputer, 0.95},
	{"desktop", DeviceTypeComputer, 0.8},
	{"laptop", DeviceTypeLaptop, 0.8},
	{"thinkpad", DeviceTypeLaptop, 0.9},

	// IoT
	{"raspberrypi", DeviceTypeIoT, 0.9},
	{"raspberry", DeviceTypeIoT, 0.8},
	{"pi-", DeviceTypeIoT, 0.7},
	{"esp", DeviceTypeIoT, 0.8},
	{"arduino", DeviceTypeIoT, 0.9},

	// Smart Home
	{"alexa", DeviceTypeSpeaker, 0.9},
	{"echo", DeviceTypeSpeaker, 0.9},
	{"nest", DeviceTypeSmartHome, 0.9},
	{"hue", DeviceTypeSmartHome, 0.9},

	// Entertainment
	{"roku", DeviceTypeStreaming, 0.95},
	{"chromecast", DeviceTypeStreaming, 0.95},
	{"appletv", DeviceTypeStreaming, 0.95},
	{"firetv", DeviceTypeStreaming, 0.95},
	{"xbox", DeviceTypeGameConsole, 0.95},
	{"playstation", DeviceTypeGameConsole, 0.95},
	{"ps4", DeviceTypeGameConsole, 0.95},
	{"ps5", DeviceTypeGameConsole, 0.95},

	// Network Equipment
	{"router", DeviceTypeRouter, 0.9},
	{"switch", DeviceTypeSwitch, 0.9},
	{"access-point", DeviceTypeRouter, 0.8},
	{"ap-", DeviceTypeRouter, 0.7},

	// Printers
	{"printer", DeviceTypePrinter, 0.9},
	{"print", DeviceTypePrinter, 0.7},
	{"laserjet", DeviceTypePrinter, 0.95},
	{"deskjet", DeviceTypePrinter, 0.95},
	{"officejet", DeviceTypePrinter, 0.95},

	// NAS
	{"nas", DeviceTypeNAS, 0.9},
	{"synology", DeviceTypeNAS, 0.95},
	{"qnap", DeviceTypeNAS, 0.95},

	// Cameras
	{"camera", DeviceTypeCamera, 0.9},
	{"cam-", DeviceTypeCamera, 0.8},
	{"ring", DeviceTypeCamera, 0.8},
}

// serviceRules defines classification rules based on mDNS service types
var serviceRules = []ServiceRule{
	// Streaming & Media
	{"_airplay._tcp", DeviceTypeStreaming, 0.9},
	{"_raop._tcp", DeviceTypeStreaming, 0.8},
	{"_googlecast._tcp", DeviceTypeStreaming, 0.95},
	{"_spotify-connect._tcp", DeviceTypeSpeaker, 0.8},

	// Printing
	{"_printer._tcp", DeviceTypePrinter, 0.95},
	{"_ipp._tcp", DeviceTypePrinter, 0.9},
	{"_pdl-datastream._tcp", DeviceTypePrinter, 0.9},

	// Scanning
	{"_scanner._tcp", DeviceTypeScanner, 0.95},
	{"_uscan._tcp", DeviceTypeScanner, 0.9},

	// Smart Home
	{"_hap._tcp", DeviceTypeSmartHome, 0.9}, // HomeKit
	{"_homekit._tcp", DeviceTypeSmartHome, 0.9},
	{"_matter._tcp", DeviceTypeSmartHome, 0.9},

	// File Sharing
	{"_smb._tcp", DeviceTypeNAS, 0.7},
	{"_afpovertcp._tcp", DeviceTypeNAS, 0.8},
	{"_nfs._tcp", DeviceTypeNAS, 0.8},

	// Workstations
	{"_workstation._tcp", DeviceTypeComputer, 0.7},
	{"_ssh._tcp", DeviceTypeComputer, 0.5},

	// Web Services
	{"_http._tcp", DeviceTypeComputer, 0.3},
	{"_https._tcp", DeviceTypeComputer, 0.3},
}

// matchVendor checks if a vendor string matches any vendor rules
func matchVendor(vendor string) (DeviceType, float64, bool) {
	if vendor == "" {
		return DeviceTypeUnknown, 0, false
	}

	vendor = strings.ToLower(vendor)

	var bestMatch DeviceType
	var bestConfidence float64

	for _, rule := range vendorRules {
		pattern := strings.ToLower(rule.Pattern)
		if strings.Contains(vendor, pattern) {
			if rule.Confidence > bestConfidence {
				bestMatch = rule.DeviceType
				bestConfidence = rule.Confidence
			}
		}
	}

	if bestConfidence > 0 {
		return bestMatch, bestConfidence, true
	}

	return DeviceTypeUnknown, 0, false
}

// matchHostname checks if a hostname matches any hostname rules
func matchHostname(hostname string) (DeviceType, float64, bool) {
	if hostname == "" {
		return DeviceTypeUnknown, 0, false
	}

	hostname = strings.ToLower(hostname)

	var bestMatch DeviceType
	var bestConfidence float64

	for _, rule := range hostnameRules {
		pattern := strings.ToLower(rule.Pattern)
		if strings.Contains(hostname, pattern) {
			if rule.Confidence > bestConfidence {
				bestMatch = rule.DeviceType
				bestConfidence = rule.Confidence
			}
		}
	}

	if bestConfidence > 0 {
		return bestMatch, bestConfidence, true
	}

	return DeviceTypeUnknown, 0, false
}

// matchServices checks if any mDNS services match service rules
func matchServices(services []string) (DeviceType, float64, bool) {
	if len(services) == 0 {
		return DeviceTypeUnknown, 0, false
	}

	var bestMatch DeviceType
	var bestConfidence float64

	for _, service := range services {
		service = strings.ToLower(service)

		for _, rule := range serviceRules {
			if strings.Contains(service, strings.ToLower(rule.ServiceType)) {
				if rule.Confidence > bestConfidence {
					bestMatch = rule.DeviceType
					bestConfidence = rule.Confidence
				}
			}
		}
	}

	if bestConfidence > 0 {
		return bestMatch, bestConfidence, true
	}

	return DeviceTypeUnknown, 0, false
}
