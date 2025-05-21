package parser

// BlockStatement represents a statement within a block command
// It captures both the command text and whether it should run in background
type BlockStatement struct {
	Command    string // The command text to execute
	Background bool   // Whether the command should run in background (with &)
}

// Definition represents a variable definition in the command file
type Definition struct {
	Name  string // The variable name
	Value string // The variable value
	Line  int    // The line number in the source file
}

// Command represents a command definition in the command file
type Command struct {
	Name    string           // The command name
	Command string           // The command text for simple commands
	Line    int              // The line number in the source file
	IsWatch bool             // Whether this is a watch command
	IsStop  bool             // Whether this is a stop command
	IsBlock bool             // Whether this is a block command
	Block   []BlockStatement // The statements for block commands
}

// CommandFile represents the parsed command file
type CommandFile struct {
	Definitions []Definition // All variable definitions
	Commands    []Command    // All command definitions
	Lines       []string     // Original file lines for error reporting
}

// ExpandVariables expands variable references in commands
func (cf *CommandFile) ExpandVariables() error {
	// Create lookup map for variables
	vars := make(map[string]string)
	for _, def := range cf.Definitions {
		vars[def.Name] = def.Value
	}

	// Expand variables in simple commands
	for i := range cf.Commands {
		cmd := &cf.Commands[i]
		if !cmd.IsBlock {
			expanded, err := expandVariablesInText(cmd.Command, vars, cmd.Line)
			if err != nil {
				return err
			}
			cmd.Command = expanded
		} else {
			// Expand variables in block statements
			for j := range cmd.Block {
				stmt := &cmd.Block[j]
				expanded, err := expandVariablesInText(stmt.Command, vars, cmd.Line)
				if err != nil {
					return err
				}
				stmt.Command = expanded
			}
		}
	}

	return nil
}

// expandVariablesInText replaces $(name) in a string with its value
func expandVariablesInText(text string, vars map[string]string, line int) (string, error) {
	var result []byte
	var varName []byte
	inVar := false
	escapeNext := false

	for i := 0; i < len(text); i++ {
		if escapeNext {
			// When a character is escaped, just output it as-is
			result = append(result, text[i])
			escapeNext = false
			continue
		}

		if text[i] == '\\' {
			// Next character will be escaped
			escapeNext = true
			continue
		}

		if !inVar {
			// Look for variable start
			if i+1 < len(text) && text[i] == '$' && text[i+1] == '(' {
				inVar = true
				varName = varName[:0] // Reset var name
				i++                   // Skip the '('
			} else {
				result = append(result, text[i])
			}
		} else {
			// In a variable reference
			if text[i] == ')' {
				// End of variable reference
				name := string(varName)
				if value, ok := vars[name]; ok {
					result = append(result, value...)
				} else {
					return "", NewParseError(line, "undefined variable: %s", name)
				}
				inVar = false
			} else {
				varName = append(varName, text[i])
			}
		}
	}

	// Check for unclosed variable reference
	if inVar {
		return "", NewParseError(line, "unclosed variable reference: $(%s", string(varName))
	}

	return string(result), nil
}
