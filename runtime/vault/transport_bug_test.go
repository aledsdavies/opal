package vault

import (
	"strings"
	"testing"
)

// ========== Bug Reproduction Tests ==========
// These tests reproduce the three transport boundary bugs:
// 1. Duplicate transport boundary check in Access()
// 2. Lazy initialization captures transport at first reference (not resolution)
// 3. Missing MarkResolved() method to set transport at resolution time

// ========== Bug 1: Duplicate Transport Boundary Check ==========

// TestBug1_DuplicateTransportCheck_RedundantValidation tests that the duplicate
// transport boundary check in Access() is redundant and can be removed.
//
// Current behavior: Access() checks transport boundary twice (lines 466-469 and 487-490)
// Expected behavior: Should only check once (before site authorization check)
//
// This test verifies that removing the second check doesn't break anything.
func TestBug1_DuplicateTransportCheck_RedundantValidation(t *testing.T) {
	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	// GIVEN: Expression resolved in local transport
	exprID := v.DeclareVariable("TOKEN", "@env.TOKEN")
	v.expressions[exprID].Value = "secret-value"
	v.expressions[exprID].Resolved = true
	v.exprTransport[exprID] = "local"

	// AND: Authorized site in local transport
	v.EnterStep()
	v.EnterDecorator("@shell")
	v.RecordReference(exprID, "command")

	// WHEN: Access at authorized site
	_, err := v.Access(exprID, "command")

	// THEN: Should succeed (first check is sufficient)
	if err != nil {
		t.Errorf("Access() should succeed, got error: %v", err)
	}

	// NOTE: After fix, remove second checkTransportBoundary call at line 487-490
	// and renumber comment "4." to "5." at line 492
}

// ========== Bug 2: Lazy Initialization Captures Transport at First Reference ==========

// TestBug2_LazyInit_CapturesTransportAtFirstReference reproduces the critical bug
// where transport is captured at first reference instead of at resolution.
//
// Current behavior: checkTransportBoundary() lazily sets exprTransport on first call
// Expected behavior: Transport should be set when expression is resolved
//
// This allows local @env secrets to leak across transport boundaries!
func TestBug2_LazyInit_CapturesTransportAtFirstReference(t *testing.T) {
	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	// GIVEN: Expression declared and resolved in LOCAL transport
	// (Simulating @env.HOME resolved in local context)
	exprID := v.DeclareVariable("HOME", "@env.HOME")
	v.expressions[exprID].Value = "/home/local-user"
	v.expressions[exprID].Resolved = true
	// NOTE: We do NOT manually set v.exprTransport[exprID] here
	// This simulates production code where transport is never recorded at resolution

	// AND: First reference happens in REMOTE SSH transport
	v.EnterTransport("ssh:server1")
	v.EnterStep()
	v.EnterDecorator("@shell")
	v.RecordReference(exprID, "command")

	// WHEN: Access is called in SSH transport (first reference)
	value, err := v.Access(exprID, "command")

	// THEN: BUG - Should FAIL with transport boundary error, but SUCCEEDS
	// Because checkTransportBoundary() lazily sets exprTransport to "ssh:server1"
	// on first call, it thinks the secret was resolved in SSH context!
	if err != nil {
		t.Logf("GOOD: Access() correctly failed with: %v", err)
		t.Logf("This means the bug is already fixed!")
	} else {
		t.Errorf("BUG REPRODUCED: Access() should fail with transport boundary error")
		t.Errorf("Got value: %q (local secret leaked to SSH transport!)", value)
		t.Errorf("Root cause: checkTransportBoundary() lazily set exprTransport to 'ssh:server1'")
		t.Errorf("Expected: exprTransport should be set to 'local' at resolution time")
	}
}

// TestBug2_LazyInit_SubsequentAccessInLocalFails demonstrates the second symptom
// of the lazy initialization bug: once transport is captured at first reference,
// subsequent access in the CORRECT transport fails!
func TestBug2_LazyInit_SubsequentAccessInLocalFails(t *testing.T) {
	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	// GIVEN: Expression resolved in LOCAL transport
	exprID := v.DeclareVariable("HOME", "@env.HOME")
	v.expressions[exprID].Value = "/home/local-user"
	v.expressions[exprID].Resolved = true
	// NOTE: No manual exprTransport assignment (simulates production)

	// AND: First reference in SSH transport (captures transport as "ssh:server1")
	v.EnterTransport("ssh:server1")
	v.EnterStep()
	v.EnterDecorator("@shell")
	v.RecordReference(exprID, "command")
	_, _ = v.Access(exprID, "command") // First access captures transport as SSH

	// AND: Second reference in LOCAL transport (where it was actually resolved)
	v.ExitDecorator()
	v.ExitTransport() // Back to local
	v.EnterStep()
	v.EnterDecorator("@shell")
	v.RecordReference(exprID, "env")

	// WHEN: Access in local transport (the CORRECT transport)
	_, err := v.Access(exprID, "env")

	// THEN: BUG - Should SUCCEED (local is correct), but FAILS
	// Because exprTransport was captured as "ssh:server1" on first reference
	if err == nil {
		t.Logf("GOOD: Access() succeeded in local transport")
		t.Logf("This means the bug is already fixed!")
	} else {
		if strings.Contains(err.Error(), "transport boundary violation") {
			t.Errorf("BUG REPRODUCED: Access() failed in LOCAL transport (the correct one!)")
			t.Errorf("Error: %v", err)
			t.Errorf("Root cause: exprTransport was set to 'ssh:server1' on first reference")
			t.Errorf("Expected: exprTransport should be 'local' (set at resolution time)")
		} else {
			t.Errorf("Unexpected error: %v", err)
		}
	}
}

// ========== Bug 3: Missing MarkResolved() Method ==========

// TestBug3_MissingMarkResolved_NoAPIToSetTransportAtResolution tests that
// there's no proper API to mark an expression as resolved and capture its transport.
//
// Current behavior: No MarkResolved() method exists
// Expected behavior: MarkResolved(exprID, value) should set Resolved=true, Value=value,
//                    and exprTransport[exprID]=currentTransport
func TestBug3_MissingMarkResolved_NoAPIToSetTransportAtResolution(t *testing.T) {
	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	// GIVEN: We're in local transport and want to resolve an @env expression
	exprID := v.DeclareVariable("HOME", "@env.HOME")

	// WHEN: We try to mark it as resolved with its value
	// Expected API: v.MarkResolved(exprID, "/home/local-user")
	// This should:
	// 1. Set expr.Resolved = true
	// 2. Set expr.Value = "/home/local-user"
	// 3. Set v.exprTransport[exprID] = v.currentTransport ("local")

	// Current workaround (what tests do manually):
	v.expressions[exprID].Value = "/home/local-user"
	v.expressions[exprID].Resolved = true
	v.exprTransport[exprID] = "local" // Manual assignment masks the bug!

	// THEN: After fix, we should have a proper API:
	// v.MarkResolved(exprID, "/home/local-user")
	// And checkTransportBoundary should error if exprTransport is missing

	// For now, just verify the workaround works
	if !v.expressions[exprID].Resolved {
		t.Error("Expression should be marked as resolved")
	}
	if v.expressions[exprID].Value != "/home/local-user" {
		t.Errorf("Expression value = %q, want %q", v.expressions[exprID].Value, "/home/local-user")
	}
	if v.exprTransport[exprID] != "local" {
		t.Errorf("Expression transport = %q, want %q", v.exprTransport[exprID], "local")
	}

	t.Log("TODO: Implement MarkResolved(exprID, value) method")
	t.Log("TODO: Remove lazy initialization from checkTransportBoundary")
	t.Log("TODO: Return error if exprTransport[exprID] is missing")
}

// ========== Integration Test: Full Bug Scenario ==========

// TestBug_FullScenario_LocalEnvLeaksToSSH reproduces the complete attack scenario
// where a local @env secret leaks to a remote SSH session.
//
// Attack scenario:
// 1. Declare @env.AWS_SECRET_KEY in local context
// 2. Resolve it to actual value in local context
// 3. First reference happens inside @ssh decorator
// 4. BUG: checkTransportBoundary() captures transport as "ssh:*"
// 5. Access succeeds, leaking local secret to remote host!
func TestBug_FullScenario_LocalEnvLeaksToSSH(t *testing.T) {
	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	// GIVEN: Local environment variable (sensitive secret)
	// In real code, this would be resolved by @env decorator in local context
	exprID := v.DeclareVariable("AWS_SECRET_KEY", "@env.AWS_SECRET_KEY")
	v.expressions[exprID].Value = "AKIAIOSFODNN7EXAMPLE" // Sensitive local secret
	v.expressions[exprID].Resolved = true
	// NOTE: No exprTransport assignment (simulates production bug)

	// AND: Code enters SSH transport (remote server)
	v.EnterTransport("ssh:production-server")
	v.EnterStep()
	v.EnterDecorator("@shell")

	// AND: Remote shell tries to use local secret
	v.RecordReference(exprID, "command")

	// WHEN: Remote decorator tries to access local @env secret
	value, err := v.Access(exprID, "command")

	// THEN: Should FAIL with transport boundary error
	if err != nil {
		if strings.Contains(err.Error(), "transport boundary violation") {
			t.Logf("GOOD: Transport boundary correctly enforced")
			t.Logf("Error: %v", err)
		} else {
			t.Errorf("Wrong error type: %v", err)
		}
	} else {
		t.Errorf("CRITICAL SECURITY BUG: Local @env secret leaked to SSH transport!")
		t.Errorf("Leaked value: %q", value)
		t.Errorf("This should have failed with transport boundary violation")
		t.Errorf("")
		t.Errorf("Attack scenario:")
		t.Errorf("1. Attacker declares @env.AWS_SECRET_KEY in local context")
		t.Errorf("2. First reference happens in @ssh decorator")
		t.Errorf("3. checkTransportBoundary() lazily captures transport as 'ssh:*'")
		t.Errorf("4. Access succeeds, leaking local AWS credentials to remote server!")
		t.Errorf("")
		t.Errorf("Fix: Set exprTransport at resolution time, not first reference")
	}
}

// ========== Test Helper: Verify Fix Doesn't Break Existing Behavior ==========

// TestAfterFix_ExistingBehaviorStillWorks verifies that after fixing the bugs,
// the existing correct behavior still works.
func TestAfterFix_ExistingBehaviorStillWorks(t *testing.T) {
	v := NewWithPlanKey([]byte("test-key-32-bytes-long!!!!!!"))

	// GIVEN: Expression properly resolved with transport set at resolution
	exprID := v.DeclareVariable("TOKEN", "@env.TOKEN")

	// TODO: After implementing MarkResolved(), use it here:
	// v.MarkResolved(exprID, "secret-value")

	// For now, simulate proper resolution:
	v.expressions[exprID].Value = "secret-value"
	v.expressions[exprID].Resolved = true
	v.exprTransport[exprID] = "local" // Set at resolution time

	// AND: Authorized site in same transport
	v.EnterStep()
	v.EnterDecorator("@shell")
	v.RecordReference(exprID, "command")

	// WHEN: Access at authorized site in same transport
	value, err := v.Access(exprID, "command")

	// THEN: Should still succeed
	if err != nil {
		t.Errorf("Access() should succeed, got error: %v", err)
	}
	if value != "secret-value" {
		t.Errorf("Access() = %q, want %q", value, "secret-value")
	}
}
