package decorators

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/aledsdavies/devcmd/pkgs/ast"
	"github.com/aledsdavies/devcmd/pkgs/plan"
	"github.com/aledsdavies/devcmd/pkgs/types"
)

// ConfirmDecorator implements the @confirm decorator for user confirmation prompts
type ConfirmDecorator struct{}

func init() {
	// Register the decorator with the global registry
	globalRegistry.RegisterBlock(&ConfirmDecorator{})
}

// Name returns the decorator name
func (c *ConfirmDecorator) Name() string {
	return "confirm"
}

// Description returns a human-readable description
func (c *ConfirmDecorator) Description() string {
	return "Prompt user for confirmation before executing commands"
}

// ParameterSchema returns the expected parameters for this decorator
func (c *ConfirmDecorator) ParameterSchema() []ParameterSchema {
	return []ParameterSchema{
		{
			Name:        "message",
			Type:        types.StringType,
			Required:    false,
			Description: "Message to display to the user (default: 'Do you want to continue?')",
		},
		{
			Name:        "defaultYes",
			Type:        types.BooleanType,
			Required:    false,
			Description: "Default to yes if user just presses enter (default: false)",
		},
		{
			Name:        "abortOnNo",
			Type:        types.BooleanType,
			Required:    false,
			Description: "Abort execution if user says no (default: true)",
		},
		{
			Name:        "caseSensitive",
			Type:        types.BooleanType,
			Required:    false,
			Description: "Make y/n matching case sensitive (default: false)",
		},
		{
			Name:        "ci",
			Type:        types.BooleanType,
			Required:    false,
			Description: "Skip confirmation in CI environments (checks CI env var, default: true)",
		},
	}
}

// Validate checks if the decorator usage is correct during parsing
func (c *ConfirmDecorator) Validate(ctx *ExecutionContext, params []ast.NamedParameter) error {
	// All parameters are optional, so no validation needed
	return nil
}

// ImportRequirements returns the dependencies needed for code generation
func (c *ConfirmDecorator) ImportRequirements() ImportRequirement {
	return ImportRequirement{
		StandardLibrary: []string{"bufio", "fmt", "os", "strings"},
	}
}

// isCI checks if we're running in a CI environment
func isCI() bool {
	// Check common CI environment variables
	ciVars := []string{
		"CI",                    // Most CI systems
		"CONTINUOUS_INTEGRATION", // Legacy/alternate
		"GITHUB_ACTIONS",        // GitHub Actions
		"TRAVIS",               // Travis CI
		"CIRCLECI",             // Circle CI
		"JENKINS_URL",          // Jenkins
		"GITLAB_CI",            // GitLab CI
		"BUILDKITE",            // Buildkite
		"BUILD_NUMBER",         // Generic build systems
	}
	
	for _, envVar := range ciVars {
		if os.Getenv(envVar) != "" {
			return true
		}
	}
	return false
}

// Run executes the decorator at runtime with the given command content
func (c *ConfirmDecorator) Run(ctx *ExecutionContext, params []ast.NamedParameter, content []ast.CommandContent) error {
	message := ast.GetStringParam(params, "message", "Do you want to continue?")
	defaultYes := ast.GetBoolParam(params, "defaultYes", false)
	abortOnNo := ast.GetBoolParam(params, "abortOnNo", true)
	caseSensitive := ast.GetBoolParam(params, "caseSensitive", false)
	skipInCI := ast.GetBoolParam(params, "ci", true)

	// Check if we should skip confirmation in CI environment
	if skipInCI && isCI() {
		// Silently proceed in CI - no output, execute commands manually for now
		fmt.Printf("CI environment detected - auto-confirming: %s\n", message)
		for i, cmd := range content {
			fmt.Printf("  Command %d: %+v\n", i, cmd)
		}
		return nil
	}

	// Display the confirmation message
	fmt.Print(message)
	if defaultYes {
		fmt.Print(" [Y/n]: ")
	} else {
		fmt.Print(" [y/N]: ")
	}

	// Read user input
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read user input: %w", err)
	}

	response = strings.TrimSpace(response)
	
	// Handle empty response (use default)
	if response == "" {
		if !defaultYes && abortOnNo {
			return fmt.Errorf("user cancelled execution")
		}
		if !defaultYes {
			// Default is no, but don't abort - just skip
			return nil
		}
		// Default is yes, continue execution
	} else {
		// Check the response
		var confirmed bool
		if caseSensitive {
			confirmed = response == "y" || response == "Y" || response == "yes" || response == "Yes"
		} else {
			lowerResponse := strings.ToLower(response)
			confirmed = lowerResponse == "y" || lowerResponse == "yes"
		}

		if !confirmed {
			if abortOnNo {
				return fmt.Errorf("user cancelled execution")
			}
			// User said no but don't abort - just skip execution
			return nil
		}
	}

	// User confirmed, execute the content manually for now
	fmt.Printf("User confirmed - executing %d commands\n", len(content))
	for i, cmd := range content {
		fmt.Printf("  Command %d: %+v\n", i, cmd)
	}
	return nil
}

// Generate produces Go code for the decorator in compiled mode
func (c *ConfirmDecorator) Generate(ctx *ExecutionContext, params []ast.NamedParameter, content []ast.CommandContent) (string, error) {
	// For now, return a placeholder - full implementation would be complex
	return "// @confirm decorator code generation not yet implemented\n", nil
}

// Plan creates a plan element describing what this decorator would do in dry run mode
func (c *ConfirmDecorator) Plan(ctx *ExecutionContext, params []ast.NamedParameter, content []ast.CommandContent) (plan.PlanElement, error) {
	message := ast.GetStringParam(params, "message", "Do you want to continue?")
	defaultYes := ast.GetBoolParam(params, "defaultYes", false)
	abortOnNo := ast.GetBoolParam(params, "abortOnNo", true)
	skipInCI := ast.GetBoolParam(params, "ci", true)

	// Context-aware planning: check current environment
	var description string
	
	if skipInCI && isCI() {
		// We're in CI and should skip confirmation
		description = fmt.Sprintf("🤖 CI Environment Detected - Auto-confirming: %s", message)
	} else {
		// Interactive mode - show what user will see
		var prompt string
		if defaultYes {
			prompt = fmt.Sprintf("%s [Y/n]", message)
		} else {
			prompt = fmt.Sprintf("%s [y/N]", message)
		}

		var behavior string
		if abortOnNo {
			behavior = "execution will abort if user declines"
		} else {
			behavior = "execution will skip if user declines"
		}

		description = fmt.Sprintf("🤔 User Prompt: %s (%s)", prompt, behavior)
	}

	return plan.Decorator("confirm").
		WithType("block").
		WithParameter("message", message).
		WithParameter("defaultYes", defaultYes).
		WithParameter("abortOnNo", abortOnNo).
		WithParameter("ci", skipInCI).
		WithDescription(description), nil
}