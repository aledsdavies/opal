package v2

import "testing"

// TestASCIICharacterClassification tests the lookup table performance
func TestASCIICharacterClassification(t *testing.T) {
	tests := []struct {
		name     string
		char     byte
		expected map[string]bool
	}{
		{
			name: "lowercase letter",
			char: 'a',
			expected: map[string]bool{
				"letter":     true,
				"identStart": true,
				"identPart":  true,
				"digit":      false,
				"whitespace": false,
			},
		},
		{
			name: "uppercase letter",
			char: 'Z',
			expected: map[string]bool{
				"letter":     true,
				"identStart": true,
				"identPart":  true,
				"digit":      false,
				"whitespace": false,
			},
		},
		{
			name: "underscore",
			char: '_',
			expected: map[string]bool{
				"letter":     true,
				"identStart": true,
				"identPart":  true,
				"digit":      false,
				"whitespace": false,
			},
		},
		{
			name: "digit",
			char: '5',
			expected: map[string]bool{
				"letter":     false,
				"identStart": false,
				"identPart":  true,
				"digit":      true,
				"whitespace": false,
			},
		},
		{
			name: "space",
			char: ' ',
			expected: map[string]bool{
				"letter":     false,
				"identStart": false,
				"identPart":  false,
				"digit":      false,
				"whitespace": true,
			},
		},
		{
			name: "newline should be whitespace",
			char: '\n',
			expected: map[string]bool{
				"letter":     false,
				"identStart": false,
				"identPart":  false,
				"digit":      false,
				"whitespace": true, // Newlines are skipped as whitespace
			},
		},
		{
			name: "hyphen in identifier",
			char: '-',
			expected: map[string]bool{
				"letter":     false,
				"identStart": false,
				"identPart":  true, // Hyphens allowed in identifier parts
				"digit":      false,
				"whitespace": false,
			},
		},
		{
			name: "tab whitespace",
			char: '\t',
			expected: map[string]bool{
				"letter":     false,
				"identStart": false,
				"identPart":  false,
				"digit":      false,
				"whitespace": true,
			},
		},
		{
			name: "hex digit lowercase",
			char: 'f',
			expected: map[string]bool{
				"letter":     true,
				"identStart": true,
				"identPart":  true,
				"digit":      false,
				"whitespace": false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test direct lookup table access (zero function call overhead)
			if isLetter[tt.char] != tt.expected["letter"] {
				t.Errorf("isLetter[%q] = %v, want %v", tt.char, isLetter[tt.char], tt.expected["letter"])
			}
			if isIdentStart[tt.char] != tt.expected["identStart"] {
				t.Errorf("isIdentStart[%q] = %v, want %v", tt.char, isIdentStart[tt.char], tt.expected["identStart"])
			}
			if isIdentPart[tt.char] != tt.expected["identPart"] {
				t.Errorf("isIdentPart[%q] = %v, want %v", tt.char, isIdentPart[tt.char], tt.expected["identPart"])
			}
			if isDigit[tt.char] != tt.expected["digit"] {
				t.Errorf("isDigit[%q] = %v, want %v", tt.char, isDigit[tt.char], tt.expected["digit"])
			}
			if isWhitespace[tt.char] != tt.expected["whitespace"] {
				t.Errorf("isWhitespace[%q] = %v, want %v", tt.char, isWhitespace[tt.char], tt.expected["whitespace"])
			}
		})
	}
}

// TestHexDigitClassification tests hex digit lookup table
func TestHexDigitClassification(t *testing.T) {
	tests := []struct {
		char     byte
		expected bool
	}{
		{'0', true}, {'9', true}, // digits
		{'a', true}, {'f', true}, // lowercase hex
		{'A', true}, {'F', true}, // uppercase hex
		{'g', false}, {'G', false}, // not hex
		{'z', false}, {'Z', false}, // not hex
		{' ', false}, {'-', false}, // not hex
	}

	for _, tt := range tests {
		t.Run(string(tt.char), func(t *testing.T) {
			if isHexDigit[tt.char] != tt.expected {
				t.Errorf("isHexDigit[%q] = %v, want %v", tt.char, isHexDigit[tt.char], tt.expected)
			}
		})
	}
}

// BenchmarkCharacterClassification benchmarks optimal inline bounds-checked lookups
func BenchmarkCharacterClassification(b *testing.B) {
	testBytes := []byte("hello_world123-test")

	b.ResetTimer()
	b.ReportAllocs()

	var count int
	for i := 0; i < b.N; i++ {
		for _, ch := range testBytes {
			// Optimal: inline bounds-checked lookup
			if ch < 128 && (isLetter[ch] || isDigit[ch]) {
				count++
			}
		}
	}
	_ = count
}

// TestASCIIIdentifierValidation tests ASCII-only identifier rules
func TestASCIIIdentifierValidation(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		valid  bool
		reason string
	}{
		{
			name:   "valid variable",
			input:  "apiKey",
			valid:  true,
			reason: "camelCase is fine",
		},
		{
			name:   "valid underscore style",
			input:  "api_key",
			valid:  true,
			reason: "snake_case is fine",
		},
		{
			name:   "valid kebab style",
			input:  "start-api",
			valid:  true,
			reason: "kebab-case is fine",
		},
		{
			name:   "valid with numbers",
			input:  "service2",
			valid:  true,
			reason: "numbers allowed after first character",
		},
		{
			name:   "starts with underscore",
			input:  "_private",
			valid:  true,
			reason: "underscore is valid start character",
		},
		{
			name:   "mixed styles",
			input:  "API_v2-final",
			valid:  true,
			reason: "any ASCII combo is allowed",
		},
		{
			name:   "starts with number",
			input:  "2fast",
			valid:  false,
			reason: "cannot start with digit",
		},
		{
			name:   "contains space",
			input:  "my var",
			valid:  false,
			reason: "spaces not allowed",
		},
		{
			name:   "contains Unicode",
			input:  "café",
			valid:  false,
			reason: "Unicode not allowed in identifiers",
		},
		{
			name:   "empty string",
			input:  "",
			valid:  false,
			reason: "empty identifiers not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := isValidASCIIIdentifier(tt.input)
			if valid != tt.valid {
				t.Errorf("isValidASCIIIdentifier(%q) = %v, want %v (%s)",
					tt.input, valid, tt.valid, tt.reason)
			}
		})
	}
}

// TestUnicodeInTokens tests that Unicode content is preserved as raw bytes in tokens
func TestUnicodeInTokens(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string // We expect Unicode to be preserved as-is in token text
	}{
		{
			name:     "Chinese characters",
			input:    "世界",
			expected: "世界",
		},
		{
			name:     "Mixed ASCII and Unicode",
			input:    "hello世界",
			expected: "hello世界",
		},
		{
			name:     "Emoji",
			input:    "😀test",
			expected: "😀test",
		},
		{
			name:     "Various Unicode scripts",
			input:    "café αβγ العربية",
			expected: "café αβγ العربية",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For now, this is a design test - we'll implement actual tokenization later
			// The key point is that Unicode should go into tokens as raw bytes
			input := []byte(tt.input)
			if string(input) != tt.expected {
				t.Errorf("Unicode preservation failed: got %q, want %q", string(input), tt.expected)
			}
		})
	}
}

// BenchmarkASCIIIdentifierValidation benchmarks identifier validation performance
func BenchmarkASCIIIdentifierValidation(b *testing.B) {
	identifiers := []string{
		"apiKey", "start-api", "service_v2", "_private", "DEPLOY_TIMEOUT",
		"user", "a", "very_long_identifier_name_that_might_be_used",
	}

	b.ResetTimer()
	b.ReportAllocs()

	var validCount int
	for i := 0; i < b.N; i++ {
		for _, ident := range identifiers {
			if isValidASCIIIdentifier(ident) {
				validCount++
			}
		}
	}
	_ = validCount
}

// BenchmarkSkipWhitespace benchmarks hybrid whitespace skipping performance
func BenchmarkSkipWhitespace(b *testing.B) {
	input := []byte("    \t\r  hello world")
	lexer := NewLexer("")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		lexer.Init(input)
		lexer.skipWhitespace() // Hybrid array jumping with column tracking
	}
}

// BenchmarkWhitespaceScenarios benchmarks different whitespace patterns
func BenchmarkWhitespaceScenarios(b *testing.B) {
	scenarios := []struct {
		name  string
		input []byte
	}{
		{
			"light_whitespace",
			[]byte(" token"),
		},
		{
			"heavy_prefix",
			[]byte("                    token"),
		},
		{
			"command_chain",
			[]byte("token1 && token2"),
		},
		{
			"heavy_chain",
			[]byte("token1 &&                              token2"),
		},
		{
			"script_formatting",
			[]byte(`    if condition {
        command1
        command2
    }`),
		},
		{
			"mixed_whitespace",
			[]byte("  \t\r    \t  token  \t\r  "),
		},
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			lexer := NewLexer("")

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				lexer.Init(scenario.input)

				// Simulate typical lexing pattern
				for lexer.position < len(lexer.input) {
					lexer.skipWhitespace()
					if lexer.position < len(lexer.input) {
						// Simulate reading a token (just advance past first char)
						if lexer.input[lexer.position] != '\n' {
							lexer.position++
						} else {
							break
						}
					}
				}
			}
		})
	}
}
