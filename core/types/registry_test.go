package types

import (
	"testing"
)

func TestRegisterDecorator(t *testing.T) {
	// Create a fresh registry for testing
	r := NewRegistry()

	// Register a simple decorator
	r.Register("var")

	// Verify it's registered
	if !r.IsRegistered("var") {
		t.Error("decorator 'var' should be registered")
	}
}

func TestRegisterMultipleDecorators(t *testing.T) {
	r := NewRegistry()

	r.Register("var")
	r.Register("env")

	if !r.IsRegistered("var") {
		t.Error("decorator 'var' should be registered")
	}

	if !r.IsRegistered("env") {
		t.Error("decorator 'env' should be registered")
	}
}

func TestUnregisteredDecorator(t *testing.T) {
	r := NewRegistry()

	// Lookup non-existent decorator
	if r.IsRegistered("unknown") {
		t.Error("IsRegistered should return false for unregistered decorator")
	}
}

func TestGlobalRegistry(t *testing.T) {
	// Global registry should exist
	g := Global()
	if g == nil {
		t.Fatal("Global() should return a registry")
	}

	// Global registry starts empty (built-ins register from their own packages)
	// This test just verifies the global registry exists and works
}
