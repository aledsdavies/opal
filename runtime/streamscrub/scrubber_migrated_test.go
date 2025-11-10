package streamscrub

import (
	"bytes"
	"fmt"
	"os"
	"sync"
	"testing"
)

// These tests verify that all scrubber features work with SecretProvider
// instead of the legacy RegisterSecret API.

// TestMigrated_SimpleSecretRedaction verifies basic secret scrubbing
func TestMigrated_SimpleSecretRedaction(t *testing.T) {
	var buf bytes.Buffer
	provider := testProvider(map[string]string{
		"my-secret-key": "<REDACTED>",
	})
	s := New(&buf, WithSecretProvider(provider))

	s.Write([]byte("The key is: my-secret-key"))
	s.Flush()

	want := "The key is: <REDACTED>"
	if got := buf.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestMigrated_ChunkBoundarySafety verifies secrets split across writes
func TestMigrated_ChunkBoundarySafety(t *testing.T) {
	var buf bytes.Buffer
	provider := testProvider(map[string]string{
		"secret-value-123": "<REDACTED>",
	})
	s := New(&buf, WithSecretProvider(provider))

	s.Write([]byte("prefix secret-"))
	s.Write([]byte("value-"))
	s.Write([]byte("123 suffix"))
	s.Flush()

	want := "prefix <REDACTED> suffix"
	if got := buf.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestMigrated_LockdownStreams verifies stdout/stderr redirection
func TestMigrated_LockdownStreams(t *testing.T) {
	var buf safeBuffer
	provider := testProvider(map[string]string{
		"my-password": "<REDACTED>",
	})
	s := New(&buf, WithSecretProvider(provider))

	restore := s.LockdownStreams()
	defer restore()

	fmt.Println("Password is: my-password")
	fmt.Fprintln(os.Stderr, "Error: my-password failed")

	restore()

	got := buf.String()
	if bytes.Contains([]byte(got), []byte("my-password")) {
		t.Errorf("secret leaked through lockdown: %q", got)
	}
	if !bytes.Contains([]byte(got), []byte("<REDACTED>")) {
		t.Errorf("placeholder not found in output: %q", got)
	}
}

// TestMigrated_ConcurrentWrites verifies thread safety
func TestMigrated_ConcurrentWrites(t *testing.T) {
	var buf safeBuffer
	provider := testProvider(map[string]string{
		"secret": "<REDACTED>",
	})
	s := New(&buf, WithSecretProvider(provider))

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			msg := fmt.Sprintf("goroutine %d: secret\n", id)
			s.Write([]byte(msg))
		}(i)
	}

	wg.Wait()
	s.Flush()

	got := buf.String()
	if bytes.Contains([]byte(got), []byte("secret")) {
		t.Errorf("secret leaked in concurrent writes: %q", got)
	}
}

// TestMigrated_OverlappingSecrets verifies longest-first matching
func TestMigrated_OverlappingSecrets(t *testing.T) {
	var buf bytes.Buffer
	provider := testProvider(map[string]string{
		"SECRET":          "<SHORT>",
		"SECRET_EXTENDED": "<LONG>",
	})
	s := New(&buf, WithSecretProvider(provider))

	s.Write([]byte("Value: SECRET_EXTENDED"))
	s.Flush()

	got := buf.String()
	// Should use longest match
	if got != "Value: <LONG>" {
		t.Errorf("got %q, want %q", got, "Value: <LONG>")
	}
	// Should not have partial replacement
	if bytes.Contains([]byte(got), []byte("SECRET")) {
		t.Errorf("secret not fully replaced: %q", got)
	}
}

// TestMigrated_SplitBoundaryRedaction verifies multi-chunk secrets
func TestMigrated_SplitBoundaryRedaction(t *testing.T) {
	var buf bytes.Buffer
	provider := testProvider(map[string]string{
		"LONG_SECRET_TOKEN": "<REDACTED>",
	})
	s := New(&buf, WithSecretProvider(provider))

	// Split across 4 writes
	s.Write([]byte("prefix LONG_"))
	s.Write([]byte("SECRET_"))
	s.Write([]byte("TOKEN"))
	s.Write([]byte(" suffix"))
	s.Flush()

	want := "prefix <REDACTED> suffix"
	if got := buf.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
