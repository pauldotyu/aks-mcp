package diagnostics

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// LogLevelMapping defines the mapping between log levels and their representations
type LogLevelMapping struct {
	AzureDiagnosticsPrefix string // Prefix used in Azure Diagnostics log_s field
	ResourceSpecificLevel  string // Level value used in resource-specific tables
}

// logLevelMappings contains the mapping for each log level
var logLevelMappings = map[string]LogLevelMapping{
	"info":    {AzureDiagnosticsPrefix: "I", ResourceSpecificLevel: "INFO"},
	"warning": {AzureDiagnosticsPrefix: "W", ResourceSpecificLevel: "WARNING"},
	"error":   {AzureDiagnosticsPrefix: "E", ResourceSpecificLevel: "ERROR"},
}

// AuditCategories defines which log categories are audit logs (different handling)
var auditCategories = map[string]bool{
	"kube-audit":       true,
	"kube-audit-admin": true,
}

// ResourceSpecificTableMapping defines the mapping from log categories to resource-specific table names
var resourceSpecificTableMapping = map[string]string{
	"kube-audit":               "AKSAudit",
	"kube-audit-admin":         "AKSAuditAdmin",
	"kube-apiserver":           "AKSControlPlane",
	"kube-controller-manager":  "AKSControlPlane",
	"kube-scheduler":           "AKSControlPlane",
	"cluster-autoscaler":       "AKSControlPlane",
	"cloud-controller-manager": "AKSControlPlane",
	"guard":                    "AKSControlPlane",
	"csi-azuredisk-controller": "AKSControlPlane",
	"csi-azurefile-controller": "AKSControlPlane",
	"csi-snapshot-controller":  "AKSControlPlane",
}

// KQLQueryBuilder builds KQL queries for AKS control plane logs
type KQLQueryBuilder struct {
	category            string    // The log category (e.g., "kube-audit", "kube-apiserver").
	logLevel            string    // The log level (e.g., "info", "warning", "error").
	maxRecords          int       // The maximum number of records to retrieve.
	clusterResourceID   string    // The resource ID of the cluster being queried.
	tableMode           TableMode // Specifies the mode of the table being queried (e.g., AzureDiagnosticsMode or ResourceSpecificMode).
	selectedTable       string    // The name of the table selected for the query.
	processedResourceID string    // The processed resource ID used in the query.
}

// TableMode represents the type of table being used
type TableMode int

const (
	AzureDiagnosticsMode TableMode = iota
	ResourceSpecificMode
)

// ValidKQLLogLevels defines valid log levels for KQL queries
var validKQLLogLevels = map[string]bool{
	"info":    true,
	"warning": true,
	"error":   true,
}

// KQL query validation constants
const (
	MinMaxRecords        = 1
	MaxMaxRecords        = 1000
	DefaultKQLMaxRecords = 100
)

// azureResourceIDPattern matches Azure resource IDs (case-insensitive, allows test IDs)
var azureResourceIDPattern = regexp.MustCompile(`(?i)^/subscriptions/[a-zA-Z0-9-]+/resourcegroups?/[^/]+/providers/microsoft\.containerservice/managedclusters/[^/]+$`)

// ValidateKQLQueryParams validates all parameters for KQL query builder
func ValidateKQLQueryParams(category, logLevel string, maxRecords int, clusterResourceID string, tableMode TableMode) error {
	// Validate category (empty not allowed, but unknown categories are permitted for forward compatibility)
	if category == "" {
		return fmt.Errorf("category cannot be empty")
	}
	// Note: We allow unknown categories for forward compatibility as Azure may add new log categories

	// Validate log level (empty is allowed)
	if logLevel != "" && !validKQLLogLevels[logLevel] {
		validLevels := make([]string, 0, len(validKQLLogLevels))
		for level := range validKQLLogLevels {
			validLevels = append(validLevels, level)
		}
		return fmt.Errorf("invalid log level '%s'. Valid levels: %s (or empty for no filtering)", logLevel, strings.Join(validLevels, ", "))
	}

	// Validate maxRecords
	if maxRecords < MinMaxRecords {
		return fmt.Errorf("maxRecords must be at least %d, got %d", MinMaxRecords, maxRecords)
	}
	if maxRecords > MaxMaxRecords {
		return fmt.Errorf("maxRecords cannot exceed %d, got %d", MaxMaxRecords, maxRecords)
	}

	// Validate clusterResourceID
	if clusterResourceID == "" {
		return fmt.Errorf("clusterResourceID cannot be empty")
	}
	if !azureResourceIDPattern.MatchString(clusterResourceID) {
		return fmt.Errorf("invalid clusterResourceID format. Expected format: /subscriptions/{subscription-id}/resourceGroups/{resource-group}/providers/Microsoft.ContainerService/managedClusters/{cluster-name}")
	}

	// Validate tableMode
	if tableMode != AzureDiagnosticsMode && tableMode != ResourceSpecificMode {
		return fmt.Errorf("invalid tableMode. Must be AzureDiagnosticsMode (%d) or ResourceSpecificMode (%d)", AzureDiagnosticsMode, ResourceSpecificMode)
	}

	return nil
}

// NewKQLQueryBuilder creates a new KQL query builder instance
func NewKQLQueryBuilder(category, logLevel string, maxRecords int, clusterResourceID string, tableMode TableMode) (*KQLQueryBuilder, error) {
	// Validate all input parameters
	if err := ValidateKQLQueryParams(category, logLevel, maxRecords, clusterResourceID, tableMode); err != nil {
		return nil, fmt.Errorf("invalid KQL query parameters: %w", err)
	}

	return &KQLQueryBuilder{
		category:          category,
		logLevel:          logLevel,
		maxRecords:        maxRecords,
		clusterResourceID: clusterResourceID,
		tableMode:         tableMode,
	}, nil
}

// determineTableStrategy decides which table to use and processes the resource ID accordingly
func (q *KQLQueryBuilder) determineTableStrategy() error {
	if q.tableMode == ResourceSpecificMode {
		if tableName, exists := resourceSpecificTableMapping[q.category]; exists {
			q.selectedTable = tableName
			// Resource-specific tables store _ResourceId in lowercase
			q.processedResourceID = strings.ToLower(q.clusterResourceID)
		} else {
			// Return error for unmapped categories in resource-specific mode
			return fmt.Errorf("category '%s' is not supported in resource-specific mode. Supported categories: %v",
				q.category, getSupportedResourceSpecificCategories())
		}
	} else {
		q.selectedTable = "AzureDiagnostics"
		q.processedResourceID = strings.ToUpper(q.clusterResourceID)
	}
	return nil
}

// getSupportedResourceSpecificCategories returns a list of supported categories for resource-specific mode
func getSupportedResourceSpecificCategories() []string {
	categories := make([]string, 0, len(resourceSpecificTableMapping))
	for category := range resourceSpecificTableMapping {
		categories = append(categories, category)
	}
	return categories
}

// buildBaseQuery creates the initial table selection and filtering clause
func (q *KQLQueryBuilder) buildBaseQuery() (string, error) {
	switch q.tableMode {
	case ResourceSpecificMode:
		baseQuery := fmt.Sprintf("%s | where _ResourceId == '%s'", q.selectedTable, q.processedResourceID)
		// For AKSControlPlane table, we need to filter by Category since multiple log categories use the same table
		if q.selectedTable == "AKSControlPlane" {
			baseQuery += fmt.Sprintf(" | where Category == '%s'", q.category)
		}
		return baseQuery, nil
	case AzureDiagnosticsMode:
		return fmt.Sprintf("%s | where Category == '%s' and ResourceId == '%s'", q.selectedTable, q.category, q.processedResourceID), nil
	default:
		// This should never happen if validation is working correctly
		return "", fmt.Errorf("unexpected table mode: %d. This indicates an internal error in query builder", q.tableMode)
	}
}

// isAuditCategory checks if the current category is an audit log category
func (q *KQLQueryBuilder) isAuditCategory() bool {
	return auditCategories[q.category]
}

// addLogLevelFilter adds log level filtering if applicable
func (q *KQLQueryBuilder) addLogLevelFilter(baseQuery string) string {
	// Skip log level filtering for audit categories or empty log level
	if q.logLevel == "" || q.isAuditCategory() {
		return baseQuery
	}

	mapping, exists := logLevelMappings[strings.ToLower(q.logLevel)]
	if !exists {
		return baseQuery // Unknown log level, skip filtering
	}

	switch q.tableMode {
	case ResourceSpecificMode:
		return baseQuery + fmt.Sprintf(" | where Level == '%s'", mapping.ResourceSpecificLevel)
	case AzureDiagnosticsMode:
		return baseQuery + fmt.Sprintf(" | where log_s startswith '%s'", mapping.AzureDiagnosticsPrefix)
	default:
		return baseQuery
	}
}

// addOrderingAndLimit adds the ordering and limit clauses
func (q *KQLQueryBuilder) addOrderingAndLimit(query string) string {
	query += " | order by TimeGenerated desc"
	query += fmt.Sprintf(" | limit %d", q.maxRecords)
	return query
}

// addProjection adds the appropriate field projection based on table type
func (q *KQLQueryBuilder) addProjection(query string) string {
	switch q.tableMode {
	case ResourceSpecificMode:
		return q.addResourceSpecificProjection(query)
	case AzureDiagnosticsMode:
		return query + " | project TimeGenerated, Level, log_s"
	default:
		return query + " | project TimeGenerated, Level, log_s"
	}
}

// addResourceSpecificProjection adds projection for resource-specific tables
func (q *KQLQueryBuilder) addResourceSpecificProjection(query string) string {
	switch q.selectedTable {
	case "AKSAudit", "AKSAuditAdmin":
		// Audit tables have structured fields
		return query + " | project TimeGenerated, Level, AuditId, Stage, RequestUri, Verb, User"
	case "AKSControlPlane":
		// Control plane table has message field
		return query + " | project TimeGenerated, Category, Level, Message, PodName"
	default:
		// Fallback projection for unknown resource-specific tables
		return query + " | project TimeGenerated, Level, Message"
	}
}

// Build constructs the complete KQL query
func (q *KQLQueryBuilder) Build() (string, error) {
	// Step 1: Determine table strategy
	if err := q.determineTableStrategy(); err != nil {
		return "", err
	}

	// Step 2: Build base query with table and resource filtering
	query, err := q.buildBaseQuery()
	if err != nil {
		return "", err
	}

	// Step 3: Add log level filtering
	query = q.addLogLevelFilter(query)

	// Step 4: Add ordering and limit
	query = q.addOrderingAndLimit(query)

	// Step 5: Add field projection
	query = q.addProjection(query)

	return query, nil
}

// BuildSafeKQLQuery builds pre-validated KQL queries to prevent injection, scoped to specific AKS cluster
// Supports both Azure Diagnostics and Resource-specific destination tables
// Returns an error if query building fails
func BuildSafeKQLQuery(category, logLevel string, maxRecords int, clusterResourceID string, isResourceSpecific bool) (string, error) {
	tableMode := AzureDiagnosticsMode
	if isResourceSpecific {
		tableMode = ResourceSpecificMode
	}

	builder, err := NewKQLQueryBuilder(category, logLevel, maxRecords, clusterResourceID, tableMode)
	if err != nil {
		return "", fmt.Errorf("failed to create KQL query builder: %w", err)
	}

	query, err := builder.Build()
	if err != nil {
		return "", fmt.Errorf("failed to build KQL query: %w", err)
	}

	return query, nil
}

// CalculateTimespan converts start/end times to Azure CLI timespan format
func CalculateTimespan(startTime, endTime string) (string, error) {
	start, err := time.Parse(time.RFC3339, startTime)
	if err != nil {
		return "", fmt.Errorf("invalid start time format: %w", err)
	}

	var end time.Time
	if endTime != "" {
		end, err = time.Parse(time.RFC3339, endTime)
		if err != nil {
			return "", fmt.Errorf("invalid end time format: %w", err)
		}
	} else {
		// Default to current time if no end time specified
		end = time.Now()
	}

	// Azure CLI timespan format: start_time/end_time in ISO8601
	timespan := fmt.Sprintf("%s/%s", start.Format(time.RFC3339), end.Format(time.RFC3339))
	return timespan, nil
}
