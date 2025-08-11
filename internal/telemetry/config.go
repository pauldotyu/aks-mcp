package telemetry

import (
	"crypto/md5"
	"fmt"
	"net"
	"os"
	"strconv"
)

const (
	defaultInstrumentationKey = "c301e561-daea-42d9-b9d1-65fca4166704"
)

// Config represents the telemetry configuration
type Config struct {
	// Enabled controls whether telemetry collection is active
	Enabled bool
	// DeviceID is a hashed MAC address for device identification
	DeviceID string
	// instrumentationKey for Azure application insights
	instrumentationKey string
	// OTLPEndpoint for OpenTelemetry Protocol export
	OTLPEndpoint string
	// ServiceName identifies the service in telemetry
	ServiceName string
	// ServiceVersion identifies the service version in telemetry
	ServiceVersion string
}

// NewConfig creates a new telemetry configuration from environment variables
func NewConfig(serviceName, serviceVersion string) *Config {
	cfg := &Config{
		ServiceName:    serviceName,
		ServiceVersion: serviceVersion,
		OTLPEndpoint:   "",
	}

	// Check if telemetry is enabled (default: true)
	cfg.Enabled = true
	if envVal := os.Getenv("AKS_MCP_COLLECT_TELEMETRY"); envVal != "" {
		if enabled, err := strconv.ParseBool(envVal); err == nil {
			cfg.Enabled = enabled
		}
	}

	// Only initialize device ID and connection strings if telemetry is enabled
	if cfg.Enabled {
		cfg.DeviceID = generateDeviceID()
		cfg.instrumentationKey = getApplicationInsightsInstrumentationKey()
	}

	return cfg
}

// generateDeviceID creates a hashed MAC address for device identification
func generateDeviceID() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		// Fallback to a default identifier if network interfaces can't be read
		return fmt.Sprintf("%x", md5.Sum([]byte("aks-mcp-fallback")))
	}

	// Find the first non-loopback interface with a MAC address
	for _, iface := range interfaces {
		if iface.Flags&net.FlagLoopback == 0 && iface.HardwareAddr != nil {
			return fmt.Sprintf("%x", md5.Sum([]byte(iface.HardwareAddr.String())))
		}
	}

	// Fallback if no suitable interface is found
	return fmt.Sprintf("%x", md5.Sum([]byte("aks-mcp-no-mac")))
}

// getApplicationInsightsInstrumentationKey retrieves the instrumentation key from environment
func getApplicationInsightsInstrumentationKey() string {
	// Check for explicit instrumentation key first
	if instrKey := os.Getenv("APPLICATIONINSIGHTS_INSTRUMENTATION_KEY"); instrKey != "" {
		return instrKey
	}

	// Default instrumentation key for AKS MCP
	return defaultInstrumentationKey
}

// HasOTLP returns whether OTLP export is configured
func (c *Config) HasOTLP() bool {
	return c.OTLPEndpoint != ""
}

// HasApplicationInsights returns whether application insinghts export is configured
func (c *Config) HasApplicationInsights() bool {
	return c.Enabled && c.instrumentationKey != ""
}

// SetOTLPEndpoint sets the OTLP endpoint for CLI configuration
func (c *Config) SetOTLPEndpoint(endpoint string) {
	c.OTLPEndpoint = endpoint
}
