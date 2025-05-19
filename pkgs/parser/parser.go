package parser

import (
	"fmt"
	"strings"

	"github.com/aledsdavies/devcmd/internal/gen"
	"github.com/antlr4-go/antlr/v4"
)

//go:generate bash -c "cd ../../grammar && antlr -Dlanguage=Go -package gen -o ../internal/gen devcmd.g4"

// ParseError represents an error that occurred during parsing
type ParseError struct {
	Line    int    // The line number where the error occurred
	Column  int    // The column number where the error occurred
	Message string // The error message
	Context string // The line of text where the error occurred
}

// Error formats the parse error as a string with visual context
func (e *ParseError) Error() string {
	if e.Context == "" {
		return fmt.Sprintf("line %d: %s", e.Line, e.Message)
	}

	// Create a visual error indicator with arrow pointing to error position
	pointer := strings.Repeat(" ", e.Column) + "^"

	return fmt.Sprintf("line %d: %s\n%s\n%s",
		e.Line,
		e.Message,
		e.Context,
		pointer)
}

// NewParseError creates a new ParseError without context
func NewParseError(line int, format string, args ...interface{}) *ParseError {
	return &ParseError{
		Line:    line,
		Message: fmt.Sprintf(format, args...),
	}
}

// NewDetailedParseError creates a ParseError with context information
func NewDetailedParseError(line int, column int, context string, format string, args ...interface{}) *ParseError {
	return &ParseError{
		Line:    line,
		Column:  column,
		Context: context,
		Message: fmt.Sprintf(format, args...),
	}
}

// Parse parses a command file content into a CommandFile structure
func Parse(content string) (*CommandFile, error) {
	// Ensure content has a trailing newline for consistent parsing
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}

	// Split the content into lines for error reporting
	lines := strings.Split(content, "\n")

	// Create input stream from the content
	input := antlr.NewInputStream(content)

	// Create lexer with error handling
	lexer := gen.NewdevcmdLexer(input)
	errorListener := &ErrorCollector{
		lines: lines,
	}
	lexer.RemoveErrorListeners()
	lexer.AddErrorListener(errorListener)

	// Create token stream and parser
	tokens := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := gen.NewdevcmdParser(tokens)
	parser.RemoveErrorListeners()
	parser.AddErrorListener(errorListener)

	// DO NOT use BailErrorStrategy as it's causing the panic
	// We're now using the default error strategy which is more robust

	// Parse the input
	tree := parser.Program()

	// Check for syntax errors
	if errorListener.HasErrors() {
		return nil, errorListener.Error()
	}

	// Create a CommandFile to store the parsing results
	commandFile := &CommandFile{
		Lines:       lines,
		Definitions: []Definition{},
		Commands:    []Command{},
	}

	// Use visitor to extract commands and definitions
	visitor := &DevcmdVisitor{
		commandFile: commandFile,
		tokenStream: tokens,
		inputStream: input,
	}
	visitor.Visit(tree)

	// Verify no duplicate definitions
	defs := make(map[string]int)
	for _, def := range commandFile.Definitions {
		if line, exists := defs[def.Name]; exists {
			defLine := lines[def.Line-1]
			return nil, NewDetailedParseError(def.Line, strings.Index(defLine, def.Name), defLine,
				"duplicate definition of '%s' (previously defined at line %d)",
				def.Name, line)
		}
		defs[def.Name] = def.Line
	}

	// Verify no duplicate commands
	cmds := make(map[string]int)
	for _, cmd := range commandFile.Commands {
		if line, exists := cmds[cmd.Name]; exists {
			cmdLine := lines[cmd.Line-1]
			return nil, NewDetailedParseError(cmd.Line, strings.Index(cmdLine, cmd.Name), cmdLine,
				"duplicate command '%s' (previously defined at line %d)",
				cmd.Name, line)
		}
		cmds[cmd.Name] = cmd.Line
	}

	// Perform semantic validation of the command file
	if err := Validate(commandFile); err != nil {
		return nil, err
	}

	return commandFile, nil
}

// ErrorCollector collects syntax errors during parsing
type ErrorCollector struct {
	antlr.DefaultErrorListener
	errors []SyntaxError
	lines  []string // Store the original source lines
}

// SyntaxError represents a syntax error with location information
type SyntaxError struct {
	Line    int
	Column  int
	Message string
}

// SyntaxError is called by ANTLR when a syntax error is encountered
func (e *ErrorCollector) SyntaxError(recognizer antlr.Recognizer, offendingSymbol interface{}, line, column int, msg string, ex antlr.RecognitionException) {
	e.errors = append(e.errors, SyntaxError{
		Line:    line,
		Column:  column,
		Message: msg,
	})
}

// HasErrors returns true if syntax errors were found
func (e *ErrorCollector) HasErrors() bool {
	return len(e.errors) > 0
}

// Error returns a ParseError for the first syntax error
func (e *ErrorCollector) Error() error {
	if len(e.errors) == 0 {
		return nil
	}

	err := e.errors[0]

	// Get the line context if available
	var context string
	if err.Line > 0 && err.Line <= len(e.lines) {
		context = e.lines[err.Line-1]
	}

	if context != "" {
		return NewDetailedParseError(err.Line, err.Column, context, "%s", err.Message)
	} else {
		return NewParseError(err.Line, "syntax error at column %d: %s", err.Column, err.Message)
	}
}

// DevcmdVisitor implements the visitor pattern for traversing the parse tree
type DevcmdVisitor struct {
	commandFile *CommandFile
	tokenStream antlr.TokenStream
	inputStream antlr.CharStream
}

// Visit is the entry point for the visitor pattern
func (v *DevcmdVisitor) Visit(tree antlr.ParseTree) {
	switch t := tree.(type) {
	case *gen.ProgramContext:
		v.visitProgram(t)
	case *gen.LineContext:
		v.visitLine(t)
	case *gen.VariableDefinitionContext:
		v.visitVariableDefinition(t)
	case *gen.CommandDefinitionContext:
		v.visitCommandDefinition(t)
	case antlr.TerminalNode:
		// Skip terminal nodes
	default:
		// Visit children for other node types
		for i := 0; i < tree.GetChildCount(); i++ {
			child := tree.GetChild(i)
			// Type assertion to convert antlr.Tree to antlr.ParseTree
			if parseTree, ok := child.(antlr.ParseTree); ok {
				v.Visit(parseTree)
			}
		}
	}
}

// visitProgram processes the root program node
func (v *DevcmdVisitor) visitProgram(ctx *gen.ProgramContext) {
	for _, line := range ctx.AllLine() {
		v.Visit(line)
	}
}

// visitLine processes a line node
func (v *DevcmdVisitor) visitLine(ctx *gen.LineContext) {
	if varDef := ctx.VariableDefinition(); varDef != nil {
		v.Visit(varDef)
	} else if cmdDef := ctx.CommandDefinition(); cmdDef != nil {
		v.Visit(cmdDef)
	}
	// Skip NEWLINE-only lines
}

// visitVariableDefinition processes a variable definition
func (v *DevcmdVisitor) visitVariableDefinition(ctx *gen.VariableDefinitionContext) {
	name := ctx.NAME().GetText()
	cmdText := ctx.CommandText()
	value := v.getOriginalText(cmdText)
	line := ctx.GetStart().GetLine()

	v.commandFile.Definitions = append(v.commandFile.Definitions, Definition{
		Name:  name,
		Value: value,
		Line:  line,
	})
}

// visitCommandDefinition processes a command definition
func (v *DevcmdVisitor) visitCommandDefinition(ctx *gen.CommandDefinitionContext) {
	name := ctx.NAME().GetText()
	line := ctx.GetStart().GetLine()

	// Check modifiers
	isWatch := ctx.WATCH() != nil
	isStop := ctx.STOP() != nil

	if simpleCmd := ctx.SimpleCommand(); simpleCmd != nil {
		// Process simple command
		cmd := v.processSimpleCommand(simpleCmd.(*gen.SimpleCommandContext))

		v.commandFile.Commands = append(v.commandFile.Commands, Command{
			Name:    name,
			Command: cmd,
			Line:    line,
			IsWatch: isWatch,
			IsStop:  isStop,
		})
	} else if blockCmd := ctx.BlockCommand(); blockCmd != nil {
		// Process block command
		blockStatements := v.processBlockCommand(blockCmd.(*gen.BlockCommandContext))

		v.commandFile.Commands = append(v.commandFile.Commands, Command{
			Name:    name,
			Line:    line,
			IsWatch: isWatch,
			IsStop:  isStop,
			IsBlock: true,
			Block:   blockStatements,
		})
	}
}

// processSimpleCommand extracts text from a simple command
func (v *DevcmdVisitor) processSimpleCommand(ctx *gen.SimpleCommandContext) string {
	// Get main text
	cmdText := v.getOriginalText(ctx.CommandText())
	cmdText = strings.TrimRight(cmdText, " \t") // keep tail blanks only for continuations

	// Process continuations
	var fullText strings.Builder
	fullText.WriteString(cmdText)

	for _, contLine := range ctx.AllContinuationLine() {
		contCtx := contLine.(*gen.ContinuationLineContext)
		contText := v.getOriginalText(contCtx.CommandText())
		contText = strings.TrimLeft(contText, " \t") // strip leading blanks only
		fullText.WriteString(" ")                    // Add a single space for continuation
		fullText.WriteString(contText)
	}

	return fullText.String()
}

// processBlockCommand extracts statements from a block command
func (v *DevcmdVisitor) processBlockCommand(ctx *gen.BlockCommandContext) []BlockStatement {
	var statements []BlockStatement

	blockStmts := ctx.BlockStatements()
	if blockStmts == nil {
		return statements
	}

	nonEmptyStmts := blockStmts.(*gen.BlockStatementsContext).NonEmptyBlockStatements()
	if nonEmptyStmts == nil {
		return statements // Empty block
	}

	// Process each statement
	nonEmptyCtx := nonEmptyStmts.(*gen.NonEmptyBlockStatementsContext)
	for _, stmt := range nonEmptyCtx.AllBlockStatement() {
		stmtCtx := stmt.(*gen.BlockStatementContext)
		command := v.getOriginalText(stmtCtx.CommandText())
		background := stmtCtx.AMPERSAND() != nil

		statements = append(statements, BlockStatement{
			Command:    command,
			Background: background,
		})
	}

	return statements
}

// getOriginalText extracts the original source text for a rule context
func (v *DevcmdVisitor) getOriginalText(ctx antlr.ParserRuleContext) string {
	if ctx == nil {
		return ""
	}

	start := ctx.GetStart().GetStart()
	stop := ctx.GetStop().GetStop()

	if start < 0 || stop < 0 || start > stop {
		return ""
	}

	text := v.inputStream.GetText(start, stop)

	// If this is a command text, trim leading whitespace
	if _, ok := ctx.(*gen.CommandTextContext); ok {
		text = strings.TrimLeft(text, " \t")
	}

	return text
}
