# Testing Module

The `testing` module provides comprehensive testing utilities and frameworks for validating devcmd decorators and components.

## Purpose

This module offers testing infrastructure that simplifies decorator development and validation:

- **Decorator Testing**: High-level API for testing decorators without implementation details
- **Test Harness**: Simulation environment for all execution modes
- **Test Builders**: Fluent interfaces for building comprehensive test cases
- **Example Tests**: Real-world test examples and patterns

## Key Components

### Testing Framework (`testing/`)
- `decorator_harness.go`: Core testing harness for decorator validation
- `test_builder.go`: Fluent test building API
- `test_decorators.go`: Comprehensive decorator test utilities
- `examples_test.go`: Example test patterns and usage

## Module Dependencies

- **Depends on**: `core` and `runtime` modules for AST types and execution context
- **Used by**: Decorator implementations for comprehensive testing
- **External**: Standard Go testing package only

## Key Features

### Execution Mode Testing
Tests decorators across all three execution modes:
1. **InterpreterMode**: Direct execution like `devcmd run`
2. **GeneratorMode**: Code generation like `devcmd build`
3. **PlanMode**: Execution planning like `devcmd --dry-run`

### Decorator Type Support
- **Function decorators**: `@var(name)`, `@env(name)`, `@cmd(name)`
- **Block decorators**: `@timeout{}`, `@parallel{}`, `@workdir{}`
- **Pattern decorators**: `@when{}`, `@try{}` with conditional logic

### Test Builder API
Fluent interface for comprehensive test construction:

```go
testing.NewTestBuilder("test_name").
    WithDecorator(myDecorator).
    AsFunctionDecorator().
    WithVariable("VAR", "value").
    WithParam("param", "value").
    ExpectSuccess().
    ExpectData("expected").
    RunTest(t)
```

## Usage Examples

### Function Decorator Testing
```go
func TestVarDecorator(t *testing.T) {
    testing.NewTestBuilder("var_decorator").
        WithDecorator(&decorators.VarDecorator{}).
        AsFunctionDecorator().
        WithVariable("APP_NAME", "myapp").
        WithParam("name", "APP_NAME").
        ExpectSuccess().
        ExpectData("myapp").
        RunTest(t)
}
```

### Block Decorator Testing
```go
func TestWorkdirDecorator(t *testing.T) {
    testing.NewTestBuilder("workdir_decorator").
        WithDecorator(&decorators.WorkdirDecorator{}).
        AsBlockDecorator().
        WithParam("path", "/tmp/test").
        WithCommands("pwd", "echo test").
        ExpectSuccess().
        ExpectCommandsExecuted(2).
        RunTest(t)
}
```

### Pattern Decorator Testing
```go
func TestWhenDecorator(t *testing.T) {
    testing.NewTestBuilder("when_decorator").
        WithDecorator(&decorators.WhenDecorator{}).
        AsPatternDecorator().
        WithEnv("NODE_ENV", "production").
        WithParam("variable", "NODE_ENV").
        WithPattern("production", "npm run build").
        WithPattern("default", "echo fallback").
        ExpectSuccess().
        RunTest(t)
}
```

## Test Validation

### Built-in Assertions
- `ExpectSuccess()`: Execution should succeed
- `ExpectFailure(message)`: Should fail with specific error
- `ExpectData(expected)`: Validate result data
- `ExpectDataContains(substring)`: Check data contains text
- `ExpectExecutionTime(duration)`: Performance validation
- `ExpectCommandsExecuted(count)`: Command execution count

### Custom Validators
```go
builder.WithCustomValidator(func(result testing.TestResult) error {
    if result.Mode == testing.GeneratorMode {
        if code, ok := result.Data.(string); ok {
            if !strings.Contains(code, "expected_pattern") {
                return fmt.Errorf("Generated code missing pattern")
            }
        }
    }
    return nil
})
```

## Architecture

### Test Harness
- Simulates full devcmd execution environment
- Manages variables, environment, and working directory
- Tracks execution history and performance metrics
- Provides isolation between test cases

### Test Suite Organization
- Group related tests with shared setup/teardown
- Comprehensive coverage across all execution modes
- Performance benchmarking and validation
- Clear test failure reporting and debugging

## Development Workflow

### Adding New Decorator Tests
1. Create test file following `*_test.go` pattern
2. Use TestBuilder for fluent test construction
3. Test all execution modes (interpreter, generator, plan)
4. Include both success and error cases
5. Validate performance where relevant

### Best Practices
- **Comprehensive Coverage**: Test all execution modes
- **Clear Naming**: Use descriptive test names
- **Error Testing**: Validate failure scenarios
- **Performance**: Check execution time for critical decorators
- **Code Validation**: Verify generated code in generator mode

## Integration

The testing module integrates seamlessly with:
- Built-in decorators in `cli/internal/builtins/`
- Custom decorator implementations
- CI/CD pipelines for automated validation
- Development workflows for rapid decorator testing

This module ensures decorator reliability and correctness across all devcmd execution scenarios.