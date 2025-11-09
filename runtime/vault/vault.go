package vault

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/aledsdavies/opal/core/invariant"
)

// Vault tracks secret-producing expressions and enforces site-based access control.
//
// In Opal's security model, ALL value decorators produce secrets:
//   - @var.X, @env.X, @aws.secret(), @vault.read() are all secrets
//   - No classification needed - if it's a value decorator, it's a secret
//
// # Usage: Three-Pass Planning
//
// The planner uses Vault in three passes to track, resolve, and authorize secrets:
//
//	// Pass 1: Scan - Track all expressions and their use-sites
//	vault := vault.NewWithPlanKey(planKey)
//
//	vault.EnterStep()
//	vault.EnterDecorator("@shell")
//
//	// Track variable declaration (returns variable name as ID)
//	apiKeyID := vault.DeclareVariable("API_KEY", "@env.API_KEY")
//
//	// Track direct decorator call (returns hash-based ID with transport)
//	homeID := vault.TrackExpression("@env.HOME")
//
//	// Record where expression is used (builds site path from current position)
//	vault.RecordReference(apiKeyID, "command")  // Site: root/step-1/@shell[0]/params/command
//
//	// Pass 2: Resolve - Wave-based resolution for touched expressions
//	vault.MarkTouched(apiKeyID)  // Mark as in execution path
//
//	// Planner resolves touched expressions (calls decorators)
//	vault.MarkResolved(apiKeyID, "ghp_abc123")  // Captures transport at resolution time
//
//	// Access for meta-programming (checks SiteID + transport boundary)
//	value, err := vault.Access(apiKeyID, "command")  // Returns "ghp_abc123"
//
//	// Pass 3: Finalize - Prune and build authorization list
//	vault.PruneUntouched()  // Remove unreachable expressions
//	uses := vault.BuildSecretUses()  // Generate SecretUse entries for executor
//
// # Access Control: Site-Based Authority
//
// Vault enforces two security checks via Access():
//
//  1. Transport Boundary (Caveat): Expression cannot cross transport boundaries
//     - @env expressions resolved in "local" cannot be used in "ssh:host"
//     - Prevents local secrets leaking to remote sessions
//     - Checked first (more fundamental security rule)
//
//  2. Site Authorization (Tuple): Expression can only be accessed at authorized sites
//     - Site: "root/retry[0]/params/apiKey" (canonical path)
//     - SiteID: HMAC(planKey, site) - unforgeable, cannot be forged
//     - Recorded via RecordReference() during Pass 1
//
// Both checks must pass for Access() to succeed.
//
// # Expression IDs: Deterministic and Transport-Aware
//
// Expression IDs must be unique and include transport context:
//
//   - Variables: Use variable name ("API_KEY")
//   - Direct calls: Hash of transport + raw expression
//   - @env.HOME in local → "local:abc123..."
//   - @env.HOME in ssh:server1 → "ssh:server1:def456..."
//
// Same decorator in different transports produces different IDs because
// they resolve to different values.
//
// # Path Tracking
//
// Vault maintains a path stack to build canonical site paths:
//
//	vault.EnterStep()           // root/step-1
//	vault.EnterDecorator("@retry")  // root/step-1/@retry[0]
//	vault.EnterDecorator("@shell")  // root/step-1/@retry[0]/@shell[0]
//
//	site := vault.BuildSitePath("command")
//	// Returns: "root/step-1/@retry[0]/@shell[0]/params/command"
//
// Multiple instances of the same decorator get different indices:
//
//	vault.EnterDecorator("@shell")  // @shell[0]
//	vault.ExitDecorator()
//	vault.EnterDecorator("@shell")  // @shell[1]
//
// # Transport Boundaries
//
// Vault tracks transport scope to prevent secret leakage:
//
//	vault.EnterTransport("ssh:untrusted")  // Entering SSH session
//
//	// This will fail - LOCAL_TOKEN resolved in "local" transport
//	err := vault.Access("LOCAL_TOKEN", "command")
//	// Error: "transport boundary violation: expression resolved in local, cannot use in ssh:untrusted"
//
// Explicit passing via decorator parameters is allowed (handled by planner).
//
// # Security Properties
//
//   - Site-based authority: Secrets only accessible at authorized sites
//   - Unforgeable SiteIDs: HMAC-based, cannot be forged without planKey
//   - Transport isolation: Secrets cannot cross transport boundaries
//   - No propagation: Parent/child decorators cannot access each other's secrets
//   - Automatic pruning: Unused and unreachable expressions removed
type Vault struct {
	// Path tracking (DAG traversal)
	pathStack       []PathSegment
	stepCount       int
	decoratorCounts map[string]int // Decorator instance counts at current level

	// Expression tracking
	expressions map[string]*Expression // exprID → Expression
	references  map[string][]SiteRef   // exprID → sites that use it
	touched     map[string]bool        // exprID → in execution path

	// Transport boundary tracking
	currentTransport string            // Current transport scope
	exprTransport    map[string]string // exprID → transport where resolved

	// Security
	planKey []byte // For HMAC-based SiteIDs
}

// Expression represents a secret-producing expression.
// In our security model: ALL expressions are secrets.
type Expression struct {
	Raw       string // Original source: "@var.X", "@aws.secret('key')", etc.
	Value     string // Resolved value (can be empty string - check Resolved flag)
	DisplayID string // Placeholder ID for plan (e.g., "opal:v:3J98t56A")
	Resolved  bool   // True if expression has been resolved (even if Value is "")
}

// Note: No ExprType, no IsSecret - everything is a secret.
// Vault stores raw values directly - access control via SiteID + transport checks.

// SiteRef represents a reference to an expression at a specific site.
type SiteRef struct {
	Site      string // "root/step-1/@shell[0]/params/command"
	SiteID    string // HMAC-based unforgeable ID
	ParamName string // "command", "apiKey", etc.
}

// PathSegment represents one level in the decorator DAG path.
type PathSegment struct {
	Type  SegmentType
	Name  string
	Index int // -1 if not applicable
}

// SegmentType identifies the type of path segment.
type SegmentType int

const (
	SegmentRoot SegmentType = iota
	SegmentStep
	SegmentDecorator
)

// New creates a new Vault.
func New() *Vault {
	return &Vault{
		pathStack:        []PathSegment{{Type: SegmentRoot, Name: "root", Index: -1}},
		stepCount:        0,
		decoratorCounts:  make(map[string]int),
		expressions:      make(map[string]*Expression),
		references:       make(map[string][]SiteRef),
		touched:          make(map[string]bool),
		currentTransport: "local",
		exprTransport:    make(map[string]string),
	}
}

// NewWithPlanKey creates a new Vault with a specific plan key for HMAC-based SiteIDs.
func NewWithPlanKey(planKey []byte) *Vault {
	v := New()
	v.planKey = planKey
	return v
}

// EnterStep pushes a new step onto the path stack and resets decorator counts.
// If there's already a step in the stack, it's replaced (steps don't nest).
func (v *Vault) EnterStep() {
	v.stepCount++
	stepID := fmt.Sprintf("step-%d", v.stepCount)

	// Pop previous step if exists (steps are siblings, not nested)
	if len(v.pathStack) > 1 && v.pathStack[len(v.pathStack)-1].Type == SegmentStep {
		v.pathStack = v.pathStack[:len(v.pathStack)-1]
	}

	v.pathStack = append(v.pathStack, PathSegment{
		Type:  SegmentStep,
		Name:  stepID,
		Index: -1,
	})

	// Reset decorator counts for new step
	v.decoratorCounts = make(map[string]int)
}

// EnterDecorator pushes a decorator onto the path stack and returns its index.
func (v *Vault) EnterDecorator(decorator string) int {
	// Get next instance index for this decorator at current level
	index := v.decoratorCounts[decorator]
	v.decoratorCounts[decorator]++

	v.pathStack = append(v.pathStack, PathSegment{
		Type:  SegmentDecorator,
		Name:  decorator,
		Index: index,
	})

	return index
}

// ExitDecorator pops the current decorator from the path stack.
func (v *Vault) ExitDecorator() {
	if len(v.pathStack) <= 1 {
		panic("cannot exit root")
	}

	// Only pop if top is a decorator
	if v.pathStack[len(v.pathStack)-1].Type == SegmentDecorator {
		v.pathStack = v.pathStack[:len(v.pathStack)-1]
	}
}

// BuildSitePath constructs the canonical site path for a parameter.
// Format: root/step-N/@decorator[index]/params/paramName
func (v *Vault) BuildSitePath(paramName string) string {
	var parts []string

	for _, seg := range v.pathStack {
		switch seg.Type {
		case SegmentRoot:
			parts = append(parts, seg.Name)
		case SegmentStep:
			parts = append(parts, seg.Name)
		case SegmentDecorator:
			// Decorator with index: @shell[0]
			parts = append(parts, fmt.Sprintf("%s[%d]", seg.Name, seg.Index))
		}
	}

	// Add parameter path
	parts = append(parts, "params", paramName)

	return strings.Join(parts, "/")
}

// DeclareVariable registers a variable declaration.
// Returns the variable name as the expression ID.
func (v *Vault) DeclareVariable(name, raw string) string {
	v.expressions[name] = &Expression{
		Raw: raw,
	}
	return name
}

// TrackExpression registers a direct decorator call (e.g., @env.HOME).
// Returns a deterministic hash-based ID that includes transport context.
// Format: "transport:hash"
func (v *Vault) TrackExpression(raw string) string {
	// Generate deterministic ID including transport
	exprID := v.generateExprID(raw)

	// Store expression if not already tracked
	if _, exists := v.expressions[exprID]; !exists {
		v.expressions[exprID] = &Expression{
			Raw: raw,
		}
	}

	return exprID
}

// generateExprID creates a deterministic expression ID including transport context.
func (v *Vault) generateExprID(raw string) string {
	// Include current transport for context-sensitive IDs
	h := sha256.New()
	h.Write([]byte(v.currentTransport))
	h.Write([]byte(":"))
	h.Write([]byte(raw))
	hash := h.Sum(nil)

	// Format: "transport:hash"
	// Use first 8 bytes of hash for reasonable ID length
	hashStr := fmt.Sprintf("%x", hash[:8])
	return fmt.Sprintf("%s:%s", v.currentTransport, hashStr)
}

// RecordReference records that an expression is used at the current site.
// Transport boundary check is deferred to Access() time (after resolution).
func (v *Vault) RecordReference(exprID, paramName string) error {
	site := v.BuildSitePath(paramName)
	siteID := v.computeSiteID(site)

	v.references[exprID] = append(v.references[exprID], SiteRef{
		Site:      site,
		SiteID:    siteID,
		ParamName: paramName,
	})

	return nil
}

// computeSiteID generates an unforgeable site identifier using HMAC.
func (v *Vault) computeSiteID(canonicalPath string) string {
	if len(v.planKey) == 0 {
		// No plan key set - return empty (tests without security)
		return ""
	}

	h := hmac.New(sha256.New, v.planKey)
	h.Write([]byte(canonicalPath))
	mac := h.Sum(nil)

	// Truncate to 16 bytes and base64 encode
	return base64.RawURLEncoding.EncodeToString(mac[:16])
}

// PruneUnused removes expressions that have no site references.
// This eliminates variables that were declared but never used.
func (v *Vault) PruneUnused() {
	for id := range v.expressions {
		if len(v.references[id]) == 0 {
			delete(v.expressions, id)
			delete(v.references, id)
			delete(v.touched, id)
			delete(v.exprTransport, id)
		}
	}
}

// BuildSecretUses constructs the final SecretUse list for the plan.
// Auto-prunes: Only includes expressions that:
// 1. Have been resolved (Resolved flag is true) - unresolved are skipped
// 2. Have at least one site reference - unreferenced are skipped
// 3. Are marked as touched - untouched are skipped
//
// In our security model: ALL value decorators are secrets.
// Note: Empty string values are valid secrets (e.g., empty env vars).
func (v *Vault) BuildSecretUses() []SecretUse {
	var uses []SecretUse

	for id, expr := range v.expressions {
		// Auto-prune: Skip unresolved expressions (check Resolved flag, not Value)
		if !expr.Resolved {
			continue
		}

		// Auto-prune: Skip expressions with no references (unused)
		refs := v.references[id]
		if len(refs) == 0 {
			continue
		}

		// Auto-prune: Skip untouched expressions (not in execution path)
		if !v.touched[id] {
			continue
		}

		// Build SecretUse for each reference site
		for _, ref := range refs {
			uses = append(uses, SecretUse{
				DisplayID: expr.DisplayID,
				SiteID:    ref.SiteID,
				Site:      ref.Site,
			})
		}
	}

	return uses
}

// SecretUse represents an authorized secret usage at a specific site.
// This is what gets added to the Plan for executor enforcement.
type SecretUse struct {
	DisplayID string // "opal:v:3J98t56A"
	SiteID    string // HMAC-based unforgeable ID
	Site      string // "root/step-1/@shell[0]/params/command" (diagnostic)
}

// MarkTouched marks an expression as touched (in execution path).
func (v *Vault) MarkTouched(exprID string) {
	v.touched[exprID] = true
}

// IsTouched checks if an expression is marked as touched.
func (v *Vault) IsTouched(exprID string) bool {
	return v.touched[exprID]
}

// PruneUntouched removes expressions not in execution path.
func (v *Vault) PruneUntouched() {
	for id := range v.expressions {
		if !v.touched[id] {
			delete(v.expressions, id)
			delete(v.references, id)
			delete(v.touched, id)
			delete(v.exprTransport, id)
		}
	}
}

// EnterTransport enters a new transport scope.
func (v *Vault) EnterTransport(scope string) {
	v.currentTransport = scope
}

// ExitTransport exits current transport scope (returns to local).
func (v *Vault) ExitTransport() {
	v.currentTransport = "local"
}

// CurrentTransport returns the current transport scope.
func (v *Vault) CurrentTransport() string {
	return v.currentTransport
}

// MarkResolved marks an expression as resolved and captures its transport.
// This MUST be called when an expression is resolved (e.g., @env.HOME → "/home/user").
// The transport is captured at resolution time, not at first reference.
//
// This is critical for security: transport must be set when the value is resolved,
// not when it's first accessed. Otherwise, a local @env secret could be first
// accessed in an @ssh block, incorrectly capturing the transport as "ssh:*".
//
// Panics if expression not found or already resolved (programmer errors).
func (v *Vault) MarkResolved(exprID, value string) {
	expr, exists := v.expressions[exprID]
	invariant.Precondition(exists, "MarkResolved: expression %q not found", exprID)
	invariant.Precondition(!expr.Resolved, "MarkResolved: expression %q already resolved", exprID)

	expr.Value = value
	expr.Resolved = true
	v.exprTransport[exprID] = v.currentTransport // CRITICAL: Capture transport NOW
}

// checkTransportBoundary checks if expression can be used in current transport.
func (v *Vault) checkTransportBoundary(exprID string) error {
	// Get transport where expression was resolved
	exprTransport, exists := v.exprTransport[exprID]

	// CRITICAL: This should NEVER happen in production!
	// If it does, it means MarkResolved() wasn't called (programmer error).
	invariant.Invariant(exists,
		"expression %q has no transport recorded (MarkResolved not called?)",
		exprID)

	// Check if crossing transport boundary (legitimate security check - return error)
	if exprTransport != v.currentTransport {
		return fmt.Errorf(
			"transport boundary violation: expression %q resolved in %q, cannot use in %q",
			exprID, exprTransport, v.currentTransport,
		)
	}

	return nil
}

// Access returns the raw value for an expression at the current site.
//
// Implements Zanzibar-style access control:
//   - Tuple (Position): Checks if (exprID, siteID) is authorized
//   - Caveat (Constraint): Checks transport boundary (if decorator requires it)
//
// Used by planner for meta-programming (e.g., @if conditionals, @for loops).
//
// Parameters:
//   - exprID: Expression identifier (from DeclareVariable or TrackExpression)
//   - paramName: Parameter name accessing the value (e.g., "command", "apiKey")
//
// Returns:
//   - Resolved value if both checks pass
//   - Error if expression not found, not resolved, unauthorized site, or transport violation
//
// Example:
//
//	vault.EnterDecorator("@shell")
//	value, err := vault.Access("API_KEY", "command")  // Checks site: root/@shell[0]/params/command
func (v *Vault) Access(exprID, paramName string) (string, error) {
	// 1. Get expression
	expr, exists := v.expressions[exprID]
	if !exists {
		return "", fmt.Errorf("expression %q not found", exprID)
	}
	if !expr.Resolved {
		return "", fmt.Errorf("expression %q not resolved yet", exprID)
	}

	// 2. Check transport boundary (Caveat - checked first as more fundamental)
	if err := v.checkTransportBoundary(exprID); err != nil {
		return "", err
	}

	// 3. Build current site with parameter name
	currentSite := v.BuildSitePath(paramName)
	currentSiteID := v.computeSiteID(currentSite)

	// 4. Check if current site is authorized (Tuple)
	authorized := false
	for _, ref := range v.references[exprID] {
		if ref.SiteID == currentSiteID {
			authorized = true
			break
		}
	}
	if !authorized {
		return "", fmt.Errorf("no authority to unwrap %q at site %q", exprID, currentSite)
	}

	// 5. Return value
	return expr.Value, nil
}
