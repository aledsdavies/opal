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
	TokenRange() TokenRange
	SemanticTokens() []lexer.Token
}

// Position represents source location information
type Position struct {
	Line   int
	Column int
	Offset int // Byte offset in source
}

// TokenRange represents the span of tokens for this AST node
type TokenRange struct {
	Start lexer.Token
	End   lexer.Token
	All   []lexer.Token
}

// Program represents the root of the CST (entire devcmd file)
// Preserves concrete syntax for LSP, Tree-sitter, and formatting tools
type Program struct {
	Variables []VariableDecl
	VarGroups []VarGroup    // Grouped variable declarations: var ( ... )
	Commands  []CommandDecl
	Pos       Position
	Tokens    TokenRange
}

func (p *Program) String() string {
	var parts []string
	for _, v := range p.Variables {
		parts = append(parts, v.String())
	}
	for _, g := range p.VarGroups {
		parts = append(parts, g.String())
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
	Value  Expression
	Pos    Position
	Tokens TokenRange

	// LSP-specific information
	NameToken  lexer.Token
	ValueToken lexer.Token
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
	tokens := []lexer.Token{v.NameToken, v.ValueToken}
	for _, token := range v.Tokens.All {
		if token.Type == lexer.IDENTIFIER && token.Value == v.Name {
			token.Semantic = lexer.SemVariable
		}
	}
	return tokens
}

// VarGroup represents grouped variable declarations: var ( NAME = value; ANOTHER = value )
// Preserves the concrete syntax for formatting and LSP features
type VarGroup struct {
	Variables []VariableDecl
	Pos       Position
	Tokens    TokenRange

	// Concrete syntax tokens for precise formatting
	VarToken   lexer.Token  // The "var" keyword
	OpenParen  lexer.Token  // The "(" token
	CloseParen lexer.Token  // The ")" token
}

func (g *VarGroup) String() string {
	var parts []string
	parts = append(parts, "var (")
	for _, v := range g.Variables {
		parts = append(parts, fmt.Sprintf("  %s = %s", v.Name, v.Value.String()))
	}
	parts = append(parts, ")")
	return strings.Join(parts, "\n")
}

func (g *VarGroup) Position() Position {
	return g.Pos
}

func (g *VarGroup) TokenRange() TokenRange {
	return g.Tokens
}

func (g *VarGroup) SemanticTokens() []lexer.Token {
	var tokens []lexer.Token

	// Add structural tokens with proper semantics
	varToken := g.VarToken
	varToken.Semantic = lexer.SemKeyword
	tokens = append(tokens, varToken)

	tokens = append(tokens, g.OpenParen)

	// Add variable tokens
	for _, v := range g.Variables {
		tokens = append(tokens, v.SemanticTokens()...)
	}

	tokens = append(tokens, g.CloseParen)

	return tokens
}

// Expression represents any expression (literals, identifiers, etc.)
type Expression interface {
	Node
	IsExpression() bool
	GetType() ExpressionType
	toTreeSitter() map[string]interface{}
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
	StringToken lexer.Token
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

func (s *StringLiteral) toTreeSitter() map[string]interface{} {
	return map[string]interface{}{
		"type": "string_literal",
		"value": s.Value,
		"raw": s.Raw,
		"start_position": map[string]int{
			"row":    s.Pos.Line - 1,
			"column": s.Pos.Column - 1,
		},
		"end_position": map[string]int{
			"row":    s.Tokens.End.Line - 1,
			"column": s.Tokens.End.Column - 1,
		},
	}
}

// NumberLiteral represents numeric values
type NumberLiteral struct {
	Value  string
	Pos    Position
	Tokens TokenRange
	Token  lexer.Token
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

func (n *NumberLiteral) toTreeSitter() map[string]interface{} {
	return map[string]interface{}{
		"type": "number_literal",
		"value": n.Value,
		"start_position": map[string]int{
			"row":    n.Pos.Line - 1,
			"column": n.Pos.Column - 1,
		},
		"end_position": map[string]int{
			"row":    n.Tokens.End.Line - 1,
			"column": n.Tokens.End.Column - 1,
		},
	}
}

// DurationLiteral represents duration values like 30s, 5m
type DurationLiteral struct {
	Value  string
	Pos    Position
	Tokens TokenRange
	Token  lexer.Token
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

func (d *DurationLiteral) toTreeSitter() map[string]interface{} {
	return map[string]interface{}{
		"type": "duration_literal",
		"value": d.Value,
		"start_position": map[string]int{
			"row":    d.Pos.Line - 1,
			"column": d.Pos.Column - 1,
		},
		"end_position": map[string]int{
			"row":    d.Tokens.End.Line - 1,
			"column": d.Tokens.End.Column - 1,
		},
	}
}

// Identifier represents identifiers
type Identifier struct {
	Name   string
	Pos    Position
	Tokens TokenRange
	Token  lexer.Token
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

func (i *Identifier) toTreeSitter() map[string]interface{} {
	return map[string]interface{}{
		"type": "identifier",
		"name": i.Name,
		"start_position": map[string]int{
			"row":    i.Pos.Line - 1,
			"column": i.Pos.Column - 1,
		},
		"end_position": map[string]int{
			"row":    i.Tokens.End.Line - 1,
			"column": i.Tokens.End.Column - 1,
		},
	}
}

// CommandDecl represents command definitions with concrete syntax preservation
type CommandDecl struct {
	Name       string
	Type       CommandType
	Body       CommandBody
	Pos        Position
	Tokens     TokenRange

	// Concrete syntax tokens for precise formatting and LSP
	TypeToken  *lexer.Token // The watch/stop keyword (nil for regular commands)
	NameToken  lexer.Token  // The command name token
	ColonToken lexer.Token  // The ":" token
}

func (c *CommandDecl) String() string {
	typeStr := ""
	switch c.Type {
	case WatchCommand:
		typeStr = "watch "
	case StopCommand:
		typeStr = "stop "
	case Command:
		typeStr = ""
	}

	return fmt.Sprintf("%s%s: %s", typeStr, c.Name, c.Body.String())
}

func (c *CommandDecl) Position() Position {
	return c.Pos
}

func (c *CommandDecl) TokenRange() TokenRange {
	return c.Tokens
}

func (c *CommandDecl) SemanticTokens() []lexer.Token {
	var tokens []lexer.Token

	if c.TypeToken != nil && c.TypeToken.Type != lexer.ILLEGAL {
		typeToken := *c.TypeToken
		typeToken.Semantic = lexer.SemKeyword
		tokens = append(tokens, typeToken)
	}

	nameToken := c.NameToken
	nameToken.Semantic = lexer.SemCommand
	tokens = append(tokens, nameToken)

	tokens = append(tokens, c.Body.SemanticTokens()...)

	return tokens
}

// CommandType represents the type of command
type CommandType int

const (
	Command      CommandType = iota
	WatchCommand
	StopCommand
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

// CommandBody represents the unified body of a command with concrete syntax preservation
type CommandBody struct {
	Content CommandContent
	IsBlock bool        // Indicates if this uses explicit block syntax {}
	Pos     Position
	Tokens  TokenRange

	// Concrete syntax tokens for precise formatting
	OpenBrace  *lexer.Token // The "{" token (nil for simple commands)
	CloseBrace *lexer.Token // The "}" token (nil for simple commands)
}

func (b *CommandBody) String() string {
	if b.IsBlock {
		return fmt.Sprintf("{ %s }", b.Content.String())
	}
	return b.Content.String()
}

func (b *CommandBody) Position() Position {
	return b.Pos
}

func (b *CommandBody) TokenRange() TokenRange {
	return b.Tokens
}

func (b *CommandBody) SemanticTokens() []lexer.Token {
	var tokens []lexer.Token

	if b.OpenBrace != nil {
		tokens = append(tokens, *b.OpenBrace)
	}

	tokens = append(tokens, b.Content.SemanticTokens()...)

	if b.CloseBrace != nil {
		tokens = append(tokens, *b.CloseBrace)
	}

	return tokens
}

// CommandContent represents the content within a command body
type CommandContent interface {
	Node
	IsCommandContent() bool
}

// ShellContent represents shell command content with potential inline decorators
// This supports mixed content like: echo "Building on port @var(PORT)"
type ShellContent struct {
	Parts  []ShellPart // Mixed content: text and inline decorators
	Pos    Position
	Tokens TokenRange
}

func (s *ShellContent) String() string {
	var parts []string
	for _, part := range s.Parts {
		parts = append(parts, part.String())
	}
	return strings.Join(parts, "")
}

func (s *ShellContent) Position() Position {
	return s.Pos
}

func (s *ShellContent) TokenRange() TokenRange {
	return s.Tokens
}

func (s *ShellContent) SemanticTokens() []lexer.Token {
	var tokens []lexer.Token
	for _, part := range s.Parts {
		tokens = append(tokens, part.SemanticTokens()...)
	}
	return tokens
}

func (s *ShellContent) IsCommandContent() bool {
	return true
}

// ShellPart represents a part of shell content (text or inline decorator)
type ShellPart interface {
	Node
	IsShellPart() bool
}

// TextPart represents plain text within shell content
type TextPart struct {
	Text   string
	Pos    Position
	Tokens TokenRange
}

func (t *TextPart) String() string {
	return t.Text
}

func (t *TextPart) Position() Position {
	return t.Pos
}

func (t *TextPart) TokenRange() TokenRange {
	return t.Tokens
}

func (t *TextPart) SemanticTokens() []lexer.Token {
	tokens := make([]lexer.Token, len(t.Tokens.All))
	copy(tokens, t.Tokens.All)

	// Mark all tokens as shell content
	for i := range tokens {
		if tokens[i].Semantic != lexer.SemCommand {
			tokens[i].Semantic = lexer.SemShellText
		}
	}

	return tokens
}

func (t *TextPart) IsShellPart() bool {
	return true
}

// DecoratedContent represents shell content with decorators
// This handles cases like: @timeout(30s) { node app.js }
// Multiple decorators in sequence within blocks are valid:
// deploy: { @parallel() {}; @retry(3) {} }
type DecoratedContent struct {
	Decorators []Decorator  // Leading decorators (valid when nested in blocks)
	Content    CommandContent // The actual content (can be ShellContent or nested DecoratedContent)
	Pos        Position
	Tokens     TokenRange
}

func (d *DecoratedContent) String() string {
	var parts []string

	for _, decorator := range d.Decorators {
		parts = append(parts, decorator.String())
	}

	parts = append(parts, d.Content.String())

	return strings.Join(parts, " ")
}

func (d *DecoratedContent) Position() Position {
	return d.Pos
}

func (d *DecoratedContent) TokenRange() TokenRange {
	return d.Tokens
}

func (d *DecoratedContent) SemanticTokens() []lexer.Token {
	var tokens []lexer.Token

	for _, decorator := range d.Decorators {
		tokens = append(tokens, decorator.SemanticTokens()...)
	}

	tokens = append(tokens, d.Content.SemanticTokens()...)

	return tokens
}

func (d *DecoratedContent) IsCommandContent() bool {
	return true
}

// Decorator represents decorators
type Decorator struct {
	Name  string
	Args  []Expression // Arguments within parentheses
	Pos   Position
	Tokens TokenRange

	// LSP support
	AtToken   lexer.Token
	NameToken lexer.Token
}

func (d *Decorator) String() string {
	name := fmt.Sprintf("@%s", d.Name)

	if len(d.Args) > 0 {
		var argStrs []string
		for _, arg := range d.Args {
			argStrs = append(argStrs, arg.String())
		}
		name += fmt.Sprintf("(%s)", strings.Join(argStrs, ", "))
	}

	return name
}

func (d *Decorator) Position() Position {
	return d.Pos
}

func (d *Decorator) TokenRange() TokenRange {
	return d.Tokens
}

func (d *Decorator) SemanticTokens() []lexer.Token {
	var tokens []lexer.Token

	atToken := d.AtToken
	atToken.Semantic = lexer.SemOperator
	tokens = append(tokens, atToken)

	nameToken := d.NameToken
	if d.Name == "var" || d.Name == "env" {
		nameToken.Semantic = lexer.SemVariable
	} else {
		nameToken.Semantic = lexer.SemDecorator
	}
	tokens = append(tokens, nameToken)

	for _, arg := range d.Args {
		tokens = append(tokens, arg.SemanticTokens()...)
	}

	return tokens
}

func (d *Decorator) toTreeSitter() map[string]interface{} {
	node := map[string]interface{}{
		"type": "decorator",
		"name": d.Name,
		"start_position": map[string]int{
			"row":    d.Pos.Line - 1,
			"column": d.Pos.Column - 1,
		},
		"end_position": map[string]int{
			"row":    d.Tokens.End.Line - 1,
			"column": d.Tokens.End.Column - 1,
		},
	}

	if len(d.Args) > 0 {
		args := make([]interface{}, len(d.Args))
		for i, arg := range d.Args {
			args[i] = arg.toTreeSitter()
		}
		node["args"] = args
	}

	return node
}

// FunctionDecorator represents inline decorators like @var(NAME) or @sh(command)
// These appear WITHIN shell content and return values
type FunctionDecorator struct {
	Name  string
	Args  []Expression
	Pos   Position
	Tokens TokenRange

	// Concrete syntax tokens for precise formatting and LSP
	AtToken     lexer.Token  // The "@" symbol
	NameToken   lexer.Token  // The decorator name token
	OpenParen   *lexer.Token // The "(" token (nil if no args)
	CloseParen  *lexer.Token // The ")" token (nil if no args)
}

func (f *FunctionDecorator) String() string {
	name := fmt.Sprintf("@%s", f.Name)

	if len(f.Args) > 0 {
		var argStrs []string
		for _, arg := range f.Args {
			argStrs = append(argStrs, arg.String())
		}
		name += fmt.Sprintf("(%s)", strings.Join(argStrs, ", "))
	}

	return name
}

func (f *FunctionDecorator) Position() Position {
	return f.Pos
}

func (f *FunctionDecorator) TokenRange() TokenRange {
	return f.Tokens
}

func (f *FunctionDecorator) SemanticTokens() []lexer.Token {
	var tokens []lexer.Token

	// @ token as operator
	atToken := f.AtToken
	atToken.Semantic = lexer.SemOperator
	tokens = append(tokens, atToken)

	// Function decorator name with proper semantic
	nameToken := f.NameToken
	if f.Name == "var" || f.Name == "sh" {
		nameToken.Semantic = lexer.SemVariable
	} else {
		nameToken.Semantic = lexer.SemDecorator
	}
	tokens = append(tokens, nameToken)

	// Add parentheses if present
	if f.OpenParen != nil {
		openParen := *f.OpenParen
		openParen.Semantic = lexer.SemOperator
		tokens = append(tokens, openParen)
	}

	// Add argument tokens
	for _, arg := range f.Args {
		tokens = append(tokens, arg.SemanticTokens()...)
	}

	if f.CloseParen != nil {
		closeParen := *f.CloseParen
		closeParen.Semantic = lexer.SemOperator
		tokens = append(tokens, closeParen)
	}

	return tokens
}

func (f *FunctionDecorator) IsExpression() bool {
	return true
}

func (f *FunctionDecorator) GetType() ExpressionType {
	return IdentifierType
}

func (f *FunctionDecorator) IsShellPart() bool {
	return true
}

func (f *FunctionDecorator) toTreeSitter() map[string]interface{} {
	node := map[string]interface{}{
		"type": "function_decorator",
		"name": f.Name,
		"start_position": map[string]int{
			"row":    f.Pos.Line - 1,
			"column": f.Pos.Column - 1,
		},
		"end_position": map[string]int{
			"row":    f.Tokens.End.Line - 1,
			"column": f.Tokens.End.Column - 1,
		},
	}

	if len(f.Args) > 0 {
		args := make([]interface{}, len(f.Args))
		for i, arg := range f.Args {
			args[i] = arg.toTreeSitter()
		}
		node["args"] = args
	}

	return node
}

// Utility functions for AST traversal and analysis

// Walk traverses the CST and calls fn for each node
func Walk(node Node, fn func(Node) bool) {
	if !fn(node) {
		return
	}

	switch n := node.(type) {
	case *Program:
		for _, v := range n.Variables {
			Walk(&v, fn)
		}
		for _, g := range n.VarGroups {
			Walk(&g, fn)
		}
		for _, c := range n.Commands {
			Walk(&c, fn)
		}
	case *VarGroup:
		for _, v := range n.Variables {
			Walk(&v, fn)
		}
	case *CommandDecl:
		Walk(&n.Body, fn)
	case *CommandBody:
		Walk(n.Content, fn)
	case *ShellContent:
		for _, part := range n.Parts {
			Walk(part, fn)
		}
	case *TextPart:
		// Leaf node - plain text
	case *DecoratedContent:
		for _, d := range n.Decorators {
			Walk(&d, fn)
		}
		Walk(n.Content, fn)
	case *Decorator:
		for _, arg := range n.Args {
			Walk(arg, fn)
		}
	case *FunctionDecorator:
		for _, arg := range n.Args {
			Walk(arg, fn)
		}
	}
}

// Helper functions for backward compatibility and convenience

// IsSimpleCommand checks if a command body represents a simple (non-decorated) command
func (b *CommandBody) IsSimpleCommand() bool {
	_, isShell := b.Content.(*ShellContent)
	return !b.IsBlock && isShell
}

// GetShellText returns the shell text if this is a simple shell command
func (b *CommandBody) GetShellText() string {
	if shell, ok := b.Content.(*ShellContent); ok {
		var textParts []string
		for _, part := range shell.Parts {
			if textPart, ok := part.(*TextPart); ok {
				textParts = append(textParts, textPart.Text)
			} else if funcDecorator, ok := part.(*FunctionDecorator); ok {
				textParts = append(textParts, funcDecorator.String())
			}
		}
		return strings.Join(textParts, "")
	}
	return ""
}

// GetInlineDecorators returns all inline decorators within shell content
func (b *CommandBody) GetInlineDecorators() []*FunctionDecorator {
	var decorators []*FunctionDecorator

	if shell, ok := b.Content.(*ShellContent); ok {
		for _, part := range shell.Parts {
			if funcDecorator, ok := part.(*FunctionDecorator); ok {
				decorators = append(decorators, funcDecorator)
			}
		}
	}

	return decorators
}

// FindVariableReferences finds all @var() decorator references in the AST
func FindVariableReferences(node Node) []*Decorator {
	var refs []*Decorator

	Walk(node, func(n Node) bool {
		if decorator, ok := n.(*Decorator); ok && decorator.Name == "var" {
			refs = append(refs, decorator)
		}
		if funcDecorator, ok := n.(*FunctionDecorator); ok && funcDecorator.Name == "var" {
			// Convert to regular decorator for compatibility
			decorator := &Decorator{
				Name:      funcDecorator.Name,
				Args:      funcDecorator.Args,
				Pos:       funcDecorator.Pos,
				Tokens:    funcDecorator.Tokens,
				AtToken:   funcDecorator.AtToken,
				NameToken: funcDecorator.NameToken,
			}
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

	// Collect defined variables from both individual and grouped declarations
	defined := make(map[string]bool)

	// Individual variables
	for _, varDecl := range program.Variables {
		defined[varDecl.Name] = true
	}

	// Grouped variables
	for _, varGroup := range program.VarGroups {
		for _, varDecl := range varGroup.Variables {
			defined[varDecl.Name] = true
		}
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

// GetDefinitionForVariable finds the variable declaration for a given reference
func GetDefinitionForVariable(program *Program, varName string) *VariableDecl {
	// Search individual variables
	for _, varDecl := range program.Variables {
		if varDecl.Name == varName {
			return &varDecl
		}
	}

	// Search grouped variables
	for _, varGroup := range program.VarGroups {
		for _, varDecl := range varGroup.Variables {
			if varDecl.Name == varName {
				return &varDecl
			}
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
