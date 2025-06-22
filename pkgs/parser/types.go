package parser

import (
	"fmt"
	"strings"
)

// CommandElement represents any element that can appear in command text
// This supports a proper AST structure for nested decorators
type CommandElement interface {
	String() string
	IsDecorator() bool
}

// TextElement represents literal text in commands
type TextElement struct {
	Text       string
}

func (t *TextElement) String() string {
	return t.Text
}

func (t *TextElement) IsDecorator() bool {
	return false
}

// DecoratorElement represents a decorator like @var(SRC) or @sh(...)
type DecoratorElement struct {
	Name    string           // "var", "sh", "parallel", etc.
	Type    string           // "function", "simple", "block"
	Args    []CommandElement // For function decorators: contents of @name(...)
	Block   []BlockStatement // For block decorators: @name: { ... }
	Command []CommandElement // For simple decorators: @name: command
}

func (d *DecoratorElement) String() string {
	switch d.Type {
	case "function":
		var argStrs []string
		for _, arg := range d.Args {
			argStrs = append(argStrs, arg.String())
		}
		return fmt.Sprintf("@%s(%s)", d.Name, strings.Join(argStrs, ""))
	case "simple":
		var cmdStrs []string
		for _, cmd := range d.Command {
			cmdStrs = append(cmdStrs, cmd.String())
		}
		return fmt.Sprintf("@%s: %s", d.Name, strings.Join(cmdStrs, ""))
	case "block":
		// Block representation would be more complex
		return fmt.Sprintf("@%s: { ... }", d.Name)
	default:
		return fmt.Sprintf("@%s", d.Name)
	}
}

func (d *DecoratorElement) IsDecorator() bool {
	return true
}

// BlockStatement represents a statement within a block command
// Enhanced to support nested decorator structures
type BlockStatement struct {
	// New AST-based approach
	Elements []CommandElement // Command broken into elements (text + decorators)

	// Legacy fields for backward compatibility
	Command        string           // Flattened command text (for compatibility)
	IsDecorated    bool             // Whether this is a decorated command
	Decorator      string           // The decorator name
	DecoratorType  string           // "function", "simple", or "block"
	DecoratedBlock []BlockStatement // For block-type decorators
}

// Helper methods for BlockStatement (updated for new structure)
func (bs *BlockStatement) IsFunction() bool {
	return bs.IsDecorated && bs.DecoratorType == "function"
}

func (bs *BlockStatement) IsSimpleDecorator() bool {
	return bs.IsDecorated && bs.DecoratorType == "simple"
}

func (bs *BlockStatement) IsBlockDecorator() bool {
	return bs.IsDecorated && bs.DecoratorType == "block"
}

func (bs *BlockStatement) GetCommand() string {
	if bs.Command != "" {
		return bs.Command // Use legacy field if available
	}

	// Generate from elements
	var parts []string
	for _, elem := range bs.Elements {
		parts = append(parts, elem.String())
	}
	return strings.Join(parts, "")
}

func (bs *BlockStatement) GetDecorator() string {
	return bs.Decorator
}

func (bs *BlockStatement) GetNestedBlock() []BlockStatement {
	return bs.DecoratedBlock
}

// GetParsedElements returns the structured command elements
// This is the new API for accessing the parsed structure
func (bs *BlockStatement) GetParsedElements() []CommandElement {
	return bs.Elements
}

// HasNestedDecorators checks if this statement contains nested decorators
func (bs *BlockStatement) HasNestedDecorators() bool {
	for _, elem := range bs.Elements {
		if elem.IsDecorator() {
			return true
		}
	}
	return false
}

// GetDecorators returns all decorator elements in this statement
func (bs *BlockStatement) GetDecorators() []*DecoratorElement {
	var decorators []*DecoratorElement
	for _, elem := range bs.Elements {
		if decorator, ok := elem.(*DecoratorElement); ok {
			decorators = append(decorators, decorator)
		}
	}
	return decorators
}

// Definition represents a variable definition in the command file
type Definition struct {
	Name  string // The variable name
	Value string // The variable value
	Line  int    // The line number in the source file
}

// Command represents a command definition in the command file
// Enhanced to support the new AST structure
type Command struct {
	Name    string           // The command name
	Command string           // The command text for simple commands (legacy)
	Line    int              // The line number in the source file
	IsWatch bool             // Whether this is a watch command
	IsStop  bool             // Whether this is a stop command
	IsBlock bool             // Whether this is a block command
	Block   []BlockStatement // The statements for block commands

	// New structured representation
	Elements []CommandElement // For simple commands broken into elements
}

// GetParsedElements returns the structured command elements for simple commands
func (c *Command) GetParsedElements() []CommandElement {
	return c.Elements
}

// HasNestedDecorators checks if this command contains nested decorators
func (c *Command) HasNestedDecorators() bool {
	if c.IsBlock {
		for _, stmt := range c.Block {
			if stmt.HasNestedDecorators() {
				return true
			}
		}
		return false
	}

	for _, elem := range c.Elements {
		if elem.IsDecorator() {
			return true
		}
	}
	return false
}

// CommandFile represents the parsed command file
type CommandFile struct {
	Definitions []Definition // All variable definitions
	Commands    []Command    // All command definitions
	Lines       []string     // Original file lines for error reporting
}

// processEscapeSequences processes escape sequences in text
func processEscapeSequences(text string) ([]byte, error) {
	var result []byte
	i := 0

	for i < len(text) {
		if text[i] == '\\' && i+1 < len(text) {
			nextChar := text[i+1]
			switch nextChar {
			case '\\':
				result = append(result, '\\')
				i += 2
			case 'n':
				result = append(result, '\n')
				i += 2
			case 'r':
				result = append(result, '\r')
				i += 2
			case 't':
				result = append(result, '\t')
				i += 2
			case '$':
				result = append(result, '$')
				i += 2
			case '{':
				result = append(result, '{')
				i += 2
			case '}':
				result = append(result, '}')
				i += 2
			case '(':
				result = append(result, '(')
				i += 2
			case ')':
				result = append(result, ')')
				i += 2
			case '"':
				result = append(result, '"')
				i += 2
			case 'x':
				// Hex escape: \xXX
				if i+3 < len(text) && isHexDigit(text[i+2]) && isHexDigit(text[i+3]) {
					hex := text[i+2 : i+4]
					if val, err := parseHexByte(hex); err == nil {
						result = append(result, val)
						i += 4
					} else {
						// Not valid hex, preserve as-is
						result = append(result, '\\')
						i++
					}
				} else {
					// Not valid hex escape, preserve as-is
					result = append(result, '\\')
					i++
				}
			case 'u':
				// Unicode escape: \uXXXX
				if i+5 < len(text) &&
					isHexDigit(text[i+2]) && isHexDigit(text[i+3]) &&
					isHexDigit(text[i+4]) && isHexDigit(text[i+5]) {
					hex := text[i+2 : i+6]
					if val, err := parseHexRune(hex); err == nil {
						result = append(result, []byte(string(val))...)
						i += 6
					} else {
						// Not valid unicode, preserve as-is
						result = append(result, '\\')
						i++
					}
				} else {
					// Not valid unicode escape, preserve as-is
					result = append(result, '\\')
					i++
				}
			default:
				// Not a devcmd escape - preserve both backslash and next char
				result = append(result, '\\')
				i++
			}
		} else {
			result = append(result, text[i])
			i++
		}
	}

	return result, nil
}

// isHexDigit checks if a character is a valid hexadecimal digit
func isHexDigit(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

// parseHexByte parses a 2-character hex string into a byte
func parseHexByte(hex string) (byte, error) {
	if len(hex) != 2 {
		return 0, fmt.Errorf("invalid hex length: expected 2, got %d", len(hex))
	}

	var result byte
	for _, c := range []byte(hex) {
		result <<= 4
		if c >= '0' && c <= '9' {
			result |= c - '0'
		} else if c >= 'a' && c <= 'f' {
			result |= c - 'a' + 10
		} else if c >= 'A' && c <= 'F' {
			result |= c - 'A' + 10
		} else {
			return 0, fmt.Errorf("invalid hex character: %c", c)
		}
	}
	return result, nil
}

// parseHexRune parses a 4-character hex string into a rune
func parseHexRune(hex string) (rune, error) {
	if len(hex) != 4 {
		return 0, fmt.Errorf("invalid hex length: expected 4, got %d", len(hex))
	}

	var result rune
	for _, c := range []byte(hex) {
		result <<= 4
		if c >= '0' && c <= '9' {
			result |= rune(c - '0')
		} else if c >= 'a' && c <= 'f' {
			result |= rune(c - 'a' + 10)
		} else if c >= 'A' && c <= 'F' {
			result |= rune(c - 'A' + 10)
		} else {
			return 0, fmt.Errorf("invalid hex character: %c", c)
		}
	}
	return result, nil
}
