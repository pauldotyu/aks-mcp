package azaks

import (
	"fmt"

	"github.com/Azure/aks-mcp/internal/azcli"
	"github.com/Azure/aks-mcp/internal/config"
)

// AksOperationsExecutor handles execution of AKS operations
type AksOperationsExecutor struct{}

// NewAksOperationsExecutor creates a new AksOperationsExecutor
func NewAksOperationsExecutor() *AksOperationsExecutor {
	return &AksOperationsExecutor{}
}

// Execute handles the AKS operations
func (e *AksOperationsExecutor) Execute(params map[string]interface{}, cfg *config.ConfigData) (string, error) {
	// Parse operation parameter
	operation, ok := params["operation"].(string)
	if !ok {
		return "", fmt.Errorf("missing or invalid 'operation' parameter")
	}

	// Parse args parameter
	args, ok := params["args"].(string)
	if !ok {
		args = ""
	}

	// Validate access for this operation
	if err := ValidateOperationAccess(operation, cfg); err != nil {
		return "", err
	}

	// Map operation to Azure CLI command
	baseCommand, err := MapOperationToCommand(operation)
	if err != nil {
		return "", err
	}

	// Build full command
	fullCommand := baseCommand
	if args != "" {
		fullCommand += " " + args
	}

	// Delegate execution to the shared az executor (handles validation and auto-login)
	exec := azcli.NewExecutor()
	return exec.Execute(map[string]interface{}{"command": fullCommand}, cfg)
}

// ExecuteSpecificCommand executes a specific operation with the given arguments (for backward compatibility)
func (e *AksOperationsExecutor) ExecuteSpecificCommand(operation string, params map[string]interface{}, cfg *config.ConfigData) (string, error) {
	// Create new params with operation
	newParams := make(map[string]interface{})
	for k, v := range params {
		newParams[k] = v
	}
	newParams["operation"] = operation

	return e.Execute(newParams, cfg)
}
