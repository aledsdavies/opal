package decorators

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/aledsdavies/devcmd/core/ast"
	"github.com/aledsdavies/devcmd/core/plan"
	"github.com/aledsdavies/devcmd/runtime/decorators"
	"github.com/aledsdavies/devcmd/runtime/execution"
)

// ConfirmDecorator implements the @confirm decorator for user confirmation prompts
type ConfirmDecorator struct{}

// Template for confirmation logic code generation (unified contract: statement blocks)
const confirmExecutionTemplate = `// Confirmation execution setup
{{if .SkipInCI}}// Check if we're in CI environment (using captured environment)
isCI := func() bool {
	ciVars := []string{"CI", "CONTINUOUS_INTEGRATION", "GITHUB_ACTIONS", "TRAVIS", "CIRCLECI", "JENKINS_URL", "GITLAB_CI", "BUILDKITE", "BUILD_NUMBER"}
	for _, envVar := range ciVars {
		if value, exists := envContext[envVar]; exists && value != "" {
			return true
		}
	}
	return false
}()

if isCI {
	// Auto-confirm in CI and execute commands
	fmt.Printf("CI environment detected - auto-confirming: %s\n", {{printf "%q" .Message}})
} else {
{{end}}	// Display the confirmation message
	fmt.Print({{printf "%q" .Message}})
	{{if .DefaultYes}}fmt.Print(" [Y/n]: "){{else}}fmt.Print(" [y/N]: "){{end}}
	
	// Read user input
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read user input: %w", err)
	}
	
	response = strings.TrimSpace(response)
	
	// Determine if user confirmed
	confirmed := false
	if response == "" {
		confirmed = {{.DefaultYes}}
	} else {
		{{if .CaseSensitive}}confirmed = response == "y" || response == "Y" || response == "yes" || response == "Yes"{{else}}lowerResponse := strings.ToLower(response)
		confirmed = lowerResponse == "y" || lowerResponse == "yes"{{end}}
	}
	
	if !confirmed {
		{{if .AbortOnNo}}return fmt.Errorf("user cancelled execution"){{else}}return nil{{end}}
	}
{{if .SkipInCI}}}{{end}}

// Execute the commands in child context
{{range $i, $cmd := .Commands}}
{{generateShellCode $cmd}}
{{end}}`

// Name returns the decorator name
func (c *ConfirmDecorator) Name() string {
	return "confirm"
}

// Description returns a human-readable description
func (c *ConfirmDecorator) Description() string {
	return "Prompt user for confirmation before executing commands"
}

// ParameterSchema returns the expected parameters for this decorator
func (c *ConfirmDecorator) ParameterSchema() []decorators.ParameterSchema {
	return []decorators.ParameterSchema{
		{
			Name:        "message",
			Type:        ast.StringType,
			Required:    false,
			Description: "Message to display to the user (default: 'Do you want to continue?')",
		},
		{
			Name:        "defaultYes",
			Type:        ast.BooleanType,
			Required:    false,
			Description: "Default to yes if user just presses enter (default: false)",
		},
		{
			Name:        "abortOnNo",
			Type:        ast.BooleanType,
			Required:    false,
			Description: "Abort execution if user says no (default: true)",
		},
		{
			Name:        "caseSensitive",
			Type:        ast.BooleanType,
			Required:    false,
			Description: "Make y/n matching case sensitive (default: false)",
		},
		{
			Name:        "ci",
			Type:        ast.BooleanType,
			Required:    false,
			Description: "Skip confirmation in CI environments (checks CI env var, default: true)",
		},
	}
}

// Validate checks if the decorator usage is correct during parsing

// ImportRequirements returns the dependencies needed for code generation
func (c *ConfirmDecorator) ImportRequirements() decorators.ImportRequirement {
	return decorators.ImportRequirement{
		StandardLibrary: []string{"bufio", "fmt", "os", "strings"},
	}
}

// isCI checks if we're running in a CI environment using captured environment
func (c *ConfirmDecorator) isCI(ctx execution.BaseContext) bool {
	// Check common CI environment variables from captured environment
	ciVars := []string{
		"CI",                     // Most CI systems
		"CONTINUOUS_INTEGRATION", // Legacy/alternate
		"GITHUB_ACTIONS",         // GitHub Actions
		"TRAVIS",                 // Travis CI
		"CIRCLECI",               // Circle CI
		"JENKINS_URL",            // Jenkins
		"GITLAB_CI",              // GitLab CI
		"BUILDKITE",              // Buildkite
		"BUILD_NUMBER",           // Generic build systems
	}

	for _, envVar := range ciVars {
		if value, exists := ctx.GetEnv(envVar); exists && value != "" {
			return true
		}
	}
	return false
}

// trackCIEnvironmentVariables tracks CI environment variables for code generation
func (c *ConfirmDecorator) trackCIEnvironmentVariables(ctx execution.GeneratorContext) {
	// Track all CI environment variables so they're included in global envContext
	ciVars := []string{
		"CI", "CONTINUOUS_INTEGRATION", "GITHUB_ACTIONS", "TRAVIS",
		"CIRCLECI", "JENKINS_URL", "GITLAB_CI", "BUILDKITE", "BUILD_NUMBER",
	}

	for _, envVar := range ciVars {
		ctx.TrackEnvironmentVariable(envVar, "")
	}
}

// ExecuteInterpreter executes confirmation prompt in interpreter mode
func (c *ConfirmDecorator) ExecuteInterpreter(ctx execution.InterpreterContext, params []ast.NamedParameter, content []ast.CommandContent) *execution.ExecutionResult {
	message, defaultYes, abortOnNo, caseSensitive, skipInCI := c.extractConfirmParams(params)
	return c.executeInterpreterImpl(ctx, message, defaultYes, abortOnNo, caseSensitive, skipInCI, content)
}

// ExecuteGenerator generates Go code for confirmation logic
func (c *ConfirmDecorator) ExecuteGenerator(ctx execution.GeneratorContext, params []ast.NamedParameter, content []ast.CommandContent) *execution.ExecutionResult {
	message, defaultYes, abortOnNo, caseSensitive, skipInCI := c.extractConfirmParams(params)
	return c.executeGeneratorImpl(ctx, message, defaultYes, abortOnNo, caseSensitive, skipInCI, content)
}

// ExecutePlan creates a plan element for dry-run mode
func (c *ConfirmDecorator) ExecutePlan(ctx execution.PlanContext, params []ast.NamedParameter, content []ast.CommandContent) *execution.ExecutionResult {
	message, defaultYes, abortOnNo, caseSensitive, skipInCI := c.extractConfirmParams(params)
	return c.executePlanImpl(ctx, message, defaultYes, abortOnNo, caseSensitive, skipInCI, content)
}

// extractConfirmParams extracts and validates confirmation parameters
func (c *ConfirmDecorator) extractConfirmParams(params []ast.NamedParameter) (string, bool, bool, bool, bool) {
	message := ast.GetStringParam(params, "message", "Do you want to continue?")
	defaultYes := ast.GetBoolParam(params, "defaultYes", false)
	abortOnNo := ast.GetBoolParam(params, "abortOnNo", true)
	caseSensitive := ast.GetBoolParam(params, "caseSensitive", false)
	skipInCI := ast.GetBoolParam(params, "ci", true)
	
	return message, defaultYes, abortOnNo, caseSensitive, skipInCI
}

// executeInterpreterImpl executes confirmation prompt in interpreter mode
func (c *ConfirmDecorator) executeInterpreterImpl(ctx execution.InterpreterContext, message string, defaultYes, abortOnNo, caseSensitive, skipInCI bool, content []ast.CommandContent) *execution.ExecutionResult {
	// Check if we should skip confirmation in CI environment
	if skipInCI && c.isCI(ctx) {
		// Auto-confirm in CI and execute commands in child context
		fmt.Printf("CI environment detected - auto-confirming: %s\n", message)
		confirmCtx := ctx.Child()
		for _, cmd := range content {
			switch c := cmd.(type) {
			case *ast.ShellContent:
				result := confirmCtx.ExecuteShell(c)
				if result.Error != nil {
					return &execution.ExecutionResult{
						Data:  nil,
						Error: fmt.Errorf("command execution failed: %w", result.Error),
					}
				}
			default:
				return &execution.ExecutionResult{
					Data:  nil,
					Error: fmt.Errorf("unsupported command content type in confirm: %T", cmd),
				}
			}
		}
		return &execution.ExecutionResult{
			Data:  nil,
			Error: nil,
		}
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
		return &execution.ExecutionResult{
			Data:  nil,
			Error: fmt.Errorf("failed to read user input: %w", err),
		}
	}

	response = strings.TrimSpace(response)

	// Determine if user confirmed
	confirmed := false
	if response == "" {
		confirmed = defaultYes
	} else {
		if caseSensitive {
			confirmed = response == "y" || response == "Y" || response == "yes" || response == "Yes"
		} else {
			lowerResponse := strings.ToLower(response)
			confirmed = lowerResponse == "y" || lowerResponse == "yes"
		}
	}

	if !confirmed {
		if abortOnNo {
			return &execution.ExecutionResult{
				Data:  nil,
				Error: fmt.Errorf("user cancelled execution"),
			}
		}
		// User said no but don't abort - just skip execution
		return &execution.ExecutionResult{
			Data:  nil,
			Error: nil,
		}
	}

	// User confirmed, execute the commands in child context
	confirmCtx := ctx.Child()
	for _, cmd := range content {
		switch c := cmd.(type) {
		case *ast.ShellContent:
			result := confirmCtx.ExecuteShell(c)
			if result.Error != nil {
				return &execution.ExecutionResult{
					Data:  nil,
					Error: fmt.Errorf("command execution failed: %w", result.Error),
				}
			}
		default:
			return &execution.ExecutionResult{
				Data:  nil,
				Error: fmt.Errorf("unsupported command content type in confirm: %T", cmd),
			}
		}
	}

	return &execution.ExecutionResult{
		Data:  nil,
		Error: nil,
	}
}

// executeGeneratorImpl generates Go code for confirmation logic
func (c *ConfirmDecorator) executeGeneratorImpl(ctx execution.GeneratorContext, message string, defaultYes, abortOnNo, caseSensitive, skipInCI bool, content []ast.CommandContent) *execution.ExecutionResult {
	// Track CI environment variables for deterministic behavior
	if skipInCI {
		c.trackCIEnvironmentVariables(ctx)
	}
	
	// Create child context for isolated execution
	confirmCtx := ctx.Child()

	// Use template to generate the full confirmation logic
	tmpl, err := template.New("confirmExecution").Funcs(confirmCtx.GetTemplateFunctions()).Parse(confirmExecutionTemplate)
	if err != nil {
		return &execution.ExecutionResult{
			Data:  "",
			Error: fmt.Errorf("failed to parse confirm template: %w", err),
		}
	}

	templateData := struct {
		Message       string
		DefaultYes    bool
		AbortOnNo     bool
		CaseSensitive bool
		SkipInCI      bool
		Commands      []ast.CommandContent
	}{
		Message:       message,
		DefaultYes:    defaultYes,
		AbortOnNo:     abortOnNo,
		CaseSensitive: caseSensitive,
		SkipInCI:      skipInCI,
		Commands:      content,
	}

	var result strings.Builder
	if err := tmpl.Execute(&result, templateData); err != nil {
		return &execution.ExecutionResult{
			Data:  "",
			Error: fmt.Errorf("failed to execute confirm template: %w", err),
		}
	}

	return &execution.ExecutionResult{
		Data:  result.String(),
		Error: nil,
	}
}

// executePlanImpl creates a plan element for dry-run mode
func (c *ConfirmDecorator) executePlanImpl(ctx execution.PlanContext, message string, defaultYes, abortOnNo, caseSensitive, skipInCI bool, content []ast.CommandContent) *execution.ExecutionResult {
	// Context-aware planning: check current environment
	var description string

	if skipInCI && c.isCI(ctx) {
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

	element := plan.Decorator("confirm").
		WithType("block").
		WithParameter("message", message).
		WithDescription(description)

	if defaultYes {
		element = element.WithParameter("defaultYes", "true")
	}
	if !abortOnNo {
		element = element.WithParameter("abortOnNo", "false")
	}
	if caseSensitive {
		element = element.WithParameter("caseSensitive", "true")
	}
	if !skipInCI {
		element = element.WithParameter("ci", "false")
	}

	return &execution.ExecutionResult{
		Data:  element,
		Error: nil,
	}
}

// init registers the confirm decorator
func init() {
	decorators.RegisterBlock(&ConfirmDecorator{})
}
