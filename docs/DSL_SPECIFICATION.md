# Opal DSL Specification

Formal grammar and syntax rules for the Opal language.

## Grammar Notation

- `|` - alternation (or)
- `*` - zero or more
- `+` - one or more
- `?` - optional (zero or one)
- `()` - grouping
- `[]` - character class
- `""` - literal string

## Lexical Elements

### Identifiers

```
identifier = letter (letter | digit | "_")*
letter     = [a-zA-Z]
digit      = [0-9]
```

**Examples**: `deploy`, `apiUrl`, `PORT`, `service_name`, `buildAndTest`

### Keywords

```
fun when if else for in try catch finally var
```

### Literals

```
string_literal   = '"' string_char* '"'
int_literal      = digit+
float_literal    = digit+ "." digit+
bool_literal     = "true" | "false"
duration_literal = duration_value duration_unit+

duration_value = digit+
duration_unit  = "ns" | "us" | "ms" | "s" | "m" | "h" | "d" | "w" | "y"
```

**Duration examples**: `30s`, `5m`, `2h`, `1h30m`, `2d12h`

### Operators

```
arithmetic = "+" | "-" | "*" | "/" | "%"
comparison = "==" | "!=" | "<" | ">" | "<=" | ">="
logical    = "&&" | "||" | "!"
assignment = "=" | "+=" | "-=" | "*=" | "/=" | "%="
increment  = "++" | "--"
shell      = "|" | "&&" | "||" | ";"
```

### Delimiters

```
( ) { } [ ] , : . @
```

## Syntax Grammar

### Source File

```
source = declaration*

declaration = function_decl
            | var_decl
```

### Function Declarations

```
function_decl = "fun" identifier param_list ("=" expression | block)

param_list = "(" (param ("," param)*)? ")"

param = identifier type_annotation? default_value?

type_annotation = ":" type_name

default_value = "=" expression

type_name = "String" | "Int" | "Float" | "Bool" | "Duration" | "Array" | "Map"
```

**Examples**:
```opal
fun deploy = kubectl apply -f k8s/
fun greet(name) = echo "Hello @var.name"
fun build(module, target = "dist") { ... }
fun deploy(env: String, replicas: Int = 3) { ... }
```

### Variable Declarations

```
var_decl = "var" (var_spec | "(" var_spec+ ")")

var_spec = identifier ("=" expression)?
```

**Examples**:
```opal
var ENV = @env.ENVIRONMENT
var PORT = 3000
var (
    API_URL = @env.API_URL
    REPLICAS = 3
)
```

### Statements

```
statement = var_decl
          | assignment
          | expression
          | if_stmt
          | when_stmt
          | for_stmt
          | try_stmt
          | block

assignment = identifier assign_op expression

assign_op = "=" | "+=" | "-=" | "*=" | "/=" | "%="
```

### Control Flow

```
if_stmt = "if" expression block ("else" (if_stmt | block))?

when_stmt = "when" expression "{" when_arm+ "}"

when_arm = pattern "->" (expression | block)

pattern = string_literal
        | pattern "|" pattern
        | "{" string_literal ("," string_literal)* "}"
        | "r" string_literal
        | int_literal ".." int_literal
        | "else"

for_stmt = "for" identifier "in" expression block

try_stmt = "try" block ("catch" block)? ("finally" block)?
```

**Examples**:
```opal
if @var.ENV == "production" { ... }

when @var.ENV {
    "production" -> { ... }
    "staging" | "dev" -> { ... }
    else -> { ... }
}

for service in @var.SERVICES { ... }

try { ... } catch { ... } finally { ... }
```

### Expressions

```
expression = primary
           | unary_expr
           | binary_expr
           | call_expr
           | decorator_expr

primary = identifier
        | literal
        | "(" expression ")"
        | array_literal
        | map_literal

array_literal = "[" (expression ("," expression)*)? "]"

map_literal = "{" (map_entry ("," map_entry)*)? "}"

map_entry = string_literal ":" expression

unary_expr = ("!" | "-" | "++" | "--") expression

binary_expr = expression binary_op expression

binary_op = arithmetic | comparison | logical | shell

call_expr = "@cmd." identifier "(" (arg ("," arg)*)? ")"

arg = identifier "=" expression
```

### Decorators

```
decorator_expr = value_decorator | execution_decorator

value_decorator = "@" decorator_path decorator_args?

execution_decorator = "@" decorator_path decorator_args? block

decorator_path = identifier ("." identifier)*

decorator_args = "(" (arg ("," arg)*)? ")"
```

**Value decorators**:
```opal
@env.PORT
@var.REPLICAS
@aws.secret.api_key(auth=prodAuth)
```

**Execution decorators**:
```opal
@retry(attempts=3, delay=2s) { ... }
@timeout(duration=5m) { ... }
@parallel { ... }
```

### Variable Interpolation

```
interpolation = "@var." identifier ("()")? 
              | "@env." identifier ("()")? 
              | decorator_expr
```

**In strings and commands**:
```opal
echo "Deploying @var.service"
kubectl scale --replicas=@var.REPLICAS deployment/@var.service
kubectl apply -f k8s/@var.service/
```

**Termination with `()`**:
```opal
echo "@var.service()_backup"  // Expands to "api_backup"
```

### Blocks

```
block = "{" statement* "}"
```

### Shell Commands

Shell commands are parsed as execution decorators internally:

```
shell_command = shell_token+

shell_token = identifier | string_literal | interpolation | shell_operator
```

**Parser transformation**:
```opal
// Source
npm run build

// Parser generates
@shell("npm run build")
```

## Operator Precedence

From highest to lowest:

1. `()` - grouping, function calls
2. `++`, `--` - increment, decrement
3. `!`, unary `-` - logical not, negation
4. `*`, `/`, `%` - multiplication, division, modulo
5. `+`, `-` - addition, subtraction
6. `<`, `>`, `<=`, `>=` - comparison
7. `==`, `!=` - equality
8. `&&` - logical and
9. `||` - logical or
10. `|` - pipe (shell)
11. `;` - sequence (shell)
12. `=`, `+=`, `-=`, etc. - assignment

## Whitespace and Comments

```
whitespace = " " | "\t" | "\n" | "\r"

comment = "//" any* "\n"
        | "/*" any* "*/"
```

Whitespace is generally insignificant except:
- Newlines separate statements (fail-fast semantics)
- Semicolons override newline semantics (continue on error)
- `HasSpaceBefore` flag preserved for shell command parsing

## Semantic Rules

### Plan-Time vs Runtime

**Plan-time constructs** (deterministic):
- `for` loops - unroll to concrete steps
- `if`/`when` - select single branch
- `fun` - template expansion
- Variable declarations and assignments (in most blocks)

**Runtime constructs**:
- `try`/`catch` - path selection based on exceptions
- Execution decorators - modify command execution
- Shell commands - actual work execution

### Scope Rules

**Outer scope mutations allowed**:
- Regular blocks: `{ ... }`
- `for` loops
- `if`/`when` branches
- `fun` bodies

**Scope isolation** (read outer, mutations stay local):
- `try`/`catch`/`finally` blocks
- Execution decorator blocks (`@retry { ... }`, etc.)

### Type System

**Optional typing** (TypeScript-style):
- Variables untyped by default
- Function parameters can have type annotations
- Type checking at plan-time when types specified
- Future: `--strict-types` flag

**Type inference**:
- From literals: `var x = 3` → Int
- From defaults: `fun f(x = "hi")` → x is String
- From decorators: `@env.PORT` → String (can be converted)

## Plan File Format

Plans are JSON with the following structure:

```json
{
  "version": "1.0",
  "source_commit": "abc123...",
  "spec_version": "0.1.0",
  "compiler_version": "0.1.0",
  "plan_hash": "sha256:def456...",
  "steps": [
    {
      "id": 0,
      "decorator": "shell",
      "args": {
        "cmd": "kubectl apply -f k8s/"
      },
      "dependencies": []
    },
    {
      "id": 1,
      "decorator": "shell",
      "args": {
        "cmd": "kubectl scale --replicas=<1:sha256:abc123> deployment/app"
      },
      "dependencies": [0]
    }
  ],
  "values": {
    "REPLICAS": "<1:sha256:abc123>"
  }
}
```

**Hash placeholder format**: `<length:algorithm:hash>`
- `length` - character count of actual value
- `algorithm` - hash algorithm (sha256)
- `hash` - truncated hash for verification

## Decorator Loading

Decorators are loaded as Go plugins (`.so` files):

```go
// Decorator plugin interface (similar to database/sql)
type ValueDecorator interface {
    Plan(ctx Context, args []Param) (Value, error)
}

type ExecutionDecorator interface {
    Plan(ctx Context, args []Param, block Block) (Plan, error)
    Execute(ctx Context, plan Plan) error
}
```

**Loading at runtime**:
```bash
# CLI discovers decorators in:
~/.opal/decorators/*.so
./.opal/decorators/*.so
```

## Future Extensions

### Struct Types
```opal
type Config {
    database: String
    port: Int
}

var config: Config = {
    database: @env.DATABASE_URL,
    port: 5432
}
```

### Pattern Matching Extensions
```opal
when @var.response {
    { status: 200 } -> echo "Success"
    { status: 404 } -> echo "Not found"
    { status: s } if s >= 500 -> echo "Server error"
}
```

### Async/Await (maybe)
```opal
var result = await @http.get("https://api.example.com")
```

## Implementation Notes

### Parser Strategy
- Event-based parse tree (rust-analyzer style)
- Dual-path: Events → Plan (execution) or Events → AST (tooling)
- Resilient parsing with error recovery
- FIRST/FOLLOW sets for predictive parsing

### Performance Targets
- Lexer: >5000 lines/ms
- Parser: >3000 lines/ms (events)
- Plan generation: <10ms for typical scripts

### Error Recovery
- Synchronization points: `}`, `;`, newline
- Error nodes in parse tree for tooling
- Continue parsing after errors for better diagnostics
