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

	// Control flow - valid if statements
	f.Add([]byte("fun test { if true { } }"))
	f.Add([]byte("fun test { if false { echo \"a\" } }"))
	f.Add([]byte("fun test { if x { } else { } }"))
	f.Add([]byte("fun test { if true { } else if false { } }"))
	f.Add([]byte("fun test { if @var.x { } }"))
	f.Add([]byte("if true { }"))                  // Top-level if (script mode)
	f.Add([]byte("if x { } else { echo \"a\" }")) // Top-level if-else

	// For loops - valid
	f.Add([]byte("fun test { for item in items { } }"))
	f.Add([]byte("fun test { for x in @var.list { echo @var.x } }"))
	f.Add([]byte("for item in items { }")) // Top-level for (script mode)

	// Control flow - malformed (error recovery)
	f.Add([]byte("fun test { if }"))
	f.Add([]byte("fun test { if { } }"))
	f.Add([]byte("fun test { if true }"))
	f.Add([]byte("fun test { if true { } else }"))
	f.Add([]byte("fun test { else { } }"))
	f.Add([]byte("fun test { if \"str\" { } }"))
	f.Add([]byte("fun test { if 42 { } }"))
	f.Add([]byte("fun test { if true { fun helper() { } } }"))      // fun inside if
	f.Add([]byte("fun test { for }"))                               // Incomplete for
	f.Add([]byte("fun test { for item }"))                          // Missing in
	f.Add([]byte("fun test { for item in }"))                       // Missing collection
	f.Add([]byte("fun test { for item in items }"))                 // Missing block
	f.Add([]byte("fun test { for item in items { fun h() { } } }")) // fun inside for

	// Try/catch/finally - valid
	f.Add([]byte("fun test { try { echo \"a\" } catch { echo \"b\" } }"))
	f.Add([]byte("fun test { try { echo \"a\" } finally { echo \"c\" } }"))
	f.Add([]byte("fun test { try { echo \"a\" } catch { echo \"b\" } finally { echo \"c\" } }"))
	f.Add([]byte("fun test { try { echo \"a\" } }"))                  // try only (catch/finally optional)
	f.Add([]byte("try { kubectl apply } catch { kubectl rollback }")) // Top-level try

	// Try/catch/finally - malformed (error recovery)
	f.Add([]byte("fun test { try }"))                             // Missing try block
	f.Add([]byte("fun test { try catch { } }"))                   // Missing try block
	f.Add([]byte("fun test { try { } catch }"))                   // Missing catch block
	f.Add([]byte("fun test { try { } finally }"))                 // Missing finally block
	f.Add([]byte("fun test { try { fun h() { } } }"))             // fun inside try
	f.Add([]byte("fun test { try { } catch { fun h() { } } }"))   // fun inside catch
	f.Add([]byte("fun test { try { } finally { fun h() { } } }")) // fun inside finally
	f.Add([]byte("fun test { catch { } }"))                       // orphan catch
	f.Add([]byte("fun test { finally { } }"))                     // orphan finally
	f.Add([]byte("catch { }"))                                    // orphan catch at top level
	f.Add([]byte("finally { }"))                                  // orphan finally at top level

	// Edge cases
	f.Add([]byte("fun"))                   // Incomplete
	f.Add([]byte("{}"))                    // Just braces
	f.Add([]byte("\"unterminated string")) // Unterminated

	// UTF-8 and line endings
	f.Add([]byte("fun test() {\r\n  echo \"hi\"\r\n}")) // CRLF
	f.Add([]byte("fun test { if\ntrue\n{\n}\n}"))       // If with newlines
	f.Add([]byte("fun test{if true{}}"))                // If no spaces
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
			e1, e2 := tree1.Errors[i], tree2.Errors[i]
			if e1.Message != e2.Message ||
				e1.Position.Line != e2.Position.Line ||
				e1.Position.Column != e2.Position.Column ||
				e1.Position.Offset != e2.Position.Offset {
				t.Errorf("Non-deterministic error[%d]: %+v vs %+v", i, e1, e2)
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

	// Control flow - if statements
	f.Add([]byte("fun test { if true { } }"))
	f.Add([]byte("fun test { if x { } else { } }"))
	f.Add([]byte("if true { }"))                               // Top-level (script mode)
	f.Add([]byte("fun test { if }"))                           // Malformed
	f.Add([]byte("fun test { if { } }"))                       // Missing condition
	f.Add([]byte("fun test { if \"str\" { } }"))               // Type error
	f.Add([]byte("fun test { if true { fun helper() { } } }")) // fun inside if

	f.Fuzz(func(t *testing.T, input []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Parser panicked: %v", r)
			}
		}()

		tree := Parse(input)
		if tree == nil {
			t.Error("Parse returned nil")
			return
		}

		// Growth cap: catch quadratic explosions
		// Generous heuristic: 10x input size + 1KB overhead
		maxStructures := 10*len(input) + 1024
		actualStructures := len(tree.Events) + len(tree.Tokens)
		if actualStructures > maxStructures {
			t.Errorf("Structure blow-up: %d events+tokens > %d (10x input + 1KB)",
				actualStructures, maxStructures)
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

	// Control flow nesting
	f.Add([]byte("fun test { if true { if false { } } }"))
	f.Add([]byte("if true { if false { } }")) // Top-level
	f.Add([]byte("fun test { if { } }"))      // Malformed

	// Deep nesting
	nested := make([]byte, 0, 200)
	nested = append(nested, bytes.Repeat([]byte("{"), 100)...)
	nested = append(nested, bytes.Repeat([]byte("}"), 100)...)
	f.Add(nested)

	f.Fuzz(func(t *testing.T, input []byte) {
		tree := Parse(input)

		// Track nesting with a type-aware stack
		// Catches cross-closing: Open(Function) ... Close(Block)
		var stack []NodeKind

		for i, event := range tree.Events {
			switch event.Kind {
			case EventOpen:
				nodeType := NodeKind(event.Data)
				stack = append(stack, nodeType)

			case EventClose:
				if len(stack) == 0 {
					t.Errorf("Close event at index %d with empty stack", i)
					return
				}
				// Pop and verify matching type
				openType := stack[len(stack)-1]
				closeType := NodeKind(event.Data)
				if openType != closeType {
					t.Errorf("Type mismatch at event %d: Open(%v) closed by Close(%v)",
						i, openType, closeType)
					return
				}
				stack = stack[:len(stack)-1]

			case EventToken:
				// Token events don't affect nesting
			}
		}

		// Stack must be empty at end
		if len(stack) != 0 {
			t.Errorf("Unclosed constructs: %d nodes remain on stack", len(stack))
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

	// Control flow
	f.Add([]byte("fun test { if true { } }"))
	f.Add([]byte("if true { }"))
	f.Add([]byte("fun test { if { } }"))

	// Unicode edge cases
	f.Add([]byte("\xEF\xBB\xBFfun a(){}"))   // UTF-8 BOM
	f.Add([]byte("fun\u200Bz(){}"))          // ZWSP
	f.Add([]byte("x\u00A0=\u00A01"))         // NBSP
	f.Add([]byte("line1\rline2\nline3\r\n")) // Mixed EOLs

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

		// Verify line/column coherence (line never decreases, column resets after newline)
		if len(tree.Tokens) > 0 {
			last := tree.Tokens[0].Position
			for i := 1; i < len(tree.Tokens); i++ {
				p := tree.Tokens[i].Position
				// Offset must increase
				if p.Offset < last.Offset {
					t.Errorf("Non-monotonic offset at token %d: %d -> %d", i, last.Offset, p.Offset)
				}
				// Line must not decrease
				if p.Line < last.Line {
					t.Errorf("Line decreased at token %d: %d -> %d", i, last.Line, p.Line)
				}
				// If same line, column must not decrease
				if p.Line == last.Line && p.Column < last.Column {
					t.Errorf("Column decreased at token %d: line %d, col %d -> %d",
						i, p.Line, last.Column, p.Column)
				}
				last = p
			}
		}

		// Verify token reference monotonicity in events
		// EventToken indices should be non-decreasing (events reference tokens in order)
		lastTok := -1
		for i, ev := range tree.Events {
			if ev.Kind == EventToken {
				idx := int(ev.Data)
				if idx < lastTok {
					t.Errorf("Token refs not non-decreasing at event %d: %d -> %d", i, lastTok, idx)
				}
				lastTok = idx
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

	// Deep if nesting
	nested3 := make([]byte, 0, 1000)
	nested3 = append(nested3, []byte("fun test { ")...)
	nested3 = append(nested3, bytes.Repeat([]byte("if true { "), 50)...)
	nested3 = append(nested3, bytes.Repeat([]byte("} "), 50)...)
	nested3 = append(nested3, []byte("}")...)
	f.Add(nested3)

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

	// Control flow error recovery
	f.Add([]byte("fun test { if }"))
	f.Add([]byte("fun test { if { } }"))
	f.Add([]byte("fun test { else { } }"))
	f.Add([]byte("fun test { if \"str\" { } }"))
	f.Add([]byte("fun test { if true { fun helper() { } } }"))

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
//
// The test reconstructs input by preserving HasSpaceBefore flags (token boundaries)
// while varying the amount/type of whitespace. This ensures:
// - Tokens don't merge (+ + stays separate, doesn't become ++)
// - Amount of whitespace doesn't matter (1 space vs 10 spaces)
// - Newlines are preserved (they're semantic in Opal)
func FuzzParserWhitespaceInvariance(f *testing.F) {
	f.Add([]byte(""))
	f.Add([]byte("fun greet(name){echo name}"))
	f.Add([]byte("var x=1\nvar y=2\nfun f(){x+y}"))
	f.Add([]byte("@retry(3){ fun a(){ } }"))

	// Control flow whitespace variations
	f.Add([]byte("fun test{if true{}}"))
	f.Add([]byte("fun test { if true { } else { } }"))
	f.Add([]byte("if true{echo \"a\"}"))

	// Unicode edge cases
	f.Add([]byte("\xEF\xBB\xBFfun a(){}")) // UTF-8 BOM
	f.Add([]byte("fun\u200Bz(){}"))        // ZWSP
	f.Add([]byte("x\u00A0=\u00A01"))       // NBSP

	f.Fuzz(func(t *testing.T, input []byte) {
		orig := Parse(input)

		// If parser didn't produce tokens, nothing to test
		if len(orig.Tokens) == 0 {
			return
		}

		// Skip inputs that are mostly ILLEGAL tokens (invalid syntax)
		// Whitespace changes can alter tokenization of invalid syntax
		// Example: "& & &" (3 ILLEGAL) → "& &&" (1 ILLEGAL + 1 AND_AND) when spaces removed
		illegalCount := 0
		for _, tk := range orig.Tokens {
			if tk.Type == lexer.ILLEGAL {
				illegalCount++
			}
		}
		// If half or more of the non-EOF tokens are ILLEGAL, skip
		// (whitespace changes can alter tokenization of invalid syntax)
		nonEOF := len(orig.Tokens)
		if nonEOF > 0 && orig.Tokens[nonEOF-1].Type == lexer.EOF {
			nonEOF--
		}
		if nonEOF > 0 && illegalCount*2 >= nonEOF {
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

		// Reconstruct input by trusting HasSpaceBefore flags
		// Key insight: HasSpaceBefore already tells us if there was whitespace
		// We just vary the amount/type while preserving token boundaries
		var buf bytes.Buffer

		// Deterministic RNG based on input
		seed := int64(0)
		for _, b := range input {
			seed = seed*31 + int64(b)
		}
		rng := struct{ state int64 }{state: seed}
		randInt := func(n int) int {
			rng.state = rng.state*1103515245 + 12345
			val := (rng.state / 65536) % int64(n)
			if val < 0 {
				val = -val
			}
			return int(val)
		}

		// Helper: get token text (from tk.Text or infer from type)
		getTokenText := func(tk lexer.Token) []byte {
			if len(tk.Text) > 0 {
				return tk.Text
			}
			// Token has nil Text - infer from type
			switch tk.Type {
			case lexer.LPAREN:
				return []byte("(")
			case lexer.RPAREN:
				return []byte(")")
			case lexer.LBRACE:
				return []byte("{")
			case lexer.RBRACE:
				return []byte("}")
			case lexer.LSQUARE:
				return []byte("[")
			case lexer.RSQUARE:
				return []byte("]")
			case lexer.COMMA:
				return []byte(",")
			case lexer.COLON:
				return []byte(":")
			case lexer.DOT:
				return []byte(".")
			case lexer.AT:
				return []byte("@")
			case lexer.SEMICOLON:
				return []byte(";")
			case lexer.PLUS:
				return []byte("+")
			case lexer.MINUS:
				return []byte("-")
			case lexer.MULTIPLY:
				return []byte("*")
			case lexer.DIVIDE:
				return []byte("/")
			case lexer.MODULO:
				return []byte("%")
			case lexer.LT:
				return []byte("<")
			case lexer.GT:
				return []byte(">")
			case lexer.NOT:
				return []byte("!")
			case lexer.EQUALS:
				return []byte("=")
			case lexer.PIPE:
				return []byte("|")
			case lexer.EQ_EQ:
				return []byte("==")
			case lexer.NOT_EQ:
				return []byte("!=")
			case lexer.LT_EQ:
				return []byte("<=")
			case lexer.GT_EQ:
				return []byte(">=")
			case lexer.AND_AND:
				return []byte("&&")
			case lexer.OR_OR:
				return []byte("||")
			case lexer.INCREMENT:
				return []byte("++")
			case lexer.DECREMENT:
				return []byte("--")
			case lexer.PLUS_ASSIGN:
				return []byte("+=")
			case lexer.MINUS_ASSIGN:
				return []byte("-=")
			case lexer.MULTIPLY_ASSIGN:
				return []byte("*=")
			case lexer.DIVIDE_ASSIGN:
				return []byte("/=")
			case lexer.MODULO_ASSIGN:
				return []byte("%=")
			case lexer.ARROW:
				return []byte("->")
			case lexer.FUN:
				return []byte("fun")
			case lexer.VAR:
				return []byte("var")
			case lexer.FOR:
				return []byte("for")
			case lexer.IN:
				return []byte("in")
			case lexer.IF:
				return []byte("if")
			case lexer.ELSE:
				return []byte("else")
			case lexer.WHEN:
				return []byte("when")
			case lexer.TRY:
				return []byte("try")
			case lexer.CATCH:
				return []byte("catch")
			case lexer.FINALLY:
				return []byte("finally")
			case lexer.NEWLINE:
				return []byte("\n")
			case lexer.COMMENT:
				// Comments have text in tk.Text, handled specially
				return nil
			case lexer.ILLEGAL:
				// ILLEGAL tokens have text in tk.Text
				return nil
			default:
				return nil
			}
		}

		// Track cursor in original input to extract newlines
		cursor := 0

		for _, tk := range orig.Tokens {
			// Extract newlines from gap before this token
			if tk.Position.Offset > cursor {
				gap := input[cursor:tk.Position.Offset]
				for _, b := range gap {
					if b == '\n' || b == '\r' {
						buf.WriteByte(b)
					}
				}
			}

			// If token had whitespace before it, emit random amount
			if tk.HasSpaceBefore {
				n := 1 + randInt(3) // 1-3 spaces/tabs
				for j := 0; j < n; j++ {
					if randInt(2) == 0 {
						buf.WriteByte(' ')
					} else {
						buf.WriteByte('\t')
					}
				}
			}

			// Write token text
			if tk.Type == lexer.COMMENT {
				// Comments need full reconstruction: /* content */ or // content
				// Check source to determine comment type
				offset := tk.Position.Offset
				if offset+1 < len(input) && input[offset] == '/' && input[offset+1] == '/' {
					// Line comment: // + content
					buf.WriteString("//")
					buf.Write(tk.Text)
					cursor = offset + 2 + len(tk.Text)
				} else if offset+1 < len(input) && input[offset] == '/' && input[offset+1] == '*' {
					// Block comment: /* + content + */ (if terminated)
					buf.WriteString("/*")
					buf.Write(tk.Text)
					// Check if terminated by looking at source
					terminated := false
					if len(input) >= offset+2+len(tk.Text)+2 {
						checkPos := offset + 2 + len(tk.Text)
						if checkPos+1 < len(input) && input[checkPos] == '*' && input[checkPos+1] == '/' {
							terminated = true
						}
					}
					if terminated {
						buf.WriteString("*/")
						cursor = offset + 2 + len(tk.Text) + 2
					} else {
						cursor = offset + 2 + len(tk.Text)
					}
				}
			} else if len(tk.Text) > 0 {
				// Token has explicit text (identifiers, strings, numbers)
				buf.Write(tk.Text)
				cursor = tk.Position.Offset + len(tk.Text)
			} else {
				// Token has nil Text (operators, keywords) - get from type
				tokenText := getTokenText(tk)
				if tokenText != nil {
					buf.Write(tokenText)
					cursor = tk.Position.Offset + len(tokenText)
				}
			}
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
