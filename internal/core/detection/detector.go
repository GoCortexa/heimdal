// Package detection provides shared anomaly detection functionality for both
// hardware and desktop products.
//
// The Detector analyzes behavioral profiles to identify unusual communication
// patterns that may indicate security threats or device malfunctions.
package detection

import (
	"fmt"
	"math"
	"time"

	"github.com/mosiko1234/heimdal/sensor/internal/database"
)

// AnomalyType categorizes the type of anomaly
type AnomalyType string

const (
	AnomalyUnexpectedDestination AnomalyType = "unexpected_destination"
	AnomalyUnusualPort           AnomalyType = "unusual_port"
	AnomalyTrafficSpike          AnomalyType = "traffic_spike"
	AnomalyNewDevice             AnomalyType = "new_device"
	AnomalyDormantDevice         AnomalyType = "dormant_device"
)

// Severity represents the severity level of an anomaly
type Severity string

const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

// Anomaly represents a detected anomalous behavior
type Anomaly struct {
	DeviceMAC   string                 `json:"device_mac"`
	Type        AnomalyType            `json:"type"`
	Severity    Severity               `json:"severity"`
	Description string                 `json:"description"`
	Timestamp   time.Time              `json:"timestamp"`
	Evidence    map[string]interface{} `json:"evidence"`
}

// Detector identifies anomalous behavior patterns
type Detector struct {
	sensitivity       float64
	baselineThreshold int64 // Minimum packets before establishing baseline
}

// Config contains configuration for the anomaly detector
type Config struct {
	// Sensitivity controls how aggressive the detector is (0.0 to 1.0)
	// Higher values = more sensitive = more anomalies detected
	Sensitivity float64

	// BaselineThreshold is the minimum number of packets before establishing baseline
	BaselineThreshold int64
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Sensitivity:       0.5, // Medium sensitivity
		BaselineThreshold: 100, // Require 100 packets for baseline
	}
}

// NewDetector creates a new anomaly detector instance
func NewDetector(cfg *Config) (*Detector, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// Validate sensitivity
	if cfg.Sensitivity < 0.0 || cfg.Sensitivity > 1.0 {
		return nil, fmt.Errorf("sensitivity must be between 0.0 and 1.0, got %f", cfg.Sensitivity)
	}

	// Validate baseline threshold
	if cfg.BaselineThreshold < 0 {
		return nil, fmt.Errorf("baseline threshold must be non-negative, got %d", cfg.BaselineThreshold)
	}

	detector := &Detector{
		sensitivity:       cfg.Sensitivity,
		baselineThreshold: cfg.BaselineThreshold,
	}

	return detector, nil
}

// Analyze examines a profile for anomalies
func (d *Detector) Analyze(profile *database.BehavioralProfile) ([]*Anomaly, error) {
	if profile == nil {
		return nil, fmt.Errorf("profile is nil")
	}

	anomalies := make([]*Anomaly, 0)

	// Check if we have enough data for baseline
	if profile.TotalPackets < d.baselineThreshold {
		// Not enough data yet, skip analysis
		return anomalies, nil
	}

	// Detect unexpected destinations
	destAnomalies := d.detectUnexpectedDestinations(profile)
	anomalies = append(anomalies, destAnomalies...)

	// Detect unusual ports
	portAnomalies := d.detectUnusualPorts(profile)
	anomalies = append(anomalies, portAnomalies...)

	// Detect traffic spikes
	spikeAnomalies := d.detectTrafficSpikes(profile)
	anomalies = append(anomalies, spikeAnomalies...)

	return anomalies, nil
}

// detectUnexpectedDestinations identifies communication with unusual destinations
func (d *Detector) detectUnexpectedDestinations(profile *database.BehavioralProfile) []*Anomaly {
	anomalies := make([]*Anomaly, 0)

	// Calculate average destination frequency
	if len(profile.Destinations) == 0 {
		return anomalies
	}

	var totalCount int64
	for _, dest := range profile.Destinations {
		totalCount += dest.Count
	}
	avgCount := float64(totalCount) / float64(len(profile.Destinations))

	// Detect destinations with unusually low frequency (potential new/suspicious destinations)
	threshold := avgCount * (1.0 - d.sensitivity)

	for ip, dest := range profile.Destinations {
		if float64(dest.Count) < threshold && dest.Count < 5 {
			// This is a rarely contacted destination
			severity := d.calculateSeverity(float64(dest.Count), avgCount)

			anomaly := &Anomaly{
				DeviceMAC:   profile.MAC,
				Type:        AnomalyUnexpectedDestination,
				Severity:    severity,
				Description: fmt.Sprintf("Device contacted unusual destination %s (count: %d, avg: %.1f)", ip, dest.Count, avgCount),
				Timestamp:   time.Now(),
				Evidence: map[string]interface{}{
					"destination_ip": ip,
					"count":          dest.Count,
					"average_count":  avgCount,
					"last_seen":      dest.LastSeen,
				},
			}
			anomalies = append(anomalies, anomaly)
		}
	}

	return anomalies
}

// detectUnusualPorts identifies communication on unusual ports
func (d *Detector) detectUnusualPorts(profile *database.BehavioralProfile) []*Anomaly {
	anomalies := make([]*Anomaly, 0)

	// Define common ports
	commonPorts := map[uint16]bool{
		80:   true, // HTTP
		443:  true, // HTTPS
		53:   true, // DNS
		123:  true, // NTP
		8080: true, // HTTP alternate
		8443: true, // HTTPS alternate
	}

	// Calculate total port usage
	var totalPortUsage int
	for _, count := range profile.Ports {
		totalPortUsage += count
	}

	if totalPortUsage == 0 {
		return anomalies
	}

	// Detect unusual ports with significant traffic
	for port, count := range profile.Ports {
		if !commonPorts[port] {
			percentage := float64(count) / float64(totalPortUsage)
			
			// If unusual port has more than sensitivity% of traffic, flag it
			if percentage > (d.sensitivity * 0.1) { // Scale sensitivity for port detection
				severity := d.calculatePortSeverity(port, percentage)

				anomaly := &Anomaly{
					DeviceMAC:   profile.MAC,
					Type:        AnomalyUnusualPort,
					Severity:    severity,
					Description: fmt.Sprintf("Device using unusual port %d (%.1f%% of traffic)", port, percentage*100),
					Timestamp:   time.Now(),
					Evidence: map[string]interface{}{
						"port":       port,
						"count":      count,
						"percentage": percentage * 100,
					},
				}
				anomalies = append(anomalies, anomaly)
			}
		}
	}

	return anomalies
}

// detectTrafficSpikes identifies unusual increases in traffic volume
func (d *Detector) detectTrafficSpikes(profile *database.BehavioralProfile) []*Anomaly {
	anomalies := make([]*Anomaly, 0)

	// Calculate average hourly activity
	var totalActivity uint64
	var activeHours int
	for _, activity := range profile.HourlyActivity {
		if activity > 0 {
			totalActivity += uint64(activity)
			activeHours++
		}
	}

	if activeHours == 0 {
		return anomalies
	}

	avgActivity := float64(totalActivity) / float64(activeHours)

	// Detect hours with activity significantly above average
	spikeThreshold := avgActivity * (1.0 + (2.0 * d.sensitivity))

	for hour, activity := range profile.HourlyActivity {
		if float64(activity) > spikeThreshold {
			severity := d.calculateSpikeSeverity(float64(activity), avgActivity)

			anomaly := &Anomaly{
				DeviceMAC:   profile.MAC,
				Type:        AnomalyTrafficSpike,
				Severity:    severity,
				Description: fmt.Sprintf("Traffic spike detected at hour %d (%.1fx average)", hour, float64(activity)/avgActivity),
				Timestamp:   time.Now(),
				Evidence: map[string]interface{}{
					"hour":            hour,
					"activity":        activity,
					"average":         avgActivity,
					"spike_magnitude": float64(activity) / avgActivity,
				},
			}
			anomalies = append(anomalies, anomaly)
		}
	}

	return anomalies
}

// calculateSeverity determines severity based on deviation from average
func (d *Detector) calculateSeverity(value, average float64) Severity {
	if average == 0 {
		return SeverityLow
	}

	ratio := value / average
	
	if ratio < 0.1 {
		return SeverityHigh
	} else if ratio < 0.3 {
		return SeverityMedium
	}
	return SeverityLow
}

// calculatePortSeverity determines severity based on port number and usage
func (d *Detector) calculatePortSeverity(port uint16, percentage float64) Severity {
	// High ports (>= 49152) are ephemeral and less concerning
	// Low ports (< 1024) are privileged and more concerning
	// Mid ports (1024-49151) are registered and moderately concerning

	if port < 1024 {
		// Privileged port
		if percentage > 0.5 {
			return SeverityCritical
		} else if percentage > 0.2 {
			return SeverityHigh
		}
		return SeverityMedium
	} else if port < 49152 {
		// Registered port
		if percentage > 0.7 {
			return SeverityHigh
		} else if percentage > 0.3 {
			return SeverityMedium
		}
		return SeverityLow
	} else {
		// Ephemeral port
		if percentage > 0.8 {
			return SeverityMedium
		}
		return SeverityLow
	}
}

// calculateSpikeSeverity determines severity based on spike magnitude
func (d *Detector) calculateSpikeSeverity(activity, average float64) Severity {
	if average == 0 {
		return SeverityLow
	}

	magnitude := activity / average

	if magnitude > 10.0 {
		return SeverityCritical
	} else if magnitude > 5.0 {
		return SeverityHigh
	} else if magnitude > 3.0 {
		return SeverityMedium
	}
	return SeverityLow
}

// SetSensitivity updates the detector's sensitivity
func (d *Detector) SetSensitivity(sensitivity float64) error {
	if sensitivity < 0.0 || sensitivity > 1.0 {
		return fmt.Errorf("sensitivity must be between 0.0 and 1.0, got %f", sensitivity)
	}
	d.sensitivity = sensitivity
	return nil
}

// GetSensitivity returns the current sensitivity setting
func (d *Detector) GetSensitivity() float64 {
	return d.sensitivity
}

// AnalyzeBatch analyzes multiple profiles and returns all detected anomalies
func (d *Detector) AnalyzeBatch(profiles []*database.BehavioralProfile) ([]*Anomaly, error) {
	allAnomalies := make([]*Anomaly, 0)

	for _, profile := range profiles {
		anomalies, err := d.Analyze(profile)
		if err != nil {
			// Log error but continue with other profiles
			continue
		}
		allAnomalies = append(allAnomalies, anomalies...)
	}

	return allAnomalies, nil
}

// FilterBySeverity returns only anomalies matching the specified severity levels
func FilterBySeverity(anomalies []*Anomaly, severities ...Severity) []*Anomaly {
	severityMap := make(map[Severity]bool)
	for _, s := range severities {
		severityMap[s] = true
	}

	filtered := make([]*Anomaly, 0)
	for _, anomaly := range anomalies {
		if severityMap[anomaly.Severity] {
			filtered = append(filtered, anomaly)
		}
	}

	return filtered
}

// FilterByType returns only anomalies matching the specified types
func FilterByType(anomalies []*Anomaly, types ...AnomalyType) []*Anomaly {
	typeMap := make(map[AnomalyType]bool)
	for _, t := range types {
		typeMap[t] = true
	}

	filtered := make([]*Anomaly, 0)
	for _, anomaly := range anomalies {
		if typeMap[anomaly.Type] {
			filtered = append(filtered, anomaly)
		}
	}

	return filtered
}

// Helper function to calculate standard deviation (used internally)
func calculateStdDev(values []float64, mean float64) float64 {
	if len(values) == 0 {
		return 0
	}

	var sumSquares float64
	for _, v := range values {
		diff := v - mean
		sumSquares += diff * diff
	}

	variance := sumSquares / float64(len(values))
	return math.Sqrt(variance)
}
