package interpreter

import (
	"context"
	"fmt"

	"github.com/aledsdavies/devcmd/pkgs/ast"
	"github.com/aledsdavies/devcmd/pkgs/decorators"
	"github.com/aledsdavies/devcmd/pkgs/engine"
)

// Interpreter handles direct execution of commands from CLI files using the execution engine
type Interpreter struct {
	engine *engine.Engine
	ctx    *decorators.ExecutionContext
	debug  bool
}

// New creates a new interpreter for the given program
func New(program *ast.Program, debug bool) *Interpreter {
	ctx := decorators.NewExecutionContext(context.Background(), program)
	ctx.Debug = debug
	
	return &Interpreter{
		engine: engine.New(engine.InterpreterMode, ctx),
		ctx:    ctx,
		debug:  debug,
	}
}

// RunCommand executes a specific command by name
func (i *Interpreter) RunCommand(commandName string, args []string) error {
	// Find the command to execute
	var targetCommand *ast.CommandDecl
	for idx := range i.ctx.Program.Commands {
		if i.ctx.Program.Commands[idx].Name == commandName {
			targetCommand = &i.ctx.Program.Commands[idx]
			break
		}
	}

	if targetCommand == nil {
		// List available commands
		var availableCommands []string
		for _, command := range i.ctx.Program.Commands {
			availableCommands = append(availableCommands, command.Name)
		}
		return fmt.Errorf("command '%s' not found. Available commands: %v", commandName, availableCommands)
	}

	// Process variables first
	if err := i.processVariables(); err != nil {
		return fmt.Errorf("failed to process variables: %w", err)
	}

	// Execute the specific command using the engine
	cmdResult, err := i.engine.ExecuteCommand(targetCommand)
	if err != nil {
		return fmt.Errorf("failed to execute command %s: %w", commandName, err)
	}

	if i.debug {
		fmt.Printf("Command %s completed with status: %s\n", commandName, cmdResult.Status)
		for _, output := range cmdResult.Output {
			fmt.Printf("  %s\n", output)
		}
	}

	if cmdResult.Status == "failed" {
		return fmt.Errorf("command %s failed: %s", commandName, cmdResult.Error)
	}

	return nil
}

// processVariables processes all variables and adds them to the execution context
func (i *Interpreter) processVariables() error {
	// Process individual variables
	for _, varDecl := range i.ctx.Program.Variables {
		value := i.getVariableValue(varDecl.Value)
		i.ctx.SetVariable(varDecl.Name, value)
	}
	
	// Process variables from groups
	for _, varGroup := range i.ctx.Program.VarGroups {
		for _, varDecl := range varGroup.Variables {
			value := i.getVariableValue(varDecl.Value)
			i.ctx.SetVariable(varDecl.Name, value)
		}
	}
	
	return nil
}

// getVariableValue extracts the string value from a variable expression
func (i *Interpreter) getVariableValue(value ast.Expression) string {
	return value.String()
}