package parser

import (
	"fmt"
	"strings"

	"github.com/aledsdavies/devcmd/pkgs/ast"
	"github.com/aledsdavies/devcmd/pkgs/lexer"
)

// Parse parses devcmd source code and returns an AST
func Parse(source string) (*ast.Program, error) {
	return ParseWithConfig(source, DefaultConfig())
}

// ParseWithConfig parses with custom configuration
func ParseWithConfig(source string, config ParserConfig) (*ast.Program, error) {
	result := ParseToResult(source, config)

	if result.HasErrors() {
		// Return the first error as the primary error
		return result.Program, result.Errors[0]
	}

	return result.Program, nil
}

// ParseToResult parses and returns detailed results including all errors
func ParseToResult(source string, config ParserConfig) ParseResult {
	// Tokenize the source
	tokens, err := lexer.GetSemanticTokens(source)
	if err != nil {
		return ParseResult{
			Program: nil,
			Errors: []ParseError{{
				Type:    SyntaxError,
				Token:   lexer.Token{Line: 1, Column: 1},
				Message: "lexer error: " + err.Error(),
				Context: "tokenization",
				Hint:    "check for invalid characters or syntax",
			}},
		}
	}

	// Create parser
	parser := &Parser{
		tokens:     tokens,
		current:    0,
		structure:  &StructureMap{},
		errors:     []ParseError{},
		config:     config,
		decorators: make(map[int]*ast.Decorator),
	}

	// Two-pass parsing
	program := parser.parse()

	return ParseResult{
		Program: program,
		Errors:  parser.errors,
	}
}

// ParseWithErrorReport parses and returns a formatted error report
func ParseWithErrorReport(source string) (*ast.Program, string) {
	sourceLines := strings.Split(source, "\n")
	result := ParseToResult(source, DefaultConfig())

	if result.HasErrors() {
		errorReport := FormatErrorReport(result.Errors, sourceLines)
		return result.Program, errorReport
	}

	return result.Program, ""
}

// ValidateOnly checks for errors without building a full AST
func ValidateOnly(source string) []ParseError {
	result := ParseToResult(source, ParserConfig{
		MaxErrors:          100,
		StrictMode:         true,
		AllowUndefinedVars: false,
	})

	return result.Errors
}

// Main parser implementation

// parse performs the two-pass parsing process
func (p *Parser) parse() *ast.Program {
	// Pass 1: Preprocessing - structural analysis
	p.preprocessTokens()

	// Build fast lookup maps for Pass 2
	p.buildDecoratorNodes()

	// Pass 2: AST construction
	program := p.buildAST()

	return program
}

// Utility functions for external use

// GetSourceContext returns lines of source around an error for display
func GetSourceContext(source string, lineNumber int, contextLines int) []string {
	lines := strings.Split(source, "\n")

	start := lineNumber - contextLines - 1
	end := lineNumber + contextLines

	if start < 0 {
		start = 0
	}
	if end > len(lines) {
		end = len(lines)
	}

	context := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		context = append(context, lines[i])
	}

	return context
}

// GetValidDecorators returns a list of valid decorator names
func GetValidDecorators() []string {
	return []string{
		"timeout",
		"retry",
		"parallel",
		"sh",
		"env",
		"cwd",
		"confirm",
		"debounce",
		"watch-files",
		"var", // @var is now just another decorator
	}
}

// ExplainError provides detailed explanation for error codes
func ExplainError(err ParseError) string {
	contextualMsgs := NewContextualErrorMessages()
	return contextualMsgs.GetHelpForError(err)
}

// SuggestFix returns quick fix suggestions for an error
func SuggestFix(err ParseError, sourceLine string) *QuickFix {
	return generateQuickFixForError(err, sourceLine)
}

// Debug utilities

// DebugTokens returns a formatted view of tokens for debugging
func DebugTokens(source string) string {
	tokens, err := lexer.GetSemanticTokens(source)
	if err != nil {
		return "Error tokenizing: " + err.Error()
	}

	var result strings.Builder
	result.WriteString("Tokens:\n")

	for i, token := range tokens {
		result.WriteString(fmt.Sprintf("%3d: %-15s %q at %d:%d\n",
			i, token.Type.String(), token.Value, token.Line, token.Column))
	}

	return result.String()
}

// DebugStructure returns a formatted view of the structure map
func DebugStructure(source string) string {
	result := ParseToResult(source, DefaultConfig())

	parser := &Parser{
		tokens:    make([]lexer.Token, 0),
		structure: &StructureMap{},
		config:    DefaultConfig(),
	}

	tokens, _ := lexer.GetSemanticTokens(source)
	parser.tokens = tokens
	parser.preprocessTokens()

	var debug strings.Builder
	debug.WriteString("Structure Map:\n\n")

	debug.WriteString("Variables:\n")
	for i, varSpan := range parser.structure.Variables {
		debug.WriteString(fmt.Sprintf("  %d: %s = tokens[%d:%d] (grouped: %v)\n",
			i, varSpan.NameToken.Value, varSpan.ValueStart, varSpan.ValueEnd, varSpan.IsGrouped))
	}

	debug.WriteString("\nCommands:\n")
	for i, cmdSpan := range parser.structure.Commands {
		cmdType := "command"
		if cmdSpan.TypeToken.Type == lexer.WATCH {
			cmdType = "watch"
		} else if cmdSpan.TypeToken.Type == lexer.STOP {
			cmdType = "stop"
		}
		debug.WriteString(fmt.Sprintf("  %d: %s %s body[%d:%d] (block: %v, decorators: %v)\n",
			i, cmdType, cmdSpan.NameToken.Value, cmdSpan.BodyStart, cmdSpan.BodyEnd,
			cmdSpan.IsBlock, len(cmdSpan.Decorators)))
	}

	debug.WriteString("\nDecorators:\n")
	for i, decorator := range parser.structure.Decorators {
		debug.WriteString(fmt.Sprintf("  %d: @%s hasArgs=%v hasBlock=%v at tokens[%d:%d]\n",
			i, decorator.NameToken.Value, decorator.HasArgs, decorator.HasBlock,
			decorator.StartIndex, decorator.EndIndex))
	}

	if len(result.Errors) > 0 {
		debug.WriteString("\nErrors:\n")
		for i, err := range result.Errors {
			debug.WriteString(fmt.Sprintf("  %d: %s\n", i, err.Error()))
		}
	}

	return debug.String()
}

// Performance utilities

// ParserStats holds statistics about parser performance
type ParserStats struct {
	TokenCount      int
	VariableCount   int
	CommandCount    int
	DecoratorCount  int
	ErrorCount      int
	PreprocessTime  int64 // nanoseconds
	BuildTime       int64 // nanoseconds
}

// ParseWithStats parses and returns performance statistics
func ParseWithStats(source string) (*ast.Program, ParserStats, error) {
	result := ParseToResult(source, DefaultConfig())

	stats := ParserStats{
		ErrorCount: len(result.Errors),
	}

	if result.Program != nil {
		stats.VariableCount = len(result.Program.Variables)
		stats.CommandCount = len(result.Program.Commands)

		// Count decorators by walking the AST
		decorators := ast.FindDecorators(result.Program)
		stats.DecoratorCount = len(decorators)
	}

	var err error
	if result.HasErrors() {
		err = result.Errors[0]
	}

	return result.Program, stats, err
}

// Batch processing utilities

// ParseMultiple parses multiple devcmd files and returns results
func ParseMultiple(sources map[string]string) map[string]ParseResult {
	results := make(map[string]ParseResult)

	for name, source := range sources {
		results[name] = ParseToResult(source, DefaultConfig())
	}

	return results
}

// ValidateMultiple validates multiple files and returns aggregated errors
func ValidateMultiple(sources map[string]string) map[string][]ParseError {
	errors := make(map[string][]ParseError)

	for name, source := range sources {
		fileErrors := ValidateOnly(source)
		if len(fileErrors) > 0 {
			errors[name] = fileErrors
		}
	}

	return errors
}

// Variable interpolation utilities

// InterpolateVariables replaces all @var() decorators in a program with their values
func InterpolateVariables(program *ast.Program) *ast.Program {
	// Build variable lookup map
	variables := make(map[string]string)
	for _, varDecl := range program.Variables {
		if stringVal, ok := varDecl.Value.(*ast.StringLiteral); ok {
			variables[varDecl.Name] = stringVal.Value
		} else {
			variables[varDecl.Name] = varDecl.Value.String()
		}
	}

	// Create a copy of the program with interpolated values
	interpolated := &ast.Program{
		Variables: program.Variables, // Keep variable declarations as-is
		Commands:  make([]ast.CommandDecl, len(program.Commands)),
		Pos:       program.Pos,
		Tokens:    program.Tokens,
	}

	// Interpolate each command
	for i, cmd := range program.Commands {
		interpolated.Commands[i] = *interpolateCommand(&cmd, variables)
	}

	return interpolated
}

// interpolateCommand interpolates variables in a single command
func interpolateCommand(cmd *ast.CommandDecl, variables map[string]string) *ast.CommandDecl {
	interpolatedCmd := *cmd // Copy command

	// Interpolate decorators
	interpolatedCmd.Decorators = make([]ast.Decorator, len(cmd.Decorators))
	for i, decorator := range cmd.Decorators {
		interpolatedCmd.Decorators[i] = *interpolateDecorator(&decorator, variables)
	}

	// Interpolate command body
	interpolatedCmd.Body = *interpolateCommandBody(&cmd.Body, variables)

	return &interpolatedCmd
}

// interpolateDecorator handles decorator interpolation
func interpolateDecorator(decorator *ast.Decorator, variables map[string]string) *ast.Decorator {
	// For @var decorators, replace with the actual value as a text element
	if decorator.Name == "var" && len(decorator.Args) > 0 {
		if identifier, ok := decorator.Args[0].(*ast.Identifier); ok {
			if value, exists := variables[identifier.Name]; exists {
				// Create a text element instead of the @var decorator
				return &ast.Decorator{
					Name: "text", // Special internal decorator for interpolated text
					Args: []ast.Expression{
						&ast.StringLiteral{
							Value: value,
							Pos:   decorator.Pos,
						},
					},
					Pos:       decorator.Pos,
					Tokens:    decorator.Tokens,
					AtToken:   decorator.AtToken,
					NameToken: decorator.NameToken,
				}
			}
		}
	}

	// For other decorators, return as-is (they might have their own interpolation logic)
	return decorator
}

// interpolateCommandBody interpolates variables in command body
func interpolateCommandBody(body *ast.CommandBody, variables map[string]string) *ast.CommandBody {
	interpolatedBody := *body // Copy body

	// Interpolate statements
	interpolatedBody.Statements = make([]ast.Statement, len(body.Statements))
	for i, stmt := range body.Statements {
		interpolatedBody.Statements[i] = interpolateStatement(stmt, variables)
	}

	return &interpolatedBody
}

// interpolateStatement interpolates variables in a statement
func interpolateStatement(stmt ast.Statement, variables map[string]string) ast.Statement {
	if shellStmt, ok := stmt.(*ast.ShellStatement); ok {
		interpolatedStmt := *shellStmt // Copy statement

		// Interpolate elements
		interpolatedStmt.Elements = make([]ast.CommandElement, 0, len(shellStmt.Elements))
		for _, element := range shellStmt.Elements {
			interpolated := interpolateCommandElement(element, variables)
			if interpolated != nil {
				interpolatedStmt.Elements = append(interpolatedStmt.Elements, interpolated)
			}
		}

		return &interpolatedStmt
	}

	return stmt
}

// interpolateCommandElement interpolates variables in command elements
func interpolateCommandElement(element ast.CommandElement, variables map[string]string) ast.CommandElement {
	if decorator, ok := element.(*ast.Decorator); ok && decorator.Name == "var" {
		// Replace @var(NAME) with actual value
		if len(decorator.Args) > 0 {
			if identifier, ok := decorator.Args[0].(*ast.Identifier); ok {
				if value, exists := variables[identifier.Name]; exists {
					return &ast.TextElement{
						Text:   value,
						Pos:    decorator.Pos,
						Tokens: decorator.Tokens,
					}
				}
			}
		}
		// If variable not found, keep the decorator (will be caught in validation)
		return decorator
	}

	if textElement, ok := element.(*ast.TextElement); ok {
		// Handle @var() within text content
		interpolatedText := interpolateTextContent(textElement.Text, variables)
		if interpolatedText != textElement.Text {
			return &ast.TextElement{
				Text:   interpolatedText,
				Pos:    textElement.Pos,
				Tokens: textElement.Tokens,
			}
		}
	}

	return element
}

// interpolateTextContent handles @var() within text strings
func interpolateTextContent(text string, variables map[string]string) string {
	result := text

	// Simple pattern replacement for @var(NAME)
	for varName, value := range variables {
		pattern := fmt.Sprintf("@var(%s)", varName)
		result = strings.ReplaceAll(result, pattern, value)
	}

	return result
}
