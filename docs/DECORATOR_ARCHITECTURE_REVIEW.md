---
title: "OPAL Decorator Architecture Review"
status: "Final Analysis"
date: "2025-10-25"
---

# OPAL Decorator Architecture Review

**Comprehensive architectural analysis of OPAL's decorator interfaces, registry system, and schema design.**

**Executive Summary:** OPAL's decorator architecture demonstrates strong foundational design with clear separation of concerns, solid type safety, and extensibility patterns. The system is well-positioned for future requirements with minor enhancements needed in specific areas.

---

## 1. Core Decorator Interfaces Analysis

### 1.1 ValueHandler Interface

**Location:** `/home/user/opal/core/types/registry.go` (lines 33-35)

```go
// ValueHandler is a function that implements a value decorator
// Returns data with no side effects (can be interpolated in strings)
type ValueHandler func(ctx Context, args Args) (Value, error)
```

**Strengths:**
- ✅ Simple, functional signature that encourages pure functions
- ✅ Returns `Value` interface{} allowing any type (string, int, struct, etc.)
- ✅ Error handling built-in for graceful degradation
- ✅ Context parameter provides access to runtime environment

**Current Capabilities:**
- Primary and named parameters via `Args` struct
- Environment variable access through `Context.Env`
- Working directory context via `Context.WorkingDir`
- Variable bindings through `Context.Variables`

**Example Implementation:**
```go
// @env decorator (lines 13-34 of env.go)
func (e *envDecorator) Handle(ctx types.Context, args types.Args) (types.Value, error) {
    if args.Primary == nil {
        return nil, fmt.Errorf("@env requires an environment variable name")
    }
    envVar := (*args.Primary).(string)
    value, exists := ctx.Env[envVar]
    if !exists {
        if args.Params != nil {
            if defaultVal, hasDefault := args.Params["default"]; hasDefault {
                return defaultVal, nil
            }
        }
        return nil, fmt.Errorf("environment variable %q not found", envVar)
    }
    return value, nil
}
```

### 1.2 ExecutionHandler Interface

**Location:** `/home/user/opal/core/types/registry.go` (lines 37-39)

```go
// ExecutionHandler is a function that implements an execution decorator
// Performs actions with side effects
type ExecutionHandler func(ctx Context, args Args) error
```

**Strengths:**
- ✅ Simple error-based return signature
- ✅ Same `Context` and `Args` as value decorators (consistency)
- ✅ Suitable for imperative, side-effect-driven operations

**Limitations:**
- ⚠️ No block parameter support in signature
- ⚠️ No exit code return for command-like decorators
- ⚠️ SDK handlers use different signature (see section 1.4)

**Note:** This interface is deprecated in favor of SDK handlers (lines 254-275).

### 1.3 Context Structure

**Location:** `/home/user/opal/core/types/registry.go` (lines 19-24)

```go
type Context struct {
    Variables  map[string]Value  // Variable bindings: var x = "value"
    Env        map[string]string // Environment variables
    WorkingDir string            // Current working directory
}
```

**Strengths:**
- ✅ Clean separation of concerns (variables, environment, working directory)
- ✅ Immutable-friendly (maps should be copied before modification)
- ✅ Sufficient for plan-time value decorator execution

**Gaps for Future Requirements:**

| Future Requirement | Gap | Impact |
|-------------------|-----|--------|
| Execution-time let bindings (OEP-001) | No runtime value bindings | Values captured at execution need separate storage |
| Parallel isolation (OEP-001) | No scope/branch context | Parallel execution needs isolated `@let` contexts |
| Execution metadata | Missing step ID, run ID | Decorators can't trace their own execution |
| Transport context switching (OEP-010) | No transport metadata | @ssh.connect, @docker.exec need context switching |
| Stdin/stdout piping | No IO streams | Pipe operator needs stdin/stdout access |

**Recommendation 1.3.1:** Extend `Context` structure for OEP-001 and OEP-010:

```go
type Context struct {
    // Existing fields
    Variables  map[string]Value
    Env        map[string]string
    WorkingDir string
    
    // New fields for execution-time features
    LetBindings map[string]Value     // Runtime bindings (@let) - OEP-001
    Scope       ScopeContext         // Block scope metadata
    ExecutionID string              // Unique execution run ID
    StepID      string              // Current step ID for tracing
    Transport   TransportContext     // Transport switching (SSH, Docker) - OEP-010
}

type ScopeContext struct {
    BranchID string  // For @parallel isolation (OEP-001)
    Depth    int     // Nesting depth
}

type TransportContext struct {
    Type     string              // "local", "ssh", "docker", "k8s"
    Config   map[string]interface{}
}
```

### 1.4 Args Structure

**Location:** `/home/user/opal/core/types/registry.go` (lines 26-31)

```go
type Args struct {
    Primary *Value           // Primary property: @env.HOME → "HOME"
    Params  map[string]Value // Named parameters: (default="", times=3)
    Block   *Block           // Lambda/block for execution decorators
}
```

**Strengths:**
- ✅ Clean representation of three parameter types
- ✅ Primary as pointer allows nil checks
- ✅ Block as *Block interface allows both presence and execution
- ✅ Schema validation ensures parameter correctness before handler invocation

**Example Usage (from env.go lines 13-34):**
```go
// Uses Primary: @env.HOME extracts primary as "HOME"
if args.Primary == nil {
    return nil, fmt.Errorf("@env requires an environment variable name")
}

// Uses Params: (default="fallback") provides named parameters
if args.Params != nil {
    if defaultVal, hasDefault := args.Params["default"]; hasDefault {
        return defaultVal, nil
    }
}

// Block would be used in execution decorators for @retry { ... }
```

**Limitations:**
- ⚠️ No parameter validation inside handler (assumes schema validation is done)
- ⚠️ Value type erasure (all params are `interface{}`, requires type assertions)
- ⚠️ No structured access for complex types (OEP-002 requires this)

**Recommendation 1.4.1:** Add type-safe parameter access helpers:

```go
type Args struct {
    Primary *Value
    Params  map[string]Value
    Block   *Block
}

// Helper methods for type-safe access
func (a *Args) GetString(name string) (string, error) {
    if v, ok := a.Params[name]; ok {
        if s, ok := v.(string); ok {
            return s, nil
        }
        return "", fmt.Errorf("parameter %q is not a string", name)
    }
    return "", fmt.Errorf("parameter %q not found", name)
}

func (a *Args) GetInt(name string) (int, error) { /* ... */ }
func (a *Args) GetDuration(name string) (time.Duration, error) { /* ... */ }
func (a *Args) GetBool(name string) (bool, error) { /* ... */ }
```

---

## 2. Registry System Architecture

### 2.1 Registration API

**Location:** `/home/user/opal/core/types/registry.go` (lines 62-323)

**Three Registration Patterns:**

#### Pattern 1: Simple Registration (Deprecated)
```go
// Lines 74-85 (RegisterValue)
func (r *Registry) RegisterValue(path string, handler ValueHandler) {
    r.decorators[path] = DecoratorInfo{
        Path:         path,
        Kind:         DecoratorKindValue,
        Schema:       DecoratorSchema{Path: path, Kind: "value"},
        ValueHandler: handler,
    }
}
```

**Strengths:** ✅ Minimal boilerplate, good for quick testing

**Weaknesses:** ❌ No schema validation, minimal metadata

#### Pattern 2: Schema-Based Registration (Current Standard)
```go
// Lines 195-204 (RegisterValueWithSchema)
func (r *Registry) RegisterValueWithSchema(schema DecoratorSchema, handler ValueHandler) error {
    return r.registerValueWithSchemaAndInstance(schema, handler, handler)
}

func (r *Registry) RegisterValueDecoratorWithSchema(
    schema DecoratorSchema, 
    instance interface{}, 
    handler ValueHandler
) error {
    return r.registerValueWithSchemaAndInstance(schema, handler, instance)
}
```

**Strengths:**
- ✅ Schema validation (lines 209-211)
- ✅ Interface checking support via instance parameter
- ✅ Full metadata storage in `DecoratorInfo`

**Usage Example (env.go lines 42-66):**
```go
schema := types.NewSchema("env", types.KindValue).
    Description("Access environment variables").
    PrimaryParam("property", types.TypeString, "Environment variable name").
    Param("default", types.TypeString).
    Description("Default value if environment variable is not set").
    Optional().
    Examples("", "/home/user", "us-east-1").
    Done().
    Returns(types.TypeString, "Value of the environment variable").
    Build()

decorator := &envDecorator{}
if err := types.Global().RegisterValueDecoratorWithSchema(schema, decorator, decorator.Handle); err != nil {
    panic(fmt.Sprintf("failed to register @env decorator: %v", err))
}
```

#### Pattern 3: SDK Handler Registration (New Standard)
```go
// Lines 260-275 (RegisterSDKHandler / RegisterSDKHandlerWithSchema)
func (r *Registry) RegisterSDKHandlerWithSchema(schema DecoratorSchema, handler interface{}) error {
    if err := ValidateSchema(schema); err != nil {
        return fmt.Errorf("invalid schema for %s: %w", schema.Path, err)
    }
    // Stores handler as generic interface{} - caller must type-assert
}
```

**Strengths:**
- ✅ Supports both WASM and native handlers
- ✅ Avoids circular dependencies (core/types doesn't import core/sdk)
- ✅ Flexible handler type (depends on decorator kind)

**Design Consideration:** Uses `interface{}` to avoid circular dependency between core/types and core/sdk. This is pragmatic but trades some type safety.

### 2.2 Lookup Mechanisms

**Location:** `/home/user/opal/core/types/registry.go` (lines 100-160)

**Strengths:**
- ✅ Thread-safe RWMutex protection (line 63: `mu sync.RWMutex`)
- ✅ Multiple lookup strategies: handler type, schema, info, SDK handler
- ✅ Kind checking prevents type confusion

**Lookup Methods:**

```go
// Handler lookup (lines 100-120)
func (r *Registry) GetValueHandler(path string) (ValueHandler, bool)
func (r *Registry) GetExecutionHandler(path string) (ExecutionHandler, bool)

// Schema lookup (lines 151-160)
func (r *Registry) GetSchema(path string) (DecoratorSchema, bool)

// Transport scope lookup (lines 162-185)
func (r *Registry) GetTransportScope(path string) TransportScope

// SDK handler lookup (lines 301-314)
func (r *Registry) GetSDKHandler(path string) (handler interface{}, kind DecoratorKind, exists bool)

// Full info lookup (lines 187-193)
func (r *Registry) GetInfo(path string) (DecoratorInfo, bool)
```

**Analysis:**
- Lookups return bool indicating existence (not error-based)
- Good pattern for optional lookups
- Transport scope has smart default (ScopeAgnostic, line 184)

### 2.3 Namespace Handling

**Current Design:**
- Flat namespace with dot-notation paths: `env`, `file.read`, `aws.s3.bucket`
- No formal namespace enforcement or conflict detection
- Plugin system (OEP-012) will add namespace prefixes (e.g., `hashicorp/aws`)

**Gap 2.3.1:** No namespace conflict detection

Currently, two plugins could register conflicting paths. Recommendation for OEP-012:

```go
// Add namespace validation
func (r *Registry) RegisterValueDecoratorWithSchema(
    schema DecoratorSchema,
    instance interface{},
    handler ValueHandler,
    namespace string,  // New parameter
) error {
    // Check for conflicts with existing namespace
    for existing := range r.decorators {
        if conflictsWith(existing, schema.Path, namespace) {
            return fmt.Errorf("namespace conflict: %s", schema.Path)
        }
    }
    // ... rest of registration
}
```

### 2.4 Conflict Resolution

**Current Approach:**
- Last registration wins (simple overwrite on line 79, 92)
- No versioning or deprecation mechanism
- No capability for gradual migration

**Gap 2.4.1:** No conflict resolution strategy for:
1. Plugin version conflicts
2. Multiple implementations of same decorator
3. Deprecation notices

**Recommendation 2.4.1:** Add versioning support:

```go
type DecoratorVersion struct {
    Major int
    Minor int
    Patch int
}

type DecoratorInfo struct {
    Path             string
    Version          DecoratorVersion  // New
    Kind             DecoratorKind
    Schema           DecoratorSchema
    Deprecated       bool              // New
    DeprecationMsg   string            // New
    ValueHandler     ValueHandler
    ExecutionHandler ExecutionHandler
    RawHandler       interface{}
}
```

---

## 3. Schema System Analysis

### 3.1 Type System Coverage

**Location:** `/home/user/opal/core/types/schema.go` (lines 1-24)

**Supported Types:**
```go
const (
    TypeString    ParamType = "string"
    TypeInt       ParamType = "integer"
    TypeFloat     ParamType = "float"
    TypeBool      ParamType = "boolean"
    TypeDuration  ParamType = "duration"
    TypeObject    ParamType = "object"
    TypeArray     ParamType = "array"
    TypeAuthHandle   ParamType = "AuthHandle"
    TypeSecretHandle ParamType = "SecretHandle"
    TypeScrubMode    ParamType = "ScrubMode"
)
```

**Strengths:**
- ✅ Core types (string, int, float, bool, duration)
- ✅ Collection types (object, array)
- ✅ Security-aware handles (AuthHandle, SecretHandle)
- ✅ Extensible enum support (ScrubMode)

**Coverage Assessment:**
- ✅ **OEP-001 (let bindings):** Object support sufficient for structured returns (lines 233: `Properties map[string]ParamSchema`)
- ✅ **OEP-002 (pipeline):** Numeric/string types cover PipeOp needs
- ✅ **OEP-010 (IaC):** Duration, int types sufficient
- ✅ **OEP-012 (plugins):** Extensible to support custom types

**Gap 3.1.1:** No null/optional type handling

```go
// Missing:
TypeNull   ParamType = "null"
TypeUnion  ParamType = "union"  // For @instance.optional_field | null
```

**Recommendation 3.1.1:** Add union and null types:

```go
const (
    TypeNull   ParamType = "null"
    TypeUnion  ParamType = "union"  // For multi-type fields
)

type ReturnSchema struct {
    Type        ParamType
    Description string
    Properties  map[string]ParamSchema
    Nullable    bool                   // New: for optional fields
    OneOf       []ParamType            // New: for union types
}
```

### 3.2 Parameter Schema Design

**Location:** `/home/user/opal/core/types/schema.go` (lines 213-227)

```go
type ParamSchema struct {
    Name        string
    Type        ParamType
    Description string
    Required    bool
    Default     interface{}
    Examples    []string
    Minimum     *int
    Maximum     *int
    Enum        []string
    Pattern     string
}
```

**Strengths:**
- ✅ Required/optional distinction
- ✅ Default values with automatic optional marking (line 526)
- ✅ Validation constraints (min, max, enum, pattern)
- ✅ Examples for documentation
- ✅ Used in tests (schema_test.go lines 142-185)

**Coverage:**
- ✅ Numeric ranges (Minimum, Maximum)
- ✅ Enum validation (lines 225, 409)
- ✅ Pattern matching (lines 226)
- ✅ Defaults with implicit optional (line 526)

**Strengths in Schema Builder:**
```go
// Lines 280-289 (Fluent API pattern)
func (b *SchemaBuilder) Param(name string, typ ParamType) *ParamBuilder {
    return &ParamBuilder{
        schemaBuilder: b,
        param: ParamSchema{
            Name: name,
            Type: typ,
        },
    }
}

// Enables readable schema definitions (from env.go):
schema := types.NewSchema("env", types.KindValue).
    Description("Access environment variables").
    PrimaryParam("property", types.TypeString, "Env var name").
    Param("default", types.TypeString).
    Description("Default value").
    Optional().
    Examples("", "/home/user").
    Done().
    Build()
```

### 3.3 Primary Property Pattern

**Location:** `/home/user/opal/core/types/schema.go` (lines 266-278)

**Design:**
```go
type DecoratorSchema struct {
    PrimaryParameter string  // Name of primary param ("property", "secretName", "")
    Parameters       map[string]ParamSchema
    // ...
}

func (b *SchemaBuilder) PrimaryParam(name string, typ ParamType, desc string) *SchemaBuilder {
    b.schema.PrimaryParameter = name
    b.schema.Parameters[name] = ParamSchema{
        Name:        name,
        Type:        typ,
        Description: desc,
        Required:    true,  // Always required (line 273)
    }
    b.parameterOrder = append(b.parameterOrder, name)
    return b
}
```

**Strengths:**
- ✅ Clean syntax: `@env.HOME` vs `@env(property="HOME")`
- ✅ Enforces required (line 273)
- ✅ Good for frequently-used single parameters
- ✅ Proper use in env.go example (lines 44-51)

**Design Validation (from DECORATOR_GUIDE.md lines 44-62):**
- ✅ Used for value decorators accessing named resources
- ✅ Single required parameter 90% of the time
- ✅ Improves readability
- ❌ Not used for execution decorators (correct)

### 3.4 Return Type Specification

**Location:** `/home/user/opal/core/types/schema.go` (lines 229-234)

```go
type ReturnSchema struct {
    Type        ParamType
    Description string
    Properties  map[string]ParamSchema  // For object returns
}
```

**Strengths:**
- ✅ Type specification with description
- ✅ Object properties support for structured returns
- ✅ Enables field access in OEP-001 (let bindings)

**Usage Example (env.go lines 52-53):**
```go
.Returns(types.TypeString, "Value of the environment variable")
```

**Sufficient for OEP-001:**
```go
// From OEP-001 proposal (field access):
let.instance = @aws.instance.deploy()
// Returns object with properties: id, public_ip, private_ip
curl http://@let.instance.public_ip/health

// This is supported by ReturnSchema.Properties map
```

### 3.5 Block Requirement Pattern

**Location:** `/home/user/opal/core/types/schema.go` (lines 72-79, 300-316)

```go
type BlockRequirement string

const (
    BlockForbidden BlockRequirement = "forbidden"  // Value decorators
    BlockOptional  BlockRequirement = "optional"   // @retry with/without block
    BlockRequired  BlockRequirement = "required"   // @parallel, @timeout
)

func (b *SchemaBuilder) WithBlock(requirement BlockRequirement) *SchemaBuilder {
    b.schema.BlockRequirement = requirement
    return b
}
```

**Design Analysis:**

| Decorator Type | Requirement | Example | Rationale |
|----------------|------------|---------|-----------|
| Value | Forbidden | @env.HOME | No side effects, returns value |
| Execution (optional) | Optional | @retry | Can retry inline commands OR blocks |
| Execution (required) | Required | @parallel, @timeout | Must have block to execute |

**Strengths:**
- ✅ Clear semantics
- ✅ Enforced at schema validation
- ✅ Used in tests (schema_test.go line 52, 313)

**Example Usage (retry.go lines 8-26):**
```go
schema := types.NewSchema("retry", types.KindExecution).
    Description("Retry failed operations").
    Param("times", types.TypeInt).
    Default(3).
    Done().
    WithBlock(types.BlockOptional).  // Can optionally have block
    Build()
```

---

## 4. Transport Scope System Analysis

### 4.1 Design and Intent

**Location:** `/home/user/opal/core/types/schema.go` (lines 43-64)

```go
type TransportScope uint8

const (
    ScopeRootOnly    TransportScope = iota  // @env, @file.read - local only, plan-time
    ScopeAgnostic                           // @var, @random - anywhere, plan-seeded
    ScopeRemoteAware                        // @proc.env(transport=...) - explicit remote (future)
)

type ValueScopeProvider interface {
    TransportScope() TransportScope
}
```

**Design Philosophy:**
- Declares where a value decorator can safely resolve
- Prevents confusing behavior (@env.HOME inside @ssh.connect)
- Opt-in interface for decorators to declare scope

### 4.2 Usage Pattern

**Implementation (env.go lines 36-40):**
```go
func (e *envDecorator) TransportScope() types.TransportScope {
    return types.ScopeRootOnly
}
```

**Registration (env.go lines 63-65):**
```go
decorator := &envDecorator{}
if err := types.Global().RegisterValueDecoratorWithSchema(schema, decorator, decorator.Handle); err != nil {
    panic(...)
}
```

**Lookup (registry.go lines 162-185):**
```go
func (r *Registry) GetTransportScope(path string) TransportScope {
    // ...
    if scopeProvider, ok := info.RawHandler.(ValueScopeProvider); ok {
        return scopeProvider.TransportScope()
    }
    return ScopeAgnostic  // Safe default
}
```

**Validation (validation.go lines 117-203):**
```go
// Prevents root-only decorators in remote contexts
if scope == types.ScopeRootOnly && transportDepth > 0 {
    return &ValidationError{
        Message: "@" + decoratorName + 
                " is root-only and cannot be used inside " + transportName,
    }
}
```

### 4.3 Completeness for @ssh.connect/@docker.exec

**Current System:**
- ✅ Detects and prevents root-only decorators in remote contexts
- ✅ Provides SwitchesTransport flag in schema (line 210)
- ✅ Tracks transport depth during validation (line 127)

**Scope Classes:**
1. **ScopeRootOnly** - @env (plan-time local environment)
2. **ScopeAgnostic** - @var, @random (deterministic anywhere)
3. **ScopeRemoteAware** (future) - @proc.env with explicit transport

**Gap 4.3.1:** ScopeRemoteAware not yet used

For OEP-010 (IaC with SSH blocks), will need:
```go
// Future @proc.env decorator
@proc.env(transport="ssh", name="HOME")  // Resolves remote HOME

// This would be ScopeRemoteAware - explicitly knows about transport context
```

**Recommendation 4.3.1:** Document ScopeRemoteAware usage pattern:

```go
// Example future decorator
type procEnvDecorator struct{}

func (p *procEnvDecorator) Handle(ctx Context, args Args) (Value, error) {
    // Resolve from specified transport, not hardcoded local env
    transport, _ := args.GetString("transport")
    name, _ := args.GetString("name")
    
    // Use transport context from ctx to resolve
    return resolveFromTransport(ctx, transport, name)
}

func (p *procEnvDecorator) TransportScope() TransportScope {
    return ScopeRemoteAware  // Can work in remote contexts
}
```

### 4.4 Environment Inheritance Rules

**Current Implementation:** Declarative (schema-based)
- Validation (validation.go) checks SwitchesTransport flag
- No runtime environment isolation yet

**For OEP-010 (SSH/Docker blocks):**
Need to define:
1. **Inherited variables:** What env vars pass through to remote?
2. **Blocked variables:** Which env vars NOT sent (security)?
3. **Remote-only variables:** What's available only remotely?

**Recommendation 4.4.1:** Add environment isolation rules:

```go
type EnvironmentPolicy struct {
    // Variables to explicitly pass through
    PassThrough []string  // e.g., ["USER", "PATH"]
    
    // Variables to explicitly block
    Block       []string  // e.g., ["AWS_SECRET_ACCESS_KEY"]
    
    // Pattern-based passing
    PassPattern string    // Regex for allowed vars
    BlockPattern string    // Regex for blocked vars
}

type TransportContext struct {
    Type             string
    Config           map[string]interface{}
    EnvironmentPolicy EnvironmentPolicy  // New
}
```

---

## 5. Future Compatibility Analysis

### 5.1 OEP-001: Runtime `let` Bindings

**Proposal Requirements:**
- Separate namespace `@let` for execution-time values
- Structured data returns with field access (`@let.instance.id`)
- Distinct scope rules (no @let in plan-time constructs)
- Block-level variable bindings

**Current Support Assessment:**

| Feature | Status | Implementation |
|---------|--------|-----------------|
| Structured returns | ✅ Ready | ReturnSchema.Properties (line 233) |
| Field access | ⚠️ Parser only | AST support needed |
| Scope isolation | ⚠️ Partial | Transport scope works, let scope new |
| Execution-time binding | ❌ Missing | Context.LetBindings needed (line 21) |
| Plan validation | ⚠️ Partial | validateEnvInRemoteTransport exists (line 125) |

**Gap Analysis:**

1. **Context Missing LetBindings:**
   ```go
   // Current (line 20-24)
   type Context struct {
       Variables  map[string]Value
       Env        map[string]string
       WorkingDir string
   }
   
   // Needs:
   // LetBindings map[string]Value  // Runtime values
   // Scope ScopeContext             // For parallel isolation
   ```

2. **No AST Support for Field Access:**
   - Parser needs to recognize `@let.instance.id`
   - Need FieldAccessNode in AST
   - Validation must prevent @let in plan-time constructs

3. **No Structured Return Support at Runtime:**
   - ReturnSchema supports types (Good!)
   - But executor doesn't instantiate/validate structured returns

**Recommendation 5.1.1:** Implement Let binding infrastructure:

```go
// In core/types/runtime.go (new file)
type LetBinding struct {
    Name      string
    Value     interface{}
    Type      ParamType
    Scope     string  // Block ID for scoping
}

type LetBindingStore struct {
    mu       sync.RWMutex
    bindings map[string]*LetBinding
    scopes   map[string][]string  // scope -> binding names
}

func (l *LetBindingStore) Bind(scope string, name string, value interface{}) error {
    l.mu.Lock()
    defer l.mu.Unlock()
    
    if _, exists := l.bindings[name]; exists {
        return fmt.Errorf("let.%s already bound", name)
    }
    
    l.bindings[name] = &LetBinding{
        Name:  name,
        Value: value,
        Scope: scope,
    }
    return nil
}

func (l *LetBindingStore) Get(name string) (interface{}, error) {
    l.mu.RLock()
    defer l.mu.RUnlock()
    
    if binding, ok := l.bindings[name]; ok {
        return binding.Value, nil
    }
    return nil, fmt.Errorf("let.%s not bound", name)
}
```

### 5.2 OEP-002: Pipeline Operators

**Proposal Requirements:**
- `|>` operator for pure, bounded, deterministic transforms
- PipeOp registry with MaxExpansionFactor
- Built-in PipeOps (json.pick, lines.grep, assert.re, etc.)
- Prevent unbounded memory growth

**Current Support Assessment:**

| Feature | Status | Implementation |
|---------|--------|-----------------|
| PipeOp type system | ❌ Missing | Need new types/interfaces |
| Traits (pure, bounded, deterministic) | ❌ Missing | Need trait declarations |
| Built-in PipeOps | ❌ Not implemented | Would be in runtime/pipeops |
| Memory bounds enforcement | ❌ Missing | Need expansion factor tracking |
| Parser support | ❌ Missing | `\|>` operator not in grammar |

**Gap Analysis:**

PipeOps need a new registry and interface system:

```go
// Missing: core/types/pipeop.go
type PipeOpTrait struct {
    Pure             bool
    Bounded          bool
    Deterministic    bool
    MaxExpansionFactor float64
    ReadsStdin       bool
    WritesStdout     bool
}

type PipeOp interface {
    Name() string
    Traits() PipeOpTrait
    Execute(input []byte) ([]byte, error)
}

type PipeOpRegistry struct {
    mu      sync.RWMutex
    pipeops map[string]PipeOp
}
```

**Recommendation 5.2.1:** Create PipeOp infrastructure:

```go
// core/types/pipeop.go
type PipeOpSchema struct {
    Name       string
    Traits     PipeOpTrait
    Parameters map[string]ParamSchema
    Returns    *ReturnSchema
}

type PipeOpHandler func(input []byte, params map[string]interface{}) ([]byte, error)

type PipeOpInfo struct {
    Schema  PipeOpSchema
    Handler PipeOpHandler
}

var pipeOpRegistry = &PipeOpRegistry{
    pipeops: make(map[string]PipeOp),
}

// Built-in PipeOps
func init() {
    registerPipeOp("json.pick", NewJsonPickOp())
    registerPipeOp("lines.grep", NewLinesGrepOp())
    registerPipeOp("lines.count", NewLinesCountOp())
    registerPipeOp("assert.re", NewAssertReOp())
    registerPipeOp("assert.num", NewAssertNumOp())
}
```

### 5.3 OEP-010: Infrastructure as Code Deploy Blocks

**Proposal Requirements:**
- Deploy blocks (run once on creation)
- SSH blocks (run every time)
- Idempotence matching with flexible keys
- Transport switching (ssh.connect, docker.exec)

**Current Support Assessment:**

| Feature | Status | Implementation |
|---------|--------|-----------------|
| Block requirement system | ✅ Ready | BlockRequired, BlockOptional (line 76-79) |
| Transport switching flag | ✅ Ready | SwitchesTransport in schema (line 210) |
| Transport validation | ✅ Ready | validateEnvInRemoteTransport (line 125) |
| Deploy block tracking | ❌ Missing | Need plan-time versus runtime block distinction |
| Idempotence matching | ❌ Missing | Providers need matching logic |
| Transport context | ⚠️ Partial | SwitchesTransport flag exists, not context switching |

**Gap Analysis:**

1. **Block Execution Tracking:**
   - Need to distinguish "deploy block" vs "ssh block"
   - Schema needs to declare this

2. **Transport Context Switching:**
   - Current validation prevents scope confusion
   - Need actual transport switching at runtime (OEP-010)
   - Requires Context.Transport field

3. **Idempotence Key Support:**
   - No schema support for idempotenceKey parameter
   - Providers need to declare matching strategy

**Recommendation 5.3.1:** Extend schema for IaC features:

```go
type BlockExecutionMode string

const (
    BlockModeOnce   BlockExecutionMode = "once"   // Deploy blocks
    BlockModeAlways BlockExecutionMode = "always" // SSH blocks
)

type DecoratorSchema struct {
    // Existing fields...
    BlockRequirement  BlockRequirement
    BlockExecutionMode BlockExecutionMode  // New: once vs always
    
    // Idempotence support
    IdempotenceKey    []string  // e.g., ["name", "type"]
    IdempotenceMatcher string   // "strict", "pragmatic", "name-only"
}

// Usage in schema builder
schema := types.NewSchema("aws.instance.deploy", types.KindExecution).
    // ...
    WithBlockMode(types.BlockModeOnce).
    WithIdempotenceKey([]string{"name"}).
    Build()
```

### 5.4 OEP-012: Plugin System

**Proposal Requirements:**
- Plugin registration via opal.mod
- WASM and native plugin support
- DIM manifest for decorator declarations
- Git repository and registry sources
- Namespace management

**Current Support Assessment:**

| Feature | Status | Implementation |
|---------|--------|-----------------|
| Registration API | ✅ Ready | RegisterSDKHandler supports interface{} (line 260) |
| Schema validation | ✅ Ready | ValidateSchema (line 545) |
| Multiple handler types | ✅ Ready | RawHandler stores any interface{} (line 58) |
| Versioning | ❌ Missing | No version tracking in DecoratorInfo |
| Namespace isolation | ❌ Missing | No conflict detection |
| Module/dependency system | ❌ Missing | Separate system needed |

**Gap Analysis:**

1. **No Versioning Support:**
   - Can't have multiple versions of same decorator
   - Registry overwrites on re-registration (line 79)
   - Plugin system needs version-aware lookup

2. **No Namespace Management:**
   - Decorators use flat dot-notation (env, aws.s3.bucket)
   - Plugin system needs namespace prefixes (hashicorp/aws)
   - Need conflict detection and scoping

3. **No Capability Declaration:**
   - Can't restrict what plugins can do
   - OEP-012 proposes capability restrictions:
     - No filesystem access
     - No network access
     - No env var scrubbing

**Recommendation 5.4.1:** Add plugin capabilities:

```go
type PluginCapability string

const (
    CapFilesystemRead  PluginCapability = "filesystem.read"
    CapFilesystemWrite PluginCapability = "filesystem.write"
    CapNetwork         PluginCapability = "network"
    CapEnv             PluginCapability = "environment"
    CapProcessControl  PluginCapability = "process.control"
)

type PluginInfo struct {
    Name         string
    Version      string
    Author       string
    MinCoreVersion string
    Capabilities []PluginCapability
    Decorators   []DecoratorInfo
}

type DecoratorInfo struct {
    // Existing fields...
    Plugin PluginInfo  // Which plugin provides this
    
    // Versioning
    Version DecoratorVersion
}
```

---

## 6. Code Quality Assessment

### 6.1 Separation of Concerns

**Excellent: Registry System**

Clear module boundaries:
- `core/types/schema.go` - Schema definitions and builders
- `core/types/registry.go` - Decorator registration and lookup
- `runtime/decorators/` - Decorator implementations
- `runtime/parser/validation.go` - Parse-time validation

**Example of Good Separation (env.go):**
```go
package decorators  // Not in core/types

import "github.com/aledsdavies/opal/core/types"

// Decorator implementation
type envDecorator struct{}

// Handler implements the interface
func (e *envDecorator) Handle(ctx types.Context, args types.Args) (types.Value, error)

// Optional interface (TransportScope)
func (e *envDecorator) TransportScope() types.TransportScope

// Registration via init()
func init() {
    types.Global().RegisterValueDecoratorWithSchema(schema, decorator, handler)
}
```

**Issues Identified:**

1. **Circular dependency risk** (lines 254-275 of registry.go comment):
   ```go
   // This is the new style that uses sdk.ExecutionContext and sdk.Step.
   // The handler type depends on the decorator kind:
   // - Value decorators: sdk.ValueHandler
   // - Execution decorators: sdk.ExecutionHandler
   //
   // This avoids circular dependencies: core/types imports core/sdk (both in core).
   ```
   
   Using `interface{}` is pragmatic but loses type safety. Good design choice given constraints.

2. **Mixed Handler Styles** (lines 56-58 of registry.go):
   ```go
   ValueHandler     ValueHandler     // Old-style handler
   ExecutionHandler ExecutionHandler // Old-style handler
   RawHandler       interface{}      // New SDK-style handler
   ```
   
   Maintains backward compatibility but DecoratorInfo is cluttered.

### 6.2 Extensibility Without Modification

**Strong Points:**
- ✅ Schema builder pattern (fluent API)
- ✅ Optional interfaces (ValueScopeProvider pattern)
- ✅ Handler registration decoupled from core types
- ✅ Registry can be extended with new methods

**Pattern: Optional Interfaces**

```go
// ValueScopeProvider is opt-in (lines 66-70)
type ValueScopeProvider interface {
    TransportScope() TransportScope
}

// Registry checks for implementation (lines 179-180)
if scopeProvider, ok := info.RawHandler.(ValueScopeProvider); ok {
    return scopeProvider.TransportScope()
}

// Default behavior if not implemented (line 184)
return ScopeAgnostic
```

This is excellent design: new features can be added without breaking existing decorators.

**Good Extensibility Pattern (schema_test.go lines 348-422):**
```go
// NewSchema creates builders with smart defaults (lines 243-258)
func NewSchema(path string, kind DecoratorKindString) *SchemaBuilder {
    blockReq := BlockForbidden
    if kind == KindExecution {
        blockReq = BlockOptional
    }
    return &SchemaBuilder{ /* ... */ }
}

// Fluent API enables readable extensions
schema := types.NewSchema("retry", types.KindExecution).
    Description("Retry failed operations").
    Param("times", types.TypeInt).Default(3).Done().
    Param("delay", types.TypeDuration).Default("1s").Done().
    WithBlock(types.BlockOptional).
    Build()
```

### 6.3 Type Safety

**Strong Type Safety for Decorators:**

```go
// Handler signature is strongly typed (line 35)
type ValueHandler func(ctx Context, args Args) (Value, error)

// Args structure is well-defined (lines 26-31)
type Args struct {
    Primary *Value
    Params  map[string]Value
    Block   *Block
}

// Schema defines types (line 216)
type ParamSchema struct {
    Type ParamType
}
```

**Weakness: Parameter Type Erasure**

At runtime, `Params` values are `interface{}`, requiring type assertions:

```go
// From env.go lines 18-19
envVar := (*args.Primary).(string)  // Type assertion!

// If user passes wrong type, panic at runtime
// Solution: validate schema BEFORE calling handler
```

**Mitigation:** Schema validation happens before handler invocation. This is good design.

**Gap 6.3.1: No Compile-Time Type Checking for Parameters**

Recommendation: Add type-safe helper:
```go
func (a *Args) GetString(name string) (string, error) {
    v, ok := a.Params[name]
    if !ok {
        return "", fmt.Errorf("parameter %q not found", name)
    }
    s, ok := v.(string)
    if !ok {
        return "", fmt.Errorf("parameter %q is not a string, got %T", name, v)
    }
    return s, nil
}
```

### 6.4 Error Handling Patterns

**Good Error Handling Pattern (env.go):**

```go
// Lines 14-30
func (e *envDecorator) Handle(ctx types.Context, args types.Args) (types.Value, error) {
    if args.Primary == nil {
        return nil, fmt.Errorf("@env requires an environment variable name")
    }
    
    envVar := (*args.Primary).(string)
    value, exists := ctx.Env[envVar]
    if !exists {
        if args.Params != nil {
            if defaultVal, hasDefault := args.Params["default"]; hasDefault {
                return defaultVal, nil
            }
        }
        return nil, fmt.Errorf("environment variable %q not found", envVar)
    }
    
    return value, nil
}
```

**Strengths:**
- ✅ Defensive checking (nil guards)
- ✅ Descriptive error messages
- ✅ Graceful fallback (default values)
- ✅ No panics (except in init())

**Weakness: Init Panics (registry.go line 66):**

```go
func init() {
    // ...
    if err := types.Global().RegisterValueDecoratorWithSchema(schema, decorator, decorator.Handle); err != nil {
        panic(fmt.Sprintf("failed to register @env decorator: %v", err))
    }
}
```

This is acceptable for init(), but makes testing harder.

**Recommendation 6.4.1:** Provide test-friendly registration:

```go
// Current
types.Global().RegisterValueDecoratorWithSchema(schema, decorator, handler)

// New: return error for testing
if err := types.Global().TryRegisterValueDecoratorWithSchema(schema, decorator, handler); err != nil {
    panic(err)  // In production (init)
}

// In tests
registry := types.NewRegistry()
if err := registry.RegisterValueDecoratorWithSchema(schema, decorator, handler); err != nil {
    t.Fatalf("registration failed: %v", err)
}
```

### 6.5 Testing Support

**Test Coverage Assessment:**

| Module | Tests | Coverage | Quality |
|--------|-------|----------|---------|
| registry.go | ✅ 28 tests | High | Comprehensive (registry_test.go) |
| schema.go | ✅ 18 tests | High | Comprehensive (schema_test.go) |
| Decorators | ✅ Basic tests | Low-Medium | Minimal examples (env_scope_test.go has 1 test) |

**Test File: registry_test.go (380 lines)**

Strong coverage:
- ✅ Basic registration (line 7-18)
- ✅ Multiple decorators (line 20-33)
- ✅ Value vs execution distinction (line 55-140)
- ✅ Handler invocation (line 142-222)
- ✅ SDK handler registration (line 297-363)

**Test File: schema_test.go (622 lines)**

Excellent coverage:
- ✅ Builder fluent API (line 5-38)
- ✅ Parameter ordering (line 142-309)
- ✅ Block requirements (line 5-60)
- ✅ I/O capabilities (line 348-621)
- ✅ Validation (line 107-346)

**Gaps in Testing:**
1. No integration tests (schema + registry together)
2. Minimal decorator implementation tests
3. No tests for circular dependency detection (future)

### 6.6 Documentation Quality

**Strong Documentation:**

1. **DECORATOR_GUIDE.md (100+ lines):**
   - Decorator anatomy
   - Primary property patterns
   - Block semantics
   - Implementation best practices

2. **Code Comments:**
   - Line 8-10 (registry.go): Clear explanation of Value type
   - Line 33-35 (registry.go): Handler signature explained
   - Line 43-50 (schema.go): TransportScope meaning
   - Line 117-124 (validation.go): Transport validation purpose

3. **In-Code Examples:**
   - env.go: Full decorator implementation (67 lines)
   - shell.go: SDK-based decorator with SinkProvider (90 lines)
   - schema_test.go: Examples of all schema patterns

**Areas Needing Documentation:**

1. **Plugin System (OEP-012):** No current documentation
2. **SDK Handlers:** Brief comments but no guide
3. **Transport Scope Usage:** Only 1 test (env_scope_test.go)
4. **Let Bindings (OEP-001):** Not yet documented

---

## 7. Recommendations for Improvements

### Priority 1: Required for OEP-001 (Let Bindings)

**Recommendation 7.1.1:** Extend Context structure
```go
type Context struct {
    // Existing
    Variables  map[string]Value
    Env        map[string]string
    WorkingDir string
    
    // New: OEP-001 support
    LetBindings map[string]Value
    Scope       ScopeContext
}

type ScopeContext struct {
    BlockID  string  // Unique block identifier
    BranchID string  // For @parallel isolation
    Depth    int     // Nesting depth
}
```

**Recommendation 7.1.2:** Implement LetBindingStore
```go
type LetBindingStore struct {
    mu              sync.RWMutex
    bindings        map[string]Value
    bindingSites    map[string]string  // name -> blockID
    scopeBindings   map[string][]string  // blockID -> names
}

func (l *LetBindingStore) Bind(scope string, name string, value Value) error {
    l.mu.Lock()
    defer l.mu.Unlock()
    
    if _, exists := l.bindings[name]; exists {
        return fmt.Errorf("let.%s already bound", name)
    }
    
    l.bindings[name] = value
    l.bindingSites[name] = scope
    l.scopeBindings[scope] = append(l.scopeBindings[scope], name)
    return nil
}
```

### Priority 2: Needed for Type Safety

**Recommendation 7.2.1:** Add parameter access helpers
```go
func (a *Args) GetString(name string) (string, error) {
    v, ok := a.Params[name]
    if !ok {
        return "", fmt.Errorf("parameter %q not found", name)
    }
    s, ok := v.(string)
    if !ok {
        return "", fmt.Errorf("parameter %q is not string, got %T", name, v)
    }
    return s, nil
}

// Similar for: GetInt, GetFloat, GetBool, GetDuration, GetObject
```

**Recommendation 7.2.2:** Add union and null types
```go
const (
    TypeNull   ParamType = "null"
    TypeUnion  ParamType = "union"
    TypeOneOf  ParamType = "oneof"  // For discriminated unions
)

type ReturnSchema struct {
    Type      ParamType
    Description string
    Properties map[string]ParamSchema
    Nullable   bool
    OneOf      []ParamType  // Union types
}
```

### Priority 3: Plugin System (OEP-012)

**Recommendation 7.3.1:** Add version support
```go
type DecoratorVersion struct {
    Major int
    Minor int
    Patch int
}

type DecoratorInfo struct {
    // Existing...
    Version DecoratorVersion
    MinCoreVersion DecoratorVersion
    Deprecated bool
    DeprecationMsg string
}
```

**Recommendation 7.3.2:** Add namespace/plugin info
```go
type PluginMetadata struct {
    Name     string
    Version  DecoratorVersion
    Namespace string  // e.g., "hashicorp/aws"
}

type DecoratorInfo struct {
    // Existing...
    Plugin *PluginMetadata
}
```

### Priority 4: Enhanced Validation

**Recommendation 7.4.1:** Add namespace conflict detection
```go
func (r *Registry) RegisterValueDecoratorWithSchema(
    schema DecoratorSchema,
    instance interface{},
    handler ValueHandler,
) error {
    // Check for conflicts
    for existing := range r.decorators {
        if r.conflicts(existing, schema.Path) {
            return fmt.Errorf("decorator path conflicts with %q", existing)
        }
    }
    // ... rest of registration
}
```

**Recommendation 7.4.2:** Add initialization validation
```go
type Registry struct {
    mu         sync.RWMutex
    decorators map[string]DecoratorInfo
    validated  bool  // Track if registry is finalized
}

// Call after all init() runs
func (r *Registry) Validate() error {
    r.mu.Lock()
    defer r.mu.Unlock()
    
    // Check for conflicts
    // Check for missing dependencies
    // Check version constraints
    
    r.validated = true
    return nil
}
```

---

## 8. Migration Path for Changes

### Phase 1: Backward Compatibility (Current)
- Keep old `ValueHandler` and `ExecutionHandler` interfaces
- Support both schema patterns
- RawHandler fallback for SDK handlers

### Phase 2: Context Expansion (OEP-001)
- Add LetBindings, Scope to Context
- Update all decorators to handle new fields (or ignore)
- Introduce LetBindingStore separately

### Phase 3: Plugin System (OEP-012)
- Add versioning to DecoratorInfo
- Implement namespace validation
- Create plugin registry separate from core registry

### Phase 4: Complete Migration (Post-v1.0)
- Deprecate old handler styles
- Require schemas for all decorators
- Unify under SDK handler pattern

---

## 9. Summary: Design Quality Assessment

### What Works Well

| Aspect | Rating | Rationale |
|--------|--------|-----------|
| Handler Simplicity | ⭐⭐⭐⭐⭐ | Function signatures are clean and focused |
| Schema System | ⭐⭐⭐⭐⭐ | Fluent API, comprehensive validation |
| Transport Scope | ⭐⭐⭐⭐ | Clever opt-in interface pattern |
| Type Safety | ⭐⭐⭐⭐ | Strong at schema level, type erasure at runtime is documented |
| Extensibility | ⭐⭐⭐⭐⭐ | Optional interfaces enable feature addition |
| Testing | ⭐⭐⭐⭐ | Comprehensive registry and schema tests |
| Documentation | ⭐⭐⭐⭐ | Good examples and guide, some gaps for advanced topics |

### What Needs Work

| Aspect | Rating | Action |
|--------|--------|--------|
| Let Binding Support | ⭐⭐ | Extend Context, implement LetBindingStore |
| Plugin Versioning | ⭐⭐ | Add version fields to DecoratorInfo |
| Namespace Management | ⭐⭐ | Implement conflict detection |
| Parameter Type Safety | ⭐⭐⭐ | Add typed accessor helpers |
| PipeOp System | ⭐ | Entirely new (OEP-002) |
| Decorator Integration Tests | ⭐⭐⭐ | More end-to-end testing |

### Ready for Production

The current decorator architecture is **production-ready for core value and execution decorators**. The design demonstrates:

- ✅ Clear semantics and intent
- ✅ Strong error handling patterns
- ✅ Good separation of concerns
- ✅ Comprehensive testing
- ✅ Extensibility without modification

### Recommended Implementation Order

1. **OEP-001 (Let Bindings):** Highest priority for runtime features
   - Requires Context extension
   - Enables infrastructure provisioning patterns
   - Moderate impact on existing code

2. **OEP-012 (Plugin System):** Critical for ecosystem
   - Requires versioning support
   - Relatively isolated changes
   - Can be implemented independently

3. **OEP-002 (Pipeline):** Important for data transformation
   - Requires new PipeOp registry
   - Lower priority (less common use case)
   - Can follow OEP-001

4. **OEP-010 (IaC):** Integration feature
   - Depends on OEP-001 for proper support
   - Builds on existing block system

---

## 10. Appendix: Code Examples

### Example: Complete Value Decorator with All Features

```go
package decorators

import (
    "fmt"
    "strings"
    
    "github.com/aledsdavies/opal/core/types"
)

// jsonExtractDecorator extracts values from JSON using JSONPath
type jsonExtractDecorator struct{}

// Handle implements ValueHandler
func (j *jsonExtractDecorator) Handle(
    ctx types.Context,
    args types.Args,
) (types.Value, error) {
    // Validate primary parameter (path)
    if args.Primary == nil {
        return nil, fmt.Errorf("@json.extract requires JSONPath")
    }
    path := (*args.Primary).(string)
    
    // Get source from params
    source, err := args.GetString("source")  // Helper method
    if err != nil {
        return nil, fmt.Errorf("@json.extract source: %w", err)
    }
    
    // Optional: format parameter
    format := "raw"
    if f, err := args.GetString("format"); err == nil {
        format = f
    }
    
    // Extract JSON (implementation details omitted)
    value := extractFromJSON(source, path)
    
    // Apply formatting if requested
    if format == "pretty" {
        return prettyPrint(value), nil
    }
    
    return value, nil
}

// TransportScope implements ValueScopeProvider
// JSON extraction is agnostic - can work anywhere
func (j *jsonExtractDecorator) TransportScope() types.TransportScope {
    return types.ScopeAgnostic
}

func init() {
    schema := types.NewSchema("json.extract", types.KindValue).
        Description("Extract values from JSON using JSONPath").
        PrimaryParam("path", types.TypeString, "JSONPath expression").
        Param("source", types.TypeString).
            Description("JSON source (string or @var reference)").
            Required().
            Done().
        Param("format", types.TypeString).
            Description("Output format").
            Default("raw").
            Examples("raw", "pretty", "compact").
            Done().
        Returns(types.TypeString, "Extracted value").
        Build()
    
    decorator := &jsonExtractDecorator{}
    if err := types.Global().RegisterValueDecoratorWithSchema(schema, decorator, decorator.Handle); err != nil {
        panic(fmt.Sprintf("failed to register @json.extract: %v", err))
    }
}
```

### Example: Decorator with Structured Return

```go
// awsInstanceDecorator would be part of a plugin system (OEP-012)
type awsInstanceDecorator struct {
    client *ec2.Client
}

// Deploy creates or reuses an EC2 instance
func (a *awsInstanceDecorator) Deploy(
    ctx types.Context,
    args types.Args,
) (types.Value, error) {
    // Get parameters
    name, _ := args.GetString("name")
    amiID, _ := args.GetString("ami")
    instanceType, _ := args.GetString("type")
    
    // Find or create instance
    instance := a.client.FindOrCreate(name, amiID, instanceType)
    
    // Return structured data (OEP-001 compatible)
    return map[string]interface{}{
        "id":         instance.ID,
        "public_ip":  instance.PublicIP,
        "private_ip": instance.PrivateIP,
        "state":      instance.State,
        "tags":       instance.Tags,
    }, nil
}

func init() {
    schema := types.NewSchema("aws.instance.deploy", types.KindExecution).
        Description("Deploy an EC2 instance").
        Param("name", types.TypeString).
            Description("Instance name").
            Required().
            Done().
        Param("ami", types.TypeString).
            Description("AMI ID").
            Required().
            Done().
        Param("type", types.TypeString).
            Description("Instance type").
            Default("t3.micro").
            Done().
        Param("idempotenceKey", types.TypeArray).
            Description("Fields to match for idempotence").
            Default([]string{"name"}).
            Done().
        WithBlock(types.BlockModeOnce).  // Deploy block
        Returns(types.TypeObject, "Instance information").
            Properties(map[string]types.ParamSchema{
                "id":         {Type: types.TypeString},
                "public_ip":  {Type: types.TypeString, Nullable: true},
                "private_ip": {Type: types.TypeString},
                "state":      {Type: types.TypeString},
            }).
            Done().
        Build()
    
    // Registration would be via plugin system
    // types.Global().RegisterSDKHandlerWithSchema(schema, handler)
}
```

---

## 11. References

### OPAL Documentation
- `/home/user/opal/docs/DECORATOR_GUIDE.md` - Decorator design patterns
- `/home/user/opal/docs/SPECIFICATION.md` - Language specification
- `/home/user/opal/docs/ARCHITECTURE.md` - System architecture

### Relevant OEPs
- OEP-001: Runtime Variable Binding with `let` (OEP-001-runtime-let-binding.md)
- OEP-002: Transform Pipeline Operator `|>` (OEP-002-transform-pipeline-operator.md)
- OEP-010: Infrastructure as Code (OEP-010-infrastructure-as-code.md)
- OEP-012: Module Composition and Plugin System (OEP-012-module-composition.md)

### Code References
- Core Registry: `/home/user/opal/core/types/registry.go`
- Schema System: `/home/user/opal/core/types/schema.go`
- Decorator Examples: `/home/user/opal/runtime/decorators/`
- Validation: `/home/user/opal/runtime/parser/validation.go`
- Tests: `/home/user/opal/core/types/{registry,schema}_test.go`

---

**Document Version:** 1.0  
**Last Updated:** 2025-10-25  
**Review Status:** Complete  
**Recommended Actions:** Implement Priority 1 and 2 recommendations before v1.0 release
