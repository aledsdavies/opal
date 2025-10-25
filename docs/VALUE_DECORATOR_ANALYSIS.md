# OPAL Value Decorators and String Interpolation - Current State Analysis

## Executive Summary

OPAL has a **complete parsing foundation** for value decorators and string interpolation but **lacks runtime resolution implementation**. The infrastructure is in place to detect and parse decorators in strings, but the planner and executor don't yet resolve them to actual values.

---

## 1. SPECIFICATION COMPLIANCE

### What the Spec Says (SPECIFICATION.md, lines 333-373)

#### Value Decorators
- **Dot syntax for simple access**: `@var.REPLICAS`, `@env.HOME`, `@aws.secret.api_token`
- **Optional parameters when needed**: `@env.PORT(default=3000)`, `@aws.secret.api_key(region="us-west-2")`
- **Resolve at plan-time**: Value decorators are evaluated during plan generation to create concrete values
- **Used in arguments**: `kubectl scale --replicas=@var.REPLICAS deployment/app`
- **Used in strings**: `echo "Environment: @env.NODE_ENV"`

#### Execution Decorators
- **Always function syntax with blocks**: `@retry(attempts=3) { ... }`, `@timeout(30s) { ... }`
- **Clear distinction**: Value decorators use dot syntax, execution decorators use function syntax

#### Key Design Principle (lines 948-1052)
Value decorators like `@env.HOME` resolve **at plan-time** using the **local** environment. They are NOT transport-aware. This enables deterministic planning and contract verification.

### What the Spec Says (DECORATOR_GUIDE.md)

#### Value Decorators (lines 113-126)
- **Return data with no side effects**
- **Can be used in expressions and string interpolation**
- **Pure functions**: Same inputs → same outputs
- **Registered with**: `RegisterValue(path)`

#### String Interpolation Examples (lines 124-125)
```opal
echo "Home directory: @env.HOME"
echo "Config: @file.read('settings.json')"
```

---

## 2. CURRENT IMPLEMENTATION STATUS

### ✅ IMPLEMENTED: Lexer Support

**File**: `/home/user/opal/runtime/lexer/decorators_test.go`

The lexer fully tokenizes decorators:
```
@var.name → AT, IDENTIFIER("var"), DOT, IDENTIFIER("name")
@env.HOME(default="/home") → AT, IDENTIFIER("env"), DOT, IDENTIFIER("HOME"), LPAREN, ...
```

### ✅ IMPLEMENTED: String Tokenization

**File**: `/home/user/opal/runtime/parser/string_tokenizer.go`

The `TokenizeString()` function (zero-allocation) splits strings into:
```go
type StringPart struct {
    Start         int  // Byte offset in content
    End           int  // Byte offset in content
    IsLiteral     bool // true = literal text, false = decorator
    PropertyStart int  // For @var.name, byte offset of "name"
    PropertyEnd   int  // End of property name
}
```

**Algorithm**:
1. Fast path: No `@` symbols → return single literal part
2. First pass: Count parts to pre-allocate
3. Second pass: Fill parts with byte offsets
4. Registry lookup: `types.Global().IsValueDecorator(decoratorName)`

**Test Coverage**: 13 test cases in `string_tokenizer_test.go`
- Single quotes (no interpolation)
- Double quotes with @var and @env
- Multiple decorators in single string
- Decorators without properties
- Unregistered decorators (stay literal)
- Execution decorators (stay literal)
- Email addresses with @ (stay literal)

### ✅ IMPLEMENTED: Parser Decorator Recognition

**File**: `/home/user/opal/runtime/parser/parser.go` (lines 2009-2116)

Functions:
- `stringNeedsInterpolation()` - Checks if STRING token requires interpolation
- `stringLiteral()` - Parses string literals with optional interpolation

**AST Nodes Created**:
- `NodeInterpolatedString` - String with decorators
- `NodeStringPart` - Each literal or decorator part
- `NodeDecorator` - Decorator reference (e.g., `@var.name`)

**Example Parse Output**:
```
Input: echo "Hello @var.name"
Parse Tree:
  NodeShellCommand
    NodeLiteral("echo")
    NodeInterpolatedString
      NodeStringPart (literal: "Hello ")
      NodeStringPart (decorator: @var.name)
```

### ✅ IMPLEMENTED: Decorator Schemas and Registration

**Files**:
- `/home/user/opal/runtime/decorators/env.go` - @env decorator
- `/home/user/opal/runtime/decorators/var.go` - @var decorator

**Example** (@env):
```go
schema := types.NewSchema("env", types.KindValue).
    Description("Access environment variables").
    PrimaryParam("property", types.TypeString, "Environment variable name").
    Param("default", types.TypeString).Optional().
    Returns(types.TypeString, "Value of the environment variable").
    Build()

types.Global().RegisterValueDecoratorWithSchema(schema, decorator, decorator.Handle)
```

**Handler Signature**:
```go
func (e *envDecorator) Handle(ctx types.Context, args types.Args) (types.Value, error)
```

### ❌ NOT IMPLEMENTED: Decorator Resolution in Planner

**File**: `/home/user/opal/runtime/planner/planner.go` (lines 490-695)

**Current Behavior**:
The planner's `planCommand()` function:
1. Collects all tokens in shell command
2. **Concatenates them into a raw string** (lines 523-535)
3. Creates `@shell` decorator with raw command as argument
4. **No decorator resolution occurs**

```go
cmd := Command{
    Decorator: "@shell",
    Args: []planfmt.Arg{
        {
            Key: "command",
            Val: planfmt.Value{
                Kind: planfmt.ValueString,
                Str:  command,  // ← RAW STRING INCLUDING @var, @env, etc.
            },
        },
    },
}
```

**Redirect Handling** (lines 593-620):
When processing redirect targets, the planner:
1. Collects tokens for redirect target
2. Builds target command string (raw)
3. **Creates `@shell("@var.OUTPUT_FILE")` literally** - does NOT resolve @var

### ❌ NOT IMPLEMENTED: String Interpolation Resolution

**Missing**: Logic to:
1. Detect `NodeInterpolatedString` in planner
2. Extract decorator parts from string
3. Resolve each decorator part
4. Build final interpolated string value

**Current Gap**:
```
Input: echo "Hello @var.name"
Expected: echo "Hello alice"
Actual: echo "Hello @var.name"  ← RAW STRING
```

### ❌ NOT IMPLEMENTED: Plan Representation of Resolved Decorators

**File**: `/home/user/opal/core/planfmt/plan.go`

**Current Structure**:
```go
type CommandNode struct {
    Decorator string  // "@shell", "@retry", etc.
    Args      []Arg   // Decorator arguments
    Block     []Step  // Nested steps
}

type Arg struct {
    Key string
    Val Value  // Union of Str, Int, Bool, Ref (placeholder index)
}
```

**Issue**: No explicit representation for:
- Which arguments contain resolved value decorators
- Secret scrubbing information
- Placeholder tracking for resolved values

### ❌ NOT IMPLEMENTED: Execution Context for Resolution

The `ExecutionContext` in `/home/user/opal/core/sdk/execution.go` doesn't include mechanisms to:
1. Register resolved value decorator results
2. Mark values as secrets for scrubbing
3. Track resolution timing (plan-time vs execution-time)

---

## 3. PLAN REPRESENTATION

### Current State
Plans use a simple `CommandNode` with raw string arguments:

```
Step {
  ID: 1,
  Tree: CommandNode {
    Decorator: "@shell",
    Args: [{Key: "command", Val: Value{Str: "echo \"Hello @var.name\""}}],
  }
}
```

### What's Missing
1. **Value decorator tracking**: Plan should record which values were resolved
2. **Secret scrubbing markers**: Where to replace values with `opal:s:ID` placeholders
3. **Resolution DAG**: Order of decorator resolution dependencies
4. **Plan seed envelopes** (for `@random.password()`, `@crypto.generate_key()`)

---

## 4. INTERPOLATION PATTERNS ANALYSIS

### Supported by Parser ✅
- String literals: `"Hello @var.name"` → parsed correctly
- Multiple decorators: `"@var.first and @var.second"` → both recognized
- Properties: `@var.CONFIG.database.url` → property offsets captured
- Termination: `@var.service()_backup` → can use `()` to terminate

### NOT RESOLVED by Planner ❌
- Shell command arguments: `echo "Hello @var.name"` → stays as-is in plan
- Shell interpolation: `echo "Token: @aws.secret.api_token"` → no resolution
- Decorator calls in strings: `"Token: @aws.secret.api_token"` → not resolved
- Field access: `@var.CONFIG.database.url` → not traversed

### Example: Missing Resolution

**Input**:
```opal
var SERVICE = "api"
var REPLICAS = 3

kubectl scale --replicas=@var.REPLICAS deployment/@var.SERVICE
```

**Current Plan** (WRONG):
```
@shell("kubectl scale --replicas=@var.REPLICAS deployment/@var.SERVICE")
```

**Expected Plan** (after resolution):
```
@shell("kubectl scale --replicas=3 deployment/api")
```

---

## 5. KEY FILES AND THEIR STATUS

### Parsing Layer
| File | Status | Notes |
|------|--------|-------|
| `runtime/lexer/` | ✅ Complete | Tokenizes decorators fully |
| `runtime/parser/string_tokenizer.go` | ✅ Complete | Zero-allocation string parsing |
| `runtime/parser/parser.go` | ✅ Complete | Builds AST with decorator nodes |
| `runtime/parser/tree.go` | ✅ Complete | Defines AST node types |
| `runtime/decorators/env.go` | ✅ Complete | @env decorator with schema |
| `runtime/decorators/var.go` | ✅ Complete | @var decorator with schema |

### Planning Layer
| File | Status | Notes |
|------|--------|-------|
| `runtime/planner/planner.go` | ⚠️ Partial | Builds commands but no resolution |
| `core/types/registry.go` | ✅ Complete | Decorator registration infrastructure |
| `core/types/schema.go` | ✅ Complete | Schema builder API |

### Plan Representation
| File | Status | Notes |
|------|--------|-------|
| `core/planfmt/plan.go` | ⚠️ Partial | No explicit value decorator tracking |
| `core/planfmt/execution_tree.go` | ✅ Complete | Operator precedence tree structure |

### Execution Layer
| File | Status | Notes |
|------|--------|-------|
| `core/sdk/execution.go` | ❌ Missing | No executor for value decorators |
| `core/sdk/secret/` | ⚠️ Partial | Secret handles exist but no resolution context |

### Testing
| File | Coverage | Notes |
|------|----------|-------|
| `runtime/parser/string_tokenizer_test.go` | ✅ 13 tests | Comprehensive string parsing tests |
| `runtime/parser/decorator_test.go` | ✅ Complete | Parser decorator recognition tests |
| `runtime/planner/planner_test.go` | ⚠️ Partial | `"redirect with variable"` test shows @var but doesn't verify resolution |

---

## 6. CRITICAL GAPS AND BLOCKERS

### 1. **No Decorator Resolution in Planner**
The planner doesn't:
- Resolve `@var.X` to actual variable values
- Resolve `@env.X` to environment variables
- Call decorator handlers to evaluate values
- Store resolved values in plan

**Blocker**: The planner needs access to:
- Variable bindings from scope
- Environment for `@env` resolution
- Registered decorator handlers
- Execution context for calling handlers

### 2. **String Interpolation Not Implemented**
Parser creates `NodeInterpolatedString` but planner doesn't:
- Walk the `NodeInterpolatedString` tree
- Extract decorator parts using `StringPart` offsets
- Resolve each decorator
- Reconstruct interpolated string

### 3. **No Secret Handling in Resolved Values**
When values are resolved, they need to:
- Be marked as secrets (for scrubbing)
- Be converted to `opal:s:ID` placeholders in display output
- Be tracked in `Plan.Secrets` slice for audit trail

### 4. **No Execution Context Access During Planning**
Planner needs to:
- Access variable bindings (scopes)
- Access environment variables
- Call decorator handlers with proper context
- Track which decorators were resolved

### 5. **Validation Missing**
Parser needs to validate:
- Required decorator parameters are provided
- Parameter types match schema
- Decorator paths are registered
- Default values are provided for optional parameters

Parser PARTIALLY does this (see `decorator_test.go` type validation tests) but isn't comprehensive.

---

## 7. EXAMPLES FROM SPECIFICATION

### Example 1: Basic Interpolation (SPECIFICATION.md line 338-345)

**Spec Shows**:
```opal
kubectl scale --replicas=@var.REPLICAS deployment/app
psql @var.CONFIG.database.url
docker run -p @var.SERVICES[0]:3000 app
```

**Current Implementation**:
- ✅ Parser recognizes `@var.REPLICAS` in arguments
- ❌ Planner doesn't resolve it
- ❌ Plan contains raw `@var.REPLICAS` string

### Example 2: String Interpolation (SPECIFICATION.md line 389)

**Spec Shows**:
```opal
echo "Deploying @var.service with @var.replicas replicas"
```

**Current Implementation**:
- ✅ Parser tokenizes string into parts
- ❌ Planner doesn't process `NodeInterpolatedString`
- ❌ Plan contains raw `"Deploying @var.service with @var.replicas replicas"`

### Example 3: Complex Properties (SPECIFICATION.md line 339)

**Spec Shows**:
```opal
psql @var.CONFIG.database.url
```

**Current Implementation**:
- ✅ Parser captures property path in `StringPart.PropertyStart/End`
- ❌ Planner doesn't traverse nested properties
- ❌ No field access evaluation

### Example 4: Parameters with Defaults (SPECIFICATION.md line 351-354)

**Spec Shows**:
```opal
kubectl apply -f @env.MANIFEST_PATH(default="k8s/")
curl @aws.secret.api_key(region="us-west-2")
```

**Current Implementation**:
- ✅ Parser recognizes parameters
- ❌ Planner doesn't evaluate parameters
- ❌ Default values aren't applied

---

## 8. IMPLEMENTATION ROADMAP (What Would Be Needed)

### Phase 1: Planner Enhancement
1. **Scope Resolution**
   - Thread variable bindings through planner
   - Track environment for `@env` resolution
   - Support nested scopes (loops, conditionals)

2. **Decorator Invocation**
   - Extend planner to call decorator handlers
   - Create execution context for handlers
   - Track resolved values with secret handles

3. **String Interpolation**
   - Detect `NodeInterpolatedString` in planner
   - Walk decorator parts from `StringPart` offsets
   - Build interpolated strings with resolved values

### Phase 2: Plan Representation
1. **Resolved Value Tracking**
   - Add `ResolvedValues` to `Plan` struct
   - Track which args contain resolved values
   - Store placeholders instead of raw values

2. **Secret Registry**
   - Populate `Plan.Secrets` with all resolved values
   - Map `DisplayID` → scrubbing information
   - Support audit trail of what was resolved

### Phase 3: Validation and Testing
1. **Parse-time Validation**
   - Type checking for decorator parameters
   - Required parameter verification
   - Schema compliance checking

2. **Plan-time Validation**
   - Value decorator cycles detection
   - Scope validation (no undefined variables)
   - Transport scope rules (no `@env` inside `@ssh.connect`)

---

## 9. CURRENT TEST COVERAGE

### Parser Tests ✅
- `string_tokenizer_test.go`: 13 comprehensive tests
- `decorator_test.go`: 
  - Decorator detection (6 tests)
  - Parameters and types (40+ tests)
  - Type validation (10+ tests)

### Planner Tests ⚠️
- Basic shell command conversion
- Redirect handling (including `@var.OUTPUT_FILE` but not validated for resolution)
- Operator precedence
- **Missing**: Tests for decorator resolution, string interpolation, variable binding

### Integration Tests ❌
- No end-to-end tests of value decorator resolution
- No string interpolation resolution tests
- No secret scrubbing tests

---

## 10. CONCLUSION

OPAL has **excellent parsing infrastructure** but **incomplete resolution implementation**:

### What Works
✅ Decorators are tokenized and parsed  
✅ Strings with decorators are recognized  
✅ Decorator schemas are defined and registered  
✅ AST nodes for decorators exist  
✅ Parser validates decorator parameter types  

### What Doesn't Work
❌ Decorators are not resolved to values in planner  
❌ String interpolation is not performed  
❌ Resolved values are not tracked in plans  
❌ Secrets are not marked for scrubbing  
❌ Value decorator handlers are not invoked  

### Next Steps to Complete Implementation
1. **Extend planner** to receive scope bindings and environment
2. **Implement decorator resolution** by calling registered handlers
3. **Add string interpolation** in planner
4. **Track resolved values** in plan structure
5. **Implement secret scrubbing** in formatters and executors
6. **Add comprehensive tests** for full resolution pipeline

---

## Appendix: Code References

### StringPart Structure
```go
type StringPart struct {
    Start         int  // Start byte offset in content
    End           int  // End byte offset in content
    IsLiteral     bool // true if literal text, false if decorator
    PropertyStart int  // Start of property name (or -1)
    PropertyEnd   int  // End of property name (or -1)
}
```

### AST Node Types
```
NodeDecorator          // @var.name, @env.HOME(...)
NodeInterpolatedString // "Hello @var.name"
NodeStringPart         // Part of interpolated string
```

### Decorator Handler Types
```go
type ValueHandler func(ctx Context, args Args) (Value, error)
type ExecutionHandler func(ctx Context, args Args) error
```

### Plan Structure
```go
type Plan struct {
    Header PlanHeader
    Target string
    Steps  []Step
    Secrets []Secret  // Values to scrub
}

type Step struct {
    ID   uint64
    Tree ExecutionNode  // Operator precedence tree
}

type CommandNode struct {
    Decorator string
    Args      []Arg
    Block     []Step
}
```
