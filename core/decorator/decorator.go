package decorator

import "github.com/aledsdavies/opal/core/types"

// Role represents behavioral capabilities of a decorator.
// Decorators can have multiple roles (e.g., @aws.s3.object is both Provider and Endpoint).
// Roles are auto-inferred from implemented interfaces.
type Role string

const (
	// RoleProvider produces data (@var, @env, @aws.secret)
	RoleProvider Role = "provider"

	// RoleWrapper wraps execution (@retry, @timeout)
	RoleWrapper Role = "wrapper"

	// RoleBoundary creates scoped context (@ssh.connect, @docker.exec)
	RoleBoundary Role = "boundary"

	// RoleEndpoint reads/writes data (@file.read, @s3.put)
	RoleEndpoint Role = "endpoint"

	// RoleAnnotate augments plan metadata (@trace, @measure)
	RoleAnnotate Role = "annotate"
)

// Decorator is the base interface all decorators must implement.
// It provides reflectable metadata for LSP, CLI, docs, and telemetry.
type Decorator interface {
	Descriptor() Descriptor
}

// Descriptor holds rich metadata about a decorator.
// This is the single source of truth for validation, documentation, and tooling.
type Descriptor struct {
	// Path is the decorator's full path (e.g., "env", "retry", "aws.s3.object")
	Path string

	// Roles are behavioral capabilities (auto-inferred from implemented interfaces)
	// A decorator can have multiple roles (e.g., @aws.s3.object is both Provider and Endpoint)
	Roles []Role

	// Version is the decorator version (semver string)
	Version string

	// Summary is a one-line description
	Summary string

	// DocURL links to full documentation
	DocURL string

	// Schema describes parameters and return type (single source of truth)
	Schema types.DecoratorSchema

	// Capabilities define execution constraints and properties
	Capabilities Capabilities
}

// TransportScope defines where a decorator can be used.
type TransportScope int

const (
	// TransportScopeAny means decorator works in any transport (local, SSH, Docker, etc.)
	TransportScopeAny TransportScope = 0

	// TransportScopeLocal means decorator only works in local transport
	TransportScopeLocal TransportScope = 1

	// TransportScopeSSH means decorator only works in SSH transport
	TransportScopeSSH TransportScope = 2

	// TransportScopeRemote means decorator works in any remote transport (SSH, Docker, etc.)
	TransportScopeRemote TransportScope = 3
)

// String returns the string representation of TransportScope.
func (s TransportScope) String() string {
	switch s {
	case TransportScopeAny:
		return "Any"
	case TransportScopeLocal:
		return "Local"
	case TransportScopeSSH:
		return "SSH"
	case TransportScopeRemote:
		return "Remote"
	default:
		return "Unknown"
	}
}

// Allows checks if the decorator's transport scope allows execution in the current scope.
func (s TransportScope) Allows(current TransportScope) bool {
	// Any scope works everywhere
	if s == TransportScopeAny {
		return true
	}

	// Remote scope allows any remote transport (SSH, Docker, etc.)
	if s == TransportScopeRemote {
		return current == TransportScopeSSH || current == TransportScopeRemote
	}

	// Otherwise, exact match required
	return s == current
}

// Capabilities define execution constraints and properties.
type Capabilities struct {
	// TransportScope defines where this decorator can be used
	TransportScope TransportScope

	// Purity indicates if the decorator is deterministic (can be cached/constant-folded)
	// Default: false (safe default - assume side effects)
	Purity bool

	// Idempotent indicates if the decorator is safe to retry
	// Default: false (safe default - assume not idempotent)
	Idempotent bool

	// Block specifies whether decorator accepts/requires a block
	// Default: BlockForbidden (safe default for value decorators)
	Block BlockRequirement

	// IO describes I/O behavior for pipe and redirect operators
	IO IOSemantics
}

// BlockRequirement specifies whether a decorator accepts/requires a block
type BlockRequirement string

const (
	// BlockForbidden means decorator cannot have a block (value decorators like @var, @env)
	BlockForbidden BlockRequirement = "forbidden"

	// BlockOptional means decorator can optionally have a block (e.g., @retry with/without block)
	BlockOptional BlockRequirement = "optional"

	// BlockRequired means decorator must have a block (e.g., @parallel, @timeout)
	BlockRequired BlockRequirement = "required"
)

// IOSemantics describes I/O capabilities for decorators.
// This is a simplified v1.0 model - decorators are inherently concurrency-safe
// by design (pure/monadic), so no ConcurrentSafe flag is needed.
type IOSemantics struct {
	// PipeIn indicates decorator can read from stdin (supports: cmd | @decorator)
	PipeIn bool

	// PipeOut indicates decorator can write to stdout (supports: @decorator | cmd)
	PipeOut bool

	// RedirectIn indicates decorator can read from file (supports: @decorator < file)
	RedirectIn bool

	// RedirectOut indicates decorator can write to file (supports: cmd > @decorator)
	RedirectOut bool
}
