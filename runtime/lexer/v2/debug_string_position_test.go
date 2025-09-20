package v2

import (
	"testing"
)

// TestDebugStringPositions helps verify that string position tracking is actually correct
func TestDebugStringPositions(t *testing.T) {
	testCases := []struct {
		name  string
		input string
	}{
		{
			name:  "simple double quote",
			input: `"hello"`,
		},
		{
			name:  "escaped quote",
			input: `"He said \"hello\""`,
		},
		{
			name:  "with spaces",
			input: ` "hello" `,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			lexer := NewLexer(tc.input)

			t.Logf("Input: %q (length %d)", tc.input, len(tc.input))
			t.Logf("Character positions:")
			for i, ch := range []byte(tc.input) {
				t.Logf("  pos %d: %q", i, ch)
			}

			// Get all tokens and their positions
			var tokens []Token
			for {
				token := lexer.NextToken()
				tokens = append(tokens, token)
				t.Logf("Token: %v %q at line %d, col %d",
					token.Type, token.Text, token.Position.Line, token.Position.Column)
				if token.Type == EOF {
					break
				}
			}

			// Manual position calculation
			t.Logf("Manual calculation:")
			if tc.input == `"hello"` {
				t.Logf("  String starts at position 0 = column 1")
				t.Logf("  String ends at position 6 (after closing quote)")
				t.Logf("  EOF should be at position 7 = column 7")
			} else if tc.input == `"He said \"hello\""` {
				t.Logf("  String starts at position 0 = column 1")
				t.Logf("  String content: positions 0-18")
				t.Logf("  EOF should be at position 19 = column 19")
			}
		})
	}
}
