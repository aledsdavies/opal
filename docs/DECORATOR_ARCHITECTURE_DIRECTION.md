---
title: "OPAL Decorator Architecture: Proposed Direction"
status: "RFC - Under Review"
date: "2025-10-25"
audience: "Core Contributors & Architects"
---

# OPAL Decorator Architecture: Proposed Direction

**Status**: RFC (Request for Comments)
**Author**: Architecture Review
**Date**: 2025-10-25

## Executive Summary

This document proposes a refined decorator architecture for OPAL that addresses current limitations while maintaining the project's philosophy of **minimal, composable design**. The proposal introduces a **Kind vs Role separation** and a **unified Session-based execution model** that elegantly handles the complexity of value resolution, execution wrapping, transport boundaries, and I/O operations.

**Key Innovation**: Decorators can have multiple roles (e.g., `@aws.s3.object` is both a **Value provider** and a **Sink endpoint**), enabling rich semantics without proliferating decorator types.

---

## 1. Motivation: Current Pain Points

Based on comprehensive analysis of the current decorator infrastructure ([DECORATOR_ARCHITECTURE_REVIEW.md](./DECORATOR_ARCHITECTURE_REVIEW.md)), we identified these limitations:

### 1.1 Overloaded Handler Types

**Current State** (`core/types/registry.go:51-65`):
```go
type DecoratorInfo struct {
    Path             string
    Kind             DecoratorKind    // Value or Execution
    ValueHandler     ValueHandler     // Maybe this?
    ExecutionHandler ExecutionHandler // Or this?
    RawHandler       interface{}      // Or this??
}
```

**Problem**: A single struct holds "maybe value, maybe execution" with ad-hoc `RawHandler`. In practice, these are fundamentally different beasts:
- **Value decorators** are pure functions (no side effects)
- **Execution decorators** wrap child execution nodes

### 1.2 Ambiguous Resolution Timing

**Current State**: No explicit contract for when value decorators resolve.

**Problem**: Makes it hard to:
- Do plan-time substitution
- Generate secret DisplayIDs deterministically
- Track provenance for scrubbing

### 1.3 Scope Not Enforced by Interface

**Current State**: `GetTransportScope()` exists but isn't wired into calls.

**Problem**: Decorators can be invoked where they shouldn't be (e.g., `@env` inside `@ssh.connect` blocks, which would read the wrong environment).

### 1.4 Schemas Aren't Authoritative

**Current State**: Parser validates types, but schemas lack:
- Enum constraints
- Min/max bounds
- Pattern validation
- Default values (for optional params)

**Problem**: Validation logic scattered across parser and handlers.

### 1.5 No Middleware Contract

**Current State**: Execution decorators are "handlers" instead of structured wrappers.

**Problem**: No clean composition model for `@timeout{ @retry{ ... } }`.

### 1.6 Missing I/O Abstraction

**Current State**: No unified model for decorators that act as data sources/sinks.

**Problem**: Future decorators like `@fs.read`, `@http.get`, `@s3.put` need a consistent I/O contract.

---

## 2. Proposed Design: Kind vs Role

### 2.1 Core Taxonomy

Instead of a binary "value or execution" classification, decorators advertise:
- **Kind**: What the decorator *is* (its primary nature)
- **Roles**: How it *behaves* (its capabilities)

This allows **multi-role decorators** that participate in multiple semantic contexts.

```go
type Kind string
const (
    KindValue     Kind = "value"     // Produces values
    KindExec      Kind = "exec"      // Wraps execution steps
    KindTransport Kind = "transport" // Session boundaries (SSH, Docker)
    KindSink      Kind = "sink"      // I/O endpoints (files, HTTP, S3)
    KindMeta      Kind = "meta"      // Annotations/policy (@trace, @measure)
)

type Role string
const (
    RoleProvider Role = "provider"  // Produces data (@var, @env, @aws.secret)
    RoleWrapper  Role = "wrapper"   // Wraps execution (@retry, @timeout)
    RoleBoundary Role = "boundary"  // Creates scoped context (@ssh.connect)
    RoleEndpoint Role = "endpoint"  // Reads/writes data (@file.read, @s3.put)
    RoleAnnotate Role = "annotate"  // Augments plan metadata (@trace)
)
```

**Example - Multi-Role Decorator**:
```go
// @aws.s3.object can both provide metadata AND write content
type S3ObjectDecorator struct{}

func (d *S3ObjectDecorator) Descriptor() Descriptor {
    return Descriptor{
        Path:  "aws.s3.object",
        Kind:  KindValue,  // Primary nature: produces values
        Roles: []Role{RoleProvider, RoleEndpoint}, // Can also act as I/O endpoint
    }
}

// As Value (RoleProvider)
func (d *S3ObjectDecorator) Resolve(ctx ValueEvalContext, call ValueCall) (any, error) {
    // Return object metadata
    return map[string]any{
        "size": 1024,
        "etag": "abc123",
        "url":  "s3://bucket/key",
    }, nil
}

// As Sink (RoleEndpoint)
func (d *S3ObjectDecorator) Open(ctx ExecContext, mode IOType) (io.ReadWriteCloser, error) {
    // Return writer for uploading content
    return s3.NewWriter(bucket, key), nil
}
```

**Usage**:
```opal
# As value provider (metadata)
var obj = @aws.s3.object(bucket="my-bucket", key="data.json")
echo "Size: @var.obj.size bytes"

# As sink endpoint (upload)
build_artifact > @aws.s3.object(bucket="artifacts", key="build-123.tar.gz")
```

---

## 3. Unified Descriptor Metadata

All decorators expose rich, reflectable metadata for LSP, CLI, docs, and telemetry:

```go
type Descriptor struct {
    Path     string        // "env", "retry", "aws.s3.object"
    Kind     Kind          // Primary classification
    Roles    []Role        // Behavioral capabilities
    Version  string        // "1.0.0" for compatibility tracking
    Summary  string        // One-line description
    DocURL   string        // Link to full documentation
    Schema   Schema        // Parameter and return spec
    Caps     Capabilities  // Execution capabilities
}
```

### 3.1 Enhanced Schema

Schemas become the **single source of truth** for validation:

```go
type Schema struct {
    Doc          string
    PrimaryParam *Param  // Property after dot: @env.HOME
    Params       []Param
    Returns      string  // Type description (for value decorators)
}

type Param struct {
    Name        string
    Type        string   // "string", "int", "bool", "duration", etc.
    Required    bool
    Default     any      // Default value if not provided
    Enum        []any    // Allowed values (if constrained)
    Min, Max    *float64 // Bounds for numeric types
    Pattern     *string  // Regex pattern for strings
    Doc         string   // Parameter description
}
```

**Example**:
```go
schema := Schema{
    Doc: "Retry a command with exponential backoff",
    Params: []Param{
        {
            Name:     "attempts",
            Type:     "int",
            Required: false,
            Default:  3,
            Min:      ptr(1.0),
            Max:      ptr(10.0),
            Doc:      "Maximum number of attempts",
        },
        {
            Name:     "delay",
            Type:     "duration",
            Required: false,
            Default:  "1s",
            Doc:      "Initial delay between retries",
        },
        {
            Name:     "strategy",
            Type:     "string",
            Required: false,
            Default:  "exponential",
            Enum:     []any{"constant", "exponential", "linear"},
            Doc:      "Backoff strategy",
        },
    },
}
```

**Benefits**:
- ✅ LSP can provide inline documentation and completions
- ✅ CLI can validate parameters before planning
- ✅ Auto-generate documentation
- ✅ Better error messages with constraint violations

### 3.2 Capabilities

Capabilities are execution constraints and properties:

```go
type Capabilities struct {
    TransportScope TransportScope // Where decorator can be used
    Purity         bool            // Deterministic (can cache/constant-fold)
    Idempotent     bool            // Safe to retry
    IO             IOSemantics     // I/O behavior
}

type TransportScope string
const (
    ScopeRootOnly   TransportScope = "root_only"   // Local env only (@env)
    ScopeSessionEnv TransportScope = "session_env" // Uses session env (@var)
    ScopeAgnostic   TransportScope = "agnostic"    // Works anywhere (@retry)
)

type IOSemantics struct {
    PipeIn, PipeOut         bool // Supports pipe operators
    RedirectIn, RedirectOut bool // Supports redirection
    ConcurrentSafe          bool // Safe for parallel execution
    AtomicWrite             bool // Writes are atomic
}
```

**Enforcement**:
```go
func (r *Registry) ResolveValue(
    ctx ValueEvalContext,
    call ValueCall,
    currentScope TransportScope,
) (ResolvedValue, error) {
    d := r.mustLookup(call.Path, KindValue)

    // Enforce transport scope
    if !d.Caps.TransportScope.Allows(currentScope) {
        return ResolvedValue{}, fmt.Errorf(
            "@%s cannot be used in %s scope (requires %s)",
            call.Path, currentScope, d.Caps.TransportScope,
        )
    }

    // ... rest of resolution
}
```

---

## 4. Session as Ambient Context

All execution—Value, Exec, Transport, Sink—happens within a **Session**.

A Session represents:
- **Environment variables** (scoped)
- **Working directory** (scoped)
- **Execution transport** (local, SSH, Docker, K8s)
- **I/O capabilities** (filesystem, network)

```go
type Session interface {
    // Execute command with arguments
    Run(argv []string, opts RunOpts) (Result, error)

    // File operations (via transport)
    Put(data []byte, path string, mode fs.FileMode) error
    Get(path string) ([]byte, error)

    // Environment (immutable snapshot)
    Env() map[string]string
    WithEnv(delta map[string]string) Session // Copy-on-write overlay

    // Working directory
    Cwd() string

    // Cleanup
    Close() error
}
```

**Why Session?**

1. **Transport Abstraction**: Same interface works for local, SSH, Docker, K8s
2. **Scope Isolation**: Each `@ssh.connect` creates a new Session
3. **Immutability**: `WithEnv()` returns a new Session (copy-on-write)
4. **Testability**: Mock Session for testing decorators

**Example - Transport Decorator**:
```go
type SSHConnectDecorator struct{}

func (d *SSHConnectDecorator) Open(parent Session, params map[string]any) (Session, error) {
    host := params["host"].(string)
    user := params["user"].(string)

    // Create SSH session with parent's env as baseline
    sshSession := ssh.NewSession(host, user)
    sshSession.SetEnv(parent.Env()) // Inherit parent env

    return sshSession, nil
}

func (d *SSHConnectDecorator) Wrap(next ExecNode, params map[string]any) ExecNode {
    return &sshExecNode{
        host:   params["host"].(string),
        user:   params["user"].(string),
        next:   next,
    }
}
```

**Usage**:
```opal
@ssh.connect(host="prod-server", user="deploy") {
    # This runs on remote server
    # @env.HOME reads REMOTE environment
    # @var.SERVICE reads local variables (plan-time)
    systemctl restart @var.SERVICE
}
```

---

## 5. Unified Decorator Interfaces

All decorators implement `Decorator` (for reflection):

```go
type Decorator interface {
    Descriptor() Descriptor
}
```

Then specialize based on Kind/Role:

### 5.1 Value (KindValue)

```go
type Value interface {
    Decorator
    Resolve(ctx ValueEvalContext, call ValueCall) (any, error)
}

type ValueEvalContext struct {
    Session Session            // Ambient execution context
    Vars    map[string]any     // Plan-time variable bindings
    Trace   Span               // Telemetry span
}

type ValueCall struct {
    Path    string            // "env", "var", "aws.secret"
    Primary *string           // Property after dot: @env.HOME → "HOME"
    Params  map[string]any    // Named parameters (validated)
}
```

**Timing**: All value decorators resolve **at plan-time** in OPAL's model.

**Secret Wrapping**:
```go
type ResolvedValue struct {
    Value     any             // Raw value
    Handle    *secret.Handle  // Tainted handle (for scrubbing)
    DisplayID string          // "opal:s:3J98t56A"
}
```

**Example**:
```go
type EnvDecorator struct{}

func (d *EnvDecorator) Resolve(ctx ValueEvalContext, call ValueCall) (any, error) {
    envVar := *call.Primary

    // Read from session environment
    value, exists := ctx.Session.Env()[envVar]
    if !exists {
        // Check for default parameter
        if defaultVal, ok := call.Params["default"]; ok {
            return defaultVal, nil
        }
        return nil, fmt.Errorf("environment variable %q not found", envVar)
    }

    return value, nil
}

func (d *EnvDecorator) Descriptor() Descriptor {
    return Descriptor{
        Path:  "env",
        Kind:  KindValue,
        Roles: []Role{RoleProvider},
        Caps: Capabilities{
            TransportScope: ScopeRootOnly, // Can't use in @ssh blocks
            Purity:         false,          // Reads environment
        },
    }
}
```

### 5.2 Exec (KindExec)

**Middleware pattern** for wrapping execution:

```go
type Exec interface {
    Decorator
    Wrap(next ExecNode, params map[string]any) ExecNode
}

type ExecNode interface {
    Execute(ctx ExecContext) (Result, error)
}

type ExecContext struct {
    Session   Session
    Deadline  time.Time
    Cancel    context.CancelFunc
    Trace     Span
}
```

**Example - Retry Decorator**:
```go
type RetryDecorator struct{}

func (d *RetryDecorator) Wrap(next ExecNode, params map[string]any) ExecNode {
    attempts := params["attempts"].(int)
    delay := params["delay"].(time.Duration)
    strategy := params["strategy"].(string)

    return &retryNode{
        next:     next,
        attempts: attempts,
        delay:    delay,
        strategy: strategy,
    }
}

type retryNode struct {
    next     ExecNode
    attempts int
    delay    time.Duration
    strategy string
}

func (n *retryNode) Execute(ctx ExecContext) (Result, error) {
    var lastErr error

    for i := 0; i < n.attempts; i++ {
        result, err := n.next.Execute(ctx)
        if err == nil {
            return result, nil
        }

        lastErr = err

        // Backoff
        if i < n.attempts-1 {
            backoff := calculateBackoff(n.delay, i, n.strategy)
            time.Sleep(backoff)
        }
    }

    return Result{}, fmt.Errorf("retry failed after %d attempts: %w", n.attempts, lastErr)
}
```

**Composition**:
```opal
@timeout(duration=30s) {
    @retry(attempts=3, delay=2s, strategy="exponential") {
        kubectl apply -f deployment.yaml
    }
}
```

Compiles to:
```
timeout.Wrap(
    retry.Wrap(
        shellNode("kubectl apply -f deployment.yaml")
    )
)
```

### 5.3 Transport (KindTransport)

**Creates scoped execution sessions**:

```go
type Trans interface {
    Decorator
    Open(parent Session, params map[string]any) (Session, error)
    Wrap(next ExecNode, params map[string]any) ExecNode
}
```

**Note**: Transport decorators implement both `Open` (to create Session) and `Wrap` (to scope execution).

**Example - SSH Decorator**:
```go
type SSHConnectDecorator struct{}

func (d *SSHConnectDecorator) Open(parent Session, params map[string]any) (Session, error) {
    host := params["host"].(string)
    user := params["user"].(string)

    // Create SSH session
    client, err := ssh.Dial("tcp", host+":22", &ssh.ClientConfig{
        User: user,
        // ... auth config
    })
    if err != nil {
        return nil, err
    }

    return &sshSession{
        client: client,
        env:    parent.Env(), // Inherit parent env
        cwd:    "/",
    }, nil
}

func (d *SSHConnectDecorator) Wrap(next ExecNode, params map[string]any) ExecNode {
    return &sshExecNode{
        params: params,
        next:   next,
    }
}

type sshExecNode struct {
    params map[string]any
    next   ExecNode
}

func (n *sshExecNode) Execute(ctx ExecContext) (Result, error) {
    // Open SSH session
    sshSession, err := openSSHSession(n.params)
    if err != nil {
        return Result{}, err
    }
    defer sshSession.Close()

    // Create new context with SSH session
    sshCtx := ctx
    sshCtx.Session = sshSession

    // Execute child in SSH context
    return n.next.Execute(sshCtx)
}
```

### 5.4 Endpoint (KindSink)

**Unified I/O abstraction** for sources and sinks:

```go
type Endpoint interface {
    Decorator
    Open(ctx ExecContext, mode IOType) (io.ReadWriteCloser, error)
}

type IOType string
const (
    IORead   IOType = "read"   // Input source
    IOWrite  IOType = "write"  // Output sink
    IODuplex IOType = "duplex" // Bidirectional
)
```

**Example - File Sink**:
```go
type FileDecorator struct{}

func (d *FileDecorator) Open(ctx ExecContext, mode IOType) (io.ReadWriteCloser, error) {
    path := ctx.Params["path"].(string)

    switch mode {
    case IORead:
        return os.Open(path)
    case IOWrite:
        return os.Create(path)
    default:
        return nil, fmt.Errorf("unsupported I/O mode: %s", mode)
    }
}
```

**Usage**:
```opal
# Input source
@file.read(path="config.json") | jq '.database'

# Output sink
build_artifact > @file.write(path="output.tar.gz")

# S3 sink
tar czf - dist/ > @aws.s3.put(bucket="artifacts", key="build-${VERSION}.tar.gz")
```

### 5.5 Meta (KindMeta)

**Augments plan metadata** without affecting execution:

```go
type Meta interface {
    Decorator
    Apply(plan *Plan, nodeID uint64, params map[string]any) error
}
```

**Example - Trace Decorator**:
```go
type TraceDecorator struct{}

func (d *TraceDecorator) Apply(plan *Plan, nodeID uint64, params map[string]any) error {
    name := params["name"].(string)

    // Add telemetry annotation to plan
    plan.Annotations = append(plan.Annotations, Annotation{
        NodeID: nodeID,
        Kind:   "trace",
        Data:   map[string]any{"span_name": name},
    })

    return nil
}
```

**Usage**:
```opal
@trace(name="deploy-app") {
    kubectl apply -f k8s/
    kubectl rollout status deployment/app
}
```

### 5.6 Lifecycle (Optional)

**Optional interface** for decorators that need lifecycle hooks:

```go
type Lifecycle interface {
    Before(ctx ExecContext) error
    After(ctx ExecContext, result Result) error
}
```

**Use Cases**:
- `@measure` - Record timing metrics
- `@trace` - Create spans
- `@audit` - Log execution events

---

## 6. Multi-Kind Registration

Instead of rigid fields, decorators register as **tagged interface unions**:

```go
type Entry struct {
    Impl Decorator // May implement multiple interfaces
}

type Registry interface {
    Register(e Entry) error
    Lookup(path string) (Entry, bool)
    Export() []Descriptor // For tooling/docs
}
```

**Runtime Assertions**:
```go
func (r *Registry) ResolveValue(ctx ValueEvalContext, call ValueCall) (ResolvedValue, error) {
    entry, ok := r.Lookup(call.Path)
    if !ok {
        return ResolvedValue{}, fmt.Errorf("decorator @%s not found", call.Path)
    }

    // Assert Value interface
    value, ok := entry.Impl.(Value)
    if !ok {
        return ResolvedValue{}, fmt.Errorf("@%s is not a value decorator", call.Path)
    }

    // Check capabilities
    desc := entry.Impl.Descriptor()
    if !desc.Caps.TransportScope.Allows(ctx.CurrentScope) {
        return ResolvedValue{}, ErrScopeViolation
    }

    // Validate params against schema
    if err := validateParams(call.Params, desc.Schema); err != nil {
        return ResolvedValue{}, err
    }

    // Call handler
    rawValue, err := value.Resolve(ctx, call)
    if err != nil {
        return ResolvedValue{}, err
    }

    // Wrap as secret (defense in depth)
    handle := wrapAsSecret(rawValue, ctx, call)

    return ResolvedValue{
        Value:     rawValue,
        Handle:    handle,
        DisplayID: handle.ID(),
    }, nil
}
```

---

## 7. Planning & Execution Pipeline

The decorator model integrates cleanly with OPAL's existing pipeline:

| Layer | Responsibility |
|-------|----------------|
| **Parser** | Parse source into syntactic nodes (Call, Redirect, Pipeline) |
| **Analyzer** | Resolve decorator kinds, enforce scope and I/O semantics |
| **Planner** | Lower into planfmt trees with session and redirect awareness |
| **Executor** | Stream data between decorators via Sessions and Endpoints |
| **Telemetry** | Capture spans and lifecycle hooks |

### 7.1 Plan-Time Resolution (Value Decorators)

```
Source: var REPLICAS = 3; kubectl scale --replicas=@var.REPLICAS deployment/app
   ↓
[Parser] Recognizes @var.REPLICAS as decorator reference
   ↓
[Analyzer] Looks up @var in registry, checks scope, validates params
   ↓
[Planner] Calls registry.ResolveValue()
   ↓ Returns: ResolvedValue{Value: 3, DisplayID: "opal:s:7Xm2Kp9"}
   ↓
[Planner] Substitutes into command string
   ↓
Plan: @shell("kubectl scale --replicas=3 deployment/app")
      Secrets: [{DisplayID: "opal:s:7Xm2Kp9", Value: <handle>}]
```

### 7.2 Execution-Time Wrapping (Exec Decorators)

```
Source: @timeout(duration=30s) { @retry(attempts=3) { kubectl apply -f k8s/ } }
   ↓
[Parser] Builds nested decorator tree
   ↓
[Planner] Stores decorator chain in plan
   ↓
Plan: Step{
        Tree: CommandNode{
          Decorator: "@timeout",
          Params: {duration: 30s},
          Block: CommandNode{
            Decorator: "@retry",
            Params: {attempts: 3},
            Block: CommandNode{
              Decorator: "@shell",
              Args: {command: "kubectl apply -f k8s/"}
            }
          }
        }
      }
   ↓
[Executor] Constructs middleware chain:
   shell := &shellNode{cmd: "kubectl apply -f k8s/"}
   retry := retryDecorator.Wrap(shell, {attempts: 3})
   timeout := timeoutDecorator.Wrap(retry, {duration: 30s})
   ↓
[Executor] Executes: timeout.Execute(ctx)
```

---

## 8. Example: AWS S3 Decorator Suite

Real-world example showing multi-role decorators:

| Decorator | Kinds | Roles | Description |
|-----------|-------|-------|-------------|
| `@aws.s3.object` | Value + Sink | Provider + Endpoint | Resolves metadata or uploads content |
| `@aws.s3.session` | Transport | Boundary | Opens temporary S3 session credentials |
| `@aws.s3.list` | Exec | Wrapper | Streams S3 listing |
| `@aws.s3.url` | Value | Provider | Generates presigned URL |

**Usage**:
```opal
# Open AWS session with temporary credentials
@aws.s3.session(role="arn:aws:iam::123:role/deployer") {

    # Get object metadata (Value role)
    var obj = @aws.s3.object(bucket="artifacts", key="build-123.tar.gz")
    echo "Size: @var.obj.size bytes"
    echo "ETag: @var.obj.etag"

    # Upload file (Sink role)
    tar czf - dist/ > @aws.s3.object(bucket="artifacts", key="build-${VERSION}.tar.gz")

    # Generate presigned URL (Value role)
    var url = @aws.s3.url(bucket="artifacts", key="build-${VERSION}.tar.gz", expires=3600)
    echo "Download: @var.url"

    # List objects (Exec role)
    @aws.s3.list(bucket="artifacts", prefix="build-") {
        echo "Found: ${key}"
    }
}
```

---

## 9. Design Rules

The refined architecture follows these invariants:

1. **Session is the authority boundary** — Nothing crosses it implicitly
   - All execution happens within a Session
   - Transport decorators create new Sessions
   - Sessions are immutable (copy-on-write)

2. **Outside-in wrapping** — Decorator order is explicit
   - `@timeout{ @retry{ @shell } }` wraps timeout → retry → shell
   - No implicit reordering

3. **Pipelines cannot span transports**
   - `local_cmd | @ssh.connect{ remote_cmd }` is invalid
   - Use explicit `@file.transfer` or `@ssh.scp`

4. **Decorators are pure unless explicitly stateful**
   - Value decorators default to `Purity: true`
   - Side effects declared in capabilities

5. **Schema is authoritative**
   - Validation, docs, and tooling use schema as source of truth
   - No scattered validation logic

6. **Telemetry is composable**
   - Any decorator can emit spans via `Lifecycle`
   - Spans nest automatically (parent-child relationships)

---

## 10. Naming & Aesthetics

**Conventions**:
- Prefer **verb-centric subpaths**: `@fs.write`, `@s3.put`, `@net.send`
- Keep **syntax minimal**: all semantics flow from descriptors and schema
- Emphasize **composability**: everything `.Wraps` or `.Resolves`

**Examples**:
- ✅ Good: `@fs.read`, `@http.get`, `@aws.secret.get`
- ❌ Avoid: `@readFile`, `@getHTTP`, `@getAWSSecret` (noun-heavy, redundant)

**Namespacing**:
- Use dot-separated paths for organization: `aws.s3.put`, `gcp.storage.put`
- Built-ins have short names: `env`, `var`, `retry`, `timeout`
- Plugins use vendor prefix: `hashicorp.vault.secret`, `datadog.span`

---

## 11. Future Decorator Families

This design enables rich decorator ecosystems:

| Domain | Example Decorators | Kind(s) | Roles |
|--------|-------------------|---------|-------|
| **Compute/Control** | `@map`, `@reduce`, `@each`, `@invariant` | Exec | Wrapper |
| **Crypto/Security** | `@sign`, `@verify`, `@seal`, `@unseal` | Value / Endpoint | Provider / Endpoint |
| **Observability** | `@trace`, `@measure`, `@span` | Meta | Annotate |
| **Infra Access** | `@aws.session`, `@gcp.session`, `@vault.session` | Transport | Boundary |
| **Artifacts** | `@artifact.put`, `@artifact.get`, `@artifact.meta` | Sink / Value | Endpoint / Provider |
| **Databases** | `@postgres.query`, `@redis.get`, `@dynamodb.put` | Value / Sink | Provider / Endpoint |
| **Messaging** | `@kafka.produce`, `@sqs.send`, `@pubsub.publish` | Exec / Sink | Wrapper / Endpoint |

---

## 12. Evaluation Against Current Implementation

### 12.1 Compatibility with Existing Code

**Current State** ([DECORATOR_FLOW_CURRENT_STATE.md](./DECORATOR_FLOW_CURRENT_STATE.md)):
- ✅ Decorator registration works (`init()` pattern)
- ✅ Parser recognizes decorators and validates parameters
- ✅ Plan structure captures decorator info
- ❌ Handlers are registered but never called
- ❌ Value resolution not implemented
- ❌ Execution decorators planned but not executed

**Migration Path**:

**Phase 1: Extend Current Interfaces** (Low Risk)
```go
// Add new interfaces alongside current ones
type Decorator interface {
    Descriptor() Descriptor
}

type Value interface {
    Decorator
    Resolve(ctx ValueEvalContext, call ValueCall) (any, error)
}

// Existing decorators can implement both old and new interfaces
type EnvDecorator struct{}

// Old interface (keep for compatibility)
func (d *EnvDecorator) Handle(ctx types.Context, args types.Args) (types.Value, error) {
    // Delegate to new interface
    return d.Resolve(ValueEvalContext{...}, ValueCall{...})
}

// New interface
func (d *EnvDecorator) Resolve(ctx ValueEvalContext, call ValueCall) (any, error) {
    // New implementation
}
```

**Phase 2: Implement Value Resolution** (First Real Change)
- Add `ResolveValue()` to planner
- Wire up secret wrapping
- Test with `@var` and `@env`

**Phase 3: Migrate Exec Decorators to Middleware**
- Implement `Wrap()` for `@retry`, `@timeout`
- Build executor middleware chain
- Deprecate old `ExecutionHandler`

**Phase 4: Add New Decorator Kinds**
- Implement `Transport` for `@ssh.connect`
- Implement `Endpoint` for I/O decorators
- Add `Meta` for telemetry

### 12.2 Secret Infrastructure Integration

The proposed design **integrates seamlessly** with existing secret infrastructure:

**Current Secret System** ([SECRETS_AND_ENV_HANDLING_ANALYSIS.md](./SECRETS_AND_ENV_HANDLING_ANALYSIS.md)):
- ✅ Two-track identity (DisplayID + Fingerprint)
- ✅ BLAKE2s-128 keyed PRF for deterministic IDs
- ✅ Capability-gated access
- ✅ Stream scrubbing ready

**Integration Point**:
```go
func (r *Registry) ResolveValue(ctx ValueEvalContext, call ValueCall) (ResolvedValue, error) {
    // ... resolve value ...

    // Wrap as secret using existing infrastructure
    handle := secret.NewHandleWithFactory(value, ctx.IDFactory, secret.IDContext{
        PlanHash:  ctx.PlanHash,
        StepPath:  ctx.StepPath,
        Decorator: call.Path,
        KeyName:   *call.Primary,
        Kind:      "s",
    })

    return ResolvedValue{
        Value:     value,
        Handle:    handle,           // Existing type
        DisplayID: handle.ID(),      // "opal:s:3J98t56A"
    }, nil
}
```

**No Changes Required** to secret infrastructure! ✅

### 12.3 OEP Compatibility

| OEP | Feature | Compatibility | Notes |
|-----|---------|---------------|-------|
| **OEP-001** | Let bindings | ✅ Excellent | `Value.Resolve()` supports structured returns |
| **OEP-002** | Pipeline ops | ✅ Good | Can add `KindPipeOp` with dedicated interface |
| **OEP-010** | IaC | ✅ Excellent | Transport + Endpoint support deploy blocks |
| **OEP-012** | Plugins | ✅ Excellent | `Descriptor.Version` + namespaced paths |

---

## 13. Benefits Summary

### 13.1 For Users

| Benefit | Example |
|---------|---------|
| **Richer Semantics** | `@aws.s3.object` can both provide metadata AND upload content |
| **Better Error Messages** | "Cannot use @env inside @ssh block (scope violation)" |
| **LSP Support** | Inline docs, completions, and warnings |
| **Predictable Behavior** | Clear scope rules, no implicit magic |

### 13.2 For Contributors

| Benefit | Example |
|---------|---------|
| **Clear Contracts** | `Descriptor` is authoritative |
| **Testable** | Mock `Session` for decorator tests |
| **Composable** | Middleware pattern is explicit |
| **Extensible** | Add new Kinds without breaking existing code |

### 13.3 For the Project

| Benefit | Example |
|---------|---------|
| **Plugin Ecosystem** | Clear ABI for third-party decorators |
| **Documentation** | Auto-generate from `Descriptor.Schema` |
| **Observability** | Built-in telemetry hooks |
| **Performance** | Pure decorators can be cached/optimized |

---

## 14. Open Questions

### Q1: Should Session be an interface or struct?

**Option A: Interface** (Proposed)
```go
type Session interface {
    Run(argv []string, opts RunOpts) (Result, error)
    Put(data []byte, path string, mode fs.FileMode) error
    Get(path string) ([]byte, error)
    Env() map[string]string
    WithEnv(delta map[string]string) Session
    Cwd() string
    Close() error
}
```

**Pros**:
- ✅ Testable (easy to mock)
- ✅ Extensible (new transports just implement interface)
- ✅ Clean abstraction boundary

**Cons**:
- ⚠️ Interface{} return from `Env()` requires casting
- ⚠️ More complex than simple struct

**Option B: Struct with Transport Field**
```go
type Session struct {
    Transport Transport // LocalTransport, SSHTransport, etc.
    Env       map[string]string
    Cwd       string
}
```

**Recommendation**: **Option A (Interface)** for better testability and abstraction.

### Q2: How to handle decorator composition conflicts?

**Scenario**: `@timeout{ @retry{ ... } }` - which timeout applies to each retry attempt?

**Options**:
1. **Timeout wraps all retries** (outer wins)
2. **Timeout applies per retry** (inner wins)
3. **Explicit control** (decorator params)

**Recommendation**: **Option 1 (outer wins)** - matches user intuition. Can add `@retry(timeout=...)` for per-attempt timeout.

### Q3: Should Value.Resolve() return `any` or `Value` union?

**Option A: `any` (interface{})** (Proposed)
```go
Resolve(ctx ValueEvalContext, call ValueCall) (any, error)
```

**Pros**: Simple, flexible, matches Go idioms

**Option B: Discriminated Union**
```go
type Value struct {
    Kind ValueKind
    Str  string
    Int  int64
    Bool bool
    Obj  map[string]any
}

Resolve(ctx ValueEvalContext, call ValueCall) (Value, error)
```

**Pros**: Type-safe, explicit

**Recommendation**: **Option A (`any`)** for simplicity. Schema provides type information.

### Q4: How to version decorators?

**Current Proposal**: `Descriptor.Version` as semver string.

**Questions**:
- Should registry support multiple versions of same decorator?
- How to handle breaking changes?
- Should plans embed decorator versions for reproducibility?

**Recommendation**: Start with single version per decorator, add multi-version support later if needed.

---

## 15. Next Steps

### Immediate (Week 1-2)

1. **Review & Feedback**
   - Circulate this RFC to core contributors
   - Gather feedback on design choices
   - Resolve open questions

2. **Prototype Session Abstraction**
   - Implement `Session` interface
   - Create `LocalSession` implementation
   - Write tests

### Short-Term (Week 3-6)

3. **Implement Value Resolution**
   - Add new `Value` interface
   - Implement for `@var` and `@env`
   - Wire up planner resolution pass
   - Integrate secret wrapping
   - Test string interpolation

4. **Update Documentation**
   - Update [DECORATOR_GUIDE.md](./DECORATOR_GUIDE.md)
   - Add examples to [SPECIFICATION.md](./SPECIFICATION.md)
   - Document migration path

### Medium-Term (Week 7-12)

5. **Implement Middleware Pattern**
   - Add `Exec` interface
   - Migrate `@retry` and `@timeout`
   - Build executor middleware chain
   - Test composition

6. **Add Transport Support**
   - Implement `Trans` interface
   - Create `@ssh.connect` decorator
   - Test scope isolation

### Long-Term (Month 4+)

7. **Add Endpoint Support**
   - Implement `Endpoint` interface
   - Create `@file.read`, `@file.write`
   - Test I/O redirection

8. **Build Plugin System**
   - Formalize plugin API
   - Add versioning support
   - Document plugin authoring

---

## 16. Conclusion

This refined decorator architecture addresses current limitations while maintaining OPAL's philosophy of **minimal, composable design**. The key innovations are:

1. **Kind vs Role separation** - Enables multi-role decorators without proliferating types
2. **Session abstraction** - Unifies execution across transports
3. **Middleware pattern** - Makes composition explicit and testable
4. **Capability-driven** - Enforces scope and semantics centrally

The design is **immediately implementable** and provides a clear migration path from the current state. It positions OPAL for rich decorator ecosystems while keeping the core simple and focused.

**Recommendation**: Adopt this direction and begin implementation with value decorator resolution.

---

## Appendix A: Complete Interface Reference

```go
// Core taxonomy
type Kind string
const (
    KindValue     Kind = "value"
    KindExec      Kind = "exec"
    KindTransport Kind = "transport"
    KindSink      Kind = "sink"
    KindMeta      Kind = "meta"
)

type Role string
const (
    RoleProvider Role = "provider"
    RoleWrapper  Role = "wrapper"
    RoleBoundary Role = "boundary"
    RoleEndpoint Role = "endpoint"
    RoleAnnotate Role = "annotate"
)

// Descriptor
type Descriptor struct {
    Path         string
    Kind         Kind
    Roles        []Role
    Version      string
    Summary      string
    DocURL       string
    Schema       Schema
    Capabilities Capabilities
}

// Schema
type Schema struct {
    Doc          string
    PrimaryParam *Param
    Params       []Param
    Returns      string
}

type Param struct {
    Name     string
    Type     string
    Required bool
    Default  any
    Enum     []any
    Min, Max *float64
    Pattern  *string
    Doc      string
}

// Capabilities
type Capabilities struct {
    TransportScope TransportScope
    Purity         bool
    Idempotent     bool
    IO             IOSemantics
}

type TransportScope string
const (
    ScopeRootOnly   TransportScope = "root_only"
    ScopeSessionEnv TransportScope = "session_env"
    ScopeAgnostic   TransportScope = "agnostic"
)

type IOSemantics struct {
    PipeIn, PipeOut         bool
    RedirectIn, RedirectOut bool
    ConcurrentSafe          bool
    AtomicWrite             bool
}

// Session
type Session interface {
    Run(argv []string, opts RunOpts) (Result, error)
    Put(data []byte, path string, mode fs.FileMode) error
    Get(path string) ([]byte, error)
    Env() map[string]string
    WithEnv(delta map[string]string) Session
    Cwd() string
    Close() error
}

// Base interface
type Decorator interface {
    Descriptor() Descriptor
}

// Value
type Value interface {
    Decorator
    Resolve(ctx ValueEvalContext, call ValueCall) (any, error)
}

type ValueEvalContext struct {
    Session   Session
    Vars      map[string]any
    IDFactory secret.IDFactory
    PlanHash  []byte
    StepPath  string
    Trace     Span
}

type ValueCall struct {
    Path    string
    Primary *string
    Params  map[string]any
}

type ResolvedValue struct {
    Value     any
    Handle    *secret.Handle
    DisplayID string
}

// Exec
type Exec interface {
    Decorator
    Wrap(next ExecNode, params map[string]any) ExecNode
}

type ExecNode interface {
    Execute(ctx ExecContext) (Result, error)
}

type ExecContext struct {
    Session  Session
    Deadline time.Time
    Cancel   context.CancelFunc
    Trace    Span
}

// Transport
type Trans interface {
    Decorator
    Open(parent Session, params map[string]any) (Session, error)
    Wrap(next ExecNode, params map[string]any) ExecNode
}

// Endpoint
type Endpoint interface {
    Decorator
    Open(ctx ExecContext, mode IOType) (io.ReadWriteCloser, error)
}

type IOType string
const (
    IORead   IOType = "read"
    IOWrite  IOType = "write"
    IODuplex IOType = "duplex"
)

// Meta
type Meta interface {
    Decorator
    Apply(plan *Plan, nodeID uint64, params map[string]any) error
}

// Lifecycle (optional)
type Lifecycle interface {
    Before(ctx ExecContext) error
    After(ctx ExecContext, result Result) error
}

// Registry
type Entry struct {
    Impl Decorator
}

type Registry interface {
    Register(e Entry) error
    Lookup(path string) (Entry, bool)
    Export() []Descriptor

    // Specialized resolution
    ResolveValue(ctx ValueEvalContext, call ValueCall, scope TransportScope) (ResolvedValue, error)
}
```

---

## Appendix B: References

### Related Documents
- [DECORATOR_ARCHITECTURE_REVIEW.md](./DECORATOR_ARCHITECTURE_REVIEW.md) - Current state analysis
- [DECORATOR_FLOW_CURRENT_STATE.md](./DECORATOR_FLOW_CURRENT_STATE.md) - Layer-by-layer flow
- [SECRETS_AND_ENV_HANDLING_ANALYSIS.md](./SECRETS_AND_ENV_HANDLING_ANALYSIS.md) - Secret infrastructure
- [VALUE_DECORATOR_ANALYSIS.md](./VALUE_DECORATOR_ANALYSIS.md) - Implementation gaps

### OEP References
- OEP-001: Runtime Variable Binding with `let`
- OEP-002: Transform Pipeline Operator `|>`
- OEP-010: Infrastructure as Code (IaC)
- OEP-012: Module Composition and Plugin System

---

**Status**: RFC - Awaiting Feedback
**Next Review**: [Date TBD]
**Reviewers**: Core Contributors
