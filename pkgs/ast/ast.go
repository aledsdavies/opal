package ast

import (
	"fmt"
	"strings"

	"github.com/aledsdavies/devcmd/pkgs/lexer"
)

// Node represents any node in the AST
type Node interface {
	String() string
	Position() Position
	// LSP and Tree-sitter integration
	TokenRange() TokenRange
	SemanticTokens() []lexer.Token
}

// Position represents source location information
type Position struct {
	Line   int
	Column int
	// Enhanced for LSP
	Offset int // Byte offset in source
}

// TokenRange represents the span of tokens for this AST node
type TokenRange struct {
	Start lexer.Token // First token of this node
	End   lexer.Token // Last token of this node
	All   []lexer.Token // All tokens that comprise this node
}

// Program represents the root of the AST (entire devcmd file)
type Program struct {
	Variables []VariableDecl
	Commands  []CommandDecl
	Pos       Position
	Tokens    TokenRange // All tokens in the file
}

func (p *Program) String() string {
	var parts []string
	for _, v := range p.Variables {
		parts = append(parts, v.String())
	}
	for _, c := range p.Commands {
		parts = append(parts, c.String())
	}
	return strings.Join(parts, "\n")
}

func (p *Program) Position() Position {
	return p.Pos
}

func (p *Program) TokenRange() TokenRange {
	return p.Tokens
}

func (p *Program) SemanticTokens() []lexer.Token {
	return p.Tokens.All
}

// VariableDecl represents variable declarations (both individual and grouped)
type VariableDecl struct {
	Name   string
	Value  Expression // Could be string, number, identifier, etc.
	Pos    Position
	Tokens TokenRange

	// LSP-specific information
	NameToken  lexer.Token // The variable name token for go-to-definition
	ValueToken lexer.Token // The value token for hover info
}

func (v *VariableDecl) String() string {
	return fmt.Sprintf("var %s = %s", v.Name, v.Value.String())
}

func (v *VariableDecl) Position() Position {
	return v.Pos
}

func (v *VariableDecl) TokenRange() TokenRange {
	return v.Tokens
}

func (v *VariableDecl) SemanticTokens() []lexer.Token {
	// Return tokens with proper semantic highlighting
	tokens := []lexer.Token{v.NameToken, v.ValueToken}
	for _, token := range v.Tokens.All {
		// Ensure variable names are marked as SemVariable
		if token.Type == lexer.IDENTIFIER && token.Value == v.Name {
			token.Semantic = lexer.SemVariable
		}
	}
	return tokens
}

// Expression represents any expression (literals, identifiers, etc.)
type Expression interface {
	Node
	IsExpression() bool
	// LSP support for expressions
	GetType() ExpressionType
}

type ExpressionType int

const (
	StringType ExpressionType = iota
	NumberType
	DurationType
	IdentifierType
)

// StringLiteral represents string values
type StringLiteral struct {
	Value  string
	Raw    string
	Pos    Position
	Tokens TokenRange

	// LSP integration
	StringToken lexer.Token // The actual string token
}

func (s *StringLiteral) String() string {
	return s.Value
}

func (s *StringLiteral) Position() Position {
	return s.Pos
}

func (s *StringLiteral) TokenRange() TokenRange {
	return s.Tokens
}

func (s *StringLiteral) SemanticTokens() []lexer.Token {
	return []lexer.Token{s.StringToken}
}

func (s *StringLiteral) IsExpression() bool {
	return true
}

func (s *StringLiteral) GetType() ExpressionType {
	return StringType
}

// NumberLiteral represents numeric values
type NumberLiteral struct {
	Value  string
	Pos    Position
	Tokens TokenRange
	Token  lexer.Token // The number token
}

func (n *NumberLiteral) String() string {
	return n.Value
}

func (n *NumberLiteral) Position() Position {
	return n.Pos
}

func (n *NumberLiteral) TokenRange() TokenRange {
	return n.Tokens
}

func (n *NumberLiteral) SemanticTokens() []lexer.Token {
	return []lexer.Token{n.Token}
}

func (n *NumberLiteral) IsExpression() bool {
	return true
}

func (n *NumberLiteral) GetType() ExpressionType {
	return NumberType
}

// DurationLiteral represents duration values like 30s, 5m
type DurationLiteral struct {
	Value  string
	Pos    Position
	Tokens TokenRange
	Token  lexer.Token // The duration token
}

func (d *DurationLiteral) String() string {
	return d.Value
}

func (d *DurationLiteral) Position() Position {
	return d.Pos
}

func (d *DurationLiteral) TokenRange() TokenRange {
	return d.Tokens
}

func (d *DurationLiteral) SemanticTokens() []lexer.Token {
	return []lexer.Token{d.Token}
}

func (d *DurationLiteral) IsExpression() bool {
	return true
}

func (d *DurationLiteral) GetType() ExpressionType {
	return DurationType
}

// CommandDecl represents command definitions
type CommandDecl struct {
	Name       string
	Type       CommandType // Command, WatchCommand, StopCommand
	Decorators []Decorator
	Body       CommandBody // Unified command body
	Pos        Position
	Tokens     TokenRange

	// LSP support
	TypeToken lexer.Token // The var/watch/stop token
	NameToken lexer.Token // The command name token
}

func (c *CommandDecl) String() string {
	var parts []string

	// Add decorators
	for _, decorator := range c.Decorators {
		parts = append(parts, decorator.String())
	}

	// Add command declaration
	typeStr := ""
	switch c.Type {
	case WatchCommand:
		typeStr = "watch "
	case StopCommand:
		typeStr = "stop "
	case Command:
		typeStr = "" // No prefix for regular commands
	}

	parts = append(parts, fmt.Sprintf("%s%s: %s", typeStr, c.Name, c.Body.String()))
	return strings.Join(parts, " ")
}

func (c *CommandDecl) Position() Position {
	return c.Pos
}

func (c *CommandDecl) TokenRange() TokenRange {
	return c.Tokens
}

func (c *CommandDecl) SemanticTokens() []lexer.Token {
	var tokens []lexer.Token

	// Add type and name tokens with proper semantics
	if c.TypeToken.Type != lexer.ILLEGAL {
		typeToken := c.TypeToken
		typeToken.Semantic = lexer.SemKeyword
		tokens = append(tokens, typeToken)
	}

	nameToken := c.NameToken
	nameToken.Semantic = lexer.SemCommand
	tokens = append(tokens, nameToken)

	// Add decorator tokens
	for _, decorator := range c.Decorators {
		tokens = append(tokens, decorator.SemanticTokens()...)
	}

	// Add body tokens
	tokens = append(tokens, c.Body.SemanticTokens()...)

	return tokens
}

// CommandType represents the type of command
type CommandType int

const (
	Command      CommandType = iota // Regular commands
	WatchCommand                    // watch NAME: ... (process management)
	StopCommand                     // stop NAME: ... (process cleanup)
)

func (ct CommandType) String() string {
	switch ct {
	case Command:
		return "command"
	case WatchCommand:
		return "watch"
	case StopCommand:
		return "stop"
	default:
		return "unknown"
	}
}

// CommandBody represents the unified body of a command
type CommandBody struct {
	// Statements represent the command content
	// For simple commands: single statement with command elements
	// For block commands: multiple statements
	Statements []Statement

	// IsBlock indicates if this was written with block syntax {}
	IsBlock bool

	Pos    Position
	Tokens TokenRange

	// For block commands with explicit braces
	OpenBrace  *lexer.Token // Optional - only for explicit blocks
	CloseBrace *lexer.Token // Optional - only for explicit blocks
}

func (b *CommandBody) String() string {
	if b.IsBlock && len(b.Statements) > 0 {
		var parts []string
		parts = append(parts, "{")
		for _, stmt := range b.Statements {
			parts = append(parts, "  "+stmt.String())
		}
		parts = append(parts, "}")
		return strings.Join(parts, "\n")
	}

	// Simple command - just concatenate statements
	var parts []string
	for _, stmt := range b.Statements {
		parts = append(parts, stmt.String())
	}
	return strings.Join(parts, "; ")
}

func (b *CommandBody) Position() Position {
	return b.Pos
}

func (b *CommandBody) TokenRange() TokenRange {
	return b.Tokens
}

func (b *CommandBody) SemanticTokens() []lexer.Token {
	var tokens []lexer.Token

	// Add structural tokens for blocks
	if b.OpenBrace != nil {
		tokens = append(tokens, *b.OpenBrace)
	}

	// Add statement tokens
	for _, stmt := range b.Statements {
		tokens = append(tokens, stmt.SemanticTokens()...)
	}

	if b.CloseBrace != nil {
		tokens = append(tokens, *b.CloseBrace)
	}

	return tokens
}

// Statement represents any statement within a command
type Statement interface {
	Node
	IsStatement() bool
}

// ShellStatement represents shell commands
type ShellStatement struct {
	Elements []CommandElement
	Pos      Position
	Tokens   TokenRange
}

func (s *ShellStatement) String() string {
	var parts []string
	for _, elem := range s.Elements {
		parts = append(parts, elem.String())
	}
	return strings.Join(parts, "")
}

func (s *ShellStatement) Position() Position {
	return s.Pos
}

func (s *ShellStatement) TokenRange() TokenRange {
	return s.Tokens
}

func (s *ShellStatement) SemanticTokens() []lexer.Token {
	var tokens []lexer.Token
	for _, elem := range s.Elements {
		tokens = append(tokens, elem.SemanticTokens()...)
	}
	return tokens
}

func (s *ShellStatement) IsStatement() bool {
	return true
}

// CommandElement represents elements within commands (text, decorators)
type CommandElement interface {
	Node
	IsCommandElement() bool
}

// TextElement represents literal text in commands
type TextElement struct {
	Text   string
	Pos    Position
	Tokens TokenRange
}

func (t *TextElement) String() string {
	return t.Text
}

func (t *TextElement) Position() Position {
	return t.Pos
}

func (t *TextElement) TokenRange() TokenRange {
	return t.Tokens
}

func (t *TextElement) SemanticTokens() []lexer.Token {
	// Text elements are shell content
	tokens := make([]lexer.Token, len(t.Tokens.All))
	copy(tokens, t.Tokens.All)

	for i := range tokens {
		if tokens[i].Semantic == lexer.SemCommand {
			// Keep command semantic
		} else {
			tokens[i].Semantic = lexer.SemCommand
		}
	}

	return tokens
}

func (t *TextElement) IsCommandElement() bool {
	return true
}

// Decorator represents decorators with unified args and block support
type Decorator struct {
	Name  string
	Args  []Expression     // Arguments within parentheses
	Block *DecoratorBlock  // Optional block content
	Pos   Position
	Tokens TokenRange

	// LSP support
	AtToken   lexer.Token // @ symbol
	NameToken lexer.Token // Decorator name
}

func (d *Decorator) String() string {
	var parts []string

	// Add decorator name
	name := fmt.Sprintf("@%s", d.Name)

	// Add arguments if present
	if len(d.Args) > 0 {
		var argStrs []string
		for _, arg := range d.Args {
			argStrs = append(argStrs, arg.String())
		}
		name += fmt.Sprintf("(%s)", strings.Join(argStrs, ", "))
	}

	parts = append(parts, name)

	// Add block if present
	if d.Block != nil {
		parts = append(parts, d.Block.String())
	}

	return strings.Join(parts, " ")
}

func (d *Decorator) Position() Position {
	return d.Pos
}

func (d *Decorator) TokenRange() TokenRange {
	return d.Tokens
}

func (d *Decorator) SemanticTokens() []lexer.Token {
	var tokens []lexer.Token

	// @ token as operator
	atToken := d.AtToken
	atToken.Semantic = lexer.SemOperator
	tokens = append(tokens, atToken)

	// Decorator name - special handling for @var
	nameToken := d.NameToken
	if d.Name == "var" || d.Name == "env" {
		nameToken.Semantic = lexer.SemVariable // @var, @env are variable-like
	} else {
		nameToken.Semantic = lexer.SemDecorator
	}
	tokens = append(tokens, nameToken)

	// Argument tokens
	for _, arg := range d.Args {
		tokens = append(tokens, arg.SemanticTokens()...)
	}

	// Block tokens
	if d.Block != nil {
		tokens = append(tokens, d.Block.SemanticTokens()...)
	}

	return tokens
}

func (d *Decorator) IsExpression() bool {
	return true
}

func (d *Decorator) GetType() ExpressionType {
	return IdentifierType
}

func (d *Decorator) IsCommandElement() bool {
	return true
}

type DecoratorBlock struct {
	Statements []Statement
	Pos        Position
	Tokens     TokenRange

	// Structural tokens
	OpenBrace  *lexer.Token
	CloseBrace *lexer.Token
}

func (b *DecoratorBlock) String() string {
	if len(b.Statements) == 0 {
		return "{}"
	}

	var parts []string
	parts = append(parts, "{")
	for _, stmt := range b.Statements {
		parts = append(parts, "  "+stmt.String())
	}
	parts = append(parts, "}")
	return strings.Join(parts, "\n")
}

func (b *DecoratorBlock) Position() Position {
	return b.Pos
}

func (b *DecoratorBlock) TokenRange() TokenRange {
	return b.Tokens
}

func (b *DecoratorBlock) SemanticTokens() []lexer.Token {
	var tokens []lexer.Token

	// Add brace tokens
	if b.OpenBrace != nil {
		tokens = append(tokens, *b.OpenBrace)
	}

	// Add statement tokens
	for _, stmt := range b.Statements {
		tokens = append(tokens, stmt.SemanticTokens()...)
	}

	if b.CloseBrace != nil {
		tokens = append(tokens, *b.CloseBrace)
	}

	return tokens
}

// Identifier represents identifiers (command names, shell commands, etc.)
type Identifier struct {
	Name   string
	Pos    Position
	Tokens TokenRange
	Token  lexer.Token // The identifier token
}

func (i *Identifier) String() string {
	return i.Name
}

func (i *Identifier) Position() Position {
	return i.Pos
}

func (i *Identifier) TokenRange() TokenRange {
	return i.Tokens
}

func (i *Identifier) SemanticTokens() []lexer.Token {
	return []lexer.Token{i.Token}
}

func (i *Identifier) IsExpression() bool {
	return true
}

func (i *Identifier) GetType() ExpressionType {
	return IdentifierType
}

func (i *Identifier) IsCommandElement() bool {
	return true
}

// Utility functions for AST traversal and analysis

// Walk traverses the AST and calls fn for each node
func Walk(node Node, fn func(Node) bool) {
	if !fn(node) {
		return
	}

	switch n := node.(type) {
	case *Program:
		for _, v := range n.Variables {
			Walk(&v, fn)
		}
		for _, c := range n.Commands {
			Walk(&c, fn)
		}
	case *CommandDecl:
		for _, d := range n.Decorators {
			Walk(&d, fn)
		}
		Walk(&n.Body, fn)
	case *CommandBody:
		for _, stmt := range n.Statements {
			Walk(stmt, fn)
		}
	case *ShellStatement:
		for _, elem := range n.Elements {
			Walk(elem, fn)
		}
	case *Decorator:
		for _, arg := range n.Args {
			Walk(arg, fn)
		}
		if n.Block != nil {
			Walk(n.Block, fn)
		}
	case *DecoratorBlock:
		for _, stmt := range n.Statements {
			Walk(stmt, fn)
		}
	}
}

// Helper functions for backward compatibility and convenience

// IsSimpleCommand checks if a command body represents a simple (non-block) command
func (b *CommandBody) IsSimpleCommand() bool {
	return !b.IsBlock && len(b.Statements) == 1
}

// GetSimpleElements returns the elements of a simple command
// Returns nil if this is not a simple command
func (b *CommandBody) GetSimpleElements() []CommandElement {
	if b.IsSimpleCommand() {
		if shell, ok := b.Statements[0].(*ShellStatement); ok {
			return shell.Elements
		}
	}
	return nil
}

// FindVariableReferences finds all @var() decorator references in the AST
func FindVariableReferences(node Node) []*Decorator {
	var refs []*Decorator

	Walk(node, func(n Node) bool {
		if decorator, ok := n.(*Decorator); ok && decorator.Name == "var" {
			refs = append(refs, decorator)
		}
		return true
	})

	return refs
}

// FindDecorators finds all decorators in the AST
func FindDecorators(node Node) []Decorator {
	var decorators []Decorator

	Walk(node, func(n Node) bool {
		if decorator, ok := n.(*Decorator); ok {
			decorators = append(decorators, *decorator)
		}
		return true
	})

	return decorators
}

// ValidateVariableReferences checks that all @var() decorator references are defined
func ValidateVariableReferences(program *Program) []error {
	var errors []error

	// Collect defined variables
	defined := make(map[string]bool)
	for _, varDecl := range program.Variables {
		defined[varDecl.Name] = true
	}

	// Check all @var() decorator references
	refs := FindVariableReferences(program)
	for _, ref := range refs {
		if len(ref.Args) > 0 {
			if identifier, ok := ref.Args[0].(*Identifier); ok {
				if !defined[identifier.Name] {
					errors = append(errors, fmt.Errorf("undefined variable '%s' at line %d", identifier.Name, ref.Pos.Line))
				}
			}
		}
	}

	return errors
}

// LSP Integration Functions

// FindNodeAtPosition finds the AST node at a specific position (for hover, go-to-def)
func FindNodeAtPosition(root Node, line, column int) Node {
	var found Node

	Walk(root, func(n Node) bool {
		tokenRange := n.TokenRange()

		// Check if position is within this node's range
		if isPositionInRange(line, column, tokenRange) {
			found = n
			return true // Continue to find most specific node
		}
		return false
	})

	return found
}

// isPositionInRange checks if a position falls within a token range
func isPositionInRange(line, column int, tokenRange TokenRange) bool {
	start := tokenRange.Start
	end := tokenRange.End

	// Single line check
	if start.Line == end.Line {
		return line == start.Line && column >= start.Column && column <= end.Column
	}

	// Multi-line check
	if line < start.Line || line > end.Line {
		return false
	}

	if line == start.Line {
		return column >= start.Column
	}

	if line == end.Line {
		return column <= end.Column
	}

	return true // Middle lines
}

// GetDefinitionForVariable finds the variable declaration for a given reference
func GetDefinitionForVariable(program *Program, varName string) *VariableDecl {
	for _, varDecl := range program.Variables {
		if varDecl.Name == varName {
			return &varDecl
		}
	}
	return nil
}

// GetReferencesForVariable finds all @var() decorator references to a specific variable
func GetReferencesForVariable(program *Program, varName string) []*Decorator {
	var references []*Decorator

	refs := FindVariableReferences(program)
	for _, ref := range refs {
		if len(ref.Args) > 0 {
			if identifier, ok := ref.Args[0].(*Identifier); ok && identifier.Name == varName {
				references = append(references, ref)
			}
		}
	}

	return references
}

// GetCompletionsAtPosition returns possible completions at a position
func GetCompletionsAtPosition(program *Program, line, column int) []CompletionItem {
	var completions []CompletionItem

	node := FindNodeAtPosition(program, line, column)
	if node == nil {
		return completions
	}

	// Add variable completions for @var() contexts
	if isInVariableContext(node) {
		for _, varDecl := range program.Variables {
			completions = append(completions, CompletionItem{
				Label:  varDecl.Name,
				Kind:   VariableCompletion,
				Detail: fmt.Sprintf("var %s = %s", varDecl.Name, varDecl.Value.String()),
			})
		}
	}

	// Add decorator completions for @ contexts
	if isInDecoratorContext(node) {
		decoratorNames := []string{"timeout", "retry", "confirm", "env", "restart-on", "debounce", "var", "parallel", "sh"}
		for _, name := range decoratorNames {
			completions = append(completions, CompletionItem{
				Label: name,
				Kind:  DecoratorCompletion,
				Detail: fmt.Sprintf("@%s decorator", name),
			})
		}
	}

	return completions
}

// CompletionItem represents an LSP completion item
type CompletionItem struct {
	Label  string
	Kind   CompletionKind
	Detail string
}

type CompletionKind int

const (
	VariableCompletion CompletionKind = iota
	DecoratorCompletion
	CommandCompletion
)

// Helper functions for context detection
func isInVariableContext(node Node) bool {
	// Check if we're in a @var() context
	if decorator, ok := node.(*Decorator); ok && decorator.Name == "var" {
		return true
	}
	return false
}

func isInDecoratorContext(node Node) bool {
	// Check if we're after an @ symbol
	if decorator, ok := node.(*Decorator); ok {
		return decorator != nil
	}
	return false
}

// Tree-sitter Integration Functions

// GetTreeSitterNode converts AST to Tree-sitter compatible structure
func (p *Program) ToTreeSitterJSON() map[string]interface{} {
	return map[string]interface{}{
		"type": "program",
		"children": []interface{}{
			p.variablesToTreeSitter(),
			p.commandsToTreeSitter(),
		},
		"start_position": map[string]int{
			"row":    p.Pos.Line - 1,
			"column": p.Pos.Column - 1,
		},
		"end_position": map[string]int{
			"row":    p.Tokens.End.Line - 1,
			"column": p.Tokens.End.Column - 1,
		},
	}
}

func (p *Program) variablesToTreeSitter() []interface{} {
	var vars []interface{}
	for _, varDecl := range p.Variables {
		vars = append(vars, map[string]interface{}{
			"type": "variable_declaration",
			"name": varDecl.Name,
			"value": varDecl.Value.String(),
			"start_position": map[string]int{
				"row":    varDecl.Pos.Line - 1,
				"column": varDecl.Pos.Column - 1,
			},
		})
	}
	return vars
}

func (p *Program) commandsToTreeSitter() []interface{} {
	var cmds []interface{}
	for _, cmdDecl := range p.Commands {
		cmd := map[string]interface{}{
			"type": "command_declaration",
			"name": cmdDecl.Name,
			"command_type": cmdDecl.Type.String(),
			"start_position": map[string]int{
				"row":    cmdDecl.Pos.Line - 1,
				"column": cmdDecl.Pos.Column - 1,
			},
		}

		if len(cmdDecl.Decorators) > 0 {
			var decorators []interface{}
			for _, decorator := range cmdDecl.Decorators {
				decorators = append(decorators, map[string]interface{}{
					"type": "decorator",
					"name": decorator.Name,
					"args": len(decorator.Args),
					"has_block": decorator.Block != nil,
				})
			}
			cmd["decorators"] = decorators
		}

		cmds = append(cmds, cmd)
	}
	return cmds
}

// GetSemanticTokensForLSP returns all semantic tokens for LSP highlighting
func (p *Program) GetSemanticTokensForLSP() []lexer.LSPSemanticToken {
	allTokens := p.SemanticTokens()
	lspTokens := make([]lexer.LSPSemanticToken, len(allTokens))

	for i, token := range allTokens {
		lspTokens[i] = token.ToLSPSemanticToken()
	}

	return lspTokens
}

// GetDiagnostics returns validation errors as LSP diagnostics
func (p *Program) GetDiagnostics() []Diagnostic {
	var diagnostics []Diagnostic

	// Variable validation
	errors := ValidateVariableReferences(p)
	for _, err := range errors {
		// Extract position from error (would need more sophisticated error types)
		diagnostics = append(diagnostics, Diagnostic{
			Message:  err.Error(),
			Severity: ErrorSeverity,
			Source:   "devcmd",
		})
	}

	return diagnostics
}

// Diagnostic represents an LSP diagnostic
type Diagnostic struct {
	Message  string
	Severity DiagnosticSeverity
	Source   string
}

type DiagnosticSeverity int

const (
	ErrorSeverity DiagnosticSeverity = iota
	WarningSeverity
	InfoSeverity
	HintSeverity
)
