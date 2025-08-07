package monitor

import (
	"fmt"
	"slices"
)

// ValidateMonitoringOperation checks if the monitoring operation is supported
func ValidateMonitoringOperation(operation string) bool {
	supportedOps := []string{
		string(OpMetrics), string(OpResourceHealth), string(OpAppInsights),
		string(OpDiagnostics), string(OpControlPlaneLogs),
	}
	return slices.Contains(supportedOps, operation)
}

// GetSupportedMonitoringOperations returns all supported monitoring operations
func GetSupportedMonitoringOperations() []string {
	return []string{
		string(OpMetrics), string(OpResourceHealth), string(OpAppInsights),
		string(OpDiagnostics), string(OpControlPlaneLogs),
	}
}

// ValidateMetricsQueryType checks if the metrics query type is supported
func ValidateMetricsQueryType(queryType string) bool {
	supportedTypes := []string{"list", "list-definitions", "list-namespaces"}
	return slices.Contains(supportedTypes, queryType)
}

// MapMetricsQueryTypeToCommand maps a metrics query type to its corresponding az command
func MapMetricsQueryTypeToCommand(queryType string) (string, error) {
	commandMap := map[string]string{
		"list":             "az monitor metrics list",
		"list-definitions": "az monitor metrics list-definitions",
		"list-namespaces":  "az monitor metrics list-namespaces",
	}

	cmd, exists := commandMap[queryType]
	if !exists {
		return "", fmt.Errorf("no command mapping for metrics query type: %s", queryType)
	}

	return cmd, nil
}
