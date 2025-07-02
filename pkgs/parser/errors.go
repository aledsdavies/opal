package parser

import (
	"fmt"
	"strings"

	"github.com/aledsdavies/devcmd/pkgs/lexer"
)

// addError adds a parse error with user-friendly context
func (p *Parser) addError(token lexer.Token, message, context, hint string) {
	if len(p.errors) >= p.config.MaxErrors {
		return // Stop collecting errors if we hit the limit
	}

	p.errors = append(p.errors, ParseError{
		Type:    SyntaxError,
		Token:   token,
		Message: message,
		Context: context,
		Hint:    hint,
	})
}

// addSemanticError adds a semantic error (like undefined variables)
func (p *Parser) addSemanticError(token lexer.Token, message, context, hint string) {
	if len(p.errors) >= p.config.MaxErrors {
		return
	}

	p.errors = append(p.errors, ParseError{
		Type:    SemanticError,
		Token:   token,
		Message: message,
		Context: context,
		Hint:    hint,
	})
}

// addReferenceError adds a reference error (like undefined variable reference)
func (p *Parser) addReferenceError(token lexer.Token, message, context, hint string) {
	if len(p.errors) >= p.config.MaxErrors {
		return
	}

	p.errors = append(p.errors, ParseError{
		Type:    ReferenceError,
		Token:   token,
		Message: message,
		Context: context,
		Hint:    hint,
	})
}

// FormatErrors returns user-friendly formatted error messages
func FormatErrors(errors []ParseError, sourceLines []string) string {
	if len(errors) == 0 {
		return ""
	}

	var result strings.Builder

	// Group errors by line for better presentation
	errorsByLine := groupErrorsByLine(errors)

	for lineNum, lineErrors := range errorsByLine {
		// Show the source line if available
		if lineNum > 0 && lineNum <= len(sourceLines) {
			sourceLine := sourceLines[lineNum-1]

			// Add some spacing for readability
			result.WriteString("\n")
			result.WriteString(fmt.Sprintf("%4d | %s\n", lineNum, sourceLine))

			// Show error indicators under the source line
			for _, err := range lineErrors {
				indicator := formatErrorIndicator(err, sourceLine)
				if indicator != "" {
					result.WriteString(indicator)
				}
			}
		}

		// Show error messages
		for _, err := range lineErrors {
			result.WriteString(formatCompilerError(err))
			result.WriteString("\n")
		}
	}

	return result.String()
}

// formatCompilerError formats a single error in compiler style
func formatCompilerError(err ParseError) string {
	// Format: filename:line:column: error: message
	position := fmt.Sprintf("%d:%d", err.Token.Line, err.Token.Column)

	var prefix string
	switch err.Type {
	case SyntaxError:
		prefix = "error"
	case SemanticError:
		prefix = "error"
	case DuplicateError:
		prefix = "error"
	case ReferenceError:
		prefix = "error"
	}

	message := fmt.Sprintf("     %s: %s: %s", position, prefix, err.Message)

	if err.Hint != "" {
		message += fmt.Sprintf("\n     note: %s", err.Hint)
	}

	return message
}

// formatErrorIndicator creates a visual indicator pointing to the error
func formatErrorIndicator(err ParseError, sourceLine string) string {
	if err.Token.Column <= 0 || err.Token.Column > len(sourceLine)+1 {
		return ""
	}

	// Create an indicator line pointing to the error location
	indicator := strings.Repeat(" ", 7) // Account for line number prefix "   1 | "

	// Add spaces up to the error column, handling tabs properly
	for i := 1; i < err.Token.Column; i++ {
		if i <= len(sourceLine) && sourceLine[i-1] == '\t' {
			indicator += "\t"
		} else {
			indicator += " "
		}
	}

	// Choose indicator symbol based on error type and severity
	symbol := "^"
	switch err.Type {
	case SyntaxError:
		symbol = "^"
	case SemanticError:
		symbol = "~"
	case DuplicateError:
		symbol = "^"
	case ReferenceError:
		symbol = "^"
	}

	indicator += symbol

	// Add additional indicators for multi-character tokens
	tokenLength := len(err.Token.Value)
	if tokenLength > 1 {
		indicator += strings.Repeat("~", tokenLength-1)
	}

	return indicator + "\n"
}

// FormatErrorsSimple returns a simple list of error messages
func FormatErrorsSimple(errors []ParseError) string {
	if len(errors) == 0 {
		return ""
	}

	var result strings.Builder
	for i, err := range errors {
		if i > 0 {
			result.WriteString("\n")
		}
		result.WriteString(formatCompilerError(err))
	}

	return result.String()
}

// GetErrorSuggestions returns helpful suggestions based on common errors
func GetErrorSuggestions(errors []ParseError) []string {
	suggestions := []string{}
	seenSuggestions := make(map[string]bool)

	for _, err := range errors {
		suggestion := getSuggestionForError(err)
		if suggestion != "" && !seenSuggestions[suggestion] {
			suggestions = append(suggestions, suggestion)
			seenSuggestions[suggestion] = true
		}
	}

	return suggestions
}

// Helper functions for error formatting

func groupErrorsByLine(errors []ParseError) map[int][]ParseError {
	groups := make(map[int][]ParseError)

	for _, err := range errors {
		line := err.Token.Line
		groups[line] = append(groups[line], err)
	}

	return groups
}

func getSuggestionForError(err ParseError) string {
	switch err.Type {
	case SyntaxError:
		return getSyntaxErrorSuggestion(err)
	case SemanticError:
		return getSemanticErrorSuggestion(err)
	case DuplicateError:
		return getDuplicateErrorSuggestion(err)
	case ReferenceError:
		return getReferenceErrorSuggestion(err)
	}
	return ""
}

func getSyntaxErrorSuggestion(err ParseError) string {
	message := strings.ToLower(err.Message)

	if strings.Contains(message, "expected ':'") {
		return "Commands must have a colon after the name: 'command-name: command-body'"
	}

	if strings.Contains(message, "expected '='") {
		return "Variables must have an equals sign: 'var NAME = value'"
	}

	if strings.Contains(message, "unclosed") {
		if strings.Contains(message, "brace") {
			return "Every '{' must have a matching '}' - check your block commands"
		}
		if strings.Contains(message, "paren") {
			return "Every '(' must have a matching ')' - check your decorators and @var() references"
		}
		if strings.Contains(message, "string") {
			return "Every quote must be closed - check your string literals"
		}
	}

	if strings.Contains(message, "invalid") && (strings.Contains(message, "command name") || strings.Contains(message, "variable name")) {
		return "Names must start with a letter and contain only letters, numbers, hyphens, and underscores"
	}

	return ""
}

func getSemanticErrorSuggestion(err ParseError) string {
	message := strings.ToLower(err.Message)

	if strings.Contains(message, "undefined variable") {
		return "Declare variables before using them: 'var VARIABLE_NAME = value'"
	}

	return ""
}

func getDuplicateErrorSuggestion(err ParseError) string {
	message := strings.ToLower(err.Message)

	if strings.Contains(message, "variable") {
		return "Each variable can only be declared once - use different names or remove the duplicate"
	}

	if strings.Contains(message, "command") {
		return "Each command can only be declared once - use different names or combine the commands"
	}

	return ""
}

func getReferenceErrorSuggestion(err ParseError) string {
	message := strings.ToLower(err.Message)

	if strings.Contains(message, "undefined variable") {
		return "Make sure to declare the variable before using it with @var()"
	}

	if strings.Contains(message, "invalid decorator") {
		return "Valid decorators include: @timeout, @retry, @parallel, @sh, @env, @cwd, @confirm, @debounce"
	}

	return ""
}

// FormatErrorReport creates a comprehensive error report
func FormatErrorReport(errors []ParseError, sourceLines []string) string {
	if len(errors) == 0 {
		return "✅ No parse errors found!"
	}

	var report strings.Builder

	// Summary in compiler style
	report.WriteString(fmt.Sprintf("error: found %d error(s)\n", len(errors)))

	// Detailed errors
	report.WriteString(FormatErrors(errors, sourceLines))

	// Suggestions section
	suggestions := GetErrorSuggestions(errors)
	if len(suggestions) > 0 {
		report.WriteString("\nhelp: common fixes:\n")
		for i, suggestion := range suggestions {
			if i >= 3 { // Limit to top 3 suggestions
				break
			}
			report.WriteString(fmt.Sprintf("  • %s\n", suggestion))
		}
	}

	return report.String()
}

// ContextualErrorMessages provides context-aware error messages
type ContextualErrorMessages struct {
	commonMistakes map[string]string
	validExamples  map[string][]string
}

func NewContextualErrorMessages() *ContextualErrorMessages {
	return &ContextualErrorMessages{
		commonMistakes: map[string]string{
			"missing_colon":           "Commands need a colon: 'build: echo hello'",
			"missing_equals":          "Variables need equals: 'var SRC = ./src'",
			"invalid_var_syntax":      "Use @var(NAME) not @var NAME or @NAME",
			"invalid_decorator":       "Decorators start with @ and use parentheses: @timeout(30s)",
			"unclosed_brace":          "Block commands need closing brace: { command1; command2 }",
			"unclosed_paren":          "Decorators need closing parenthesis: @retry(3)",
			"space_in_command_name":   "Command names cannot have spaces - use hyphens: 'build-all'",
			"number_start_name":       "Names cannot start with numbers - use letters: 'server1' not '1server'",
			"missing_var_declaration": "Declare variables before using: 'var NAME = value' then '@var(NAME)'",
			"duplicate_declaration":   "Each name can only be used once per type (but watch/stop can share names)",
		},
		validExamples: map[string][]string{
			"variable_declaration": {
				"var SRC = ./src",
				"var PORT = 8080",
				"var ( SRC = ./src; PORT = 8080 )",
			},
			"command_declaration": {
				"build: echo hello",
				"watch server: npm start",
				"stop server: pkill node",
			},
			"block_command": {
				"setup: { npm install; npm run build }",
				"services: @parallel { server; client }",
			},
			"decorators": {
				"@timeout(30s) { long-running-task }",
				"@retry(3) { flaky-command }",
				"@sh(echo hello && echo world)",
			},
			"variable_reference": {
				"build: cd @var(SRC)",
				"serve: go run @var(MAIN) --port=@var(PORT)",
			},
		},
	}
}

func (c *ContextualErrorMessages) GetHelpForError(err ParseError) string {
	message := strings.ToLower(err.Message)

	// Map error messages to help categories
	if strings.Contains(message, "expected ':'") {
		return c.getHelpText("missing_colon", "command_declaration")
	}

	if strings.Contains(message, "expected '='") {
		return c.getHelpText("missing_equals", "variable_declaration")
	}

	if strings.Contains(message, "unclosed") && strings.Contains(message, "brace") {
		return c.getHelpText("unclosed_brace", "block_command")
	}

	if strings.Contains(message, "unclosed") && strings.Contains(message, "paren") {
		return c.getHelpText("unclosed_paren", "decorators")
	}

	if strings.Contains(message, "undefined variable") {
		return c.getHelpText("missing_var_declaration", "variable_reference")
	}

	if strings.Contains(message, "invalid decorator") {
		return c.getHelpText("invalid_decorator", "decorators")
	}

	if strings.Contains(message, "duplicate") {
		return c.getHelpText("duplicate_declaration", "")
	}

	return ""
}

func (c *ContextualErrorMessages) getHelpText(mistakeKey, exampleKey string) string {
	help := c.commonMistakes[mistakeKey]

	if exampleKey != "" && len(c.validExamples[exampleKey]) > 0 {
		help += "\n\nValid examples:"
		for _, example := range c.validExamples[exampleKey] {
			help += "\n  " + example
		}
	}

	return help
}

// QuickFix suggests automated fixes for common errors
type QuickFix struct {
	Description string
	LineNumber  int
	OldText     string
	NewText     string
	Confidence  float64 // 0.0 to 1.0
}

func GenerateQuickFixes(errors []ParseError, sourceLines []string) []QuickFix {
	fixes := []QuickFix{}

	for _, err := range errors {
		if err.Token.Line <= 0 || err.Token.Line > len(sourceLines) {
			continue
		}

		line := sourceLines[err.Token.Line-1]
		fix := generateQuickFixForError(err, line)
		if fix != nil {
			fixes = append(fixes, *fix)
		}
	}

	return fixes
}

func generateQuickFixForError(err ParseError, sourceLine string) *QuickFix {
	message := strings.ToLower(err.Message)

	// Missing colon fix
	if strings.Contains(message, "expected ':'") {
		// Find the command name and suggest adding colon
		words := strings.Fields(sourceLine)
		if len(words) > 0 {
			commandName := words[0]
			if !strings.HasSuffix(commandName, ":") {
				return &QuickFix{
					Description: "Add missing colon after command name",
					LineNumber:  err.Token.Line,
					OldText:     commandName,
					NewText:     commandName + ":",
					Confidence:  0.9,
				}
			}
		}
	}

	// Missing equals fix
	if strings.Contains(message, "expected '='") {
		// Look for pattern: var NAME value
		if strings.HasPrefix(strings.TrimSpace(sourceLine), "var ") {
			parts := strings.Fields(sourceLine)
			if len(parts) >= 3 {
				return &QuickFix{
					Description: "Add missing equals sign in variable declaration",
					LineNumber:  err.Token.Line,
					OldText:     fmt.Sprintf("var %s %s", parts[1], parts[2]),
					NewText:     fmt.Sprintf("var %s = %s", parts[1], parts[2]),
					Confidence:  0.8,
				}
			}
		}
	}

	// Unclosed parenthesis fix
	if strings.Contains(message, "unclosed") && strings.Contains(message, "paren") {
		openCount := strings.Count(sourceLine, "(")
		closeCount := strings.Count(sourceLine, ")")
		if openCount > closeCount {
			return &QuickFix{
				Description: "Add missing closing parenthesis",
				LineNumber:  err.Token.Line,
				OldText:     sourceLine,
				NewText:     sourceLine + ")",
				Confidence:  0.7,
			}
		}
	}

	return nil
}

// Enhanced error reporting for devcmd-specific patterns

func FormatDevcmdError(err ParseError, sourceLine string) string {
	var report strings.Builder

	// Standard error formatting
	report.WriteString(formatCompilerError(err))

	// Add context-specific help
	contextualMsgs := NewContextualErrorMessages()
	help := contextualMsgs.GetHelpForError(err)
	if help != "" {
		report.WriteString("\n     help: " + help)
	}

	return report.String()
}

// Validation helpers for user input

func ValidateCommandName(name string) error {
	if len(name) == 0 {
		return fmt.Errorf("command name cannot be empty")
	}

	if !IsValidIdentifier(name) {
		return fmt.Errorf("invalid command name '%s' - must start with letter and contain only letters, numbers, hyphens, and underscores", name)
	}

	// Check for reserved words
	reserved := []string{"var", "watch", "stop", "if", "then", "else", "fi", "for", "do", "done", "while"}
	for _, word := range reserved {
		if name == word {
			return fmt.Errorf("'%s' is a reserved word and cannot be used as a command name", name)
		}
	}

	return nil
}

func ValidateVariableName(name string) error {
	if len(name) == 0 {
		return fmt.Errorf("variable name cannot be empty")
	}

	if !IsValidIdentifier(name) {
		return fmt.Errorf("invalid variable name '%s' - must start with letter and contain only letters, numbers, hyphens, and underscores", name)
	}

	return nil
}

func ValidateDecoratorName(name string) error {
	validDecorators := []string{"timeout", "retry", "parallel", "sh", "env", "cwd", "confirm", "debounce", "watch-files"}

	for _, valid := range validDecorators {
		if name == valid {
			return nil
		}
	}

	return fmt.Errorf("invalid decorator '@%s' - valid decorators are: %s",
		name, strings.Join(validDecorators, ", "))
}

// IsValidIdentifier checks if a string is a valid devcmd identifier
func IsValidIdentifier(name string) bool {
	if len(name) == 0 {
		return false
	}

	// Must start with letter or underscore
	first := name[0]
	if !((first >= 'a' && first <= 'z') || (first >= 'A' && first <= 'Z') || first == '_') {
		return false
	}

	// Rest can be letters, digits, underscores, or hyphens
	for i := 1; i < len(name); i++ {
		ch := name[i]
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
			 (ch >= '0' && ch <= '9') || ch == '_' || ch == '-') {
			return false
		}
	}

	return true
}
