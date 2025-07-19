package interpreter

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/aledsdavies/devcmd/pkgs/ast"
)

// Interpreter handles direct execution of commands from CLI files
type Interpreter struct {
	program *ast.Program
	debug   bool
}

// New creates a new interpreter for the given program
func New(program *ast.Program, debug bool) *Interpreter {
	return &Interpreter{
		program: program,
		debug:   debug,
	}
}

// RunCommand executes a specific command by name
func (i *Interpreter) RunCommand(commandName string, args []string) error {
	// Find the command to execute
	var targetCommand *ast.CommandDecl
	for idx := range i.program.Commands {
		if i.program.Commands[idx].Name == commandName {
			targetCommand = &i.program.Commands[idx]
			break
		}
	}

	if targetCommand == nil {
		// List available commands
		var availableCommands []string
		for _, command := range i.program.Commands {
			availableCommands = append(availableCommands, command.Name)
		}
		return fmt.Errorf("command '%s' not found. Available commands: %v", commandName, availableCommands)
	}

	// Create variable definitions map for expansion
	definitions := i.createDefinitionMap()

	// Execute the command
	return i.executeCommand(targetCommand, definitions, args)
}

// createDefinitionMap creates a map of variable definitions
func (i *Interpreter) createDefinitionMap() map[string]string {
	definitions := make(map[string]string)
	
	// Add individual variables
	for _, varDecl := range i.program.Variables {
		definitions[varDecl.Name] = i.getVariableValue(varDecl.Value)
	}
	
	// Add variables from groups
	for _, varGroup := range i.program.VarGroups {
		for _, varDecl := range varGroup.Variables {
			definitions[varDecl.Name] = i.getVariableValue(varDecl.Value)
		}
	}
	
	return definitions
}

// getVariableValue extracts the string value from a variable
func (i *Interpreter) getVariableValue(value ast.Expression) string {
	return value.String()
}

// executeCommand executes a single command with variable expansion
// NOTE: This is a basic implementation that only handles shell content.
// Decorators like @watch, @stop, @parallel are not supported in interpreted mode.
func (i *Interpreter) executeCommand(command *ast.CommandDecl, definitions map[string]string, args []string) error {
	if len(command.Body.Content) == 0 {
		return fmt.Errorf("command '%s' has no body", command.Name)
	}

	// Execute each content item in the command body
	for _, content := range command.Body.Content {
		if shellContent, ok := content.(*ast.ShellContent); ok {
			// Convert shell content to string and expand variables
			shellCommand := shellContent.String()
			expandedCmd := i.expandVariables(shellCommand, definitions)
			
			if i.debug {
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
		} else {
			// Skip non-shell content (decorators, etc.) with a warning in debug mode
			if i.debug {
				fmt.Fprintf(os.Stderr, "Skipping non-shell content (decorators not supported in interpreted mode): %T\n", content)
			}
		}
	}
	
	return nil
}

// expandVariables expands @var(NAME) patterns in a string
func (i *Interpreter) expandVariables(input string, definitions map[string]string) string {
	result := input
	for name, value := range definitions {
		pattern := "@var(" + name + ")"
		result = strings.ReplaceAll(result, pattern, value)
	}
	return result
}