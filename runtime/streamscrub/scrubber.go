// Package streamscrub provides streaming secret redaction.
package streamscrub

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"io"
	"os"
	"sync"

	"github.com/aledsdavies/opal/core/invariant"
)

// Scrubber redacts secrets from output streams.
type Scrubber struct {
	out     io.Writer
	secrets []secretEntry
	frames  []frame
	carry   []byte // Rolling window for chunk-boundary secrets
	maxLen  int    // Longest registered secret
}

// secretEntry holds a secret pattern and its placeholder.
type secretEntry struct {
	pattern     []byte
	placeholder []byte
}

// frame represents a buffering scope.
type frame struct {
	label string
	buf   bytes.Buffer
}

// New creates a new Scrubber that writes to w.
func New(w io.Writer) *Scrubber {
	// INPUT CONTRACT
	invariant.NotNil(w, "writer")

	s := &Scrubber{out: w}

	// OUTPUT CONTRACT
	invariant.Postcondition(s.out != nil, "scrubber must have output writer")
	invariant.Postcondition(len(s.frames) == 0, "scrubber must start with no active frames")
	invariant.Postcondition(len(s.secrets) == 0, "scrubber must start with no registered secrets")

	return s
}

// RegisterSecret registers a secret to be redacted.
func (s *Scrubber) RegisterSecret(secret, placeholder []byte) {
	// INPUT CONTRACT
	invariant.Precondition(len(secret) > 0, "secret cannot be empty")
	invariant.Precondition(len(placeholder) > 0, "placeholder cannot be empty")

	oldMaxLen := s.maxLen
	oldSecretCount := len(s.secrets)

	s.secrets = append(s.secrets, secretEntry{
		pattern:     secret,
		placeholder: placeholder,
	})

	// Update maxLen to track longest secret
	if len(secret) > s.maxLen {
		s.maxLen = len(secret)
	}

	// OUTPUT CONTRACT
	invariant.Postcondition(len(s.secrets) == oldSecretCount+1, "secret must be registered")
	invariant.Postcondition(s.maxLen >= oldMaxLen, "maxLen must not decrease")
	invariant.Postcondition(s.maxLen >= len(secret), "maxLen must be at least as long as new secret")
}

// StartFrame begins a new buffering scope.
func (s *Scrubber) StartFrame(label string) {
	// INPUT CONTRACT
	invariant.Precondition(label != "", "frame label cannot be empty")

	oldFrameCount := len(s.frames)

	s.frames = append(s.frames, frame{
		label: label,
		buf:   bytes.Buffer{},
	})

	// OUTPUT CONTRACT
	invariant.Postcondition(len(s.frames) == oldFrameCount+1, "frame must be pushed onto stack")
	invariant.Postcondition(s.frames[len(s.frames)-1].label == label, "frame label must match")
}

// EndFrame ends the current frame and flushes scrubbed output.
func (s *Scrubber) EndFrame(secrets [][]byte) error {
	// INPUT CONTRACT
	invariant.Precondition(len(s.frames) > 0, "cannot end frame when no frame is active")

	oldFrameCount := len(s.frames)
	oldSecretCount := len(s.secrets)

	// Pop current frame
	currentFrame := s.frames[len(s.frames)-1]
	s.frames = s.frames[:len(s.frames)-1]

	// Register secrets with generated placeholders
	// LOOP INVARIANT: Track progress through secrets slice
	prevIdx := -1
	for idx, secret := range secrets {
		// Assert loop makes progress
		invariant.Postcondition(idx > prevIdx, "loop must make progress")
		prevIdx = idx

		if len(secret) > 0 {
			placeholder := generatePlaceholder(secret)
			s.RegisterSecret(secret, []byte(placeholder))
		}
	}

	// Scrub frame buffer with all known secrets
	scrubbed := currentFrame.buf.Bytes()

	// LOOP INVARIANT: Track progress through secrets
	prevIdx = -1
	for idx, entry := range s.secrets {
		// Assert loop makes progress
		invariant.Postcondition(idx > prevIdx, "loop must make progress")
		prevIdx = idx

		scrubbed = bytes.ReplaceAll(scrubbed, entry.pattern, entry.placeholder)
	}

	// OUTPUT CONTRACT
	invariant.Postcondition(len(s.frames) == oldFrameCount-1, "frame must be popped from stack")
	invariant.Postcondition(len(s.secrets) >= oldSecretCount, "secrets must be registered")

	// Flush to output
	_, err := s.out.Write(scrubbed)
	return err
}

// RegisterSecretWithVariants registers a secret and all its encoding variants.
func (s *Scrubber) RegisterSecretWithVariants(secret []byte) {
	// INPUT CONTRACT
	invariant.Precondition(len(secret) > 0, "secret cannot be empty")

	placeholder := []byte(generatePlaceholder(secret))

	// Register raw secret
	s.RegisterSecret(secret, placeholder)

	// Register encoding variants
	s.registerVariants(secret, placeholder)
}

// registerVariants registers all encoding variants of a secret.
func (s *Scrubber) registerVariants(secret, placeholder []byte) {
	// Hex: lowercase and uppercase
	hexLower := []byte(toHex(secret))
	hexUpper := []byte(toUpperHex(toHex(secret)))
	s.RegisterSecret(hexLower, placeholder)
	s.RegisterSecret(hexUpper, placeholder)

	// Base64: standard encoding
	b64 := []byte(toBase64(secret))
	s.RegisterSecret(b64, placeholder)

	// TODO: Add more variants (URL encoding, path escape, separators, etc.)
}

// SecretCount returns the number of registered secret patterns (for testing/debugging).
func (s *Scrubber) SecretCount() int {
	return len(s.secrets)
}

// MaxPatternLen returns the longest registered secret pattern (for testing/debugging).
func (s *Scrubber) MaxPatternLen() int {
	return s.maxLen
}

// generatePlaceholder creates a placeholder for a secret.
// For now, just use a simple format. We'll add keyed hashing later.
func generatePlaceholder(secret []byte) string {
	// Simple hash for now - we'll improve this
	return "opal:s:xxxxxxxx"
}

// Helper functions for encoding variants

func toHex(b []byte) string {
	return hex.EncodeToString(b)
}

func toUpperHex(s string) string {
	result := make([]byte, len(s))
	for i := range s {
		if s[i] >= 'a' && s[i] <= 'f' {
			result[i] = s[i] - 32 // 'a' -> 'A'
		} else {
			result[i] = s[i]
		}
	}
	return string(result)
}

func toBase64(b []byte) string {
	return base64.StdEncoding.EncodeToString(b)
}

// LockdownStreams redirects stdout and stderr through the scrubber.
// Returns a restore function that MUST be deferred to restore original streams.
//
// Usage:
//
//	scrubber := streamscrub.New(os.Stdout)
//	restore := scrubber.LockdownStreams()
//	defer restore()
//	// All stdout/stderr now goes through scrubber
func (s *Scrubber) LockdownStreams() func() {
	// INPUT CONTRACT
	invariant.Precondition(s.out != nil, "scrubber must have output writer")

	// Save original streams
	originalStdout := os.Stdout
	originalStderr := os.Stderr

	// Create pipes for stdout and stderr
	rOut, wOut, err := os.Pipe()
	if err != nil {
		panic("streamscrub: failed to create stdout pipe: " + err.Error())
	}

	rErr, wErr, err := os.Pipe()
	if err != nil {
		panic("streamscrub: failed to create stderr pipe: " + err.Error())
	}

	// Redirect os.Stdout and os.Stderr to write ends of pipes
	os.Stdout = wOut
	os.Stderr = wErr

	// Copy from pipes to scrubber in background
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		_, _ = io.Copy(s, rOut)
	}()

	go func() {
		defer wg.Done()
		_, _ = io.Copy(s, rErr)
	}()

	// Return restore function
	return func() {
		// Close write ends to signal EOF to copy goroutines
		_ = wOut.Close()
		_ = wErr.Close()

		// Wait for copy goroutines to finish
		wg.Wait()

		// Close read ends
		_ = rOut.Close()
		_ = rErr.Close()

		// Restore original streams
		os.Stdout = originalStdout
		os.Stderr = originalStderr

		// Flush any remaining buffered data
		_ = s.Flush()
	}
}

// Write implements io.Writer - scrubs secrets before writing.
func (s *Scrubber) Write(p []byte) (int, error) {
	// INPUT CONTRACT
	invariant.Precondition(s.out != nil, "output writer must not be nil")

	if len(p) == 0 {
		return 0, nil
	}

	// If we're in a frame, buffer the output
	if len(s.frames) > 0 {
		currentFrame := &s.frames[len(s.frames)-1]
		n, err := currentFrame.buf.Write(p)

		// OUTPUT CONTRACT (frame mode)
		invariant.Postcondition(n == len(p) || err != nil, "must write all bytes or return error")
		return n, err
	}

	// Streaming mode: merge with carry from previous write
	buf := append(append([]byte{}, s.carry...), p...)

	// Scrub all secrets
	// LOOP INVARIANT: Track progress through secrets
	result := buf
	prevIdx := -1
	for idx, entry := range s.secrets {
		// Assert loop makes progress
		invariant.Postcondition(idx > prevIdx, "loop must make progress")
		prevIdx = idx

		result = bytes.ReplaceAll(result, entry.pattern, entry.placeholder)
	}

	// Keep last maxLen-1 bytes as carry for next write
	// (in case secret is split across chunk boundary)
	carrySize := 0
	if s.maxLen > 0 {
		carrySize = s.maxLen - 1
		// UTF-8 safety: hold back at least 3 bytes for multi-byte code points
		if carrySize < 3 {
			carrySize = 3
		}
	}

	// INVARIANT: carrySize must be reasonable
	invariant.Postcondition(carrySize >= 0, "carrySize must be non-negative")
	invariant.Postcondition(carrySize < 1024*1024, "carrySize must be reasonable (<1MB)")

	if carrySize > 0 && len(result) > carrySize {
		// Write everything except the carry
		toWrite := result[:len(result)-carrySize]
		s.carry = append(s.carry[:0], result[len(result)-carrySize:]...)

		// INVARIANT: carry size matches expected
		invariant.Postcondition(len(s.carry) == carrySize, "carry must be exactly carrySize bytes")

		_, err := s.out.Write(toWrite)
		if err != nil {
			return 0, err
		}
	} else if carrySize > 0 {
		// Buffer is smaller than carry size, accumulate
		s.carry = append(s.carry[:0], result...)

		// INVARIANT: carry doesn't exceed expected size
		invariant.Postcondition(len(s.carry) <= carrySize, "carry must not exceed carrySize")
	} else {
		// No secrets registered, write everything immediately
		_, err := s.out.Write(result)
		if err != nil {
			return 0, err
		}
	}

	// OUTPUT CONTRACT (streaming mode)
	// Return original length (io.Writer contract)
	return len(p), nil
}

// Flush writes any remaining carry bytes after redaction.
func (s *Scrubber) Flush() error {
	if len(s.carry) == 0 {
		return nil
	}

	// Scrub carry one final time
	// LOOP INVARIANT: Track progress through secrets
	result := s.carry
	prevIdx := -1
	for idx, entry := range s.secrets {
		// Assert loop makes progress
		invariant.Postcondition(idx > prevIdx, "loop must make progress")
		prevIdx = idx

		result = bytes.ReplaceAll(result, entry.pattern, entry.placeholder)
	}

	// Write and clear carry
	_, err := s.out.Write(result)
	s.carry = s.carry[:0]

	// OUTPUT CONTRACT
	invariant.Postcondition(len(s.carry) == 0, "carry must be cleared after flush")

	return err
}

// Close flushes remaining data and zeroizes sensitive buffers.
// Callers MUST defer Close() to prevent secret leakage.
func (s *Scrubber) Close() error {
	// Flush any remaining data
	err := s.Flush()

	// Zeroize carry buffer
	for i := range s.carry {
		s.carry[i] = 0
	}
	s.carry = s.carry[:0]

	// Zeroize any open frame buffers
	// LOOP INVARIANT: Track progress through frames
	prevIdx := -1
	for idx := range s.frames {
		// Assert loop makes progress
		invariant.Postcondition(idx > prevIdx, "loop must make progress")
		prevIdx = idx

		buf := s.frames[idx].buf.Bytes()
		for j := range buf {
			buf[j] = 0
		}
	}
	s.frames = s.frames[:0]

	// OUTPUT CONTRACT
	invariant.Postcondition(len(s.carry) == 0, "carry must be cleared")
	invariant.Postcondition(len(s.frames) == 0, "frames must be cleared")

	return err
}
