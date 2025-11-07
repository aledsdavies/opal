package planner

import (
	"errors"
	"strings"
	"testing"
)

func TestScopeGraphBasics(t *testing.T) {
	g := NewScopeGraph("local")

	// Store variable in root scope
	g.Store("HOME", "literal", "/home/alice", VarClassData, VarTaintAgnostic)

	// Resolve from same scope
	val, scope, err := g.Resolve("HOME")
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}
	if val != "/home/alice" {
		t.Errorf("Expected /home/alice, got %v", val)
	}
	if scope.sessionID != "local" {
		t.Errorf("Expected scope local, got %s", scope.sessionID)
	}
}

func TestScopeGraphTraversal(t *testing.T) {
	g := NewScopeGraph("local")

	// Store in root scope
	g.Store("HOME", "literal", "/home/alice", VarClassData, VarTaintAgnostic)

	// Enter child scope (non-transport)
	g.EnterScope("retry", false)

	// Should find variable in parent
	val, scope, err := g.Resolve("HOME")
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}
	if val != "/home/alice" {
		t.Errorf("Expected /home/alice, got %v", val)
	}
	if scope.sessionID != "local" {
		t.Errorf("Expected scope local, got %s", scope.sessionID)
	}
}

func TestScopeGraphSiblingIsolation(t *testing.T) {
	g := NewScopeGraph("local")

	// Enter first child
	g.EnterScope("ssh:server1", true)
	g.Store("REMOTE_HOME", "@env.HOME", "/home/bob", VarClassData, VarTaintAgnostic)

	// Exit and enter second child
	g.ExitScope()
	g.EnterScope("ssh:server2", true)

	// Should NOT find sibling's variable (will get transport boundary error
	// because it tries to look in parent, but variable isn't there)
	_, _, err := g.Resolve("REMOTE_HOME")
	if err == nil {
		t.Fatal("Expected error for sibling variable, got nil")
	}
	// Error can be either "not found" or "transport boundary" depending on
	// whether it checks parent scope first
}

func TestScopeGraphNesting(t *testing.T) {
	g := NewScopeGraph("local")

	// Root scope
	g.Store("LOCAL", "literal", "local-value", VarClassData, VarTaintAgnostic)

	// First level (SSH)
	g.EnterScope("ssh:server", true)
	g.Store("SSH_VAR", "literal", "ssh-value", VarClassData, VarTaintAgnostic)

	// Second level (Docker inside SSH)
	g.EnterScope("docker:container", true)
	g.Store("DOCKER_VAR", "literal", "docker-value", VarClassData, VarTaintAgnostic)

	// Should find current scope variable
	val, _, err := g.Resolve("DOCKER_VAR")
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}
	if val != "docker-value" {
		t.Errorf("Expected docker-value, got %v", val)
	}

	// Check transport depth
	if g.TransportDepth() != 2 {
		t.Errorf("Expected transport depth 2, got %d", g.TransportDepth())
	}
}

func TestScopeGraphExitRoot(t *testing.T) {
	g := NewScopeGraph("local")

	// Try to exit root scope
	err := g.ExitScope()
	if err == nil {
		t.Fatal("Expected error when exiting root scope")
	}
	if !strings.Contains(err.Error(), "cannot exit root scope") {
		t.Errorf("Expected 'cannot exit root scope' error, got: %v", err)
	}
}

func TestTransportBoundarySealing(t *testing.T) {
	g := NewScopeGraph("local")

	// Store secret in local scope
	g.Store("SECRET", "@env.API_KEY", "secret-value", VarClassSecret, VarTaintLocalOnly)

	// Enter transport boundary (SSH)
	g.EnterScope("ssh:server", true)

	// Should be sealed
	if !g.IsSealed() {
		t.Error("Expected scope to be sealed at transport boundary")
	}

	// Try to access parent variable without import
	_, _, err := g.Resolve("SECRET")
	if err == nil {
		t.Fatal("Expected TransportBoundaryError, got nil")
	}

	var boundaryErr *TransportBoundaryError
	if !errors.As(err, &boundaryErr) {
		t.Fatalf("Expected TransportBoundaryError, got %T: %v", err, err)
	}

	// Verify error details
	if boundaryErr.VarName != "SECRET" {
		t.Errorf("Expected VarName=SECRET, got %s", boundaryErr.VarName)
	}
	if boundaryErr.ParentScope != "local" {
		t.Errorf("Expected ParentScope=local, got %s", boundaryErr.ParentScope)
	}
	if boundaryErr.CurrentScope != "ssh:server" {
		t.Errorf("Expected CurrentScope=ssh:server, got %s", boundaryErr.CurrentScope)
	}
}

func TestTransportBoundaryWithImport(t *testing.T) {
	g := NewScopeGraph("local")

	// Store variable in local scope
	g.Store("CONFIG", "literal", "config-value", VarClassConfig, VarTaintAgnostic)

	// Enter transport boundary
	g.EnterScope("ssh:server", true)

	// Import the variable
	err := g.Import("CONFIG")
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Now should be able to access it
	val, _, err := g.Resolve("CONFIG")
	if err != nil {
		t.Fatalf("Expected success after import, got error: %v", err)
	}
	if val != "config-value" {
		t.Errorf("Expected config-value, got %v", val)
	}
}

func TestNonTransportBoundaryNotSealed(t *testing.T) {
	g := NewScopeGraph("local")

	// Store variable in root
	g.Store("VAR", "literal", "value", VarClassData, VarTaintAgnostic)

	// Enter non-transport scope (e.g., @retry)
	g.EnterScope("retry", false)

	// Should NOT be sealed
	if g.IsSealed() {
		t.Error("Expected scope NOT to be sealed for non-transport boundary")
	}

	// Should be able to access parent without import
	val, _, err := g.Resolve("VAR")
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}
	if val != "value" {
		t.Errorf("Expected value, got %v", val)
	}
}

func TestImportNonexistentVariable(t *testing.T) {
	g := NewScopeGraph("local")

	// Enter child scope
	g.EnterScope("ssh:server", true)

	// Try to import nonexistent variable
	err := g.Import("NONEXISTENT")
	if err == nil {
		t.Fatal("Expected error when importing nonexistent variable")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}
}

func TestGetEntry(t *testing.T) {
	g := NewScopeGraph("local")

	// Store variable with metadata
	g.Store("SECRET", "@env.API_KEY", "secret-value", VarClassSecret, VarTaintLocalOnly)

	// Get entry
	entry, err := g.GetEntry("SECRET")
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}

	// Verify metadata
	if entry.Value != "secret-value" {
		t.Errorf("Expected secret-value, got %v", entry.Value)
	}
	if entry.Origin != "@env.API_KEY" {
		t.Errorf("Expected @env.API_KEY, got %s", entry.Origin)
	}
	if entry.Class != VarClassSecret {
		t.Errorf("Expected VarClassSecret, got %d", entry.Class)
	}
	if entry.Taint != VarTaintLocalOnly {
		t.Errorf("Expected VarTaintLocalOnly, got %d", entry.Taint)
	}
}

func TestScopePath(t *testing.T) {
	g := NewScopeGraph("local")

	// Check root path
	path := g.ScopePath()
	if len(path) != 1 || path[0] != "local" {
		t.Errorf("Expected [local], got %v", path)
	}

	// Enter child
	g.EnterScope("ssh:server", true)
	path = g.ScopePath()
	if len(path) != 2 || path[0] != "local" || path[1] != "ssh:server" {
		t.Errorf("Expected [local ssh:server], got %v", path)
	}

	// Enter grandchild
	g.EnterScope("docker:container", true)
	path = g.ScopePath()
	if len(path) != 3 {
		t.Errorf("Expected 3 elements, got %v", path)
	}
}

func TestDebugPrint(t *testing.T) {
	g := NewScopeGraph("local")
	g.Store("HOME", "literal", "/home/alice", VarClassData, VarTaintAgnostic)

	g.EnterScope("ssh:server", true)
	g.Import("HOME")
	g.Store("REMOTE", "@env.HOME", "/home/bob", VarClassData, VarTaintAgnostic)

	output := g.DebugPrint()

	// Check that output contains expected elements
	if !strings.Contains(output, "root") {
		t.Error("Expected output to contain 'root'")
	}
	if !strings.Contains(output, "local") {
		t.Error("Expected output to contain 'local'")
	}
	if !strings.Contains(output, "ssh:server") {
		t.Error("Expected output to contain 'ssh:server'")
	}
	if !strings.Contains(output, "[SEALED]") {
		t.Error("Expected output to contain '[SEALED]'")
	}
	if !strings.Contains(output, "HOME") {
		t.Error("Expected output to contain 'HOME'")
	}
}
