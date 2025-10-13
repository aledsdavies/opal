package planfmt

import "fmt"

// Plan is the in-memory representation of an execution plan.
// This is the stable contract between planner, executor, and formatters.
type Plan struct {
	Header PlanHeader
	Target string // Function/command being executed (e.g., "deploy")
	Root   *Step  // Root of execution tree (nil for empty plan)
}

// PlanHeader contains metadata about the plan.
// Fields are designed for forward compatibility and versioning.
// Total size: 44 bytes (fixed)
type PlanHeader struct {
	SchemaID  [16]byte // UUID for this format schema version
	CreatedAt uint64   // Unix nanoseconds (UTC)
	Compiler  [16]byte // Build/commit fingerprint
	PlanKind  uint8    // 0=view, 1=contract, 2=executed
	_reserved [3]byte  // Reserved for future use (align to 8 bytes)
}

// StepKind identifies the type of step
type StepKind uint8

const (
	KindDecorator StepKind = iota // Execution decorator (@shell, @retry, @parallel)
	KindTryCatch                  // Runtime branching (try/catch/finally)
)

// Step represents a single step in the execution tree.
// Invariants:
// - ID must be unique within a plan
// - Args must be sorted by Key (for determinism)
// - Children order is semantically significant
type Step struct {
	ID       uint64   // Unique identifier (stable across plan versions)
	Kind     StepKind // Decorator or TryCatch
	Op       string   // "shell", "retry", "parallel", "try", "catch", "finally"
	Args     []Arg    // Sorted by Key for deterministic encoding
	Children []*Step  // Nested steps (order matters)
}

// Arg represents a typed argument to a decorator.
// Args are sorted by Key to ensure deterministic encoding.
type Arg struct {
	Key string
	Val Value
}

// Value is a union type for decorator arguments.
// Only one field should be set based on Kind.
type Value struct {
	Kind ValueKind

	// Union fields (only one valid per Kind)
	Str  string // For ValueString
	Int  int64  // For ValueInt
	Bool bool   // For ValueBool
	Ref  uint32 // For ValuePlaceholder (index into placeholder table)
}

// ValueKind identifies which field in Value is valid
type ValueKind uint8

const (
	ValueString      ValueKind = iota // Str field valid
	ValueInt                          // Int field valid
	ValueBool                         // Bool field valid
	ValuePlaceholder                  // Ref field valid (placeholder table index)
)

// Validate checks plan invariants
func (p *Plan) Validate() error {
	if p.Root == nil {
		return nil // Empty plan is valid
	}

	// Check for duplicate step IDs
	seen := make(map[uint64]bool)
	return p.Root.validate(seen)
}

// validate checks step invariants recursively
func (s *Step) validate(seen map[uint64]bool) error {
	// Check ID uniqueness
	if seen[s.ID] {
		return fmt.Errorf("duplicate step ID: %d", s.ID)
	}
	seen[s.ID] = true

	// Check args are sorted
	for i := 1; i < len(s.Args); i++ {
		if s.Args[i-1].Key >= s.Args[i].Key {
			return fmt.Errorf("step %d: args not sorted (key %q >= %q)",
				s.ID, s.Args[i-1].Key, s.Args[i].Key)
		}
	}

	// Validate children recursively
	for _, child := range s.Children {
		if err := child.validate(seen); err != nil {
			return err
		}
	}

	return nil
}
