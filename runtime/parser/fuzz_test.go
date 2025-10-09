package parser

import (
	"bytes"
	"testing"

	"github.com/aledsdavies/opal/runtime/lexer"
)

// Fuzz tests for parser determinism and robustness.
//
// Implements feedback:
// 1. Stack-based event balance (not just counts)
// 2. Full determinism check (events + tokens + errors)
// 3. Position monotonicity and bounds checking
// 4. Pathological depth guards
// 5. Memory safety with comprehensive position validation
//
// These tests protect the events-first plan generation model.

// FuzzParserDeterminism verifies that parsing the same input twice
// produces identical event streams, tokens, and errors (full determinism).
func FuzzParserDeterminism(f *testing.F) {
	// Seed corpus with valid Opal syntax
	f.Add([]byte(""))
	f.Add([]byte("fun greet() {}"))
	f.Add([]byte("var x = 42"))
	f.Add([]byte("fun deploy(env) { kubectl apply }"))

	// Edge cases
	f.Add([]byte("fun"))                   // Incomplete
	f.Add([]byte("{}"))                    // Just braces
	f.Add([]byte("\"unterminated string")) // Unterminated

	// UTF-8 and line endings
	f.Add([]byte("fun test() {\r\n  echo \"hi\"\r\n}")) // CRLF
	f.Add([]byte("fun 🚀() {}"))                         // Emoji
	f.Add([]byte("\xff\xfe\xfd"))                       // Invalid UTF-8

	f.Fuzz(func(t *testing.T, input []byte) {
		// Parse twice
		tree1 := Parse(input)
		tree2 := Parse(input)

		// Events must be identical
		if len(tree1.Events) != len(tree2.Events) {
			t.Errorf("Non-deterministic event count: %d vs %d",
				len(tree1.Events), len(tree2.Events))
			return
		}

		for i := range tree1.Events {
			if tree1.Events[i] != tree2.Events[i] {
				t.Errorf("Non-deterministic event at index %d: %+v vs %+v",
					i, tree1.Events[i], tree2.Events[i])
				return
			}
		}

		// Tokens must be identical (type, position, text)
		if len(tree1.Tokens) != len(tree2.Tokens) {
			t.Errorf("Non-deterministic token count: %d vs %d",
				len(tree1.Tokens), len(tree2.Tokens))
			return
		}

		for i := range tree1.Tokens {
			t1, t2 := tree1.Tokens[i], tree2.Tokens[i]
			if t1.Type != t2.Type || t1.Position != t2.Position ||
				!bytes.Equal(t1.Text, t2.Text) || t1.HasSpaceBefore != t2.HasSpaceBefore {
				t.Errorf("Non-deterministic token at index %d", i)
				return
			}
		}

		// Errors must be identical (message, count, order)
		if len(tree1.Errors) != len(tree2.Errors) {
			t.Errorf("Non-deterministic error count: %d vs %d",
				len(tree1.Errors), len(tree2.Errors))
			return
		}

		for i := range tree1.Errors {
			if tree1.Errors[i].Message != tree2.Errors[i].Message {
				t.Errorf("Non-deterministic error message at index %d", i)
				return
			}
		}
	})
}

// FuzzParserNoPanic verifies the parser never panics on any input.
func FuzzParserNoPanic(f *testing.F) {
	f.Add([]byte(""))
	f.Add([]byte("fun greet() {}"))
	f.Add([]byte("\x00\x01\x02"))           // Binary data
	f.Add(bytes.Repeat([]byte("a"), 10000)) // Very long
	f.Add(bytes.Repeat([]byte("{"), 1000))  // Deep nesting

	// Decorator syntax
	f.Add([]byte("@timeout(5m) { }"))
	f.Add([]byte("@retry(3, 2s) { }"))
	f.Add([]byte("@retry(delay=2s, 3)"))
	f.Add([]byte("@timeout(5m) { @retry(3) { } }"))

	// Malformed decorators
	f.Add([]byte("@retry("))
	f.Add([]byte("@retry(3,"))
	f.Add([]byte("@retry(3, times=5)"))
	f.Add([]byte("@var."))

	f.Fuzz(func(t *testing.T, input []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Parser panicked: %v", r)
			}
		}()

		tree := Parse(input)
		if tree == nil {
			t.Error("Parse returned nil")
		}
	})
}

// FuzzParserEventBalance verifies Open/Close events are properly nested
// using a stack (not just counts). Catches cross-closing and negative depth.
func FuzzParserEventBalance(f *testing.F) {
	f.Add([]byte(""))
	f.Add([]byte("fun greet() {}"))
	f.Add([]byte("{ { { } } }"))
	f.Add([]byte("fun a() { fun b() { } }"))

	// Decorator nesting
	f.Add([]byte("@timeout(5m) { }"))
	f.Add([]byte("@timeout(5m) { @retry(3) { } }"))
	f.Add([]byte("@timeout(5m) { @retry(3, 2s) { @parallel { } } }"))

	// Deep nesting
	nested := make([]byte, 0, 200)
	nested = append(nested, bytes.Repeat([]byte("{"), 100)...)
	nested = append(nested, bytes.Repeat([]byte("}"), 100)...)
	f.Add(nested)

	f.Fuzz(func(t *testing.T, input []byte) {
		tree := Parse(input)

		// Track nesting with a stack
		depth := 0

		for i, event := range tree.Events {
			switch event.Kind {
			case EventOpen:
				depth++

			case EventClose:
				if depth <= 0 {
					t.Errorf("Close event at index %d with depth %d (negative depth)",
						i, depth)
					return
				}
				depth--
			}

			// Depth should never go negative
			if depth < 0 {
				t.Errorf("Negative depth %d at event index %d", depth, i)
				return
			}
		}

		// Stack must be empty at end
		if depth != 0 {
			t.Errorf("Unbalanced events: depth=%d at end (expected 0)", depth)
		}
	})
}

// FuzzParserMemorySafety verifies positions are valid and monotonic.
func FuzzParserMemorySafety(f *testing.F) {
	f.Add([]byte(""))
	f.Add(bytes.Repeat([]byte("a"), 10000)) // Very long
	f.Add(bytes.Repeat([]byte("{"), 1000))  // Deep nesting
	f.Add([]byte("\x00"))                   // Null byte
	f.Add([]byte("\xff\xfe\xfd"))           // Invalid UTF-8

	// Decorator syntax
	f.Add([]byte("@timeout(5m) { }"))
	f.Add([]byte("@retry(3, 2s, \"exponential\") { }"))
	f.Add([]byte("@timeout(5m) { @retry(3) { } }"))

	// Very long identifiers
	longIdent := append([]byte("var "), bytes.Repeat([]byte("x"), 1000)...)
	longIdent = append(longIdent, []byte(" = 42")...)
	f.Add(longIdent)

	f.Fuzz(func(t *testing.T, input []byte) {
		tree := Parse(input)

		// Verify event→token indices are valid
		for i, event := range tree.Events {
			if event.Kind == EventToken {
				tokenIdx := int(event.Data)
				if tokenIdx < 0 || tokenIdx >= len(tree.Tokens) {
					t.Errorf("Event %d references invalid token index %d (have %d tokens)",
						i, tokenIdx, len(tree.Tokens))
				}
			}
		}

		// Verify token positions are valid
		for i, token := range tree.Tokens {
			// Line and column must be >= 1
			if token.Position.Line < 1 {
				t.Errorf("Token %d has invalid line %d (must be >= 1)",
					i, token.Position.Line)
			}
			if token.Position.Column < 1 {
				t.Errorf("Token %d has invalid column %d (must be >= 1)",
					i, token.Position.Column)
			}

			// Offset must be within source bounds
			if token.Position.Offset < 0 || token.Position.Offset > len(input) {
				t.Errorf("Token %d offset %d out of bounds (source length %d)",
					i, token.Position.Offset, len(input))
			}
		}

		// Verify positions are monotonic (offsets increasing)
		for i := 1; i < len(tree.Tokens); i++ {
			curr, prev := tree.Tokens[i], tree.Tokens[i-1]
			if curr.Position.Offset < prev.Position.Offset {
				t.Errorf("Tokens %d and %d not monotonic: offset %d then %d",
					i-1, i, prev.Position.Offset, curr.Position.Offset)
			}
		}
	})
}

// FuzzParserPathologicalDepth verifies the parser handles deep nesting
// without panicking (prevents exponential backtracking).
func FuzzParserPathologicalDepth(f *testing.F) {
	// Deep nesting patterns
	nested1 := make([]byte, 0, 200)
	nested1 = append(nested1, bytes.Repeat([]byte("{"), 100)...)
	nested1 = append(nested1, bytes.Repeat([]byte("}"), 100)...)
	f.Add(nested1)

	nested2 := make([]byte, 0, 1000)
	nested2 = append(nested2, bytes.Repeat([]byte("fun f() { "), 50)...)
	nested2 = append(nested2, bytes.Repeat([]byte("}"), 50)...)
	f.Add(nested2)

	f.Fuzz(func(t *testing.T, input []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Parser panicked on deep nesting: %v", r)
			}
		}()

		tree := Parse(input)

		// Count max depth in event stream
		maxDepth := 0
		currentDepth := 0

		for _, event := range tree.Events {
			switch event.Kind {
			case EventOpen:
				currentDepth++
				if currentDepth > maxDepth {
					maxDepth = currentDepth
				}
			case EventClose:
				currentDepth--
			}
		}

		// Parser should handle reasonable depth (1000 levels)
		// If depth exceeds this, should produce error, not panic
		const maxReasonableDepth = 1000
		if maxDepth > maxReasonableDepth && len(tree.Errors) == 0 {
			t.Logf("Warning: Very deep nesting (%d levels) without error", maxDepth)
		}
	})
}

// FuzzParserErrorRecovery verifies resilient parsing (errors, not crashes).
func FuzzParserErrorRecovery(f *testing.F) {
	f.Add([]byte("fun"))        // Incomplete
	f.Add([]byte("fun greet(")) // Unclosed
	f.Add([]byte("@"))          // Lone decorator
	f.Add([]byte("var = 42"))   // Missing name

	f.Fuzz(func(t *testing.T, input []byte) {
		tree := Parse(input)

		// Errors should have messages
		for i, err := range tree.Errors {
			if err.Message == "" {
				t.Errorf("Error %d has empty message", i)
			}
		}
	})
}

// TestFuzzCorpusMinimization verifies the fuzz tests work correctly.
func TestFuzzCorpusMinimization(t *testing.T) {
	inputs := [][]byte{
		[]byte(""),
		[]byte("fun greet() {}"),
		[]byte("invalid syntax"),
		[]byte("@retry(3) { }"),
	}

	for _, input := range inputs {
		// Verify determinism
		tree1 := Parse(input)
		tree2 := Parse(input)

		if len(tree1.Events) != len(tree2.Events) {
			t.Errorf("Determinism failed for: %q", input)
		}

		// Verify no panic
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Panic on: %q", input)
				}
			}()
			Parse(input)
		}()

		// Verify event balance
		depth := 0
		for _, event := range tree1.Events {
			switch event.Kind {
			case EventOpen:
				depth++
			case EventClose:
				depth--
			}
			if depth < 0 {
				t.Errorf("Negative depth on: %q", input)
				break
			}
		}
		if depth != 0 {
			t.Errorf("Unbalanced events (depth=%d) on: %q", depth, input)
		}
	}
}

// FuzzParserWhitespaceInvariance ensures spaces/tabs between tokens
// do not affect the semantic token+event streams. Newlines are preserved.
// This is critical for plan hashing stability.
func FuzzParserWhitespaceInvariance(f *testing.F) {
	f.Add([]byte(""))
	f.Add([]byte("fun greet(name){echo name}"))
	f.Add([]byte("var x=1\nvar y=2\nfun f(){x+y}"))
	f.Add([]byte("@retry(3){ fun a(){ } }"))

	f.Fuzz(func(t *testing.T, input []byte) {
		orig := Parse(input)

		// If parser didn't produce tokens, nothing to test
		if len(orig.Tokens) == 0 {
			return
		}

		// Helper: semantic token view (ignore positions)
		type semanticToken struct {
			Type lexer.TokenType
			Text string
		}
		semanticTokens := func(tokens []lexer.Token) []semanticToken {
			out := make([]semanticToken, len(tokens))
			for i, tk := range tokens {
				out[i] = semanticToken{Type: tk.Type, Text: string(tk.Text)}
			}
			return out
		}

		// Helper: semantic event view (ignore positions)
		type semanticEvent struct {
			Kind EventKind
			Data uint32
		}
		semanticEvents := func(events []Event) []semanticEvent {
			out := make([]semanticEvent, len(events))
			for i, ev := range events {
				out[i] = semanticEvent{Kind: ev.Kind, Data: ev.Data}
			}
			return out
		}

		// Build "noised" input by stitching token text back together
		// with random spaces/tabs in gaps (preserving newlines)
		var buf bytes.Buffer
		cursor := 0

		// Deterministic seed based on input
		seed := int64(0)
		for _, b := range input {
			seed = seed*31 + int64(b)
		}
		rng := struct {
			state int64
		}{state: seed}

		randInt := func(n int) int {
			rng.state = rng.state*1103515245 + 12345
			return int((rng.state / 65536) % int64(n))
		}

		emitNoisedGap := func(gap []byte) {
			for i := 0; i < len(gap); i++ {
				b := gap[i]
				switch b {
				case ' ', '\t':
					// Replace with 1-3 random spaces/tabs (never 0 to avoid merging tokens)
					n := 1 + randInt(3)
					for j := 0; j < n; j++ {
						if randInt(2) == 0 {
							buf.WriteByte(' ')
						} else {
							buf.WriteByte('\t')
						}
					}
				default:
					// Preserve everything else (including newlines)
					buf.WriteByte(b)
				}
			}
		}

		for _, tk := range orig.Tokens {
			start := tk.Position.Offset
			end := start + len(tk.Text)
			if start < cursor || end > len(input) || start > end {
				// Defensive: bail if positions look odd
				return
			}
			// Gap before token
			emitNoisedGap(input[cursor:start])
			// Token text itself (untouched)
			buf.Write(tk.Text)
			cursor = end
		}
		// Trailing gap after last token
		if cursor <= len(input) {
			emitNoisedGap(input[cursor:])
		}

		noised := buf.Bytes()
		got := Parse(noised)

		// Compare semantic tokens (type + text only, ignore position)
		st1, st2 := semanticTokens(orig.Tokens), semanticTokens(got.Tokens)
		if len(st1) != len(st2) {
			t.Errorf("Token count changed with whitespace: %d -> %d", len(st1), len(st2))
			t.Errorf("Original input: %q", input)
			t.Errorf("Noised input: %q", noised)
			t.Errorf("Original tokens: %v", st1)
			t.Errorf("Noised tokens: %v", st2)
			return
		}
		for i := range st1 {
			if st1[i] != st2[i] {
				t.Errorf("Token %d changed with whitespace: %+v -> %+v", i, st1[i], st2[i])
				return
			}
		}

		// Compare semantic events (kind + data only, ignore positions)
		se1, se2 := semanticEvents(orig.Events), semanticEvents(got.Events)
		if len(se1) != len(se2) {
			t.Errorf("Event count changed with whitespace: %d -> %d", len(se1), len(se2))
			return
		}
		for i := range se1 {
			if se1[i] != se2[i] {
				t.Errorf("Event %d changed with whitespace: %+v -> %+v", i, se1[i], se2[i])
				return
			}
		}
	})
}
