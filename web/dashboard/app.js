// API base URL - adjust if needed
const API_BASE = '/api/v1';

// Auto-refresh interval (10 seconds)
const REFRESH_INTERVAL = 10000;

// Global state
let refreshTimer = null;
let currentDeviceMAC = null;

// WebSocket connection
let ws = null;

// Initialize dashboard on page load
document.addEventListener('DOMContentLoaded', () => {
    console.log('Heimdal Dashboard initialized');
    loadDashboard();
    startAutoRefresh();
    connectWebSocket();
});

// Start auto-refresh
function startAutoRefresh() {
    if (refreshTimer) {
        clearInterval(refreshTimer);
    }
    
    refreshTimer = setInterval(() => {
        loadDashboard();
        if (currentDeviceMAC) {
            loadProfile(currentDeviceMAC);
        }
    }, REFRESH_INTERVAL);
}

// Load all dashboard data
async function loadDashboard() {
    try {
        // Show refresh indicator
        const indicator = document.getElementById('refreshIndicator');
        indicator.classList.add('active');

        // Load stats and devices in parallel
        await Promise.all([
            loadStats(),
            loadDevices()
        ]);

        // Update last update time
        document.getElementById('lastUpdate').textContent = new Date().toLocaleString();

        // Hide refresh indicator
        setTimeout(() => {
            indicator.classList.remove('active');
        }, 500);
    } catch (error) {
        console.error('Failed to load dashboard:', error);
    }
}

// Load system statistics
async function loadStats() {
    try {
        const response = await fetch(`${API_BASE}/stats`);
        if (!response.ok) throw new Error('Failed to fetch stats');
        
        const stats = await response.json();
        
        document.getElementById('totalDevices').textContent = stats.total_devices || 0;
        document.getElementById('activeDevices').textContent = stats.active_devices || 0;
        document.getElementById('totalPackets').textContent = formatNumber(stats.total_packets || 0);
        document.getElementById('uptime').textContent = stats.uptime || '-';
    } catch (error) {
        console.error('Failed to load stats:', error);
    }
}

// Load devices list
async function loadDevices() {
    try {
        const response = await fetch(`${API_BASE}/devices`);
        if (!response.ok) throw new Error('Failed to fetch devices');
        
        const devices = await response.json();
        
        const tbody = document.getElementById('devicesTableBody');
        
        if (devices.length === 0) {
            tbody.innerHTML = '<tr><td colspan="7" class="empty">No devices discovered yet</td></tr>';
            return;
        }

        // Sort devices: active first, then by last seen
        devices.sort((a, b) => {
            if (a.is_active !== b.is_active) {
                return b.is_active - a.is_active;
            }
            return new Date(b.last_seen) - new Date(a.last_seen);
        });

        tbody.innerHTML = devices.map(device => `
            <tr class="${device.is_active ? 'active' : 'inactive'}">
                <td>
                    <span class="status-indicator ${device.is_active ? 'active' : 'inactive'}">
                        ${device.is_active ? '●' : '○'}
                    </span>
                </td>
                <td class="mono">${escapeHtml(device.mac)}</td>
                <td class="mono">${escapeHtml(device.ip)}</td>
                <td>${escapeHtml(device.name || 'Unknown')}</td>
                <td>${escapeHtml(device.vendor || '-')}</td>
                <td>${formatTimestamp(device.last_seen)}</td>
                <td>
                    <button class="btn-view" onclick="viewProfile('${escapeHtml(device.mac)}')">
                        View Profile
                    </button>
                </td>
            </tr>
        `).join('');
        
        // Update stats
        const totalDevices = devices.length;
        const activeDevices = devices.filter(d => d.is_active).length;
        document.getElementById('totalDevices').textContent = totalDevices;
        document.getElementById('activeDevices').textContent = activeDevices;
    } catch (error) {
        console.error('Failed to load devices:', error);
        const tbody = document.getElementById('devicesTableBody');
        tbody.innerHTML = '<tr><td colspan="7" class="error">Failed to load devices</td></tr>';
    }
}

// View device profile
async function viewProfile(mac) {
    currentDeviceMAC = mac;
    await loadProfile(mac);
    
    // Show profile section
    const profileSection = document.getElementById('profileSection');
    profileSection.style.display = 'block';
    profileSection.scrollIntoView({ behavior: 'smooth' });
}

// Load behavioral profile
async function loadProfile(mac) {
    try {
        const response = await fetch(`${API_BASE}/profiles/${encodeURIComponent(mac)}`);
        if (!response.ok) {
            if (response.status === 404) {
                showProfileError('No behavioral profile available yet');
                return;
            }
            throw new Error('Failed to fetch profile');
        }
        
        const profile = await response.json();
        displayProfile(mac, profile);
    } catch (error) {
        console.error('Failed to load profile:', error);
        showProfileError('Failed to load profile');
    }
}

// Display profile data
function displayProfile(mac, profile) {
    // Update header
    document.getElementById('profileDeviceMAC').textContent = `MAC: ${mac}`;
    
    // Update traffic summary
    document.getElementById('profilePackets').textContent = formatNumber(profile.total_packets || 0);
    document.getElementById('profileBytes').textContent = formatBytes(profile.total_bytes || 0);
    document.getElementById('profileFirstSeen').textContent = formatTimestamp(profile.first_seen);
    document.getElementById('profileLastSeen').textContent = formatTimestamp(profile.last_seen);

    // Display top destinations
    displayDestinations(profile.destinations || {});
    
    // Display top ports
    displayPorts(profile.ports || {});
    
    // Display protocols
    displayProtocols(profile.protocols || {});
    
    // Display hourly activity
    displayActivity(profile.hourly_activity || []);
}

// Display destinations list
function displayDestinations(destinations) {
    const container = document.getElementById('profileDestinations');
    
    // Convert to array and sort by count
    const destArray = Object.entries(destinations).map(([ip, info]) => ({
        ip: ip,
        count: info.count || 0,
        last_seen: info.last_seen
    }));
    
    destArray.sort((a, b) => b.count - a.count);
    
    if (destArray.length === 0) {
        container.innerHTML = '<p class="empty">No destinations recorded</p>';
        return;
    }

    container.innerHTML = destArray.slice(0, 10).map(dest => `
        <div class="list-item">
            <span class="mono">${escapeHtml(dest.ip)}</span>
            <span class="count">${formatNumber(dest.count)}</span>
        </div>
    `).join('');
}

// Display ports list
function displayPorts(ports) {
    const container = document.getElementById('profilePorts');
    
    // Convert to array and sort by count
    const portArray = Object.entries(ports).map(([port, count]) => ({
        port: parseInt(port),
        count: count
    }));
    
    portArray.sort((a, b) => b.count - a.count);
    
    if (portArray.length === 0) {
        container.innerHTML = '<p class="empty">No ports recorded</p>';
        return;
    }

    container.innerHTML = portArray.slice(0, 10).map(p => `
        <div class="list-item">
            <span>${p.port} <span class="port-name">${getPortName(p.port)}</span></span>
            <span class="count">${formatNumber(p.count)}</span>
        </div>
    `).join('');
}

// Display protocols list
function displayProtocols(protocols) {
    const container = document.getElementById('profileProtocols');
    
    // Convert to array and sort by count
    const protoArray = Object.entries(protocols).map(([proto, count]) => ({
        protocol: proto,
        count: count
    }));
    
    protoArray.sort((a, b) => b.count - a.count);
    
    if (protoArray.length === 0) {
        container.innerHTML = '<p class="empty">No protocols recorded</p>';
        return;
    }

    container.innerHTML = protoArray.map(p => `
        <div class="list-item">
            <span>${escapeHtml(p.protocol)}</span>
            <span class="count">${formatNumber(p.count)}</span>
        </div>
    `).join('');
}

// Display 24-hour activity chart
function displayActivity(hourlyActivity) {
    const container = document.getElementById('profileActivity');
    
    if (!hourlyActivity || hourlyActivity.length !== 24) {
        container.innerHTML = '<p class="empty">No activity data available</p>';
        return;
    }

    const maxActivity = Math.max(...hourlyActivity, 1);
    const currentHour = new Date().getHours();

    container.innerHTML = hourlyActivity.map((count, hour) => {
        const height = (count / maxActivity) * 100;
        const isCurrent = hour === currentHour;
        
        return `
            <div class="activity-bar ${isCurrent ? 'current' : ''}" 
                 style="height: ${height}%"
                 title="${hour}:00 - ${count} packets">
                <span class="hour-label">${hour}</span>
            </div>
        `;
    }).join('');
}

// Show profile error
function showProfileError(message) {
    document.getElementById('profilePackets').textContent = '-';
    document.getElementById('profileBytes').textContent = '-';
    document.getElementById('profileFirstSeen').textContent = '-';
    document.getElementById('profileLastSeen').textContent = '-';
    document.getElementById('profileDestinations').innerHTML = `<p class="empty">${message}</p>`;
    document.getElementById('profilePorts').innerHTML = `<p class="empty">${message}</p>`;
    document.getElementById('profileProtocols').innerHTML = `<p class="empty">${message}</p>`;
    document.getElementById('profileActivity').innerHTML = `<p class="empty">${message}</p>`;
}

// Close profile view
function closeProfile() {
    currentDeviceMAC = null;
    document.getElementById('profileSection').style.display = 'none';
}

// Utility functions

function formatNumber(num) {
    if (num >= 1000000) {
        return (num / 1000000).toFixed(1) + 'M';
    } else if (num >= 1000) {
        return (num / 1000).toFixed(1) + 'K';
    }
    return num.toString();
}

function formatBytes(bytes) {
    if (bytes >= 1073741824) {
        return (bytes / 1073741824).toFixed(2) + ' GB';
    } else if (bytes >= 1048576) {
        return (bytes / 1048576).toFixed(2) + ' MB';
    } else if (bytes >= 1024) {
        return (bytes / 1024).toFixed(2) + ' KB';
    }
    return bytes + ' B';
}

function formatTimestamp(timestamp) {
    if (!timestamp) return '-';
    const date = new Date(timestamp);
    const now = new Date();
    const diff = now - date;
    
    // Less than 1 minute
    if (diff < 60000) {
        return 'Just now';
    }
    // Less than 1 hour
    if (diff < 3600000) {
        const minutes = Math.floor(diff / 60000);
        return `${minutes}m ago`;
    }
    // Less than 24 hours
    if (diff < 86400000) {
        const hours = Math.floor(diff / 3600000);
        return `${hours}h ago`;
    }
    // More than 24 hours
    return date.toLocaleString();
}

function getPortName(port) {
    const commonPorts = {
        20: 'FTP Data',
        21: 'FTP',
        22: 'SSH',
        23: 'Telnet',
        25: 'SMTP',
        53: 'DNS',
        80: 'HTTP',
        110: 'POP3',
        143: 'IMAP',
        443: 'HTTPS',
        3306: 'MySQL',
        5432: 'PostgreSQL',
        6379: 'Redis',
        8080: 'HTTP Alt',
        8443: 'HTTPS Alt'
    };
    return commonPorts[port] ? `(${commonPorts[port]})` : '';
}

function escapeHtml(text) {
    if (!text) return '';
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}


// WebSocket connection for real-time updates
function connectWebSocket() {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/ws`;
    
    console.log('Connecting to WebSocket:', wsUrl);
    
    ws = new WebSocket(wsUrl);
    
    ws.onopen = () => {
        console.log('WebSocket connected');
    };
    
    ws.onmessage = (event) => {
        try {
            const message = JSON.parse(event.data);
            handleWebSocketMessage(message);
        } catch (error) {
            console.error('Failed to parse WebSocket message:', error);
        }
    };
    
    ws.onerror = (error) => {
        console.error('WebSocket error:', error);
    };
    
    ws.onclose = () => {
        console.log('WebSocket disconnected, reconnecting in 5 seconds...');
        setTimeout(connectWebSocket, 5000);
    };
}

// Handle WebSocket messages
function handleWebSocketMessage(message) {
    console.log('WebSocket message received:', message.type);
    
    switch (message.type) {
        case 'device':
            // New device discovered or device updated
            loadDevices();
            break;
            
        case 'traffic':
            // Traffic update - refresh current profile if viewing
            if (currentDeviceMAC) {
                loadProfile(currentDeviceMAC);
            }
            break;
            
        case 'anomaly':
            // Anomaly detected - could show notification
            console.log('Anomaly detected:', message.payload);
            loadDevices(); // Refresh to show any status changes
            break;
            
        case 'profile':
            // Profile updated - refresh if viewing this device
            if (currentDeviceMAC && message.payload && message.payload.mac === currentDeviceMAC) {
                loadProfile(currentDeviceMAC);
            }
            break;
            
        default:
            console.log('Unknown message type:', message.type);
    }
}
