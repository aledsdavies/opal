package streamscrub

// SecretProvider identifies secrets in data streams without revealing them.
//
// Implementations detect secrets using any strategy (Vault, config, regex, etc.)
// and provide replacements without exposing the secret values to the scrubber.
//
// # Security Model
//
// The provider NEVER gives secret values to the scrubber proactively.
// Instead, the scrubber asks "is this chunk a secret?" for each piece of data.
// The provider answers with pattern + placeholder if found.
//
// This ensures:
//   - Scrubber only sees values that appear in output
//   - Provider controls what is considered a secret
//   - Minimal exposure (defense in depth)
//
// # Performance Considerations
//
// For optimal performance with Aho-Corasick or similar algorithms:
//   - Provider can build automaton from all known secrets
//   - FindSecret() runs automaton on chunk (O(n) scan)
//   - Returns longest match for overlapping secrets
//
// Current scrubber implementation uses simple linear search, which is sufficient
// for typical use cases (10-100 secrets). Future optimization can use
// Aho-Corasick without changing this interface.
//
// # Example Implementation
//
//	type VaultProvider struct {
//	    expressions map[string]*Expression
//	}
//
//	func (v *VaultProvider) FindSecret(chunk []byte) ([]byte, []byte, bool) {
//	    // Check all resolved expressions (longest-first)
//	    for _, expr := range v.sortedExpressions() {
//	        if bytes.Contains(chunk, []byte(expr.Value)) {
//	            return []byte(expr.Value), []byte(expr.DisplayID), true
//	        }
//	    }
//	    return nil, nil, false
//	}
type SecretProvider interface {
	// FindSecret finds the first (longest) secret in chunk.
	//
	// Returns:
	//   - pattern: The secret bytes to replace
	//   - placeholder: The replacement bytes (e.g., "opal:v:abc123")
	//   - found: true if secret found, false otherwise
	//
	// For multiple secrets in chunk, returns longest match (greedy).
	// Caller should loop until no more secrets found.
	//
	// Thread-safety: Implementations must be safe for concurrent calls.
	FindSecret(chunk []byte) (pattern []byte, placeholder []byte, found bool)
}
