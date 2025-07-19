package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/aledsdavies/devcmd/pkgs/ast"
	"github.com/aledsdavies/devcmd/pkgs/generator"
	"github.com/aledsdavies/devcmd/pkgs/parser"
	"github.com/spf13/cobra"
)

// Build-time variables - can be set via ldflags
var (
	Version   string = "dev"
	BuildTime string = "unknown"
	GitCommit string = "unknown"
)

// Global flags
var (
	commandsFile string
	templateFile string
	binaryName   string
	output       string
	debug        bool
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "devcmd [flags]",
	Short: "Generate Go CLI applications from command definitions",
	Long: `devcmd generates standalone Go CLI executables from simple command definition files.
It reads .cli files containing command definitions and outputs Go source code or compiled binaries.
By default, it looks for commands.cli in the current directory.`,
	Args: cobra.NoArgs,
	RunE: generateCommand,
}

var buildCmd = &cobra.Command{
	Use:   "build [flags]",
	Short: "Build CLI binary from command definitions",
	Long: `Build a compiled Go CLI binary from command definitions.
This generates the Go source code and compiles it into an executable binary.
By default, it looks for commands.cli in the current directory.`,
	Args: cobra.NoArgs,
	RunE: buildCommand,
}

var runCmd = &cobra.Command{
	Use:   "run <command> [args...]",
	Short: "Run a command directly from command definitions",
	Long: `Execute a command directly from the CLI file without compilation.
This interprets and runs the command immediately, useful for development and testing.
By default, it looks for commands.cli in the current directory.`,
	Args: cobra.MinimumNArgs(1),
	RunE: runCommand,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  "Display version, build time, and git commit information for devcmd.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("devcmd %s\n", Version)
		fmt.Printf("Built: %s\n", BuildTime)
		fmt.Printf("Commit: %s\n", GitCommit)
	},
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVarP(&commandsFile, "file", "f", "commands.cli", "Path to commands file")
	rootCmd.PersistentFlags().StringVar(&templateFile, "template", "", "Custom template file for generation")
	rootCmd.PersistentFlags().StringVar(&binaryName, "binary", "dev", "Binary name for the generated CLI")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug output")

	// Build command specific flags
	buildCmd.Flags().StringVarP(&output, "output", "o", "", "Output binary path (default: ./<binary-name>)")

	// Add subcommands
	rootCmd.AddCommand(buildCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(versionCmd)
}

func generateCommand(cmd *cobra.Command, args []string) error {
	// Read command file
	content, err := os.ReadFile(commandsFile)
	if err != nil {
		return fmt.Errorf("error reading file %s: %w", commandsFile, err)
	}

	// Parse the command definitions
	program, err := parser.Parse(string(content))
	if err != nil {
		return fmt.Errorf("error parsing commands: %w", err)
	}

	// Generate Go output
	output, err := generateGo(program, templateFile, binaryName)
	if err != nil {
		return fmt.Errorf("error generating Go output: %w", err)
	}

	// Output the result
	fmt.Print(output)
	return nil
}

func buildCommand(cmd *cobra.Command, args []string) error {
	// Read and parse command file
	content, err := os.ReadFile(commandsFile)
	if err != nil {
		return fmt.Errorf("error reading file %s: %w", commandsFile, err)
	}

	program, err := parser.Parse(string(content))
	if err != nil {
		return fmt.Errorf("error parsing commands: %w", err)
	}

	// Generate Go source code
	goSource, err := generateGo(program, templateFile, binaryName)
	if err != nil {
		return fmt.Errorf("error generating Go source: %w", err)
	}

	// Determine output path
	outputPath := output
	if outputPath == "" {
		outputPath = "./" + binaryName
	}
	
	// Make output path absolute
	if !filepath.IsAbs(outputPath) {
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("error getting working directory: %w", err)
		}
		outputPath = filepath.Join(wd, outputPath)
	}

	// Create temporary directory for build
	tempDir, err := os.MkdirTemp("", "devcmd-build-*")
	if err != nil {
		return fmt.Errorf("error creating temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Write Go source to temp directory
	mainGoPath := filepath.Join(tempDir, "main.go")
	if err := os.WriteFile(mainGoPath, []byte(goSource), 0644); err != nil {
		return fmt.Errorf("error writing Go source: %w", err)
	}

	// Create go.mod file
	moduleName := strings.ReplaceAll(binaryName, "-", "_")
	goModContent := fmt.Sprintf("module %s\n\ngo 1.21\n", moduleName)
	goModPath := filepath.Join(tempDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		return fmt.Errorf("error writing go.mod: %w", err)
	}

	// Build the binary
	buildCmd := exec.Command("go", "build", "-o", outputPath, ".")
	buildCmd.Dir = tempDir
	buildCmd.Stderr = os.Stderr

	if debug {
		fmt.Fprintf(os.Stderr, "Building binary: %s\n", outputPath)
		buildCmd.Stdout = os.Stderr
	}

	if err := buildCmd.Run(); err != nil {
		return fmt.Errorf("error building binary: %w", err)
	}

	if debug {
		fmt.Fprintf(os.Stderr, "✅ Successfully built: %s\n", outputPath)
	}

	return nil
}

func runCommand(cmd *cobra.Command, args []string) error {
	commandName := args[0]
	commandArgs := args[1:]

	// Read and parse command file
	content, err := os.ReadFile(commandsFile)
	if err != nil {
		return fmt.Errorf("error reading file %s: %w", commandsFile, err)
	}

	program, err := parser.Parse(string(content))
	if err != nil {
		return fmt.Errorf("error parsing commands: %w", err)
	}

	// Find the command to execute
	var targetCommand *ast.CommandDecl
	for i := range program.Commands {
		if program.Commands[i].Name == commandName {
			targetCommand = &program.Commands[i]
			break
		}
	}

	if targetCommand == nil {
		// List available commands
		var availableCommands []string
		for _, command := range program.Commands {
			availableCommands = append(availableCommands, command.Name)
		}
		return fmt.Errorf("command '%s' not found. Available commands: %v", commandName, availableCommands)
	}

	// Create variable definitions map for expansion
	definitions := createDefinitionMapFromProgram(program)

	// Execute the command
	return executeCommand(targetCommand, definitions, commandArgs)
}

// createDefinitionMapFromProgram creates a map of variable definitions
func createDefinitionMapFromProgram(program *ast.Program) map[string]string {
	definitions := make(map[string]string)
	
	// Add individual variables
	for _, varDecl := range program.Variables {
		definitions[varDecl.Name] = getVariableValue(varDecl.Value)
	}
	
	// Add variables from groups
	for _, varGroup := range program.VarGroups {
		for _, varDecl := range varGroup.Variables {
			definitions[varDecl.Name] = getVariableValue(varDecl.Value)
		}
	}
	
	return definitions
}

// getVariableValue extracts the string value from a variable
func getVariableValue(value ast.Expression) string {
	return value.String()
}

// executeCommand executes a single command with variable expansion
func executeCommand(command *ast.CommandDecl, definitions map[string]string, args []string) error {
	if len(command.Body.Content) == 0 {
		return fmt.Errorf("command '%s' has no body", command.Name)
	}

	// Execute each content item in the command body
	for _, content := range command.Body.Content {
		if shellContent, ok := content.(*ast.ShellContent); ok {
			// Convert shell content to string and expand variables
			shellCommand := shellContent.String()
			expandedCmd := expandVariables(shellCommand, definitions)
			
			if debug {
				fmt.Fprintf(os.Stderr, "Executing: %s\n", expandedCmd)
			}
			
			// Execute the shell command
			execCmd := exec.Command("sh", "-c", expandedCmd)
			execCmd.Stdout = os.Stdout
			execCmd.Stderr = os.Stderr
			execCmd.Stdin = os.Stdin
			
			if err := execCmd.Run(); err != nil {
				if exitError, ok := err.(*exec.ExitError); ok {
					os.Exit(exitError.ExitCode())
				}
				return fmt.Errorf("command failed: %w", err)
			}
		}
	}
	
	return nil
}

// expandVariables expands @var(NAME) patterns in a string
func expandVariables(input string, definitions map[string]string) string {
	result := input
	for name, value := range definitions {
		pattern := "@var(" + name + ")"
		result = strings.ReplaceAll(result, pattern, value)
	}
	return result
}

// generateGo generates Go CLI output
func generateGo(program *ast.Program, templateFile string, binaryName string) (string, error) {
	if templateFile != "" {
		templateContent, err := os.ReadFile(templateFile)
		if err != nil {
			return "", fmt.Errorf("error reading template file: %w", err)
		}
		return generator.GenerateGoWithTemplate(program, string(templateContent))
	}
	return generator.GenerateGoWithBinaryName(program, binaryName)
}
