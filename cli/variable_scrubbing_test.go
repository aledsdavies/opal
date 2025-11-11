package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestVariableScrubbing_EndToEnd tests the complete flow:
// 1. CLI creates Vault
// 2. Planner declares variables in Vault
// 3. Scrubber uses Vault.SecretProvider() to scrub output
func TestVariableScrubbing_EndToEnd(t *testing.T) {
	// Create temporary opal file
	tmpDir := t.TempDir()
	opalFile := filepath.Join(tmpDir, "test.opl")

	source := `var SECRET = "my-secret-value"
echo "The secret is: my-secret-value"`

	err := os.WriteFile(opalFile, []byte(source), 0o644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Capture stdout/stderr
	var output bytes.Buffer
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stdout = w
	os.Stderr = w

	// Run CLI (this will call main() which we can't easily test)
	// Instead, we'll test the runCommand function directly
	// For now, just verify the file was created
	if _, err := os.Stat(opalFile); os.IsNotExist(err) {
		t.Fatal("Test file was not created")
	}

	// Restore stdout/stderr
	w.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr
	output.ReadFrom(r)

	// Note: This test is a placeholder. The real test would need to:
	// 1. Call runCommand directly with a test vault
	// 2. Verify the output contains DisplayID instead of raw secret
	// 3. Verify the raw secret is NOT in the output

	t.Log("End-to-end test placeholder created")
}

// TestVariableScrubbing_SingleVariable tests scrubbing of a single variable
func TestVariableScrubbing_SingleVariable(t *testing.T) {
	tmpDir := t.TempDir()
	opalFile := filepath.Join(tmpDir, "single.opl")

	source := `var API_KEY = "sk-secret-123"
echo "API key: sk-secret-123"`

	err := os.WriteFile(opalFile, []byte(source), 0o644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// TODO: Execute CLI and capture output
	// For now, verify file exists
	if _, err := os.Stat(opalFile); os.IsNotExist(err) {
		t.Fatal("Test file was not created")
	}

	t.Log("Single variable test file created")
}

// TestVariableScrubbing_MultipleVariables tests scrubbing of multiple variables
func TestVariableScrubbing_MultipleVariables(t *testing.T) {
	tmpDir := t.TempDir()
	opalFile := filepath.Join(tmpDir, "multiple.opl")

	source := `var API_KEY = "sk-secret-123"
var TOKEN = "token-456"
var PASSWORD = "pass-789"
echo "API: sk-secret-123, Token: token-456, Pass: pass-789"`

	err := os.WriteFile(opalFile, []byte(source), 0o644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// TODO: Execute CLI and capture output
	// Verify all three secrets are scrubbed
	// Verify output contains three different DisplayIDs

	t.Log("Multiple variables test file created")
}

// TestVariableScrubbing_NoLeakage tests that raw values never appear in output
func TestVariableScrubbing_NoLeakage(t *testing.T) {
	tmpDir := t.TempDir()
	opalFile := filepath.Join(tmpDir, "noleak.opl")

	secretValue := "super-secret-password-12345"
	source := `var PASSWORD = "` + secretValue + `"
echo "Password is: ` + secretValue + `"`

	err := os.WriteFile(opalFile, []byte(source), 0o644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// TODO: Execute CLI and capture output
	// CRITICAL: Verify secretValue does NOT appear in output
	// Verify output contains "opal:v:" DisplayID

	t.Log("No leakage test file created")
}

// TestVariableScrubbing_LongestFirst tests that longer secrets are matched first
func TestVariableScrubbing_LongestFirst(t *testing.T) {
	tmpDir := t.TempDir()
	opalFile := filepath.Join(tmpDir, "longest.opl")

	source := `var SHORT = "secret"
var LONG = "secret-key-123"
echo "Value: secret-key-123 and secret"`

	err := os.WriteFile(opalFile, []byte(source), 0o644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// TODO: Execute CLI and capture output
	// Verify both secrets are scrubbed
	// Verify "secret-key-123" is matched before "secret"

	t.Log("Longest-first test file created")
}

// Helper function to execute CLI and capture output (to be implemented)
func executeCLI(t *testing.T, opalFile string) string {
	// This would execute the CLI with the given file and return the output
	// For now, it's a placeholder
	t.Helper()
	return ""
}

// Helper function to verify scrubbing (to be implemented)
func verifyScrubbed(t *testing.T, output, rawSecret string) {
	t.Helper()

	// Verify raw secret is NOT in output
	if strings.Contains(output, rawSecret) {
		t.Errorf("Output contains raw secret %q - scrubbing failed!", rawSecret)
		t.Logf("Output: %s", output)
	}

	// Verify output contains DisplayID marker
	if !strings.Contains(output, "opal:v:") {
		t.Error("Output should contain DisplayID marker (opal:v:...)")
		t.Logf("Output: %s", output)
	}
}
