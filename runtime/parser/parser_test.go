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

// TDD Iteration 1, Step 2: Empty file should produce a File node

func TestEmptyFileProducesFileNode(t *testing.T) {
	input := ""

	tree := ParseString(input)

	// Should have exactly 2 events: Open(File) and Close
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
