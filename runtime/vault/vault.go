package vault

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
)

// Vault is an in-memory secret and variable manager that tracks decorator
// DAG traversal, enforces transport boundaries, and enables wave-based
// batch resolution.
//
// Replaces ScopeGraph with enhanced security and path tracking.
type Vault struct {
	// Path tracking (DAG traversal)
	pathStack       []PathSegment
	stepCount       int
	decoratorCounts map[string]int // Decorator instance counts at current level

	// Expression tracking
	expressions map[string]*Expression // varName/exprID → Expression
	references  map[string][]SiteRef   // varName/exprID → sites that use it

	// Security
	planKey []byte // For HMAC-based SiteIDs
}

// ExprType identifies the type of expression.
type ExprType int

const (
	ExprVariable  ExprType = iota // @var.X
	ExprDecorator                 // @aws.secret(), @env.X, @vault.read()
	ExprNested                    // @var.X where X = @aws.secret()
)

// Expression represents a secret-producing expression.
type Expression struct {
	Raw       string   // Original expression: "sk-secret", "@aws.secret('key')"
	Type      ExprType // Variable, Decorator, Nested
	Resolved  bool     // Has this been resolved?
	IsSecret  bool     // Is this a secret? (determined after resolution)
	DisplayID string   // Placeholder ID (set after resolution if secret)
}

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
		pathStack:       []PathSegment{{Type: SegmentRoot, Name: "root", Index: -1}},
		stepCount:       0,
		decoratorCounts: make(map[string]int),
		expressions:     make(map[string]*Expression),
		references:      make(map[string][]SiteRef),
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
func (v *Vault) DeclareVariable(name, value string) {
	v.expressions[name] = &Expression{
		Raw:  value,
		Type: ExprVariable,
	}
}

// TrackExpression registers an expression (e.g., @aws.secret() call).
func (v *Vault) TrackExpression(id, raw string, typ ExprType) {
	v.expressions[id] = &Expression{
		Raw:  raw,
		Type: typ,
	}
}

// RecordExpressionReference records that an expression is used at the current site.
func (v *Vault) RecordExpressionReference(exprID, paramName string) {
	site := v.BuildSitePath(paramName)
	siteID := v.computeSiteID(site)

	v.references[exprID] = append(v.references[exprID], SiteRef{
		Site:      site,
		SiteID:    siteID,
		ParamName: paramName,
	})
}

// GetExpression retrieves an expression by ID.
func (v *Vault) GetExpression(id string) (*Expression, bool) {
	expr, exists := v.expressions[id]
	return expr, exists
}

// GetReferences retrieves all site references for an expression.
func (v *Vault) GetReferences(exprID string) []SiteRef {
	return v.references[exprID]
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
