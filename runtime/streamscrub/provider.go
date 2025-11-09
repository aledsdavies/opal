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
//	    mu          sync.Mutex
//	    expressions map[string]*Expression
//	}
//
//	func (v *VaultProvider) FindSecret(chunk []byte) ([]byte, []byte, bool) {
//	    v.mu.Lock()
//	    defer v.mu.Unlock()
//	    
//	    // Build list of resolved expressions
//	    var secrets []struct {
//	        value       []byte
//	        placeholder []byte
//	    }
//	    for _, expr := range v.expressions {
//	        if expr.Resolved && expr.Value != "" {
//	            secrets = append(secrets, struct{value, placeholder []byte}{
//	                value:       []byte(expr.Value),
//	                placeholder: []byte(expr.DisplayID),
//	            })
//	        }
//	    }
//	    
//	    // Sort by descending length (longest first)
//	    sort.Slice(secrets, func(i, j int) bool {
//	        return len(secrets[i].value) > len(secrets[j].value)
//	    })
//	    
//	    // Check longest first
//	    for _, secret := range secrets {
//	        if bytes.Contains(chunk, secret.value) {
//	            return secret.value, secret.placeholder, true
//	        }
//	    }
//	    
//	    return nil, nil, false
//	}
type SecretProvider interface {
	// FindSecret finds the longest secret in chunk.
	//
	// Returns:
	//   - pattern: The secret bytes to replace
	//   - placeholder: The replacement bytes (e.g., "opal:v:abc123")
	//   - found: true if secret found, false otherwise
	//
	// # Implementation Requirements
	//
	// 1. Longest-match (greedy): When multiple secrets match, return the longest.
	//    This prevents partial leakage when secrets overlap.
	//    Example: If chunk contains "SECRET_EXTENDED" and you know both
	//    "SECRET" and "SECRET_EXTENDED", return "SECRET_EXTENDED".
	//
	// 2. Single secret per call: Return only ONE secret per call.
	//    Caller loops until no more secrets found to handle multiple
	//    non-overlapping secrets in the same chunk.
	//
	// 3. Thread-safety: Must be safe for concurrent calls.
	//
	// # Implementation Pattern
	//
	// Sort secrets by descending length, check longest first:
	//
	//	func (p *Provider) FindSecret(chunk []byte) ([]byte, []byte, bool) {
	//	    // Sort secrets by length (longest first)
	//	    sorted := p.sortSecretsByLength()
	//	    
	//	    // Check longest first
	//	    for _, secret := range sorted {
	//	        if bytes.Contains(chunk, secret.pattern) {
	//	            return secret.pattern, secret.placeholder, true
	//	        }
	//	    }
	//	    return nil, nil, false
	//	}
	FindSecret(chunk []byte) (pattern []byte, placeholder []byte, found bool)
}
