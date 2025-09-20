package lexer

import (
	"strings"
	"testing"

	"github.com/aledsdavies/devcmd/core/types"
)

func TestProcessorInterface(t *testing.T) {
	input := `var PORT = 8080
build: npm run build`

	// Test that both interfaces work
	legacyLexer := New(strings.NewReader(input))
	processor := NewProcessor(strings.NewReader(input))

	// Test NextToken consistency
	legacyTokens := legacyLexer.TokenizeToSlice()
	processorTokens := processor.AllTokens()

	if len(legacyTokens) != len(processorTokens) {
		t.Errorf("Token count mismatch: legacy %d vs processor %d", len(legacyTokens), len(processorTokens))
	}

	// Verify tokens are identical
	for i := 0; i < len(legacyTokens) && i < len(processorTokens); i++ {
		if legacyTokens[i].Type != processorTokens[i].Type {
			t.Errorf("Token %d type mismatch: legacy %s vs processor %s",
				i, legacyTokens[i].Type, processorTokens[i].Type)
		}
		if legacyTokens[i].Value != processorTokens[i].Value {
			t.Errorf("Token %d value mismatch: legacy %s vs processor %s",
				i, legacyTokens[i].Value, processorTokens[i].Value)
		}
	}
}

func TestProcessorFeatureFlag(t *testing.T) {
	input := `@var(PORT)`

	// Test with legacy (default)
	processor := NewProcessor(strings.NewReader(input))
	tokens := processor.AllTokens()

	// Should have at least @ VAR ( PORT ) tokens
	if len(tokens) < 4 {
		t.Errorf("Expected at least 4 tokens, got %d", len(tokens))
	}

	// First token should be @
	if tokens[0].Type != types.AT {
		t.Errorf("Expected first token to be AT, got %s", tokens[0].Type)
	}
}

func BenchmarkProcessorVsLegacy(b *testing.B) {
	input := `var PORT = 8080
var HOST = "localhost"
build: @timeout(30s) {
    npm run build
    npm run test
}
deploy: @parallel {
    docker build .
    kubectl apply -f k8s/
}`

	b.Run("Legacy", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			lexer := New(strings.NewReader(input))
			tokens := lexer.TokenizeToSlice()
			_ = tokens
		}
	})

	b.Run("Processor", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			processor := NewProcessor(strings.NewReader(input))
			tokens := processor.AllTokens()
			_ = tokens
		}
	})
}
