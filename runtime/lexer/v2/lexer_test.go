package v2

import (
	"runtime"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// tokenExpectation represents an expected token for testing
type tokenExpectation struct {
	Type   TokenType
	Text   string
	Line   int
	Column int
}

// assertTokens compares actual tokens with expected, providing clear error messages
func assertTokens(t *testing.T, name string, input string, expected []tokenExpectation) {
	t.Helper()

	lexer := NewLexer(input)
	var actual []tokenExpectation

	for {
		token := lexer.NextToken()
		actual = append(actual, tokenExpectation{
			Type:   token.Type,
			Text:   token.Text,
			Line:   token.Position.Line,
			Column: token.Position.Column,
		})
		if token.Type == EOF {
			break
		}
	}

	// Use cmp.Diff for clean, exact output comparison
	if diff := cmp.Diff(expected, actual); diff != "" {
		t.Errorf("%s: token mismatch (-expected +actual):\n%s", name, diff)
	}
}

// TestEmptyInput tests the most basic case - empty input should return EOF
func TestEmptyInput(t *testing.T) {
	input := ""
	expected := []tokenExpectation{
		{EOF, "", 1, 1},
	}

	assertTokens(t, "empty input", input, expected)
}

// TestPerTokenTiming tests that lexer accumulates timing per token
func TestPerTokenTiming(t *testing.T) {
	input := "test"
	lexer := NewLexer(input)

	// Before any tokens, duration should be zero
	duration := lexer.Duration()
	if duration != 0 {
		t.Errorf("Duration should be zero before tokenizing, got %v", duration)
	}

	// Process first token - should accumulate time
	token1 := lexer.NextToken()
	duration1 := lexer.Duration()

	if duration1 <= 0 {
		t.Errorf("Duration should be positive after first token, got %v", duration1)
	}

	// Process second token - should accumulate more time
	token2 := lexer.NextToken()
	duration2 := lexer.Duration()

	if duration2 <= duration1 {
		t.Errorf("Duration should increase after second token, was %v now %v", duration1, duration2)
	}

	// Verify tokens are meaningful (not all ILLEGAL)
	if token1.Type == ILLEGAL && token2.Type == ILLEGAL {
		t.Errorf("Expected some meaningful tokens, got ILLEGAL for both")
	}

	// Duration should be measurable (not zero, not negative)
	if duration2 <= 0 {
		t.Errorf("Final duration should be positive, got %v", duration2)
	}
}

// TestZeroAllocation tests that tokenization doesn't allocate after lexer init
func TestZeroAllocation(t *testing.T) {
	input := "test input"
	lexer := NewLexer(input)

	// Force garbage collection and get baseline memory stats
	runtime.GC()
	var m1, m2 runtime.MemStats
	runtime.ReadMemStats(&m1)

	// Process tokens - this should not allocate
	var tokenCount int
	for {
		token := lexer.NextToken()
		tokenCount++
		if token.Type == EOF {
			break
		}
		// Prevent infinite loop in case lexer isn't working properly
		if tokenCount > 100 {
			t.Fatal("Too many tokens, possible infinite loop")
		}
	}

	// Check memory stats after tokenization
	runtime.ReadMemStats(&m2)

	// Calculate allocations during tokenization
	allocsDiff := m2.Mallocs - m1.Mallocs

	// Should have zero allocations during tokenization
	if allocsDiff > 0 {
		t.Errorf("Expected zero allocations during tokenization, got %d allocations", allocsDiff)
	}

	// Verify we processed some tokens
	if tokenCount < 1 {
		t.Errorf("Expected at least 1 token (EOF), got %d", tokenCount)
	}
}

// TestLexerResetWithInit tests that lexer can be reset with new input using Init
func TestLexerResetWithInit(t *testing.T) {
	lexer := NewLexer("first")

	// Process first input
	token1 := lexer.NextToken()
	_ = lexer.Duration() // Ignore first duration

	// Reset with new input using Init pattern (like Go's scanner)
	lexer.Init([]byte("second"))

	// Duration should reset to zero
	if lexer.Duration() != 0 {
		t.Errorf("Duration should reset to zero after Init, got %v", lexer.Duration())
	}

	// Should be able to process new input
	token2 := lexer.NextToken()
	duration2 := lexer.Duration()

	// Verify reset worked
	if duration2 <= 0 {
		t.Errorf("Duration should be positive after processing reset input, got %v", duration2)
	}

	// Tokens should be meaningful
	if token1.Type == ILLEGAL || token2.Type == ILLEGAL {
		t.Errorf("Expected meaningful tokens, got %v and %v", token1.Type, token2.Type)
	}
}

// BenchmarkLexerZeroAlloc benchmarks lexing performance with allocation tracking
func BenchmarkLexerZeroAlloc(b *testing.B) {
	inputBytes := []byte("echo hello world") // Pre-convert to avoid allocation in benchmark
	lexer := NewLexer("")                    // Create once

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		lexer.Init(inputBytes) // Reset with pre-converted bytes

		// This inner loop should have zero allocations
		for {
			token := lexer.NextToken()
			if token.Type == EOF {
				break
			}
		}
	}
}

// BenchmarkLexerWithDebug benchmarks lexing performance with debug enabled
func BenchmarkLexerWithDebug(b *testing.B) {
	inputBytes := []byte("echo hello world")
	lexer := NewLexerWithOpts("", WithDebug()) // Debug enabled

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		lexer.Init(inputBytes)

		for {
			token := lexer.NextToken()
			if token.Type == EOF {
				break
			}
		}
	}
}

// TestBenchmarkPerformanceRequirements verifies benchmark meets performance requirements
func TestBenchmarkPerformanceRequirements(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping benchmark performance test in short mode")
	}

	// Run the benchmark
	result := testing.Benchmark(BenchmarkLexerZeroAlloc)

	// Performance requirements - realistic target for actual lexing work
	// Current baseline: 76ns doing nothing, so 200ns for real tokenization is reasonable
	maxNsPerOp := int64(200)   // Realistic target: 200ns per token (1250+ lines/ms)
	maxAllocsPerOp := int64(0) // Zero allocations required
	maxBytesPerOp := int64(0)  // Zero bytes allocated required

	// Check timing requirement
	if result.NsPerOp() > maxNsPerOp {
		t.Errorf("Performance regression: %d ns/op exceeds limit of %d ns/op",
			result.NsPerOp(), maxNsPerOp)
	}

	// Check allocation requirements
	if result.AllocsPerOp() > maxAllocsPerOp {
		t.Errorf("Allocation regression: %d allocs/op exceeds limit of %d allocs/op",
			result.AllocsPerOp(), maxAllocsPerOp)
	}

	if result.AllocedBytesPerOp() > maxBytesPerOp {
		t.Errorf("Memory regression: %d bytes/op exceeds limit of %d bytes/op",
			result.AllocedBytesPerOp(), maxBytesPerOp)
	}

	// Report current performance for visibility
	t.Logf("Current performance: %d ns/op, %d allocs/op, %d bytes/op",
		result.NsPerOp(), result.AllocsPerOp(), result.AllocedBytesPerOp())
}

// TestDebugTelemetryZeroAlloc tests that debug disabled maintains zero allocation
func TestDebugTelemetryZeroAlloc(t *testing.T) {
	input := "test input"

	// Create lexer without debug (should be zero alloc)
	lexer := NewLexer(input)

	runtime.GC()
	var m1, m2 runtime.MemStats
	runtime.ReadMemStats(&m1)

	// Process tokens - should have zero allocations
	for {
		token := lexer.NextToken()
		if token.Type == EOF {
			break
		}
	}

	runtime.ReadMemStats(&m2)
	allocsDiff := m2.Mallocs - m1.Mallocs

	if allocsDiff > 0 {
		t.Errorf("Expected zero allocations with debug disabled, got %d", allocsDiff)
	}
}

// TestUnicodePositionTracking tests that position tracking works correctly with Unicode
func TestUnicodePositionTracking(t *testing.T) {
	lexer := NewLexer("")

	tests := []struct {
		name        string
		input       string
		advances    int // How many times to call advanceChar
		expectedPos struct {
			position int
			line     int
			column   int
		}
	}{
		{
			name:     "ASCII characters",
			input:    "hello",
			advances: 3,
			expectedPos: struct {
				position int
				line     int
				column   int
			}{3, 1, 4}, // position 3, column 4 (1-indexed)
		},
		{
			name:     "2-byte Unicode (café)",
			input:    "café",
			advances: 3, // c, a, f -> should be at é
			expectedPos: struct {
				position int
				line     int
				column   int
			}{3, 1, 4}, // position 3 (at é), column 4
		},
		{
			name:     "4-byte Unicode emoji",
			input:    "😀test",
			advances: 1, // Just advance past emoji
			expectedPos: struct {
				position int
				line     int
				column   int
			}{4, 1, 2}, // position 4 (past 4-byte emoji), column 2
		},
		{
			name:     "Mixed Unicode and newlines",
			input:    "hello\n世界",
			advances: 7, // past hello\n世
			expectedPos: struct {
				position int
				line     int
				column   int
			}{9, 2, 2}, // line 2, column 2 (past 世, at 界)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer.Init([]byte(tt.input))

			// Advance the specified number of characters
			for i := 0; i < tt.advances; i++ {
				lexer.advanceChar()
			}

			// Check position tracking
			if lexer.position != tt.expectedPos.position {
				t.Errorf("position = %d, want %d", lexer.position, tt.expectedPos.position)
			}
			if lexer.line != tt.expectedPos.line {
				t.Errorf("line = %d, want %d", lexer.line, tt.expectedPos.line)
			}
			if lexer.column != tt.expectedPos.column {
				t.Errorf("column = %d, want %d", lexer.column, tt.expectedPos.column)
			}
		})
	}
}

// TestDebugTelemetryEnabled tests that debug mode provides telemetry
func TestDebugTelemetryEnabled(t *testing.T) {
	input := "test input"

	// Create lexer with debug enabled
	lexer := NewLexerWithOpts(input, WithDebug())

	// Process tokens
	for {
		token := lexer.NextToken()
		if token.Type == EOF {
			break
		}
	}

	// Should have debug telemetry available
	if !lexer.HasDebugTelemetry() {
		t.Error("Expected debug telemetry to be available when debug enabled")
	}

	// Should be able to get token timing stats
	stats := lexer.GetTokenStats()
	if len(stats) == 0 {
		t.Error("Expected token stats when debug enabled")
	}
}
