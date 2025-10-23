package planfmt

// ExecutionNode represents a node in the operator precedence tree.
// This handles operator chaining within a single step.
//
// The tree structure captures operator precedence:
//
//	Precedence (high to low): | > redirect > && > || > ;
//
// Example: echo "a" | grep "a" > file.txt && echo "b" || echo "c"
//
//	Parsed as: (((echo "a" | grep "a") > file.txt) && echo "b") || echo "c"
type ExecutionNode interface {
	isExecutionNode()
}

// CommandNode is a leaf node - represents a single decorator invocation.
type CommandNode struct {
	Decorator string // "@shell", "@retry", "@parallel", etc.
	Args      []Arg  // Decorator arguments (sorted by Key)
	Block     []Step // Nested steps (for decorators with blocks)
}

func (*CommandNode) isExecutionNode() {}

// PipelineNode executes a chain of piped commands (cmd1 | cmd2 | cmd3).
// All commands run concurrently with stdout→stdin streaming.
type PipelineNode struct {
	Commands []CommandNode // All commands in the pipeline
}

func (*PipelineNode) isExecutionNode() {}

// AndNode executes left, then right only if left succeeded (exit 0).
// Implements bash && operator semantics.
type AndNode struct {
	Left  ExecutionNode
	Right ExecutionNode
}

func (*AndNode) isExecutionNode() {}

// OrNode executes left, then right only if left failed (exit != 0).
// Implements bash || operator semantics.
type OrNode struct {
	Left  ExecutionNode
	Right ExecutionNode
}

func (*OrNode) isExecutionNode() {}

// SequenceNode executes all nodes sequentially (semicolon operator).
// Always executes all nodes regardless of exit codes.
// Returns exit code of last node.
type SequenceNode struct {
	Nodes []ExecutionNode
}

func (*SequenceNode) isExecutionNode() {}

// RedirectMode specifies how to open the sink (overwrite or append).
type RedirectMode int

const (
	RedirectOverwrite RedirectMode = iota // > (truncate file)
	RedirectAppend                        // >> (append to file)
)

// RedirectSinkKind specifies what kind of sink target.
type RedirectSinkKind int

const (
	RedirectSinkPath      RedirectSinkKind = iota // Static file path
	RedirectSinkDecorator                         // Decorator sink (Phase 2)
)

// RedirectSink specifies where output goes.
type RedirectSink struct {
	Kind RedirectSinkKind

	// For Kind == RedirectSinkPath:
	Path string // File path (may contain variables like @var.OUTPUT_FILE)

	// For Kind == RedirectSinkDecorator (Phase 2):
	Decorator string // Decorator name (e.g., "@file.temp", "@aws.s3.object")
	Args      []Arg  // Decorator arguments
}

// RedirectNode redirects stdout from Source to Sink.
// Precedence: higher than &&, lower than |
//
// Examples:
//   - echo "hello" > output.txt
//   - cmd1 | cmd2 >> log.txt
//   - build > @file.temp() (Phase 2)
type RedirectNode struct {
	Source ExecutionNode // Command/pipeline producing output
	Sink   RedirectSink  // Where output goes
	Mode   RedirectMode  // Overwrite or Append
}

func (*RedirectNode) isExecutionNode() {}
