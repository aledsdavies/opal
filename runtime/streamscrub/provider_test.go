package streamscrub

import (
	"bytes"
	"testing"
)

// mockProvider is a simple SecretProvider for testing
type mockProvider struct {
	secrets map[string]string // secret → placeholder
}

func newMockProvider() *mockProvider {
	return &mockProvider{
		secrets: make(map[string]string),
	}
}

func (m *mockProvider) AddSecret(secret, placeholder string) {
	m.secrets[secret] = placeholder
}

func (m *mockProvider) FindSecret(chunk []byte) ([]byte, []byte, bool) {
	// Find longest match (greedy)
	var longestSecret string
	var longestPlaceholder string
	
	for secret, placeholder := range m.secrets {
		if bytes.Contains(chunk, []byte(secret)) {
			if len(secret) > len(longestSecret) {
				longestSecret = secret
				longestPlaceholder = placeholder
			}
		}
	}
	
	if longestSecret != "" {
		return []byte(longestSecret), []byte(longestPlaceholder), true
	}
	
	return nil, nil, false
}

// TestSecretProvider_NilProvider tests scrubber with no provider (pass-through)
func TestSecretProvider_NilProvider(t *testing.T) {
	var buf bytes.Buffer
	s := New(&buf, WithSecretProvider(nil))
	
	input := []byte("secret data here")
	s.Write(input)
	s.Flush()
	
	// Should pass through unchanged (no provider)
	if got := buf.String(); got != "secret data here" {
		t.Errorf("got %q, want %q", got, "secret data here")
	}
}

// TestSecretProvider_MockProvider tests scrubber with mock provider
func TestSecretProvider_MockProvider(t *testing.T) {
	var buf bytes.Buffer
	provider := newMockProvider()
	provider.AddSecret("secret123", "opal:v:abc")
	
	s := New(&buf, WithSecretProvider(provider))
	
	input := []byte("The value is: secret123")
	s.Write(input)
	s.Flush()
	
	want := "The value is: opal:v:abc"
	if got := buf.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestSecretProvider_MultipleSecrets tests multiple secrets in output
func TestSecretProvider_MultipleSecrets(t *testing.T) {
	var buf bytes.Buffer
	provider := newMockProvider()
	provider.AddSecret("secret1", "opal:v:aaa")
	provider.AddSecret("secret2", "opal:v:bbb")
	
	s := New(&buf, WithSecretProvider(provider))
	
	input := []byte("First: secret1, Second: secret2")
	s.Write(input)
	s.Flush()
	
	got := buf.String()
	// Both secrets should be replaced
	if bytes.Contains([]byte(got), []byte("secret1")) {
		t.Errorf("secret1 not scrubbed: %q", got)
	}
	if bytes.Contains([]byte(got), []byte("secret2")) {
		t.Errorf("secret2 not scrubbed: %q", got)
	}
	if !bytes.Contains([]byte(got), []byte("opal:v:aaa")) {
		t.Errorf("placeholder opal:v:aaa not found: %q", got)
	}
	if !bytes.Contains([]byte(got), []byte("opal:v:bbb")) {
		t.Errorf("placeholder opal:v:bbb not found: %q", got)
	}
}

// TestSecretProvider_LongestMatch tests that longest secret wins
func TestSecretProvider_LongestMatch(t *testing.T) {
	var buf bytes.Buffer
	provider := newMockProvider()
	provider.AddSecret("SECRET", "opal:v:short")
	provider.AddSecret("SECRET_EXTENDED", "opal:v:long")
	
	s := New(&buf, WithSecretProvider(provider))
	
	input := []byte("Value: SECRET_EXTENDED")
	s.Write(input)
	s.Flush()
	
	got := buf.String()
	// Should use longest match
	if !bytes.Contains([]byte(got), []byte("opal:v:long")) {
		t.Errorf("longest match not used: %q", got)
	}
	if bytes.Contains([]byte(got), []byte("opal:v:short")) {
		t.Errorf("short match incorrectly used: %q", got)
	}
	if bytes.Contains([]byte(got), []byte("SECRET")) {
		t.Errorf("secret not scrubbed: %q", got)
	}
}

// TestSecretProvider_NoSecrets tests provider that finds no secrets
func TestSecretProvider_NoSecrets(t *testing.T) {
	var buf bytes.Buffer
	provider := newMockProvider()
	provider.AddSecret("secret123", "opal:v:abc")
	
	s := New(&buf, WithSecretProvider(provider))
	
	input := []byte("No secrets here")
	s.Write(input)
	s.Flush()
	
	// Should pass through unchanged (no secrets found)
	if got := buf.String(); got != "No secrets here" {
		t.Errorf("got %q, want %q", got, "No secrets here")
	}
}

// TestSecretProvider_ChunkBoundary tests secret split across writes
// NOTE: Provider-based scrubbing with chunk boundaries requires the carry buffer
// to query the provider. This is a known limitation that will be addressed when
// we fully remove the legacy RegisterSecret API.
func TestSecretProvider_ChunkBoundary(t *testing.T) {
	t.Skip("Provider-based chunk boundary handling not yet implemented - tracked in Phase 2.3")
	
	var buf bytes.Buffer
	provider := newMockProvider()
	provider.AddSecret("SECRET_TOKEN", "opal:v:xyz")
	
	s := New(&buf, WithSecretProvider(provider))
	
	// Split secret across two writes
	s.Write([]byte("Value: SECRET_"))
	s.Write([]byte("TOKEN here"))
	s.Flush()
	
	got := buf.String()
	// Secret should be scrubbed even though split
	if bytes.Contains([]byte(got), []byte("SECRET_TOKEN")) {
		t.Errorf("secret leaked across boundary: %q", got)
	}
	if !bytes.Contains([]byte(got), []byte("opal:v:xyz")) {
		t.Errorf("placeholder not found: %q", got)
	}
}
