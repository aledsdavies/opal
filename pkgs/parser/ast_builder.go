package parser

import (
	"strconv"
	"strings"

	"github.com/aledsdavies/devcmd/pkgs/ast"
	"github.com/aledsdavies/devcmd/pkgs/lexer"
)

// buildAST performs Pass 2: construct the full AST from structure map
func (p *Parser) buildAST() *ast.Program {
	program := &ast.Program{
		Variables: make([]ast.VariableDecl, 0, len(p.structure.Variables)),
		Commands:  make([]ast.CommandDecl, 0, len(p.structure.Commands)),
		Pos: ast.Position{
			Line:   1,
			Column: 1,
			Offset: 0,
		},
	}

	// Build variables
	for _, varSpan := range p.structure.Variables {
		if varDecl := p.buildVariable(varSpan); varDecl != nil {
			program.Variables = append(program.Variables, *varDecl)
		}
	}

	// Build commands - unified handling for all command types
	for _, cmdSpan := range p.structure.Commands {
		if cmdDecl := p.buildCommand(cmdSpan); cmdDecl != nil {
			program.Commands = append(program.Commands, *cmdDecl)
		}
	}

	// Set program token range
	if len(p.tokens) > 0 {
		program.Tokens = ast.TokenRange{
			Start: p.tokens[0],
			End:   p.tokens[len(p.tokens)-1],
			All:   p.tokens,
		}
	}

	return program
}

// buildVariable constructs a VariableDecl from a VariableSpan
func (p *Parser) buildVariable(span VariableSpan) *ast.VariableDecl {
	if span.ValueStart > span.ValueEnd || span.ValueEnd >= len(p.tokens) {
		p.addError(span.NameToken, "invalid variable value range",
			"variable declaration", "check variable syntax")
		return nil
	}

	// Parse the variable value
	valueTokens := p.tokens[span.ValueStart : span.ValueEnd+1]
	value := p.buildExpression(valueTokens)

	if value == nil {
		p.addError(span.NameToken, "could not parse variable value",
			"variable declaration", "check value syntax")
		return nil
	}

	// Build token range for the entire variable declaration
	startIndex := span.ValueStart - 3 // account for 'var NAME ='
	if span.IsGrouped {
		startIndex = span.GroupStart
	}

	var allTokens []lexer.Token
	if startIndex >= 0 && span.ValueEnd < len(p.tokens) {
		allTokens = p.tokens[startIndex : span.ValueEnd+1]
	}

	return &ast.VariableDecl{
		Name:  span.NameToken.Value,
		Value: value,
		Pos: ast.Position{
			Line:   span.NameToken.Line,
			Column: span.NameToken.Column,
			Offset: 0,
		},
		Tokens: ast.TokenRange{
			Start: span.NameToken,
			End:   p.tokens[span.ValueEnd],
			All:   allTokens,
		},
		NameToken:  span.NameToken,
		ValueToken: p.tokens[span.ValueStart],
	}
}

// buildCommand constructs a CommandDecl from a CommandSpan
func (p *Parser) buildCommand(span CommandSpan) *ast.CommandDecl {
	// Determine command type based on type token
	cmdType := ast.Command
	if span.TypeToken.Type == lexer.WATCH {
		cmdType = ast.WatchCommand
	} else if span.TypeToken.Type == lexer.STOP {
		cmdType = ast.StopCommand
	}

	// Build decorators
	decorators := make([]ast.Decorator, 0, len(span.Decorators))
	for _, decoratorIndex := range span.Decorators {
		if decoratorIndex < len(p.structure.Decorators) {
			decoratorSpan := p.structure.Decorators[decoratorIndex]
			if decorator := p.buildDecorator(decoratorSpan); decorator != nil {
				decorators = append(decorators, *decorator)
			}
		}
	}

	// Build command body
	body := p.buildCommandBody(span)
	if body == nil {
		p.addError(span.NameToken, "could not parse command body",
			"command declaration", "check command syntax")
		return nil
	}

	// Calculate token range
	startToken := span.NameToken
	if span.TypeToken.Type != lexer.ILLEGAL {
		startToken = span.TypeToken
	}

	endToken := p.tokens[span.BodyEnd]
	if span.BodyEnd >= len(p.tokens) {
		endToken = p.tokens[len(p.tokens)-1]
	}

	var allTokens []lexer.Token
	startIndex := p.findTokenIndex(startToken)
	endIndex := span.BodyEnd
	if startIndex >= 0 && endIndex < len(p.tokens) {
		allTokens = p.tokens[startIndex : endIndex+1]
	}

	return &ast.CommandDecl{
		Name:       span.NameToken.Value,
		Type:       cmdType,
		Decorators: decorators,
		Body:       *body,
		Pos: ast.Position{
			Line:   span.NameToken.Line,
			Column: span.NameToken.Column,
		},
		Tokens: ast.TokenRange{
			Start: startToken,
			End:   endToken,
			All:   allTokens,
		},
		TypeToken: span.TypeToken,
		NameToken: span.NameToken,
	}
}

// buildCommandBody constructs a unified CommandBody
func (p *Parser) buildCommandBody(span CommandSpan) *ast.CommandBody {
	if span.BodyStart > span.BodyEnd || span.BodyStart >= len(p.tokens) {
		return &ast.CommandBody{
			Statements: []ast.Statement{},
			IsBlock:    span.IsBlock,
			Pos: ast.Position{
				Line:   span.NameToken.Line,
				Column: span.NameToken.Column,
			},
		}
	}

	if span.IsBlock {
		if span.BodyStart < len(p.tokens) && p.tokens[span.BodyStart].Type == lexer.LBRACE {
			return p.buildExplicitBlockCommandBody(span)
		} else {
			return p.buildImplicitBlockCommandBody(span)
		}
	}

	return p.buildSimpleCommandBody(span)
}

// buildSimpleCommandBody constructs a CommandBody for simple commands
func (p *Parser) buildSimpleCommandBody(span CommandSpan) *ast.CommandBody {
	if span.BodyStart > span.BodyEnd {
		return &ast.CommandBody{
			Statements: []ast.Statement{},
			IsBlock:    false,
			Pos: ast.Position{
				Line:   span.NameToken.Line,
				Column: span.NameToken.Column,
			},
		}
	}

	elements := p.buildCommandElements(span.BodyStart, span.BodyEnd)

	var tokens []lexer.Token
	if span.BodyStart <= span.BodyEnd && span.BodyEnd < len(p.tokens) {
		tokens = p.tokens[span.BodyStart : span.BodyEnd+1]
	}

	stmt := &ast.ShellStatement{
		Elements: elements,
		Pos: ast.Position{
			Line:   p.tokens[span.BodyStart].Line,
			Column: p.tokens[span.BodyStart].Column,
		},
		Tokens: ast.TokenRange{
			Start: p.tokens[span.BodyStart],
			End:   p.tokens[span.BodyEnd],
			All:   tokens,
		},
	}

	return &ast.CommandBody{
		Statements: []ast.Statement{stmt},
		IsBlock:    false,
		Pos: ast.Position{
			Line:   span.NameToken.Line,
			Column: span.NameToken.Column,
		},
		Tokens: ast.TokenRange{
			Start: p.tokens[span.BodyStart],
			End:   p.tokens[span.BodyEnd],
			All:   tokens,
		},
	}
}

// buildExplicitBlockCommandBody handles explicit blocks
func (p *Parser) buildExplicitBlockCommandBody(span CommandSpan) *ast.CommandBody {
	var blockRange *BlockRange
	for i := range p.structure.BlockRanges {
		br := &p.structure.BlockRanges[i]
		if br.StartIndex == span.BodyStart {
			blockRange = br
			break
		}
	}

	if blockRange == nil {
		if span.BodyStart < len(p.tokens) && p.tokens[span.BodyStart].Type == lexer.LBRACE {
			braceEnd := p.findMatchingBrace(span.BodyStart)
			if braceEnd > span.BodyStart {
				statements := p.parseBlockStatementsManual(span.BodyStart+1, braceEnd)
				return &ast.CommandBody{
					Statements: statements,
					IsBlock:    true,
					Pos: ast.Position{
						Line:   p.tokens[span.BodyStart].Line,
						Column: p.tokens[span.BodyStart].Column,
					},
					Tokens: ast.TokenRange{
						Start: p.tokens[span.BodyStart],
						End:   p.tokens[braceEnd],
						All:   p.tokens[span.BodyStart : braceEnd+1],
					},
					OpenBrace:  &p.tokens[span.BodyStart],
					CloseBrace: &p.tokens[braceEnd],
				}
			}
		}

		return &ast.CommandBody{
			Statements: []ast.Statement{},
			IsBlock:    true,
			Pos: ast.Position{Line: span.NameToken.Line, Column: span.NameToken.Column},
		}
	}

	statements := make([]ast.Statement, 0, len(blockRange.Statements))
	for _, stmtSpan := range blockRange.Statements {
		if stmt := p.buildStatement(stmtSpan); stmt != nil {
			statements = append(statements, stmt)
		}
	}

	return &ast.CommandBody{
		Statements: statements,
		IsBlock:    true,
		Pos: ast.Position{
			Line:   blockRange.OpenBrace.Line,
			Column: blockRange.OpenBrace.Column,
		},
		Tokens: ast.TokenRange{
			Start: blockRange.OpenBrace,
			End:   blockRange.CloseBrace,
			All:   p.tokens[blockRange.StartIndex : blockRange.EndIndex+1],
		},
		OpenBrace:  &blockRange.OpenBrace,
		CloseBrace: &blockRange.CloseBrace,
	}
}

// parseBlockStatementsManual manually parses block statements
func (p *Parser) parseBlockStatementsManual(start, end int) []ast.Statement {
	statements := []ast.Statement{}

	i := start
	for i < end {
		for i < end && (isWhitespace(p.tokens[i]) || p.tokens[i].Type == lexer.NEWLINE) {
			i++
		}

		if i >= end {
			break
		}

		stmtStart := i
		stmtEnd := i

		for stmtEnd < end {
			if p.tokens[stmtEnd].Type == lexer.NEWLINE {
				stmtEnd--
				break
			}
			if p.tokens[stmtEnd].Type == lexer.IDENTIFIER && p.tokens[stmtEnd].Value == ";" {
				stmtEnd--
				break
			}
			stmtEnd++
		}

		if stmtEnd >= stmtStart {
			elements := p.buildCommandElements(stmtStart, stmtEnd)

			if len(elements) > 0 {
				stmt := &ast.ShellStatement{
					Elements: elements,
					Pos: ast.Position{
						Line:   p.tokens[stmtStart].Line,
						Column: p.tokens[stmtStart].Column,
					},
					Tokens: ast.TokenRange{
						Start: p.tokens[stmtStart],
						End:   p.tokens[stmtEnd],
						All:   p.tokens[stmtStart : stmtEnd+1],
					},
				}
				statements = append(statements, stmt)
			}
		}

		i = stmtEnd + 1
		for i < end && (p.tokens[i].Type == lexer.NEWLINE ||
			(p.tokens[i].Type == lexer.IDENTIFIER && p.tokens[i].Value == ";")) {
			i++
		}
	}

	return statements
}

// buildImplicitBlockCommandBody handles implicit blocks
func (p *Parser) buildImplicitBlockCommandBody(span CommandSpan) *ast.CommandBody {
	return &ast.CommandBody{
		Statements: []ast.Statement{},
		IsBlock:    true,
		Pos: ast.Position{
			Line:   span.NameToken.Line,
			Column: span.NameToken.Column,
		},
	}
}

// buildStatement constructs a Statement
func (p *Parser) buildStatement(span StatementSpan) ast.Statement {
	elements := p.buildCommandElements(span.Start, span.End)

	return &ast.ShellStatement{
		Elements: elements,
		Pos: ast.Position{
			Line:   p.tokens[span.Start].Line,
			Column: p.tokens[span.Start].Column,
		},
		Tokens: ast.TokenRange{
			Start: p.tokens[span.Start],
			End:   p.tokens[span.End],
			All:   p.tokens[span.Start : span.End+1],
		},
	}
}

// buildCommandElements parses tokens into CommandElements including decorators
func (p *Parser) buildCommandElements(start, end int) []ast.CommandElement {
	if start > end || end >= len(p.tokens) {
		return []ast.CommandElement{}
	}

	elements := []ast.CommandElement{}
	i := start

	for i <= end {
		for i <= end && isWhitespace(p.tokens[i]) {
			i++
		}

		if i > end {
			break
		}

		// Check for decorators first
		if decorator, exists := p.decorators[i]; exists {
			elements = append(elements, decorator)
			decoratorEnd := p.findDecoratorEnd(i)
			i = decoratorEnd + 1
			continue
		}

		// Collect text tokens until we hit a special construct
		textTokens := []lexer.Token{}

		for i <= end {
			if _, isDecorator := p.decorators[i]; isDecorator {
				break
			}

			textTokens = append(textTokens, p.tokens[i])
			i++
		}

		if len(textTokens) > 0 {
			text := p.combineTokensToText(textTokens)
			if len(text) > 0 {
				elements = append(elements, &ast.TextElement{
					Text: text,
					Pos: ast.Position{
						Line:   textTokens[0].Line,
						Column: textTokens[0].Column,
					},
					Tokens: ast.TokenRange{
						Start: textTokens[0],
						End:   textTokens[len(textTokens)-1],
						All:   textTokens,
					},
				})
			}
		}
	}

	return elements
}

// buildDecorator constructs a Decorator from a DecoratorSpan with unified args and block support
func (p *Parser) buildDecorator(span DecoratorSpan) *ast.Decorator {
	decorator := &ast.Decorator{
		Name: span.NameToken.Value,
		Args: []ast.Expression{},
		Pos: ast.Position{
			Line:   span.AtToken.Line,
			Column: span.AtToken.Column,
		},
		AtToken:   span.AtToken,
		NameToken: span.NameToken,
	}

	// Build arguments if present
	if span.HasArgs {
		for _, argSpan := range span.Args {
			if argSpan.ValueEnd >= len(p.tokens) {
				continue
			}

			argTokens := p.tokens[argSpan.ValueStart : argSpan.ValueEnd+1]
			expr := p.buildExpression(argTokens)
			if expr != nil {
				if argSpan.IsNamed {
					decorator.Args = append(decorator.Args, &ast.DecoratorArgument{
						Name:  argSpan.Name,
						Value: expr,
						Pos: ast.Position{
							Line:   argTokens[0].Line,
							Column: argTokens[0].Column,
						},
						Tokens: ast.TokenRange{
							Start: argTokens[0],
							End:   argTokens[len(argTokens)-1],
							All:   argTokens,
						},
					})
				} else {
					decorator.Args = append(decorator.Args, expr)
				}
			}
		}
	}

	// Build block if present
	if span.HasBlock {
		decorator.Block = p.buildDecoratorBlock(span)
	}

	// Set token range
	endToken := span.NameToken
	if span.HasArgs && span.ArgsEnd < len(p.tokens) {
		endToken = p.tokens[span.ArgsEnd]
	}
	if span.HasBlock && span.BlockEnd < len(p.tokens) {
		endToken = p.tokens[span.BlockEnd]
	}

	var allTokens []lexer.Token
	if span.StartIndex >= 0 && span.EndIndex < len(p.tokens) {
		allTokens = p.tokens[span.StartIndex : span.EndIndex+1]
	}

	decorator.Tokens = ast.TokenRange{
		Start: span.AtToken,
		End:   endToken,
		All:   allTokens,
	}

	return decorator
}

// buildDecoratorBlock constructs a DecoratorBlock from decorator span
func (p *Parser) buildDecoratorBlock(span DecoratorSpan) *ast.DecoratorBlock {
	if !span.HasBlock {
		return nil
	}

	// Parse statements within the decorator block
	statements := p.parseBlockStatementsManual(span.BlockStart+1, span.BlockEnd)

	return &ast.DecoratorBlock{
		Statements: statements,
		Pos: ast.Position{
			Line:   p.tokens[span.BlockStart].Line,
			Column: p.tokens[span.BlockStart].Column,
		},
		Tokens: ast.TokenRange{
			Start: p.tokens[span.BlockStart],
			End:   p.tokens[span.BlockEnd],
			All:   p.tokens[span.BlockStart : span.BlockEnd+1],
		},
		OpenBrace:  &p.tokens[span.BlockStart],
		CloseBrace: &p.tokens[span.BlockEnd],
	}
}

// buildExpression constructs an Expression from tokens
func (p *Parser) buildExpression(tokens []lexer.Token) ast.Expression {
	if len(tokens) == 0 {
		return nil
	}

	if len(tokens) == 1 {
		return p.buildSingleTokenExpression(tokens[0])
	}

	return p.buildComplexExpression(tokens)
}

// buildSingleTokenExpression constructs an Expression from a single token
func (p *Parser) buildSingleTokenExpression(token lexer.Token) ast.Expression {
	pos := ast.Position{
		Line:   token.Line,
		Column: token.Column,
	}

	tokenRange := ast.TokenRange{
		Start: token,
		End:   token,
		All:   []lexer.Token{token},
	}

	switch token.Type {
	case lexer.STRING:
		value := token.Value
		raw := token.Raw
		if raw == "" {
			raw = value
		}

		return &ast.StringLiteral{
			Value:        value,
			Raw:          raw,
			HasVariables: false, // Will be handled by decorators now
			Variables:    []ast.VariableRef{},
			Pos:          pos,
			Tokens:       tokenRange,
			StringToken:  token,
		}

	case lexer.NUMBER:
		return &ast.NumberLiteral{
			Value:  token.Value,
			Pos:    pos,
			Tokens: tokenRange,
			Token:  token,
		}

	case lexer.DURATION:
		return &ast.DurationLiteral{
			Value:  token.Value,
			Pos:    pos,
			Tokens: tokenRange,
			Token:  token,
		}

	case lexer.IDENTIFIER:
		if _, err := strconv.ParseInt(token.Value, 10, 64); err == nil {
			return &ast.NumberLiteral{
				Value:  token.Value,
				Pos:    pos,
				Tokens: tokenRange,
				Token:  token,
			}
		}

		if isDuration(token.Value) {
			return &ast.DurationLiteral{
				Value:  token.Value,
				Pos:    pos,
				Tokens: tokenRange,
				Token:  token,
			}
		}

		return &ast.Identifier{
			Name:   token.Value,
			Pos:    pos,
			Tokens: tokenRange,
			Token:  token,
		}

	case lexer.AT:
		// Check if this is a decorator
		if decorator, exists := p.decorators[p.findTokenIndex(token)]; exists {
			return decorator
		}
		return &ast.Identifier{
			Name:   token.Value,
			Pos:    pos,
			Tokens: tokenRange,
			Token:  token,
		}

	default:
		return &ast.Identifier{
			Name:   token.Value,
			Pos:    pos,
			Tokens: tokenRange,
			Token:  token,
		}
	}
}

// buildComplexExpression handles multi-token expressions
func (p *Parser) buildComplexExpression(tokens []lexer.Token) ast.Expression {
	if len(tokens) == 0 {
		return nil
	}

	// Check if this is a decorator pattern
	if len(tokens) >= 4 && tokens[0].Type == lexer.AT &&
		tokens[1].Type == lexer.IDENTIFIER &&
		tokens[2].Type == lexer.LPAREN {

		// This might be a decorator like @var(NAME) - check if we have it preprocessed
		if decorator, exists := p.decorators[p.findTokenIndex(tokens[0])]; exists {
			return decorator
		}
	}

	// Otherwise, combine tokens into a string expression
	firstToken := tokens[0]
	lastToken := tokens[len(tokens)-1]

	var combined strings.Builder
	for i, token := range tokens {
		if i > 0 {
			prevToken := tokens[i-1]
			if prevToken.Line == token.Line && prevToken.EndColumn < token.Column {
				spaces := token.Column - prevToken.EndColumn
				for j := 0; j < spaces; j++ {
					combined.WriteByte(' ')
				}
			}
		}
		combined.WriteString(token.Value)
	}

	value := combined.String()

	return &ast.StringLiteral{
		Value:        value,
		Raw:          value,
		HasVariables: false,
		Variables:    []ast.VariableRef{},
		Pos: ast.Position{
			Line:   firstToken.Line,
			Column: firstToken.Column,
		},
		Tokens: ast.TokenRange{
			Start: firstToken,
			End:   lastToken,
			All:   tokens,
		},
		StringToken: firstToken,
	}
}

// Helper function to check if a value is a duration
func isDuration(value string) bool {
	if len(value) < 2 {
		return false
	}

	suffixes := []string{"s", "m", "h", "ms", "us", "ns"}
	for _, suffix := range suffixes {
		if strings.HasSuffix(value, suffix) {
			prefix := value[:len(value)-len(suffix)]
			if _, err := strconv.ParseFloat(prefix, 64); err == nil {
				return true
			}
		}
	}

	return false
}

// Helper methods

func (p *Parser) findTokenIndex(target lexer.Token) int {
	for i, token := range p.tokens {
		if token.Line == target.Line &&
		   token.Column == target.Column &&
		   token.Value == target.Value {
			return i
		}
	}
	return -1
}

func (p *Parser) findDecoratorEnd(start int) int {
	for _, decorator := range p.structure.Decorators {
		if decorator.StartIndex == start {
			return decorator.EndIndex
		}
	}
	return start
}

func (p *Parser) findMatchingBrace(start int) int {
	if start >= len(p.tokens) || p.tokens[start].Type != lexer.LBRACE {
		return -1
	}

	depth := 1
	for i := start + 1; i < len(p.tokens); i++ {
		switch p.tokens[i].Type {
		case lexer.LBRACE:
			depth++
		case lexer.RBRACE:
			depth--
			if depth == 0 {
				return i
			}
		}
	}

	return -1
}

func (p *Parser) combineTokensToText(tokens []lexer.Token) string {
	if len(tokens) == 0 {
		return ""
	}

	if len(tokens) == 1 {
		return tokens[0].Value
	}

	var result strings.Builder
	for i, token := range tokens {
		if i > 0 {
			prevToken := tokens[i-1]
			if prevToken.Line == token.Line &&
			   prevToken.EndColumn < token.Column {
				spaces := token.Column - prevToken.EndColumn
				for j := 0; j < spaces; j++ {
					result.WriteByte(' ')
				}
			} else if prevToken.Line < token.Line {
				if token.Type != lexer.NEWLINE && prevToken.Type != lexer.NEWLINE {
					result.WriteByte(' ')
				}
			}
		}

		if token.Type != lexer.NEWLINE {
			result.WriteString(token.Value)
		}
	}

	return result.String()
}

// buildDecoratorNodes pre-builds Decorator nodes during preprocessing
func (p *Parser) buildDecoratorNodes() {
	for _, decoratorSpan := range p.structure.Decorators {
		decorator := p.buildDecorator(decoratorSpan)
		if decorator != nil {
			p.decorators[decoratorSpan.StartIndex] = decorator
		}
	}
}
