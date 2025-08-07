package monitor

import (
	"fmt"
	"slices"

	"github.com/mark3labs/mcp-go/mcp"
)

// MonitoringOperationType defines the type of monitoring operation
type MonitoringOperationType string

const (
	OpMetrics          MonitoringOperationType = "metrics"
	OpResourceHealth   MonitoringOperationType = "resource_health"
	OpAppInsights      MonitoringOperationType = "app_insights"
	OpDiagnostics      MonitoringOperationType = "diagnostics"
	OpControlPlaneLogs MonitoringOperationType = "control_plane_logs"
)

// RegisterAzMonitoring registers the monitoring tool
func RegisterAzMonitoring() mcp.Tool {
	description := `Unified tool for Azure monitoring and diagnostics operations for AKS clusters.

SUPPORTED OPERATIONS:

1. METRICS - Query Azure Monitor metrics for AKS clusters and nodes
   - list: Get metric values for specific metrics
   - list-definitions: Get available metrics for a resource
   - list-namespaces: Get metric namespaces for a resource
   
   Use for: CPU usage, memory consumption, network traffic, pod counts, node health
   Required parameters: resource (AKS cluster resource ID), metrics, aggregation
   Optional: start-time, end-time, interval, filter

2. RESOURCE_HEALTH - Get Azure Resource Health events for AKS clusters
   Use for: Cluster availability issues, platform problems, service health events
   Required parameters: subscription_id, resource_group, cluster_name, start_time
   Optional: end_time, status (Available, Unavailable, Degraded, Unknown)

3. APP_INSIGHTS - Execute KQL queries against Application Insights telemetry
   Use for: Application performance monitoring, custom telemetry analysis, trace correlation
   Required parameters: subscription_id, resource_group, app_insights_name, query
   Optional: start_time, end_time, timespan

4. DIAGNOSTICS - Check AKS cluster diagnostic settings configuration
   Use for: Verify logging is enabled, check log retention, validate diagnostic configuration
   Required parameters: subscription_id, resource_group, cluster_name

5. CONTROL_PLANE_LOGS - Query AKS control plane logs with safety constraints
   SUPPORTED LOG CATEGORIES:
   - kube-apiserver: Kubernetes API server logs (authentication, authorization, admission controllers)
   - kube-audit: Kubernetes audit logs (API calls, security events)
   - kube-audit-admin: Administrative audit logs
   - kube-controller-manager: Controller manager logs (deployments, services, pods lifecycle)
   - kube-scheduler: Scheduler logs (pod placement decisions)
   - cluster-autoscaler: Cluster autoscaler logs (node scaling events)
   - cloud-controller-manager: Cloud provider integration logs
   - guard: Azure AD integration and RBAC logs
   - csi-azuredisk-controller: Azure disk CSI driver logs
   - csi-azurefile-controller: Azure file CSI driver logs
   - csi-snapshot-controller: CSI snapshot controller logs
   - fleet-member-agent: Fleet management agent logs
   - fleet-member-net-controller-manager: Fleet network controller logs
   - fleet-mcs-controller-manager: Fleet multi-cluster service logs

USE THIS TOOL WHEN YOU NEED TO:
- Monitor cluster performance and resource usage (use metrics)
- Check cluster availability and platform health (use resource_health)
- Analyze application telemetry and performance (use app_insights)
- Verify diagnostic logging configuration (use diagnostics)
- Debug Kubernetes API server issues (use control_plane_logs with kube-apiserver)
- Investigate authentication/authorization problems (use control_plane_logs with kube-audit, guard)
- Troubleshoot pod scheduling issues (use control_plane_logs with kube-scheduler)
- Check storage-related problems (use control_plane_logs with csi-azuredisk-controller, csi-azurefile-controller)
- Analyze cluster scaling behavior (use control_plane_logs with cluster-autoscaler)
- Review security audit events (use control_plane_logs with kube-audit, kube-audit-admin)

DETAILED EXAMPLES:

METRICS EXAMPLES:
- Get CPU usage: operation="metrics", query_type="list", parameters="{\"resource\":\"/subscriptions/sub-id/resourceGroups/rg/providers/Microsoft.ContainerService/managedClusters/cluster\", \"metrics\":\"node_cpu_usage_percentage\", \"aggregation\":\"Average\", \"start-time\":\"2025-01-01T00:00:00Z\", \"end-time\":\"2025-01-01T01:00:00Z\"}"
- List available metrics: operation="metrics", query_type="list-definitions", parameters="{\"resource\":\"/subscriptions/sub-id/resourceGroups/rg/providers/Microsoft.ContainerService/managedClusters/cluster\"}"
- Get metric namespaces: operation="metrics", query_type="list-namespaces", parameters="{\"resource\":\"/subscriptions/sub-id/resourceGroups/rg/providers/Microsoft.ContainerService/managedClusters/cluster\"}"
- Monitor node memory: operation="metrics", query_type="list", parameters="{\"resource\":\"/subscriptions/sub-id/resourceGroups/rg/providers/Microsoft.ContainerService/managedClusters/cluster\", \"metrics\":\"node_memory_working_set_percentage\", \"aggregation\":\"Average\", \"interval\":\"PT5M\"}"

RESOURCE HEALTH EXAMPLES:
- Check recent cluster health: operation="resource_health", subscription_id="<subscription-id>", resource_group="<resource-group>", cluster_name="<cluster-name>", parameters="{\"start_time\":\"2025-01-01T00:00:00Z\"}"
- Filter by health status: operation="resource_health", subscription_id="<subscription-id>", resource_group="<resource-group>", cluster_name="<cluster-name>", parameters="{\"start_time\":\"2025-01-01T00:00:00Z\", \"end_time\":\"2025-01-02T00:00:00Z\", \"status\":\"Unavailable\"}"
- Get health events for last 24h: operation="resource_health", subscription_id="<subscription-id>", resource_group="<resource-group>", cluster_name="<cluster-name>", parameters="{\"start_time\":\"2025-01-06T00:00:00Z\", \"end_time\":\"2025-01-07T00:00:00Z\"}"

APPLICATION INSIGHTS EXAMPLES:
- Query request telemetry: operation="app_insights", subscription_id="<subscription-id>", resource_group="<resource-group>", parameters="{\"app_insights_name\":\"myapp-insights\", \"query\":\"requests | where timestamp > ago(1h) | summarize count() by bin(timestamp, 5m)\"}"
- Analyze exceptions: operation="app_insights", subscription_id="<subscription-id>", resource_group="<resource-group>", parameters="{\"app_insights_name\":\"myapp-insights\", \"query\":\"exceptions | where timestamp > ago(24h) | summarize count() by type, bin(timestamp, 1h)\"}"
- Performance monitoring: operation="app_insights", subscription_id="<subscription-id>", resource_group="<resource-group>", parameters="{\"app_insights_name\":\"myapp-insights\", \"query\":\"performanceCounters | where timestamp > ago(1h) | where category == 'Processor' | summarize avg(value) by bin(timestamp, 5m)\", \"timespan\":\"PT1H\"}"

DIAGNOSTICS EXAMPLES:
- Check logging configuration: operation="diagnostics", parameters="{\"subscription_id\":\"<subscription-id>\", \"resource_group\":\"<resource-group>\", \"cluster_name\":\"<cluster-name>\"}"
- Verify diagnostic settings: operation="diagnostics", subscription_id="<subscription-id>", resource_group="<resource-group>", cluster_name="<cluster-name>", parameters="{}"

CONTROL PLANE LOGS EXAMPLES:
- Query API server logs: operation="control_plane_logs", subscription_id="<subscription-id>", resource_group="<resource-group>", cluster_name="<cluster-name>", parameters="{\"log_category\":\"kube-apiserver\", \"start_time\":\"2025-01-01T10:00:00Z\", \"end_time\":\"2025-01-01T11:00:00Z\", \"max_records\":\"50\"}"
- Debug authentication issues: operation="control_plane_logs", subscription_id="<subscription-id>", resource_group="<resource-group>", cluster_name="<cluster-name>", parameters="{\"log_category\":\"guard\", \"start_time\":\"2025-01-01T10:00:00Z\", \"end_time\":\"2025-01-01T11:00:00Z\", \"max_records\":\"100\"}"
- Analyze audit events: operation="control_plane_logs", subscription_id="<subscription-id>", resource_group="<resource-group>", cluster_name="<cluster-name>", parameters="{\"log_category\":\"kube-audit\", \"log_level\":\"error\", \"start_time\":\"2025-01-01T10:00:00Z\", \"end_time\":\"2025-01-01T11:00:00Z\", \"max_records\":\"50\"}"
- Check scheduler decisions: operation="control_plane_logs", subscription_id="<subscription-id>", resource_group="<resource-group>", cluster_name="<cluster-name>", parameters="{\"log_category\":\"kube-scheduler\", \"start_time\":\"2025-01-01T10:00:00Z\", \"end_time\":\"2025-01-01T11:00:00Z\", \"max_records\":\"75\"}"
- Monitor autoscaler activity: operation="control_plane_logs", subscription_id="<subscription-id>", resource_group="<resource-group>", cluster_name="<cluster-name>", parameters="{\"log_category\":\"cluster-autoscaler\", \"start_time\":\"2025-01-01T10:00:00Z\", \"end_time\":\"2025-01-01T11:00:00Z\", \"max_records\":\"50\"}"
- Storage troubleshooting: operation="control_plane_logs", subscription_id="<subscription-id>", resource_group="<resource-group>", cluster_name="<cluster-name>", parameters="{\"log_category\":\"csi-azuredisk-controller\", \"log_level\":\"error\", \"start_time\":\"2025-01-01T10:00:00Z\", \"end_time\":\"2025-01-01T11:00:00Z\", \"max_records\":\"25\"}"
`

	return mcp.NewTool("az_monitoring",
		mcp.WithDescription(description),
		mcp.WithString("operation",
			mcp.Required(),
			mcp.Description("The monitoring operation to perform: 'metrics' (CPU/memory/network), 'resource_health' (cluster availability), 'app_insights' (telemetry analysis), 'diagnostics' (logging config), 'control_plane_logs' (Kubernetes logs like kube-apiserver, kube-audit, guard, etc.)"),
		),
		mcp.WithString("query_type",
			mcp.Description("For metrics operations only: 'list' (get metric values), 'list-definitions' (available metrics), 'list-namespaces' (metric categories)"),
		),
		mcp.WithString("parameters",
			mcp.Required(),
			mcp.Description("JSON string with operation parameters. METRICS: resource, metrics, aggregation, start-time, end-time. RESOURCE_HEALTH: start_time, end_time, status. APP_INSIGHTS: app_insights_name, query, timespan. DIAGNOSTICS: none required. CONTROL_PLANE_LOGS: log_category (kube-apiserver/kube-audit/guard/etc), start_time, end_time, max_records, log_level"),
		),
		mcp.WithString("subscription_id",
			mcp.Description("Azure subscription ID (required for resource_health, app_insights, diagnostics, control_plane_logs)"),
		),
		mcp.WithString("resource_group",
			mcp.Description("Resource group name (required for resource_health, app_insights, diagnostics, control_plane_logs)"),
		),
		mcp.WithString("cluster_name",
			mcp.Description("AKS cluster name (required for resource_health, diagnostics, control_plane_logs)"),
		),
	)
}

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

// GetControlPlaneLogCategoriesHelp returns help text for control plane log categories
func GetControlPlaneLogCategoriesHelp() map[string]string {
	return map[string]string{
		"kube-apiserver":                      "Kubernetes API server logs - authentication, authorization, admission controllers",
		"kube-audit":                          "Kubernetes audit logs - API calls, security events, resource access",
		"kube-audit-admin":                    "Administrative audit logs - cluster admin operations",
		"kube-controller-manager":             "Controller manager logs - deployments, services, pods lifecycle management",
		"kube-scheduler":                      "Scheduler logs - pod placement decisions, resource constraints",
		"cluster-autoscaler":                  "Cluster autoscaler logs - node scaling events, capacity decisions",
		"cloud-controller-manager":            "Cloud provider integration logs - load balancers, storage, networking",
		"guard":                               "Azure AD integration and RBAC logs - authentication and authorization",
		"csi-azuredisk-controller":            "Azure disk CSI driver logs - persistent volume operations",
		"csi-azurefile-controller":            "Azure file CSI driver logs - file share operations",
		"csi-snapshot-controller":             "CSI snapshot controller logs - volume snapshot operations",
		"fleet-member-agent":                  "Fleet management agent logs - multi-cluster operations",
		"fleet-member-net-controller-manager": "Fleet network controller logs - cross-cluster networking",
		"fleet-mcs-controller-manager":        "Fleet multi-cluster service logs - service discovery across clusters",
	}
}

// SuggestLogCategoryForIssue suggests appropriate log categories based on common issues
func SuggestLogCategoryForIssue(issueType string) []string {
	suggestions := map[string][]string{
		"authentication": {"guard", "kube-audit", "kube-apiserver"},
		"authorization":  {"guard", "kube-audit", "kube-apiserver"},
		"pod_scheduling": {"kube-scheduler", "cluster-autoscaler", "kube-controller-manager"},
		"storage":        {"csi-azuredisk-controller", "csi-azurefile-controller", "csi-snapshot-controller"},
		"networking":     {"cloud-controller-manager", "fleet-member-net-controller-manager"},
		"scaling":        {"cluster-autoscaler", "kube-controller-manager"},
		"api_issues":     {"kube-apiserver", "kube-audit"},
		"security":       {"kube-audit", "kube-audit-admin", "guard"},
		"fleet":          {"fleet-member-agent", "fleet-member-net-controller-manager", "fleet-mcs-controller-manager"},
	}

	if categories, exists := suggestions[issueType]; exists {
		return categories
	}
	return []string{"kube-apiserver"} // default fallback
}
