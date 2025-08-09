package azcli

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/Azure/aks-mcp/internal/command"
	"github.com/Azure/aks-mcp/internal/config"
	"github.com/Azure/aks-mcp/internal/security"
	"github.com/Azure/aks-mcp/internal/tools"
)

// AzExecutor implements the CommandExecutor interface for az commands
type AzExecutor struct{}

// This line ensures AzExecutor implements the CommandExecutor interface
var _ tools.CommandExecutor = (*AzExecutor)(nil)

// NewExecutor creates a new AzExecutor instance
func NewExecutor() *AzExecutor {
	return &AzExecutor{}
}

// loginOnce ensures we only attempt automated az login once per process
var loginOnce sync.Once
var loginErr error

const azLoginPrompt = "Please run 'az login'"

// isAzLoggedIn checks whether Azure CLI appears authenticated.
func isAzLoggedIn(probe *command.ShellProcess) bool {
	out, _ := probe.Run("account show --query id -o tsv")
	trimmed := strings.TrimSpace(out)
	return trimmed != "" && !strings.Contains(trimmed, azLoginPrompt)
}

// setSubscriptionIfProvided sets the active subscription if AZURE_SUBSCRIPTION_ID is present.
func setSubscriptionIfProvided(probe *command.ShellProcess) {
	if subID, ok := os.LookupEnv("AZURE_SUBSCRIPTION_ID"); ok {
		if s := strings.TrimSpace(subID); s != "" {
			_, _ = probe.Run(fmt.Sprintf("account set --subscription %s", s))
		}
	}
}

// ensureAzLogin performs an automatic 'az login --service-principal' if az appears
// to be unauthenticated AND the standard service principal environment variables
// are present. This allows the MCP server to rely on AZURE_* env vars without
// requiring a manual az login step inside the container.
// EnsureAzLogin attempts to log into Azure CLI using service principal env vars once per process.
func EnsureAzLogin(timeout int) error {
	loginOnce.Do(func() {
		// Quick probe to see if az is already logged in
		probe := command.NewShellProcess("az", timeout)
		if isAzLoggedIn(probe) {
			// Ensure requested subscription is active
			setSubscriptionIfProvided(probe)
			return
		}

		clientID, _ := os.LookupEnv("AZURE_CLIENT_ID")
		tenantID, _ := os.LookupEnv("AZURE_TENANT_ID")
		clientSecret, hasSecret := os.LookupEnv("AZURE_CLIENT_SECRET")
		// Note: subscription selection is applied after a successful login via setSubscriptionIfProvided

		// Try login methods in order of preference:
		// 1) Service Principal secret (if provided)
		if hasSecret && clientID != "" && tenantID != "" {
			lp := command.NewShellProcess("az", timeout)
			loginCmd := fmt.Sprintf("login --service-principal -u %s -p %s --tenant %s --only-show-errors", clientID, clientSecret, tenantID)
			if _, err := lp.Run(loginCmd); err != nil {
				loginErr = fmt.Errorf("automatic az login (sp secret) failed: %v", err)
				return
			}
			setSubscriptionIfProvided(probe)
			// Verify
			if !isAzLoggedIn(probe) {
				loginErr = fmt.Errorf("automatic az login (sp secret) appeared to succeed but account still unavailable")
			}
			return
		}

		// 2) Workload Identity (federated token file)
		if clientID != "" && tenantID != "" {
			if tokenFile, ok := os.LookupEnv("AZURE_FEDERATED_TOKEN_FILE"); ok && strings.TrimSpace(tokenFile) != "" {
				// Read token content
				data, err := os.ReadFile(tokenFile)
				if err == nil {
					token := strings.TrimSpace(string(data))
					if token != "" {
						lp := command.NewShellProcess("az", timeout)
						// Note: token will appear in process args; acceptable for now. Optionally, we could explore safer passing.
						loginCmd := fmt.Sprintf("login --service-principal -u %s --tenant %s --federated-token %s --only-show-errors", clientID, tenantID, token)
						if _, err := lp.Run(loginCmd); err != nil {
							loginErr = fmt.Errorf("automatic az login (federated token) failed: %v", err)
							return
						}
						setSubscriptionIfProvided(probe)
						if !isAzLoggedIn(probe) {
							loginErr = fmt.Errorf("automatic az login (federated token) appeared to succeed but account still unavailable")
						}
						return
					}
				}
			}
		}

		// 3) Managed Identity (if available)
		// Try system-assigned or user-assigned (with -u clientID) if clientID is provided
		{
			lp := command.NewShellProcess("az", timeout)
			miCmd := "login --identity --only-show-errors"
			if clientID != "" {
				miCmd = fmt.Sprintf("login --identity -u %s --only-show-errors", clientID)
			}
			if _, err := lp.Run(miCmd); err == nil {
				setSubscriptionIfProvided(probe)
				if !isAzLoggedIn(probe) {
					loginErr = fmt.Errorf("automatic az login (managed identity) appeared to succeed but account still unavailable")
				}
				return
			}
		}

		// If none of the methods applied, leave loginErr nil so caller can surface CLI message
	})
	return loginErr
}

// Execute handles general az command execution
func (e *AzExecutor) Execute(params map[string]interface{}, cfg *config.ConfigData) (string, error) {
	azCmd, ok := params["command"].(string)
	if !ok {
		return "", fmt.Errorf("invalid command parameter")
	}

	// Attempt automatic login before executing any az command
	_ = EnsureAzLogin(cfg.Timeout)

	// Validate the command against security settings
	validator := security.NewValidator(cfg.SecurityConfig)
	err := validator.ValidateCommand(azCmd, security.CommandTypeAz)
	if err != nil {
		return "", err
	}

	// Extract binary name and arguments from command
	cmdParts := strings.Fields(azCmd)
	if len(cmdParts) == 0 {
		return "", fmt.Errorf("empty command")
	}

	// Use the first part as the binary name
	binaryName := cmdParts[0]

	// The rest of the command becomes the arguments
	cmdArgs := ""
	if len(cmdParts) > 1 {
		cmdArgs = strings.Join(cmdParts[1:], " ")
	}

	// If the command is not an az command, return an error
	if binaryName != "az" {
		return "", fmt.Errorf("command must start with 'az'")
	}

	// Execute the command
	process := command.NewShellProcess(binaryName, cfg.Timeout)
	return process.Run(cmdArgs)
}

// ExecuteSpecificCommand executes a specific az command with the given arguments
func (e *AzExecutor) ExecuteSpecificCommand(cmd string, params map[string]interface{}, cfg *config.ConfigData) (string, error) {
	args, ok := params["args"].(string)
	if !ok {
		args = ""
	}

	fullCmd := cmd
	if args != "" {
		fullCmd += " " + args
	}

	// Validate the command against security settings
	validator := security.NewValidator(cfg.SecurityConfig)
	err := validator.ValidateCommand(fullCmd, security.CommandTypeAz)
	if err != nil {
		return "", err
	}

	// Extract binary name from command (should be "az")
	cmdParts := strings.Fields(fullCmd)
	if len(cmdParts) == 0 {
		return "", fmt.Errorf("empty command")
	}

	// Use the first part as the binary name
	binaryName := cmdParts[0]

	// The rest of the command becomes the arguments
	cmdArgs := ""
	if len(cmdParts) > 1 {
		cmdArgs = strings.Join(cmdParts[1:], " ")
	}

	// If the command is not an az command, return an error
	if binaryName != "az" {
		return "", fmt.Errorf("command must start with 'az'")
	}

	// Attempt automatic login before executing any az command
	_ = EnsureAzLogin(cfg.Timeout)

	// Execute the command
	process := command.NewShellProcess(binaryName, cfg.Timeout)
	return process.Run(cmdArgs)
}

// CreateCommandExecutorFunc creates a CommandExecutor for a specific az command
func CreateCommandExecutorFunc(cmd string) tools.CommandExecutorFunc {
	f := func(params map[string]interface{}, cfg *config.ConfigData) (string, error) {
		executor := NewExecutor()
		return executor.ExecuteSpecificCommand(cmd, params, cfg)
	}
	return tools.CommandExecutorFunc(f)
}
