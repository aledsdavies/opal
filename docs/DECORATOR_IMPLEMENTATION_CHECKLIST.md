# Value Decorator Implementation Checklist

## Current Status: PARSING COMPLETE, RESOLUTION INCOMPLETE

### Parsing Pipeline ✅ 100% Complete

- [x] **Lexer** - Tokenizes `@`, identifiers, dots, parentheses, parameters
  - Files: `runtime/lexer/` (all files)
  - Tests: `runtime/lexer/decorators_test.go` ✅
  - Examples: `@var`, `@env.HOME(default="...")`, `@aws.secret.api_token`

- [x] **String Tokenization** - Zero-allocation string splitting  
  - File: `runtime/parser/string_tokenizer.go`
  - Tests: `runtime/parser/string_tokenizer_test.go` ✅ (13 tests)
  - Returns: `StringPart` structs with byte offsets
  - Smart: Only processes registered value decorators

- [x] **Parser** - Builds AST with decorator nodes
  - File: `runtime/parser/parser.go`
  - Nodes: `NodeDecorator`, `NodeInterpolatedString`, `NodeStringPart`
  - Tests: `runtime/parser/decorator_test.go` ✅ (40+ tests)
  - Type validation: Works for decorator parameters

- [x] **Decorator Registry** - Schema definition and registration
  - Files: `runtime/decorators/env.go`, `runtime/decorators/var.go`
  - Types: `core/types/registry.go`, `core/types/schema.go`
  - Handlers: `ValueHandler` type defined
  - Tests: `runtime/decorators/env_scope_test.go`

---

### Planning Pipeline ❌ 0% Complete

- [ ] **Scope/Binding Access**
  - Need: Thread variable bindings through planner
  - File: `runtime/planner/planner.go`
  - Missing: `planner.Config` needs `Bindings` field
  - Blocker: How do variable assignments flow to planner?

- [ ] **Decorator Resolution**
  - Need: Call decorator handlers during planning
  - File: `runtime/planner/planner.go`
  - Missing: Handler invocation infrastructure
  - Challenge: Handlers need execution context with environment

- [ ] **String Interpolation Processing**
  - Need: Detect `NodeInterpolatedString` in planner
  - File: `runtime/planner/planner.go`
  - Missing: Logic to walk decorator parts
  - Challenge: Reconstruct string from resolved parts

- [ ] **Resolved Value Tracking**
  - Need: Store resolved values in plan
  - File: `core/planfmt/plan.go`
  - Missing: `ResolvedValues` or similar tracking
  - Challenge: Secret scrubbing integration

---

### Plan Representation ⚠️ 50% Complete

- [x] **Operator Precedence Tree**
  - File: `core/planfmt/execution_tree.go`
  - Structures: `CommandNode`, `PipelineNode`, `AndNode`, `OrNode`, etc.
  - Status: Complete and used

- [x] **Command Arguments**
  - File: `core/planfmt/plan.go`
  - Structure: `Arg` with `Key` and `Value`
  - Values: `ValueString`, `ValueInt`, `ValueBool`, `ValuePlaceholder`
  - Status: Works for commands

- [ ] **Value Decorator Markers**
  - Need: Track which args contain resolved decorators
  - Field: `Arg` needs metadata about decorator source
  - Missing: No way to distinguish `@var.X` from literal "X"

- [ ] **Secret Tracking**
  - File: `core/planfmt/plan.go`
  - Field: `Plan.Secrets` exists but never populated
  - Missing: Logic to add secrets during resolution

- [ ] **Placeholder Format**
  - Need: `opal:s:ID` placeholders for display
  - Missing: Integration with secret scrubbing
  - Spec: SPECIFICATION.md section "Plans: Three Execution Modes"

---

### Execution Layer ❌ 0% Complete

- [ ] **Value Decorator Executor**
  - Need: New phase before `@shell` execution
  - File: `core/sdk/execution.go` (doesn't exist yet)
  - Missing: Entire execution engine
  - Challenge: Resolve decorators with environment access

- [ ] **Secret Scrubbing**
  - Need: Replace resolved values with placeholders
  - Files: `core/sdk/secret/`, `core/planfmt/formatter/`
  - Missing: Integration with output streams
  - Challenge: Scrub before any output display

- [ ] **Executor Context**
  - Need: Pass scope bindings to decorators
  - File: `core/sdk/execution.go`
  - Missing: Full execution context type
  - Challenge: Thread context through nested decorators

---

### Testing ⚠️ 30% Complete

- [x] **String Tokenization Tests**
  - File: `runtime/parser/string_tokenizer_test.go`
  - Count: 13 tests
  - Coverage: Comprehensive

- [x] **Decorator Parsing Tests**
  - File: `runtime/parser/decorator_test.go`
  - Count: 40+ tests
  - Coverage: Detection, parameters, type validation

- [ ] **Planner Tests for Resolution**
  - File: `runtime/planner/planner_test.go`
  - Status: Tests basic shell commands only
  - Missing: Tests for `@var`, `@env` resolution
  - Missing: String interpolation tests

- [ ] **Integration Tests**
  - Missing: End-to-end value decorator resolution
  - Missing: String interpolation in real scripts
  - Missing: Secret scrubbing verification

- [ ] **Plan Verification Tests**
  - Missing: Contract verification with resolved values
  - Missing: Hash generation with resolved decorators
  - Missing: Plan roundtrip tests

---

## Concrete Implementation Steps (Priority Order)

### IMMEDIATE (Blocking Resolution)

**1. Extend Planner Config** (1-2 hours)
```go
type Config struct {
    Target         string
    Bindings       map[string]types.Value  // Variable bindings
    Environment    map[string]string        // For @env
    IDFactory      secret.IDFactory
    Telemetry      TelemetryLevel
    Debug          DebugLevel
}
```

**2. Create Resolution Context** (2-4 hours)
```go
type ResolutionContext struct {
    Bindings   map[string]types.Value
    Environment map[string]string
    Secrets    []Secret
    IDFactory  secret.IDFactory
    Cache      map[string]types.Value  // Memoization
}
```

**3. Implement Decorator Invocation** (4-6 hours)
```go
func (p *planner) resolveDecorator(name string, args ...) (types.Value, error) {
    // Call registered handler
    // Track in secrets
    // Return resolved value
}
```

### SHORT-TERM (Full Resolution)

**4. String Interpolation in Planner** (4-6 hours)
- Detect `NodeInterpolatedString`
- Extract parts using `StringPart` offsets
- Resolve each decorator part
- Reconstruct string

**5. Variable Binding Integration** (2-4 hours)
- How to get bindings from `@var`?
- Need loop/conditional scope handling
- Need function parameter bindings

**6. Secret Tracking in Plans** (2-3 hours)
- Add resolved values to `Plan.Secrets`
- Generate `DisplayID` placeholders
- Update `Arg` struct if needed

### MEDIUM-TERM (Execution)

**7. Executor Decorator Resolution** (6-8 hours)
- Fresh resolution at execution time
- Contract verification
- Deterministic vs dynamic handling

**8. Secret Scrubbing** (4-6 hours)
- Implement output filtering
- Replace values with placeholders
- Maintain audit trail

---

## Files That Need Changes

| File | Change | Impact | Priority |
|------|--------|--------|----------|
| `runtime/planner/planner.go` | Add scope/env handling, resolve decorators | Major | CRITICAL |
| `runtime/planner/planner.go` | Detect and process `NodeInterpolatedString` | Major | CRITICAL |
| `runtime/planner/planner.go` | Track resolved values | Medium | HIGH |
| `core/planfmt/plan.go` | Populate `Plan.Secrets` | Medium | HIGH |
| `core/planfmt/plan.go` | Track resolved values in `Arg` | Medium | HIGH |
| `core/types/registry.go` | No changes needed | - | - |
| `runtime/decorators/*.go` | Handlers already exist | - | - |
| `runtime/parser/*.go` | No changes needed | - | - |

---

## What CAN Work Today (with Workarounds)

1. **Parse `@var`, `@env` in decorators** - Yes
2. **Type validation of parameters** - Yes
3. **Inline command expansion** - Maybe (via string concat)
4. **Simple echo statements** - If hand-expanded

## What CANNOT Work Today

1. **`@var.X` resolution in commands** - No
2. **`@env.X` resolution in commands** - No
3. **String interpolation** - No
4. **Contract verification** - No
5. **Secret scrubbing** - No

---

## Estimated Implementation Effort

| Phase | Estimate | Complexity |
|-------|----------|-----------|
| Basic resolution (items 1-3) | 8-12 hours | Medium |
| Full planner integration (items 4-6) | 8-12 hours | Medium-High |
| Executor and scrubbing (items 7-8) | 10-14 hours | High |
| Testing and validation | 15-20 hours | High |
| **Total** | **41-58 hours** | **Medium-High** |

---

## Risk Areas

1. **Scope Management** - How to thread bindings through nested scopes?
2. **Circular Dependencies** - What if `@var.X` depends on another `@var.Y`?
3. **Contract Determinism** - Values must be deterministic for verification
4. **Error Handling** - What if a decorator fails during planning?
5. **Performance** - Resolving many decorators could slow planning

