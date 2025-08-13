package compute

import (
	"fmt"
	"slices"

	"github.com/Azure/aks-mcp/internal/config"
	"github.com/mark3labs/mcp-go/mcp"
)

// ComputeOperationType defines the type of compute operation
type ComputeOperationType string

// ResourceType defines the compute resource type
type ResourceType string

const (
	// VM operations
	OpVMShow            ComputeOperationType = "show"
	OpVMList            ComputeOperationType = "list"
	OpVMCreate          ComputeOperationType = "create"
	OpVMDelete          ComputeOperationType = "delete"
	OpVMStart           ComputeOperationType = "start"
	OpVMStop            ComputeOperationType = "stop"
	OpVMRestart         ComputeOperationType = "restart"
	OpVMDeallocate      ComputeOperationType = "deallocate"
	OpVMResize          ComputeOperationType = "resize"
	OpVMUpdate          ComputeOperationType = "update"
	OpVMGetInstanceView ComputeOperationType = "get-instance-view"
	OpVMRunCommand      ComputeOperationType = "run-command"

	// VMSS operations (some overlap with VM)
	OpVMSSShow            ComputeOperationType = "show"
	OpVMSSList            ComputeOperationType = "list"
	OpVMSSCreate          ComputeOperationType = "create"
	OpVMSSDelete          ComputeOperationType = "delete"
	OpVMSSStart           ComputeOperationType = "start"
	OpVMSSStop            ComputeOperationType = "stop"
	OpVMSSRestart         ComputeOperationType = "restart"
	OpVMSSDeallocate      ComputeOperationType = "deallocate"
	OpVMSSScale           ComputeOperationType = "scale"
	OpVMSSUpdate          ComputeOperationType = "update"
	OpVMSSReimage         ComputeOperationType = "reimage"
	OpVMSSGetInstanceView ComputeOperationType = "get-instance-view"
	OpVMSSRunCommand      ComputeOperationType = "run-command"

	// Resource types
	ResourceTypeVM   ResourceType = "vm"
	ResourceTypeVMSS ResourceType = "vmss"
)

// generateToolDescription creates a tool description based on access level
func generateToolDescription(accessLevel string) string {
	baseDesc := `Unified tool for managing Azure Virtual Machines (VMs) and Virtual Machine Scale Sets (VMSS) using Azure CLI.

Use resource_type="vm" for single virtual machines or resource_type="vmss" for virtual machine scale sets.

Available operation values:`

	// Add operations by access level
	desc := baseDesc + "\n"

	// Basic operations for all access levels
	desc += "- show: Get details of a VM/VMSS\n"
	desc += "- list: List VMs/VMSS in subscription or resource group\n"
	desc += "- get-instance-view: Get runtime status\n"

	// Management operations for readwrite/admin
	if accessLevel == "readwrite" || accessLevel == "admin" {
		desc += "- start: Start VM/VMSS\n"
		desc += "- stop: Stop VM/VMSS\n"
		desc += "- restart: Restart VM/VMSS\n"
		desc += "- deallocate: Stop and deallocate VM/VMSS\n"
		desc += "- run-command: Execute commands remotely\n"
		desc += "- scale: Change VMSS instance count\n"
		desc += "- reimage: Reset VMSS instances\n"
	}

	// Admin operations
	if accessLevel == "admin" {
		desc += "\nAdmin-only operation values:\n"
		desc += "- create: Create new VM/VMSS\n"
		desc += "- delete: Delete VM/VMSS\n"
		desc += "- update: Update VM/VMSS configuration\n"
		desc += "- resize: Change VM size\n"
	}

	// Examples
	desc += "\nEXAMPLES:\n"
	desc += `List VMSS: operation="list", resource_type="vmss", args="--resource-group myRG"` + "\n"
	desc += `Show VMSS: operation="show", resource_type="vmss", args="--name myVMSS --resource-group myRG"` + "\n"
	desc += `List VMs: operation="list", resource_type="vm", args="--resource-group myRG"` + "\n"

	if accessLevel == "readwrite" || accessLevel == "admin" {
		desc += `Start VMSS: operation="start", resource_type="vmss", args="--name myVMSS --resource-group myRG"` + "\n"
		desc += `Run command on VM: operation="run-command", resource_type="vm", args="--name myVM --resource-group myRG --command-id RunShellScript --scripts 'echo hello'"` + "\n"
		desc += `Run command on VMSS: operation="run-command", resource_type="vmss", args="--name myVMSS --resource-group myRG --command-id RunShellScript --scripts 'hostname' --instance-id 0"` + "\n"
	}

	if accessLevel == "admin" {
		desc += `Create VMSS: operation="create", resource_type="vmss", args="--name newVMSS --resource-group myRG --image Ubuntu2204 --admin-username azuser --instance-count 3 --vm-sku Standard_B2s"` + "\n"
		desc += `Update VMSS: operation="update", resource_type="vmss", args="--name myVMSS --resource-group myRG --set upgradePolicy.mode=Automatic"` + "\n"
		desc += `Delete VMSS: operation="delete", resource_type="vmss", args="--name myVMSS --resource-group myRG"` + "\n"
		desc += `Create VM: operation="create", resource_type="vm", args="--name newVM --resource-group myRG --image Ubuntu2204 --admin-username azuser --generate-ssh-keys"` + "\n"
	}

	return desc
}

// RegisterAzComputeOperations registers the unified compute operations tool
func RegisterAzComputeOperations(cfg *config.ConfigData) mcp.Tool {
	description := generateToolDescription(cfg.AccessLevel)

	return mcp.NewTool("az_compute_operations",
		mcp.WithDescription(description),
		mcp.WithString("operation",
			mcp.Required(),
			mcp.Description("Operation to perform. Common operations: list, show, start, stop, restart, deallocate, run-command, scale, etc."),
		),
		mcp.WithString("resource_type",
			mcp.Required(),
			mcp.Description("Resource type: 'vm' (single virtual machine) or 'vmss' (virtual machine scale set)"),
		),
		mcp.WithString("args",
			mcp.Required(),
			mcp.Description("Azure CLI arguments: '--resource-group myRG' (required for most operations), '--name myVM' (for specific resources), '--new-capacity 3' (for scaling)"),
		),
	)
}

// GetOperationAccessLevel returns the required access level for an operation
func GetOperationAccessLevel(operation string) string {
	readOnlyOps := []string{
		string(OpVMShow), string(OpVMList), string(OpVMGetInstanceView),
		string(OpVMSSShow), string(OpVMSSList), string(OpVMSSGetInstanceView),
	}

	readWriteOps := []string{
		string(OpVMStart), string(OpVMStop), string(OpVMRestart),
		string(OpVMDeallocate), string(OpVMRunCommand),
		string(OpVMSSStart), string(OpVMSSStop), string(OpVMSSRestart),
		string(OpVMSSDeallocate), string(OpVMSSScale), string(OpVMSSReimage),
		string(OpVMSSRunCommand),
	}

	adminOps := []string{
		string(OpVMCreate), string(OpVMDelete), string(OpVMResize), string(OpVMUpdate),
		string(OpVMSSCreate), string(OpVMSSDelete), string(OpVMSSUpdate),
	}

	if slices.Contains(readOnlyOps, operation) {
		return "readonly"
	}

	if slices.Contains(readWriteOps, operation) {
		return "readwrite"
	}

	if slices.Contains(adminOps, operation) {
		return "admin"
	}

	return "unknown"
}

// ValidateOperationAccess checks if the operation is allowed for the given access level
func ValidateOperationAccess(operation string, cfg *config.ConfigData) error {
	requiredLevel := GetOperationAccessLevel(operation)

	switch requiredLevel {
	case "admin":
		if cfg.AccessLevel != "admin" {
			return fmt.Errorf("operation '%s' requires admin access level", operation)
		}
	case "readwrite":
		if cfg.AccessLevel != "readwrite" && cfg.AccessLevel != "admin" {
			return fmt.Errorf("operation '%s' requires readwrite or admin access level", operation)
		}
	case "readonly":
		// All access levels can perform readonly operations
	case "unknown":
		return fmt.Errorf("unknown operation: %s", operation)
	}

	return nil
}

// MapOperationToCommand maps an operation and resource type to its corresponding az command
func MapOperationToCommand(operation string, resourceType string) (string, error) {
	// Validate resource type
	if resourceType != string(ResourceTypeVM) && resourceType != string(ResourceTypeVMSS) {
		return "", fmt.Errorf("invalid resource type: %s (must be 'vm' or 'vmss')", resourceType)
	}

	commandMap := map[string]map[string]string{
		string(ResourceTypeVM): {
			string(OpVMShow):            "az vm show",
			string(OpVMList):            "az vm list",
			string(OpVMCreate):          "az vm create",
			string(OpVMDelete):          "az vm delete",
			string(OpVMStart):           "az vm start",
			string(OpVMStop):            "az vm stop",
			string(OpVMRestart):         "az vm restart",
			string(OpVMDeallocate):      "az vm deallocate",
			string(OpVMResize):          "az vm resize",
			string(OpVMUpdate):          "az vm update",
			string(OpVMGetInstanceView): "az vm get-instance-view",
			string(OpVMRunCommand):      "az vm run-command invoke",
		},
		string(ResourceTypeVMSS): {
			string(OpVMSSShow):            "az vmss show",
			string(OpVMSSList):            "az vmss list",
			string(OpVMSSCreate):          "az vmss create",
			string(OpVMSSDelete):          "az vmss delete",
			string(OpVMSSStart):           "az vmss start",
			string(OpVMSSStop):            "az vmss stop",
			string(OpVMSSRestart):         "az vmss restart",
			string(OpVMSSDeallocate):      "az vmss deallocate",
			string(OpVMSSScale):           "az vmss scale",
			string(OpVMSSUpdate):          "az vmss update",
			string(OpVMSSReimage):         "az vmss reimage",
			string(OpVMSSGetInstanceView): "az vmss get-instance-view",
			string(OpVMSSRunCommand):      "az vmss run-command invoke",
		},
	}

	resourceCommands, exists := commandMap[resourceType]
	if !exists {
		return "", fmt.Errorf("unsupported resource type: %s", resourceType)
	}

	cmd, exists := resourceCommands[operation]
	if !exists {
		return "", fmt.Errorf("unsupported operation '%s' for resource type '%s'", operation, resourceType)
	}

	return cmd, nil
}
