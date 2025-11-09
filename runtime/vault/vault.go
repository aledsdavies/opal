package vault

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
)

// Vault is the single source of truth for secret tracking and management.
//
// # Responsibilities
//
// 1. Expression Tracking - Track ALL secret-producing expressions:
//   - Direct decorator calls: @env.HOME, @aws.secret("key")
//   - Variable declarations: var API_KEY = @aws.secret("key")
//   - Nested expressions: @aws.secret("@var.ENV-key")
//
// 2. Site Recording - Track WHERE each expression is used:
//   - Build canonical site paths: root/step-1/@shell[0]/params/command
//   - Generate unforgeable SiteIDs (HMAC-based)
//   - Create SecretUse entries for executor enforcement
//
// 3. Resolution - Resolve expressions by calling decorators:
//   - Wave-based: Planner marks which expressions are "touched" in execution path
//   - Only resolve touched expressions (prune unreachable code)
//   - Store resolved values directly (access control via SiteID + transport)
//   - Support meta-programming: Planner can unwrap for conditionals
//
// 4. Transport Boundaries - Enforce security boundaries:
//   - Track current transport scope (local, ssh:host, docker:container)
//   - Error if expression crosses transport boundary without explicit passing
//   - Example: Local @env.TOKEN cannot be used inside @ssh block
//
// 5. Pruning - Remove unused/unreachable expressions:
//   - Auto-prune: Expressions not marked "touched" are unreachable
//   - Auto-prune: Expressions with no references are unused
//   - BuildSecretUses only includes resolved + referenced expressions
//
// # Expression IDs
//
// Expression IDs must be deterministic and unique:
//   - Variables: Use variable name ("API_KEY")
//   - Direct calls: Hash of decorator + params ("@env.HOME" → "env:HOME")
//   - Nested: Hash of full expression ("@aws.secret('@var.ENV-key')" → "aws.secret:...")
//
// Deterministic IDs enable:
//   - Consistent tracking across planning phases
//   - Deduplication of identical expressions
//   - Reproducible SecretUse generation
//
// # Security Model: Zanzibar-Style Access Control
//
// ALL value decorators produce secrets:
//   - @var.X → secret
//   - @env.X → secret
//   - @aws.secret() → secret
//   - @vault.read() → secret
//
// No classification needed - if it's a value decorator, it's a secret.
//
// Access control uses Tuple (Position) + Caveat (Constraint):
//
// Tuple (Position): (exprID, siteID)
//   - Secret can only be accessed at authorized sites
//   - Site: "root/retry[0]/params/apiKey"
//   - SiteID: HMAC(planKey, site) - unforgeable
//   - Recorded via RecordReference() during planning
//
// Caveat (Constraint): Transport isolation (decorator-specific)
//   - Currently: @env expressions cannot cross transport boundaries
//   - Future: Any decorator can declare transport isolation requirement
//   - Prevents local secrets leaking to remote SSH sessions
//
// Both checks must pass for Access() to succeed.
//
// # Planner-Vault Collaboration
//
// # Planner-Vault Collaboration
//
// Planner orchestrates, Vault resolves and stores:
//
//	Pass 1 - Scan:
//	  exprID := vault.DeclareVariable(name, raw)  // Returns variable name as ID
//	  exprID := vault.TrackExpression(raw)        // Returns hash-based ID with transport
//	  vault.RecordReference(exprID, paramName)    // Record use-site
//
//	Pass 2 - Resolve (wave-based):
//	  vault.MarkTouched(exprID)                   // Mark in execution path
//	  vault.ResolveTouched(ctx)                   // Vault calls decorators
//	  value, _ := vault.Access(exprID, paramName) // Planner accesses for @if (checks SiteID + transport)
//
//	Pass 3 - Finalize:
//	  vault.PruneUntouched()                      // Remove unreachable
//	  uses := vault.BuildSecretUses()             // Generate authorization list
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
// Returns error if expression crosses transport boundary.
func (v *Vault) RecordReference(exprID, paramName string) error {
	// Check transport boundary
	if err := v.checkTransportBoundary(exprID); err != nil {
		return err
	}

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

// checkTransportBoundary checks if expression can be used in current transport.
func (v *Vault) checkTransportBoundary(exprID string) error {
	// Get transport where expression was resolved
	exprTransport, exists := v.exprTransport[exprID]
	if !exists {
		// Expression not resolved yet, record current transport for later
		v.exprTransport[exprID] = v.currentTransport
		return nil
	}

	// Check if crossing transport boundary
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
//   vault.EnterDecorator("@shell")
//   value, err := vault.Access("API_KEY", "command")  // Checks site: root/@shell[0]/params/command
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

	// 4. Check transport boundary
	if err := v.checkTransportBoundary(exprID); err != nil {
		return "", err
	}

	// 5. Return value
	return expr.Value, nil
}
