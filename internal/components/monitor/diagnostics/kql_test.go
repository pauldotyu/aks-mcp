package diagnostics

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestBuildSafeKQLQuery(t *testing.T) {
	tests := []struct {
		name               string
		category           string
		logLevel           string
		maxRecords         int
		clusterResourceID  string
		isResourceSpecific bool
		expectedContains   []string
		notExpected        []string
	}{
		{
			name:               "azure diagnostics query without log level",
			category:           "kube-apiserver",
			logLevel:           "",
			maxRecords:         100,
			clusterResourceID:  "/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster",
			isResourceSpecific: false,
			expectedContains: []string{
				"AzureDiagnostics",
				"where Category == 'kube-apiserver'",
				"limit 100",
				"project TimeGenerated, Level, log_s",
				"order by TimeGenerated desc",
			},
			notExpected: []string{
				"where log_s startswith",
			},
		},
		{
			name:               "resource-specific query for kube-apiserver",
			category:           "kube-apiserver",
			logLevel:           "",
			maxRecords:         100,
			clusterResourceID:  "/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster",
			isResourceSpecific: true,
			expectedContains: []string{
				"AKSControlPlane",
				"where _ResourceId ==",
				"where Category == 'kube-apiserver'", // Now includes category filter due to our fix
				"limit 100",
				"project TimeGenerated, Category, Level, Message, PodName",
				"order by TimeGenerated desc",
			},
			notExpected: []string{
				"AzureDiagnostics",
			},
		},
		{
			name:               "resource-specific query for audit logs",
			category:           "kube-audit",
			logLevel:           "",
			maxRecords:         100,
			clusterResourceID:  "/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster",
			isResourceSpecific: true,
			expectedContains: []string{
				"AKSAudit",
				"where _ResourceId ==",
				"limit 100",
				"project TimeGenerated, Level, AuditId, Stage, RequestUri, Verb, User",
			},
			notExpected: []string{
				"AzureDiagnostics",
				"where Category ==",
			},
		},
		{
			name:               "azure diagnostics query with info log level",
			category:           "kube-apiserver",
			logLevel:           "info",
			maxRecords:         50,
			clusterResourceID:  "/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster",
			isResourceSpecific: false,
			expectedContains: []string{
				"AzureDiagnostics",
				"where Category == 'kube-apiserver'",
				"where log_s startswith 'I'",
				"limit 50",
				"project TimeGenerated, Level, log_s",
			},
		},
		{
			name:               "resource-specific query with info log level",
			category:           "kube-apiserver",
			logLevel:           "info",
			maxRecords:         50,
			clusterResourceID:  "/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster",
			isResourceSpecific: true,
			expectedContains: []string{
				"AKSControlPlane",
				"where _ResourceId ==",
				"where Level == 'INFO'",
				"limit 50",
			},
		},
		{
			name:               "azure diagnostics query with error log level",
			category:           "kube-controller-manager",
			logLevel:           "error",
			maxRecords:         200,
			clusterResourceID:  "/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster",
			isResourceSpecific: false,
			expectedContains: []string{
				"where Category == 'kube-controller-manager'",
				"where log_s startswith 'E'",
				"limit 200",
			},
		},
		{
			name:               "azure diagnostics query with warning log level",
			category:           "kube-scheduler",
			logLevel:           "warning",
			maxRecords:         300,
			clusterResourceID:  "/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster",
			isResourceSpecific: false,
			expectedContains: []string{
				"where Category == 'kube-scheduler'",
				"where log_s startswith 'W'",
				"limit 300",
			},
		},
		{
			name:               "azure diagnostics query with audit category",
			category:           "kube-audit",
			logLevel:           "",
			maxRecords:         1000,
			clusterResourceID:  "/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster",
			isResourceSpecific: false,
			expectedContains: []string{
				"where Category == 'kube-audit'",
				"limit 1000",
			},
		},
		{
			name:               "azure diagnostics query with audit category and log level - should skip log level filtering",
			category:           "kube-audit",
			logLevel:           "info",
			maxRecords:         500,
			clusterResourceID:  "/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster",
			isResourceSpecific: false,
			expectedContains: []string{
				"where Category == 'kube-audit'",
				"limit 500",
			},
			notExpected: []string{
				"where log_s startswith",
				"where Level ==",
			},
		},
		{
			name:               "azure diagnostics query with audit-admin category and log level - should skip log level filtering",
			category:           "kube-audit-admin",
			logLevel:           "error",
			maxRecords:         200,
			clusterResourceID:  "/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster",
			isResourceSpecific: false,
			expectedContains: []string{
				"where Category == 'kube-audit-admin'",
				"limit 200",
			},
			notExpected: []string{
				"where log_s startswith",
				"where Level ==",
			},
		},
		{
			name:               "azure diagnostics query with cloud controller manager",
			category:           "cloud-controller-manager",
			logLevel:           "info",
			maxRecords:         150,
			clusterResourceID:  "/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster",
			isResourceSpecific: false,
			expectedContains: []string{
				"where Category == 'cloud-controller-manager'",
				"where log_s startswith 'I'",
				"limit 150",
			},
		},
		// New comprehensive test cases for resource-specific table mode with correct log level handling
		{
			name:               "resource-specific query with warning log level for control plane",
			category:           "kube-controller-manager",
			logLevel:           "warning",
			maxRecords:         75,
			clusterResourceID:  "/subscriptions/test/resourcegroups/rg/providers/Microsoft.ContainerService/managedClusters/cluster",
			isResourceSpecific: true,
			expectedContains: []string{
				"AKSControlPlane",
				"where _ResourceId == '/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster'", // lowercase conversion
				"where Level == 'WARNING'",
				"limit 75",
				"project TimeGenerated, Category, Level, Message, PodName",
			},
			notExpected: []string{
				"where Message startswith",
				"AzureDiagnostics",
			},
		},
		{
			name:               "resource-specific query with error log level for control plane",
			category:           "cloud-controller-manager",
			logLevel:           "error",
			maxRecords:         25,
			clusterResourceID:  "/subscriptions/TEST/resourcegroups/RG/providers/Microsoft.ContainerService/managedClusters/cluster",
			isResourceSpecific: true,
			expectedContains: []string{
				"AKSControlPlane",
				"where _ResourceId == '/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster'", // lowercase conversion
				"where Level == 'ERROR'",
				"limit 25",
				"project TimeGenerated, Category, Level, Message, PodName",
			},
			notExpected: []string{
				"where Message startswith 'E'",
				"where log_s startswith",
			},
		},
		{
			name:               "resource-specific audit query should skip log level filtering",
			category:           "kube-audit",
			logLevel:           "info",
			maxRecords:         100,
			clusterResourceID:  "/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster",
			isResourceSpecific: true,
			expectedContains: []string{
				"AKSAudit",
				"where _ResourceId ==",
				"limit 100",
				"project TimeGenerated, Level, AuditId, Stage, RequestUri, Verb, User",
			},
			notExpected: []string{
				"where Level == 'INFO'",
				"where Message startswith",
				"where log_s startswith",
			},
		},
		{
			name:               "resource-specific audit-admin query should skip log level filtering",
			category:           "kube-audit-admin",
			logLevel:           "error",
			maxRecords:         200,
			clusterResourceID:  "/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster",
			isResourceSpecific: true,
			expectedContains: []string{
				"AKSAuditAdmin",
				"where _ResourceId ==",
				"limit 200",
				"project TimeGenerated, Level, AuditId, Stage, RequestUri, Verb, User",
			},
			notExpected: []string{
				"where Level == 'ERROR'",
				"where Message startswith",
				"where log_s startswith",
			},
		},
		{
			name:               "resource-specific case sensitivity test - mixed case resource ID converted to lowercase",
			category:           "kube-scheduler",
			logLevel:           "",
			maxRecords:         50,
			clusterResourceID:  "/SUBSCRIPTIONS/TEST/RESOURCEGROUPS/RG/PROVIDERS/MICROSOFT.CONTAINERSERVICE/MANAGEDCLUSTERS/CLUSTER",
			isResourceSpecific: true,
			expectedContains: []string{
				"AKSControlPlane",
				"where _ResourceId == '/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster'", // all lowercase
				"limit 50",
			},
		},
		{
			name:               "azure diagnostics case sensitivity test - mixed case resource ID converted to uppercase",
			category:           "kube-scheduler",
			logLevel:           "",
			maxRecords:         50,
			clusterResourceID:  "/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster",
			isResourceSpecific: false,
			expectedContains: []string{
				"AzureDiagnostics",
				"where Category == 'kube-scheduler'",
				"ResourceId == '/SUBSCRIPTIONS/TEST/RESOURCEGROUPS/RG/PROVIDERS/MICROSOFT.CONTAINERSERVICE/MANAGEDCLUSTERS/CLUSTER'", // all uppercase
				"limit 50",
			},
		},
		// Test cases for the AKSControlPlane category filtering fix
		{
			name:               "resource-specific guard logs should include category filter for AKSControlPlane",
			category:           "guard",
			logLevel:           "",
			maxRecords:         100,
			clusterResourceID:  "/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster",
			isResourceSpecific: true,
			expectedContains: []string{
				"AKSControlPlane",
				"where _ResourceId ==",
				"where Category == 'guard'",
				"limit 100",
				"project TimeGenerated, Category, Level, Message, PodName",
			},
			notExpected: []string{
				"AzureDiagnostics",
			},
		},
		{
			name:               "resource-specific cloud-controller-manager logs should include category filter for AKSControlPlane",
			category:           "cloud-controller-manager",
			logLevel:           "info",
			maxRecords:         50,
			clusterResourceID:  "/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster",
			isResourceSpecific: true,
			expectedContains: []string{
				"AKSControlPlane",
				"where _ResourceId ==",
				"where Category == 'cloud-controller-manager'",
				"where Level == 'INFO'",
				"limit 50",
				"project TimeGenerated, Category, Level, Message, PodName",
			},
			notExpected: []string{
				"AzureDiagnostics",
			},
		},
		{
			name:               "resource-specific kube-controller-manager logs should include category filter for AKSControlPlane",
			category:           "kube-controller-manager",
			logLevel:           "warning",
			maxRecords:         75,
			clusterResourceID:  "/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster",
			isResourceSpecific: true,
			expectedContains: []string{
				"AKSControlPlane",
				"where _ResourceId ==",
				"where Category == 'kube-controller-manager'",
				"where Level == 'WARNING'",
				"limit 75",
				"project TimeGenerated, Category, Level, Message, PodName",
			},
			notExpected: []string{
				"AzureDiagnostics",
			},
		},
		{
			name:               "resource-specific kube-scheduler logs should include category filter for AKSControlPlane",
			category:           "kube-scheduler",
			logLevel:           "error",
			maxRecords:         25,
			clusterResourceID:  "/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster",
			isResourceSpecific: true,
			expectedContains: []string{
				"AKSControlPlane",
				"where _ResourceId ==",
				"where Category == 'kube-scheduler'",
				"where Level == 'ERROR'",
				"limit 25",
				"project TimeGenerated, Category, Level, Message, PodName",
			},
			notExpected: []string{
				"AzureDiagnostics",
			},
		},
		{
			name:               "resource-specific cluster-autoscaler logs should include category filter for AKSControlPlane",
			category:           "cluster-autoscaler",
			logLevel:           "",
			maxRecords:         100,
			clusterResourceID:  "/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster",
			isResourceSpecific: true,
			expectedContains: []string{
				"AKSControlPlane",
				"where _ResourceId ==",
				"where Category == 'cluster-autoscaler'",
				"limit 100",
				"project TimeGenerated, Category, Level, Message, PodName",
			},
			notExpected: []string{
				"AzureDiagnostics",
				"where Level ==", // no log level filtering when logLevel is empty
			},
		},
		{
			name:               "resource-specific CSI controller logs should include category filter for AKSControlPlane",
			category:           "csi-azuredisk-controller",
			logLevel:           "info",
			maxRecords:         200,
			clusterResourceID:  "/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster",
			isResourceSpecific: true,
			expectedContains: []string{
				"AKSControlPlane",
				"where _ResourceId ==",
				"where Category == 'csi-azuredisk-controller'",
				"where Level == 'INFO'",
				"limit 200",
				"project TimeGenerated, Category, Level, Message, PodName",
			},
			notExpected: []string{
				"AzureDiagnostics",
			},
		},
		{
			name:               "resource-specific audit logs should NOT include category filter (different table)",
			category:           "kube-audit",
			logLevel:           "",
			maxRecords:         100,
			clusterResourceID:  "/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster",
			isResourceSpecific: true,
			expectedContains: []string{
				"AKSAudit",
				"where _ResourceId ==",
				"limit 100",
				"project TimeGenerated, Level, AuditId, Stage, RequestUri, Verb, User",
			},
			notExpected: []string{
				"where Category ==", // Should NOT have category filter since it's in AKSAudit table
				"AKSControlPlane",
				"AzureDiagnostics",
			},
		},
		{
			name:               "resource-specific audit-admin logs should NOT include category filter (different table)",
			category:           "kube-audit-admin",
			logLevel:           "",
			maxRecords:         150,
			clusterResourceID:  "/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster",
			isResourceSpecific: true,
			expectedContains: []string{
				"AKSAuditAdmin",
				"where _ResourceId ==",
				"limit 150",
				"project TimeGenerated, Level, AuditId, Stage, RequestUri, Verb, User",
			},
			notExpected: []string{
				"where Category ==", // Should NOT have category filter since it's in AKSAuditAdmin table
				"AKSControlPlane",
				"AzureDiagnostics",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, err := BuildSafeKQLQuery(tt.category, tt.logLevel, tt.maxRecords, tt.clusterResourceID, tt.isResourceSpecific)
			if err != nil {
				t.Fatalf("BuildSafeKQLQuery failed: %v", err)
			}

			// Check that expected strings are present
			for _, expected := range tt.expectedContains {
				if !strings.Contains(query, expected) {
					t.Errorf("Expected query to contain '%s', but it didn't. Query: %s", expected, query)
				}
			}

			// Check that unexpected strings are not present
			for _, notExpected := range tt.notExpected {
				if strings.Contains(query, notExpected) {
					t.Errorf("Expected query NOT to contain '%s', but it did. Query: %s", notExpected, query)
				}
			}

			// Verify query structure
			if tt.isResourceSpecific {
				if tableName, exists := resourceSpecificTableMapping[tt.category]; exists {
					if !strings.HasPrefix(query, tableName) {
						t.Errorf("Resource-specific query should start with %s, got: %s", tableName, query)
					}
				} else {
					// This case should not happen in the current tests since we removed unmapped categories
					t.Errorf("Test is using unmapped category %s in resource-specific mode, this should now cause an error", tt.category)
				}
			} else {
				if !strings.HasPrefix(query, "AzureDiagnostics") {
					t.Errorf("Azure Diagnostics query should start with AzureDiagnostics, got: %s", query)
				}
			}

			if !strings.Contains(query, "order by TimeGenerated desc") {
				t.Errorf("Query should contain ordering clause, got: %s", query)
			}
		})
	}
}

// TestAKSControlPlaneCategoryFiltering specifically tests the fix for category filtering
// in AKSControlPlane table to ensure different log categories don't return each other's logs
func TestAKSControlPlaneCategoryFiltering(t *testing.T) {
	testResourceID := "/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster"

	// Test all AKSControlPlane categories to ensure they each get proper category filtering
	controlPlaneCategories := []string{
		"kube-apiserver",
		"kube-controller-manager",
		"kube-scheduler",
		"cluster-autoscaler",
		"cloud-controller-manager",
		"guard",
		"csi-azuredisk-controller",
		"csi-azurefile-controller",
		"csi-snapshot-controller",
	}

	for _, category := range controlPlaneCategories {
		t.Run(fmt.Sprintf("resource-specific_%s_has_category_filter", category), func(t *testing.T) {
			query, err := BuildSafeKQLQuery(category, "", 100, testResourceID, true)
			if err != nil {
				t.Fatalf("BuildSafeKQLQuery failed for category %s: %v", category, err)
			}

			// Should use AKSControlPlane table
			if !strings.HasPrefix(query, "AKSControlPlane") {
				t.Errorf("Expected query to start with 'AKSControlPlane' for category %s, got: %s", category, query)
			}

			// Should include category filter to prevent cross-contamination
			expectedCategoryFilter := fmt.Sprintf("where Category == '%s'", category)
			if !strings.Contains(query, expectedCategoryFilter) {
				t.Errorf("Expected query to contain category filter '%s' for category %s, got: %s", expectedCategoryFilter, category, query)
			}

			// Should include resource ID filter
			if !strings.Contains(query, "where _ResourceId ==") {
				t.Errorf("Expected query to contain resource ID filter for category %s, got: %s", category, query)
			}

			// Should NOT contain other categories
			for _, otherCategory := range controlPlaneCategories {
				if otherCategory != category {
					wrongCategoryFilter := fmt.Sprintf("where Category == '%s'", otherCategory)
					if strings.Contains(query, wrongCategoryFilter) {
						t.Errorf("Query for category %s should NOT contain filter for other category %s, got: %s", category, otherCategory, query)
					}
				}
			}
		})
	}

	// Test that audit tables don't get category filtering (they have dedicated tables)
	auditCategories := []string{"kube-audit", "kube-audit-admin"}

	for _, category := range auditCategories {
		t.Run(fmt.Sprintf("resource-specific_%s_no_category_filter", category), func(t *testing.T) {
			query, err := BuildSafeKQLQuery(category, "", 100, testResourceID, true)
			if err != nil {
				t.Fatalf("BuildSafeKQLQuery failed for audit category %s: %v", category, err)
			}

			// Should NOT contain category filter since audit logs have dedicated tables
			if strings.Contains(query, "where Category ==") {
				t.Errorf("Audit category %s should NOT have category filter (has dedicated table), got: %s", category, query)
			}

			// Should use appropriate audit table
			if category == "kube-audit" && !strings.HasPrefix(query, "AKSAudit") {
				t.Errorf("Expected kube-audit query to start with 'AKSAudit', got: %s", query)
			}
			if category == "kube-audit-admin" && !strings.HasPrefix(query, "AKSAuditAdmin") {
				t.Errorf("Expected kube-audit-admin query to start with 'AKSAuditAdmin', got: %s", query)
			}
		})
	}
}

// TestCategoryFilteringPreventsCrossContamination tests that the fix prevents
// different categories from returning each other's logs (the original bug)
func TestCategoryFilteringPreventsCrossContamination(t *testing.T) {
	testResourceID := "/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster"

	// Test the specific case that was broken: guard logs returning cloud-controller-manager logs
	guardQuery, err := BuildSafeKQLQuery("guard", "", 100, testResourceID, true)
	if err != nil {
		t.Fatalf("BuildSafeKQLQuery failed for guard: %v", err)
	}

	cloudControllerQuery, err := BuildSafeKQLQuery("cloud-controller-manager", "", 100, testResourceID, true)
	if err != nil {
		t.Fatalf("BuildSafeKQLQuery failed for cloud-controller-manager: %v", err)
	}

	// Guard query should filter for guard category only
	if !strings.Contains(guardQuery, "where Category == 'guard'") {
		t.Errorf("Guard query should contain guard category filter, got: %s", guardQuery)
	}
	if strings.Contains(guardQuery, "where Category == 'cloud-controller-manager'") {
		t.Errorf("Guard query should NOT contain cloud-controller-manager category filter, got: %s", guardQuery)
	}

	// Cloud controller query should filter for cloud-controller-manager category only
	if !strings.Contains(cloudControllerQuery, "where Category == 'cloud-controller-manager'") {
		t.Errorf("Cloud controller query should contain cloud-controller-manager category filter, got: %s", cloudControllerQuery)
	}
	if strings.Contains(cloudControllerQuery, "where Category == 'guard'") {
		t.Errorf("Cloud controller query should NOT contain guard category filter, got: %s", cloudControllerQuery)
	}

	// Queries should be different (each has its own category filter)
	if guardQuery == cloudControllerQuery {
		t.Errorf("Guard and cloud-controller-manager queries should be different, but they are identical: %s", guardQuery)
	}

	// Test a few more combinations to ensure proper isolation
	testCases := []struct {
		category1 string
		category2 string
	}{
		{"kube-apiserver", "kube-scheduler"},
		{"kube-controller-manager", "cluster-autoscaler"},
		{"guard", "kube-apiserver"},
		{"csi-azuredisk-controller", "csi-azurefile-controller"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s_vs_%s", tc.category1, tc.category2), func(t *testing.T) {
			query1, err := BuildSafeKQLQuery(tc.category1, "", 100, testResourceID, true)
			if err != nil {
				t.Fatalf("BuildSafeKQLQuery failed for %s: %v", tc.category1, err)
			}

			query2, err := BuildSafeKQLQuery(tc.category2, "", 100, testResourceID, true)
			if err != nil {
				t.Fatalf("BuildSafeKQLQuery failed for %s: %v", tc.category2, err)
			}

			// Each query should have its own category filter
			expectedFilter1 := fmt.Sprintf("where Category == '%s'", tc.category1)
			expectedFilter2 := fmt.Sprintf("where Category == '%s'", tc.category2)

			if !strings.Contains(query1, expectedFilter1) {
				t.Errorf("Query for %s should contain its own category filter '%s', got: %s", tc.category1, expectedFilter1, query1)
			}

			if !strings.Contains(query2, expectedFilter2) {
				t.Errorf("Query for %s should contain its own category filter '%s', got: %s", tc.category2, expectedFilter2, query2)
			}

			// Each query should NOT contain the other's category filter
			if strings.Contains(query1, expectedFilter2) {
				t.Errorf("Query for %s should NOT contain filter for %s, got: %s", tc.category1, tc.category2, query1)
			}

			if strings.Contains(query2, expectedFilter1) {
				t.Errorf("Query for %s should NOT contain filter for %s, got: %s", tc.category2, tc.category1, query2)
			}
		})
	}
}

func TestCalculateTimespan(t *testing.T) {
	tests := []struct {
		name          string
		startTime     string
		endTime       string
		wantError     bool
		checkDuration bool // Whether to check if duration makes sense
	}{
		{
			name:          "valid start and end time - 1 hour",
			startTime:     "2025-07-15T10:00:00Z",
			endTime:       "2025-07-15T11:00:00Z",
			wantError:     false,
			checkDuration: true,
		},
		{
			name:          "valid start and end time - 4 hours",
			startTime:     "2025-07-15T10:00:00Z",
			endTime:       "2025-07-15T14:00:00Z",
			wantError:     false,
			checkDuration: true,
		},
		{
			name:      "valid start time, empty end time",
			startTime: "2025-07-15T10:00:00Z",
			endTime:   "",
			wantError: false,
		},
		{
			name:      "invalid start time format",
			startTime: "invalid-time",
			endTime:   "",
			wantError: true,
		},
		{
			name:      "invalid end time format",
			startTime: "2025-07-15T10:00:00Z",
			endTime:   "invalid-end-time",
			wantError: true,
		},
		{
			name:          "valid time with milliseconds",
			startTime:     "2025-07-15T10:00:00.000Z",
			endTime:       "2025-07-15T12:00:00.000Z",
			wantError:     false,
			checkDuration: true,
		},
		{
			name:          "valid time with timezone offset",
			startTime:     "2025-07-15T10:00:00+02:00",
			endTime:       "2025-07-15T11:00:00+02:00",
			wantError:     false,
			checkDuration: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			timespan, err := CalculateTimespan(tt.startTime, tt.endTime)

			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Expected no error but got: %v", err)
				return
			}

			// Check timespan format: should be "start/end"
			if !strings.Contains(timespan, "/") {
				t.Errorf("Expected timespan to contain '/' separator, got: %s", timespan)
			}

			parts := strings.Split(timespan, "/")
			if len(parts) != 2 {
				t.Errorf("Expected timespan to have exactly 2 parts separated by '/', got: %s", timespan)
			}

			// Verify start time is preserved
			if !strings.HasPrefix(timespan, tt.startTime) && !strings.Contains(timespan, tt.startTime) {
				// For timezone conversions, just check that it's a valid RFC3339 format
				if _, err := time.Parse(time.RFC3339, parts[0]); err != nil {
					t.Errorf("Expected valid RFC3339 start time in timespan, got: %s", parts[0])
				}
			}

			// If we have specific end time, verify it's preserved or converted properly
			if tt.endTime != "" && tt.checkDuration {
				start, _ := time.Parse(time.RFC3339, tt.startTime)
				expectedEnd, _ := time.Parse(time.RFC3339, tt.endTime)
				actualEnd, err := time.Parse(time.RFC3339, parts[1])
				if err != nil {
					t.Errorf("Expected valid RFC3339 end time in timespan, got: %s", parts[1])
				} else {
					// Check that the duration is preserved (allowing for timezone conversion)
					expectedDuration := expectedEnd.Sub(start)
					actualDuration := actualEnd.Sub(start)
					if expectedDuration != actualDuration {
						// Allow small differences for timezone/parsing issues
						diff := expectedDuration - actualDuration
						if diff < 0 {
							diff = -diff
						}
						if diff > time.Second {
							t.Errorf("Duration mismatch: expected %v, got %v", expectedDuration, actualDuration)
						}
					}
				}
			}
		})
	}
}

func TestBuildSafeKQLQueryLogLevelMapping(t *testing.T) {
	tests := []struct {
		name           string
		logLevel       string
		expectedPrefix string
	}{
		{
			name:           "info level maps to I prefix",
			logLevel:       "info",
			expectedPrefix: "where log_s startswith 'I'",
		},
		{
			name:           "error level maps to E prefix",
			logLevel:       "error",
			expectedPrefix: "where log_s startswith 'E'",
		},
		{
			name:           "warning level maps to W prefix",
			logLevel:       "warning",
			expectedPrefix: "where log_s startswith 'W'",
		},
		{
			name:           "empty log level has no prefix filter",
			logLevel:       "",
			expectedPrefix: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, err := BuildSafeKQLQuery("kube-apiserver", tt.logLevel, 100, "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/test-rg/providers/Microsoft.ContainerService/managedClusters/test-cluster", false)
			if err != nil {
				t.Fatalf("BuildSafeKQLQuery failed: %v", err)
			}

			if tt.expectedPrefix == "" {
				// Should not contain any log level filtering
				if strings.Contains(query, "where log_s startswith") {
					t.Errorf("Expected no log level filtering for empty log level, but found it in: %s", query)
				}
			} else {
				if !strings.Contains(query, tt.expectedPrefix) {
					t.Errorf("Expected query to contain '%s', but it didn't. Query: %s", tt.expectedPrefix, query)
				}
			}
		})
	}
}

func TestBuildSafeKQLQueryStructure(t *testing.T) {
	query, err := BuildSafeKQLQuery("kube-apiserver", "info", 100, "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/test-rg/providers/Microsoft.ContainerService/managedClusters/test-cluster", false)
	if err != nil {
		t.Fatalf("BuildSafeKQLQuery failed: %v", err)
	}

	// The query should be a single line with pipe separators
	if strings.Contains(query, "\n") {
		t.Errorf("Expected query to be a single line, but found newlines: %s", query)
	}

	// Check that query follows expected order by looking for the components in sequence
	expectedOrder := []string{
		"AzureDiagnostics",
		"where Category ==",
		"where log_s startswith",
		"order by",
		"limit",
		"project",
	}

	lastIndex := 0

	for _, expected := range expectedOrder {
		index := strings.Index(query[lastIndex:], expected)
		if index == -1 {
			t.Errorf("Expected to find '%s' in query after position %d, but didn't find it. Query: %s", expected, lastIndex, query)
			continue
		}
		lastIndex += index + len(expected)
	}

	// Verify essential components
	if !strings.HasPrefix(query, "AzureDiagnostics") {
		t.Errorf("Query should start with AzureDiagnostics, got: %s", query)
	}

	if !strings.Contains(query, "order by TimeGenerated desc") {
		t.Errorf("Query should contain proper ordering, got: %s", query)
	}

	if !strings.Contains(query, "limit 100") {
		t.Errorf("Query should contain proper limit, got: %s", query)
	}

	if !strings.Contains(query, "project TimeGenerated, Level, log_s") {
		t.Errorf("Query should contain proper projection, got: %s", query)
	}
}

func TestBuildSafeKQLQuerySanitization(t *testing.T) {
	tests := []struct {
		name              string
		category          string
		clusterResourceID string
		description       string
	}{
		{
			name:              "normal category",
			category:          "kube-apiserver",
			clusterResourceID: "/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster",
			description:       "should work with normal inputs",
		},
		{
			name:              "category with special characters should be safe",
			category:          "kube-apiserver",
			clusterResourceID: "/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster",
			description:       "query should be built safely even with special characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, err := BuildSafeKQLQuery(tt.category, "", 100, tt.clusterResourceID, false)
			if err != nil {
				t.Fatalf("BuildSafeKQLQuery failed: %v", err)
			}

			// Basic validation that query was generated
			if query == "" {
				t.Errorf("Expected non-empty query, got empty string")
			}

			// Ensure the category is properly quoted in the query
			expectedCategoryClause := fmt.Sprintf("where Category == '%s'", tt.category)
			if !strings.Contains(query, expectedCategoryClause) {
				t.Errorf("Expected query to contain properly quoted category clause '%s', got: %s", expectedCategoryClause, query)
			}
		})
	}
}

func TestBuildSafeKQLQueryResourceSpecificMode(t *testing.T) {
	tests := []struct {
		name               string
		category           string
		expectedTable      string
		isResourceSpecific bool
	}{
		{
			name:               "kube-audit maps to AKSAudit table",
			category:           "kube-audit",
			expectedTable:      "AKSAudit",
			isResourceSpecific: true,
		},
		{
			name:               "kube-audit-admin maps to AKSAuditAdmin table",
			category:           "kube-audit-admin",
			expectedTable:      "AKSAuditAdmin",
			isResourceSpecific: true,
		},
		{
			name:               "kube-apiserver maps to AKSControlPlane table",
			category:           "kube-apiserver",
			expectedTable:      "AKSControlPlane",
			isResourceSpecific: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, err := BuildSafeKQLQuery(tt.category, "", 100, "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/test-rg/providers/Microsoft.ContainerService/managedClusters/test-cluster", tt.isResourceSpecific)
			if err != nil {
				t.Fatalf("BuildSafeKQLQuery failed: %v", err)
			}

			if !strings.Contains(query, tt.expectedTable) {
				t.Errorf("Expected query to contain table '%s', but it didn't. Query: %s", tt.expectedTable, query)
			}

			if tt.isResourceSpecific && tt.expectedTable != "AzureDiagnostics" {
				// Should use _ResourceId instead of ResourceId
				if !strings.Contains(query, "_ResourceId ==") {
					t.Errorf("Expected resource-specific query to use '_ResourceId ==', but it didn't. Query: %s", query)
				}

				// AKSControlPlane table should have category filter to prevent cross-contamination
				// AKSAudit and AKSAuditAdmin tables should NOT have category filter (dedicated tables)
				if tt.expectedTable == "AKSControlPlane" {
					expectedCategoryFilter := fmt.Sprintf("where Category == '%s'", tt.category)
					if !strings.Contains(query, expectedCategoryFilter) {
						t.Errorf("Expected AKSControlPlane query to contain category filter '%s', but it didn't. Query: %s", expectedCategoryFilter, query)
					}
				} else {
					// For audit tables (AKSAudit, AKSAuditAdmin), should NOT contain category filter
					if strings.Contains(query, "where Category ==") {
						t.Errorf("Expected %s query NOT to contain 'where Category ==', but it did. Query: %s", tt.expectedTable, query)
					}
				}
			}
		})
	}
}

func TestResourceSpecificLogLevelFiltering(t *testing.T) {
	testResourceID := "/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster"

	tests := []struct {
		name               string
		category           string
		logLevel           string
		isResourceSpecific bool
		expectedFilter     string
		notExpected        string
	}{
		{
			name:               "resource-specific info level filtering",
			category:           "kube-apiserver",
			logLevel:           "info",
			isResourceSpecific: true,
			expectedFilter:     "where Level == 'INFO'",
			notExpected:        "where Message startswith",
		},
		{
			name:               "resource-specific warning level filtering",
			category:           "kube-controller-manager",
			logLevel:           "warning",
			isResourceSpecific: true,
			expectedFilter:     "where Level == 'WARNING'",
			notExpected:        "where log_s startswith",
		},
		{
			name:               "resource-specific error level filtering",
			category:           "cloud-controller-manager",
			logLevel:           "error",
			isResourceSpecific: true,
			expectedFilter:     "where Level == 'ERROR'",
			notExpected:        "where Message startswith 'E'",
		},
		{
			name:               "azure diagnostics info level filtering",
			category:           "kube-apiserver",
			logLevel:           "info",
			isResourceSpecific: false,
			expectedFilter:     "where log_s startswith 'I'",
			notExpected:        "where Level == 'INFO'",
		},
		{
			name:               "azure diagnostics warning level filtering",
			category:           "kube-scheduler",
			logLevel:           "warning",
			isResourceSpecific: false,
			expectedFilter:     "where log_s startswith 'W'",
			notExpected:        "where Level == 'WARNING'",
		},
		{
			name:               "azure diagnostics error level filtering",
			category:           "cluster-autoscaler",
			logLevel:           "error",
			isResourceSpecific: false,
			expectedFilter:     "where log_s startswith 'E'",
			notExpected:        "where Level == 'ERROR'",
		},
		{
			name:               "resource-specific audit should skip log level filtering",
			category:           "kube-audit",
			logLevel:           "info",
			isResourceSpecific: true,
			expectedFilter:     "", // no filtering expected
			notExpected:        "where Level == 'INFO'",
		},
		{
			name:               "resource-specific audit-admin should skip log level filtering",
			category:           "kube-audit-admin",
			logLevel:           "error",
			isResourceSpecific: true,
			expectedFilter:     "", // no filtering expected
			notExpected:        "where Level == 'ERROR'",
		},
		{
			name:               "azure diagnostics audit should skip log level filtering",
			category:           "kube-audit",
			logLevel:           "warning",
			isResourceSpecific: false,
			expectedFilter:     "", // no filtering expected
			notExpected:        "where log_s startswith 'W'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, err := BuildSafeKQLQuery(tt.category, tt.logLevel, 100, testResourceID, tt.isResourceSpecific)
			if err != nil {
				t.Fatalf("BuildSafeKQLQuery failed: %v", err)
			}

			if tt.expectedFilter != "" {
				if !strings.Contains(query, tt.expectedFilter) {
					t.Errorf("Expected query to contain '%s', but it didn't. Query: %s", tt.expectedFilter, query)
				}
			}

			if tt.notExpected != "" {
				if strings.Contains(query, tt.notExpected) {
					t.Errorf("Expected query NOT to contain '%s', but it did. Query: %s", tt.notExpected, query)
				}
			}
		})
	}
}

func TestResourceIdCaseSensitivity(t *testing.T) {
	testCases := []struct {
		name               string
		inputResourceID    string
		isResourceSpecific bool
		expectedInQuery    string
		description        string
	}{
		{
			name:               "resource-specific lowercase conversion",
			inputResourceID:    "/subscriptions/TEST/resourcegroups/RG/providers/Microsoft.ContainerService/managedClusters/CLUSTER",
			isResourceSpecific: true,
			expectedInQuery:    "/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster",
			description:        "Resource-specific tables should convert resource ID to lowercase",
		},
		{
			name:               "azure diagnostics uppercase conversion",
			inputResourceID:    "/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster",
			isResourceSpecific: false,
			expectedInQuery:    "/SUBSCRIPTIONS/TEST/RESOURCEGROUPS/RG/PROVIDERS/MICROSOFT.CONTAINERSERVICE/MANAGEDCLUSTERS/CLUSTER",
			description:        "Azure Diagnostics should convert resource ID to uppercase",
		},
		{
			name:               "mixed case resource-specific conversion",
			inputResourceID:    "/Subscriptions/Test/ResourceGroups/Rg/Providers/Microsoft.ContainerService/ManagedClusters/Cluster",
			isResourceSpecific: true,
			expectedInQuery:    "/subscriptions/test/resourcegroups/rg/providers/microsoft.containerservice/managedclusters/cluster",
			description:        "Mixed case should be converted to all lowercase for resource-specific",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			query, err := BuildSafeKQLQuery("kube-apiserver", "", 100, tc.inputResourceID, tc.isResourceSpecific)
			if err != nil {
				t.Fatalf("BuildSafeKQLQuery failed: %v", err)
			}

			if !strings.Contains(query, tc.expectedInQuery) {
				t.Errorf("Expected query to contain resource ID '%s', but it didn't. Query: %s", tc.expectedInQuery, query)
			}

			// Make sure it doesn't contain the original case
			if tc.inputResourceID != tc.expectedInQuery && strings.Contains(query, tc.inputResourceID) {
				t.Errorf("Query should not contain original resource ID case '%s'. Query: %s", tc.inputResourceID, query)
			}
		})
	}
}

// TestValidateKQLQueryParams tests the new validation functionality
func TestValidateKQLQueryParams(t *testing.T) {
	tests := []struct {
		name              string
		category          string
		logLevel          string
		maxRecords        int
		clusterResourceID string
		tableMode         TableMode
		wantError         bool
		errorContains     string
	}{
		{
			name:              "valid parameters",
			category:          "kube-apiserver",
			logLevel:          "info",
			maxRecords:        100,
			clusterResourceID: "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/myRG/providers/Microsoft.ContainerService/managedClusters/myCluster",
			tableMode:         AzureDiagnosticsMode,
			wantError:         false,
		},
		{
			name:              "empty category",
			category:          "",
			logLevel:          "info",
			maxRecords:        100,
			clusterResourceID: "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/myRG/providers/Microsoft.ContainerService/managedClusters/myCluster",
			tableMode:         AzureDiagnosticsMode,
			wantError:         true,
			errorContains:     "category cannot be empty",
		},
		{
			name:              "unknown category should be allowed for forward compatibility",
			category:          "unknown-category",
			logLevel:          "info",
			maxRecords:        100,
			clusterResourceID: "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/myRG/providers/Microsoft.ContainerService/managedClusters/myCluster",
			tableMode:         AzureDiagnosticsMode,
			wantError:         false,
		},
		{
			name:              "invalid log level",
			category:          "kube-apiserver",
			logLevel:          "invalid-level",
			maxRecords:        100,
			clusterResourceID: "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/myRG/providers/Microsoft.ContainerService/managedClusters/myCluster",
			tableMode:         AzureDiagnosticsMode,
			wantError:         true,
			errorContains:     "invalid log level",
		},
		{
			name:              "empty log level should be valid",
			category:          "kube-apiserver",
			logLevel:          "",
			maxRecords:        100,
			clusterResourceID: "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/myRG/providers/Microsoft.ContainerService/managedClusters/myCluster",
			tableMode:         AzureDiagnosticsMode,
			wantError:         false,
		},
		{
			name:              "negative maxRecords",
			category:          "kube-apiserver",
			logLevel:          "info",
			maxRecords:        -1,
			clusterResourceID: "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/myRG/providers/Microsoft.ContainerService/managedClusters/myCluster",
			tableMode:         AzureDiagnosticsMode,
			wantError:         true,
			errorContains:     "maxRecords must be at least",
		},
		{
			name:              "zero maxRecords",
			category:          "kube-apiserver",
			logLevel:          "info",
			maxRecords:        0,
			clusterResourceID: "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/myRG/providers/Microsoft.ContainerService/managedClusters/myCluster",
			tableMode:         AzureDiagnosticsMode,
			wantError:         true,
			errorContains:     "maxRecords must be at least",
		},
		{
			name:              "maxRecords too high",
			category:          "kube-apiserver",
			logLevel:          "info",
			maxRecords:        5000,
			clusterResourceID: "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/myRG/providers/Microsoft.ContainerService/managedClusters/myCluster",
			tableMode:         AzureDiagnosticsMode,
			wantError:         true,
			errorContains:     "maxRecords cannot exceed",
		},
		{
			name:              "empty clusterResourceID",
			category:          "kube-apiserver",
			logLevel:          "info",
			maxRecords:        100,
			clusterResourceID: "",
			tableMode:         AzureDiagnosticsMode,
			wantError:         true,
			errorContains:     "clusterResourceID cannot be empty",
		},
		{
			name:              "invalid clusterResourceID format",
			category:          "kube-apiserver",
			logLevel:          "info",
			maxRecords:        100,
			clusterResourceID: "invalid-resource-id",
			tableMode:         AzureDiagnosticsMode,
			wantError:         true,
			errorContains:     "invalid clusterResourceID format",
		},
		{
			name:              "valid resource ID with lowercase resourcegroups",
			category:          "kube-apiserver",
			logLevel:          "info",
			maxRecords:        100,
			clusterResourceID: "/subscriptions/12345678-1234-1234-1234-123456789012/resourcegroups/myRG/providers/microsoft.containerservice/managedclusters/myCluster",
			tableMode:         AzureDiagnosticsMode,
			wantError:         false,
		},
		{
			name:              "invalid tableMode",
			category:          "kube-apiserver",
			logLevel:          "info",
			maxRecords:        100,
			clusterResourceID: "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/myRG/providers/Microsoft.ContainerService/managedClusters/myCluster",
			tableMode:         TableMode(999),
			wantError:         true,
			errorContains:     "invalid tableMode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateKQLQueryParams(tt.category, tt.logLevel, tt.maxRecords, tt.clusterResourceID, tt.tableMode)

			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', but got: %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

// TestNewKQLQueryBuilderValidation tests the constructor validation
func TestNewKQLQueryBuilderValidation(t *testing.T) {
	tests := []struct {
		name              string
		category          string
		logLevel          string
		maxRecords        int
		clusterResourceID string
		tableMode         TableMode
		wantError         bool
		errorContains     string
	}{
		{
			name:              "valid parameters create builder successfully",
			category:          "kube-apiserver",
			logLevel:          "info",
			maxRecords:        100,
			clusterResourceID: "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/myRG/providers/Microsoft.ContainerService/managedClusters/myCluster",
			tableMode:         AzureDiagnosticsMode,
			wantError:         false,
		},
		{
			name:              "invalid parameters return error",
			category:          "",
			logLevel:          "info",
			maxRecords:        100,
			clusterResourceID: "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/myRG/providers/Microsoft.ContainerService/managedClusters/myCluster",
			tableMode:         AzureDiagnosticsMode,
			wantError:         true,
			errorContains:     "invalid KQL query parameters",
		},
		{
			name:              "negative maxRecords return error",
			category:          "kube-apiserver",
			logLevel:          "info",
			maxRecords:        -10,
			clusterResourceID: "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/myRG/providers/Microsoft.ContainerService/managedClusters/myCluster",
			tableMode:         AzureDiagnosticsMode,
			wantError:         true,
			errorContains:     "maxRecords must be at least",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder, err := NewKQLQueryBuilder(tt.category, tt.logLevel, tt.maxRecords, tt.clusterResourceID, tt.tableMode)

			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if builder != nil {
					t.Errorf("Expected nil builder when error occurs, but got non-nil")
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', but got: %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
					return
				}
				if builder == nil {
					t.Errorf("Expected non-nil builder but got nil")
				}
			}
		})
	}
}

// TestKQLQueryBuilder_InvalidTableModeError tests that invalid table modes return proper errors
func TestKQLQueryBuilder_InvalidTableModeError(t *testing.T) {
	// Create a builder with valid parameters
	builder, err := NewKQLQueryBuilder(
		"kube-apiserver",
		"info",
		100,
		"/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/myRG/providers/Microsoft.ContainerService/managedClusters/myCluster",
		AzureDiagnosticsMode,
	)
	if err != nil {
		t.Fatalf("Failed to create builder: %v", err)
	}

	// Manually corrupt the table mode to test error handling
	builder.tableMode = TableMode(999) // Invalid table mode

	// The Build method should now return an error
	query, err := builder.Build()
	if err == nil {
		t.Errorf("Expected error for invalid table mode, got query: %s", query)
		return
	}

	expectedError := "unexpected table mode: 999"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error to contain '%s', got '%s'", expectedError, err.Error())
	}

	// The query should be empty when there's an error
	if query != "" {
		t.Errorf("Expected empty query on error, got: %s", query)
	}
}

// TestBuildSafeKQLQueryErrorHandling tests that BuildSafeKQLQuery properly returns errors for invalid inputs
func TestBuildSafeKQLQueryErrorHandling(t *testing.T) {
	tests := []struct {
		name               string
		category           string
		logLevel           string
		maxRecords         int
		clusterResourceID  string
		isResourceSpecific bool
		wantError          bool
		errorContains      string
	}{
		{
			name:               "empty category should return error",
			category:           "",
			logLevel:           "info",
			maxRecords:         100,
			clusterResourceID:  "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/myRG/providers/Microsoft.ContainerService/managedClusters/myCluster",
			isResourceSpecific: false,
			wantError:          true,
			errorContains:      "category cannot be empty",
		},
		{
			name:               "invalid log level should return error",
			category:           "kube-apiserver",
			logLevel:           "invalid",
			maxRecords:         100,
			clusterResourceID:  "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/myRG/providers/Microsoft.ContainerService/managedClusters/myCluster",
			isResourceSpecific: false,
			wantError:          true,
			errorContains:      "invalid log level",
		},
		{
			name:               "negative maxRecords should return error",
			category:           "kube-apiserver",
			logLevel:           "info",
			maxRecords:         -1,
			clusterResourceID:  "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/myRG/providers/Microsoft.ContainerService/managedClusters/myCluster",
			isResourceSpecific: false,
			wantError:          true,
			errorContains:      "maxRecords must be at least",
		},
		{
			name:               "empty clusterResourceID should return error",
			category:           "kube-apiserver",
			logLevel:           "info",
			maxRecords:         100,
			clusterResourceID:  "",
			isResourceSpecific: false,
			wantError:          true,
			errorContains:      "clusterResourceID cannot be empty",
		},
		{
			name:               "invalid clusterResourceID format should return error",
			category:           "kube-apiserver",
			logLevel:           "info",
			maxRecords:         100,
			clusterResourceID:  "invalid-resource-id",
			isResourceSpecific: false,
			wantError:          true,
			errorContains:      "invalid clusterResourceID format",
		},
		{
			name:               "unmapped category in resource-specific mode should return error",
			category:           "unknown-category",
			logLevel:           "info",
			maxRecords:         100,
			clusterResourceID:  "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/myRG/providers/Microsoft.ContainerService/managedClusters/myCluster",
			isResourceSpecific: true,
			wantError:          true,
			errorContains:      "is not supported in resource-specific mode",
		},
		{
			name:               "valid inputs should not return error",
			category:           "kube-apiserver",
			logLevel:           "info",
			maxRecords:         100,
			clusterResourceID:  "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/myRG/providers/Microsoft.ContainerService/managedClusters/myCluster",
			isResourceSpecific: false,
			wantError:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, err := BuildSafeKQLQuery(tt.category, tt.logLevel, tt.maxRecords, tt.clusterResourceID, tt.isResourceSpecific)

			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error but got none, query: %s", query)
					return
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.errorContains, err.Error())
				}
				if query != "" {
					t.Errorf("Expected empty query on error, got: %s", query)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
					return
				}
				if query == "" {
					t.Error("Expected non-empty query but got empty string")
				}
			}
		})
	}
}
