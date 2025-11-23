package hostname

import (
	"testing"
	"time"
)

func TestReverseDNS(t *testing.T) {
	resolver := NewResolver(2 * time.Second)

	// Test with a well-known DNS server (should have PTR record)
	hostname, err := resolver.reverseDNS("8.8.8.8")
	if err != nil {
		t.Logf("Reverse DNS for 8.8.8.8 failed (expected on some networks): %v", err)
	} else {
		t.Logf("Reverse DNS for 8.8.8.8: %s", hostname)
		if hostname == "" {
			t.Error("Expected non-empty hostname")
		}
	}
}

func TestResolve(t *testing.T) {
	resolver := NewResolver(2 * time.Second)

	tests := []struct {
		name     string
		ip       string
		mdnsName string
		wantErr  bool
	}{
		{
			name:     "mDNS name provided",
			ip:       "192.168.1.100",
			mdnsName: "my-device.local",
			wantErr:  false,
		},
		{
			name:     "no mDNS, try DNS",
			ip:       "8.8.8.8",
			mdnsName: "",
			wantErr:  false, // May fail on some networks, but shouldn't error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hostname, method, err := resolver.Resolve(tt.ip, tt.mdnsName)

			if tt.mdnsName != "" {
				if hostname != tt.mdnsName {
					t.Errorf("Expected hostname %s, got %s", tt.mdnsName, hostname)
				}
				if method != "mdns" {
					t.Errorf("Expected method 'mdns', got %s", method)
				}
			}

			t.Logf("Resolve(%s, %s) = hostname:%s, method:%s, err:%v",
				tt.ip, tt.mdnsName, hostname, method, err)
		})
	}
}

func TestResolveAsync(t *testing.T) {
	resolver := NewResolver(1 * time.Second)

	resultCh := resolver.ResolveAsync("8.8.8.8", "")

	select {
	case result := <-resultCh:
		t.Logf("Async resolve result: hostname=%s, method=%s, err=%v",
			result.Hostname, result.Method, result.Error)
	case <-time.After(3 * time.Second):
		t.Error("Async resolve timeout")
	}
}

func TestBulkResolve(t *testing.T) {
	resolver := NewResolver(2 * time.Second)

	ips := []string{"8.8.8.8", "1.1.1.1"}
	mdnsNames := map[string]string{
		"1.1.1.1": "cloudflare-dns.local",
	}

	results := resolver.BulkResolve(ips, mdnsNames)

	if len(results) == 0 {
		t.Error("Expected at least one result")
	}

	for ip, result := range results {
		t.Logf("BulkResolve[%s]: hostname=%s, method=%s, err=%v",
			ip, result.Hostname, result.Method, result.Error)
	}
}
