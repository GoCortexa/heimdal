// Package hostname provides hostname resolution for discovered devices
// using multiple methods: reverse DNS, NetBIOS, and mDNS cache.
package hostname

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"
)

// Resolver orchestrates multiple hostname lookup methods
type Resolver struct {
	timeout time.Duration
}

// NewResolver creates a new hostname resolver
func NewResolver(timeout time.Duration) *Resolver {
	if timeout == 0 {
		timeout = 2 * time.Second
	}
	return &Resolver{
		timeout: timeout,
	}
}

// Resolve attempts to resolve a hostname for an IP address
// It tries multiple methods in order: mDNS cache, reverse DNS, NetBIOS
// Returns the first successful result
func (r *Resolver) Resolve(ip string, mdnsName string) (string, string, error) {
	// Method 1: Use mDNS name if available (highest priority)
	if mdnsName != "" {
		return mdnsName, "mdns", nil
	}

	// Method 2: Try reverse DNS lookup
	if hostname, err := r.reverseDNS(ip); err == nil && hostname != "" {
		return hostname, "reverse_dns", nil
	}

	// Method 3: Try NetBIOS (Windows networks)
	if hostname, err := r.netBIOS(ip); err == nil && hostname != "" {
		return hostname, "netbios", nil
	}

	return "", "", fmt.Errorf("no hostname resolution method succeeded")
}

// ResolveAsync resolves hostname asynchronously with timeout
func (r *Resolver) ResolveAsync(ip string, mdnsName string) <-chan ResolveResult {
	resultCh := make(chan ResolveResult, 1)

	go func() {
		defer close(resultCh)

		ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
		defer cancel()

		// Use a channel to receive result from goroutine
		doneCh := make(chan ResolveResult, 1)

		go func() {
			hostname, method, err := r.Resolve(ip, mdnsName)
			doneCh <- ResolveResult{
				Hostname: hostname,
				Method:   method,
				Error:    err,
			}
		}()

		// Wait for result or timeout
		select {
		case result := <-doneCh:
			resultCh <- result
		case <-ctx.Done():
			resultCh <- ResolveResult{
				Hostname: "",
				Method:   "",
				Error:    fmt.Errorf("hostname resolution timeout"),
			}
		}
	}()

	return resultCh
}

// ResolveResult contains the result of a hostname resolution
type ResolveResult struct {
	Hostname string
	Method   string // "mdns", "reverse_dns", "netbios"
	Error    error
}

// reverseDNS performs reverse DNS lookup (PTR record)
func (r *Resolver) reverseDNS(ip string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()

	resolver := &net.Resolver{}
	names, err := resolver.LookupAddr(ctx, ip)
	if err != nil {
		return "", fmt.Errorf("reverse DNS lookup failed: %w", err)
	}

	if len(names) == 0 {
		return "", fmt.Errorf("no PTR records found")
	}

	// Return first hostname, removing trailing dot
	hostname := strings.TrimSuffix(names[0], ".")
	return hostname, nil
}

// netBIOS performs NetBIOS name query (Windows networks)
// This is a simplified implementation - full NetBIOS requires UDP port 137
func (r *Resolver) netBIOS(ip string) (string, error) {
	// NetBIOS name resolution requires sending UDP packets to port 137
	// For now, this is a placeholder that could be implemented with:
	// 1. Send NetBIOS Name Query Request to <ip>:137
	// 2. Parse NetBIOS Name Query Response
	// 3. Extract computer name from response

	// This would require implementing the NetBIOS protocol
	// Libraries like github.com/stacktic/smb could help

	return "", fmt.Errorf("NetBIOS resolution not yet implemented")
}

// BulkResolve resolves hostnames for multiple IPs concurrently
func (r *Resolver) BulkResolve(ips []string, mdnsNames map[string]string) map[string]ResolveResult {
	results := make(map[string]ResolveResult)
	resultCh := make(chan struct {
		ip     string
		result ResolveResult
	}, len(ips))

	// Launch concurrent resolutions
	for _, ip := range ips {
		go func(ipAddr string) {
			mdnsName := mdnsNames[ipAddr]
			hostname, method, err := r.Resolve(ipAddr, mdnsName)
			resultCh <- struct {
				ip     string
				result ResolveResult
			}{
				ip: ipAddr,
				result: ResolveResult{
					Hostname: hostname,
					Method:   method,
					Error:    err,
				},
			}
		}(ip)
	}

	// Collect results with timeout
	timeout := time.After(r.timeout * 2)
	for i := 0; i < len(ips); i++ {
		select {
		case result := <-resultCh:
			results[result.ip] = result.result
		case <-timeout:
			// Timeout reached, return what we have
			return results
		}
	}

	return results
}
