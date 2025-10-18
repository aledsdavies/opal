package streamscrub

import (
	"bytes"
	"fmt"
	"os"
	"testing"
)

// TestBasicScrubbing - simplest possible test
func TestBasicScrubbing(t *testing.T) {
	var buf bytes.Buffer
	s := New(&buf)

	// Write some output
	input := []byte("hello world")
	n, err := s.Write(input)

	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(input) {
		t.Fatalf("Write returned %d, want %d", n, len(input))
	}

	// Flush to get output
	if err := s.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	// Should pass through unchanged (no secrets registered)
	if got := buf.String(); got != "hello world" {
		t.Errorf("got %q, want %q", got, "hello world")
	}
}

// TestSimpleSecretRedaction - register a secret and scrub it
func TestSimpleSecretRedaction(t *testing.T) {
	var buf bytes.Buffer
	s := New(&buf)

	// Register a secret
	secret := []byte("my-secret-key")
	placeholder := []byte("<REDACTED>")
	s.RegisterSecret(secret, placeholder)

	// Write output containing the secret
	input := []byte("The key is: my-secret-key")
	s.Write(input)
	s.Flush()

	// Secret should be redacted
	want := "The key is: <REDACTED>"
	if got := buf.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestFrameBuffering - buffer output during frame, flush after
func TestFrameBuffering(t *testing.T) {
	var buf bytes.Buffer
	s := New(&buf)

	// Start a frame
	s.StartFrame("test-frame")

	// Write during frame - should be buffered
	s.Write([]byte("buffered output"))

	// Nothing should be written yet
	if buf.Len() != 0 {
		t.Errorf("output written during frame, want buffered")
	}

	// End frame with a secret
	secret := []byte("secret123")
	s.EndFrame([][]byte{secret})

	// Now output should be flushed (no secret in this output, so unchanged)
	want := "buffered output"
	if got := buf.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestFrameScrubbingHierarchical - secret registered in frame scrubs frame output
func TestFrameScrubbingHierarchical(t *testing.T) {
	var buf bytes.Buffer
	s := New(&buf)

	// Start a frame
	s.StartFrame("decorator-frame")

	// Decorator prints its secret during execution
	s.Write([]byte("Loading secret: secret123"))

	// End frame and register the secret
	secret := []byte("secret123")
	s.EndFrame([][]byte{secret})

	// Frame output should be scrubbed before flushing
	want := "Loading secret: opal:s:" // Placeholder will be generated
	got := buf.String()
	if !bytes.HasPrefix([]byte(got), []byte(want)) {
		t.Errorf("got %q, want prefix %q", got, want)
	}
}

// TestChunkBoundarySafety - secret split across multiple writes
func TestChunkBoundarySafety(t *testing.T) {
	var buf bytes.Buffer
	s := New(&buf)

	// Register a secret
	secret := []byte("secret-value-123")
	placeholder := []byte("<REDACTED>")
	s.RegisterSecret(secret, placeholder)

	// Write secret split across 3 chunks
	s.Write([]byte("prefix secret-"))
	s.Write([]byte("value-"))
	s.Write([]byte("123 suffix"))

	// Flush to get final output
	s.Flush()

	// Secret should be scrubbed even though split
	want := "prefix <REDACTED> suffix"
	if got := buf.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestNestedFrames - inner frame can access outer frame's secrets
func TestNestedFrames(t *testing.T) {
	var buf bytes.Buffer
	s := New(&buf)

	// Outer frame
	s.StartFrame("outer")
	s.Write([]byte("outer: "))

	// Register outer secret
	outerSecret := []byte("outer-secret")
	s.EndFrame([][]byte{outerSecret})

	// Inner frame
	s.StartFrame("inner")
	s.Write([]byte("inner prints outer: outer-secret"))

	// Register inner secret
	innerSecret := []byte("inner-secret")
	s.EndFrame([][]byte{innerSecret})

	// Both secrets should be scrubbed
	got := buf.String()
	if bytes.Contains([]byte(got), outerSecret) {
		t.Errorf("outer secret leaked: %q", got)
	}
	if bytes.Contains([]byte(got), innerSecret) {
		t.Errorf("inner secret leaked: %q", got)
	}
}

// TestEncodingVariants - secrets in hex/base64 are also scrubbed
func TestEncodingVariants(t *testing.T) {
	var buf bytes.Buffer
	s := New(&buf)

	// Register secret with variants
	secret := []byte("test")
	s.RegisterSecretWithVariants(secret)

	// Write secret in various encodings
	// Hex: 74657374
	s.Write([]byte("hex: 74657374\n"))
	// Base64: dGVzdA==
	s.Write([]byte("base64: dGVzdA==\n"))
	// Raw
	s.Write([]byte("raw: test\n"))

	s.Flush()

	got := buf.String()
	// All variants should be scrubbed
	if bytes.Contains([]byte(got), []byte("74657374")) {
		t.Errorf("hex variant leaked: %q", got)
	}
	if bytes.Contains([]byte(got), []byte("dGVzdA")) {
		t.Errorf("base64 variant leaked: %q", got)
	}
	if bytes.Contains([]byte(got), secret) {
		t.Errorf("raw secret leaked: %q", got)
	}
}

// TestLockdownStreams - stdout/stderr are redirected through scrubber
func TestLockdownStreams(t *testing.T) {
	var buf bytes.Buffer
	s := New(&buf)

	// Register a secret
	secret := []byte("my-password")
	placeholder := []byte("<REDACTED>")
	s.RegisterSecret(secret, placeholder)

	// Lockdown streams
	restore := s.LockdownStreams()
	defer restore()

	// Print to stdout (should go through scrubber)
	fmt.Println("Password is: my-password")

	// Print to stderr (should also go through scrubber)
	fmt.Fprintln(os.Stderr, "Error: my-password failed")

	// Restore streams (defer will call this, but we call explicitly for testing)
	restore()

	// Check output was scrubbed
	got := buf.String()
	if bytes.Contains([]byte(got), secret) {
		t.Errorf("secret leaked through lockdown: %q", got)
	}
	if !bytes.Contains([]byte(got), placeholder) {
		t.Errorf("placeholder not found in output: %q", got)
	}
}

// TestCloseZeroization - Close() zeroizes sensitive buffers
func TestCloseZeroization(t *testing.T) {
	var buf bytes.Buffer
	s := New(&buf)

	// Write some data to carry buffer
	secret := []byte("secret123")
	s.RegisterSecret(secret, []byte("<REDACTED>"))
	s.Write([]byte("partial"))

	// Start a frame with buffered data
	s.StartFrame("test")
	s.Write([]byte("buffered data"))

	// Close should flush and zeroize
	if err := s.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Verify buffers are cleared
	if len(s.carry) != 0 {
		t.Errorf("carry not cleared after Close, len=%d", len(s.carry))
	}
	if len(s.frames) != 0 {
		t.Errorf("frames not cleared after Close, len=%d", len(s.frames))
	}
}
