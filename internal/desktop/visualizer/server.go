// Package visualizer provides the local web dashboard for the Heimdal Desktop product.
//
// The LocalVisualizer serves a web-based interface for network visualization,
// device management, and real-time traffic monitoring. It provides both HTTP
// endpoints for the dashboard UI and REST API endpoints for device data.
package visualizer

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/mosiko1234/heimdal/sensor/internal/desktop/featuregate"
	"github.com/mosiko1234/heimdal/sensor/internal/platform"
)

// Visualizer serves the local web dashboard
type Visualizer struct {
	server      *http.Server
	storage     platform.StorageProvider
	featureGate *featuregate.FeatureGate
	wsHub       *WebSocketHub
	port        int
	mu          sync.RWMutex
	running     bool
}

// Config contains configuration for the visualizer
type Config struct {
	Port        int
	Storage     platform.StorageProvider
	FeatureGate *featuregate.FeatureGate
}

// NewVisualizer creates a new LocalVisualizer instance
func NewVisualizer(cfg *Config) (*Visualizer, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	if cfg.Storage == nil {
		return nil, fmt.Errorf("storage provider is required")
	}
	if cfg.Port <= 0 || cfg.Port > 65535 {
		return nil, fmt.Errorf("invalid port: %d", cfg.Port)
	}

	// Create WebSocket hub for real-time updates
	wsHub := NewWebSocketHub()

	v := &Visualizer{
		storage:     cfg.Storage,
		featureGate: cfg.FeatureGate,
		wsHub:       wsHub,
		port:        cfg.Port,
		running:     false,
	}

	// Create HTTP server with configured routes
	mux := http.NewServeMux()
	v.setupRoutes(mux)

	v.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return v, nil
}

// setupRoutes configures HTTP routes for the visualizer
func (v *Visualizer) setupRoutes(mux *http.ServeMux) {
	// API endpoints
	mux.HandleFunc("/api/v1/devices", v.HandleDevices)
	mux.HandleFunc("/api/v1/devices/", v.HandleDeviceByMAC)
	mux.HandleFunc("/api/v1/profiles/", v.HandleProfileByMAC)
	mux.HandleFunc("/api/v1/tier", v.HandleTierInfo)
	
	// WebSocket endpoint for real-time updates
	mux.HandleFunc("/ws", v.handleWebSocket)
	
	// Static file serving for dashboard UI
	// In production, this would serve from embedded files or a static directory
	mux.HandleFunc("/", v.handleDashboard)
}

// Start begins serving the dashboard
func (v *Visualizer) Start() error {
	v.mu.Lock()
	if v.running {
		v.mu.Unlock()
		return fmt.Errorf("visualizer is already running")
	}
	v.running = true
	v.mu.Unlock()

	log.Printf("[Visualizer] Starting HTTP server on port %d...", v.port)

	// Start WebSocket hub
	go v.wsHub.Run()

	// Start HTTP server in a goroutine
	go func() {
		if err := v.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[Visualizer] HTTP server error: %v", err)
		}
	}()

	log.Printf("[Visualizer] Dashboard available at http://localhost:%d", v.port)
	return nil
}

// Stop gracefully shuts down the visualizer
func (v *Visualizer) Stop() error {
	v.mu.Lock()
	if !v.running {
		v.mu.Unlock()
		return nil
	}
	v.running = false
	v.mu.Unlock()

	log.Println("[Visualizer] Stopping HTTP server...")

	// Stop WebSocket hub
	v.wsHub.Stop()

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Gracefully shutdown HTTP server
	if err := v.server.Shutdown(ctx); err != nil {
		log.Printf("[Visualizer] Error during shutdown: %v", err)
		return fmt.Errorf("failed to shutdown server: %w", err)
	}

	log.Println("[Visualizer] Stopped successfully")
	return nil
}

// IsRunning returns whether the visualizer is currently running
func (v *Visualizer) IsRunning() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.running
}

// GetPort returns the port the visualizer is configured to use
func (v *Visualizer) GetPort() int {
	return v.port
}

// BroadcastUpdate sends a real-time update to all connected WebSocket clients
func (v *Visualizer) BroadcastUpdate(updateType string, payload interface{}) {
	v.wsHub.Broadcast(&UpdateMessage{
		Type:    updateType,
		Payload: payload,
	})
}

// handleDashboard serves the main dashboard HTML page and static files
func (v *Visualizer) handleDashboard(w http.ResponseWriter, r *http.Request) {
	// Serve static files from web/dashboard directory
	if r.URL.Path == "/" {
		http.ServeFile(w, r, "web/dashboard/index.html")
		return
	}
	
	// Serve other static files (CSS, JS)
	if r.URL.Path == "/styles.css" {
		w.Header().Set("Content-Type", "text/css")
		http.ServeFile(w, r, "web/dashboard/styles.css")
		return
	}
	
	if r.URL.Path == "/app.js" {
		w.Header().Set("Content-Type", "application/javascript")
		http.ServeFile(w, r, "web/dashboard/app.js")
		return
	}
	
	// 404 for other paths
	http.NotFound(w, r)
}
