package parser

import (
	"fmt"
	"strings"
)

// ParseError represents an error that occurred during parsing
type ParseError struct {
	Line    int    // The line number where the error occurred
	Message string // The error message
}

// Error formats the parse error as a string
func (e *ParseError) Error() string {
	return fmt.Sprintf("line %d: %s", e.Line, e.Message)
}

// NewParseError creates a new ParseError
func NewParseError(line int, format string, args ...interface{}) *ParseError {
	return &ParseError{
		Line:    line,
		Message: fmt.Sprintf(format, args...),
	}
}

// ValidationError checks if commands and definitions are valid
type ValidationError struct {
	Errors []string
}

// Error formats all validation errors as a single string
func (e *ValidationError) Error() string {
	return strings.Join(e.Errors, "\n")
}

// NewValidationError creates a new ValidationError
func NewValidationError(errors []string) *ValidationError {
	return &ValidationError{
		Errors: errors,
	}
}

// Add adds a new error message to the validation error
func (e *ValidationError) Add(format string, args ...interface{}) {
	e.Errors = append(e.Errors, fmt.Sprintf(format, args...))
}

// HasErrors returns true if there are validation errors
func (e *ValidationError) HasErrors() bool {
	return len(e.Errors) > 0
}

// Validate performs semantic validation on a command file
func Validate(file *CommandFile) error {
	validationError := &ValidationError{}

	// Create variable name lookup
	varNames := make(map[string]bool)
	for _, def := range file.Definitions {
		varNames[def.Name] = true
	}

	// 1. Check for matching watch/stop commands
	watchCmds := make(map[string]int)
	stopCmds := make(map[string]int)

	for _, cmd := range file.Commands {
		name := strings.TrimPrefix(cmd.Name, ".")

		if cmd.IsWatch {
			watchCmds[name] = cmd.Line
		}

		if cmd.IsStop {
			stopCmds[name] = cmd.Line
		}
	}

	// Check that every watch has a matching stop
	for name, line := range watchCmds {
		if _, ok := stopCmds[name]; !ok {
			validationError.Add("watch command '%s' at line %d has no matching stop command",
				name, line)
		}
	}

	// Check that every stop has a matching watch
	for name, line := range stopCmds {
		if _, ok := watchCmds[name]; !ok {
			validationError.Add("stop command '%s' at line %d has no matching watch command",
				name, line)
		}
	}

	// 2. Check for variable references in command text
	checkVarReferences := func(text string, line int) {
		// Find all $(var) references in text
		var inVar bool
		var varName strings.Builder

		for i := 0; i < len(text); i++ {
			if !inVar {
				if i+1 < len(text) && text[i] == '$' && text[i+1] == '(' {
					inVar = true
					varName.Reset()
					i++ // Skip the '('
				}
			} else {
				if text[i] == ')' {
					// End of variable reference
					name := varName.String()
					if !varNames[name] {
						validationError.Add("undefined variable '%s' at line %d", name, line)
					}
					inVar = false
				} else {
					varName.WriteByte(text[i])
				}
			}
		}

		if inVar {
			validationError.Add("unclosed variable reference '$(...)' at line %d", line)
		}
	}

	// Check variables in commands
	for _, cmd := range file.Commands {
		if !cmd.IsBlock {
			checkVarReferences(cmd.Command, cmd.Line)
		} else {
			for _, stmt := range cmd.Block {
				checkVarReferences(stmt.Command, cmd.Line)
			}
		}
	}

	if validationError.HasErrors() {
		return validationError
	}

	return nil
}
