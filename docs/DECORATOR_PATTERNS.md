# Decorator Patterns

**Design patterns for building composable, deterministic decorators in Opal**

This document covers common patterns for implementing decorators that maintain Opal's core guarantees: deterministic planning, contract verification, and safe execution.

## Core Principles

Every decorator must maintain these invariants:

1. **Referential transparency** - Same inputs produce same plan
2. **No side effects during planning** - Planning only computes what will execute
3. **Deterministic resolution** - Ambiguity causes plan-time errors
4. **Observable execution** - All actions are traceable and auditable

## Pattern: Opaque Capability Handles

**Use case**: Pass authentication, configuration, or connection context between decorators without embedding secrets or breaking determinism.

**Examples**: Auth credentials, database connections, API clients, cloud provider contexts

### Design

**Value decorator returns a handle:**
```opal
var prodAuth = @aws.auth(profile="prod", role_arn="arn:aws:iam::123:role/ci")
var dbConn = @postgres.connection(host="db.prod", database="app")
```

The value is a **pure spec** (immutable). It contains only *parameters*, not live connections or credentials.

**Other decorators accept the handle:**
```opal
# Secrets using auth handle
var db_password = @aws.secret.db_password(prodAuth)
var api_key = @aws.secret.api_key(prodAuth)

# Database operations using connection handle
var users = @postgres.query("SELECT * FROM users", dbConn)
```

**Plan representation:**
```json
{
  "steps": [
    {
      "decorator": "aws.secret.db_password",
      "args": { "auth": "<auth:aws:3e8f...>" }
    },
    {
      "decorator": "postgres.query",
      "args": { "query": "SELECT * FROM users", "conn": "<conn:postgres:a7b2...>" }
    }
  ]
}
```

At execution time, handles are resolved to live resources (memoized per run).

### Scoped vs Handle Style

**Scoped** (ergonomic for blocks):
```opal
@aws.auth(profile="prod") {
    var db_pass = @aws.secret.db_password
    var api_key = @aws.secret.api_key
}
```

**Handle** (composable, passable to functions):
```opal
var prodAuth = @aws.auth(profile="prod")

fun deploy(auth) {
    var secret = @aws.secret.db_password(@var.auth)
    kubectl apply -f k8s/
}

deploy(auth=prodAuth)
```

## Pattern: Resource Collections

**Use case**: Work with multiple cloud resources (instances, containers, files) as a collection.

**Examples**: EC2 instances, S3 objects, Kubernetes pods, Docker containers

### Design

**Value decorator returns collection:**
```opal
# Query instances matching criteria
var webServers = @aws.ec2.instances(
    tags={role: "web", env: "prod"},
    state="running"
)

# Query pods
var appPods = @k8s.pods(
    namespace="production",
    labels={app: "api"}
)
```

**Execution decorator operates on collection:**
```opal
# Execute command on all instances
@aws.ec2.run(instances=webServers, transport="ssm") {
    sudo systemctl restart nginx
}

# Execute in all pods
@k8s.exec(pods=appPods) {
    curl -f http://localhost:8080/health
}
```

**Iteration over collection:**
```opal
for instance in @var.webServers {
    echo "Checking @var.instance.id at @var.instance.private_ip"
}
```

### Collection Properties

Collections expose structured data:
```opal
var instances = @aws.ec2.instances(tags={role: "web"})

# Access properties
echo "Found @var.instances.count instances"
echo "IDs: @var.instances.ids"
echo "IPs: @var.instances.private_ips"
```

## Pattern: Hierarchical Namespaces

**Use case**: Organize related decorators in a logical hierarchy.

**Examples**: Cloud provider services, configuration sources, monitoring systems

### Design

**Dot notation for hierarchy:**
```opal
# AWS services
@aws.secret.db_password
@aws.ec2.instances
@aws.s3.objects
@aws.rds.databases

# Kubernetes resources
@k8s.pods
@k8s.deployments
@k8s.services

# Configuration sources
@config.app.database_url
@config.app.api_key
@env.HOME
@var.MY_VAR
```

**Namespace structure:**
```
@aws
  ├─ auth()          - Authentication handle
  ├─ secret
  │   ├─ db_password - Secret value
  │   └─ api_key     - Secret value
  ├─ ec2
  │   ├─ instances() - Query instances
  │   └─ run()       - Execute on instances
  └─ s3
      ├─ objects()   - Query objects
      └─ upload()    - Upload files
```

## Pattern: Memoized Resolution

**Use case**: Avoid redundant API calls for the same value.

**Implementation**: Cache resolved values by decorator + arguments hash.

### Example

```opal
# First access: API call to fetch secret
var db_pass = @aws.secret.db_password(prodAuth)

# Second access: Uses cached value (no API call)
var db_pass_copy = @aws.secret.db_password(prodAuth)

# Different args: New API call
var api_key = @aws.secret.api_key(prodAuth)
```

**Performance:**
- First access: ~150ms (API call)
- Subsequent access: <1ms (cache hit)

## Pattern: Batch Resolution

**Use case**: Multiple decorators fetching from the same provider should batch requests.

**Implementation**: Collect requests during planning, execute in bulk.

### Example

```opal
var prodAuth = @aws.auth(profile="prod")

# All three collected during planning
var db_pass = @aws.secret.db_password(prodAuth)
var api_key = @aws.secret.api_key(prodAuth)
var cert = @aws.secret.tls_cert(prodAuth)

# Executed as single batch API call:
# BatchGetSecretValue(["db_password", "api_key", "tls_cert"])
```

**Performance:**
- Without batching: 3 API calls × 150ms = 450ms
- With batching: 1 API call = 150ms
- With batching + memoization: 150ms → <1ms on subsequent runs

## Pattern: Path-Aware Resolution

**Use case**: Don't resolve values on code paths that won't execute.

**Implementation**: Only resolve decorators on the active execution path.

### Example

```opal
when @var.ENV {
    "production" -> {
        var secret = @aws.secret.prod_db(prodAuth)  # Only if ENV=production
    }
    "staging" -> {
        var secret = @aws.secret.staging_db(stagingAuth)  # Only if ENV=staging
    }
}
```

If `ENV=production`, the staging secret is never fetched (saves API call + cost).

## Pattern: Deterministic Fallbacks

**Use case**: Provide sensible defaults while maintaining determinism.

**Implementation**: Well-defined resolution chain with explicit priority.

### Example

```opal
# Resolution order (highest to lowest priority):
# 1. Explicit parameter
# 2. Scoped context
# 3. Project config
# 4. Environment variable
# 5. Default value

# Explicit wins
var auth = @aws.auth(profile="prod")
var secret = @aws.secret.db_password(auth)  # Uses "prod" profile

# Scoped context
@aws.auth(profile="staging") {
    var secret = @aws.secret.db_password  # Uses "staging" profile
}

# Project config (if no explicit/scoped)
# Reads from .opal/config.yml: aws.profile = "dev"
var secret = @aws.secret.db_password  # Uses "dev" profile

# Ambiguous = plan-time error
# ERROR: Multiple auth sources, unclear which to use
```

## Pattern: Type-Safe Handles

**Use case**: Prevent passing wrong handle type to decorators.

**Implementation**: Use nominal types with optional type annotations.

### Example

```opal
fun deploy(auth: Auth[AWS], db: Connection[Postgres]) {
    var secret = @aws.secret.db_password(@var.auth)
    var users = @postgres.query("SELECT * FROM users", @var.db)
}

var prodAuth = @aws.auth(profile="prod")  # Type: Auth[AWS]
var dbConn = @postgres.connection(host="db.prod")  # Type: Connection[Postgres]

deploy(auth=prodAuth, db=dbConn)  # ✓ Types match

# Type error caught at plan-time:
deploy(auth=dbConn, db=prodAuth)  # ✗ Type mismatch
```

## Best Practices

### 1. Fail Fast at Plan-Time

Catch errors during planning, not execution:

```opal
# Good: Clear error at plan-time
var secret = @aws.secret.db_password  # ERROR: no auth specified

# Bad: Would fail at execution time
var secret = @aws.secret.db_password  # Silently plans, fails at runtime
```

### 2. Make Ambiguity Explicit

```opal
# Bad: Implicit, ambiguous
var instances = @aws.ec2.instances  # Which region? Which account?

# Good: Explicit, deterministic
var prodAuth = @aws.auth(profile="prod", region="us-east-1")
var instances = @aws.ec2.instances(tags={env: "prod"}, auth=prodAuth)
```

### 3. Design for Observability

Every decorator should emit telemetry:

```
Decorator execution summary:
  aws.secret.db_password (auth=<3e8f...>): 145ms, success
  aws.secret.api_key (auth=<3e8f...>): <1ms, cached
  postgres.query (conn=<a7b2...>): 23ms, 150 rows
```

### 4. Redact Secrets

Never log, print, or store raw credentials:

```opal
var secret = @aws.secret.db_password(prodAuth)
echo "Secret: @var.secret"  # Output: "Secret: <secret:redacted>"
```

### 5. Support Composition

Decorators should compose naturally:

```opal
# Auth handle used by multiple services
var prodAuth = @aws.auth(profile="prod")

# Connection uses secret from auth context
var dbPass = @aws.secret.db_password(prodAuth)
var dbConn = @postgres.connection(
    host="db.prod",
    password=dbPass
)

# Query uses connection
var users = @postgres.query("SELECT * FROM users", dbConn)
```

## Common Decorator Types

### Value Decorators

Return pure values (no side effects during planning):

- `@aws.auth()` - Auth handle
- `@aws.secret.NAME` - Secret value
- `@aws.ec2.instances()` - Instance collection
- `@env.VAR` - Environment variable
- `@var.NAME` - Script variable
- `@config.path.to.value` - Config value

### Execution Decorators

Perform actions (side effects during execution):

- `@aws.ec2.run()` - Execute on instances
- `@k8s.exec()` - Execute in pods
- `@retry()` - Retry with backoff
- `@parallel()` - Parallel execution
- `@shell()` - Shell command

### Scoped Decorators

Create context for nested blocks:

- `@aws.auth() { ... }` - Auth scope
- `@workdir() { ... }` - Working directory
- `@env() { ... }` - Environment variables
- `@timeout() { ... }` - Timeout constraint

## Summary

These patterns enable:
- **Composable handles** - Pass context between decorators
- **Deterministic planning** - Same inputs always produce same plan
- **Efficient execution** - Memoization and batching reduce API calls
- **Observable operations** - Full traceability without exposing secrets
- **Type safety** - Optional types catch errors at plan-time
- **Natural composition** - Decorators work together seamlessly

All while maintaining Opal's core guarantee: **resolved plans are execution contracts**.
