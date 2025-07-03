package parser

import (
	"fmt"
	"strings"

	"github.com/aledsdavies/devcmd/pkgs/ast"
	"github.com/aledsdavies/devcmd/pkgs/lexer"
	"github.com/aledsdavies/devcmd/pkgs/stdlib"
)

// Parser implements a recursive descent parser for the Devcmd language
type Parser struct {
	lexer    *lexer.Lexer
	current  lexer.Token
	previous lexer.Token
	tokens   []lexer.Token
	pos      int
	inVariableValueContext bool // Track if we're parsing a variable value
}

// Parse creates a parser and parses the input string into an AST
func Parse(input string) (*ast.Program, error) {
	lex := lexer.New(input)
	tokens := lex.TokenizeToSlice()

	parser := &Parser{
		lexer:  lex,
		tokens: tokens,
		pos:    0,
	}

	if len(tokens) > 0 {
		parser.current = tokens[0]
	}

	return parser.parseProgram()
}

// advance moves to the next token
func (p *Parser) advance() {
	if p.pos < len(p.tokens)-1 {
		p.previous = p.current
		p.pos++
		p.current = p.tokens[p.pos]
	}
}

// peek returns the next token without consuming it
func (p *Parser) peek() lexer.Token {
	if p.pos+1 < len(p.tokens) {
		return p.tokens[p.pos+1]
	}
	return lexer.Token{Type: lexer.EOF}
}

// match checks if current token matches any of the given types
func (p *Parser) match(types ...lexer.TokenType) bool {
	for _, t := range types {
		if p.current.Type == t {
			return true
		}
	}
	return false
}

// consume advances if current token matches, otherwise returns error
func (p *Parser) consume(tokenType lexer.TokenType, message string) error {
	if p.current.Type == tokenType {
		p.advance()
		return nil
	}
	return fmt.Errorf("%s at line %d, column %d: expected %s, got %s",
		message, p.current.Line, p.current.Column, tokenType, p.current.Type)
}

// synchronize recovers from parse errors by finding the next statement boundary
func (p *Parser) synchronize() {
	for p.current.Type != lexer.EOF {
		if p.current.Type == lexer.NEWLINE {
			p.advance()
			return
		}
		p.advance()
	}
}

// parseProgram parses the entire program
func (p *Parser) parseProgram() (*ast.Program, error) {
	program := &ast.Program{
		Variables: []ast.VariableDecl{},
		VarGroups: []ast.VarGroup{},
		Commands:  []ast.CommandDecl{},
	}

	// Skip initial newlines and comments
	p.skipWhitespaceAndComments()

	for p.current.Type != lexer.EOF {
		switch p.current.Type {
		case lexer.VAR:
			if p.peek().Type == lexer.LPAREN {
				// Grouped variables: var ( ... )
				varGroup, err := p.parseVarGroup()
				if err != nil {
					return nil, err
				}
				program.VarGroups = append(program.VarGroups, *varGroup)
			} else {
				// Individual variable: var NAME = VALUE
				varDecl, err := p.parseVariableDecl()
				if err != nil {
					return nil, err
				}
				program.Variables = append(program.Variables, *varDecl)
			}
		case lexer.WATCH, lexer.STOP, lexer.IDENTIFIER:
			// Commands
			command, err := p.parseCommand()
			if err != nil {
				return nil, err
			}
			program.Commands = append(program.Commands, *command)
		case lexer.NEWLINE, lexer.COMMENT, lexer.MULTILINE_COMMENT:
			p.advance()
		default:
			return nil, fmt.Errorf("unexpected token %s at line %d, column %d",
				p.current.Type, p.current.Line, p.current.Column)
		}
	}

	return program, nil
}

// parseVariableDecl parses: var NAME = VALUE
func (p *Parser) parseVariableDecl() (*ast.VariableDecl, error) {
	startPos := p.current

	if err := p.consume(lexer.VAR, "expected 'var'"); err != nil {
		return nil, err
	}

	if !p.match(lexer.IDENTIFIER) {
		return nil, fmt.Errorf("expected variable name at line %d, column %d",
			p.current.Line, p.current.Column)
	}

	nameToken := p.current
	name := p.current.Value
	p.advance()

	if err := p.consume(lexer.EQUALS, "expected '=' after variable name"); err != nil {
		return nil, err
	}

	// Set variable value context
	p.inVariableValueContext = true
	value, err := p.parseExpression()
	p.inVariableValueContext = false
	if err != nil {
		return nil, err
	}

	// Skip optional newline
	if p.match(lexer.NEWLINE) {
		p.advance()
	}

	return &ast.VariableDecl{
		Name:       name,
		Value:      value,
		Pos:        ast.Position{Line: startPos.Line, Column: startPos.Column},
		NameToken:  nameToken,
		ValueToken: p.previous,
	}, nil
}

// parseVarGroup parses: var ( NAME1 = VALUE1; NAME2 = VALUE2; ... )
func (p *Parser) parseVarGroup() (*ast.VarGroup, error) {
	startPos := p.current

	varToken := p.current
	if err := p.consume(lexer.VAR, "expected 'var'"); err != nil {
		return nil, err
	}

	openParen := p.current
	if err := p.consume(lexer.LPAREN, "expected '(' after 'var'"); err != nil {
		return nil, err
	}

	var variables []ast.VariableDecl
	p.skipWhitespaceAndComments()

	for !p.match(lexer.RPAREN) && p.current.Type != lexer.EOF {
		if !p.match(lexer.IDENTIFIER) {
			return nil, fmt.Errorf("expected variable name at line %d, column %d",
				p.current.Line, p.current.Column)
		}

		nameToken := p.current
		name := p.current.Value
		p.advance()

		if err := p.consume(lexer.EQUALS, "expected '=' after variable name"); err != nil {
			return nil, err
		}

		// Set variable value context
		p.inVariableValueContext = true
		value, err := p.parseExpression()
		p.inVariableValueContext = false
		if err != nil {
			return nil, err
		}

		variables = append(variables, ast.VariableDecl{
			Name:       name,
			Value:      value,
			Pos:        ast.Position{Line: nameToken.Line, Column: nameToken.Column},
			NameToken:  nameToken,
			ValueToken: p.previous,
		})

		p.skipWhitespaceAndComments()
	}

	closeParen := p.current
	if err := p.consume(lexer.RPAREN, "expected ')' after variable group"); err != nil {
		return nil, err
	}

	// Skip optional newline
	if p.match(lexer.NEWLINE) {
		p.advance()
	}

	return &ast.VarGroup{
		Variables:  variables,
		Pos:        ast.Position{Line: startPos.Line, Column: startPos.Column},
		VarToken:   varToken,
		OpenParen:  openParen,
		CloseParen: closeParen,
	}, nil
}

// parseCommand parses: [watch|stop] NAME: BODY
func (p *Parser) parseCommand() (*ast.CommandDecl, error) {
	startPos := p.current

	var cmdType ast.CommandType = ast.Command
	var typeToken *lexer.Token

	// Check for watch/stop keywords
	if p.match(lexer.WATCH) {
		cmdType = ast.WatchCommand
		token := p.current
		typeToken = &token
		p.advance()
	} else if p.match(lexer.STOP) {
		cmdType = ast.StopCommand
		token := p.current
		typeToken = &token
		p.advance()
	}

	if !p.match(lexer.IDENTIFIER) {
		return nil, fmt.Errorf("expected command name at line %d, column %d",
			p.current.Line, p.current.Column)
	}

	nameToken := p.current
	name := p.current.Value
	p.advance()

	colonToken := p.current
	if err := p.consume(lexer.COLON, "expected ':' after command name"); err != nil {
		return nil, err
	}

	body, err := p.parseCommandBody()
	if err != nil {
		return nil, err
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

// parseCommandBody parses the body of a command (simple or block)
func (p *Parser) parseCommandBody() (*ast.CommandBody, error) {
	startPos := p.current

	// Skip whitespace after colon
	for p.match(lexer.NEWLINE) {
		p.advance()
	}

	// Check for explicit block: { ... }
	if p.match(lexer.LBRACE) {
		return p.parseExplicitBlock()
	}

	// Parse simple command (with potential syntax sugar)
	content, err := p.parseCommandContent()
	if err != nil {
		return nil, err
	}

	return &ast.CommandBody{
		Content: content,
		IsBlock: false,
		Pos:     ast.Position{Line: startPos.Line, Column: startPos.Column},
	}, nil
}

// parseExplicitBlock parses: { content }
func (p *Parser) parseExplicitBlock() (*ast.CommandBody, error) {
	startPos := p.current

	openBrace := p.current
	if err := p.consume(lexer.LBRACE, "expected '{'"); err != nil {
		return nil, err
	}

	p.skipWhitespaceAndComments()

	content, err := p.parseCommandContent()
	if err != nil {
		return nil, err
	}

	p.skipWhitespaceAndComments()

	closeBrace := p.current
	if err := p.consume(lexer.RBRACE, "expected '}'"); err != nil {
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

// parseCommandContent parses the content within a command body
func (p *Parser) parseCommandContent() (ast.CommandContent, error) {
	// Check for decorators first
	if p.match(lexer.AT) {
		return p.parseDecoratedContent()
	}

	// Parse shell content
	return p.parseShellContent()
}

// parseDecoratedContent parses: @decorator1 @decorator2 ... content
func (p *Parser) parseDecoratedContent() (*ast.DecoratedContent, error) {
	startPos := p.current

	var decorators []ast.Decorator

	// Parse all leading decorators
	for p.match(lexer.AT) {
		decorator, err := p.parseDecorator()
		if err != nil {
			return nil, err
		}
		decorators = append(decorators, *decorator)
	}

	// Parse the content that follows
	content, err := p.parseCommandContent()
	if err != nil {
		return nil, err
	}

	return &ast.DecoratedContent{
		Decorators: decorators,
		Content:    content,
		Pos:        ast.Position{Line: startPos.Line, Column: startPos.Column},
	}, nil
}

// parseDecorator parses: @name or @name(args)
func (p *Parser) parseDecorator() (*ast.Decorator, error) {
	startPos := p.current

	atToken := p.current
	if err := p.consume(lexer.AT, "expected '@'"); err != nil {
		return nil, err
	}

	if !p.match(lexer.IDENTIFIER) {
		return nil, fmt.Errorf("expected decorator name at line %d, column %d",
			p.current.Line, p.current.Column)
	}

	nameToken := p.current
	name := p.current.Value
	p.advance()

	// Validate decorator exists
	if !stdlib.IsValidDecorator(name) {
		return nil, fmt.Errorf("unknown decorator @%s at line %d, column %d",
			name, nameToken.Line, nameToken.Column)
	}

	// Validate this is a block decorator
	if !stdlib.IsBlockDecorator(name) {
		return nil, fmt.Errorf("@%s is not a block decorator at line %d, column %d",
			name, nameToken.Line, nameToken.Column)
	}

	var args []ast.Expression

	// Check for arguments: @name(arg1, arg2, ...)
	if p.match(lexer.LPAREN) {
		p.advance()

		// Parse arguments
		for !p.match(lexer.RPAREN) && p.current.Type != lexer.EOF {
			arg, err := p.parseExpression()
			if err != nil {
				return nil, err
			}
			args = append(args, arg)

			// Comma separator
			if p.match(lexer.COMMA) {
				p.advance()
			} else if !p.match(lexer.RPAREN) {
				return nil, fmt.Errorf("expected ',' or ')' in decorator arguments at line %d, column %d",
					p.current.Line, p.current.Column)
			}
		}

		if err := p.consume(lexer.RPAREN, "expected ')' after decorator arguments"); err != nil {
			return nil, err
		}
	}

	// TODO: Validate arguments match decorator signature
	// This would require converting ast.Expression to string for validation

	return &ast.Decorator{
		Name:      name,
		Args:      args,
		Pos:       ast.Position{Line: startPos.Line, Column: startPos.Column},
		AtToken:   atToken,
		NameToken: nameToken,
	}, nil
}

// parseShellContent parses mixed shell content with text and function decorators
func (p *Parser) parseShellContent() (*ast.ShellContent, error) {
	startPos := p.current

	var parts []ast.ShellPart

	// Parse until we hit a terminator
	for !p.isCommandTerminator() {
		if p.match(lexer.AT) {
			// Check if this is a function decorator
			if p.isFunctionDecorator() {
				funcDec, err := p.parseFunctionDecorator()
				if err != nil {
					return nil, err
				}
				parts = append(parts, funcDec)
			} else {
				// This @ is part of regular text (like email)
				textPart, err := p.parseTextPart()
				if err != nil {
					return nil, err
				}
				parts = append(parts, textPart)
			}
		} else {
			// Regular text
			textPart, err := p.parseTextPart()
			if err != nil {
				return nil, err
			}
			parts = append(parts, textPart)
		}
	}

	return &ast.ShellContent{
		Parts: parts,
		Pos:   ast.Position{Line: startPos.Line, Column: startPos.Column},
	}, nil
}

// parseFunctionDecorator parses: @name(args) - inline function decorators
func (p *Parser) parseFunctionDecorator() (*ast.FunctionDecorator, error) {
	startPos := p.current

	atToken := p.current
	if err := p.consume(lexer.AT, "expected '@'"); err != nil {
		return nil, err
	}

	if !p.match(lexer.IDENTIFIER) {
		return nil, fmt.Errorf("expected function decorator name at line %d, column %d",
			p.current.Line, p.current.Column)
	}

	nameToken := p.current
	name := p.current.Value
	p.advance()

	// Validate decorator exists
	if !stdlib.IsValidDecorator(name) {
		return nil, fmt.Errorf("unknown decorator @%s at line %d, column %d",
			name, nameToken.Line, nameToken.Column)
	}

	// Validate this is a function decorator
	if !stdlib.IsFunctionDecorator(name) {
		return nil, fmt.Errorf("@%s is not a function decorator at line %d, column %d",
			name, nameToken.Line, nameToken.Column)
	}

	var args []ast.Expression
	var openParen, closeParen *lexer.Token

	// Parse arguments if present
	if p.match(lexer.LPAREN) {
		open := p.current
		openParen = &open
		p.advance()

		// Parse arguments
		for !p.match(lexer.RPAREN) && p.current.Type != lexer.EOF {
			arg, err := p.parseExpression()
			if err != nil {
				return nil, err
			}
			args = append(args, arg)

			// Comma separator
			if p.match(lexer.COMMA) {
				p.advance()
			} else if !p.match(lexer.RPAREN) {
				return nil, fmt.Errorf("expected ',' or ')' in function decorator arguments at line %d, column %d",
					p.current.Line, p.current.Column)
			}
		}

		close := p.current
		closeParen = &close
		if err := p.consume(lexer.RPAREN, "expected ')' after function decorator arguments"); err != nil {
			return nil, err
		}
	}

	// TODO: Validate arguments match decorator signature

	return &ast.FunctionDecorator{
		Name:       name,
		Args:       args,
		Pos:        ast.Position{Line: startPos.Line, Column: startPos.Column},
		AtToken:    atToken,
		NameToken:  nameToken,
		OpenParen:  openParen,
		CloseParen: closeParen,
	}, nil
}

// parseTextPart parses plain text within shell content
func (p *Parser) parseTextPart() (*ast.TextPart, error) {
	startPos := p.current

	var textBuilder strings.Builder

	for !p.isCommandTerminator() && !p.isFunctionDecorator() {
		if p.match(lexer.LINE_CONT) {
			// Handle line continuation: \ + newline
			p.advance()
			textBuilder.WriteRune(' ') // Replace continuation with space
		} else {
			textBuilder.WriteString(p.current.Value)
			p.advance()
		}

		// Stop if we hit a function decorator
		if p.match(lexer.AT) && p.isFunctionDecorator() {
			break
		}
	}

	text := textBuilder.String()
	// Only trim leading whitespace, preserve trailing spaces for shell commands
	text = strings.TrimLeft(text, " \t\r\f")

	return &ast.TextPart{
		Text: text,
		Pos:  ast.Position{Line: startPos.Line, Column: startPos.Column},
	}, nil
}

// parseExpression parses literals, identifiers, and function decorators
func (p *Parser) parseExpression() (ast.Expression, error) {
	switch p.current.Type {
	case lexer.STRING:
		return p.parseStringLiteral()
	case lexer.NUMBER:
		return p.parseNumberLiteral()
	case lexer.DURATION:
		return p.parseDurationLiteral()
	case lexer.IDENTIFIER:
		// In variable value context, treat identifiers as string literals
		if p.inVariableValueContext {
			return p.parseIdentifierAsStringLiteral()
		}
		return p.parseIdentifier()
	case lexer.AT:
		if p.isFunctionDecorator() {
			return p.parseFunctionDecorator()
		}
		return nil, fmt.Errorf("unexpected '@' at line %d, column %d",
			p.current.Line, p.current.Column)
	default:
		return nil, fmt.Errorf("unexpected token in expression: %s at line %d, column %d",
			p.current.Type, p.current.Line, p.current.Column)
	}
}

// parseIdentifierAsStringLiteral parses an identifier token as a string literal
func (p *Parser) parseIdentifierAsStringLiteral() (*ast.StringLiteral, error) {
	startPos := p.current

	token := p.current
	value := p.current.Value
	p.advance()

	return &ast.StringLiteral{
		Value:       value,
		Raw:         value, // For unquoted identifiers, raw and value are the same
		Pos:         ast.Position{Line: startPos.Line, Column: startPos.Column},
		StringToken: token,
	}, nil
}

// parseStringLiteral parses string literals
func (p *Parser) parseStringLiteral() (*ast.StringLiteral, error) {
	startPos := p.current

	token := p.current
	value := p.current.Value
	p.advance()

	return &ast.StringLiteral{
		Value:       value,
		Raw:         token.Raw,
		Pos:         ast.Position{Line: startPos.Line, Column: startPos.Column},
		StringToken: token,
	}, nil
}

// parseNumberLiteral parses number literals
func (p *Parser) parseNumberLiteral() (*ast.NumberLiteral, error) {
	startPos := p.current

	token := p.current
	value := p.current.Value
	p.advance()

	return &ast.NumberLiteral{
		Value: value,
		Pos:   ast.Position{Line: startPos.Line, Column: startPos.Column},
		Token: token,
	}, nil
}

// parseDurationLiteral parses duration literals
func (p *Parser) parseDurationLiteral() (*ast.DurationLiteral, error) {
	startPos := p.current

	token := p.current
	value := p.current.Value
	p.advance()

	return &ast.DurationLiteral{
		Value: value,
		Pos:   ast.Position{Line: startPos.Line, Column: startPos.Column},
		Token: token,
	}, nil
}

// parseIdentifier parses identifiers
func (p *Parser) parseIdentifier() (*ast.Identifier, error) {
	startPos := p.current

	token := p.current
	name := p.current.Value
	p.advance()

	return &ast.Identifier{
		Name:  name,
		Pos:   ast.Position{Line: startPos.Line, Column: startPos.Column},
		Token: token,
	}, nil
}

// Helper methods

// skipWhitespaceAndComments skips whitespace and comments
func (p *Parser) skipWhitespaceAndComments() {
	for p.match(lexer.NEWLINE, lexer.COMMENT, lexer.MULTILINE_COMMENT) {
		p.advance()
	}
}

// isCommandTerminator checks if we've reached the end of a command
func (p *Parser) isCommandTerminator() bool {
	return p.match(lexer.NEWLINE, lexer.RBRACE, lexer.EOF)
}

// isFunctionDecorator checks if @ starts a function decorator
func (p *Parser) isFunctionDecorator() bool {
	if !p.match(lexer.AT) {
		return false
	}

	// Look ahead to see if it's @identifier
	if p.pos+1 < len(p.tokens) && p.tokens[p.pos+1].Type == lexer.IDENTIFIER {
		name := p.tokens[p.pos+1].Value

		// Check if it's a known function decorator
		if stdlib.IsFunctionDecorator(name) {
			return true
		}

		// Check if it has parentheses (likely a function decorator)
		if p.pos+2 < len(p.tokens) && p.tokens[p.pos+2].Type == lexer.LPAREN {
			return stdlib.IsValidDecorator(name)
		}
	}

	return false
}
