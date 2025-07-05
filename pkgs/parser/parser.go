package parser

import (
	"fmt"
	"strings"

	"github.com/aledsdavies/devcmd/pkgs/ast"
	"github.com/aledsdavies/devcmd/pkgs/lexer"
	"github.com/aledsdavies/devcmd/pkgs/stdlib"
)

// Parser implements a fast, spec-compliant recursive descent parser for the Devcmd language.
// It trusts the lexer to have correctly handled whitespace and tokenization, focusing
// purely on assembling the Abstract Syntax Tree (AST).
type Parser struct {
	input  string // The raw input string for accurate value slicing
	tokens []lexer.Token
	pos    int // current position in the token slice

	// errors is a slice of errors encountered during parsing.
	// This allows for better error reporting by collecting multiple errors.
	errors []string
}

// Parse tokenizes and parses the input string into a complete AST.
// It returns the Program node and any errors encountered.
func Parse(input string) (*ast.Program, error) {
	lex := lexer.New(input)
	p := &Parser{
		input:  input, // Store the raw input
		tokens: lex.TokenizeToSlice(),
	}
	program := p.parseProgram()

	if len(p.errors) > 0 {
		return nil, fmt.Errorf("parsing failed:\n- %s", strings.Join(p.errors, "\n- "))
	}
	return program, nil
}

// --- Main Parsing Logic ---

// parseProgram is the top-level entry point for parsing.
// It iterates through the tokens and parses all top-level statements.
// Program = { VariableDecl | VarGroup | CommandDecl }*
func (p *Parser) parseProgram() *ast.Program {
	program := &ast.Program{}

	for !p.isAtEnd() {
		p.skipNewlines()
		if p.isAtEnd() {
			break
		}

		switch p.current().Type {
		case lexer.VAR:
			if p.peek().Type == lexer.LPAREN {
				varGroup, err := p.parseVarGroup()
				if err != nil {
					p.addError(err)
					p.synchronize()
				} else {
					program.VarGroups = append(program.VarGroups, *varGroup)
				}
			} else {
				varDecl, err := p.parseVariableDecl()
				if err != nil {
					p.addError(err)
					p.synchronize()
				} else {
					program.Variables = append(program.Variables, *varDecl)
				}
			}
		case lexer.IDENTIFIER, lexer.WATCH, lexer.STOP, lexer.AT:
			// A command can start with a name (IDENTIFIER), a keyword (WATCH/STOP),
			// or a decorator (@).
			cmd, err := p.parseCommandDecl()
			if err != nil {
				p.addError(err)
				p.synchronize()
			} else {
				program.Commands = append(program.Commands, *cmd)
			}
		default:
			p.addError(fmt.Errorf("unexpected token %s, expected a top-level declaration (var, command)", p.current().Type))
			p.synchronize()
		}
	}

	return program
}

// parseCommandDecl parses a full command declaration.
// CommandDecl = { Decorator }* [ "watch" | "stop" ] IDENTIFIER ":" CommandBody
func (p *Parser) parseCommandDecl() (*ast.CommandDecl, error) {
	startPos := p.current()

	// 1. Parse optional decorators at command level (before command name)
	decorators := p.parseDecorators()

	// 2. Parse command type (watch, stop, or regular)
	var cmdType ast.CommandType = ast.Command
	var typeToken *lexer.Token
	if p.match(lexer.WATCH) {
		cmdType = ast.WatchCommand
		token := p.current()
		typeToken = &token
		p.advance()
	} else if p.match(lexer.STOP) {
		cmdType = ast.StopCommand
		token := p.current()
		typeToken = &token
		p.advance()
	}

	// 3. Parse command name
	nameToken, err := p.consume(lexer.IDENTIFIER, "expected command name")
	if err != nil {
		return nil, err
	}
	name := nameToken.Value

	// 4. Parse colon
	colonToken, err := p.consume(lexer.COLON, "expected ':' after command name")
	if err != nil {
		return nil, err
	}

	// 5. Parse command body (this will handle post-colon decorators and syntax sugar)
	body, err := p.parseCommandBody()
	if err != nil {
		return nil, err
	}

	// If we have command-level decorators, we need to wrap the body content.
	if len(decorators) > 0 {
		body.Content = &ast.DecoratedContent{
			Decorators: decorators,
			Content:    body.Content,
			Pos:        ast.Position{Line: startPos.Line, Column: startPos.Column},
		}
	}

	return &ast.CommandDecl{
		Name:       name,
		Type:       cmdType,
		Body:       *body,
		Pos:        ast.Position{Line: startPos.Line, Column: startPos.Column},
		TypeToken:  typeToken,
		NameToken:  nameToken,
		ColonToken: colonToken,
	}, nil
}

// parseCommandBody parses the content after the command's colon.
// It handles the syntax sugar for simple vs. block commands.
// **UPDATED**: Now handles pattern decorators and decorator syntax sugar correctly.
// CommandBody = "{" CommandContent "}" | DecoratorSugar | CommandContent
func (p *Parser) parseCommandBody() (*ast.CommandBody, error) {
	startPos := p.current()

	// **UPDATED**: Check for decorator syntax sugar: @decorator(args) { ... }
	// This should be equivalent to: { @decorator(args) { ... } }
	if p.match(lexer.AT) {
		// Save position in case we need to backtrack
		savedPos := p.pos

		// Try to parse decorators after the colon
		decorators := p.parseDecorators()

		// After decorators, we expect either:
		// 1. A block { ... } (syntax sugar - should be treated as IsBlock=true)
		// 2. Simple shell content (only valid for function decorators)

		if p.match(lexer.LBRACE) {
			// **SYNTAX SUGAR**: @decorator(args) { ... } becomes { @decorator(args) { ... } }
			openBrace, _ := p.consume(lexer.LBRACE, "") // already checked
			content, err := p.parseCommandContent(true) // Pass inBlock=true
			if err != nil {
				return nil, err
			}
			closeBrace, err := p.consume(lexer.RBRACE, "expected '}' to close command block")
			if err != nil {
				return nil, err
			}

			// Wrap content with decorators
			decoratedContent := &ast.DecoratedContent{
				Decorators: decorators,
				Content:    content,
				Pos:        ast.Position{Line: startPos.Line, Column: startPos.Column},
			}

			return &ast.CommandBody{
				Content:    decoratedContent,
				IsBlock:    true, // **CRITICAL**: This is block syntax sugar
				Pos:        ast.Position{Line: startPos.Line, Column: startPos.Column},
				OpenBrace:  &openBrace,
				CloseBrace: &closeBrace,
			}, nil
		} else {
			// Decorators without braces - check if they're all function decorators
			allFunctionDecorators := true
			for _, decorator := range decorators {
				if !stdlib.IsFunctionDecorator(decorator.Name) {
					allFunctionDecorators = false
					break
				}
			}

			if !allFunctionDecorators {
				// Block decorators must be followed by braces
				return nil, fmt.Errorf("expected '{' after block decorator(s) (at %d:%d, got %s)",
					p.current().Line, p.current().Column, p.current().Type)
			}

			// All function decorators - backtrack and parse as shell content
			p.pos = savedPos
			content, err := p.parseCommandContent(false)
			if err != nil {
				return nil, err
			}
			return &ast.CommandBody{
				Content: content,
				IsBlock: false,
				Pos:     ast.Position{Line: startPos.Line, Column: startPos.Column},
			}, nil
		}
	}

	// Explicit block: { ... }
	if p.match(lexer.LBRACE) {
		openBrace, _ := p.consume(lexer.LBRACE, "") // already checked
		content, err := p.parseCommandContent(true) // Pass inBlock=true
		if err != nil {
			return nil, err
		}
		closeBrace, err := p.consume(lexer.RBRACE, "expected '}' to close command block")
		if err != nil {
			return nil, err
		}
		return &ast.CommandBody{
			Content:    content,
			IsBlock:    true,
			Pos:        ast.Position{Line: startPos.Line, Column: startPos.Column},
			OpenBrace:  &openBrace,
			CloseBrace: &closeBrace,
		}, nil
	}

	// Simple command (no braces, ends at newline)
	content, err := p.parseCommandContent(false) // Pass inBlock=false
	if err != nil {
		return nil, err
	}
	return &ast.CommandBody{
		Content: content,
		IsBlock: false,
		Pos:     ast.Position{Line: startPos.Line, Column: startPos.Column},
	}, nil
}

// parseCommandContent parses the actual content of a command, which can be
// shell text, decorators, or pattern content.
// It is context-aware via the `inBlock` parameter.
func (p *Parser) parseCommandContent(inBlock bool) (ast.CommandContent, error) {
	startPos := p.current()

	// Check for pattern decorators (@when, @try)
	if p.match(lexer.AT) && p.isPatternDecorator() {
		return p.parsePatternContent()
	}

	// Check for block decorators
	if p.match(lexer.AT) && !p.isFunctionDecorator() {
		decorators := p.parseDecorators()
		content, err := p.parseCommandContent(inBlock) // Recursive call
		if err != nil {
			return nil, err
		}
		return &ast.DecoratedContent{
			Decorators: decorators,
			Content:    content,
			Pos:        ast.Position{Line: startPos.Line, Column: startPos.Column},
		}, nil
	}

	// Otherwise, it must be shell content.
	return p.parseShellContent(inBlock)
}

// parsePatternContent parses pattern-matching decorator content (@when, @try)
// This handles syntax like: @when(VAR) { pattern: command; pattern: command }
func (p *Parser) parsePatternContent() (*ast.PatternContent, error) {
	startPos := p.current()

	// Parse the pattern decorator
	decorator, err := p.parsePatternDecorator()
	if err != nil {
		return nil, err
	}

	// Expect opening brace
	openBrace, err := p.consume(lexer.LBRACE, "expected '{' after pattern decorator")
	if err != nil {
		return nil, err
	}

	// Parse pattern branches
	var patterns []ast.PatternBranch
	for !p.match(lexer.RBRACE) && !p.isAtEnd() {
		p.skipNewlines()
		if p.match(lexer.RBRACE) {
			break
		}

		branch, err := p.parsePatternBranch()
		if err != nil {
			return nil, err
		}
		patterns = append(patterns, *branch)
		p.skipNewlines()
	}

	// Expect closing brace
	closeBrace, err := p.consume(lexer.RBRACE, "expected '}' to close pattern block")
	if err != nil {
		return nil, err
	}

	return &ast.PatternContent{
		Decorator:  *decorator,
		Patterns:   patterns,
		Pos:        ast.Position{Line: startPos.Line, Column: startPos.Column},
		OpenBrace:  openBrace,
		CloseBrace: closeBrace,
	}, nil
}

// parsePatternBranch parses a single pattern branch: pattern: command
func (p *Parser) parsePatternBranch() (*ast.PatternBranch, error) {
	startPos := p.current()

	// Parse pattern (identifier or wildcard)
	var pattern ast.Pattern
	if p.match(lexer.ASTERISK) {
		token := p.current()
		p.advance()
		pattern = &ast.WildcardPattern{
			Pos:   ast.Position{Line: token.Line, Column: token.Column},
			Token: token,
		}
	} else if p.match(lexer.IDENTIFIER) {
		token := p.current()
		p.advance()
		pattern = &ast.IdentifierPattern{
			Name:  token.Value,
			Pos:   ast.Position{Line: token.Line, Column: token.Column},
			Token: token,
		}
	} else {
		return nil, fmt.Errorf("expected pattern identifier or '*', got %s", p.current().Type)
	}

	// Parse colon
	colonToken, err := p.consume(lexer.COLON, "expected ':' after pattern")
	if err != nil {
		return nil, err
	}

	// Parse command content
	content, err := p.parseCommandContent(true) // Pattern branches are always in block context
	if err != nil {
		return nil, err
	}

	return &ast.PatternBranch{
		Pattern:    pattern,
		Command:    content,
		Pos:        ast.Position{Line: startPos.Line, Column: startPos.Column},
		ColonToken: colonToken,
	}, nil
}

// parseShellContent parses shell content, extracting inline function decorators
// **UPDATED**: Now processes full SHELL_TEXT tokens and extracts inline decorators
func (p *Parser) parseShellContent(inBlock bool) (*ast.ShellContent, error) {
	startPos := p.current()
	var parts []ast.ShellPart

	for !p.isCommandTerminator(inBlock) {
		if p.match(lexer.SHELL_TEXT) {
			// Extract inline function decorators from shell text
			shellToken := p.current()
			p.advance()

			extractedParts, err := p.extractInlineDecorators(shellToken.Value)
			if err != nil {
				return nil, err
			}
			parts = append(parts, extractedParts...)
		} else {
			// This case handles inline function decorators that might be separate tokens
			if p.match(lexer.AT) && p.isFunctionDecorator() {
				decorator, err := p.parseFunctionDecorator()
				if err != nil {
					return nil, err
				}
				parts = append(parts, decorator)
			} else {
				// Fallback for any other unexpected tokens; treat them as text.
				// This makes parsing more resilient.
				tok := p.current()
				parts = append(parts, &ast.TextPart{Text: tok.Value})
				p.advance()
			}
		}
	}

	return &ast.ShellContent{
		Parts: parts,
		Pos:   ast.Position{Line: startPos.Line, Column: startPos.Column},
	}, nil
}

// extractInlineDecorators extracts function decorators from shell text
// This handles cases like: "echo Building on port @var(PORT)"
func (p *Parser) extractInlineDecorators(shellText string) ([]ast.ShellPart, error) {
	var parts []ast.ShellPart
	var currentText strings.Builder

	i := 0
	for i < len(shellText) {
		// Look for @var( pattern
		if i+5 < len(shellText) && shellText[i] == '@' && shellText[i+1:i+5] == "var(" {
			// Flush any pending text
			if currentText.Len() > 0 {
				parts = append(parts, &ast.TextPart{Text: currentText.String()})
				currentText.Reset()
			}

			// Find the closing parenthesis
			start := i
			i += 5 // Skip "@var("
			parenCount := 1
			argStart := i

			for i < len(shellText) && parenCount > 0 {
				if shellText[i] == '(' {
					parenCount++
				} else if shellText[i] == ')' {
					parenCount--
				}
				i++
			}

			if parenCount == 0 {
				// We found a complete @var(...) expression
				argEnd := i - 1 // Position of closing ')'
				argText := shellText[argStart:argEnd]

				// Create the function decorator
				parts = append(parts, &ast.FunctionDecorator{
					Name: "var",
					Args: []ast.Expression{
						&ast.Identifier{
							Name: strings.TrimSpace(argText),
						},
					},
				})
			} else {
				// Unclosed parenthesis - treat as text
				currentText.WriteString(shellText[start:i])
			}
		} else {
			// Regular character
			currentText.WriteByte(shellText[i])
			i++
		}
	}

	// Flush any remaining text
	if currentText.Len() > 0 {
		parts = append(parts, &ast.TextPart{Text: currentText.String()})
	}

	return parts, nil
}

// A helper for the above function
func (p *Parser) isFunctionDecoratorFromToken(tokens []lexer.Token) bool {
	if len(tokens) < 2 {
		return false
	}
	if tokens[0].Type != lexer.AT {
		return false
	}
	name := tokens[1].Value
	return stdlib.IsFunctionDecorator(name)
}


// parseInlineDecoratorArgs parses arguments from inline decorator text
func (p *Parser) parseInlineDecoratorArgs(argsText string) ([]ast.Expression, error) {
	var args []ast.Expression

	// Split by comma, but be careful about nested parentheses
	argParts := p.splitDecoratorArgs(argsText)

	for _, argPart := range argParts {
		argPart = strings.TrimSpace(argPart)
		if argPart == "" {
			continue
		}

		// Create an identifier expression for the argument
		args = append(args, &ast.Identifier{
			Name: argPart,
		})
	}

	return args, nil
}

// splitDecoratorArgs splits decorator arguments by comma, respecting nested parentheses
func (p *Parser) splitDecoratorArgs(argsText string) []string {
	var parts []string
	var current strings.Builder
	parenDepth := 0

	for _, char := range argsText {
		switch char {
		case '(':
			parenDepth++
			current.WriteRune(char)
		case ')':
			parenDepth--
			current.WriteRune(char)
		case ',':
			if parenDepth == 0 {
				parts = append(parts, current.String())
				current.Reset()
			} else {
				current.WriteRune(char)
			}
		default:
			current.WriteRune(char)
		}
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}

// --- Expression and Literal Parsing ---

// parseExpression parses any valid expression (literals, identifiers, function decorators).
// This is used for parsing decorator arguments, where an identifier can be complex.
func (p *Parser) parseExpression() (ast.Expression, error) {
	switch p.current().Type {
	case lexer.STRING:
		tok := p.current()
		p.advance()
		return &ast.StringLiteral{Value: tok.Value, Raw: tok.Raw, StringToken: tok}, nil
	case lexer.NUMBER:
		tok := p.current()
		p.advance()
		return &ast.NumberLiteral{Value: tok.Value, Token: tok}, nil
	case lexer.DURATION:
		tok := p.current()
		p.advance()
		return &ast.DurationLiteral{Value: tok.Value, Token: tok}, nil
	case lexer.IDENTIFIER:
		// For decorator arguments, an "identifier" can be a complex value.
		// This function consumes tokens until a separator is found.
		return p.parseDecoratorArgument()
	case lexer.AT:
		// Function decorators can appear as expressions in decorator arguments
		if p.isFunctionDecorator() {
			return p.parseFunctionDecorator()
		}
	}
	return nil, fmt.Errorf("unexpected token %s, expected an expression (literal, identifier, or @var)", p.current().Type)
}

// parseDecoratorArgument handles complex decorator arguments.
// **UPDATED**: This version is now robust and handles nested parentheses correctly,
// ensuring it consumes the entire intended argument without overrunning.
func (p *Parser) parseDecoratorArgument() (ast.Expression, error) {
    startToken := p.current()
    startOffset := startToken.Span.Start.Offset

    // We need to find the end of the argument, which is either a comma or a closing parenthesis
    // at the same parenthesis level.
    parenDepth := 0
    searchPos := p.pos

    for searchPos < len(p.tokens) {
        tok := p.tokens[searchPos]
        if (tok.Type == lexer.COMMA || tok.Type == lexer.RPAREN) && parenDepth == 0 {
            break
        }
        if tok.Type == lexer.LPAREN {
            parenDepth++
        } else if tok.Type == lexer.RPAREN {
            parenDepth--
        }
        searchPos++
    }

    // The argument ends at the start of the terminator token, or the end of the last token if at EOF.
    var endOffset int
    if searchPos < len(p.tokens) {
        endOffset = p.tokens[searchPos].Span.Start.Offset
        // Trim trailing space
        for endOffset > startOffset && strings.ContainsRune(" \t", rune(p.input[endOffset-1])) {
            endOffset--
        }
    } else {
        endOffset = p.tokens[len(p.tokens)-1].Span.End.Offset // EOF
    }

    value := p.input[startOffset:endOffset]

    // Advance parser position past the consumed tokens for the argument.
    p.pos = searchPos

    return &ast.Identifier{
        Name:  value,
        Token: lexer.Token{Value: value, Line: startToken.Line, Column: startToken.Column},
    }, nil
}


// --- Variable Parsing ---

// parseVariableDecl parses a variable declaration.
// **FIXED**: Now properly handles complex multi-token values
func (p *Parser) parseVariableDecl() (*ast.VariableDecl, error) {
	startPos := p.current()
	_, err := p.consume(lexer.VAR, "expected 'var'")
	if err != nil {
		return nil, err
	}

	name, err := p.consume(lexer.IDENTIFIER, "expected variable name")
	if err != nil {
		return nil, err
	}
	_, err = p.consume(lexer.EQUALS, "expected '=' after variable name")
	if err != nil {
		return nil, err
	}

	// Parse variable value - can be complex multi-token values
	value, err := p.parseVariableValue()
	if err != nil {
		return nil, err
	}

	return &ast.VariableDecl{
		Name:      name.Value,
		Value:     value,
		Pos:       ast.Position{Line: startPos.Line, Column: startPos.Column},
		NameToken: name,
	}, nil
}

// parseVariableValue parses variable values, correctly handling complex unquoted strings.
func (p *Parser) parseVariableValue() (ast.Expression, error) {
	startToken := p.current()

	// Handle standard literals first.
	switch startToken.Type {
	case lexer.STRING:
		p.advance()
		return &ast.StringLiteral{Value: startToken.Value, Raw: startToken.Raw, StringToken: startToken}, nil
	case lexer.NUMBER:
		p.advance()
		return &ast.NumberLiteral{Value: startToken.Value, Token: startToken}, nil
	case lexer.DURATION:
		p.advance()
		return &ast.DurationLiteral{Value: startToken.Value, Token: startToken}, nil
	case lexer.AT:
		if p.isFunctionDecorator() {
			return p.parseFunctionDecorator()
		}
	}

	// For everything else (unquoted strings like paths, URLs, commands),
	// consume tokens until a terminator and slice the raw input.
	valueStartOffset := startToken.Span.Start.Offset
	var valueEndOffset int

	// Advance until we find a terminator.
	for !p.isAtEnd() && !p.isVariableValueTerminator() {
		p.advance()
	}

	// The value ends at the start of the current token (the terminator)
	// or the end of the input if we ran out of tokens.
	if p.isAtEnd() {
		valueEndOffset = len(p.input)
	} else {
		valueEndOffset = p.current().Span.Start.Offset
	}

	// Backtrack to remove trailing whitespace from the raw slice.
	for valueEndOffset > valueStartOffset && strings.ContainsRune(" \t\r\n", rune(p.input[valueEndOffset-1])) {
		valueEndOffset--
	}

	value := p.input[valueStartOffset:valueEndOffset]

	// If value is empty, it's an error unless we are at the end.
	if value == "" && !p.isAtEnd() {
		return nil, fmt.Errorf("missing variable value at line %d, col %d", startToken.Line, startToken.Column)
	}

	return &ast.Identifier{
		Name:  value,
		Token: lexer.Token{Value: value, Line: startToken.Line, Column: startToken.Column},
	}, nil
}

// isVariableValueTerminator checks if the current token marks the end of a variable's value.
func (p *Parser) isVariableValueTerminator() bool {
	if p.isAtEnd() {
		return true
	}

	// Newlines or comments always terminate a value.
	switch p.current().Type {
	case lexer.NEWLINE, lexer.EOF, lexer.COMMENT, lexer.MULTILINE_COMMENT:
		return true
	// A new top-level declaration also terminates.
	case lexer.VAR, lexer.WATCH, lexer.STOP:
		return true
	// A new command declaration (e.g., `build:`) terminates the variable value.
	case lexer.IDENTIFIER:
		return p.pos+1 < len(p.tokens) && p.tokens[p.pos+1].Type == lexer.COLON
	case lexer.AT:
		// A decorator starting a new command terminates the value.
		return p.pos+2 < len(p.tokens) && p.tokens[p.pos+2].Type == lexer.COLON
	default:
		return false
	}
}

func (p *Parser) parseVarGroup() (*ast.VarGroup, error) {
	startPos := p.current()
	_, err := p.consume(lexer.VAR, "expected 'var'")
	if err != nil {
		return nil, err
	}
	openParen, err := p.consume(lexer.LPAREN, "expected '(' for var group")
	if err != nil {
		return nil, err
	}

	var variables []ast.VariableDecl
	for !p.match(lexer.RPAREN) && !p.isAtEnd() {
		p.skipNewlines()
		if p.match(lexer.RPAREN) {
			break
		}
		if p.current().Type != lexer.IDENTIFIER {
			p.addError(fmt.Errorf("expected variable name inside var group, got %s", p.current().Type))
			p.synchronize()
			continue
		}

		varDecl, err := p.parseGroupedVariableDecl()
		if err != nil {
			return nil, err // Be strict inside var groups
		}
		variables = append(variables, *varDecl)
		p.skipNewlines()
	}

	closeParen, err := p.consume(lexer.RPAREN, "expected ')' to close var group")
	if err != nil {
		return nil, err
	}

	return &ast.VarGroup{
		Variables:  variables,
		Pos:        ast.Position{Line: startPos.Line, Column: startPos.Column},
		OpenParen:  openParen,
		CloseParen: closeParen,
	}, nil
}

// parseGroupedVariableDecl is a helper for parsing `NAME = VALUE` lines within a `var (...)` block.
func (p *Parser) parseGroupedVariableDecl() (*ast.VariableDecl, error) {
	name, err := p.consume(lexer.IDENTIFIER, "expected variable name")
	if err != nil {
		return nil, err
	}
	_, err = p.consume(lexer.EQUALS, "expected '=' after variable name")
	if err != nil {
		return nil, err
	}

	// Use the same robust value parsing logic.
	value, err := p.parseVariableValue()
	if err != nil {
		return nil, err
	}

	return &ast.VariableDecl{
		Name:      name.Value,
		Value:     value,
		Pos:       ast.Position{Line: name.Line, Column: name.Column},
		NameToken: name,
	}, nil
}

// --- Decorator Parsing ---

// parseDecorators parses a sequence of one or more block decorators.
func (p *Parser) parseDecorators() []ast.Decorator {
	var decorators []ast.Decorator
	for p.match(lexer.AT) && !p.isFunctionDecorator() && !p.isPatternDecorator() {
		decorator, err := p.parseBlockDecorator()
		if err != nil {
			p.addError(err)
			p.synchronize()
			return decorators // Stop parsing decorators on error
		}
		decorators = append(decorators, *decorator)
	}
	return decorators
}

func (p *Parser) parseBlockDecorator() (*ast.Decorator, error) {
	startPos := p.current()
	atToken, _ := p.consume(lexer.AT, "expected '@'")

	// **FIXED**: Handle keywords like "var" that appear as decorator names
	var nameToken lexer.Token
	var err error

	if p.current().Type == lexer.IDENTIFIER {
		nameToken, err = p.consume(lexer.IDENTIFIER, "expected decorator name")
	} else {
		// Handle special cases where keywords appear as decorator names
		// This is needed for decorators like @var, @env, etc.
		nameToken = p.current()
		if !p.isValidDecoratorName(nameToken) {
			return nil, fmt.Errorf("expected decorator name, got %s", nameToken.Type)
		}
		p.advance()
	}

	if err != nil {
		return nil, err
	}

	decoratorName := nameToken.Value
	if nameToken.Type != lexer.IDENTIFIER {
		// Convert token value to string for non-identifier tokens
		decoratorName = strings.ToLower(nameToken.Value)
	}

	if !stdlib.IsBlockDecorator(decoratorName) {
		return nil, fmt.Errorf("@%s is not a block decorator", decoratorName)
	}

	var args []ast.Expression
	if p.match(lexer.LPAREN) {
		p.advance() // consume '('
		args, err = p.parseArgumentList()
		if err != nil {
			return nil, err
		}
		_, err = p.consume(lexer.RPAREN, "expected ')' after decorator arguments")
		if err != nil {
			return nil, err
		}
	}

	return &ast.Decorator{
		Name:      decoratorName,
		Args:      args,
		Pos:       ast.Position{Line: startPos.Line, Column: startPos.Column},
		AtToken:   atToken,
		NameToken: nameToken,
	}, nil
}

func (p *Parser) parsePatternDecorator() (*ast.Decorator, error) {
	startPos := p.current()
	atToken, _ := p.consume(lexer.AT, "expected '@'")

	var nameToken lexer.Token
	var err error

	if p.current().Type == lexer.IDENTIFIER {
		nameToken, err = p.consume(lexer.IDENTIFIER, "expected decorator name")
	} else {
		nameToken = p.current()
		if !p.isValidDecoratorName(nameToken) {
			return nil, fmt.Errorf("expected decorator name, got %s", nameToken.Type)
		}
		p.advance()
	}

	if err != nil {
		return nil, err
	}

	decoratorName := nameToken.Value
	if nameToken.Type != lexer.IDENTIFIER {
		decoratorName = strings.ToLower(nameToken.Value)
	}

	if !stdlib.IsPatternDecorator(decoratorName) {
		return nil, fmt.Errorf("@%s is not a pattern decorator", decoratorName)
	}

	var args []ast.Expression
	if p.match(lexer.LPAREN) {
		p.advance() // consume '('
		args, err = p.parseArgumentList()
		if err != nil {
			return nil, err
		}
		_, err = p.consume(lexer.RPAREN, "expected ')' after decorator arguments")
		if err != nil {
			return nil, err
		}
	}

	return &ast.Decorator{
		Name:      decoratorName,
		Args:      args,
		Pos:       ast.Position{Line: startPos.Line, Column: startPos.Column},
		AtToken:   atToken,
		NameToken: nameToken,
	}, nil
}

func (p *Parser) parseFunctionDecorator() (*ast.FunctionDecorator, error) {
	startPos := p.current()
	atToken, _ := p.consume(lexer.AT, "expected '@'")

	var nameToken lexer.Token
	var err error

	if p.current().Type == lexer.IDENTIFIER {
		nameToken, err = p.consume(lexer.IDENTIFIER, "expected decorator name")
	} else {
		nameToken = p.current()
		if !p.isValidDecoratorName(nameToken) {
			return nil, fmt.Errorf("expected decorator name, got %s", nameToken.Type)
		}
		p.advance()
	}

	if err != nil {
		return nil, err
	}

	decoratorName := nameToken.Value
	if nameToken.Type != lexer.IDENTIFIER {
		decoratorName = strings.ToLower(nameToken.Value)
	}

	if !stdlib.IsFunctionDecorator(decoratorName) {
		return nil, fmt.Errorf("@%s is not a function decorator", decoratorName)
	}

	var args []ast.Expression
	var openParen, closeParen *lexer.Token
	if p.match(lexer.LPAREN) {
		open := p.current()
		openParen = &open
		p.advance() // consume '('
		args, err = p.parseArgumentList()
		if err != nil {
			return nil, err
		}
		close, err := p.consume(lexer.RPAREN, "expected ')' after decorator arguments")
		if err != nil {
			return nil, err
		}
		closeParen = &close
	}

	return &ast.FunctionDecorator{
		Name:       decoratorName,
		Args:       args,
		Pos:        ast.Position{Line: startPos.Line, Column: startPos.Column},
		AtToken:    atToken,
		NameToken:  nameToken,
		OpenParen:  openParen,
		CloseParen: closeParen,
	}, nil
}

// isValidDecoratorName checks if a token can be used as a decorator name
func (p *Parser) isValidDecoratorName(token lexer.Token) bool {
	switch token.Type {
	case lexer.IDENTIFIER:
		return true
	case lexer.VAR:
		// "var" can be used as a decorator name for @var()
		return true
	case lexer.WHEN, lexer.TRY:
		// Pattern decorator keywords
		return true
	default:
		return false
	}
}

// parseArgumentList parses a comma-separated list of expressions.
func (p *Parser) parseArgumentList() ([]ast.Expression, error) {
	var args []ast.Expression
	if p.match(lexer.RPAREN) {
		return args, nil // No arguments
	}

	for {
		arg, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		args = append(args, arg)
		if !p.match(lexer.COMMA) {
			break
		}
		p.advance() // consume ','
	}
	return args, nil
}

// --- Utility and Helper Methods ---

func (p *Parser) advance() lexer.Token {
	if !p.isAtEnd() {
		p.pos++
	}
	return p.previous()
}

func (p *Parser) current() lexer.Token { return p.tokens[p.pos] }
func (p *Parser) previous() lexer.Token { return p.tokens[p.pos-1] }
func (p *Parser) peek() lexer.Token     { return p.tokens[p.pos+1] }

func (p *Parser) isAtEnd() bool { return p.current().Type == lexer.EOF }

func (p *Parser) match(types ...lexer.TokenType) bool {
	for _, t := range types {
		if p.current().Type == t {
			return true
		}
	}
	return false
}

func (p *Parser) consume(t lexer.TokenType, message string) (lexer.Token, error) {
	if p.match(t) {
		tok := p.current()
		p.advance()
		return tok, nil
	}
	return lexer.Token{}, fmt.Errorf("%s (at line %d, col %d, got %s)", message, p.current().Line, p.current().Column, p.current().Type)
}

func (p *Parser) skipNewlines() {
	for p.match(lexer.NEWLINE) {
		p.advance()
	}
}

// isCommandTerminator is the context-aware function to check for end of command.
func (p *Parser) isCommandTerminator(inBlock bool) bool {
	if p.isAtEnd() {
		return true
	}
	if inBlock {
		// In a block, only a '}' terminates the command content.
		return p.match(lexer.RBRACE)
	}
	// In a simple command, a newline terminates it.
	return p.match(lexer.NEWLINE)
}

// isFunctionDecorator checks if the current '@' token starts a function decorator.
func (p *Parser) isFunctionDecorator() bool {
	if !p.match(lexer.AT) {
		return false
	}
	if p.pos+1 < len(p.tokens) {
		nextToken := p.tokens[p.pos+1]
		var name string

		if nextToken.Type == lexer.IDENTIFIER {
			name = nextToken.Value
		} else if nextToken.Type == lexer.VAR {
			name = "var" // Handle @var() case
		} else {
			return false
		}

		return stdlib.IsFunctionDecorator(name)
	}
	return false
}

// isPatternDecorator checks if the current '@' token starts a pattern decorator.
func (p *Parser) isPatternDecorator() bool {
	if !p.match(lexer.AT) {
		return false
	}
	if p.pos+1 < len(p.tokens) {
		nextToken := p.tokens[p.pos+1]
		var name string

		if nextToken.Type == lexer.IDENTIFIER {
			name = nextToken.Value
		} else if nextToken.Type == lexer.WHEN {
			name = "when"
		} else if nextToken.Type == lexer.TRY {
			name = "try"
		} else {
			return false
		}

		return stdlib.IsPatternDecorator(name)
	}
	return false
}

// addError records an error and allows parsing to continue.
func (p *Parser) addError(err error) {
	p.errors = append(p.errors, err.Error())
}

// synchronize advances the parser until it finds a probable statement boundary,
// allowing it to recover from an error and report more than one error per file.
func (p *Parser) synchronize() {
	p.advance()
	for !p.isAtEnd() {
		// A newline is a good place to resume.
		if p.previous().Type == lexer.NEWLINE {
			return
		}
		// A new top-level keyword is also a good place.
		switch p.current().Type {
		case lexer.VAR, lexer.WATCH, lexer.STOP:
			return
		}
		p.advance()
	}
}
