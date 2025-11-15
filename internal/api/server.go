// Package api provides the REST API and web dashboard for the Heimdal sensor.
//
// The APIServer component serves both a REST API for programmatic access and a web-based
// dashboard for monitoring discovered devices and their behavioral profiles. It uses the
// gorilla/mux router for HTTP routing and implements per-IP rate limiting for security.
//
// API Endpoints:
//   GET  /api/v1/devices              → List all discovered devices
//   GET  /api/v1/devices/:mac         → Get device details by MAC address
//   GET  /api/v1/profiles/:mac        → Get behavioral profile by MAC address
//   GET  /api/v1/stats                → System statistics (uptime, device counts, etc.)
//   GET  /api/v1/health               → Health check endpoint
//   GET  /                            → Dashboard HTML (static files)
//
// Dashboard Features:
//   - Device list table with MAC, IP, Name, and Status
//   - Click device to view detailed behavioral profile
//   - Visual representation of top destinations
//   - Port usage chart
//   - Activity timeline (24-hour)
//   - Auto-refresh every 10 seconds
//
// Security:
//   - Rate limiting: configurable requests per minute per IP (default: 100/min)
//   - Input validation on all endpoints
//   - CORS enabled for local network access
//   - No authentication (local network trust model)
//
// The server listens on a configurable host and port (default: 0.0.0.0:8080) and
// implements graceful shutdown via context cancellation.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/mosiko1234/heimdal/sensor/internal/database"
	"golang.org/x/time/rate"
)

// APIServer provides HTTP API and dashboard for Heimdal sensor
type APIServer struct {
	db          *database.DatabaseManager
	router      *mux.Router
	server      *http.Server
	port        int
	host        string
	rateLimiter *rateLimiterMiddleware
	startTime   time.Time
	mu          sync.RWMutex
}

// rateLimiterMiddleware implements per-IP rate limiting
type rateLimiterMiddleware struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	rate     int // requests per minute
}

// NewAPIServer creates a new API server instance
func NewAPIServer(db *database.DatabaseManager, host string, port int, rateLimit int) *APIServer {
	router := mux.NewRouter()
	
	server := &APIServer{
		db:     db,
		router: router,
		port:   port,
		host:   host,
		rateLimiter: &rateLimiterMiddleware{
			limiters: make(map[string]*rate.Limiter),
			rate:     rateLimit,
		},
		startTime: time.Now(),
	}

	// Configure routes
	server.setupRoutes()

	// Create HTTP server
	server.server = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", host, port),
		Handler:      server.router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return server
}

// setupRoutes configures all API routes and middleware
func (s *APIServer) setupRoutes() {
	// Apply middleware
	s.router.Use(s.corsMiddleware)
	s.router.Use(s.rateLimiter.middleware)
	s.router.Use(s.loggingMiddleware)

	// API routes
	api := s.router.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/devices", s.handleGetDevices).Methods("GET")
	api.HandleFunc("/devices/{mac}", s.handleGetDevice).Methods("GET")
	api.HandleFunc("/profiles/{mac}", s.handleGetProfile).Methods("GET")
	api.HandleFunc("/stats", s.handleGetStats).Methods("GET")
	api.HandleFunc("/health", s.handleGetHealth).Methods("GET")

	// Static file serving for dashboard
	s.router.PathPrefix("/").Handler(http.FileServer(http.Dir("web/dashboard")))
}

// corsMiddleware adds CORS headers for local network access
func (s *APIServer) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// loggingMiddleware logs all HTTP requests
func (s *APIServer) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("API: %s %s - %v", r.Method, r.URL.Path, time.Since(start))
	})
}

// middleware implements rate limiting per IP address
func (rl *rateLimiterMiddleware) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract IP address
		ip := r.RemoteAddr

		// Get or create limiter for this IP
		rl.mu.Lock()
		limiter, exists := rl.limiters[ip]
		if !exists {
			// Create new limiter: rate per minute converted to per second
			limiter = rate.NewLimiter(rate.Limit(float64(rl.rate)/60.0), rl.rate)
			rl.limiters[ip] = limiter
		}
		rl.mu.Unlock()

		// Check if request is allowed
		if !limiter.Allow() {
			respondError(w, http.StatusTooManyRequests, "rate limit exceeded")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Start begins serving HTTP requests
func (s *APIServer) Start(ctx context.Context) error {
	log.Printf("API: Starting server on %s:%d", s.host, s.port)

	// Start server in goroutine
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("API: Server error: %v", err)
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()
	return s.Stop()
}

// Stop gracefully shuts down the API server
func (s *APIServer) Stop() error {
	log.Println("API: Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown API server: %w", err)
	}

	log.Println("API: Server stopped")
	return nil
}

// Name returns the component name
func (s *APIServer) Name() string {
	return "APIServer"
}

// respondJSON sends a JSON response
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("API: Failed to encode JSON response: %v", err)
	}
}

// respondError sends an error response
func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{
		"error": message,
	})
}

// DeviceResponse represents the response for device list endpoint
type DeviceResponse struct {
	Devices []*database.Device `json:"devices"`
	Count   int                `json:"count"`
}

// StatsResponse represents system statistics
type StatsResponse struct {
	TotalDevices  int       `json:"total_devices"`
	ActiveDevices int       `json:"active_devices"`
	TotalPackets  int64     `json:"total_packets"`
	Uptime        string    `json:"uptime"`
	LastUpdate    time.Time `json:"last_update"`
}

// HealthResponse represents health check status
type HealthResponse struct {
	Status     string    `json:"status"`
	Uptime     string    `json:"uptime"`
	Database   string    `json:"database"`
	Timestamp  time.Time `json:"timestamp"`
}

// handleGetDevices returns a list of all discovered devices
func (s *APIServer) handleGetDevices(w http.ResponseWriter, r *http.Request) {
	devices, err := s.db.GetAllDevices()
	if err != nil {
		log.Printf("API: Failed to get devices: %v", err)
		respondError(w, http.StatusInternalServerError, "failed to retrieve devices")
		return
	}

	response := DeviceResponse{
		Devices: devices,
		Count:   len(devices),
	}

	respondJSON(w, http.StatusOK, response)
}

// handleGetDevice returns details for a specific device by MAC address
func (s *APIServer) handleGetDevice(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	mac := vars["mac"]

	if mac == "" {
		respondError(w, http.StatusBadRequest, "MAC address is required")
		return
	}

	device, err := s.db.GetDevice(mac)
	if err != nil {
		log.Printf("API: Failed to get device %s: %v", mac, err)
		respondError(w, http.StatusNotFound, "device not found")
		return
	}

	respondJSON(w, http.StatusOK, device)
}

// handleGetProfile returns the behavioral profile for a specific device
func (s *APIServer) handleGetProfile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	mac := vars["mac"]

	if mac == "" {
		respondError(w, http.StatusBadRequest, "MAC address is required")
		return
	}

	profile, err := s.db.GetProfile(mac)
	if err != nil {
		log.Printf("API: Failed to get profile %s: %v", mac, err)
		respondError(w, http.StatusNotFound, "profile not found")
		return
	}

	respondJSON(w, http.StatusOK, profile)
}

// handleGetStats returns system statistics
func (s *APIServer) handleGetStats(w http.ResponseWriter, r *http.Request) {
	devices, err := s.db.GetAllDevices()
	if err != nil {
		log.Printf("API: Failed to get devices for stats: %v", err)
		respondError(w, http.StatusInternalServerError, "failed to retrieve statistics")
		return
	}

	// Count active devices
	activeCount := 0
	for _, device := range devices {
		if device.IsActive {
			activeCount++
		}
	}

	// Calculate total packets from all profiles
	var totalPackets int64
	profiles, err := s.db.GetAllProfiles()
	if err == nil {
		for _, profile := range profiles {
			totalPackets += profile.TotalPackets
		}
	}

	// Calculate uptime
	uptime := time.Since(s.startTime)
	uptimeStr := fmt.Sprintf("%dd %dh %dm %ds",
		int(uptime.Hours())/24,
		int(uptime.Hours())%24,
		int(uptime.Minutes())%60,
		int(uptime.Seconds())%60,
	)

	response := StatsResponse{
		TotalDevices:  len(devices),
		ActiveDevices: activeCount,
		TotalPackets:  totalPackets,
		Uptime:        uptimeStr,
		LastUpdate:    time.Now(),
	}

	respondJSON(w, http.StatusOK, response)
}

// handleGetHealth returns health check status
func (s *APIServer) handleGetHealth(w http.ResponseWriter, r *http.Request) {
	// Check database connectivity by attempting to get devices
	dbStatus := "healthy"
	_, err := s.db.GetAllDevices()
	if err != nil {
		dbStatus = "unhealthy"
	}

	// Calculate uptime
	uptime := time.Since(s.startTime)
	uptimeStr := fmt.Sprintf("%dd %dh %dm %ds",
		int(uptime.Hours())/24,
		int(uptime.Hours())%24,
		int(uptime.Minutes())%60,
		int(uptime.Seconds())%60,
	)

	response := HealthResponse{
		Status:    "ok",
		Uptime:    uptimeStr,
		Database:  dbStatus,
		Timestamp: time.Now(),
	}

	respondJSON(w, http.StatusOK, response)
}
