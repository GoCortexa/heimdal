package schemas

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/mosiko1234/heimdal/sensor/internal/database"
)

// DeviceToMessage converts a database.Device to a DeviceMessage
func DeviceToMessage(device *database.Device, sensorID, networkID, gateway string) *DeviceMessage {
	if device == nil {
		return nil
	}

	return &DeviceMessage{
		MAC:          device.MAC,
		IP:           device.IP,
		Name:         device.Name,
		Vendor:       device.Vendor,
		Manufacturer: device.Manufacturer,
		DeviceType:   device.DeviceType,
		Hostname:     device.Hostname,
		Services:     device.Services,
		FirstSeen:    device.FirstSeen,
		LastSeen:     device.LastSeen,
		IsActive:     device.IsActive,
		NetworkID:    networkID,
		Gateway:      gateway,
	}
}

// ProfileToMessage converts a database.BehavioralProfile to a ProfileMessage
func ProfileToMessage(profile *database.BehavioralProfile) *ProfileMessage {
	if profile == nil {
		return nil
	}

	msg := &ProfileMessage{
		MAC:                profile.MAC,
		TotalPackets:       profile.TotalPackets,
		TotalBytes:         profile.TotalBytes,
		FirstSeen:          profile.FirstSeen,
		LastSeen:           profile.LastSeen,
		UniqueDestinations: len(profile.Destinations),
		UniquePorts:        len(profile.Ports),
		HourlyActivity:     profile.HourlyActivity,
	}

	// Calculate protocol distribution percentages
	msg.ProtocolDistribution = make(map[string]float64)
	totalProtocolPackets := int64(0)
	for _, count := range profile.Protocols {
		totalProtocolPackets += int64(count)
	}
	if totalProtocolPackets > 0 {
		for protocol, count := range profile.Protocols {
			msg.ProtocolDistribution[protocol] = float64(count) / float64(totalProtocolPackets)
		}
	}

	// Extract top 20 destinations
	msg.TopDestinations = getTopDestinations(profile.Destinations, 20)

	// Extract top 10 ports
	msg.TopPorts = getTopPorts(profile.Ports, 10)

	// Include baseline if available
	if profile.Baseline != nil {
		msg.Baseline = &BaselineMetrics{
			AvgPacketsPerHour:     profile.Baseline.AvgPacketsPerHour,
			StdDevPacketsPerHour:  profile.Baseline.StdDevPacketsPerHour,
			AvgPacketsPerDay:      profile.Baseline.AvgPacketsPerDay,
			StdDevPacketsPerDay:   profile.Baseline.StdDevPacketsPerDay,
			AvgUniqueDestinations: profile.Baseline.AvgUniqueDestinations,
			StdDevDestinations:    profile.Baseline.StdDevDestinations,
			ProtocolDistribution:  profile.Baseline.ProtocolDistribution,
			LastCalculated:        profile.Baseline.LastCalculated,
			SampleCount:           profile.Baseline.SampleCount,
		}
	}

	return msg
}

// getTopDestinations returns the top N destinations by packet count
func getTopDestinations(destinations map[string]*database.DestInfo, limit int) []DestinationSummary {
	// Convert to slice
	destSlice := make([]DestinationSummary, 0, len(destinations))
	for ip, info := range destinations {
		destSlice = append(destSlice, DestinationSummary{
			IP:       ip,
			Count:    info.Count,
			LastSeen: info.LastSeen,
		})
	}

	// Sort by count (descending)
	sort.Slice(destSlice, func(i, j int) bool {
		return destSlice[i].Count > destSlice[j].Count
	})

	// Return top N
	if len(destSlice) > limit {
		destSlice = destSlice[:limit]
	}

	return destSlice
}

// getTopPorts returns the top N ports by usage count
func getTopPorts(ports map[uint16]int, limit int) []PortSummary {
	// Calculate total
	totalCount := 0
	for _, count := range ports {
		totalCount += count
	}

	if totalCount == 0 {
		return nil
	}

	// Convert to slice
	portSlice := make([]PortSummary, 0, len(ports))
	for port, count := range ports {
		portSlice = append(portSlice, PortSummary{
			Port:       port,
			Count:      count,
			Percentage: float64(count) / float64(totalCount),
		})
	}

	// Sort by count (descending)
	sort.Slice(portSlice, func(i, j int) bool {
		return portSlice[i].Count > portSlice[j].Count
	})

	// Return top N
	if len(portSlice) > limit {
		portSlice = portSlice[:limit]
	}

	return portSlice
}

// WrapMessage wraps a payload in a CloudMessage envelope
func WrapMessage(messageType MessageType, sensorID string, payload interface{}) (*CloudMessage, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	return &CloudMessage{
		MessageType: messageType,
		SensorID:    sensorID,
		Timestamp:   time.Now(),
		Version:     "1.0",
		Payload:     payloadBytes,
	}, nil
}

// UnwrapMessage extracts the payload from a CloudMessage
func UnwrapMessage(msg *CloudMessage, payload interface{}) error {
	if msg == nil {
		return fmt.Errorf("message is nil")
	}

	if err := json.Unmarshal(msg.Payload, payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	return nil
}

// SerializeMessage converts a CloudMessage to JSON bytes
func SerializeMessage(msg *CloudMessage) ([]byte, error) {
	if msg == nil {
		return nil, fmt.Errorf("message is nil")
	}

	return json.Marshal(msg)
}

// DeserializeMessage parses JSON bytes into a CloudMessage
func DeserializeMessage(data []byte) (*CloudMessage, error) {
	var msg CloudMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal message: %w", err)
	}

	return &msg, nil
}
