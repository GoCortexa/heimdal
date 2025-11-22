package visualizer

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/mosiko1234/heimdal/sensor/internal/database"
	"github.com/mosiko1234/heimdal/sensor/internal/desktop/featuregate"
)

// DeviceResponse represents the JSON response for a device
type DeviceResponse struct {
	MAC       string `json:"mac"`
	IP        string `json:"ip"`
	Name      string `json:"name"`
	Vendor    string `json:"vendor"`
	FirstSeen string `json:"first_seen"`
	LastSeen  string `json:"last_seen"`
	IsActive  bool   `json:"is_active"`
}

// ProfileResponse represents the JSON response for a behavioral profile
type ProfileResponse struct {
	MAC            string                       `json:"mac"`
	Destinations   map[string]*DestinationInfo  `json:"destinations"`
	Ports          map[string]int               `json:"ports"` // Changed to string keys for JSON
	Protocols      map[string]int               `json:"protocols"`
	TotalPackets   int64                        `json:"total_packets"`
	TotalBytes     int64                        `json:"total_bytes"`
	FirstSeen      string                       `json:"first_seen"`
	LastSeen       string                       `json:"last_seen"`
	HourlyActivity [24]int                      `json:"hourly_activity"`
}

// DestinationInfo represents destination information in the API response
type DestinationInfo struct {
	IP       string `json:"ip"`
	Count    int64  `json:"count"`
	LastSeen string `json:"last_seen"`
}

// TierInfoResponse represents the JSON response for tier information
type TierInfoResponse struct {
	Tier     string   `json:"tier"`
	Features []string `json:"features"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// HandleDevices handles GET /api/v1/devices - list all devices
func (v *Visualizer) HandleDevices(w http.ResponseWriter, r *http.Request) {
	// Only allow GET method
	if r.Method != http.MethodGet {
		v.sendError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only GET method is allowed")
		return
	}

	// Check feature gate access
	if v.featureGate != nil {
		if err := v.featureGate.CheckAccess(featuregate.FeatureNetworkVisibility); err != nil {
			v.sendError(w, http.StatusForbidden, "access_denied", err.Error())
			return
		}
	}

	// Retrieve all devices from storage
	devices, err := v.getAllDevicesFromStorage()
	if err != nil {
		log.Printf("[Visualizer] Error retrieving devices: %v", err)
		v.sendError(w, http.StatusInternalServerError, "storage_error", "Failed to retrieve devices")
		return
	}

	// Convert to response format
	response := make([]DeviceResponse, 0, len(devices))
	for _, device := range devices {
		response = append(response, v.deviceToResponse(device))
	}

	// Send JSON response
	v.sendJSON(w, http.StatusOK, response)
}

// HandleDeviceByMAC handles GET /api/v1/devices/:mac - get device details
func (v *Visualizer) HandleDeviceByMAC(w http.ResponseWriter, r *http.Request) {
	// Only allow GET method
	if r.Method != http.MethodGet {
		v.sendError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only GET method is allowed")
		return
	}

	// Check feature gate access
	if v.featureGate != nil {
		if err := v.featureGate.CheckAccess(featuregate.FeatureNetworkVisibility); err != nil {
			v.sendError(w, http.StatusForbidden, "access_denied", err.Error())
			return
		}
	}

	// Extract MAC address from URL path
	// Path format: /api/v1/devices/:mac
	mac := strings.TrimPrefix(r.URL.Path, "/api/v1/devices/")
	if mac == "" || mac == "/api/v1/devices/" {
		v.sendError(w, http.StatusBadRequest, "invalid_mac", "MAC address is required")
		return
	}

	// Retrieve device from storage
	device, err := v.getDeviceFromStorage(mac)
	if err != nil {
		log.Printf("[Visualizer] Error retrieving device %s: %v", mac, err)
		v.sendError(w, http.StatusNotFound, "device_not_found", fmt.Sprintf("Device not found: %s", mac))
		return
	}

	// Convert to response format
	response := v.deviceToResponse(device)

	// Send JSON response
	v.sendJSON(w, http.StatusOK, response)
}

// HandleProfileByMAC handles GET /api/v1/profiles/:mac - get behavioral profile
func (v *Visualizer) HandleProfileByMAC(w http.ResponseWriter, r *http.Request) {
	// Only allow GET method
	if r.Method != http.MethodGet {
		v.sendError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only GET method is allowed")
		return
	}

	// Check feature gate access
	if v.featureGate != nil {
		if err := v.featureGate.CheckAccess(featuregate.FeatureNetworkVisibility); err != nil {
			v.sendError(w, http.StatusForbidden, "access_denied", err.Error())
			return
		}
	}

	// Extract MAC address from URL path
	// Path format: /api/v1/profiles/:mac
	mac := strings.TrimPrefix(r.URL.Path, "/api/v1/profiles/")
	if mac == "" || mac == "/api/v1/profiles/" {
		v.sendError(w, http.StatusBadRequest, "invalid_mac", "MAC address is required")
		return
	}

	// Retrieve profile from storage
	profile, err := v.getProfileFromStorage(mac)
	if err != nil {
		log.Printf("[Visualizer] Error retrieving profile %s: %v", mac, err)
		v.sendError(w, http.StatusNotFound, "profile_not_found", fmt.Sprintf("Profile not found: %s", mac))
		return
	}

	// Convert to response format
	response := v.profileToResponse(profile)

	// Send JSON response
	v.sendJSON(w, http.StatusOK, response)
}

// HandleTierInfo handles GET /api/v1/tier - get tier information
func (v *Visualizer) HandleTierInfo(w http.ResponseWriter, r *http.Request) {
	// Only allow GET method
	if r.Method != http.MethodGet {
		v.sendError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only GET method is allowed")
		return
	}

	var response TierInfoResponse

	if v.featureGate != nil {
		tier := v.featureGate.GetTier()
		response.Tier = string(tier)
		
		// List available features for this tier
		features := []string{}
		allFeatures := []featuregate.Feature{
			featuregate.FeatureNetworkVisibility,
			featuregate.FeatureTrafficBlocking,
			featuregate.FeatureAdvancedFiltering,
			featuregate.FeatureCloudSync,
			featuregate.FeatureMultiDevice,
			featuregate.FeatureAPIAccess,
		}
		
		for _, feature := range allFeatures {
			if v.featureGate.CanAccess(feature) {
				features = append(features, string(feature))
			}
		}
		
		response.Features = features
	} else {
		// No feature gate configured, assume all features available
		response.Tier = "enterprise"
		response.Features = []string{
			"network_visibility",
			"traffic_blocking",
			"advanced_filtering",
			"cloud_sync",
			"multi_device",
			"api_access",
		}
	}

	// Send JSON response
	v.sendJSON(w, http.StatusOK, response)
}

// getAllDevicesFromStorage retrieves all devices from the storage provider
func (v *Visualizer) getAllDevicesFromStorage() ([]*database.Device, error) {
	// List all device keys
	keys, err := v.storage.List("device:")
	if err != nil {
		return nil, fmt.Errorf("failed to list devices: %w", err)
	}

	devices := make([]*database.Device, 0, len(keys))
	for _, key := range keys {
		// Get device data
		data, err := v.storage.Get(key)
		if err != nil {
			log.Printf("[Visualizer] Warning: failed to get device %s: %v", key, err)
			continue
		}

		// Deserialize device
		var device database.Device
		if err := json.Unmarshal(data, &device); err != nil {
			log.Printf("[Visualizer] Warning: failed to unmarshal device %s: %v", key, err)
			continue
		}

		devices = append(devices, &device)
	}

	return devices, nil
}

// getDeviceFromStorage retrieves a single device from the storage provider
func (v *Visualizer) getDeviceFromStorage(mac string) (*database.Device, error) {
	key := "device:" + mac
	
	// Get device data
	data, err := v.storage.Get(key)
	if err != nil {
		return nil, fmt.Errorf("device not found: %w", err)
	}

	// Deserialize device
	var device database.Device
	if err := json.Unmarshal(data, &device); err != nil {
		return nil, fmt.Errorf("failed to unmarshal device: %w", err)
	}

	return &device, nil
}

// getProfileFromStorage retrieves a behavioral profile from the storage provider
func (v *Visualizer) getProfileFromStorage(mac string) (*database.BehavioralProfile, error) {
	key := "profile:" + mac
	
	// Get profile data
	data, err := v.storage.Get(key)
	if err != nil {
		return nil, fmt.Errorf("profile not found: %w", err)
	}

	// Deserialize profile
	var profile database.BehavioralProfile
	if err := json.Unmarshal(data, &profile); err != nil {
		return nil, fmt.Errorf("failed to unmarshal profile: %w", err)
	}

	return &profile, nil
}

// deviceToResponse converts a database.Device to a DeviceResponse
func (v *Visualizer) deviceToResponse(device *database.Device) DeviceResponse {
	return DeviceResponse{
		MAC:       device.MAC,
		IP:        device.IP,
		Name:      device.Name,
		Vendor:    device.Vendor,
		FirstSeen: device.FirstSeen.Format("2006-01-02T15:04:05Z07:00"),
		LastSeen:  device.LastSeen.Format("2006-01-02T15:04:05Z07:00"),
		IsActive:  device.IsActive,
	}
}

// profileToResponse converts a database.BehavioralProfile to a ProfileResponse
func (v *Visualizer) profileToResponse(profile *database.BehavioralProfile) ProfileResponse {
	// Convert destinations
	destinations := make(map[string]*DestinationInfo)
	for ip, destInfo := range profile.Destinations {
		destinations[ip] = &DestinationInfo{
			IP:       destInfo.IP,
			Count:    destInfo.Count,
			LastSeen: destInfo.LastSeen.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	// Convert ports map (uint16 keys to string keys for JSON)
	ports := make(map[string]int)
	for port, count := range profile.Ports {
		ports[fmt.Sprintf("%d", port)] = count
	}

	return ProfileResponse{
		MAC:            profile.MAC,
		Destinations:   destinations,
		Ports:          ports,
		Protocols:      profile.Protocols,
		TotalPackets:   profile.TotalPackets,
		TotalBytes:     profile.TotalBytes,
		FirstSeen:      profile.FirstSeen.Format("2006-01-02T15:04:05Z07:00"),
		LastSeen:       profile.LastSeen.Format("2006-01-02T15:04:05Z07:00"),
		HourlyActivity: profile.HourlyActivity,
	}
}

// sendJSON sends a JSON response
func (v *Visualizer) sendJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("[Visualizer] Error encoding JSON response: %v", err)
	}
}

// sendError sends an error response
func (v *Visualizer) sendError(w http.ResponseWriter, statusCode int, errorCode string, message string) {
	response := ErrorResponse{
		Error:   errorCode,
		Message: message,
	}
	v.sendJSON(w, statusCode, response)
}
