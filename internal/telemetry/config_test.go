package telemetry

import (
	"os"
	"testing"
)

func TestNewConfig(t *testing.T) {
	config := NewConfig("test-service", "v1.0.0")

	if config.ServiceName != "test-service" {
		t.Errorf("Expected service name 'test-service', got %s", config.ServiceName)
	}

	if config.ServiceVersion != "v1.0.0" {
		t.Errorf("Expected service version 'v1.0.0', got %s", config.ServiceVersion)
	}

	if !config.Enabled {
		t.Error("Expected telemetry to be enabled by default")
	}

	if config.DeviceID == "" {
		t.Error("Expected device ID to be generated")
	}
}

func TestConfigHasAzureMonitor(t *testing.T) {
	config := NewConfig("test", "v1.0.0")

	// Should be true with default connection string (valid connection string)
	if !config.HasApplicationInsights() {
		t.Error("Expected Azure Monitor to be enabled with default connection string")
	}

	// Set a real-looking connection string
	config.instrumentationKey = "12345678-1234-1234-1234-123456789012"
	if !config.HasApplicationInsights() {
		t.Error("Expected Azure Monitor to be enabled with valid connection string")
	}
}

func TestConfigHasOTLP(t *testing.T) {
	config := NewConfig("test", "v1.0.0")

	// Should be false by default (empty endpoint)
	if config.HasOTLP() {
		t.Error("Expected OTLP to be disabled by default (empty endpoint)")
	}

	// Set an endpoint and it should be enabled
	config.OTLPEndpoint = "localhost:4317"
	if !config.HasOTLP() {
		t.Error("Expected OTLP to be enabled with non-empty endpoint")
	}

	// Set empty endpoint and it should be disabled again
	config.OTLPEndpoint = ""
	if config.HasOTLP() {
		t.Error("Expected OTLP to be disabled with empty endpoint")
	}
}

func TestGenerateDeviceID(t *testing.T) {
	deviceID1 := generateDeviceID()
	deviceID2 := generateDeviceID()

	if deviceID1 == "" {
		t.Error("Expected device ID to be generated")
	}

	// Device ID should be consistent between calls
	if deviceID1 != deviceID2 {
		t.Error("Expected device ID to be consistent between calls")
	}

	// Device ID should be a hex string
	if len(deviceID1) != 32 { // MD5 hash is 32 hex characters
		t.Errorf("Expected device ID to be 32 characters, got %d", len(deviceID1))
	}
}

func TestSetOTLPEndpoint(t *testing.T) {
	config := NewConfig("test", "v1.0.0")

	// Initially should be empty and disabled
	if config.OTLPEndpoint != "" {
		t.Errorf("Expected empty OTLP endpoint initially, got %s", config.OTLPEndpoint)
	}
	if config.HasOTLP() {
		t.Error("Expected OTLP to be disabled initially")
	}

	// Set endpoint via SetOTLPEndpoint
	endpoint := "localhost:4317"
	config.SetOTLPEndpoint(endpoint)
	if config.OTLPEndpoint != endpoint {
		t.Errorf("Expected OTLP endpoint to be %s, got %s", endpoint, config.OTLPEndpoint)
	}
	if !config.HasOTLP() {
		t.Error("Expected OTLP to be enabled after setting endpoint")
	}

	// Clear endpoint
	config.SetOTLPEndpoint("")
	if config.OTLPEndpoint != "" {
		t.Errorf("Expected empty OTLP endpoint after clearing, got %s", config.OTLPEndpoint)
	}
	if config.HasOTLP() {
		t.Error("Expected OTLP to be disabled after clearing endpoint")
	}
}

func TestTelemetryDisableViaEnvironment(t *testing.T) {
	// Test that AKS_MCP_COLLECT_TELEMETRY=false disables telemetry
	os.Setenv("AKS_MCP_COLLECT_TELEMETRY", "false")
	defer os.Unsetenv("AKS_MCP_COLLECT_TELEMETRY")

	config := NewConfig("test-service", "v1.0.0")

	if config.Enabled {
		t.Error("Expected telemetry to be disabled when AKS_MCP_COLLECT_TELEMETRY=false")
	}

	// Device ID should be empty when telemetry is disabled
	if config.DeviceID != "" {
		t.Error("Expected device ID to be empty when telemetry is disabled")
	}
}

func TestTelemetryEnableViaEnvironment(t *testing.T) {
	// Test that AKS_MCP_COLLECT_TELEMETRY=true enables telemetry
	os.Setenv("AKS_MCP_COLLECT_TELEMETRY", "true")
	defer os.Unsetenv("AKS_MCP_COLLECT_TELEMETRY")

	config := NewConfig("test-service", "v1.0.0")

	if !config.Enabled {
		t.Error("Expected telemetry to be enabled when AKS_MCP_COLLECT_TELEMETRY=true")
	}

	// Device ID should be generated when telemetry is enabled
	if config.DeviceID == "" {
		t.Error("Expected device ID to be generated when telemetry is enabled")
	}
}
