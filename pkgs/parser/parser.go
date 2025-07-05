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
// **CRITICAL FIX**: Now handles decorator syntax sugar correctly.
// CommandBody = "{" CommandContent "}" | DecoratorSugar | CommandContent
func (p *Parser) parseCommandBody() (*ast.CommandBody, error) {
	startPos := p.current()

	// **FIX**: Check for decorator syntax sugar: @decorator(args) { ... }
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
				if stdlib.IsBlockDecorator(decorator.Name) {
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
// shell text, decorators, or nested content.
// It is context-aware via the `inBlock` parameter.
func (p *Parser) parseCommandContent(inBlock bool) (ast.CommandContent, error) {
	startPos := p.current()

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

// parseShellContent consumes tokens as shell content until a terminator is found.
// This is where the parser "trusts the lexer" by concatenating IDENTIFIER tokens
// without modification.
// ShellContent = { TextPart | FunctionDecorator }*
func (p *Parser) parseShellContent(inBlock bool) (*ast.ShellContent, error) {
	startPos := p.current()
	var parts []ast.ShellPart
	var textBuilder strings.Builder

	// Flush the current text builder into a TextPart node.
	flushText := func() {
		if textBuilder.Len() > 0 {
			parts = append(parts, &ast.TextPart{Text: textBuilder.String()})
			textBuilder.Reset()
		}
	}

	for !p.isCommandTerminator(inBlock) {
		if p.isFunctionDecorator() {
			flushText() // Finalize any preceding text
			decorator, err := p.parseFunctionDecorator()
			if err != nil {
				return nil, err
			}
			parts = append(parts, decorator)
			continue
		}

		// This is the core "trust the lexer" logic.
		// We append the token's value directly, preserving all whitespace.
		textBuilder.WriteString(p.current().Value)
		p.advance()
	}

	flushText() // Add any remaining text

	return &ast.ShellContent{
		Parts: parts,
		Pos:   ast.Position{Line: startPos.Line, Column: startPos.Column},
	}, nil
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
// **FIX**: This version is now robust and handles nested parentheses correctly,
// ensuring it consumes the entire intended argument without overrunning.
func (p *Parser) parseDecoratorArgument() (ast.Expression, error) {
	startPos := p.current()
	var buffer strings.Builder
	parenDepth := 0

	for !p.isAtEnd() {
		curr := p.current()

		// Terminate if we see a comma or closing paren at the top level of the argument.
		if (curr.Type == lexer.COMMA || curr.Type == lexer.RPAREN) && parenDepth == 0 {
			break
		}

		if curr.Type == lexer.LPAREN {
			parenDepth++
		} else if curr.Type == lexer.RPAREN {
			parenDepth--
		}

		buffer.WriteString(curr.Value)
		p.advance()
	}

	value := buffer.String()
	// The test suite expects certain arguments to be treated as a single identifier.
	// E.g., for @sh, "ls @var(SRC) | wc -l" is a single identifier.
	// For @env, "NODE_ENV=@var(ENV)" is also a single identifier.
	return &ast.Identifier{
		Name:  value,
		Token: lexer.Token{Value: value, Line: startPos.Line, Column: startPos.Column},
	}, nil
}


// isDecoratorArgumentTerminator checks if we've reached the end of a decorator argument
func (p *Parser) isDecoratorArgumentTerminator() bool {
	switch p.current().Type {
	case lexer.COMMA, lexer.RPAREN, lexer.EOF:
		return true
	default:
		return false
	}
}

// --- Variable Parsing ---

// parseVariableDecl parses a variable declaration.
// It contains its own logic for parsing the variable's value to correctly handle
// termination at newlines.
func (p *Parser) parseVariableDecl() (*ast.VariableDecl, error) {
	startPos := p.current()
	p.consume(lexer.VAR, "expected 'var'")

	name, err := p.consume(lexer.IDENTIFIER, "expected variable name")
	if err != nil {
		return nil, err
	}
	p.consume(lexer.EQUALS, "expected '=' after variable name")

	// --- Custom Value Parsing Logic for Variables ---
	var value ast.Expression
	var errVal error

	switch p.current().Type {
	case lexer.STRING:
		tok := p.current()
		p.advance()
		value = &ast.StringLiteral{Value: tok.Value, Raw: tok.Raw, StringToken: tok}
	case lexer.NUMBER:
		tok := p.current()
		p.advance()
		value = &ast.NumberLiteral{Value: tok.Value, Token: tok}
	case lexer.DURATION:
		tok := p.current()
		p.advance()
		value = &ast.DurationLiteral{Value: tok.Value, Token: tok}
	case lexer.IDENTIFIER:
		// CRITICAL FIX: For variables, an identifier is a single token.
		tok := p.current()
		p.advance()
		value = &ast.Identifier{Name: tok.Value, Token: tok}
	case lexer.AT:
		if p.isFunctionDecorator() {
			value, errVal = p.parseFunctionDecorator()
		} else {
			errVal = fmt.Errorf("unexpected block decorator in variable value")
		}
	default:
		errVal = fmt.Errorf("unexpected token %s, expected a variable value", p.current().Type)
	}

	if errVal != nil {
		return nil, errVal
	}
	// --- End Custom Logic ---

	return &ast.VariableDecl{
		Name:      name.Value,
		Value:     value,
		Pos:       ast.Position{Line: startPos.Line, Column: startPos.Column},
		NameToken: name,
	}, nil
}

func (p *Parser) parseVarGroup() (*ast.VarGroup, error) {
	startPos := p.current()
	p.consume(lexer.VAR, "expected 'var'")
	openParen, _ := p.consume(lexer.LPAREN, "expected '(' for var group")

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
	p.consume(lexer.EQUALS, "expected '=' after variable name")

	// Use the same custom value parsing logic as parseVariableDecl
	var value ast.Expression
	var errVal error

	switch p.current().Type {
	case lexer.STRING:
		tok := p.current()
		p.advance()
		value = &ast.StringLiteral{Value: tok.Value, Raw: tok.Raw, StringToken: tok}
	case lexer.NUMBER:
		tok := p.current()
		p.advance()
		value = &ast.NumberLiteral{Value: tok.Value, Token: tok}
	case lexer.DURATION:
		tok := p.current()
		p.advance()
		value = &ast.DurationLiteral{Value: tok.Value, Token: tok}
	case lexer.IDENTIFIER:
		tok := p.current()
		p.advance()
		value = &ast.Identifier{Name: tok.Value, Token: tok}
	case lexer.AT:
		if p.isFunctionDecorator() {
			value, errVal = p.parseFunctionDecorator()
		} else {
			errVal = fmt.Errorf("unexpected block decorator in variable value")
		}
	default:
		errVal = fmt.Errorf("unexpected token %s, expected a variable value", p.current().Type)
	}

	if errVal != nil {
		return nil, errVal
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
	for p.match(lexer.AT) && !p.isFunctionDecorator() {
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
	nameToken, err := p.consume(lexer.IDENTIFIER, "expected decorator name")
	if err != nil {
		return nil, err
	}
	if !stdlib.IsBlockDecorator(nameToken.Value) {
		return nil, fmt.Errorf("@%s is not a block decorator", nameToken.Value)
	}

	var args []ast.Expression
	if p.match(lexer.LPAREN) {
		p.advance() // consume '('
		args, err = p.parseArgumentList()
		if err != nil {
			return nil, err
		}
		p.consume(lexer.RPAREN, "expected ')' after decorator arguments")
	}

	return &ast.Decorator{
		Name:      nameToken.Value,
		Args:      args,
		Pos:       ast.Position{Line: startPos.Line, Column: startPos.Column},
		AtToken:   atToken,
		NameToken: nameToken,
	}, nil
}

func (p *Parser) parseFunctionDecorator() (*ast.FunctionDecorator, error) {
	startPos := p.current()
	atToken, _ := p.consume(lexer.AT, "expected '@'")
	nameToken, err := p.consume(lexer.IDENTIFIER, "expected decorator name")
	if err != nil {
		return nil, err
	}
	if !stdlib.IsFunctionDecorator(nameToken.Value) {
		return nil, fmt.Errorf("@%s is not a function decorator", nameToken.Value)
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
		Name:       nameToken.Value,
		Args:       args,
		Pos:        ast.Position{Line: startPos.Line, Column: startPos.Column},
		AtToken:    atToken,
		NameToken:  nameToken,
		OpenParen:  openParen,
		CloseParen: closeParen,
	}, nil
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
		name := p.tokens[p.pos+1].Value
		return stdlib.IsFunctionDecorator(name)
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
