package ast

import (
	"fmt"

	"github.com/aledsdavies/devcmd/pkgs/lexer"
)

// LSP Integration Functions
// This file contains all Language Server Protocol specific functionality

// FindNodeAtPosition finds the AST node at a specific position (for hover, go-to-def)
func FindNodeAtPosition(root Node, line, column int) Node {
	var found Node

	Walk(root, func(n Node) bool {
		tokenRange := n.TokenRange()

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

// GetCompletionsAtPosition returns possible completions at a position
func GetCompletionsAtPosition(program *Program, line, column int) []CompletionItem {
	var completions []CompletionItem

	node := FindNodeAtPosition(program, line, column)
	if node == nil {
		return completions
	}

	// Add variable completions for @var() contexts
	if isInVariableContext(node) {
		// Individual variables
		for _, varDecl := range program.Variables {
			completions = append(completions, CompletionItem{
				Label:  varDecl.Name,
				Kind:   VariableCompletion,
				Detail: fmt.Sprintf("var %s = %s", varDecl.Name, varDecl.Value.String()),
			})
		}

		// Grouped variables
		for _, varGroup := range program.VarGroups {
			for _, varDecl := range varGroup.Variables {
				completions = append(completions, CompletionItem{
					Label:  varDecl.Name,
					Kind:   VariableCompletion,
					Detail: fmt.Sprintf("var %s = %s", varDecl.Name, varDecl.Value.String()),
				})
			}
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
	if funcDecorator, ok := node.(*FunctionDecorator); ok && funcDecorator.Name == "var" {
		return true
	}
	return false
}

func isInDecoratorContext(node Node) bool {
	// Check if we're after an @ symbol
	if decorator, ok := node.(*Decorator); ok {
		return decorator != nil
	}
	if funcDecorator, ok := node.(*FunctionDecorator); ok {
		return funcDecorator != nil
	}
	return false
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
	Range    *DiagnosticRange // Position information
}

type DiagnosticSeverity int

const (
	ErrorSeverity DiagnosticSeverity = iota
	WarningSeverity
	InfoSeverity
	HintSeverity
)

type DiagnosticRange struct {
	Start Position
	End   Position
}

// GetHoverInformation returns hover information for a position
func GetHoverInformation(program *Program, line, column int) *HoverInfo {
	node := FindNodeAtPosition(program, line, column)
	if node == nil {
		return nil
	}

	switch n := node.(type) {
	case *VariableDecl:
		return &HoverInfo{
			Contents: fmt.Sprintf("**Variable**: `%s`\n\n**Value**: `%s`", n.Name, n.Value.String()),
			Range:    &n.Tokens,
		}
	case *Decorator:
		return &HoverInfo{
			Contents: fmt.Sprintf("**Decorator**: `@%s`\n\n%s", n.Name, getDecoratorDescription(n.Name)),
			Range:    &n.Tokens,
		}
	case *FunctionDecorator:
		return &HoverInfo{
			Contents: fmt.Sprintf("**Function Decorator**: `@%s`\n\n%s", n.Name, getDecoratorDescription(n.Name)),
			Range:    &n.Tokens,
		}
	case *CommandDecl:
		return &HoverInfo{
			Contents: fmt.Sprintf("**Command**: `%s`\n\n**Type**: %s", n.Name, n.Type.String()),
			Range:    &n.Tokens,
		}
	}

	return nil
}

// HoverInfo represents hover information for LSP
type HoverInfo struct {
	Contents string
	Range    *TokenRange
}

// getDecoratorDescription returns description text for decorators
func getDecoratorDescription(name string) string {
	descriptions := map[string]string{
		"timeout":    "Sets a timeout for command execution",
		"retry":      "Retries command execution on failure",
		"confirm":    "Prompts for confirmation before execution",
		"env":        "Sets environment variables",
		"parallel":   "Executes commands in parallel",
		"var":        "References a variable value",
		"sh":         "Executes shell command and returns output",
		"debounce":   "Debounces command execution",
		"restart-on": "Restarts command on file changes",
	}

	if desc, found := descriptions[name]; found {
		return desc
	}
	return "Custom decorator"
}

// GetDefinitionLocation returns the definition location for a symbol
func GetDefinitionLocation(program *Program, line, column int) *DefinitionLocation {
	node := FindNodeAtPosition(program, line, column)
	if node == nil {
		return nil
	}

	// Handle @var() references
	if decorator, ok := node.(*Decorator); ok && decorator.Name == "var" {
		if len(decorator.Args) > 0 {
			if identifier, ok := decorator.Args[0].(*Identifier); ok {
				if varDecl := GetDefinitionForVariable(program, identifier.Name); varDecl != nil {
					return &DefinitionLocation{
						Range: varDecl.Tokens,
						Node:  varDecl,
					}
				}
			}
		}
	}

	// Handle function decorator @var() references
	if funcDecorator, ok := node.(*FunctionDecorator); ok && funcDecorator.Name == "var" {
		if len(funcDecorator.Args) > 0 {
			if identifier, ok := funcDecorator.Args[0].(*Identifier); ok {
				if varDecl := GetDefinitionForVariable(program, identifier.Name); varDecl != nil {
					return &DefinitionLocation{
						Range: varDecl.Tokens,
						Node:  varDecl,
					}
				}
			}
		}
	}

	return nil
}

// DefinitionLocation represents a definition location for LSP
type DefinitionLocation struct {
	Range TokenRange
	Node  Node
}

// GetReferences returns all references to a symbol
func GetReferences(program *Program, line, column int) []ReferenceLocation {
	node := FindNodeAtPosition(program, line, column)
	if node == nil {
		return nil
	}

	var references []ReferenceLocation

	// Handle variable declaration - find all references
	if varDecl, ok := node.(*VariableDecl); ok {
		refs := GetReferencesForVariable(program, varDecl.Name)
		for _, ref := range refs {
			references = append(references, ReferenceLocation{
				Range: ref.Tokens,
				Node:  ref,
			})
		}
	}

	return references
}

// ReferenceLocation represents a reference location for LSP
type ReferenceLocation struct {
	Range TokenRange
	Node  Node
}

// GetDocumentSymbols returns all symbols in the document for outline view
func GetDocumentSymbols(program *Program) []DocumentSymbol {
	var symbols []DocumentSymbol

	// Add individual variable symbols
	for _, varDecl := range program.Variables {
		symbols = append(symbols, DocumentSymbol{
			Name:   varDecl.Name,
			Kind:   VariableSymbol,
			Range:  varDecl.Tokens,
			Detail: fmt.Sprintf("var %s = %s", varDecl.Name, varDecl.Value.String()),
		})
	}

	// Add grouped variable symbols
	for _, varGroup := range program.VarGroups {
		groupSymbol := DocumentSymbol{
			Name:   "var (...)",
			Kind:   VariableSymbol,
			Range:  varGroup.Tokens,
			Detail: fmt.Sprintf("var group with %d variables", len(varGroup.Variables)),
		}

		// Add individual variables as children
		for _, varDecl := range varGroup.Variables {
			groupSymbol.Children = append(groupSymbol.Children, DocumentSymbol{
				Name:   varDecl.Name,
				Kind:   VariableSymbol,
				Range:  varDecl.Tokens,
				Detail: fmt.Sprintf("%s = %s", varDecl.Name, varDecl.Value.String()),
			})
		}

		symbols = append(symbols, groupSymbol)
	}

	// Add command symbols
	for _, cmdDecl := range program.Commands {
		symbol := DocumentSymbol{
			Name:   cmdDecl.Name,
			Kind:   FunctionSymbol,
			Range:  cmdDecl.Tokens,
			Detail: fmt.Sprintf("%s command", cmdDecl.Type.String()),
		}

		// Add decorators as children
		for _, decorator := range findCommandDecorators(&cmdDecl) {
			symbol.Children = append(symbol.Children, DocumentSymbol{
				Name:   decorator.Name,
				Kind:   DecoratorSymbol,
				Range:  decorator.Tokens,
				Detail: fmt.Sprintf("@%s decorator", decorator.Name),
			})
		}

		symbols = append(symbols, symbol)
	}

	return symbols
}

// DocumentSymbol represents a symbol in the document
type DocumentSymbol struct {
	Name     string
	Kind     SymbolKind
	Range    TokenRange
	Detail   string
	Children []DocumentSymbol
}

type SymbolKind int

const (
	VariableSymbol SymbolKind = iota
	FunctionSymbol
	DecoratorSymbol
)

// findCommandDecorators extracts decorators from a command
func findCommandDecorators(cmd *CommandDecl) []Decorator {
	var decorators []Decorator

	Walk(&cmd.Body, func(n Node) bool {
		if decorator, ok := n.(*Decorator); ok {
			decorators = append(decorators, *decorator)
		}
		return true
	})

	return decorators
}

// GetFoldingRanges returns ranges that can be folded in the editor
func GetFoldingRanges(program *Program) []FoldingRange {
	var ranges []FoldingRange

	// Add folding ranges for command blocks
	for _, cmdDecl := range program.Commands {
		if cmdDecl.Body.IsBlock {
			ranges = append(ranges, FoldingRange{
				Start: Position{
					Line:   cmdDecl.Body.Tokens.Start.Line,
					Column: cmdDecl.Body.Tokens.Start.Column,
					Offset: 0, // Could be calculated if needed
				},
				End: Position{
					Line:   cmdDecl.Body.Tokens.End.Line,
					Column: cmdDecl.Body.Tokens.End.Column,
					Offset: 0, // Could be calculated if needed
				},
				Kind: RegionFold,
			})
		}
	}

	return ranges
}

// FoldingRange represents a foldable range
type FoldingRange struct {
	Start Position
	End   Position
	Kind  FoldingKind
}

type FoldingKind int

const (
	RegionFold FoldingKind = iota
	CommentFold
	ImportFold
)
