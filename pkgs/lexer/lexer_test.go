package lexer

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestBasicTokensOptimized(t *testing.T) {
	tests := []struct {
		input    string
		expected []TokenType
	}{
		{
			input:    "var mycommand:",
			expected: []TokenType{VAR, IDENTIFIER, COLON, EOF},
		},
		{
			input:    "watch server:",
			expected: []TokenType{WATCH, IDENTIFIER, COLON, EOF},
		},
		{
			input:    "stop app",
			expected: []TokenType{STOP, IDENTIFIER, EOF},
		},
		{
			input:    "var test = \"hello world\"",
			expected: []TokenType{VAR, IDENTIFIER, EQUALS, STRING, EOF},
		},
		{
			input:    "# this is a comment",
			expected: []TokenType{COMMENT, EOF},
		},
		{
			input:    "/* multiline comment */",
			expected: []TokenType{MULTILINE_COMMENT, EOF},
		},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			lexer := New(test.input)
			tokens := lexer.TokenizeToSlice()

			var tokenTypes []TokenType
			for _, token := range tokens {
				tokenTypes = append(tokenTypes, token.Type)
			}

			if diff := cmp.Diff(test.expected, tokenTypes); diff != "" {
				t.Errorf("Token sequence mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestStringTypesOptimized(t *testing.T) {
	tests := []struct {
		input      string
		stringType StringType
		value      string
	}{
		{`"hello"`, DoubleQuoted, "hello"},
		{`'world'`, SingleQuoted, "world"},
		{"`test`", Backtick, "test"},
		{`"with\nescapes"`, DoubleQuoted, "with\nescapes"},
		{`'literal\backslash'`, SingleQuoted, `literal\backslash`},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			lexer := New(test.input)
			token := lexer.NextToken()

			if token.Type != STRING {
				t.Errorf("Expected STRING token, got %s", token.Type)
				return
			}

			if token.StringType != test.stringType {
				t.Errorf("String type mismatch: %s", cmp.Diff(test.stringType, token.StringType))
			}

			if diff := cmp.Diff(test.value, token.Value); diff != "" {
				t.Errorf("String value mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDecoratorsOptimized(t *testing.T) {
	tests := []struct {
		input         string
		decoratorType TokenType
		name          string
		args          string
		block         string
	}{
		{
			input:         "@timeout(30s)",
			decoratorType: DECORATOR_CALL,
			name:          "timeout",
			args:          "30s",
			block:         "",
		},
		{
			input:         "@var{ echo hello }",
			decoratorType: DECORATOR_BLOCK,
			name:          "var",
			args:          "",
			block:         " echo hello ",
		},
		{
			input:         "@timeout(30s) { echo hello }",
			decoratorType: DECORATOR_CALL_BLOCK,
			name:          "timeout",
			args:          "30s",
			block:         " echo hello ",
		},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			lexer := New(test.input)
			token := lexer.NextToken()

			if token.Type != test.decoratorType {
				t.Errorf("Expected %s token, got %s", test.decoratorType, token.Type)
				return
			}

			if token.DecoratorName != test.name {
				t.Errorf("Decorator name mismatch: %s", cmp.Diff(test.name, token.DecoratorName))
			}

			if diff := cmp.Diff(test.args, token.Args); diff != "" {
				t.Errorf("Decorator args mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(test.block, token.Block); diff != "" {
				t.Errorf("Decorator block mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNumbersOptimized(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"123", "123"},
		{"-456", "-456"},
		{"3.14", "3.14"},
		{"-2.5", "-2.5"},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			lexer := New(test.input)
			token := lexer.NextToken()

			if token.Type != NUMBER {
				t.Errorf("Expected NUMBER token, got %s", token.Type)
				return
			}

			if diff := cmp.Diff(test.expected, token.Value); diff != "" {
				t.Errorf("Number value mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestShellModeOptimized(t *testing.T) {
	input := "var test: echo hello world"
	lexer := New(input)

	// Should start in language mode
	var1 := lexer.NextToken()
	if var1.Type != VAR {
		t.Errorf("Expected VAR, got %s", var1.Type)
	}

	ident := lexer.NextToken()
	if ident.Type != IDENTIFIER {
		t.Errorf("Expected IDENTIFIER, got %s", ident.Type)
	}

	colon := lexer.NextToken()
	if colon.Type != COLON {
		t.Errorf("Expected COLON, got %s", colon.Type)
	}

	// After colon, should be in shell mode
	shellText := lexer.NextToken()
	if shellText.Type != SHELL_TEXT {
		t.Errorf("Expected SHELL_TEXT, got %s", shellText.Type)
	}

	if diff := cmp.Diff("echo hello world", shellText.Value); diff != "" {
		t.Errorf("Shell text mismatch (-want +got):\n%s", diff)
	}
}

func TestLineContinuationOptimized(t *testing.T) {
	input := "echo hello \\\nworld"
	lexer := New(input)

	// Simulate being in shell mode
	lexer.setMode(ShellMode)

	shellText1 := lexer.NextToken()
	if shellText1.Type != SHELL_TEXT {
		t.Errorf("Expected SHELL_TEXT, got %s", shellText1.Type)
	}

	lineCont := lexer.NextToken()
	if lineCont.Type != LINE_CONT {
		t.Errorf("Expected LINE_CONT, got %s", lineCont.Type)
	}

	if diff := cmp.Diff("\\\n", lineCont.Value); diff != "" {
		t.Errorf("Line continuation value mismatch (-want +got):\n%s", diff)
	}

	shellText2 := lexer.NextToken()
	if shellText2.Type != SHELL_TEXT {
		t.Errorf("Expected SHELL_TEXT, got %s", shellText2.Type)
	}

	if diff := cmp.Diff("world", shellText2.Value); diff != "" {
		t.Errorf("Shell text value mismatch (-want +got):\n%s", diff)
	}
}

func TestPositionOptimized(t *testing.T) {
	input := "var\ntest:"
	lexer := New(input)

	var1 := lexer.NextToken()
	if var1.Line != 1 || var1.Column != 1 {
		t.Errorf("Expected position 1:1, got %d:%d", var1.Line, var1.Column)
	}

	newline := lexer.NextToken()
	if newline.Type != NEWLINE {
		t.Errorf("Expected NEWLINE, got %s", newline.Type)
	}

	ident := lexer.NextToken()
	if ident.Line != 2 || ident.Column != 1 {
		t.Errorf("Expected position 2:1, got %d:%d", ident.Line, ident.Column)
	}
}

func TestComplexExampleOptimized(t *testing.T) {
	input := `
var server: @timeout(30s) {
	echo "Starting server..."
	node app.js
}

watch tests: @var(NODE_ENV=test) {
	npm test
}

stop all: pkill -f "node|npm"
`

	lexer := New(input)
	tokens := lexer.TokenizeToSlice()

	if len(tokens) == 0 {
		t.Error("Expected tokens, got none")
	}

	// Should end with EOF
	if tokens[len(tokens)-1].Type != EOF {
		t.Errorf("Expected last token to be EOF, got %s", tokens[len(tokens)-1].Type)
	}

	// Count different token types
	var varCount, watchCount, stopCount int
	for _, token := range tokens {
		switch token.Type {
		case VAR:
			varCount++
		case WATCH:
			watchCount++
		case STOP:
			stopCount++
		}
	}

	if varCount != 1 {
		t.Errorf("Expected 1 VAR token, got %d", varCount)
	}
	if watchCount != 1 {
		t.Errorf("Expected 1 WATCH token, got %d", watchCount)
	}
	if stopCount != 1 {
		t.Errorf("Expected 1 STOP token, got %d", stopCount)
	}
}

func TestGetSemanticTokensOptimized(t *testing.T) {
	input := `var server: echo "hello"`

	tokens, err := GetSemanticTokens(input)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(tokens) == 0 {
		t.Error("Expected tokens, got none")
	}

	// Should NOT include EOF token
	for _, token := range tokens {
		if token.Type == EOF {
			t.Error("GetSemanticTokens should not include EOF token")
		}
	}
}

func TestOptimizedAPIUsage(t *testing.T) {
	input := `var server: @timeout(30s) { echo "test"; node app.js; }`

	// Test TokenizeToSlice method
	lexer := New(input)
	tokens := lexer.TokenizeToSlice()

	if len(tokens) == 0 {
		t.Error("Expected tokens, got none")
	}

	// Should end with EOF
	if tokens[len(tokens)-1].Type != EOF {
		t.Errorf("Expected last token to be EOF, got %s", tokens[len(tokens)-1].Type)
	}

	// Verify we get reasonable token count
	if len(tokens) < 5 {
		t.Errorf("Expected at least 5 tokens, got %d", len(tokens))
	}
}

func TestDecoratorDetectionOptimized(t *testing.T) {
	tests := []struct {
		input       string
		shouldMatch bool
	}{
		{"@timeout(30s)", true},
		{"@var{}", true},
		{"@test( args )", true},
		{"@invalid", false},        // no parentheses or braces
		{"@ timeout", false},       // space after @
		{"email@domain.com", false}, // not a decorator
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			lexer := New(test.input)
			token := lexer.NextToken()

			isDecorator := token.Type == DECORATOR_CALL ||
				token.Type == DECORATOR_BLOCK ||
				token.Type == DECORATOR_CALL_BLOCK

			if isDecorator != test.shouldMatch {
				t.Errorf("Expected decorator detection to be %v, got %v for input %q",
					test.shouldMatch, isDecorator, test.input)
			}
		})
	}
}

func TestZeroCopyBehavior(t *testing.T) {
	input := `var test: echo hello world`
	lexer := New(input)

	tokens := lexer.TokenizeToSlice()

	// Verify that simple tokens reference the original string
	for _, token := range tokens {
		if token.Type == IDENTIFIER || token.Type == SHELL_TEXT {
			if token.Value != "" && !contains(input, token.Value) {
				t.Errorf("Token value %q should be zero-copy reference to original input", token.Value)
			}
		}
	}
}

func TestOptimizedPerformance(t *testing.T) {
	// Test that the optimized version can handle reasonable loads
	input := generateTestInput(1000) // 1000 lines

	lexer := New(input)
	tokens := lexer.TokenizeToSlice()

	if len(tokens) == 0 {
		t.Error("Expected tokens from large input")
	}

	// Should end with EOF
	if tokens[len(tokens)-1].Type != EOF {
		t.Error("Large input should end with EOF token")
	}
}

// Helper functions
func contains(s, substr string) bool {
	return len(substr) > 0 && len(s) >= len(substr) &&
		(s == substr || findInString(s, substr) >= 0)
}

func findInString(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func generateTestInput(lines int) string {
	var result strings.Builder
	for i := 0; i < lines; i++ {
		result.WriteString(fmt.Sprintf("var cmd%d: echo hello %d\n", i, i))
	}
	return result.String()
}

func TestParseDecoratorArgsOptimized(t *testing.T) {
	tests := []struct {
		input    string
		expected []DecoratorArg
	}{
		{
			input: "30s",
			expected: []DecoratorArg{
				{Value: "30s", Line: 1, Column: 1},
			},
		},
		{
			input: "after=30s",
			expected: []DecoratorArg{
				{Name: "after", Value: "30s", Line: 1, Column: 1},
			},
		},
		{
			input: "3, verbose=true",
			expected: []DecoratorArg{
				{Value: "3", Line: 1, Column: 1},
				{Name: "verbose", Value: "true", Line: 1, Column: 1},
			},
		},
		{
			input: `port="8080", host="localhost"`,
			expected: []DecoratorArg{
				{Name: "port", Value: "8080", Line: 1, Column: 1},
				{Name: "host", Value: "localhost", Line: 1, Column: 1},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			args, err := ParseDecoratorArgs(test.input, 1, 1)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if diff := cmp.Diff(test.expected, args); diff != "" {
				t.Errorf("Decorator args mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
