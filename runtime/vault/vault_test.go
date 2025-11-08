package vault

import (
	"testing"
)

// TestVault_PathTracking_SingleDecorator tests building a simple path
// with one decorator instance.
func TestVault_PathTracking_SingleDecorator(t *testing.T) {
	v := New()

	// GIVEN: We enter a step and a decorator
	v.EnterStep()
	v.EnterDecorator("@shell")

	// WHEN: We build the site path for a parameter
	path := v.BuildSitePath("command")

	// THEN: Path should include decorator with index [0]
	expected := "root/step-1/@shell[0]/params/command"
	if path != expected {
		t.Errorf("BuildSitePath() = %q, want %q", path, expected)
	}
}

// TestVault_PathTracking_MultipleInstances tests that multiple instances
// of the same decorator get different indices.
func TestVault_PathTracking_MultipleInstances(t *testing.T) {
	v := New()

	// GIVEN: Three shell commands in three steps
	v.EnterStep()
	v.EnterDecorator("@shell")
	path1 := v.BuildSitePath("command")
	v.ExitDecorator()

	v.EnterStep()
	v.EnterDecorator("@shell")
	path2 := v.BuildSitePath("command")
	v.ExitDecorator()

	v.EnterStep()
	v.EnterDecorator("@shell")
	path3 := v.BuildSitePath("command")
	v.ExitDecorator()

	// THEN: Each should have different step but same decorator index [0]
	expected := []string{
		"root/step-1/@shell[0]/params/command",
		"root/step-2/@shell[0]/params/command",
		"root/step-3/@shell[0]/params/command",
	}

	paths := []string{path1, path2, path3}
	for i, path := range paths {
		if path != expected[i] {
			t.Errorf("Path[%d] = %q, want %q", i, path, expected[i])
		}
	}
}

// TestVault_PathTracking_NestedDecorators tests building paths through
// nested decorator contexts.
func TestVault_PathTracking_NestedDecorators(t *testing.T) {
	v := New()

	// GIVEN: Nested decorators @retry -> @timeout -> @shell
	v.EnterDecorator("@retry")
	v.EnterDecorator("@timeout")
	v.EnterDecorator("@shell")

	// WHEN: We build the site path
	path := v.BuildSitePath("command")

	// THEN: Path should show full nesting
	expected := "root/@retry[0]/@timeout[0]/@shell[0]/params/command"
	if path != expected {
		t.Errorf("BuildSitePath() = %q, want %q", path, expected)
	}
}

// TestVault_PathTracking_MultipleDecoratorsAtSameLevel tests that
// different decorators at the same level get independent indices.
func TestVault_PathTracking_MultipleDecoratorsAtSameLevel(t *testing.T) {
	v := New()

	v.EnterStep()

	// First shell command
	v.EnterDecorator("@shell")
	path1 := v.BuildSitePath("command")
	v.ExitDecorator()

	// Second shell command
	v.EnterDecorator("@shell")
	path2 := v.BuildSitePath("command")
	v.ExitDecorator()

	// A retry decorator
	v.EnterDecorator("@retry")
	path3 := v.BuildSitePath("times")
	v.ExitDecorator()

	// THEN: Shell commands get [0] and [1], retry gets [0]
	expected := []string{
		"root/step-1/@shell[0]/params/command",
		"root/step-1/@shell[1]/params/command",
		"root/step-1/@retry[0]/params/times",
	}

	paths := []string{path1, path2, path3}
	for i, path := range paths {
		if path != expected[i] {
			t.Errorf("Path[%d] = %q, want %q", i, path, expected[i])
		}
	}
}

// TestVault_PathTracking_ResetCountsPerStep tests that decorator counts
// reset when entering a new step.
func TestVault_PathTracking_ResetCountsPerStep(t *testing.T) {
	v := New()

	// Step 1: Two shell commands
	v.EnterStep()
	v.EnterDecorator("@shell")
	v.ExitDecorator()
	v.EnterDecorator("@shell")
	path1 := v.BuildSitePath("command")
	v.ExitDecorator()

	// Step 2: One shell command (should be [0] again)
	v.EnterStep()
	v.EnterDecorator("@shell")
	path2 := v.BuildSitePath("command")
	v.ExitDecorator()

	// THEN: Step 1 has @shell[1], Step 2 has @shell[0]
	if path1 != "root/step-1/@shell[1]/params/command" {
		t.Errorf("Step 1 path = %q, want %q", path1, "root/step-1/@shell[1]/params/command")
	}
	if path2 != "root/step-2/@shell[0]/params/command" {
		t.Errorf("Step 2 path = %q, want %q", path2, "root/step-2/@shell[0]/params/command")
	}
}

// TestVault_DeclareVariable tests declaring and retrieving variables.
func TestVault_DeclareVariable(t *testing.T) {
	v := New()

	// GIVEN: We declare a variable
	v.DeclareVariable("API_KEY", "sk-secret-123")

	// WHEN: We check if it exists
	expr, exists := v.GetExpression("API_KEY")

	// THEN: Should be registered
	if !exists {
		t.Error("Variable should be registered")
	}
	if expr.Raw != "sk-secret-123" {
		t.Errorf("Expression.Raw = %q, want %q", expr.Raw, "sk-secret-123")
	}
	if expr.Type != ExprVariable {
		t.Errorf("Expression.Type = %v, want ExprVariable", expr.Type)
	}
}

// TestVault_TrackDecoratorCall tests tracking decorator call expressions.
func TestVault_TrackDecoratorCall(t *testing.T) {
	v := New()

	// GIVEN: We track a decorator call
	v.TrackExpression("secret1", "@aws.secret('prod-key')", ExprDecorator)

	// WHEN: We retrieve it
	expr, exists := v.GetExpression("secret1")

	// THEN: Should be registered as decorator type
	if !exists {
		t.Error("Expression should be registered")
	}
	if expr.Type != ExprDecorator {
		t.Errorf("Expression.Type = %v, want ExprDecorator", expr.Type)
	}
}

// TestVault_RecordExpressionReference tests recording expression references at sites.
func TestVault_RecordExpressionReference(t *testing.T) {
	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	// GIVEN: We have an expression and we're at a site
	v.DeclareVariable("API_KEY", "sk-secret")
	v.EnterStep()
	v.EnterDecorator("@shell")

	// WHEN: We record that this site uses the expression
	v.RecordExpressionReference("API_KEY", "command")

	// THEN: Should track the reference
	refs := v.GetReferences("API_KEY")
	if len(refs) != 1 {
		t.Fatalf("Expected 1 reference, got %d", len(refs))
	}

	if refs[0].Site != "root/step-1/@shell[0]/params/command" {
		t.Errorf("Reference.Site = %q, want %q", refs[0].Site, "root/step-1/@shell[0]/params/command")
	}
}

// TestVault_MultipleReferencesToSameExpression tests multiple references to the same expression.
func TestVault_MultipleReferencesToSameExpression(t *testing.T) {
	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	v.DeclareVariable("API_KEY", "sk-secret")

	// Site 1
	v.EnterStep()
	v.EnterDecorator("@shell")
	v.RecordExpressionReference("API_KEY", "command")
	v.ExitDecorator()

	// Site 2
	v.EnterStep()
	v.EnterDecorator("@shell")
	v.RecordExpressionReference("API_KEY", "command")
	v.ExitDecorator()

	// THEN: Should have 2 references
	refs := v.GetReferences("API_KEY")
	if len(refs) != 2 {
		t.Fatalf("Expected 2 references, got %d", len(refs))
	}

	// Different sites
	if refs[0].Site == refs[1].Site {
		t.Error("References should be at different sites")
	}

	// Different SiteIDs
	if refs[0].SiteID == refs[1].SiteID {
		t.Error("References should have different SiteIDs")
	}
}

// TestVault_UnusedExpression tests tracking unused expressions (no references).
func TestVault_UnusedExpression(t *testing.T) {
	v := New()

	// GIVEN: We declare a variable but never reference it
	v.DeclareVariable("UNUSED", "sk-old-key")

	// WHEN: We check references
	refs := v.GetReferences("UNUSED")

	// THEN: Should have no references
	if len(refs) != 0 {
		t.Errorf("Expected 0 references, got %d", len(refs))
	}
}

// ========== Phase 2B: Pruning Tests ==========

// TestVault_PruneUnusedExpressions tests removing expressions with no references.
func TestVault_PruneUnusedExpressions(t *testing.T) {
	v := New()

	// GIVEN: Two variables, one used, one unused
	v.DeclareVariable("USED", "sk-used")
	v.DeclareVariable("UNUSED", "sk-unused")

	v.EnterStep()
	v.EnterDecorator("@shell")
	v.RecordExpressionReference("USED", "command")

	// WHEN: We prune unused expressions
	v.PruneUnused()

	// THEN: Only USED should remain
	_, usedExists := v.GetExpression("USED")
	_, unusedExists := v.GetExpression("UNUSED")

	if !usedExists {
		t.Error("USED expression should still exist")
	}
	if unusedExists {
		t.Error("UNUSED expression should be pruned")
	}
}

// TestVault_BuildSecretUses tests building final SecretUse list.
// In our security model: ALL value decorators are secrets.
func TestVault_BuildSecretUses(t *testing.T) {
	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	// GIVEN: Variables with references (all are secrets in our model)
	v.DeclareVariable("API_KEY", "sk-secret")
	v.EnterStep()
	v.EnterDecorator("@shell")
	v.RecordExpressionReference("API_KEY", "command")
	v.ExitDecorator()

	v.EnterStep()
	v.EnterDecorator("@shell")
	v.RecordExpressionReference("API_KEY", "command")

	// Assign DisplayID (normally done during resolution)
	expr, _ := v.GetExpression("API_KEY")
	expr.DisplayID = "opal:v:ABC123"

	// WHEN: We build final SecretUses
	uses := v.BuildSecretUses()

	// THEN: Should have 2 SecretUse entries
	if len(uses) != 2 {
		t.Fatalf("Expected 2 SecretUses, got %d", len(uses))
	}

	// Both should have same DisplayID
	if uses[0].DisplayID != uses[1].DisplayID {
		t.Error("Same secret should have same DisplayID")
	}

	// Different SiteIDs
	if uses[0].SiteID == uses[1].SiteID {
		t.Error("Different sites should have different SiteIDs")
	}
}

// TestVault_BuildSecretUses_RequiresDisplayID tests that expressions without
// DisplayID are skipped (not yet resolved).
func TestVault_BuildSecretUses_RequiresDisplayID(t *testing.T) {
	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	// GIVEN: Variable with reference but no DisplayID (not resolved yet)
	v.DeclareVariable("UNRESOLVED", "value")

	v.EnterStep()
	v.EnterDecorator("@shell")
	v.RecordExpressionReference("UNRESOLVED", "command")

	// Don't assign DisplayID - simulates unresolved expression

	// WHEN: We build SecretUses
	uses := v.BuildSecretUses()

	// THEN: Should be empty (no DisplayID = not resolved = skip)
	if len(uses) != 0 {
		t.Fatalf("Expected 0 SecretUses (unresolved), got %d", len(uses))
	}
}

// TestVault_EndToEnd_PruneAndBuild tests complete workflow.
func TestVault_EndToEnd_PruneAndBuild(t *testing.T) {
	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	// GIVEN: Multiple variables, some used, some unused
	v.DeclareVariable("USED_SECRET", "sk-used")
	v.DeclareVariable("UNUSED_SECRET", "sk-unused")
	v.DeclareVariable("ANOTHER_USED", "value")

	// Only reference USED_SECRET and ANOTHER_USED
	v.EnterStep()
	v.EnterDecorator("@shell")
	v.RecordExpressionReference("USED_SECRET", "command")
	v.RecordExpressionReference("ANOTHER_USED", "command")

	// WHEN: We prune and build
	v.PruneUnused()

	// Assign DisplayIDs (normally done during resolution)
	if expr, exists := v.GetExpression("USED_SECRET"); exists {
		expr.DisplayID = "opal:v:SECRET1"
	}
	if expr, exists := v.GetExpression("ANOTHER_USED"); exists {
		expr.DisplayID = "opal:v:SECRET2"
	}

	uses := v.BuildSecretUses()

	// THEN: Should have 2 SecretUses (UNUSED_SECRET pruned)
	if len(uses) != 2 {
		t.Fatalf("Expected 2 SecretUses, got %d", len(uses))
	}

	// Verify UNUSED_SECRET was pruned
	if _, exists := v.GetExpression("UNUSED_SECRET"); exists {
		t.Error("UNUSED_SECRET should have been pruned")
	}
}
