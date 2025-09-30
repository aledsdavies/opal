package parser

import (
	"testing"
)

// TDD Iteration 1, Step 1: Can we parse an empty file?

func TestParseEmptyFile(t *testing.T) {
	input := ""

	tree := ParseString(input)

	if tree == nil {
		t.Fatal("Parse() returned nil for empty input")
	}
}

// TDD Iteration 1, Step 2: Empty source should produce a Source node

func TestEmptyFileProducesFileNode(t *testing.T) {
	input := ""

	tree := ParseString(input)

	// Should have exactly 2 events: Open(Source) and Close
	if len(tree.Events) != 2 {
		t.Fatalf("Expected 2 events, got %d", len(tree.Events))
	}

	// First event should be Open
	if tree.Events[0].Kind != EventOpen {
		t.Errorf("Expected first event to be Open, got %v", tree.Events[0].Kind)
	}

	// Second event should be Close
	if tree.Events[1].Kind != EventClose {
		t.Errorf("Expected second event to be Close, got %v", tree.Events[1].Kind)
	}
}

// TDD Iteration 1, Step 3: Parse simple function declaration

func TestParseFunctionDeclaration(t *testing.T) {
	input := "fun greet() {}"

	tree := ParseString(input)

	// Should have no errors
	if len(tree.Errors) != 0 {
		t.Errorf("Expected no errors, got: %v", tree.Errors)
	}

	// Should have tokens from lexer
	if len(tree.Tokens) == 0 {
		t.Fatal("Expected tokens from lexer")
	}

	// Should have events (we'll verify structure next)
	if len(tree.Events) == 0 {
		t.Fatal("Expected events in parse tree")
	}
}

// TDD Iteration 1, Step 4: Function should produce correct event structure

func TestFunctionEventStructure(t *testing.T) {
	input := "fun greet() {}"

	tree := ParseString(input)

	// Expected event structure:
	// Open(Source)
	//   Open(Function)
	//     Token(FUN)
	//     Token(IDENTIFIER)
	//     Open(ParamList)
	//       Token(LPAREN)
	//       Token(RPAREN)
	//     Close(ParamList)
	//     Open(Block)
	//       Token(LBRACE)
	//       Token(RBRACE)
	//     Close(Block)
	//   Close(Function)
	// Close(Source)

	expectedEvents := []struct {
		kind EventKind
		data uint32 // NodeKind for Open/Close, token index for Token
	}{
		{EventOpen, 0},  // Source
		{EventOpen, 1},  // Function
		{EventToken, 0}, // fun
		{EventToken, 1}, // greet
		{EventOpen, 2},  // ParamList
		{EventToken, 2}, // (
		{EventToken, 3}, // )
		{EventClose, 2}, // ParamList
		{EventOpen, 3},  // Block
		{EventToken, 4}, // {
		{EventToken, 5}, // }
		{EventClose, 3}, // Block
		{EventClose, 1}, // Function
		{EventClose, 0}, // Source
	}

	if len(tree.Events) != len(expectedEvents) {
		t.Fatalf("Expected %d events, got %d", len(expectedEvents), len(tree.Events))
	}

	for i, expected := range expectedEvents {
		if tree.Events[i].Kind != expected.kind {
			t.Errorf("Event %d: expected kind %v, got %v", i, expected.kind, tree.Events[i].Kind)
		}
		if tree.Events[i].Data != expected.data {
			t.Errorf("Event %d: expected data %v, got %v", i, expected.data, tree.Events[i].Data)
		}
	}
}
