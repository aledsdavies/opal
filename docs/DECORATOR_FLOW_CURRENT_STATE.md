# OPAL Decorator Flow: Current State Analysis

This document describes the **current, working state** of how decorators flow through OPAL's architecture. It focuses on what EXISTS NOW, with real code examples and test data.

## Overview: The Decorator Journey

Decorators in OPAL follow a 5-layer transformation pipeline:

```
Source Code
    ↓
[Layer 1: Lexer] → Token Stream
    ↓
[Layer 2: Parser] → AST Events
    ↓
[Layer 3: Planner] → Execution Plan
    ↓
[Layer 4: Executor] → Command Execution
    ↓
[Layer 5: Registry] → Handler Resolution
```

### What Currently Works

✅ **Decorator Registration**: Built-in decorators (@var, @env, @shell, @retry, @timeout, @parallel) register themselves in `init()` blocks

✅ **Lexer Tokenization**: Tokenizes `@` symbol and decorator names into separate tokens

✅ **Parser Recognition**: Identifies registered decorators and builds AST nodes, validates parameters

✅ **String Interpolation**: Detects `@var.name` and `@env.VARIABLE` patterns in double-quoted strings and marks them as decorator references

✅ **Plan Generation**: Converts AST to executable plan with decorator commands

✅ **Value Decorators in Strings**: @var and @env can be interpolated in shell command strings at plan time

❌ **Execution Decorator Handlers**: Currently registered but no execution framework yet

❌ **Decorator Resolution**: Value decorators are NOT resolved during execution planning (only detected)

---

## Layer 1: Source Code → Lexer

**File**: `/home/user/opal/runtime/lexer/lexer.go` and related files

### The Process

The lexer scans source code character-by-character and produces tokens.

#### Input: Raw OPAL Source

```opal
var REPLICAS = 3
deploy: kubectl scale --replicas=@var.REPLICAS deployment/app
```

#### Tokenization

The lexer treats `@` as a single-character token (type `AT`), followed by decorator names as `IDENTIFIER` tokens:

```
Tokens produced:

Token 0: VAR (type=50)         Text="var"       Line=1 Col=1
Token 1: IDENTIFIER            Text="REPLICAS" Line=1 Col=5
Token 2: EQUALS               Text="="         Line=1 Col=14
Token 3: INTEGER              Text="3"         Line=1 Col=16
Token 4: NEWLINE              Text="\n"        Line=1 Col=17

Token 5: AT (type=28)          Text=""         Line=2 Col=26    ← Decorator start
Token 6: IDENTIFIER            Text="var"      Line=2 Col=27    ← Decorator name
Token 7: DOT                   Text="."        Line=2 Col=30    ← Property accessor
Token 8: IDENTIFIER            Text="REPLICAS" Line=2 Col=31    ← Property name
Token 9: ...
```

**Key Facts**:
- `AT` is defined in `/home/user/opal/runtime/lexer/tokens.go` line 28
- `DOT` is line 29
- Decorator names are lexed as `IDENTIFIER` tokens, not special keywords
- Only `@var` (the keyword) gets special token type VAR; others are IDENTIFIER

#### Example Test Output

From `/home/user/opal/runtime/lexer/decorators_test.go` lines 22-27:

```go
{
    name:  "var decorator",
    input: "@var",
    expected: []tokenExpectation{
        {AT, "", 1, 1},
        {VAR, "var", 1, 2},      // "var" is a keyword, gets VAR token type
        {EOF, "", 1, 5},
    },
},
```

But for `@env`:

```go
{
    name:  "env decorator",
    input: "@env",
    expected: []tokenExpectation{
        {AT, "", 1, 1},
        {IDENTIFIER, "env", 1, 2},  // "env" is not a keyword, IDENTIFIER
        {EOF, "", 1, 5},
    },
},
```

### Lexer Data Structures

**Token Type** (`Token` struct, `/home/user/opal/runtime/lexer/tokens.go` line 95):

```go
type Token struct {
    Type     TokenType      // AT=28, IDENTIFIER=82, DOT=29, etc.
    Text     []byte         // "var", "REPLICAS", "deploy", etc.
    Position Position       // Line, column, byte offset
    HasSpaceBefore bool    // Parsing hint (e.g., "--release" vs "-- release")
}
```

**Position tracking** (line 113):

```go
type Position struct {
    Line   int // 1-based
    Column int // 1-based
    Offset int // 0-based byte offset
}
```

### What Lexer DOES NOT Do

- Does NOT check if decorator names are registered
- Does NOT resolve decorator values
- Does NOT validate decorator syntax (parentheses, parameters)
- Does NOT interpolate strings (string content is opaque at lexer level)

---

## Layer 2: Lexer → Parser

**Files**: 
- `/home/user/opal/runtime/parser/parser.go`
- `/home/user/opal/runtime/parser/tree.go`
- `/home/user/opal/runtime/parser/string_tokenizer.go`

### The Process

The parser consumes tokens and builds an Abstract Syntax Tree (AST) represented as a stream of events.

### Input: Token Stream

From the previous example:
```
Token 5: AT
Token 6: IDENTIFIER "var"
Token 7: DOT
Token 8: IDENTIFIER "REPLICAS"
```

### Decorator Detection & Parsing

The parser has a dedicated `decorator()` function (`/home/user/opal/runtime/parser/parser.go`):

```go
func (p *parser) decorator() {
    // Look ahead to check if this is a registered decorator
    atPos := p.pos
    p.advance() // Move past @
    
    // Check if next token is an identifier or VAR keyword
    if !p.at(lexer.IDENTIFIER) && !p.at(lexer.VAR) {
        return
    }
    
    // Build decorator path by trying progressively longer dot-separated names
    // Try "var", then "var.name", etc. until finding registered decorator
    decoratorName := string(p.current().Text)
    // ... look up in registry ...
    
    if types.Global().IsRegistered(decoratorName) {
        // It's a registered decorator - parse it fully
        p.parseDecorator()
    }
}
```

**Registry Lookup** (`/home/user/opal/runtime/parser/decorator_test.go` line 11):

```go
// Decorator detection uses global registry
func TestDecoratorDetection(t *testing.T) {
    tests := []struct {
        input       string
        isDecorator bool
    }{
        {
            input:       "@var.name",
            isDecorator: true,  // "var" is registered
        },
        {
            input:       "@env.HOME",
            isDecorator: true,  // "env" is registered
        },
        {
            input:       "@unknown.field",
            isDecorator: false, // "unknown" is NOT registered
        },
    }
}
```

### Output: AST Events

The parser produces an event stream. For `@var.REPLICAS`:

```
Event sequence (from decorator_test.go lines 137-149):

EventOpen(NodeDecorator)
EventToken(0)    // @ symbol
EventToken(1)    // "var"
EventToken(2)    // "." symbol
EventToken(3)    // "REPLICAS" property name
EventClose(NodeDecorator)
```

### Node Types for Decorators

From `/home/user/opal/runtime/parser/tree.go` lines 42-101:

```go
// Key node types for decorators:

NodeDecorator               // @var.name, @env.HOME (line 67)
NodeParamList              // Parameter list with () (line 45)
NodeParam                  // Single parameter (line 47)
NodeInterpolatedString     // String with decorators: "Hello @var.name" (line 63)
NodeStringPart             // Part of interpolated string (line 64)
```

### String Interpolation Detection

**Separate Path for String Content**:

The parser uses `TokenizeString()` from `/home/user/opal/runtime/parser/string_tokenizer.go` to parse string contents:

```go
// TokenizeString splits string content into literal and decorator parts
func TokenizeString(content []byte, quoteType byte) []StringPart

// Example: "Hello @var.name"
// Returns:
// StringPart{Start: 0, End: 6, IsLiteral: true}           // "Hello "
// StringPart{Start: 7, End: 10, IsLiteral: false}         // @var.name
//     PropertyStart: 11, PropertyEnd: 15
```

**Test Example** (`/home/user/opal/runtime/parser/string_tokenizer_test.go` lines 33-40):

```go
{
    name:      "double quote with var interpolation",
    content:   "Hello @var.name",
    quoteType: '"',
    expected: []StringPart{
        {Start: 0, End: 6, IsLiteral: true, PropertyStart: -1, PropertyEnd: -1},
        {Start: 7, End: 10, IsLiteral: false, PropertyStart: 11, PropertyEnd: 15},
    },
},
```

### Parameter Validation

The parser validates decorator parameters at parse time:

From `/home/user/opal/runtime/parser/decorator_test.go` lines 243-306:

```go
func TestDecoratorParameterTypeValidation(t *testing.T) {
    tests := []struct {
        input     string
        wantError bool
    }{
        {
            input:     `@env.HOME(default="")`,  // String param with string ✅
            wantError: false,
        },
        {
            input:     `@env.HOME(default=42)`,  // String param with int ❌
            wantError: true,
            wantMessage: "parameter 'default' expects string, got integer",
        },
    }
}
```

### Parser Data Structures

**Event Types** (`/home/user/opal/runtime/parser/tree.go` lines 25-33):

```go
type Event struct {
    Kind EventKind
    Data uint32
}

type EventKind uint8
const (
    EventOpen      EventKind = iota  // Open syntax node
    EventClose                       // Close syntax node
    EventToken                       // Consume token
    EventStepEnter                   // Enter a step
    EventStepExit                    // Exit a step
)
```

**ParseTree Output** (`/home/user/opal/runtime/parser/tree.go` lines 9-16):

```go
type ParseTree struct {
    Source      []byte          // Original source
    Tokens      []lexer.Token   // Tokens from lexer
    Events      []Event         // Parse events ← THIS IS THE OUTPUT
    Errors      []ParseError    // Parse errors
    Telemetry   *ParseTelemetry // Performance metrics
    DebugEvents []DebugEvent    // Debug events
}
```

### What Parser DOES

✅ Recognizes registered decorators by checking registry
✅ Parses decorator syntax (@name.property, parameters with parentheses)
✅ Validates parameter types against schema
✅ Detects string interpolation with @decorator syntax
✅ Reports parse errors with suggestions
✅ Produces typed AST events

### What Parser DOES NOT Do

❌ Resolve decorator values (that's planner/executor's job)
❌ Execute decorators
❌ Validate decorator implementations

---

## Layer 3: Parser → Planner

**Files**: 
- `/home/user/opal/runtime/planner/planner.go`
- `/home/user/opal/core/planfmt/plan.go`
- `/home/user/opal/core/planfmt/execution_tree.go`

### The Process

The planner consumes parser events and builds an execution plan. For decorators, this means:

1. Converting decorator events to `Command` structures
2. Building an operator precedence tree
3. Generating serializable plan steps

### Input: Parser Events

```
EventStepEnter
EventOpen(NodeShellCommand)
EventToken(...)  // "echo", '"Hello', etc.
EventClose(NodeShellCommand)
EventStepExit
```

### Command Representation (Internal)

From `/home/user/opal/runtime/planner/planner.go` lines 37-47:

```go
type Command struct {
    Decorator      string         // "@shell", "@retry", "@parallel"
    Args           []planfmt.Arg  // Decorator arguments
    Block          []planfmt.Step // Nested steps (for decorators with blocks)
    Operator       string         // "&&", "||", "|", ";" (chain operator)
    RedirectMode   string         // ">", ">>" (redirect)
    RedirectTarget *Command       // For redirect operators
}
```

### Output: Execution Plan

The plan is serializable and executable:

From `/home/user/opal/core/planfmt/plan.go` lines 25-30:

```go
type Plan struct {
    Header  PlanHeader
    Target  string   // Function/command being executed
    Steps   []Step   // List of steps (newline-separated)
    Secrets []Secret // Secrets to scrub from output
}

type Step struct {
    ID   uint64        // Unique identifier
    Tree ExecutionNode // Operator precedence tree (REQUIRED)
}
```

### Execution Tree Nodes

From `/home/user/opal/core/planfmt/execution_tree.go` lines 13-95:

```go
// CommandNode - leaf node for a single decorator invocation
type CommandNode struct {
    Decorator string     // "@shell", "@retry", etc.
    Args      []Arg      // Decorator arguments (sorted by key)
    Block     []Step     // Nested steps (for decorators with blocks)
}

// PipelineNode - piped commands
type PipelineNode struct {
    Commands []ExecutionNode
}

// AndNode, OrNode - control flow
// RedirectNode - output redirection (> >>)
```

### Planner Test Example

From `/home/user/opal/runtime/planner/planner_test.go` lines 44-82:

```go
// Input: simple shell command
source := []byte(`echo "Hello, World!"`)

// Parse
tree := parser.Parse(source)

// Plan (script mode - no target)
plan, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{
    Target: "", // Script mode
})

// Output plan structure:
// Plan.Steps[0]:
//   Tree: CommandNode{
//     Decorator: "@shell",
//     Args: [{Key: "command", Val: Value{Kind: ValueString, Str: "echo \"Hello, World!\""}}],
//   }
```

### Decorator Concatenation

**Current Implementation**: Shell commands are automatically wrapped in `@shell` decorator:

From the test output above, this is what the planner DOES produce:

```
Input:  kubectl scale --replicas=@var.REPLICAS deployment/app
        ↓ (planner processes)
Output: Step{
          Tree: CommandNode{
            Decorator: "@shell",
            Args: [
              {Key: "command", Val: "kubectl scale --replicas=@var.REPLICAS deployment/app"}
            ],
          }
        }
```

**IMPORTANT**: The decorator references like `@var.REPLICAS` are **NOT RESOLVED** at plan time. They remain as literal text in the command string.

### Argument Types

From `/home/user/opal/core/planfmt/plan.go` lines 84-111:

```go
type Arg struct {
    Key string
    Val Value
}

type Value struct {
    Kind ValueKind
    Str  string   // For ValueString
    Int  int64    // For ValueInt
    Bool bool     // For ValueBool
    Ref  uint32   // For ValuePlaceholder (index into placeholder table)
}

type ValueKind uint8
const (
    ValueString      ValueKind = iota
    ValueInt
    ValueBool
    ValuePlaceholder // Placeholder reference
)
```

### What Planner DOES

✅ Converts parser events to plan structure
✅ Handles operator precedence and chaining
✅ Builds execution tree with operators (&&, ||, |, >)
✅ Preserves decorator parameter values as-is
✅ Stores string arguments (including decorator references) in plan

### What Planner DOES NOT Do

❌ Resolve decorator references in strings
❌ Validate that decorators can be executed
❌ Call decorator handlers

---

## Layer 4: Plan → Execution

**Files**:
- `/home/user/opal/runtime/executor/*.go` (empty - planning phase only)
- `/home/user/opal/core/sdk/executor/*.go` (current executor)

### The Current State

**IMPORTANT**: OPAL currently operates in **planning mode only**. The executor layer is minimal.

From `/home/user/opal/runtime/planner/operators_integration_test.go` lines 139-144:

```go
// Current integration test flow:
// Parse → Plan → Execute (shell/bash commands)

// Execute via SDK
steps := planfmt.ToSDKSteps(plan.Steps)
result, err := executor.Execute(context.Background(), steps, executor.Config{})
```

### Step Execution

The executor receives steps and executes them:

```
Step{
  Tree: CommandNode{
    Decorator: "@shell",
    Args: [{Key: "command", Val: "echo \"Hello, World!\""}],
  }
}
↓
executor.Execute() → runs bash command
```

### What Currently Works in Execution

✅ Shell command execution via @shell decorator
✅ Operator chaining (&&, ||, |)
✅ Exit codes and control flow
✅ Basic piping between commands

### What DOES NOT Work Yet

❌ Value decorator resolution (@var, @env)
❌ Execution decorators (@retry, @timeout, @parallel)
❌ Block execution for decorators with blocks
❌ Secret scrubbing from resolved values

---

## Layer 5: Registry & Schemas

**File**: `/home/user/opal/core/types/registry.go`

### Registration Mechanism

Decorators self-register in `init()` functions:

**@var Decorator** (`/home/user/opal/runtime/decorators/var.go` lines 9-20):

```go
func init() {
    schema := types.NewSchema("var", types.KindValue).
        Description("Access script variables").
        PrimaryParam("name", types.TypeString, "Variable name").
        Returns(types.TypeString, "Value of the variable").
        Build()
    
    if err := types.Global().RegisterValueWithSchema(schema, varHandler); err != nil {
        panic(fmt.Sprintf("failed to register @var decorator: %v", err))
    }
}

func varHandler(ctx types.Context, args types.Args) (types.Value, error) {
    if args.Primary == nil {
        return nil, fmt.Errorf("@var requires a variable name")
    }
    
    varName := (*args.Primary).(string)
    value, exists := ctx.Variables[varName]
    if !exists {
        return nil, fmt.Errorf("variable %q not found", varName)
    }
    return value, nil
}
```

**@env Decorator** (`/home/user/opal/runtime/decorators/env.go` lines 42-67):

```go
func init() {
    schema := types.NewSchema("env", types.KindValue).
        Description("Access environment variables").
        PrimaryParam("property", types.TypeString, "Environment variable name").
        Param("default", types.TypeString).
        Description("Default value if environment variable is not set").
        Optional().
        Done().
        Returns(types.TypeString, "Value of the environment variable").
        Build()
    
    decorator := &envDecorator{}
    if err := types.Global().RegisterValueDecoratorWithSchema(schema, decorator, decorator.Handle); err != nil {
        panic(fmt.Sprintf("failed to register @env decorator: %v", err))
    }
}
```

### Registry Data Structures

From `/home/user/opal/core/types/registry.go` lines 51-65:

```go
type DecoratorInfo struct {
    Path             string           // "var", "env", "file.read"
    Kind             DecoratorKind    // Value or Execution
    Schema           DecoratorSchema  // Describes interface
    ValueHandler     ValueHandler     // Handler for values
    ExecutionHandler ExecutionHandler // Handler for execution
    RawHandler       interface{}      // Raw handler (new style)
}

type Registry struct {
    mu         sync.RWMutex
    decorators map[string]DecoratorInfo
}
```

### Handler Types

From `/home/user/opal/core/types/registry.go` lines 33-39:

```go
// ValueHandler returns data with no side effects
type ValueHandler func(ctx Context, args Args) (Value, error)

// ExecutionHandler performs actions with side effects
type ExecutionHandler func(ctx Context, args Args) error

type DecoratorKind int
const (
    DecoratorKindValue DecoratorKind = iota
    DecoratorKindExecution
)
```

### Context & Arguments

From `/home/user/opal/core/types/registry.go` lines 19-31:

```go
type Context struct {
    Variables  map[string]Value  // Variable bindings
    Env        map[string]string // Environment variables
    WorkingDir string            // Current working directory
}

type Args struct {
    Primary *Value           // Primary property: @env.HOME → "HOME"
    Params  map[string]Value // Named parameters: (default="")
    Block   *Block           // Lambda/block for execution decorators
}
```

### Lookup Functions

From `/home/user/opal/core/types/registry.go` lines 100-149:

```go
// IsValueDecorator checks if a decorator is a value decorator
func (r *Registry) IsValueDecorator(path string) bool {
    r.mu.RLock()
    defer r.mu.RUnlock()
    info, exists := r.decorators[path]
    return exists && info.Kind == DecoratorKindValue
}

// GetSchema retrieves the schema for a decorator
func (r *Registry) GetSchema(path string) (DecoratorSchema, bool) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    info, exists := r.decorators[path]
    if !exists {
        return DecoratorSchema{}, false
    }
    return info.Schema, true
}

// GetTransportScope retrieves the transport scope
func (r *Registry) GetTransportScope(path string) TransportScope {
    // Used to determine where decorators can be used
    // ScopeRootOnly: @env (reads local environment)
    // ScopeAgnostic: decorators that work anywhere
}
```

### Test Verification

From `/home/user/opal/runtime/decorators/builtin_test.go` lines 9-18:

```go
func TestBuiltinDecoratorsRegistered(t *testing.T) {
    if !types.Global().IsRegistered("var") {
        t.Error("built-in decorator 'var' should be registered")
    }
    
    if !types.Global().IsRegistered("env") {
        t.Error("built-in decorator 'env' should be registered")
    }
}
```

---

## Complete End-to-End Examples

### Example 1: Simple Variable Reference in Shell Command

**Source Code**:
```opal
var REPLICAS = 3
kubectl scale --replicas=@var.REPLICAS deployment/app
```

#### Layer 1: Lexer Output

```
Tokens:
[0] VAR "var"
[1] IDENTIFIER "REPLICAS"
[2] EQUALS "="
[3] INTEGER "3"
[4] NEWLINE "\n"
[5] IDENTIFIER "kubectl"
[6] IDENTIFIER "scale"
[7] DECREMENT "--"  (Note: "--" is parsed as DECREMENT token!)
[8] IDENTIFIER "replicas"
[9] EQUALS "="
[10] AT ""          ← Decorator marker
[11] VAR "var"      ← Decorator name
[12] DOT "."        ← Property accessor
[13] IDENTIFIER "REPLICAS" ← Property name
[14] IDENTIFIER "deployment"
[15] DIVIDE "/"     (Forward slash tokenized separately)
[16] IDENTIFIER "app"
```

#### Layer 2: Parser Events

```
EventOpen(NodeSource)
  EventOpen(NodeVarDecl)
    EventToken(0)  // var
    EventToken(1)  // REPLICAS
    EventOpen(NodeLiteral)
      EventToken(3)  // 3
    EventClose(NodeLiteral)
  EventClose(NodeVarDecl)
  
  EventStepEnter
  EventOpen(NodeShellCommand)
    EventToken(5)  // kubectl
    EventToken(6)  // scale
    EventToken(7)  // --
    EventToken(8)  // replicas
    EventToken(9)  // =
    EventOpen(NodeDecorator)
      EventToken(10) // @
      EventToken(11) // var
      EventToken(12) // .
      EventToken(13) // REPLICAS
    EventClose(NodeDecorator)
    EventToken(14)  // deployment
    EventToken(15)  // /
    EventToken(16)  // app
  EventClose(NodeShellCommand)
  EventStepExit
EventClose(NodeSource)
```

#### Layer 3: Plan Output

```go
Plan{
  Steps: []Step{
    {
      ID: 1,
      Tree: CommandNode{
        Decorator: "@shell",
        Args: []Arg{
          {
            Key: "command",
            Val: Value{
              Kind: ValueString,
              Str:  `kubectl scale --replicas=@var.REPLICAS deployment/app`,
            },
          },
        },
      },
    },
  },
  Secrets: []Secret{}, // No resolved secrets yet
}
```

**Key Points**:
- The `@var.REPLICAS` reference is **NOT resolved** at plan time
- It remains as literal text in the command string
- The planner has NO KNOWLEDGE of what REPLICAS is

#### Layer 4: Execution

```
When executing @shell("kubectl scale --replicas=@var.REPLICAS deployment/app"):
1. @shell decorator extracts command string
2. Passes to bash executor
3. Bash sees literal "@var.REPLICAS" in command string
4. Bash treats it as a literal argument (not a variable reference)
5. Kubectl receives: "--replicas=@var.REPLICAS"
6. Command FAILS because kubectl doesn't recognize "@var.REPLICAS"
```

**Current Status**: ❌ NOT WORKING

---

### Example 2: Environment Variable with Default

**Source Code**:
```opal
echo "Database: @env.DATABASE_URL(default='localhost')"
```

#### Layer 1: Lexer Output

```
Tokens:
[0] IDENTIFIER "echo"
[1] STRING "\"Database: @env.DATABASE_URL(default='localhost')\""
```

#### Layer 2: Parser Events - String Interpolation

```
EventOpen(NodeShellCommand)
  EventToken(0) // echo
  EventOpen(NodeInterpolatedString)
    EventToken(1) // STRING token (with quotes)
    // String content is tokenized separately by TokenizeString()
    StringPart{
      Start: 10,
      End: 13,
      IsLiteral: false,  // This is a decorator reference
      PropertyStart: 14,
      PropertyEnd: 28,
    } // @env.DATABASE_URL
  EventClose(NodeInterpolatedString)
EventClose(NodeShellCommand)
```

#### Layer 3: Plan Output

```go
Step{
  ID: 1,
  Tree: CommandNode{
    Decorator: "@shell",
    Args: []Arg{
      {
        Key: "command",
        Val: Value{
          Kind: ValueString,
          Str:  `echo "Database: @env.DATABASE_URL(default='localhost')"`,
        },
      },
    },
  },
}
```

#### What Happens During Execution

```
@shell executes: echo "Database: @env.DATABASE_URL(default='localhost')"
Output: Database: @env.DATABASE_URL(default='localhost')
```

**Current Status**: ❌ Environment variable NOT resolved

---

### Example 3: Decorator with Block (@retry)

**Source Code**:
```opal
@retry(times=3) {
  kubectl apply -f deployment.yaml
  kubectl rollout status deployment/app
}
```

#### Layer 2: Parser Events

```
EventStepEnter
EventOpen(NodeDecorator)
  EventToken(0) // @
  EventToken(1) // retry
  EventOpen(NodeParamList)
    EventToken(2) // (
    EventOpen(NodeParam)
      EventToken(3) // times
      EventToken(4) // =
      EventToken(5) // 3
    EventClose(NodeParam)
    EventToken(6) // )
  EventClose(NodeParamList)
  EventOpen(NodeBlock)
    EventToken(7) // {
    EventOpen(NodeShellCommand)
      // ... shell command events ...
    EventClose(NodeShellCommand)
    EventToken(...) // }
  EventClose(NodeBlock)
EventClose(NodeDecorator)
EventStepExit
```

#### Layer 3: Plan Output

```go
Step{
  ID: 1,
  Tree: CommandNode{
    Decorator: "@retry",
    Args: []Arg{
      {Key: "times", Val: Value{Kind: ValueInt, Int: 3}},
    },
    Block: []Step{  // Nested steps
      {
        ID: 2,
        Tree: CommandNode{
          Decorator: "@shell",
          Args: []Arg{
            {Key: "command", Val: Value{Kind: ValueString, Str: "kubectl apply -f deployment.yaml"}},
          },
        },
      },
      {
        ID: 3,
        Tree: CommandNode{
          Decorator: "@shell",
          Args: []Arg{
            {Key: "command", Val: Value{Kind: ValueString, Str: "kubectl rollout status deployment/app"}},
          },
        },
      },
    },
  },
}
```

**Current Status**: 
- ✅ Parser recognizes @retry decorator
- ✅ Plan captures parameters and block structure
- ❌ Executor doesn't actually implement retry logic yet

---

## Data Structures at Each Layer

### Lexer Output

**Token**:
```go
type Token struct {
    Type     TokenType  // AT=28, IDENTIFIER=82, VAR=50, DOT=29
    Text     []byte     // Raw text (zero-copy)
    Position Position   // Line, column, byte offset
    HasSpaceBefore bool // Parsing hint
}
```

**Example**: Token{Type: AT, Text: []byte(""), Position: {Line: 2, Column: 26, Offset: 50}}

### Parser Output

**Event**:
```go
type Event struct {
    Kind EventKind // EventOpen, EventClose, EventToken, EventStepEnter, EventStepExit
    Data uint32    // For EventToken: index into Tokens array. For EventOpen/Close: NodeKind
}
```

**Example**: Event{Kind: EventOpen, Data: uint32(NodeDecorator)}

### Planner Output

**Step**:
```go
type Step struct {
    ID   uint64
    Tree ExecutionNode  // CommandNode, PipelineNode, AndNode, OrNode, etc.
}
```

**CommandNode**:
```go
type CommandNode struct {
    Decorator string     // "@shell", "@retry", "@env", etc.
    Args      []Arg      // Sorted by Key
    Block     []Step     // For decorators with blocks
}
```

**Arg**:
```go
type Arg struct {
    Key string         // "command", "times", "delay", "default"
    Val Value          // {Kind: ValueString, Str: "..."}
}
```

### Execution Context

**Context**:
```go
type Context struct {
    Variables  map[string]Value  // var name → value
    Env        map[string]string // env var → value
    WorkingDir string
}
```

**Args**:
```go
type Args struct {
    Primary *Value           // Extracted from @decorator.PROPERTY
    Params  map[string]Value // From parentheses
    Block   *Block           // For control-flow decorators
}
```

---

## Code References

### Lexer Files
- `/home/user/opal/runtime/lexer/tokens.go` - Token types and single/two-char token mapping
- `/home/user/opal/runtime/lexer/lexer.go` - Lexer main implementation
- `/home/user/opal/runtime/lexer/decorators_test.go` - Test examples (@ symbol, decorator names)

### Parser Files
- `/home/user/opal/runtime/parser/parser.go` - Main parser, `decorator()` function (line ~260+)
- `/home/user/opal/runtime/parser/tree.go` - AST node types (NodeDecorator, NodeInterpolatedString, etc.)
- `/home/user/opal/runtime/parser/string_tokenizer.go` - String interpolation tokenization
- `/home/user/opal/runtime/parser/decorator_test.go` - Decorator parsing tests

### Planner Files
- `/home/user/opal/runtime/planner/planner.go` - Planner main, Command structure
- `/home/user/opal/core/planfmt/plan.go` - Plan, Step, Arg, Value structures
- `/home/user/opal/core/planfmt/execution_tree.go` - ExecutionNode types (CommandNode, etc.)
- `/home/user/opal/runtime/planner/planner_test.go` - Test examples

### Registry & Decorators
- `/home/user/opal/core/types/registry.go` - Registry, DecoratorInfo, registration methods
- `/home/user/opal/runtime/decorators/var.go` - @var decorator implementation
- `/home/user/opal/runtime/decorators/env.go` - @env decorator implementation
- `/home/user/opal/runtime/decorators/shell.go` - @shell decorator implementation
- `/home/user/opal/runtime/decorators/retry.go` - @retry decorator skeleton
- `/home/user/opal/runtime/decorators/builtin_test.go` - Test that decorators are registered

---

## Current Behavior Matrix

| Decorator | Lexer | Parser | Planner | Executor | Status |
|-----------|-------|--------|---------|----------|--------|
| **@var** | ✅ Tokenized | ✅ Recognized | ✅ Planned | ❌ Not resolved | Value decorator registered, but NOT executed |
| **@env** | ✅ Tokenized | ✅ Recognized | ✅ Planned | ❌ Not resolved | Value decorator registered, but NOT executed |
| **@shell** | ✅ Tokenized | ✅ Implicit wrapper | ✅ All shell cmds wrapped | ✅ Executes | Works |
| **@retry** | ✅ Tokenized | ✅ Recognized, validates params | ✅ Planned with block | ❌ Block not executed | Execution decorator registered but no handler |
| **@timeout** | ✅ Tokenized | ✅ Recognized | ✅ Planned | ❌ Not implemented | Registered but no execution |
| **@parallel** | ✅ Tokenized | ✅ Recognized, requires block | ✅ Planned with block | ❌ Not implemented | Registered but no execution |

---

## Summary: What EXISTS vs What's MISSING

### ✅ FULLY WORKING
1. **Decorator Registration**: `init()` self-registration pattern works
2. **Lexer**: Correctly tokenizes @ symbol and decorator names
3. **Parser**: Recognizes registered decorators, validates parameters, detects string interpolation
4. **Plan Structure**: Captures complete decorator information (name, parameters, blocks)
5. **@shell**: Full implementation and execution
6. **Parameter Validation**: Parse-time type checking of decorator parameters

### ⚠️ PARTIALLY WORKING
1. **String Interpolation**: Parser detects `@var.name` in strings but doesn't mark them for special handling
2. **Value Decorators**: Registered with handlers, but handlers are never called

### ❌ NOT IMPLEMENTED
1. **Value Decorator Resolution**: `@var.name` and `@env.VAR` are not resolved during execution
2. **Execution Decorators**: @retry, @timeout, @parallel are registered but have no handlers
3. **Block Execution**: Decorator blocks are planned but never executed
4. **Secret Scrubbing**: Resolved values are not collected for output scrubbing
5. **Transport Scope**: Environment variable scope limitations not enforced

