package classifier

import (
	"strings"
)

// Classifier classifies devices based on multiple signals
type Classifier struct {
	// Future: could add ML models, custom rules, etc.
}

// NewClassifier creates a new device classifier
func NewClassifier() *Classifier {
	return &Classifier{}
}

// ClassifyDevice determines the device type based on available information
// It combines signals from vendor, hostname, and mDNS services with weighted confidence
func (c *Classifier) ClassifyDevice(vendor, manufacturer, hostname string, services []string) *DeviceInfo {
	signals := make([]string, 0, 4)
	var totalConfidence float64
	var weightedType map[DeviceType]float64 = make(map[DeviceType]float64)

	// Signal 1: Vendor/Manufacturer matching
	if deviceType, confidence, matched := matchVendor(vendor); matched {
		weightedType[deviceType] += confidence * 1.0
		totalConfidence += confidence
		signals = append(signals, "vendor")
	} else if deviceType, confidence, matched := matchVendor(manufacturer); matched {
		weightedType[deviceType] += confidence * 0.9 // Slightly lower weight for manufacturer
		totalConfidence += confidence * 0.9
		signals = append(signals, "manufacturer")
	}

	// Signal 2: Hostname matching (higher confidence)
	if deviceType, confidence, matched := matchHostname(hostname); matched {
		weightedType[deviceType] += confidence * 1.8 // Higher weight for hostname
		totalConfidence += confidence * 1.8
		signals = append(signals, "hostname")
	}

	// Signal 3: mDNS services (highest confidence)
	if deviceType, confidence, matched := matchServices(services); matched {
		weightedType[deviceType] += confidence * 2.0 // Highest weight for explicit services
		totalConfidence += confidence * 2.0
		signals = append(signals, "mdns_service")
	}

	// Find device type with highest weighted score
	var finalType DeviceType = DeviceTypeUnknown
	var maxWeight float64 = 0

	for deviceType, weight := range weightedType {
		if weight > maxWeight {
			maxWeight = weight
			finalType = deviceType
		}
	}

	// Calculate final confidence (normalize to 0-1 range)
	// Max possible weight is ~4.0 (vendor + hostname*1.8 + service*2.0)
	finalConfidence := maxWeight / 4.0
	if finalConfidence > 1.0 {
		finalConfidence = 1.0
	}

	// Apply special rules for refinement
	finalType = c.refineClassification(finalType, vendor, manufacturer, hostname, services)

	return &DeviceInfo{
		Type:       finalType,
		Category:   finalType.GetCategory(),
		Confidence: finalConfidence,
		Signals:    signals,
	}
}

// refineClassification applies special rules to refine the classification
func (c *Classifier) refineClassification(deviceType DeviceType, vendor, manufacturer, hostname string, services []string) DeviceType {
	vendor = strings.ToLower(vendor)
	manufacturer = strings.ToLower(manufacturer)
	hostname = strings.ToLower(hostname)

	// Apple device refinement
	if strings.Contains(vendor, "apple") || strings.Contains(manufacturer, "apple") {
		// Check hostname for specific Apple device types
		if strings.Contains(hostname, "iphone") {
			return DeviceTypePhone
		}
		if strings.Contains(hostname, "ipad") {
			return DeviceTypeTablet
		}
		if strings.Contains(hostname, "macbook") {
			return DeviceTypeLaptop
		}
		if strings.Contains(hostname, "imac") || strings.Contains(hostname, "mac-mini") {
			return DeviceTypeComputer
		}
		if strings.Contains(hostname, "appletv") || strings.Contains(hostname, "apple-tv") {
			return DeviceTypeStreaming
		}
		if strings.Contains(hostname, "watch") {
			return DeviceTypeWearable
		}

		// Check services
		for _, service := range services {
			if strings.Contains(service, "_airplay") {
				return DeviceTypeStreaming
			}
		}
	}

	// Google device refinement
	if strings.Contains(vendor, "google") || strings.Contains(manufacturer, "google") {
		if strings.Contains(hostname, "chromecast") {
			return DeviceTypeStreaming
		}
		if strings.Contains(hostname, "nest") {
			return DeviceTypeSmartHome
		}
		if strings.Contains(hostname, "pixel") {
			return DeviceTypePhone
		}
	}

	// Amazon device refinement
	if strings.Contains(vendor, "amazon") || strings.Contains(manufacturer, "amazon") {
		if strings.Contains(hostname, "echo") || strings.Contains(hostname, "alexa") {
			return DeviceTypeSpeaker
		}
		if strings.Contains(hostname, "fire") {
			return DeviceTypeStreaming
		}
		if strings.Contains(hostname, "ring") {
			return DeviceTypeCamera
		}
	}

	// Raspberry Pi is usually IoT unless hostname suggests otherwise
	if strings.Contains(vendor, "raspberry") || strings.Contains(hostname, "raspberrypi") {
		// Check if it's being used as a server
		if strings.Contains(hostname, "server") || strings.Contains(hostname, "nas") {
			return DeviceTypeServer
		}
		return DeviceTypeIoT
	}

	// If we have printer service but classified as computer, upgrade to printer
	if deviceType == DeviceTypeComputer {
		for _, service := range services {
			if strings.Contains(service, "_printer") || strings.Contains(service, "_ipp") {
				return DeviceTypePrinter
			}
		}
	}
	
	// HP/Hewlett Packard with "LaserJet", "DeskJet", etc. in name/hostname is always a printer
	if (strings.Contains(vendor, "hp") || strings.Contains(vendor, "hewlett") || 
		strings.Contains(manufacturer, "hp") || strings.Contains(manufacturer, "hewlett")) &&
		(strings.Contains(hostname, "laserjet") || strings.Contains(hostname, "deskjet") || 
		 strings.Contains(hostname, "officejet") || strings.Contains(hostname, "printer")) {
		return DeviceTypePrinter
	}
	
	return deviceType
}

// ClassifyByVendorOnly provides a quick classification based only on vendor
// Useful for initial classification before other signals are available
func (c *Classifier) ClassifyByVendorOnly(vendor string) DeviceType {
	deviceType, _, matched := matchVendor(vendor)
	if matched {
		return deviceType
	}
	return DeviceTypeUnknown
}

// GetConfidenceLevel returns a human-readable confidence level
func GetConfidenceLevel(confidence float64) string {
	if confidence >= 0.8 {
		return "high"
	} else if confidence >= 0.5 {
		return "medium"
	} else if confidence > 0 {
		return "low"
	}
	return "none"
}
