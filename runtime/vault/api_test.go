package vault

import (
	"testing"

)

// ========== DeclareVariable Tests ==========

// TestVault_DeclareVariable_ReturnsVariableName tests that DeclareVariable returns the variable name as ID.
func TestVault_DeclareVariable_ReturnsVariableName(t *testing.T) {
	v := New()

	// WHEN: We declare a variable
	exprID := v.DeclareVariable("API_KEY", "@env.API_KEY")

	// THEN: Should return variable name as ID
	if exprID != "API_KEY" {
		t.Errorf("DeclareVariable() = %q, want %q", exprID, "API_KEY")
	}
}

// TestVault_DeclareVariable_StoresExpression tests that the expression is stored.
func TestVault_DeclareVariable_StoresExpression(t *testing.T) {
	v := New()

	// WHEN: We declare a variable
	exprID := v.DeclareVariable("API_KEY", "@env.API_KEY")

	// THEN: Expression should be stored (check via internal state)
	if v.expressions[exprID] == nil {
		t.Error("Expression should be stored")
	}
	if v.expressions[exprID].Raw != "@env.API_KEY" {
		t.Errorf("Expression.Raw = %q, want %q", v.expressions[exprID].Raw, "@env.API_KEY")
	}
}

// ========== TrackExpression Tests ==========

// TestVault_TrackExpression_ReturnsHashBasedID tests that TrackExpression returns hash-based ID.
func TestVault_TrackExpression_ReturnsHashBasedID(t *testing.T) {
	v := New()

	// WHEN: We track a direct decorator call
	exprID := v.TrackExpression("@env.HOME")

	// THEN: Should return hash-based ID with transport
	// Format: "transport:decorator:params:hash"
	if exprID == "" {
		t.Error("TrackExpression() should return non-empty ID")
	}
	if exprID == "@env.HOME" {
		t.Error("TrackExpression() should not return raw expression as ID")
	}
	// Should include transport prefix
	if len(exprID) < 6 || exprID[:6] != "local:" {
		t.Errorf("TrackExpression() = %q, should start with 'local:'", exprID)
	}
}

// TestVault_TrackExpression_IncludesTransport tests that expression ID includes transport context.
func TestVault_TrackExpression_IncludesTransport(t *testing.T) {
	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	// GIVEN: We're in local transport
	localID := v.TrackExpression("@env.HOME")

	// WHEN: We enter SSH transport
	v.EnterTransport("ssh:server1")
	sshID := v.TrackExpression("@env.HOME")

	// THEN: IDs should be different (different transport context)
	if localID == sshID {
		t.Errorf("Same expression in different transports should have different IDs: local=%q, ssh=%q", localID, sshID)
	}

	// Both should include their transport
	if len(localID) < 6 || localID[:6] != "local:" {
		t.Errorf("Local ID should start with 'local:', got %q", localID)
	}
	if len(sshID) < 12 || sshID[:12] != "ssh:server1:" {
		t.Errorf("SSH ID should start with 'ssh:server1:', got %q", sshID)
	}
}

// TestVault_TrackExpression_Deterministic tests that same expression returns same ID.
func TestVault_TrackExpression_Deterministic(t *testing.T) {
	v := New()

	// WHEN: We track the same expression twice
	id1 := v.TrackExpression("@env.HOME")
	id2 := v.TrackExpression("@env.HOME")

	// THEN: Should return same ID (deterministic)
	if id1 != id2 {
		t.Errorf("Same expression should return same ID: id1=%q, id2=%q", id1, id2)
	}
}

// ========== ResolveTouched Tests ==========

// TestVault_ResolveTouched_CallsDecorators tests that ResolveTouched calls decorators.
func TestVault_ResolveTouched_CallsDecorators(t *testing.T) {
	t.Skip("TODO: Implement after decorator integration")
	// This test requires decorator registry integration
	// Will implement in Phase 2D (Planner Integration)
}

// ========== Access Tests ==========

// TestVault_Access_ChecksSiteID tests that Access checks SiteID authorization.
func TestVault_Access_ChecksSiteID(t *testing.T) {
	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	// GIVEN: Variable with resolved value
	exprID := v.DeclareVariable("API_KEY", "@env.API_KEY")

	// Manually set Handle for testing (normally done by ResolveTouched)
	// TODO: Replace with proper resolution once ResolveTouched is implemented
	v.expressions[exprID].Value =("sk-secret-123")

	// Record authorized site
	v.EnterStep()
	v.EnterDecorator("@shell")
	v.RecordReference(exprID, "command")

	// WHEN: We try to unwrap at authorized site (current site)
	value, err := v.Access(exprID)

	// THEN: Should succeed
	if err != nil {
		t.Errorf("Access() at authorized site should succeed, got error: %v", err)
	}
	if value != "sk-secret-123" {
		t.Errorf("Access() = %q, want %q", value, "sk-secret-123")
	}
}

// TestVault_Access_RejectsUnauthorizedSite tests that Access rejects unauthorized sites.
func TestVault_Access_RejectsUnauthorizedSite(t *testing.T) {
	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	// GIVEN: Variable with resolved value
	exprID := v.DeclareVariable("API_KEY", "@env.API_KEY")
	v.expressions[exprID].Value =("sk-secret-123")

	// Record authorized site
	v.EnterStep()
	v.EnterDecorator("@shell")
	v.RecordReference(exprID, "command")
	v.ExitDecorator()

	// WHEN: We try to unwrap at different site (not authorized)
	v.EnterStep()
	v.EnterDecorator("@timeout")
	value, err := v.Access(exprID)

	// THEN: Should fail with authorization error
	if err == nil {
		t.Error("Access() at unauthorized site should fail")
	}
	if value != "" {
		t.Errorf("Access() should return empty string on error, got %q", value)
	}
	if err != nil && !containsString(err.Error(), "no authority") {
		t.Errorf("Error should mention 'no authority', got: %v", err)
	}
}

// TestVault_Access_ChecksTransportBoundary tests that Access checks transport boundaries.
func TestVault_Access_ChecksTransportBoundary(t *testing.T) {
	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	// GIVEN: Variable resolved in local transport
	exprID := v.DeclareVariable("LOCAL_TOKEN", "@env.TOKEN")
	v.expressions[exprID].Value =("secret-token")
	v.exprTransport[exprID] = "local"

	// Record use-site in local transport
	v.EnterStep()
	v.EnterDecorator("@shell")
	v.RecordReference(exprID, "command")
	v.ExitDecorator()

	// WHEN: We enter SSH transport and try to unwrap
	v.EnterTransport("ssh:untrusted")
	v.EnterStep()
	v.EnterDecorator("@shell")

	// Record reference in SSH transport (should fail at RecordReference, but test Access too)
	// Skip RecordReference for this test, just test Access

	value, err := v.Access(exprID)

	// THEN: Should fail with transport boundary error
	if err == nil {
		t.Error("Access() across transport boundary should fail")
	}
	if value != "" {
		t.Errorf("Access() should return empty string on error, got %q", value)
	}
	if err != nil && !containsString(err.Error(), "transport") {
		t.Errorf("Error should mention 'transport', got: %v", err)
	}
}

// TestVault_Access_UnresolvedExpression tests that Access fails for unresolved expressions.
func TestVault_Access_UnresolvedExpression(t *testing.T) {
	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	// GIVEN: Variable that hasn't been resolved yet
	exprID := v.DeclareVariable("UNRESOLVED", "@env.FOO")

	// Record use-site
	v.EnterStep()
	v.EnterDecorator("@shell")
	v.RecordReference(exprID, "command")

	// WHEN: We try to unwrap
	value, err := v.Access(exprID)

	// THEN: Should fail (not resolved yet)
	if err == nil {
		t.Error("Access() on unresolved expression should fail")
	}
	if value != "" {
		t.Errorf("Access() should return empty string on error, got %q", value)
	}
	if err != nil && !containsString(err.Error(), "not resolved") {
		t.Errorf("Error should mention 'not resolved', got: %v", err)
	}
}


