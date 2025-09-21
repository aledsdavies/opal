package v2

import (
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

// generateRealisticInput creates input with mixed token types for testing
func generateRealisticInput(tokenCount int) string {
	var parts []string

	// Realistic devcmd-style content
	patterns := []string{
		"var service_name = \"api-server\"",
		"for env in @var(ENVIRONMENTS) {",
		"kubectl apply -f deployment.yaml",
		"@confirm(\"Deploy to production?\")",
		"echo \"Deploying @var(service_name)\"",
		"if @var(ENV) == \"production\" {",
		"@retry(3) { curl -f @var(HEALTH_URL) }",
		"} else {",
		"echo \"Development deployment\"",
		"}",
	}

	// Generate tokens by repeating patterns
	for i := 0; i < tokenCount; {
		pattern := patterns[i%len(patterns)]
		parts = append(parts, pattern)

		// Rough estimate: each pattern has ~5-8 tokens
		i += 6
	}

	return strings.Join(parts, "\n")
}

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
	tokens := lexer.GetTokens() // Use batch interface
	var actual []tokenExpectation

	for _, token := range tokens {
		actual = append(actual, tokenExpectation{
			Type:   token.Type,
			Text:   token.String(),
			Line:   token.Position.Line,
			Column: token.Position.Column,
		})
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
	lexer := NewLexer(input, WithTiming())

	// Before any tokens, duration should be zero
	duration := lexer.Duration()
	if duration != 0 {
		t.Errorf("Duration should be zero before tokenizing, got %v", duration)
	}

	// Process first token - timing should be available immediately
	token1 := lexer.NextToken()
	duration1 := lexer.Duration()

	if duration1 <= 0 {
		t.Errorf("Duration should be positive after first token, got %v", duration1)
	}

	// Process second token - should have more time
	token2 := lexer.NextToken()
	duration2 := lexer.Duration()

	if duration2 < duration1 {
		t.Errorf("Duration should not decrease, was %v now %v", duration1, duration2)
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

// TestBatchTiming tests that batch interface provides timing
func TestBatchTiming(t *testing.T) {
	input := "test input more tokens here"
	lexer := NewLexer(input, WithTiming())

	// Before processing, duration should be zero
	duration := lexer.Duration()
	if duration != 0 {
		t.Errorf("Duration should be zero before processing, got %v", duration)
	}

	// Get all tokens at once
	tokens := lexer.GetTokens()
	duration = lexer.Duration()

	// Should have timing after batch processing
	if duration <= 0 {
		t.Errorf("Duration should be positive after batch processing, got %v", duration)
	}

	// Should have gotten some tokens
	if len(tokens) < 2 {
		t.Errorf("Expected multiple tokens, got %d", len(tokens))
	}

	// Last token should be EOF
	if len(tokens) > 0 && tokens[len(tokens)-1].Type != EOF {
		t.Errorf("Expected last token to be EOF, got %v", tokens[len(tokens)-1].Type)
	}
}

// TestBufferBoundaries tests buffering across realistic input sizes
func TestBufferBoundaries(t *testing.T) {
	tests := []struct {
		name        string
		tokenCount  int
		description string
	}{
		{"small_file", 100, "Small config file"},
		{"medium_file", 1000, "Medium script file"},
		{"large_file", 5000, "Large deployment script"},
		{"very_large_file", 20000, "Very large complex script"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Generate realistic input
			input := generateRealisticInput(tt.tokenCount)

			lexer := NewLexer(input)

			// Test streaming interface - count actual tokens
			var streamTokenCount int
			for {
				token := lexer.NextToken()
				streamTokenCount++
				if token.Type == EOF {
					break
				}
			}

			// Reset and test batch interface
			lexer.Init([]byte(input))
			batchTokens := lexer.GetTokens()

			// Both should give same count
			if len(batchTokens) != streamTokenCount {
				t.Errorf("Token count mismatch: streaming=%d, batch=%d", streamTokenCount, len(batchTokens))
			}

			// Verify we got a reasonable number of tokens (at least target ballpark)
			if streamTokenCount < tt.tokenCount/2 {
				t.Errorf("Too few tokens: expected ~%d, got %d", tt.tokenCount, streamTokenCount)
			}

			t.Logf("%s: %d tokens processed successfully", tt.description, streamTokenCount)
		})
	}
}

// TestBufferRefillConsistency tests that tokens are consistent across buffer refills
func TestBufferRefillConsistency(t *testing.T) {
	// Create realistic input that will definitely span multiple buffers
	input := generateRealisticInput(1000) // ~1000 realistic tokens

	lexer1 := NewLexer(input)
	lexer2 := NewLexer(input)

	// Get all tokens via streaming
	var streamTokens []Token
	for {
		token := lexer1.NextToken()
		streamTokens = append(streamTokens, token)
		if token.Type == EOF {
			break
		}
	}

	// Get all tokens via batch
	batchTokens := lexer2.GetTokens()

	// Should be identical
	if len(streamTokens) != len(batchTokens) {
		t.Fatalf("Token count mismatch: stream=%d, batch=%d", len(streamTokens), len(batchTokens))
	}

	for i, streamToken := range streamTokens {
		batchToken := batchTokens[i]
		if streamToken.Type != batchToken.Type {
			t.Errorf("Token %d type mismatch: stream=%v, batch=%v", i, streamToken.Type, batchToken.Type)
		}
		if string(streamToken.Text) != string(batchToken.Text) {
			t.Errorf("Token %d text mismatch: stream=%q, batch=%q", i, streamToken.String(), batchToken.String())
		}
		if streamToken.Position != batchToken.Position {
			t.Errorf("Token %d position mismatch: stream=%v, batch=%v", i, streamToken.Position, batchToken.Position)
		}
	}
}

// TestBufferSizePerformance tests different buffer sizes to find optimal setting
func TestBufferSizePerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping buffer size performance test in short mode")
	}

	// Large realistic input for meaningful performance testing
	input := generateRealisticInput(5000) // ~5000 tokens of realistic content

	// Test realistic buffer sizes that would be used in production
	bufferSizes := []int{32, 64, 128, 256, 512, 1024}
	results := make(map[int]time.Duration)

	for _, size := range bufferSizes {
		t.Run(fmt.Sprintf("buffer_size_%d", size), func(t *testing.T) {
			// We'll need to expose buffer size configuration for this test
			// For now, just test with default and record the pattern

			start := time.Now()

			lexer := NewLexer(input)
			tokens := lexer.GetTokens()

			elapsed := time.Since(start)
			results[size] = elapsed

			// Sanity check
			if len(tokens) < 1000 {
				t.Errorf("Expected many tokens, got %d", len(tokens))
			}

			t.Logf("Buffer size %d: %v (%d tokens)", size, elapsed, len(tokens))
		})
	}

	// Log results for analysis
	t.Logf("Buffer size performance results:")
	for _, size := range bufferSizes {
		t.Logf("  Size %d: %v", size, results[size])
	}
}

// TestMixedAccessPatterns tests combining streaming and batch operations
func TestMixedAccessPatterns(t *testing.T) {
	input := "first second third fourth fifth"

	t.Run("get_tokens_after_next_token", func(t *testing.T) {
		lexer := NewLexer(input)

		// Consume first two tokens via streaming
		token1 := lexer.NextToken()
		token2 := lexer.NextToken()

		if token1.String() != "first" || token2.String() != "second" {
			t.Errorf("Streaming failed: got %q, %q", token1.String(), token2.String())
		}

		// Now get all tokens (should include the ones already consumed)
		allTokens := lexer.GetTokens()

		// Should get all tokens including the previously consumed ones
		expectedTexts := []string{"first", "second", "third", "fourth", "fifth", ""}
		if len(allTokens) != len(expectedTexts) {
			t.Fatalf("Expected %d tokens, got %d", len(expectedTexts), len(allTokens))
		}

		for i, expected := range expectedTexts {
			if allTokens[i].String() != expected {
				t.Errorf("Token %d: expected %q, got %q", i, expected, allTokens[i].String())
			}
		}

		// Last token should be EOF
		if allTokens[len(allTokens)-1].Type != EOF {
			t.Errorf("Expected last token to be EOF, got %v", allTokens[len(allTokens)-1].Type)
		}
	})

	t.Run("next_token_after_get_tokens", func(t *testing.T) {
		lexer := NewLexer(input)

		// Get all tokens first
		allTokens := lexer.GetTokens()

		// Verify we got all tokens
		if len(allTokens) < 5 || allTokens[len(allTokens)-1].Type != EOF {
			t.Errorf("GetTokens failed: got %d tokens", len(allTokens))
		}

		// Now try NextToken() - should return EOF since we're at the end
		nextToken := lexer.NextToken()
		if nextToken.Type != EOF {
			t.Errorf("Expected EOF after GetTokens(), got %v", nextToken.Type)
		}
	})

	t.Run("stream_then_batch", func(t *testing.T) {
		lexer := NewLexer(input)

		// Get a few tokens via streaming
		token1 := lexer.NextToken()
		token2 := lexer.NextToken()

		if token1.String() != "first" || token2.String() != "second" {
			t.Errorf("Streaming failed: got %q, %q", token1.String(), token2.String())
		}

		// Reset and use batch
		lexer.Init([]byte(input))
		tokens := lexer.GetTokens()

		if len(tokens) < 5 || tokens[0].String() != "first" {
			t.Errorf("Batch after streaming failed")
		}
	})

	t.Run("reset_buffer_state", func(t *testing.T) {
		lexer := NewLexer(input)

		// Consume some tokens to populate buffer
		lexer.NextToken() // first
		lexer.NextToken() // second

		// Reset with new input
		newInput := "alpha beta gamma"
		lexer.Init([]byte(newInput))

		// First token from new input should be correct
		token := lexer.NextToken()
		if token.String() != "alpha" {
			t.Errorf("Reset failed: expected 'alpha', got %q", token.String())
		}
	})
}

// TestTimingLevels tests that different timing levels work correctly
func TestTimingLevels(t *testing.T) {
	input := "test input"

	t.Run("no_timing", func(t *testing.T) {
		lexer := NewLexer(input, WithNoTiming())

		_ = lexer.NextToken()
		if lexer.Duration() != 0 {
			t.Errorf("Expected no timing with TimingNone, got %v", lexer.Duration())
		}

		_ = lexer.GetTokens()
		if lexer.Duration() != 0 {
			t.Errorf("Expected no timing with TimingNone, got %v", lexer.Duration())
		}
	})

	t.Run("timing_enabled", func(t *testing.T) {
		lexer := NewLexer(input, WithTiming())

		// With timing enabled, both streaming and batch should have timing
		_ = lexer.NextToken()
		if lexer.Duration() <= 0 {
			t.Errorf("Expected timing for streaming with WithTiming(), got %v", lexer.Duration())
		}

		lexer.Init([]byte(input))
		_ = lexer.GetTokens()
		if lexer.Duration() <= 0 {
			t.Errorf("Expected timing for batch with WithTiming(), got %v", lexer.Duration())
		}
	})

	t.Run("fine_grain_timing", func(t *testing.T) {
		lexer := NewLexer(input, WithFineGrainTiming())

		_ = lexer.NextToken()
		if lexer.Duration() <= 0 {
			t.Errorf("Expected timing for streaming with WithFineGrainTiming(), got %v", lexer.Duration())
		}
	})
}

// BenchmarkGetTokensAfterConsuming measures performance impact of including consumed tokens
func BenchmarkGetTokensAfterConsuming(b *testing.B) {
	input := generateRealisticInput(1000) // ~1000 tokens

	b.Run("fresh_get_tokens", func(b *testing.B) {
		lexer := NewLexer("")
		inputBytes := []byte(input)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			lexer.Init(inputBytes)
			_ = lexer.GetTokens() // Get all tokens fresh
		}
	})

	b.Run("get_tokens_after_consuming_10", func(b *testing.B) {
		lexer := NewLexer("")
		inputBytes := []byte(input)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			lexer.Init(inputBytes)

			// Consume 10 tokens first
			for j := 0; j < 10; j++ {
				lexer.NextToken()
			}

			_ = lexer.GetTokens() // Get remaining + consumed tokens
		}
	})

	b.Run("get_tokens_after_consuming_100", func(b *testing.B) {
		lexer := NewLexer("")
		inputBytes := []byte(input)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			lexer.Init(inputBytes)

			// Consume 100 tokens first
			for j := 0; j < 100; j++ {
				lexer.NextToken()
			}

			_ = lexer.GetTokens() // Get remaining + consumed tokens
		}
	})
}

// BenchmarkTokenCopyOverhead measures just the copying overhead
func BenchmarkTokenCopyOverhead(b *testing.B) {
	// Create a buffer with 1000 tokens
	tokens := make([]Token, 1000)
	for i := range tokens {
		tokens[i] = Token{Type: IDENTIFIER, Text: []byte("test"), Position: Position{Line: 1, Column: i}}
	}

	b.Run("copy_0_tokens", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var result []Token
			// Copy 0 tokens
			for j := 0; j < 0; j++ {
				result = append(result, tokens[j])
			}
			_ = result
		}
	})

	b.Run("copy_10_tokens", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var result []Token
			// Copy 10 tokens
			for j := 0; j < 10; j++ {
				result = append(result, tokens[j])
			}
			_ = result
		}
	})

	b.Run("copy_100_tokens", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var result []Token
			// Copy 100 tokens
			for j := 0; j < 100; j++ {
				result = append(result, tokens[j])
			}
			_ = result
		}
	})
}

// TestConsistentAPI demonstrates the consistent behavior between NextToken and GetTokens
func TestConsistentAPI(t *testing.T) {
	input := "var name = value"

	// Scenario: Parser consumes a few tokens, then formatter wants all tokens
	lexer := NewLexer(input)

	// Parser consumes first few tokens
	varToken := lexer.NextToken()  // "var"
	nameToken := lexer.NextToken() // "name"

	// Formatter wants all tokens (including the ones parser already consumed)
	allTokens := lexer.GetTokens()

	// Should get: ["var", "name", "=", "value", EOF]
	expected := []string{"var", "name", "=", "value", ""}
	if len(allTokens) != len(expected) {
		t.Fatalf("Expected %d tokens, got %d", len(expected), len(allTokens))
	}

	for i, expectedText := range expected {
		if allTokens[i].String() != expectedText {
			t.Errorf("Token %d: expected %q, got %q", i, expectedText, allTokens[i].String())
		}
	}

	t.Logf("✓ Parser consumed: %q, %q", varToken.String(), nameToken.String())
	t.Logf("✓ GetTokens() returned all %d tokens correctly", len(allTokens))
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
	lexer := NewLexer("first", WithTiming())

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

	// Verify reset worked - should have positive duration immediately
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
	lexer := NewLexer("", WithDebug()) // Debug enabled

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

	// Performance requirements - no timing overhead (default mode)
	// With no timing: 200ns is excellent performance (5000+ lines/ms)
	maxNsPerOp := int64(200)   // Realistic target: 200ns per token without timing (5000+ lines/ms)
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

	// Create lexer with debug and fine-grain timing enabled (required for token stats)
	lexer := NewLexer(input, WithDebug(), WithFineGrainTiming())

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

	// Should be able to get token timing stats (requires fine-grain timing)
	stats := lexer.GetTokenStats()
	if len(stats) == 0 {
		t.Error("Expected token stats when debug and fine-grain timing enabled")
	}
}

// TestPositionTrackingWithWhitespace tests accurate position tracking through various whitespace scenarios
func TestPositionTrackingWithWhitespace(t *testing.T) {
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
			name:     "advance through spaces",
			input:    "   hello",
			advances: 3, // Advance past the 3 spaces
			expectedPos: struct {
				position int
				line     int
				column   int
			}{3, 1, 4}, // position 3, column 4 (at 'h')
		},
		{
			name:     "advance through tabs",
			input:    "\t\thello",
			advances: 2, // Advance past 2 tabs
			expectedPos: struct {
				position int
				line     int
				column   int
			}{2, 1, 3}, // position 2, column 3 (at 'h')
		},
		{
			name:     "advance through mixed whitespace",
			input:    " \t hello",
			advances: 3, // Advance past " \t "
			expectedPos: struct {
				position int
				line     int
				column   int
			}{3, 1, 4}, // position 3, column 4 (at 'h')
		},
		{
			name:     "advance through newlines",
			input:    "line1\nline2",
			advances: 6, // Advance past "line1\n"
			expectedPos: struct {
				position int
				line     int
				column   int
			}{6, 2, 1}, // position 6, line 2, column 1 (at 'l' in line2)
		},
		{
			name:     "advance through multiple newlines",
			input:    "first\n\n\nsecond",
			advances: 8, // Advance past "first\n\n\n"
			expectedPos: struct {
				position int
				line     int
				column   int
			}{8, 4, 1}, // position 8, line 4, column 1 (at 's' in second)
		},
		{
			name:     "advance through Unicode characters",
			input:    "café test",
			advances: 4, // Advance past "café" (é is 2 bytes)
			expectedPos: struct {
				position int
				line     int
				column   int
			}{5, 1, 5}, // position 5 (byte position), column 5 (character position)
		},
		{
			name:     "advance through 4-byte Unicode emoji",
			input:    "😀test",
			advances: 1, // Advance past emoji (4 bytes)
			expectedPos: struct {
				position int
				line     int
				column   int
			}{4, 1, 2}, // position 4 (byte position), column 2 (character position)
		},
		{
			name:     "complex whitespace and Unicode",
			input:    " \t😀\n 世界",
			advances: 5, // Advance past " \t😀\n " to be at '世'
			expectedPos: struct {
				position int
				line     int
				column   int
			}{8, 2, 2}, // position 8, line 2, column 2 (at '世')
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

// TestWhitespaceSkipping tests that whitespace is properly skipped without creating tokens
func TestWhitespaceSkipping(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "only spaces",
			input: "     ",
			expected: []tokenExpectation{
				{EOF, "", 1, 6}, // EOF at column 6 (after 5 spaces)
			},
		},
		{
			name:  "only tabs",
			input: "\t\t\t",
			expected: []tokenExpectation{
				{EOF, "", 1, 4}, // EOF at column 4 (after 3 tabs)
			},
		},
		{
			name:  "only newlines",
			input: "\n\n\n",
			expected: []tokenExpectation{
				{EOF, "", 4, 1}, // EOF at line 4, column 1
			},
		},
		{
			name:  "mixed whitespace only",
			input: " \t\n \t\n ",
			expected: []tokenExpectation{
				{EOF, "", 3, 2}, // EOF at line 3, column 2 (after final space)
			},
		},
		{
			name:  "identifier with surrounding whitespace",
			input: "  \t hello \n\t ",
			expected: []tokenExpectation{
				{IDENTIFIER, "hello", 1, 5}, // "hello" at column 5 (after "  \t ")
				{EOF, "", 2, 3},             // EOF at line 2, column 3 (after "\t ")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.name, tt.input, tt.expected)
		})
	}
}
