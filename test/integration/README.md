# Integration Tests

This directory contains integration tests for the Heimdal sensor that verify the interaction between multiple components.

## Test Files

### device_discovery_test.go
Tests the integration between device discovery and database persistence:
- **TestDeviceDiscoveryToDatabaseFlow**: Verifies devices discovered by the scanner are properly persisted to the database
- **TestDeviceBatchPersistence**: Tests batch device save operations
- **TestDeviceLifecycle**: Tests device creation, update, and deletion operations

### profiler_test.go
Tests the integration between packet analyzer, profiler, and database:
- **TestPacketAnalyzerToProfilerToDatabaseFlow**: Verifies packets are processed by the profiler and profiles are persisted to the database
- **TestProfileBatchPersistence**: Tests batch profile save operations
- **TestProfileAggregation**: Tests profile aggregation logic with multiple devices
- **TestHourlyActivityTracking**: Tests the hourly activity tracking feature

### api_test.go
Tests the integration between database and API responses:
- **TestDatabaseToAPIResponseFlow**: Verifies API endpoints return correct data from the database
  - GET /api/v1/devices
  - GET /api/v1/devices/:mac
  - GET /api/v1/profiles/:mac
  - GET /api/v1/stats
  - GET /api/v1/health
- **TestAPIRateLimiting**: Tests the rate limiting functionality
- **TestAPICORS**: Tests CORS headers are properly set

### orchestrator_test.go
Tests the orchestrator shutdown sequence and component lifecycle:
- **TestOrchestratorShutdownSequence**: Verifies graceful shutdown of all components
- **TestOrchestratorComponentInitialization**: Tests component initialization order
- **TestOrchestratorWithMinimalConfig**: Tests orchestrator with minimal configuration
- **TestOrchestratorComponentHealth**: Tests component health tracking
- **TestOrchestratorConfigValidation**: Tests configuration validation
- **TestOrchestratorDatabasePersistence**: Tests data persists across restarts

## Running Tests

Run all integration tests:
```bash
go test -v ./test/integration -timeout 60s
```

Run specific test:
```bash
go test -v ./test/integration -run TestDeviceLifecycle -timeout 30s
```

Run tests matching a pattern:
```bash
go test -v ./test/integration -run "TestDevice.*" -timeout 30s
```

## Platform Notes

Some tests may be skipped on certain platforms:
- **macOS**: Network detection tests are skipped because macOS doesn't have `/proc/net/route`
- **Non-root environments**: Tests requiring network capabilities (ARP spoofing, packet capture) are skipped

## Test Requirements

- Go 1.21+
- Temporary directory access for database files
- Network access for API tests (localhost only)
- Some tests require root/sudo for network operations (these are skipped in non-privileged environments)

## Coverage

These integration tests cover the following requirements from the design document:
- **Requirement 2.1-2.4**: Device discovery and lifecycle management
- **Requirement 4.1-4.5**: Packet analysis and behavioral profiling
- **Requirement 5.1-5.4**: Local data persistence
- **Requirement 6.1-6.4**: Web API and dashboard
- **Requirement 9.1-9.4**: Application architecture and graceful shutdown

## Test Data

All tests use temporary directories created by `t.TempDir()` which are automatically cleaned up after test completion. No persistent test data is created.
