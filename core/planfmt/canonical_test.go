package planfmt_test

import (
	"bytes"
	"testing"

	"github.com/aledsdavies/opal/core/planfmt"
)

// TestCanonicalFormByteStability verifies that canonical form is deterministic
// Same plan → same canonical bytes (100 runs)
func TestCanonicalFormByteStability(t *testing.T) {
	// Create a test plan with various types
	plan := &planfmt.Plan{
		Steps: []planfmt.Step{
			{
				ID: 1,
				Tree: &planfmt.CommandNode{
					Decorator: "@shell",
					Args: []planfmt.Arg{
						{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo hello"}},
					},
				},
			},
		},
	}

	// Run 100 times to verify byte-for-byte stability
	var firstCanonical []byte
	for i := 0; i < 100; i++ {
		canonical, err := plan.Canonicalize()
		if err != nil {
			t.Fatalf("run %d: canonicalization failed: %v", i, err)
		}

		bytes, err := canonical.MarshalBinary()
		if err != nil {
			t.Fatalf("run %d: marshal failed: %v", i, err)
		}

		if i == 0 {
			firstCanonical = bytes
		} else {
			if !bytesEqual(firstCanonical, bytes) {
				t.Fatalf("run %d: canonical form not stable\nwant: %x\ngot:  %x", i, firstCanonical, bytes)
			}
		}
	}

	t.Logf("Canonical form stable across 100 runs (%d bytes)", len(firstCanonical))
}

// TestCanonicalFormWithComplexTypes tests determinism with maps, arrays, and mixed types
func TestCanonicalFormWithComplexTypes(t *testing.T) {
	tests := []struct {
		name string
		plan *planfmt.Plan
	}{
		{
			name: "empty plan",
			plan: &planfmt.Plan{},
		},
		{
			name: "plan with map args (unsorted keys)",
			plan: &planfmt.Plan{
				Steps: []planfmt.Step{
					{
						ID: 1,
						Tree: &planfmt.CommandNode{
							Decorator: "@http",
							Args: []planfmt.Arg{
								{Key: "zebra", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "last"}},
								{Key: "alpha", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "first"}},
								{Key: "middle", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "mid"}},
							},
						},
					},
				},
			},
		},
		{
			name: "plan with unicode",
			plan: &planfmt.Plan{
				Steps: []planfmt.Step{
					{
						ID: 1,
						Tree: &planfmt.CommandNode{
							Decorator: "@shell",
							Args: []planfmt.Arg{
								{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo 你好世界 🔒"}},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Run 10 times to verify stability
			var firstBytes []byte
			for i := 0; i < 10; i++ {
				canonical, err := tt.plan.Canonicalize()
				if err != nil {
					t.Fatalf("canonicalization failed: %v", err)
				}

				bytes, err := canonical.MarshalBinary()
				if err != nil {
					t.Fatalf("marshal failed: %v", err)
				}

				if i == 0 {
					firstBytes = bytes
				} else if !bytesEqual(firstBytes, bytes) {
					t.Fatalf("canonical form not stable on run %d", i)
				}
			}
		})
	}
}

// TestCanonicalVersion verifies version field is included
func TestCanonicalVersion(t *testing.T) {
	plan := &planfmt.Plan{
		Steps: []planfmt.Step{
			{ID: 1, Tree: &planfmt.CommandNode{Decorator: "@shell"}},
		},
	}

	canonical, err := plan.Canonicalize()
	if err != nil {
		t.Fatalf("canonicalization failed: %v", err)
	}

	// Version should be set
	if canonical.Version == 0 {
		t.Error("expected canonical version to be set, got 0")
	}

	t.Logf("Canonical version: %d", canonical.Version)
}

// TestCanonicalTargetUnlinkability verifies that different targets produce different hashes
// This ensures deploy and destroy commands with identical steps are unlinkable
func TestCanonicalTargetUnlinkability(t *testing.T) {
	// Deploy plan
	deploy := &planfmt.Plan{
		Target: "deploy",
		Steps: []planfmt.Step{
			{
				ID: 1,
				Tree: &planfmt.CommandNode{
					Decorator: "@shell",
					Args: []planfmt.Arg{
						{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "kubectl apply"}},
					},
				},
			},
		},
	}

	// Destroy plan with identical steps but different target
	destroy := &planfmt.Plan{
		Target: "destroy",
		Steps: []planfmt.Step{
			{
				ID: 1,
				Tree: &planfmt.CommandNode{
					Decorator: "@shell",
					Args: []planfmt.Arg{
						{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "kubectl apply"}},
					},
				},
			},
		},
	}

	// Both should produce different canonical hashes
	hash1, err := deploy.Canonicalize()
	if err != nil {
		t.Fatalf("deploy canonicalization failed: %v", err)
	}
	bytes1, err := hash1.MarshalBinary()
	if err != nil {
		t.Fatalf("deploy marshal failed: %v", err)
	}

	hash2, err := destroy.Canonicalize()
	if err != nil {
		t.Fatalf("destroy canonicalization failed: %v", err)
	}
	bytes2, err := hash2.MarshalBinary()
	if err != nil {
		t.Fatalf("destroy marshal failed: %v", err)
	}

	if bytesEqual(bytes1, bytes2) {
		t.Errorf("Different targets produced same canonical hash - deploy and destroy should be unlinkable\ndeploy: %x\ndestroy: %x", bytes1, bytes2)
	}
}

// Helper function for byte comparison
func bytesEqual(a, b []byte) bool {
	return bytes.Equal(a, b)
}
