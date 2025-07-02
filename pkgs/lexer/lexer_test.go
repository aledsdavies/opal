package lexer

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestBasicTokens(t *testing.T) {
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

func TestAtToken(t *testing.T) {
	tests := []struct {
		input    string
		expected []TokenType
	}{
		{
			input:    "@timeout",
			expected: []TokenType{AT, IDENTIFIER, EOF},
		},
		{
			input:    "@timeout(30s)",
			expected: []TokenType{AT, IDENTIFIER, LPAREN, DURATION, RPAREN, EOF},
		},
		{
			input:    "@timeout(30s) { echo hello }",
			expected: []TokenType{AT, IDENTIFIER, LPAREN, DURATION, RPAREN, LBRACE, IDENTIFIER, RBRACE, EOF},
		},
		{
			input:    "@retry(attempts=3, delay=1.5s)",
			expected: []TokenType{AT, IDENTIFIER, LPAREN, IDENTIFIER, EQUALS, NUMBER, COMMA, IDENTIFIER, EQUALS, DURATION, RPAREN, EOF},
		},
		{
			input:    "@watch-files(debounce=500ms)",
			expected: []TokenType{AT, IDENTIFIER, LPAREN, IDENTIFIER, EQUALS, DURATION, RPAREN, EOF},
		},
		{
			input:    "@var{ echo hello }",
			expected: []TokenType{AT, IDENTIFIER, LBRACE, IDENTIFIER, RBRACE, EOF},
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

func TestDecoratorSemanticTypes(t *testing.T) {
	tests := []struct {
		input            string
		identifierValues []string
		semanticTypes    []SemanticTokenType
	}{
		{
			input:            "@timeout(30s)",
			identifierValues: []string{"timeout"},
			semanticTypes:    []SemanticTokenType{SemDecorator},
		},
		{
			input:            "var server",
			identifierValues: []string{"server"},
			semanticTypes:    []SemanticTokenType{SemCommand},
		},
		{
			input:            "@var(PORT=8080) server",
			identifierValues: []string{"var", "PORT", "server"},
			semanticTypes:    []SemanticTokenType{SemDecorator, SemParameter, SemCommand},
		},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			lexer := New(test.input)
			tokens := lexer.TokenizeToSlice()

			var identifierTokens []Token
			for _, token := range tokens {
				if token.Type == IDENTIFIER {
					identifierTokens = append(identifierTokens, token)
				}
			}

			if len(identifierTokens) != len(test.identifierValues) {
				t.Errorf("Expected %d identifiers, got %d", len(test.identifierValues), len(identifierTokens))

				// Debug: show all tokens
				t.Logf("All tokens:")
				for i, token := range tokens {
					t.Logf("  %d: %s %q (semantic: %v)", i, token.Type, token.Value, token.Semantic)
				}
				return
			}

			for i, token := range identifierTokens {
				if token.Value != test.identifierValues[i] {
					t.Errorf("Identifier %d: expected %q, got %q", i, test.identifierValues[i], token.Value)
				}
				if token.Semantic != test.semanticTypes[i] {
					t.Errorf("Identifier %d semantic: expected %v (%d), got %v (%d)", i, test.semanticTypes[i], test.semanticTypes[i], token.Semantic, token.Semantic)
				}
			}
		})
	}
}

func TestStringTypes(t *testing.T) {
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
				t.Errorf("String type mismatch: expected %v, got %v", test.stringType, token.StringType)
			}

			if diff := cmp.Diff(test.value, token.Value); diff != "" {
				t.Errorf("String value mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDurations(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"30s", "30s"},
		{"5m", "5m"},
		{"1h", "1h"},
		{"500ms", "500ms"},
		{"2.5s", "2.5s"},
		{"1.5h", "1.5h"},
		{"100ns", "100ns"},
		{"250us", "250us"}, // ASCII microsecond notation
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			lexer := New(test.input)
			token := lexer.NextToken()

			if token.Type != DURATION {
				t.Errorf("Expected DURATION token, got %s", token.Type)
				return
			}

			if diff := cmp.Diff(test.expected, token.Value); diff != "" {
				t.Errorf("Duration value mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNumberVsDuration(t *testing.T) {
	tests := []struct {
		input        string
		expectedType TokenType
		expectedValue string
	}{
		{"123", NUMBER, "123"},
		{"123s", DURATION, "123s"},
		{"3.14", NUMBER, "3.14"},
		{"3.14s", DURATION, "3.14s"},
		{"-42", NUMBER, "-42"},
		{"42ms", DURATION, "42ms"},
		{"8080", NUMBER, "8080"}, // Port number stays as NUMBER
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			lexer := New(test.input)
			token := lexer.NextToken()

			if token.Type != test.expectedType {
				t.Errorf("Expected %s token, got %s", test.expectedType, token.Type)
				return
			}

			if diff := cmp.Diff(test.expectedValue, token.Value); diff != "" {
				t.Errorf("Value mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNumbers(t *testing.T) {
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

func TestCommandMode(t *testing.T) {
	input := "build: echo hello world"
	lexer := New(input)

	// Should start in language mode
	ident := lexer.NextToken()
	if ident.Type != IDENTIFIER {
		t.Errorf("Expected IDENTIFIER, got %s", ident.Type)
	}

	colon := lexer.NextToken()
	if colon.Type != COLON {
		t.Errorf("Expected COLON, got %s", colon.Type)
	}

	// After colon, should switch to command mode for simple commands
	shellText := lexer.NextToken()
	if shellText.Type != IDENTIFIER {
		t.Errorf("Expected IDENTIFIER, got %s", shellText.Type)
	}

	if diff := cmp.Diff("echo hello world", shellText.Value); diff != "" {
		t.Errorf("Shell text mismatch (-want +got):\n%s", diff)
	}
}

func TestSemicolonInShellCommand(t *testing.T) {
	// Test that semicolons are part of shell commands, not separators
	input := "build: echo hello; echo world"
	lexer := New(input)

	ident := lexer.NextToken()
	if ident.Type != IDENTIFIER || ident.Value != "build" {
		t.Errorf("Expected IDENTIFIER 'build', got %s %q", ident.Type, ident.Value)
	}

	colon := lexer.NextToken()
	if colon.Type != COLON {
		t.Errorf("Expected COLON, got %s", colon.Type)
	}

	// The entire shell command should be one token including the semicolon
	shellText := lexer.NextToken()
	if shellText.Type != IDENTIFIER {
		t.Errorf("Expected IDENTIFIER, got %s", shellText.Type)
	}

	expectedShellText := "echo hello; echo world"
	if diff := cmp.Diff(expectedShellText, shellText.Value); diff != "" {
		t.Errorf("Shell text mismatch (-want +got):\n%s", diff)
	}
}

func TestBlockWithSemicolons(t *testing.T) {
	// Test that semicolons inside blocks are part of shell commands
	input := "deploy: { cd src; make clean; make install }"
	lexer := New(input)

	// deploy
	ident := lexer.NextToken()
	if ident.Type != IDENTIFIER || ident.Value != "deploy" {
		t.Errorf("Expected IDENTIFIER 'deploy', got %s %q", ident.Type, ident.Value)
	}

	// :
	colon := lexer.NextToken()
	if colon.Type != COLON {
		t.Errorf("Expected COLON, got %s", colon.Type)
	}

	// {
	lbrace := lexer.NextToken()
	if lbrace.Type != LBRACE {
		t.Errorf("Expected LBRACE, got %s", lbrace.Type)
	}

	// The shell command with semicolons should be a single token
	shellCmd := lexer.NextToken()
	if shellCmd.Type != IDENTIFIER {
		t.Errorf("Expected IDENTIFIER, got %s", shellCmd.Type)
	}

	expectedCmd := "cd src; make clean; make install"
	if diff := cmp.Diff(expectedCmd, shellCmd.Value); diff != "" {
		t.Errorf("Shell command mismatch (-want +got):\n%s", diff)

		// Debug: show all remaining tokens
		t.Logf("Remaining tokens:")
		for {
			tok := lexer.NextToken()
			t.Logf("  %s %q", tok.Type, tok.Value)
			if tok.Type == EOF {
				break
			}
		}
	}

	// }
	rbrace := lexer.NextToken()
	if rbrace.Type != RBRACE {
		t.Errorf("Expected RBRACE, got %s", rbrace.Type)
	}
}

func TestLineContinuation(t *testing.T) {
	input := "echo hello \\\nworld"
	lexer := New(input)

	// Simulate being in command mode
	lexer.setMode(CommandMode)

	shellText1 := lexer.NextToken()
	if shellText1.Type != IDENTIFIER {
		t.Errorf("Expected IDENTIFIER, got %s", shellText1.Type)
	}

	lineCont := lexer.NextToken()
	if lineCont.Type != LINE_CONT {
		t.Errorf("Expected LINE_CONT, got %s", lineCont.Type)
	}

	if diff := cmp.Diff("\\\n", lineCont.Value); diff != "" {
		t.Errorf("Line continuation value mismatch (-want +got):\n%s", diff)
	}

	shellText2 := lexer.NextToken()
	if shellText2.Type != IDENTIFIER {
		t.Errorf("Expected IDENTIFIER, got %s", shellText2.Type)
	}

	if diff := cmp.Diff("world", shellText2.Value); diff != "" {
		t.Errorf("Shell text value mismatch (-want +got):\n%s", diff)
	}
}

func TestPosition(t *testing.T) {
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

func TestComplexExample(t *testing.T) {
	input := `
var PORT = 8080

server: @timeout(30s) {
	echo "Starting server..."
	node app.js
}

watch tests: npm test

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
	var varCount, watchCount, stopCount, atCount int
	for _, token := range tokens {
		switch token.Type {
		case VAR:
			varCount++
		case WATCH:
			watchCount++
		case STOP:
			stopCount++
		case AT:
			atCount++
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
	if atCount != 1 {
		t.Errorf("Expected 1 AT token, got %d", atCount)
	}
}

func TestNestedDecorators(t *testing.T) {
	input := "@timeout(30s) { @parallel { npm run api } }"
	lexer := New(input)
	tokens := lexer.TokenizeToSlice()

	// The new lexer properly handles nested decorators, so we expect:
	// @timeout(30s) { @parallel { npm run api } }
	// AT IDENTIFIER LPAREN DURATION RPAREN LBRACE AT IDENTIFIER LBRACE IDENTIFIER RBRACE RBRACE EOF
	expected := []TokenType{
		AT, IDENTIFIER, // @timeout
		LPAREN, DURATION, RPAREN, // (30s)
		LBRACE, // {
		AT, IDENTIFIER, // @parallel - now properly tokenized as decorator
		LBRACE, // {
		IDENTIFIER, // npm run api - shell text as single identifier
		RBRACE, // }
		RBRACE, // }
		EOF,
	}

	var tokenTypes []TokenType
	for _, token := range tokens {
		tokenTypes = append(tokenTypes, token.Type)
	}

	if diff := cmp.Diff(expected, tokenTypes); diff != "" {
		t.Errorf("Nested decorator token sequence mismatch (-want +got):\n%s", diff)

		// Debug: show actual tokens
		t.Logf("Actual tokens:")
		for i, token := range tokens {
			t.Logf("  %d: %s %q", i, token.Type, token.Value)
		}
	}

	// Check that decorator names have correct semantic types
	decoratorCount := 0
	for _, token := range tokens {
		if token.Type == IDENTIFIER && token.Semantic == SemDecorator {
			if token.Value == "timeout" || token.Value == "parallel" {
				decoratorCount++
			}
		}
	}

	if decoratorCount != 2 { // Now expecting both timeout and parallel
		t.Errorf("Expected 2 decorator identifiers (timeout, parallel), found %d", decoratorCount)
	}
}

func TestModeTransitions(t *testing.T) {
	// Debug: Simple command case
	t.Run("debug simple command", func(t *testing.T) {
		input := "build: echo hello"
		lexer := New(input)

		// Step 0: build
		token := lexer.NextToken()
		t.Logf("Step 0: token=%s, mode=%v (LanguageMode=0, CommandMode=1)", token.Type, lexer.mode)
		if token.Type != IDENTIFIER || lexer.mode != LanguageMode {
			t.Errorf("Step 0: expected IDENTIFIER in LanguageMode, got %s in mode %v", token.Type, lexer.mode)
		}

		// Step 1: :
		token = lexer.NextToken()
		t.Logf("Step 1: token=%s, mode=%v (LanguageMode=0, CommandMode=1)", token.Type, lexer.mode)
		if token.Type != COLON {
			t.Errorf("Step 1: expected COLON, got %s", token.Type)
		}
		// For simple commands, COLON should switch to CommandMode
		if lexer.mode != CommandMode {
			t.Errorf("Step 1: expected CommandMode (1) after COLON, got mode %v", lexer.mode)
		}

		// Step 2: echo hello
		token = lexer.NextToken()
		t.Logf("Step 2: token=%s, mode=%v, value=%q", token.Type, lexer.mode, token.Value)
		if token.Type != IDENTIFIER || lexer.mode != CommandMode {
			t.Errorf("Step 2: expected IDENTIFIER in CommandMode, got %s in mode %v", token.Type, lexer.mode)
		}

		// Step 3: EOF
		token = lexer.NextToken()
		t.Logf("Step 3: token=%s, mode=%v", token.Type, lexer.mode)
		if token.Type != EOF {
			t.Errorf("Step 3: expected EOF, got %s", token.Type)
		}
		// EOF should switch back to LanguageMode
		if lexer.mode != LanguageMode {
			t.Errorf("Step 3: expected LanguageMode (0) after EOF, got mode %v", lexer.mode)
		}
	})

	tests := []struct {
		name     string
		input    string
		expected []struct {
			token TokenType
			mode  LexerMode
		}
	}{
		{
			name:  "simple command transitions",
			input: "build: echo hello",
			expected: []struct {
				token TokenType
				mode  LexerMode
			}{
				{IDENTIFIER, LanguageMode}, // build (starts in LanguageMode)
				{COLON, CommandMode},       // : (switches to CommandMode because shouldSwitchToCommandMode() = true)
				{IDENTIFIER, CommandMode},  // echo hello (stays in CommandMode)
				{EOF, LanguageMode},        // EOF (switches back to LanguageMode on newline/EOF)
			},
		},
		{
			name:  "explicit block transitions",
			input: "build: { echo hello }",
			expected: []struct {
				token TokenType
				mode  LexerMode
			}{
				{IDENTIFIER, LanguageMode}, // build (starts in LanguageMode)
				{COLON, LanguageMode},      // : (stays in LanguageMode because shouldSwitchToCommandMode() = false due to {)
				{LBRACE, CommandMode},      // { (switches to CommandMode)
				{IDENTIFIER, CommandMode},  // echo hello (stays in CommandMode)
				{RBRACE, LanguageMode},     // } (switches back to LanguageMode)
				{EOF, LanguageMode},        // EOF (stays in LanguageMode)
			},
		},
		{
			name:  "decorator in command",
			input: "build: { @timeout(30s) { echo hello } }",
			expected: []struct {
				token TokenType
				mode  LexerMode
			}{
				{IDENTIFIER, LanguageMode}, // build (starts in LanguageMode)
				{COLON, LanguageMode},      // : (stays in LanguageMode because { follows)
				{LBRACE, CommandMode},      // { (switches to CommandMode)
				{AT, LanguageMode},         // @ (switches to LanguageMode for decorator parsing)
				{IDENTIFIER, LanguageMode}, // timeout (stays in LanguageMode)
				{LPAREN, LanguageMode},     // ( (stays in LanguageMode)
				{DURATION, LanguageMode},   // 30s (stays in LanguageMode)
				{RPAREN, LanguageMode},     // ) (stays in LanguageMode)
				{LBRACE, CommandMode},      // { (switches to CommandMode)
				{IDENTIFIER, CommandMode},  // echo hello (stays in CommandMode)
				{RBRACE, LanguageMode},     // } (switches back to LanguageMode)
				{RBRACE, LanguageMode},     // } (stays in LanguageMode)
				{EOF, LanguageMode},        // EOF (stays in LanguageMode)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			lexer := New(test.input)

			for i, expected := range test.expected {
				token := lexer.NextToken()
				if token.Type != expected.token {
					t.Errorf("Step %d: expected token %s, got %s", i, expected.token, token.Type)
				}

				// Check mode after token is processed
				if lexer.mode != expected.mode {
					t.Errorf("Step %d: expected mode %v, got %v (after token %s)",
						i, expected.mode, lexer.mode, token.Type)
				}
			}
		})
	}
}

func TestSyntaxSugarDetection(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		shouldSwitchToCommandMode bool
	}{
		{
			name:           "simple command gets sugar",
			input:          "build: echo hello",
			shouldSwitchToCommandMode: true,
		},
		{
			name:           "explicit block no sugar",
			input:          "build: { echo hello }",
			shouldSwitchToCommandMode: false,
		},
		{
			name:           "decorator no sugar",
			input:          "build: @timeout(30s) { echo hello }",
			shouldSwitchToCommandMode: false,
		},
		{
			name:           "empty command no sugar",
			input:          "build:",
			shouldSwitchToCommandMode: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			lexer := New(test.input)

			// Skip to colon
			lexer.NextToken() // identifier
			lexer.NextToken() // colon

			// After colon, check if we switched to CommandMode
			nextToken := lexer.NextToken()

			if test.shouldSwitchToCommandMode {
				// Should be in CommandMode, so next token should be shell text
				if nextToken.Type != IDENTIFIER || nextToken.Semantic != SemCommand {
					t.Errorf("Expected shell command token, got %s with semantic %v",
						nextToken.Type, nextToken.Semantic)
				}
			} else {
				// Should not switch to CommandMode immediately
				// Next token should be structural (LBRACE, AT) or EOF/NEWLINE
				if nextToken.Type == IDENTIFIER && nextToken.Semantic == SemCommand {
					t.Errorf("Unexpected switch to CommandMode, got shell command token: %s %q",
						nextToken.Type, nextToken.Value)
				}
			}
		})
	}
}

func TestDecoratorVsInlineVar(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectedAT  int // Number of AT tokens expected
		description string
	}{
		{
			name:        "decorator @timeout",
			input:       "@timeout(30s)",
			expectedAT:  1,
			description: "Should tokenize @timeout as decorator",
		},
		{
			name:        "inline @var usage",
			input:       "echo @var(PORT)",
			expectedAT:  1,
			description: "Should tokenize @ in @var() as AT token, let parser handle semantics",
		},
		{
			name:        "mixed usage",
			input:       "@timeout(30s) { echo @var(PORT) }",
			expectedAT:  2,
			description: "Both @ symbols should be tokenized",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			lexer := New(test.input)
			tokens := lexer.TokenizeToSlice()

			atCount := 0
			for _, token := range tokens {
				if token.Type == AT {
					atCount++
				}
			}

			if atCount != test.expectedAT {
				t.Errorf("%s: expected %d AT tokens, got %d",
					test.description, test.expectedAT, atCount)

				// Debug output
				t.Logf("All tokens:")
				for i, token := range tokens {
					t.Logf("  %d: %s %q", i, token.Type, token.Value)
				}
			}
		})
	}
}

func TestGetSemanticTokens(t *testing.T) {
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

func TestTokenizeToSlice(t *testing.T) {
	input := `var server: @timeout(30s) { echo "test"; node app.js }`

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

	// Verify we get reasonable token count (adjusted for shell text consolidation)
	if len(tokens) < 7 {
		t.Errorf("Expected at least 7 tokens for this input, got %d", len(tokens))
	}
}

func TestAtTokenDetection(t *testing.T) {
	tests := []struct {
		input       string
		shouldMatch bool
		atCount     int
	}{
		{"@timeout(30s)", true, 1},
		{"@var{}", true, 1},
		{"@test( args )", true, 1},
		{"@timeout @retry", true, 2},
		{"email@domain.com", true, 1}, // @ will be tokenized, parser decides semantics
		{"@ timeout", true, 1},        // @ will be tokenized, parser decides semantics
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			lexer := New(test.input)
			tokens := lexer.TokenizeToSlice()

			atTokens := 0
			for _, token := range tokens {
				if token.Type == AT {
					atTokens++
				}
			}

			if (atTokens > 0) != test.shouldMatch {
				t.Errorf("Expected AT token detection to be %v, got %v for input %q",
					test.shouldMatch, atTokens > 0, test.input)

				// Debug: show all tokens
				t.Logf("All tokens for %q:", test.input)
				for i, token := range tokens {
					t.Logf("  %d: %s %q", i, token.Type, token.Value)
				}
			}

			if atTokens != test.atCount {
				t.Errorf("Expected %d AT tokens, got %d for input %q",
					test.atCount, atTokens, test.input)
			}
		})
	}
}

func TestParseDecoratorArgs(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    []DecoratorArg
		expectError bool
	}{
		{
			name:  "positional argument",
			input: "30s",
			expected: []DecoratorArg{
				{Value: "30s", Line: 1, Column: 1},
			},
		},
		{
			name:  "named argument",
			input: "timeout=30s",
			expected: []DecoratorArg{
				{Name: "timeout", Value: "30s", Line: 1, Column: 1},
			},
		},
		{
			name:  "mixed positional and named",
			input: "30s, graceful=true",
			expected: []DecoratorArg{
				{Value: "30s", Line: 1, Column: 1},
				{Name: "graceful", Value: "true", Line: 1, Column: 1},
			},
		},
		{
			name:  "all named arguments",
			input: "timeout=30s, graceful=true",
			expected: []DecoratorArg{
				{Name: "timeout", Value: "30s", Line: 1, Column: 1},
				{Name: "graceful", Value: "true", Line: 1, Column: 1},
			},
		},
		{
			name:  "reordered named arguments",
			input: "graceful=true, timeout=30s",
			expected: []DecoratorArg{
				{Name: "graceful", Value: "true", Line: 1, Column: 1},
				{Name: "timeout", Value: "30s", Line: 1, Column: 1},
			},
		},
		{
			name:  "complex values with quotes",
			input: `port="8080", host="localhost"`,
			expected: []DecoratorArg{
				{Name: "port", Value: "8080", Line: 1, Column: 1},
				{Name: "host", Value: "localhost", Line: 1, Column: 1},
			},
		},
		{
			name:  "hyphenated parameter names",
			input: "retry-count=3, max-delay=10s",
			expected: []DecoratorArg{
				{Name: "retry-count", Value: "3", Line: 1, Column: 1},
				{Name: "max-delay", Value: "10s", Line: 1, Column: 1},
			},
		},
		{
			name:  "underscore parameter names",
			input: "max_attempts=5, wait_time=1s",
			expected: []DecoratorArg{
				{Name: "max_attempts", Value: "5", Line: 1, Column: 1},
				{Name: "wait_time", Value: "1s", Line: 1, Column: 1},
			},
		},
		{
			name:        "positional after named (should error)",
			input:       "timeout=30s, graceful",
			expectError: true,
		},
		{
			name:        "invalid parameter name with number start",
			input:       "1timeout=30s",
			expectError: true,
		},
		{
			name:        "invalid parameter name with special chars",
			input:       "time@out=30s",
			expectError: true,
		},
		{
			name:     "empty arguments",
			input:    "",
			expected: nil,
		},
		{
			name:     "whitespace only",
			input:    "   ",
			expected: nil,
		},
		{
			name:  "single quoted values",
			input: "command='echo hello'",
			expected: []DecoratorArg{
				{Name: "command", Value: "echo hello", Line: 1, Column: 1},
			},
		},
		{
			name:  "backtick quoted values",
			input: "script=`date +%Y-%m-%d`",
			expected: []DecoratorArg{
				{Name: "script", Value: "date +%Y-%m-%d", Line: 1, Column: 1},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			args, err := ParseDecoratorArgs(test.input, 1, 1)

			if test.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

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

func TestKotlinLikeParameterRules(t *testing.T) {
	// Test Kotlin-like parameter validation rules
	tests := []struct {
		name        string
		input       string
		shouldError bool
		errorMsg    string
	}{
		{
			name:        "valid positional only",
			input:       "30s, true, 5",
			shouldError: false,
		},
		{
			name:        "valid named only",
			input:       "timeout=30s, graceful=true, attempts=5",
			shouldError: false,
		},
		{
			name:        "valid mixed (positional first)",
			input:       "30s, graceful=true, attempts=5",
			shouldError: false,
		},
		{
			name:        "invalid mixed (positional after named)",
			input:       "timeout=30s, true",
			shouldError: true,
			errorMsg:    "positional argument follows named argument",
		},
		{
			name:        "invalid parameter name (starts with number)",
			input:       "1timeout=30s",
			shouldError: true,
			errorMsg:    "invalid parameter name",
		},
		{
			name:        "invalid parameter name (special chars)",
			input:       "time@out=30s",
			shouldError: true,
			errorMsg:    "invalid parameter name",
		},
		{
			name:        "valid parameter name with underscore",
			input:       "_timeout=30s",
			shouldError: false,
		},
		{
			name:        "valid parameter name with hyphen",
			input:       "retry-count=3",
			shouldError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := ParseDecoratorArgs(test.input, 1, 1)

			if test.shouldError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if !strings.Contains(err.Error(), test.errorMsg) {
					t.Errorf("Expected error containing %q, got %q", test.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestDecoratorExamples(t *testing.T) {
	// Test examples from the spec
	examples := []struct {
		name     string
		input    string
		expected []DecoratorArg
	}{
		{
			name:  "timeout with duration",
			input: "30s",
			expected: []DecoratorArg{
				{Value: "30s", Line: 1, Column: 1},
			},
		},
		{
			name:  "retry with named attempts",
			input: "attempts=3",
			expected: []DecoratorArg{
				{Name: "attempts", Value: "3", Line: 1, Column: 1},
			},
		},
		{
			name:  "timeout with graceful shutdown",
			input: "30s, graceful=true",
			expected: []DecoratorArg{
				{Value: "30s", Line: 1, Column: 1},
				{Name: "graceful", Value: "true", Line: 1, Column: 1},
			},
		},
		{
			name:  "var with environment variable",
			input: "NODE_ENV=development",
			expected: []DecoratorArg{
				{Name: "NODE_ENV", Value: "development", Line: 1, Column: 1},
			},
		},
		{
			name:  "parallel with worker count",
			input: "workers=4",
			expected: []DecoratorArg{
				{Name: "workers", Value: "4", Line: 1, Column: 1},
			},
		},
		{
			name:  "watch-files with pattern",
			input: `pattern="src/**/*"`,
			expected: []DecoratorArg{
				{Name: "pattern", Value: "src/**/*", Line: 1, Column: 1},
			},
		},
		{
			name:  "complex timeout with named duration",
			input: "duration=5m, graceful=true",
			expected: []DecoratorArg{
				{Name: "duration", Value: "5m", Line: 1, Column: 1},
				{Name: "graceful", Value: "true", Line: 1, Column: 1},
			},
		},
		{
			name:  "retry with delay duration",
			input: "attempts=3, delay=1.5s",
			expected: []DecoratorArg{
				{Name: "attempts", Value: "3", Line: 1, Column: 1},
				{Name: "delay", Value: "1.5s", Line: 1, Column: 1},
			},
		},
		{
			name:  "debounce with milliseconds",
			input: "debounce=500ms",
			expected: []DecoratorArg{
				{Name: "debounce", Value: "500ms", Line: 1, Column: 1},
			},
		},
	}

	for _, example := range examples {
		t.Run(example.name, func(t *testing.T) {
			args, err := ParseDecoratorArgs(example.input, 1, 1)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if diff := cmp.Diff(example.expected, args); diff != "" {
				t.Errorf("Example mismatch (-want +got):\n%s", diff)
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
		if token.Type == IDENTIFIER {
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
