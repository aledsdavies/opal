# OPAL Secrets and Environment Variables Infrastructure Analysis

## Executive Summary

This document analyzes the current infrastructure for handling secrets and environment variables in OPAL, focusing on:
1. Secret infrastructure and DisplayID tracking
2. Executor context design
3. Environment variable handling and "always treated as secret" semantics
4. Integration points between planner and SDK
5. Key patterns and examples for value decorator resolution

---

## 1. SECRET INFRASTRUCTURE IN core/sdk/secret/

### 1.1 Two-Track Identity Pattern

**Location**: `/home/user/opal/core/sdk/secret/handle.go`

OPAL implements a clever two-track identity system for secrets:

#### Display Track (User-Visible)
- **DisplayID**: Opaque identifier like `opal:s:3J98t56A`
- **Purpose**: Shown in logs, plans, output summaries
- **Properties**: No length leak, no correlation between runs
- **Access Method**: `Handle.ID()` or `Handle.Placeholder()`

```go
// Get display ID for logs/output
id := secretHandle.ID()  // Returns: "opal:s:3J98t56A"
idWithEmoji := secretHandle.IDWithEmoji()  // Returns: "🔒 opal:s:3J98t56A"
```

#### Internal Track (Runtime-Only)
- **Fingerprint**: BLAKE2b-256 keyed hash for scrubber matching
- **Purpose**: Used by stream scrubber to detect/replace secrets
- **Properties**: Per-run key prevents correlation across runs
- **Access Method**: `Handle.Fingerprint(key []byte)`

```go
// Get internal fingerprint for scrubber (NOT user-visible)
fp := secretHandle.Fingerprint(runKey)  // Returns: hex string of hash
```

### 1.2 Handle Generation and Lifecycle

**File**: `/home/user/opal/core/sdk/secret/handle.go` (lines 43-82)

#### Creation Paths

**Path 1: Runtime Creation (ModeRun)**
```go
// For direct execution - random per run
h := secret.NewHandle(value)  // Random displayID, non-deterministic
```

**Path 2: Plan-Time Creation (ModePlan)**
```go
// For resolved plans - deterministic, context-aware
factory := secret.NewIDFactory(secret.ModePlan, derivedKey)
ctx := secret.IDContext{
    PlanHash:  planHash,           // e.g., "abc123def456"
    StepPath:  "deploy.step[0]",   // e.g., deployment step location
    Decorator: "@env",             // which decorator
    KeyName:   "DB_PASSWORD",      // which env var
    Kind:      "s",                // "s" for secret
}
h := secret.NewHandleWithFactory(value, factory, ctx)
// Result: Deterministic displayID for contract verification
```

### 1.3 IDFactory and DisplayID Generation

**Location**: `/home/user/opal/core/sdk/secret/idfactory.go`

#### IDFactory Interface
```go
type IDFactory interface {
    Make(ctx IDContext, value []byte) string
}
```

#### Implementation: keyedIDFactory
Uses **BLAKE2s-128 keyed PRF** for DisplayID generation:

```
PRF(key, input) = BLAKE2s-128(key, planhash || context || BLAKE2b-256(value))

Where:
  - key: 32-byte key (from PSE or random)
  - planhash: Canonical plan digest
  - context: step_path || decorator || key_name || kind
  - BLAKE2b-256(value): Hash of value (prevents length leak)
  - Output: base58(first 8 bytes) → "opal:s:3J98t56A"
```

**Code Example** (lines 68-102):
```go
func (f *keyedIDFactory) Make(ctx IDContext, value []byte) string {
    var input bytes.Buffer
    
    // Build deterministic input
    input.Write(ctx.PlanHash)
    input.WriteString(ctx.StepPath)
    input.WriteString("\x00")
    input.WriteString(ctx.Decorator)
    // ... more context fields
    
    // Hash of value (prevents length leak)
    valueHash := blake2b.Sum256(value)
    input.Write(valueHash[:])
    
    // Keyed PRF with BLAKE2s-128
    digest, _ := blake2s.New128(f.key)
    digest.Write(input.Bytes())
    hash := digest.Sum(nil)
    
    // Base58 encode first 8 bytes
    encoded := EncodeBase58(hash[:8])
    return fmt.Sprintf("opal:%s:%s", ctx.Kind, encoded)
}
```

### 1.4 Placeholder Format

**Format**: `opal:[kind]:[base58]`

**Examples**:
- `opal:s:3J98t56A` - Secret (from @env, @aws.secret, etc.)
- `opal:v:5K2Lm9Np` - Value (from @var, @git.commit_hash)
- `opal:st:7Qr4Vxyz` - Step result
- `opal:pl:8Tuv5Wab` - Plan signature

**Base58 Encoding** (lines 6-48 in `base58.go`):
- Input: 8 bytes (64 bits)
- Output: ~11 characters (base58 is ~1.4 bits per character)
- Alphabet: Bitcoin-style (no 0/O/I/l ambiguity)

### 1.5 Capability-Gated Access

**File**: `/home/user/opal/core/sdk/secret/handle.go` (lines 18-31, 132-169)

```go
type Capability struct {
    token uint64  // Opaque token
}

// Global capability (set by executor at runtime)
var globalCapability *Capability

func SetCapability(cap *Capability) {
    globalCapability = cap
}

// Access methods require capability
func (h *Handle) UnsafeUnwrap() string {
    if DebugMode {
        panic("UnsafeUnwrap() in debug mode")
    }
    if globalCapability == nil {
        panic("UnsafeUnwrap() requires capability")
    }
    return h.value
}

func (h *Handle) ForEnv(key string) string {
    if globalCapability == nil {
        panic("ForEnv() requires capability")
    }
    return key + "=" + h.value
}

func (h *Handle) Bytes() []byte {
    if globalCapability == nil {
        panic("Bytes() requires capability")
    }
    return []byte(h.value)
}
```

**Security Model**:
- Only executor can call `SetCapability()`
- Decorators running in executor context have capability
- Unauthorized calls panic in debug mode, prevent leaks in production
- Safe alternatives: `Mask()`, `UnwrapWithMask()`, `UnwrapLast4()`

### 1.6 Safe Display Methods

```go
// Safe for logging (shows: sec***123)
masked := secretHandle.UnwrapWithMask()

// Safe for partial display (shows: ...-123)
last4 := secretHandle.UnwrapLast4()

// Custom masking (shows: se***23 for n=2)
custom := secretHandle.Mask(2)

// Safe for all %v, %s, %#v formatting
fmt.Printf("%v", secretHandle)  // Prints: opal:s:3J98t56A
```

### 1.7 Integration with Plan Representation

**File**: `/home/user/opal/core/planfmt/plan.go` (lines 32-43)

```go
type Secret struct {
    Key          string  // Variable name (e.g., "DB_PASSWORD", "HOME")
    RuntimeValue string  // Actual resolved value (runtime only)
    DisplayID    string  // Opaque ID for display: "opal:s:3J98t56A"
}

type Plan struct {
    Header  PlanHeader
    Target  string
    Steps   []Step
    Secrets []Secret  // ALL resolved value decorators
}
```

**Key**: Plan.Secrets contains **ALL** resolved values (not just sensitive ones):
- `@env.HOME` → Secret
- `@var.SERVICE_NAME` → Secret
- `@git.commit_hash` → Secret
- **Even non-sensitive values are scrubbed** (defense in depth)

---

## 2. EXECUTOR CONTEXT IN core/sdk/executor/

### 2.1 Transport Abstraction

**File**: `/home/user/opal/core/sdk/executor/transport.go` (lines 66-130)

```go
type Transport interface {
    // Execute command
    Exec(ctx context.Context, argv []string, opts ExecOpts) (exitCode int, err error)
    
    // File operations
    Put(ctx context.Context, src io.Reader, dst string, mode fs.FileMode) error
    Get(ctx context.Context, src string, dst io.Writer) error
    OpenFileWriter(ctx context.Context, path string, mode RedirectMode, perm fs.FileMode) (io.WriteCloser, error)
    
    // Cleanup
    Close() error
}

type ExecOpts struct {
    Stdin  io.Reader              // Command input
    Stdout io.Writer              // Command output (scrubbed)
    Stderr io.Writer              // Command error output (scrubbed)
    Env    map[string]string      // Decorator-added environment only
    Dir    string                 // Working directory
}
```

**Environment Isolation Contract** (lines 141-151):
```
ExecOpts.Env contains ONLY decorator-added variables.
Each Transport merges these with ITS OWN base environment:

  LocalTransport:  base = os.Environ() (local machine)
  SSHTransport:    base = remote server's environment (via SSH)
  DockerTransport: base = container's environment (via docker exec)

LOCAL environment NEVER leaks to remote transports.
Only decorator-added variables cross transport boundaries.
```

### 2.2 LocalTransport Implementation

**File**: `/home/user/opal/core/sdk/executor/transport.go` (lines 189-281)

```go
func (t *LocalTransport) Exec(ctx context.Context, argv []string, opts ExecOpts) (int, error) {
    cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
    
    // Use local environment as base
    if len(opts.Env) > 0 {
        // CRITICAL: Use os.Environ() as base
        // This includes PATH, HOME, LANG, etc.
        // Decorator variables override base
        cmd.Env = MergeEnvironment(os.Environ(), opts.Env)
    }
    
    // Wire I/O through scrubber
    cmd.Stdin = opts.Stdin
    cmd.Stdout = opts.Stdout  // Already scrubbed
    cmd.Stderr = opts.Stderr  // Already scrubbed
    
    // Execute
    if err := cmd.Run(); err != nil {
        // Handle context cancellation (exit code 124)
        if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
            return ExitTimeout, nil
        }
        // ... handle other errors
    }
    return ExitSuccess, nil
}
```

### 2.3 Command Wrapper

**File**: `/home/user/opal/core/sdk/executor/command.go` (lines 14-175)

```go
type Cmd struct {
    transport Transport         // LocalTransport by default
    ctx       context.Context
    argv      []string
    stdin     io.Reader
    stdout    io.Writer         // Locked down by CLI, scrubbed
    stderr    io.Writer         // Locked down by CLI, scrubbed
    env       map[string]string // Decorator-added only
    dir       string
}

// Safe creation (default to LocalTransport)
func CommandContext(ctx context.Context, name string, args ...string) *Cmd {
    return &Cmd{
        transport: &LocalTransport{},
        ctx:       ctx,
        argv:      []string{name, args...},
        stdout:    os.Stdout,  // Locked down by CLI
        stderr:    os.Stderr,  // Locked down by CLI
        env:       make(map[string]string),
    }
}

// Environment composition
func (c *Cmd) AppendEnv(kv map[string]string) *Cmd {
    for k, v := range kv {
        c.env[k] = v  // Store for transport merging
    }
    return c
}

// Execution (transport decides environment merging)
func (c *Cmd) Run() (int, error) {
    opts := ExecOpts{
        Stdin:  c.stdin,
        Stdout: c.stdout,
        Stderr: c.stderr,
        Env:    c.env,  // Transport will merge with base
        Dir:    c.dir,
    }
    return c.transport.Exec(c.ctx, c.argv, opts)
}
```

### 2.4 ExecutionContext Interface

**File**: `/home/user/opal/core/sdk/execution.go` (lines 302-383)

```go
type ExecutionContext interface {
    // Execute nested steps
    ExecuteBlock(steps []Step) (exitCode int, err error)
    
    // Go context for cancellation/deadlines
    Context() context.Context
    
    // Decorator arguments (immutable snapshot)
    ArgString(key string) string
    ArgInt(key string) int64
    ArgBool(key string) bool
    ArgDuration(key string) time.Duration
    Args() map[string]interface{}
    
    // Captured at context creation time (immutable)
    Environ() map[string]string  // All environment variables
    Workdir() string             // Current working directory
    
    // Copy-on-write context modifications
    WithContext(ctx context.Context) ExecutionContext
    WithEnviron(env map[string]string) ExecutionContext
    WithWorkdir(dir string) ExecutionContext
    
    // I/O for pipes
    Stdin() io.Reader
    StdoutPipe() io.Writer
    
    // Cloning for child commands
    Clone(args map[string]interface{}, stdin io.Reader, stdoutPipe io.Writer) ExecutionContext
    
    // Transport for file operations
    Transport() interface{}  // Actually executor.Transport
}
```

### 2.5 Runtime ExecutionContext Implementation

**File**: `/home/user/opal/runtime/executor/context.go` (lines 18-265)

```go
type executionContext struct {
    executor   *executor
    args       map[string]interface{}
    ctx        context.Context
    environ    map[string]string  // Immutable snapshot
    workdir    string
    stdin      io.Reader
    stdoutPipe io.Writer
}

// Create context - captures environment at creation time
func newExecutionContext(args map[string]interface{}, exec *executor, ctx context.Context) sdk.ExecutionContext {
    wd, _ := os.Getwd()
    
    return &executionContext{
        executor:   exec,
        args:       args,
        ctx:        ctx,
        environ:    captureEnviron(),  // Snapshot here
        workdir:    wd,
    }
}

// Immutable copy-on-write
func (e *executionContext) WithEnviron(env map[string]string) sdk.ExecutionContext {
    envCopy := make(map[string]string, len(env))
    for k, v := range env {
        envCopy[k] = v
    }
    
    return &executionContext{
        executor:   e.executor,
        args:       e.args,
        ctx:        e.ctx,
        environ:    envCopy,      // New copy
        workdir:    e.workdir,
        stdin:      e.stdin,
        stdoutPipe: e.stdoutPipe,
    }
}

// Cloning - inherits environment
func (e *executionContext) Clone(args map[string]interface{}, stdin io.Reader, stdoutPipe io.Writer) sdk.ExecutionContext {
    return &executionContext{
        executor:   e.executor,
        args:       args,          // NEW
        ctx:        e.ctx,         // INHERIT
        environ:    e.environ,     // INHERIT
        workdir:    e.workdir,     // INHERIT
        stdin:      stdin,         // NEW
        stdoutPipe: stdoutPipe,    // NEW
    }
}

// Capture current environment
func captureEnviron() map[string]string {
    env := make(map[string]string)
    for _, e := range os.Environ() {
        if idx := strings.IndexByte(e, '='); idx > 0 {
            env[e[:idx]] = e[idx+1:]
        }
    }
    return env
}
```

---

## 3. ENVIRONMENT HANDLING AND FRESH RESOLUTION

### 3.1 Environment Capture Semantics

**Key Principle**: Environment is captured as an **immutable snapshot** at context creation time.

**Location**: `/home/user/opal/runtime/executor/context.go` (lines 256-265)

```go
// Environment snapshot - created once, never changes
func captureEnviron() map[string]string {
    env := make(map[string]string)
    for _, e := range os.Environ() {
        if idx := strings.IndexByte(e, '='); idx > 0 {
            env[e[:idx]] = e[idx+1:]  // Parse KEY=VALUE
        }
    }
    return env  // Immutable copy
}
```

**Why Immutable**:
1. **Isolation** - Changes to `os.Setenv()` don't affect running commands
2. **Predictability** - Same context gives same results
3. **Testability** - Environment frozen at specific point
4. **Thread-safety** - No race conditions

### 3.2 Fresh Resolution Semantics

For value decorators like `@env`:

#### Plan-Time Resolution
```
WHEN: During planning (before execution)
WHERE: Planner captures current os.Environ()
RESULT: Deterministic DisplayID in plan (for contract verification)

Example:
  Step 1: Plan created, captures HOME=/home/alice → DisplayID = opal:s:XYZ
  Step 2: Shell environment changed, HOME=/home/bob
  Step 3: Plan executed, contract verified with captured DisplayID
  Result: Both decorator and environment unchanged
```

#### Execution-Time Resolution (Fresh)
```
WHEN: During @shell execution
WHERE: Shell process gets environment from ExecutionContext
RESULT: Fresh capture at execution time, may differ from plan

Example:
  Step 1: Plan says @shell("echo $HOME")
  Step 2: Execution context has HOME=/home/bob (fresh at exec time)
  Step 3: Shell outputs /home/bob (different from plan-time value)
```

### 3.3 "Always Treated as Secret" Implementation

**Location**: `/home/user/opal/core/planfmt/plan.go` (lines 32-43)

**Design**: ALL resolved values are treated as secrets, not just sensitive ones:

```go
type Secret struct {
    Key          string  // Variable name
    RuntimeValue string  // The actual value
    DisplayID    string  // Opaque placeholder
}

type Plan struct {
    // ...
    Secrets []Secret  // Contains ALL resolved values
}
```

**Why All Values**:
1. **Conservative** - Can't predict what's sensitive
2. **Defense-in-depth** - No missed secrets
3. **Consistent** - No special cases
4. **Audit trail** - Shows all resolved decorators

### 3.4 Environment in Decorators

**File**: `/home/user/opal/runtime/decorators/env.go` (lines 9-34)

```go
type envDecorator struct{}

// Value decorator handler
func (e *envDecorator) Handle(ctx types.Context, args types.Args) (types.Value, error) {
    if args.Primary == nil {
        return nil, fmt.Errorf("@env requires variable name")
    }
    
    envVar := (*args.Primary).(string)
    
    // Look up in context environment
    value, exists := ctx.Env[envVar]
    if !exists {
        // Check for default
        if args.Params != nil {
            if defaultVal, hasDefault := args.Params["default"]; hasDefault {
                return defaultVal, nil
            }
        }
        return nil, fmt.Errorf("environment variable %q not found", envVar)
    }
    
    return value, nil
}

// Transport scope - @env only works at root level
func (e *envDecorator) TransportScope() types.TransportScope {
    return types.ScopeRootOnly  // Not allowed in @ssh.connect blocks
}
```

---

## 4. INTEGRATION POINTS: PLANNER TO SDK

### 4.1 Conversion Boundary

**File**: `/home/user/opal/core/planfmt/sdk.go` (lines 13-116)

```go
// Boundary between binary format (planfmt) and execution (sdk)
func ToSDKSteps(planSteps []Step) []sdk.Step {
    sdkSteps := make([]sdk.Step, len(planSteps))
    for i, planStep := range planSteps {
        sdkSteps[i] = toSDKStep(planStep)
    }
    return sdkSteps
}

func toSDKStep(planStep Step) sdk.Step {
    return sdk.Step{
        ID:   planStep.ID,
        Tree: toSDKTree(planStep.Tree),  // Recursive conversion
    }
}

// Convert planfmt.CommandNode to sdk.CommandNode
func toSDKTree(node ExecutionNode) sdk.TreeNode {
    switch n := node.(type) {
    case *CommandNode:
        return &sdk.CommandNode{
            Name:  n.Decorator,
            Args:  ToSDKArgs(n.Args),        // Convert args
            Block: ToSDKSteps(n.Block),      // Recursive for blocks
        }
    case *RedirectNode:
        sink := commandNodeToSink(&n.Target)  // Create sink from target
        return &sdk.RedirectNode{
            Source: toSDKTree(n.Source),
            Sink:   sink,
            Mode:   sdk.RedirectMode(n.Mode),
        }
    // ... handle other nodes
    }
}

// Convert argument values
func ToSDKArgs(planArgs []Arg) map[string]interface{} {
    args := make(map[string]interface{})
    for _, arg := range planArgs {
        switch arg.Val.Kind {
        case ValueString:
            args[arg.Key] = arg.Val.Str  // Already resolved if it was decorated
        case ValueInt:
            args[arg.Key] = arg.Val.Int
        case ValueBool:
            args[arg.Key] = arg.Val.Bool
        }
    }
    return args
}
```

**Critical**: By the time SDK receives a Step, all value decorator resolution should be COMPLETE:
- `@var.X` → actual value
- `@env.HOME` → actual path
- String interpolation → final string
- All stored as simple values in `Arg.Val`

### 4.2 Decorator Handler Invocation During Planning

**Missing Implementation** - This is where value decorators need to be resolved.

**Required Architecture**:
```go
// In planner (NOT YET IMPLEMENTED)
type ResolutionContext struct {
    Bindings    map[string]types.Value  // From var declarations
    Environment map[string]string        // Captured at plan time
    Secrets     []planfmt.Secret
    IDFactory   secret.IDFactory
    Cache       map[string]types.Value   // Memoization
}

func (p *planner) resolveDecorator(name string, property string, params map[string]interface{}) (types.Value, error) {
    // Get handler from registry
    handler, exists := types.Global().GetValueHandler(name)
    if !exists {
        return nil, fmt.Errorf("unknown decorator @%s", name)
    }
    
    // Build args
    args := types.Args{
        Primary: &property,
        Params:  params,
    }
    
    // Invoke handler with resolution context
    ctx := types.Context{
        Variables:  p.resolution.Bindings,
        Env:        p.resolution.Environment,
        WorkingDir: ".",  // Current plan directory
    }
    
    value, err := handler(ctx, args)
    if err != nil {
        return nil, err
    }
    
    // Track in secrets
    secretHandle := secret.NewHandleWithFactory(
        value.(string),
        p.resolution.IDFactory,
        secret.IDContext{
            PlanHash:  p.planHash,
            StepPath:  p.currentStepPath,
            Decorator: name,
            KeyName:   property,
            Kind:      "v",  // Value
        },
    )
    
    p.resolution.Secrets = append(p.resolution.Secrets, planfmt.Secret{
        Key:       property,
        RuntimeValue: value.(string),
        DisplayID: secretHandle.ID(),
    })
    
    return value, nil
}
```

### 4.3 String Interpolation During Planning

**Missing Implementation** - Parser creates NodeInterpolatedString, planner needs to process it.

**Required Logic**:
```go
// In planner (NOT YET IMPLEMENTED)
func (p *planner) resolveInterpolatedString(node *parser.NodeInterpolatedString) (string, error) {
    // Get string parts from parser result
    parts := node.Parts  // StringPart with byte offsets
    
    var result strings.Builder
    
    for _, part := range parts {
        if part.IsLiteral {
            // Add literal text as-is
            result.WriteString(part.Content)
        } else {
            // Resolve decorator
            decoratorName := part.DecoratorName  // e.g., "var", "env"
            property := part.Property             // e.g., "HOME"
            params := part.Parameters             // e.g., {"default": "..."}
            
            value, err := p.resolveDecorator(decoratorName, property, params)
            if err != nil {
                return "", err
            }
            
            // Append resolved value
            result.WriteString(value.(string))
        }
    }
    
    return result.String(), nil
}
```

---

## 5. KEY PATTERNS AND EXAMPLES

### 5.1 Pattern: Tainted Handle Lifecycle

**File**: `/home/user/opal/core/sdk/secret/handle.go`

```go
// 1. Create tainted handle
secret := secret.NewHandle("my-password")
assert.True(t, secret.IsTainted())

// 2. Can't print tainted secrets
assert.Panics(t, func() {
    fmt.Printf("%v", secret)  // Panics
})

// 3. Can mask for logging
masked := secret.UnwrapWithMask()  // Safe: "my***ord"
fmt.Printf("Secret: %s\n", masked)  // OK

// 4. Only inside executor (with capability)
secret.SetCapability(&Capability{token: 12345})
{
    value := secret.UnsafeUnwrap()  // "my-password"
    // Pass to subprocess, etc.
}
secret.SetCapability(nil)  // Clear capability
```

### 5.2 Pattern: Environment Isolation for @ssh.connect

**Design** (from `/home/user/opal/core/sdk/executor/transport.go` comments):

```
@ssh.connect decorator pattern:

1. SSH decorator creates SSHTransport
2. SSH decorator wraps ExecutionContext
3. Wrapped context returns SSHTransport from Transport()
4. When @shell executes:
   - Gets environment from context
   - Calls context.Transport().Exec()
   - SSHTransport merges decorator env with REMOTE environment
   - Local os.Environ() NEVER sent to remote

Code Pattern:
type sshExecutionContext struct {
    parent    sdk.ExecutionContext
    transport executor.Transport  // SSHTransport
}

func (s *sshExecutionContext) Transport() interface{} {
    return s.transport  // Return SSH, not local
}

func (s *sshExecutionContext) ExecuteBlock(steps []sdk.Step) (int, error) {
    for _, step := range steps {
        // Decorators call context.Transport()
        // They get SSHTransport, so commands run remote
    }
    return 0, nil
}
```

### 5.3 Pattern: Immutable Context Modifications

**File**: `/home/user/opal/runtime/executor/context.go`

```go
// Original context
ctx := newExecutionContext(args, exec, goCtx)

// Decorator needs to run with different environment
ctx2 := ctx.WithEnviron(map[string]string{
    "AWS_PROFILE": "prod",
    "NODE_ENV":    "production",
})

// Original is unchanged
assert.NotEqual(t, ctx.Environ()["AWS_PROFILE"], 
                ctx2.Environ()["AWS_PROFILE"])

// Both share other fields
assert.Equal(t, ctx.Context(), ctx2.Context())
assert.Equal(t, ctx.Workdir(), ctx2.Workdir())

// For chained decorators
ctx3 := ctx.WithEnviron(env1).WithWorkdir("/app").WithContext(timeout)
// Each creates new context, original unchanged
```

### 5.4 Pattern: Value Decorator Resolution Testing

**File**: `/home/user/opal/runtime/decorators/env.go` + tests

```go
// Test case pattern
func TestEnvDecorator(t *testing.T) {
    // Create context with environment
    ctx := types.Context{
        Env: map[string]string{
            "HOME":     "/home/alice",
            "NODE_ENV": "production",
        },
    }
    
    decorator := &envDecorator{}
    
    // Test 1: Resolve existing variable
    args := types.Args{
        Primary: &[]interface{}{"HOME"}[0].(interface{}),
    }
    value, err := decorator.Handle(ctx, args)
    assert.NoError(t, err)
    assert.Equal(t, "/home/alice", value)
    
    // Test 2: Missing variable without default
    args = types.Args{
        Primary: &[]interface{}{"MISSING"}[0].(interface{}),
    }
    value, err := decorator.Handle(ctx, args)
    assert.Error(t, err)
    
    // Test 3: Missing variable with default
    args = types.Args{
        Primary: &[]interface{}{"MISSING"}[0].(interface{}),
        Params: map[string]types.Value{
            "default": "/tmp",
        },
    }
    value, err := decorator.Handle(ctx, args)
    assert.NoError(t, err)
    assert.Equal(t, "/tmp", value)
}
```

### 5.5 Pattern: Secret Scrubber Integration

**File**: `/home/user/opal/runtime/streamscrub/scrubber.go`

```go
// Create scrubber with output writer
scrubber := streamscrub.New(os.Stdout)

// Register secrets (from Plan.Secrets)
for _, secret := range plan.Secrets {
    scrubber.RegisterSecret(
        []byte(secret.RuntimeValue),
        []byte(secret.DisplayID),
    )
}

// Capture frame of output
scrubber.StartFrame("shell-output")
{
    // ... command writes to scrubber
    // If output contains secret value, it gets replaced
}
secrets := [][]byte{plan.Secrets[0].RuntimeValue}
scrubber.EndFrame(secrets)  // Registers and scrubs

// Output shows DisplayID instead of actual values
```

---

## SUMMARY

### Secrets Infrastructure
- **Two-track identity**: DisplayID (visible) + Fingerprint (internal)
- **BLAKE2s-128 keyed PRF** for deterministic DisplayID generation
- **Context-aware**: Step path, decorator name, key name included in hash
- **Capability gating**: Only executor can access raw values
- **Safe masking**: Multiple methods for logging (Mask, UnwrapWithMask, UnwrapLast4)

### Environment Handling
- **Immutable snapshots**: Captured at context creation, never changes
- **Transport isolation**: Local env never leaks to remote (SSH/Docker)
- **Copy-on-write**: WithEnviron creates new context with modified env
- **Fresh resolution**: Execution-time environment may differ from plan-time
- **All values as secrets**: Even non-sensitive values scrubbed (defense in depth)

### Integration Flow
1. **Parser** creates AST with decorator nodes and string parts
2. **Planner** (NOT YET DONE) should resolve decorators, creating plans with resolved values
3. **Converter** transforms planfmt.Step to sdk.Step with resolved arguments
4. **Executor** receives sdk.Step with resolved values, no further decoration needed
5. **Scrubber** replaces resolved values with DisplayID placeholders in output

### Key Files Summary
- **Secrets**: `/home/user/opal/core/sdk/secret/handle.go`, `idfactory.go`
- **Transport**: `/home/user/opal/core/sdk/executor/transport.go`, `command.go`
- **Context**: `/home/user/opal/runtime/executor/context.go`
- **Decorators**: `/home/user/opal/runtime/decorators/env.go`, `var.go`
- **Scrubbing**: `/home/user/opal/runtime/streamscrub/scrubber.go`
