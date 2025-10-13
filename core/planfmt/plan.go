package planfmt

// Plan is the in-memory representation of an execution plan.
// This is the stable contract between planner, executor, and formatters.
type Plan struct {
	Header PlanHeader
	Target string          // Function/command being executed (e.g., "deploy")
	Steps  []ExecutionStep // Execution tree (all steps are decorators)
}

// PlanHeader contains metadata about the plan.
// Fields are designed for forward compatibility and versioning.
type PlanHeader struct {
	SchemaID  [16]byte // UUID for this format schema version
	CreatedAt uint64   // Unix nanoseconds (UTC)
	Compiler  [16]byte // Build/commit fingerprint
	PlanKind  uint8    // 0=view, 1=contract, 2=executed
	_reserved [13]byte // Reserved for future use
}

// ExecutionStep represents a single step in the execution tree.
// All steps are decorators (shell commands are @shell decorators).
type ExecutionStep struct {
	Decorator string                 // "@shell", "@retry", "@parallel", etc.
	Args      map[string]interface{} // Decorator arguments
	Block     []ExecutionStep        // Nested steps for decorators with blocks
}
