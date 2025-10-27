package decorator

import (
	"os"
	"strings"
	"testing"
)

// TestSSHSessionToLocalhost tests real SSH connection to localhost
// This requires SSH server running on localhost with key-based auth
func TestSSHSessionToLocalhost(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping SSH integration test in short mode")
	}

	// Check if we can connect to localhost
	if !canSSHToLocalhost(t) {
		t.Skip("Cannot SSH to localhost - skipping test")
	}

	// Create SSH session
	session, err := NewSSHSession(map[string]any{
		"host": "localhost",
		"port": 22,
		"user": os.Getenv("USER"),
	})
	if err != nil {
		t.Fatalf("Failed to create SSH session: %v", err)
	}
	defer session.Close()

	// Test Run()
	result, err := session.Run([]string{"echo", "hello"}, RunOpts{})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("ExitCode: got %d, want 0", result.ExitCode)
	}

	output := strings.TrimSpace(string(result.Stdout))
	if output != "hello" {
		t.Errorf("Output: got %q, want %q", output, "hello")
	}
}

// TestSSHSessionEnv tests reading remote environment
func TestSSHSessionEnv(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping SSH integration test in short mode")
	}

	if !canSSHToLocalhost(t) {
		t.Skip("Cannot SSH to localhost - skipping test")
	}

	session, err := NewSSHSession(map[string]any{
		"host": "localhost",
		"user": os.Getenv("USER"),
	})
	if err != nil {
		t.Fatalf("Failed to create SSH session: %v", err)
	}
	defer session.Close()

	// Get remote environment
	env := session.Env()

	// Should have common env vars
	if env["HOME"] == "" {
		t.Error("Remote HOME is empty")
	}

	if env["USER"] == "" {
		t.Error("Remote USER is empty")
	}

	// Verify it matches what we expect
	expectedUser := os.Getenv("USER")
	if env["USER"] != expectedUser {
		t.Errorf("Remote USER: got %q, want %q", env["USER"], expectedUser)
	}
}

// TestSSHSessionIsolation tests that SSH sessions are isolated from local
func TestSSHSessionIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping SSH integration test in short mode")
	}

	if !canSSHToLocalhost(t) {
		t.Skip("Cannot SSH to localhost - skipping test")
	}

	// Create local session
	local := NewLocalSession()
	localModified := local.WithEnv(map[string]string{
		"OPAL_TEST_VAR": "local_value",
	})

	// Create SSH session
	ssh, err := NewSSHSession(map[string]any{
		"host": "localhost",
		"user": os.Getenv("USER"),
	})
	if err != nil {
		t.Fatalf("Failed to create SSH session: %v", err)
	}
	defer ssh.Close()

	// Verify local session has the var
	localEnv := localModified.Env()
	if localEnv["OPAL_TEST_VAR"] != "local_value" {
		t.Errorf("Local OPAL_TEST_VAR: got %q, want %q", localEnv["OPAL_TEST_VAR"], "local_value")
	}

	// Verify SSH session does NOT have the var
	sshEnv := ssh.Env()
	if _, ok := sshEnv["OPAL_TEST_VAR"]; ok {
		t.Error("SSH session has OPAL_TEST_VAR from local session (leaked!)")
	}
}

// TestSSHSessionPooling tests that sessions are reused
func TestSSHSessionPooling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping SSH integration test in short mode")
	}

	if !canSSHToLocalhost(t) {
		t.Skip("Cannot SSH to localhost - skipping test")
	}

	pool := NewSessionPool()
	defer pool.CloseAll()

	transport := NewMonitoredTransport(&SSHTransport{})
	parent := NewLocalSession()
	params := map[string]any{
		"host": "localhost",
		"user": os.Getenv("USER"),
	}

	// First call creates session
	session1, err := pool.GetOrCreate(transport, parent, params)
	if err != nil {
		t.Fatalf("GetOrCreate failed: %v", err)
	}

	// Second call reuses session
	session2, err := pool.GetOrCreate(transport, parent, params)
	if err != nil {
		t.Fatalf("GetOrCreate failed: %v", err)
	}

	// Verify only one Open call
	if transport.OpenCalls != 1 {
		t.Errorf("OpenCalls: got %d, want 1 (should reuse)", transport.OpenCalls)
	}

	// Verify same session instance
	if session1 != session2 {
		t.Error("Expected same session instance for same params")
	}
}

// canSSHToLocalhost checks if we can SSH to localhost
func canSSHToLocalhost(t *testing.T) bool {
	t.Helper()

	// Try to create an SSH session - NewSSHSession will handle auth
	session, err := NewSSHSession(map[string]any{
		"host": "localhost",
		"user": os.Getenv("USER"),
	})
	if err != nil {
		t.Logf("Cannot SSH to localhost: %v", err)
		return false
	}
	session.Close()
	return true
}
