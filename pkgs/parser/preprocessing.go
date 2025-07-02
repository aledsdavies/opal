package parser

import (
	"fmt"

	"github.com/aledsdavies/devcmd/pkgs/ast"
	"github.com/aledsdavies/devcmd/pkgs/lexer"
)

// preprocessTokens performs Pass 1: fast structural analysis
func (p *Parser) preprocessTokens() {
	p.estimateCapacities()

	i := 0
	for i < len(p.tokens) {
		token := p.tokens[i]

		switch token.Type {
		case lexer.VAR:
			i = p.preprocessVariable(i)
		case lexer.WATCH, lexer.STOP:
			i = p.preprocessCommand(i)
		case lexer.IDENTIFIER:
			// Check if this is a command (not prefixed by var/watch/stop)
			if p.isCommandDeclaration(i) {
				i = p.preprocessCommand(i)
			} else {
				i++
			}
		case lexer.AT:
			i = p.preprocessAtSymbol(i)
		case lexer.LBRACE:
			i = p.preprocessBlock(i)
		default:
			i++
		}
	}

	// Post-process to validate structure
	p.validateStructure()
}

// estimateCapacities pre-allocates slices based on token count
func (p *Parser) estimateCapacities() {
	tokenCount := len(p.tokens)

	// Conservative estimates based on typical devcmd files
	p.structure.Variables = make([]VariableSpan, 0, tokenCount/20)
	p.structure.Commands = make([]CommandSpan, 0, tokenCount/15)
	p.structure.Decorators = make([]DecoratorSpan, 0, tokenCount/25)
	p.structure.BlockRanges = make([]BlockRange, 0, tokenCount/30)

	// Initialize maps
	p.decorators = make(map[int]*ast.Decorator, tokenCount/25)
}

// preprocessVariable handles variable declarations
func (p *Parser) preprocessVariable(start int) int {
	if start >= len(p.tokens) || p.tokens[start].Type != lexer.VAR {
		return start + 1
	}

	// Check for grouped variables: var ( ... )
	if start+1 < len(p.tokens) && p.tokens[start+1].Type == lexer.LPAREN {
		return p.preprocessGroupedVariables(start)
	}

	// Single variable: var NAME = VALUE
	return p.preprocessSingleVariable(start)
}

// preprocessSingleVariable handles: var NAME = VALUE
func (p *Parser) preprocessSingleVariable(start int) int {
	if start+1 >= len(p.tokens) {
		p.addError(p.tokens[start], "incomplete variable declaration",
			"variable declaration", "expected 'var NAME = VALUE'")
		return len(p.tokens)
	}

	nameToken := p.tokens[start+1]

	// Validate variable name
	if nameToken.Type != lexer.IDENTIFIER {
		p.addError(nameToken, "expected variable name after 'var'",
			"variable declaration", "use a valid identifier like 'SRC' or 'PORT'")
		return start + 2
	}

	// Validate name starts with letter
	if !isValidIdentifierName(nameToken.Value) {
		p.addError(nameToken, fmt.Sprintf("invalid variable name '%s'", nameToken.Value),
			"variable declaration", "variable names must start with a letter and contain only letters, numbers, hyphens, and underscores")
		return start + 2
	}

	if start+2 >= len(p.tokens) {
		p.addError(nameToken, "incomplete variable declaration",
			"variable declaration", "expected '=' after variable name")
		return len(p.tokens)
	}

	equalsToken := p.tokens[start+2]
	if equalsToken.Type != lexer.EQUALS {
		p.addError(equalsToken, "expected '=' after variable name",
			"variable declaration", "syntax is 'var NAME = VALUE'")
		return start + 3
	}

	// Find the value (everything until newline or EOF)
	valueStart := start + 3
	valueEnd := p.findVariableValueEnd(valueStart)

	if valueStart > valueEnd {
		p.addError(equalsToken, "missing variable value",
			"variable declaration", "provide a value after '='")
		return valueEnd + 1
	}

	p.structure.Variables = append(p.structure.Variables, VariableSpan{
		NameToken:  nameToken,
		ValueStart: valueStart,
		ValueEnd:   valueEnd,
		IsGrouped:  false,
	})

	return valueEnd + 1
}

// preprocessGroupedVariables handles: var ( NAME = VALUE; NAME = VALUE )
func (p *Parser) preprocessGroupedVariables(start int) int {
	if start+1 >= len(p.tokens) || p.tokens[start+1].Type != lexer.LPAREN {
		return start + 1
	}

	groupStart := start + 2 // after var (
	groupEnd := p.findMatchingParen(start + 1)

	if groupEnd == -1 {
		p.addError(p.tokens[start+1], "unclosed variable group",
			"variable declaration", "add closing ')' after variable definitions")
		return len(p.tokens)
	}

	// Parse individual variables within the group
	i := groupStart
	for i < groupEnd {
		if p.tokens[i].Type == lexer.IDENTIFIER {
			nameToken := p.tokens[i]

			// Validate variable name
			if !isValidIdentifierName(nameToken.Value) {
				p.addError(nameToken, fmt.Sprintf("invalid variable name '%s'", nameToken.Value),
					"variable group", "variable names must start with a letter")
			}

			// Expect: NAME = VALUE
			if i+2 < groupEnd && p.tokens[i+1].Type == lexer.EQUALS {
				valueStart := i + 2
				valueEnd := p.findGroupVariableEnd(valueStart, groupEnd)

				p.structure.Variables = append(p.structure.Variables, VariableSpan{
					NameToken:  nameToken,
					ValueStart: valueStart,
					ValueEnd:   valueEnd,
					IsGrouped:  true,
					GroupStart: start,
					GroupEnd:   groupEnd,
				})

				i = valueEnd + 1
			} else {
				p.addError(nameToken, "expected '=' after variable name",
					"variable group", "syntax is 'NAME = VALUE'")
				i++
			}
		} else {
			i++
		}
	}

	return groupEnd + 1
}

// preprocessCommand handles command declarations - unified for all command types
func (p *Parser) preprocessCommand(start int) int {
	var typeToken, nameToken, colonToken lexer.Token
	commandStart := start

	// Determine command type and extract tokens
	switch p.tokens[start].Type {
	case lexer.WATCH, lexer.STOP:
		if start+2 >= len(p.tokens) {
			commandType := "watch"
			if p.tokens[start].Type == lexer.STOP {
				commandType = "stop"
			}
			p.addError(p.tokens[start], "incomplete command declaration",
				"command declaration", fmt.Sprintf("expected '%s NAME:'", commandType))
			return len(p.tokens)
		}

		typeToken = p.tokens[start]
		nameToken = p.tokens[start+1]

		// Validate command name
		if nameToken.Type != lexer.IDENTIFIER {
			p.addError(nameToken, "expected command name",
				"command declaration", "use a valid identifier like 'server' or 'api'")
			return start + 2
		}

		if !isValidIdentifierName(nameToken.Value) {
			p.addError(nameToken, fmt.Sprintf("invalid command name '%s'", nameToken.Value),
				"command declaration", "command names must start with a letter and contain only letters, numbers, hyphens, and underscores")
			return start + 2
		}

		if start+2 >= len(p.tokens) {
			p.addError(nameToken, "incomplete command declaration",
				"command declaration", "expected ':' after command name")
			return len(p.tokens)
		}

		colonToken = p.tokens[start+2]
		commandStart = start + 3

	case lexer.IDENTIFIER:
		// Regular command: NAME:
		nameToken = p.tokens[start]

		// Validate command name
		if !isValidIdentifierName(nameToken.Value) {
			p.addError(nameToken, fmt.Sprintf("invalid command name '%s'", nameToken.Value),
				"command declaration", "command names must start with a letter and contain only letters, numbers, hyphens, and underscores")
			return start + 1
		}

		if start+1 >= len(p.tokens) {
			p.addError(p.tokens[start], "incomplete command declaration",
				"command declaration", "expected ':' after command name")
			return len(p.tokens)
		}

		// No type token for regular commands
		typeToken = lexer.Token{Type: lexer.ILLEGAL} // Use ILLEGAL to indicate no prefix
		colonToken = p.tokens[start+1]
		commandStart = start + 2

	default:
		p.addError(p.tokens[start], "unexpected token in command declaration",
			"command declaration", "expected command name or 'watch'/'stop' keyword")
		return start + 1
	}

	// Validate colon
	if colonToken.Type != lexer.COLON {
		p.addError(colonToken, "expected ':' after command name",
			"command declaration", "syntax is 'NAME: command' or 'watch NAME: command'")
		return start + 3
	}

	// Skip any whitespace/newlines after colon
	for commandStart < len(p.tokens) && (isWhitespace(p.tokens[commandStart]) || p.tokens[commandStart].Type == lexer.NEWLINE) {
		commandStart++
	}

	// Determine block type and body boundaries correctly
	var bodyStart, bodyEnd int
	var isBlock bool
	var decoratorIndices []int

	if commandStart < len(p.tokens) && p.tokens[commandStart].Type == lexer.LBRACE {
		// Explicit block: { statements }
		isBlock = true
		bodyStart = commandStart // Points to the opening brace
		bodyEnd = p.findMatchingBracePreprocessing(commandStart)
		if bodyEnd == -1 {
			p.addError(p.tokens[commandStart], "unclosed block",
				"block command", "add closing '}' after block statements")
			return len(p.tokens)
		}
	} else {
		// Check if we have decorators with blocks at the start
		currentPos := commandStart
		hasDecoratorWithBlock := false

		// Peek ahead to see if we have a decorator with a block
		if currentPos < len(p.tokens) && p.tokens[currentPos].Type == lexer.AT {
			// Look for decorator pattern: @name(...) { ... }
			tempPos := currentPos
			if tempDecEnd := p.peekDecoratorEnd(tempPos); tempDecEnd > tempPos {
				// Check if there's a block after the decorator
				afterDec := tempDecEnd + 1
				// Skip whitespace
				for afterDec < len(p.tokens) && (isWhitespace(p.tokens[afterDec]) || p.tokens[afterDec].Type == lexer.NEWLINE) {
					afterDec++
				}
				if afterDec < len(p.tokens) && p.tokens[afterDec].Type == lexer.LBRACE {
					hasDecoratorWithBlock = true
				}
			}
		}

		if hasDecoratorWithBlock {
			// Command with decorator that has a block
			// The entire thing becomes an implicit block
			isBlock = true
			bodyStart = commandStart
			// Find the end of the decorator and its block
			bodyEnd = p.findDecoratorWithBlockEnd(commandStart)
		} else {
			// Simple command or command with regular decorators
			// Parse any decorators first
			originalBodyStart := commandStart
			currentPos := commandStart

			// Collect decorators (only those without blocks)
			for currentPos < len(p.tokens) && p.tokens[currentPos].Type == lexer.AT {
				// Peek to see if this decorator will have a block
				tempDecEnd := p.peekDecoratorEnd(currentPos)
				afterDec := tempDecEnd + 1
				// Skip whitespace
				for afterDec < len(p.tokens) && (isWhitespace(p.tokens[afterDec]) || p.tokens[afterDec].Type == lexer.NEWLINE) {
					afterDec++
				}

				if afterDec < len(p.tokens) && p.tokens[afterDec].Type == lexer.LBRACE {
					// This decorator has a block - don't add it to decorators list
					// The whole thing becomes an implicit block
					isBlock = true
					bodyStart = currentPos
					bodyEnd = p.findDecoratorWithBlockEnd(currentPos)
					decoratorIndices = []int{} // Clear any decorators we collected
					break
				} else {
					// Regular decorator without block
					decoratorIndex := len(p.structure.Decorators)
					decoratorEnd := p.preprocessDecorator(currentPos)
					if decoratorEnd > currentPos {
						decoratorIndices = append(decoratorIndices, decoratorIndex)
						currentPos = decoratorEnd

						// Skip whitespace after decorator
						for currentPos < len(p.tokens) && (isWhitespace(p.tokens[currentPos]) || p.tokens[currentPos].Type == lexer.NEWLINE) {
							currentPos++
						}
					} else {
						break
					}
				}
			}

			if !isBlock {
				// Simple command body starts after decorators
				bodyStart = currentPos
				bodyEnd = p.findCommandBodyEnd(bodyStart, false)
			}
		}
	}

	p.structure.Commands = append(p.structure.Commands, CommandSpan{
		TypeToken:  typeToken,
		NameToken:  nameToken,
		ColonToken: colonToken,
		BodyStart:  bodyStart,
		BodyEnd:    bodyEnd,
		IsBlock:    isBlock,
		Decorators: decoratorIndices,
	})

	return bodyEnd + 1
}

// preprocessAtSymbol handles all @ symbols as decorators
func (p *Parser) preprocessAtSymbol(start int) int {
	if start+1 >= len(p.tokens) {
		return start + 1
	}

	nextToken := p.tokens[start+1]

	// All @identifier patterns are treated as decorators
	if nextToken.Type == lexer.IDENTIFIER && isValidIdentifierName(nextToken.Value) {
		if p.isDecoratorContext(start) {
			return p.preprocessDecorator(start)
		}
	}

	// Not a decorator context, treat as regular text
	return start + 1
}

// preprocessDecorator handles all decorator forms: @name, @name(), @name{}, @name(){}
func (p *Parser) preprocessDecorator(start int) int {
	if start+1 >= len(p.tokens) {
		return start + 1
	}

	atToken := p.tokens[start]
	nameToken := p.tokens[start+1]

	if nameToken.Type != lexer.IDENTIFIER {
		p.addError(nameToken, "expected decorator name after '@'",
			"decorator", "syntax is @decoratorname, @decoratorname(args), @decoratorname{}, or @decoratorname(args){}")
		return start + 2
	}

	if !isValidIdentifierName(nameToken.Value) {
		p.addError(nameToken, fmt.Sprintf("invalid identifier '%s' after '@'", nameToken.Value),
			"decorator", "decorator names must be valid identifiers")
		return start + 2
	}

	decoratorSpan := DecoratorSpan{
		AtToken:    atToken,
		NameToken:  nameToken,
		StartIndex: start,
		EndIndex:   start + 1,
		HasArgs:    false,
		HasBlock:   false,
	}

	currentPos := start + 2

	// Check for arguments: @decorator(args)
	if currentPos < len(p.tokens) && p.tokens[currentPos].Type == lexer.LPAREN {
		argsStart := currentPos
		argsEnd := p.findMatchingParen(argsStart)

		if argsEnd == -1 {
			p.addError(p.tokens[argsStart], "unclosed decorator arguments",
				"decorator", "add closing ')' after decorator arguments")
			return start + 3
		}

		decoratorSpan.HasArgs = true
		decoratorSpan.ArgsStart = argsStart
		decoratorSpan.ArgsEnd = argsEnd
		decoratorSpan.EndIndex = argsEnd

		// Parse decorator arguments
		decoratorSpan.Args = p.parseDecoratorArgs(argsStart+1, argsEnd)

		currentPos = argsEnd + 1
	}

	// Skip whitespace after args
	for currentPos < len(p.tokens) && (isWhitespace(p.tokens[currentPos]) || p.tokens[currentPos].Type == lexer.NEWLINE) {
		currentPos++
	}

	// Check for block: @decorator{} or @decorator(){}
	if currentPos < len(p.tokens) && p.tokens[currentPos].Type == lexer.LBRACE {
		blockStart := currentPos
		blockEnd := p.findMatchingBracePreprocessing(blockStart)

		if blockEnd == -1 {
			p.addError(p.tokens[blockStart], "unclosed decorator block",
				"decorator", "add closing '}' after decorator block")
			return currentPos + 1
		}

		decoratorSpan.HasBlock = true
		decoratorSpan.BlockStart = blockStart
		decoratorSpan.BlockEnd = blockEnd
		decoratorSpan.EndIndex = blockEnd

		currentPos = blockEnd + 1
	}

	p.structure.Decorators = append(p.structure.Decorators, decoratorSpan)
	return decoratorSpan.EndIndex + 1
}

// preprocessBlock handles block commands { ... }
func (p *Parser) preprocessBlock(start int) int {
	openBrace := p.tokens[start]
	closeIndex := p.findMatchingBracePreprocessing(start)

	if closeIndex == -1 {
		p.addError(openBrace, "unclosed block",
			"block command", "add closing '}' after block statements")
		return len(p.tokens)
	}

	closeBrace := p.tokens[closeIndex]

	// Parse statements within the block
	statements := p.parseBlockStatements(start+1, closeIndex)

	p.structure.BlockRanges = append(p.structure.BlockRanges, BlockRange{
		OpenBrace:  openBrace,
		CloseBrace: closeBrace,
		StartIndex: start,
		EndIndex:   closeIndex,
		Statements: statements,
	})

	return closeIndex + 1
}

// Helper methods for structure discovery

// isCommandDeclaration checks if we have a command declaration pattern
func (p *Parser) isCommandDeclaration(index int) bool {
	// A command declaration is an identifier followed by a colon
	// But NOT if it's preceded by var/watch/stop
	if index > 0 {
		prevToken := p.tokens[index-1]
		if prevToken.Type == lexer.VAR || prevToken.Type == lexer.WATCH || prevToken.Type == lexer.STOP {
			return false
		}
	}

	if index+1 >= len(p.tokens) {
		return false
	}

	return p.tokens[index].Type == lexer.IDENTIFIER &&
		p.tokens[index+1].Type == lexer.COLON
}

// hasDecoratorBlocks checks if any decorators have their own blocks
func (p *Parser) hasDecoratorBlocks(decoratorIndices []int) bool {
	for _, idx := range decoratorIndices {
		if idx < len(p.structure.Decorators) {
			if p.structure.Decorators[idx].HasBlock {
				return true
			}
		}
	}
	return false
}

// hasImplicitBlockDecorator checks for decorators that create implicit blocks
func (p *Parser) hasImplicitBlockDecorator(decoratorIndices []int) bool {
	for _, idx := range decoratorIndices {
		if idx < len(p.structure.Decorators) {
			dec := p.structure.Decorators[idx]
			// Only @sh clearly contains commands at the structural level
			if !dec.HasBlock && dec.NameToken.Value == "sh" {
				return true
			}
		}
	}
	return false
}

func (p *Parser) findMatchingParen(start int) int {
	if start >= len(p.tokens) || p.tokens[start].Type != lexer.LPAREN {
		return -1
	}

	depth := 1
	for i := start + 1; i < len(p.tokens); i++ {
		switch p.tokens[i].Type {
		case lexer.LPAREN:
			depth++
		case lexer.RPAREN:
			depth--
			if depth == 0 {
				return i
			}
		}
	}

	return -1
}

func (p *Parser) findMatchingBracePreprocessing(start int) int {
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

// findVariableValueEnd finds the end of a variable value
func (p *Parser) findVariableValueEnd(start int) int {
	// Find end of variable value (until newline, semicolon identifier, or EOF)
	for i := start; i < len(p.tokens); i++ {
		if p.tokens[i].Type == lexer.NEWLINE ||
			p.tokens[i].Type == lexer.EOF ||
			(p.tokens[i].Type == lexer.IDENTIFIER && p.tokens[i].Value == ";") {
			// Skip the semicolon if found
			if p.tokens[i].Type == lexer.IDENTIFIER && p.tokens[i].Value == ";" {
				// Don't include the semicolon in the value
				return i - 1
			}
			return i - 1
		}
	}
	return len(p.tokens) - 1
}

func (p *Parser) findGroupVariableEnd(start, groupEnd int) int {
	// Find end of variable value within group (until newline or group end)
	for i := start; i < groupEnd; i++ {
		if p.tokens[i].Type == lexer.NEWLINE {
			return i - 1
		}
	}
	return groupEnd - 1
}

// findCommandBodyEnd finds the end of a command body
func (p *Parser) findCommandBodyEnd(start int, isBlock bool) int {
	if isBlock {
		// For decorators like @sh that create implicit blocks,
		// the body end is right before the block start
		if start >= len(p.tokens) {
			return start - 1
		}

		// For block commands, find the matching closing brace
		if p.tokens[start].Type == lexer.LBRACE {
			braceEnd := p.findMatchingBracePreprocessing(start)
			if braceEnd != -1 {
				return braceEnd
			}
		}

		// For implicit blocks (e.g., @sh decorator), body ends at decorator
		return start - 1
	}

	// For simple commands, find end of line or end of file
	for i := start; i < len(p.tokens); i++ {
		if p.tokens[i].Type == lexer.NEWLINE || p.tokens[i].Type == lexer.EOF {
			// Return the last non-whitespace token before newline
			for j := i - 1; j >= start; j-- {
				if !isWhitespace(p.tokens[j]) && p.tokens[j].Type != lexer.NEWLINE {
					return j
				}
			}
			return start
		}
	}

	return len(p.tokens) - 1
}

func (p *Parser) parseDecoratorArgs(start, end int) []DecoratorArgSpan {
	args := []DecoratorArgSpan{}

	i := start
	for i < end {
		// Skip whitespace and commas
		for i < end && (p.tokens[i].Type == lexer.COMMA || isWhitespace(p.tokens[i])) {
			i++
		}

		if i >= end {
			break
		}

		// Check for named argument (NAME = VALUE)
		isNamed := false
		name := ""

		if i+2 < end &&
			p.tokens[i].Type == lexer.IDENTIFIER &&
			p.tokens[i+1].Type == lexer.EQUALS {
			isNamed = true
			name = p.tokens[i].Value
			i += 2 // Skip name and =
		}

		// Find end of argument value
		argValueStart := i
		argEnd := p.findDecoratorArgEnd(i, end)

		if argEnd >= argValueStart {
			args = append(args, DecoratorArgSpan{
				Name:       name,
				ValueStart: argValueStart,
				ValueEnd:   argEnd,
				IsNamed:    isNamed,
			})
		}

		i = argEnd + 1
	}

	return args
}

func (p *Parser) findDecoratorArgEnd(start, maxEnd int) int {
	depth := 0
	inQuotes := false
	quoteChar := byte(0)

	for i := start; i < maxEnd; i++ {
		token := p.tokens[i]

		// Handle quoted strings
		if token.Type == lexer.STRING && len(token.Value) > 0 {
			if !inQuotes && (token.Value[0] == '"' || token.Value[0] == '\'') {
				inQuotes = true
				quoteChar = token.Value[0]
			} else if inQuotes && len(token.Value) > 0 && token.Value[len(token.Value)-1] == quoteChar {
				inQuotes = false
			}
		}

		if !inQuotes {
			switch token.Type {
			case lexer.LPAREN:
				depth++
			case lexer.RPAREN:
				if depth == 0 {
					return i - 1
				}
				depth--
			case lexer.COMMA:
				if depth == 0 {
					return i - 1
				}
			}
		}
	}
	return maxEnd - 1
}

func (p *Parser) parseBlockStatements(start, end int) []StatementSpan {
	statements := []StatementSpan{}

	i := start
	for i < end {
		// Skip whitespace and newlines
		for i < end && (isWhitespace(p.tokens[i]) || p.tokens[i].Type == lexer.NEWLINE) {
			i++
		}

		if i >= end {
			break
		}

		stmtStart := i
		hasDecorator := false
		decoratorIndex := -1

		// Check for decorator at start of statement
		if p.tokens[i].Type == lexer.AT {
			hasDecorator = true
			decoratorIndex = len(p.structure.Decorators)
			decoratorEnd := p.preprocessDecorator(i)
			i = decoratorEnd
		}

		// Find end of statement (semicolon or newline)
		stmtEnd := p.findStatementEnd(i, end)

		statements = append(statements, StatementSpan{
			Start:          stmtStart,
			End:            stmtEnd,
			HasDecorator:   hasDecorator,
			DecoratorIndex: decoratorIndex,
		})

		i = stmtEnd + 1
	}

	return statements
}

func (p *Parser) findStatementEnd(start, maxEnd int) int {
	// Find end of statement (newline, semicolon, or end of block)
	for i := start; i < maxEnd; i++ {
		if p.tokens[i].Type == lexer.NEWLINE {
			// Backtrack to find last non-whitespace token
			for j := i - 1; j >= start; j-- {
				if !isWhitespace(p.tokens[j]) {
					return j
				}
			}
			return start
		}
		// Also check for semicolon as statement separator
		if p.tokens[i].Type == lexer.IDENTIFIER && p.tokens[i].Value == ";" {
			return i - 1
		}
	}
	return maxEnd - 1
}

func (p *Parser) validateStructure() {
	// Check for duplicate declarations
	p.checkDuplicateDeclarations()
}

func (p *Parser) checkDuplicateDeclarations() {
	// Check duplicate variables
	varNames := make(map[string]lexer.Token)
	for _, varSpan := range p.structure.Variables {
		name := varSpan.NameToken.Value
		if existing, exists := varNames[name]; exists {
			p.addDuplicateError(varSpan.NameToken, existing, "variable", name)
		} else {
			varNames[name] = varSpan.NameToken
		}
	}

	// Check duplicate commands - but allow different types (watch/stop/regular) with same name
	cmdNames := make(map[string]map[string]lexer.Token) // [name][type]token
	for _, cmdSpan := range p.structure.Commands {
		name := cmdSpan.NameToken.Value

		// Determine command type
		cmdType := "command"
		if cmdSpan.TypeToken.Type == lexer.WATCH {
			cmdType = "watch"
		} else if cmdSpan.TypeToken.Type == lexer.STOP {
			cmdType = "stop"
		}

		if cmdNames[name] == nil {
			cmdNames[name] = make(map[string]lexer.Token)
		}

		if existing, exists := cmdNames[name][cmdType]; exists {
			p.addDuplicateError(cmdSpan.NameToken, existing, cmdType+" command", name)
		} else {
			cmdNames[name][cmdType] = cmdSpan.NameToken
		}
	}
}

func (p *Parser) addDuplicateError(current, existing lexer.Token, itemType, name string) {
	p.errors = append(p.errors, ParseError{
		Type:    DuplicateError,
		Token:   current,
		Message: fmt.Sprintf("duplicate %s '%s'", itemType, name),
		Context: fmt.Sprintf("%s declaration", itemType),
		Hint:    fmt.Sprintf("previous declaration at line %d:%d", existing.Line, existing.Column),
		Related: []lexer.Token{existing},
	})
}

// Helper function to check if token represents whitespace
func isWhitespace(token lexer.Token) bool {
	return token.Value == " " || token.Value == "\t" || token.Value == "\r"
}

// isDecoratorContext checks if @ is in a position where it could be a decorator
func (p *Parser) isDecoratorContext(atIndex int) bool {
	// Decorator must be at start of command body or after whitespace/newline
	if atIndex == 0 {
		return true
	}

	prevToken := p.tokens[atIndex-1]
	return prevToken.Type == lexer.NEWLINE ||
		prevToken.Type == lexer.COLON ||
		prevToken.Type == lexer.LBRACE ||
		isWhitespace(prevToken)
}

// isValidIdentifierName checks if a name is a valid identifier
func isValidIdentifierName(name string) bool {
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

// Add these helper methods at the end of the file

// peekDecoratorEnd looks ahead to find the end of a decorator without modifying state
func (p *Parser) peekDecoratorEnd(start int) int {
	if start >= len(p.tokens) || p.tokens[start].Type != lexer.AT {
		return start
	}

	i := start + 1

	// Skip decorator name
	if i < len(p.tokens) && p.tokens[i].Type == lexer.IDENTIFIER {
		i++
	} else {
		return start
	}

	// Check for arguments
	if i < len(p.tokens) && p.tokens[i].Type == lexer.LPAREN {
		depth := 1
		i++
		for i < len(p.tokens) && depth > 0 {
			if p.tokens[i].Type == lexer.LPAREN {
				depth++
			} else if p.tokens[i].Type == lexer.RPAREN {
				depth--
			}
			i++
		}
	}

	return i - 1
}

// findDecoratorWithBlockEnd finds the end of a decorator with its block
func (p *Parser) findDecoratorWithBlockEnd(start int) int {
	if start >= len(p.tokens) || p.tokens[start].Type != lexer.AT {
		return start
	}

	// Process the decorator
	decoratorEnd := p.preprocessDecorator(start)

	// The decorator preprocessing should have found the block
	if len(p.structure.Decorators) > 0 {
		lastDecorator := p.structure.Decorators[len(p.structure.Decorators)-1]
		if lastDecorator.HasBlock {
			return lastDecorator.BlockEnd
		}
	}

	return decoratorEnd
}
