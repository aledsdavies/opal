package executor

import (
	"context"
	"testing"

	"github.com/aledsdavies/opal/core/planfmt"
)

// TestExecutionContext_ArgString tests retrieving string arguments
func TestExecutionContext_ArgString(t *testing.T) {
	// Given: A command with string argument
	cmd := planfmt.Command{
		Decorator: "shell",
		Args: []planfmt.Arg{
			{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo hello"}},
		},
	}

	// When: Creating execution context
	ctx := newExecutionContext(cmd, nil, context.Background())

	// Then: Can retrieve argument
	got := ctx.ArgString("command")
	want := "echo hello"
	if got != want {
		t.Errorf("ArgString() = %q, want %q", got, want)
	}
}

// TestExecutionContext_ArgString_Missing tests missing argument returns empty string
func TestExecutionContext_ArgString_Missing(t *testing.T) {
	// Given: A command without the requested argument
	cmd := planfmt.Command{
		Decorator: "shell",
		Args:      []planfmt.Arg{},
	}

	// When: Creating execution context
	ctx := newExecutionContext(cmd, nil, context.Background())

	// Then: Returns empty string for missing argument
	got := ctx.ArgString("missing")
	want := ""
	if got != want {
		t.Errorf("ArgString(missing) = %q, want %q", got, want)
	}
}

// TestExecutionContext_Context tests retrieving Go context
func TestExecutionContext_Context(t *testing.T) {
	// Given: Execution context with Go context
	cmd := planfmt.Command{Decorator: "test"}
	goCtx := context.Background()

	// When: Creating execution context
	ctx := newExecutionContext(cmd, nil, goCtx)

	// Then: Can retrieve Go context
	if ctx.Context() != goCtx {
		t.Error("Context() did not return the provided context")
	}
}

// TestExecutionContext_WithContext tests context wrapping
func TestExecutionContext_WithContext(t *testing.T) {
	// Given: Execution context
	cmd := planfmt.Command{Decorator: "test"}
	ctx := newExecutionContext(cmd, nil, context.Background())

	// When: Wrapping with new context
	newGoCtx := context.WithValue(context.Background(), "key", "value")
	wrapped := ctx.WithContext(newGoCtx)

	// Then: New context has the wrapped Go context
	if wrapped.Context() != newGoCtx {
		t.Error("WithContext() did not wrap the context")
	}

	// And: Original context unchanged
	if ctx.Context() == newGoCtx {
		t.Error("WithContext() modified original context")
	}
}

// TestExecutionContext_WithEnviron tests environment isolation
func TestExecutionContext_WithEnviron(t *testing.T) {
	// Given: Execution context with original environment
	cmd := planfmt.Command{Decorator: "test"}
	ctx := newExecutionContext(cmd, nil, context.Background())
	originalEnv := ctx.Environ()

	// When: Creating new context with modified environment
	newEnv := map[string]string{
		"TEST_VAR": "test_value",
		"FOO":      "bar",
	}
	wrapped := ctx.WithEnviron(newEnv)

	// Then: New context has the new environment
	wrappedEnv := wrapped.Environ()
	if wrappedEnv["TEST_VAR"] != "test_value" {
		t.Errorf("WithEnviron() new context missing TEST_VAR")
	}
	if wrappedEnv["FOO"] != "bar" {
		t.Errorf("WithEnviron() new context missing FOO")
	}

	// And: Original context unchanged
	if _, exists := originalEnv["TEST_VAR"]; exists {
		t.Error("WithEnviron() modified original context environment")
	}

	// And: Modifying the input map doesn't affect the context
	newEnv["ADDED"] = "after"
	if _, exists := wrapped.Environ()["ADDED"]; exists {
		t.Error("WithEnviron() did not deep copy environment")
	}
}

// TestExecutionContext_WithWorkdir tests working directory isolation
func TestExecutionContext_WithWorkdir(t *testing.T) {
	// Given: Execution context with original workdir
	cmd := planfmt.Command{Decorator: "test"}
	ctx := newExecutionContext(cmd, nil, context.Background())
	originalWd := ctx.Workdir()

	// When: Creating new context with different workdir
	newWd := "/tmp"
	wrapped := ctx.WithWorkdir(newWd)

	// Then: New context has the new workdir
	if wrapped.Workdir() != newWd {
		t.Errorf("WithWorkdir() = %q, want %q", wrapped.Workdir(), newWd)
	}

	// And: Original context unchanged
	if ctx.Workdir() != originalWd {
		t.Error("WithWorkdir() modified original context")
	}
}

// TestExecutionContext_IsolationForParallel tests that contexts are properly isolated
// This is critical for @parallel decorator where each branch must be independent
func TestExecutionContext_IsolationForParallel(t *testing.T) {
	// Given: Base execution context
	cmd := planfmt.Command{Decorator: "test"}
	baseCtx := newExecutionContext(cmd, nil, context.Background())

	// When: Creating two "parallel" contexts with different environments
	env1 := map[string]string{"BRANCH": "A", "VALUE": "1"}
	env2 := map[string]string{"BRANCH": "B", "VALUE": "2"}
	ctx1 := baseCtx.WithEnviron(env1).WithWorkdir("/tmp/branch-a")
	ctx2 := baseCtx.WithEnviron(env2).WithWorkdir("/tmp/branch-b")

	// Then: Each context has its own isolated state
	if ctx1.Environ()["BRANCH"] != "A" {
		t.Error("ctx1 does not have isolated environment")
	}
	if ctx2.Environ()["BRANCH"] != "B" {
		t.Error("ctx2 does not have isolated environment")
	}

	// And: Workdirs are isolated
	if ctx1.Workdir() != "/tmp/branch-a" {
		t.Error("ctx1 does not have isolated workdir")
	}
	if ctx2.Workdir() != "/tmp/branch-b" {
		t.Error("ctx2 does not have isolated workdir")
	}

	// And: Base context unchanged
	if _, exists := baseCtx.Environ()["BRANCH"]; exists {
		t.Error("base context was modified")
	}
}

// TestExecutionContext_ChainedWrapping tests multiple levels of context wrapping
func TestExecutionContext_ChainedWrapping(t *testing.T) {
	// Given: Base execution context
	cmd := planfmt.Command{Decorator: "test"}
	baseCtx := newExecutionContext(cmd, nil, context.Background())

	// When: Chaining multiple context modifications
	ctx1 := baseCtx.WithWorkdir("/tmp")
	ctx2 := ctx1.WithEnviron(map[string]string{"VAR": "value"})
	ctx3 := ctx2.WithContext(context.WithValue(context.Background(), "key", "val"))

	// Then: Each level preserves previous modifications
	if ctx3.Workdir() != "/tmp" {
		t.Error("chained context lost workdir")
	}
	if ctx3.Environ()["VAR"] != "value" {
		t.Error("chained context lost environment")
	}
	if ctx3.Context().Value("key") != "val" {
		t.Error("chained context lost Go context value")
	}

	// And: Earlier contexts unchanged
	if ctx1.Environ()["VAR"] == "value" {
		t.Error("earlier context was modified")
	}
}

// TestExecutionContext_ArgInt tests integer argument retrieval
func TestExecutionContext_ArgInt(t *testing.T) {
	// Given: Command with int argument
	cmd := planfmt.Command{
		Decorator: "retry",
		Args: []planfmt.Arg{
			{Key: "times", Val: planfmt.Value{Kind: planfmt.ValueInt, Int: 3}},
		},
	}

	// When: Creating execution context
	ctx := newExecutionContext(cmd, nil, context.Background())

	// Then: Can retrieve int argument
	if got := ctx.ArgInt("times"); got != 3 {
		t.Errorf("ArgInt(times) = %d, want 3", got)
	}

	// And: Missing argument returns 0
	if got := ctx.ArgInt("missing"); got != 0 {
		t.Errorf("ArgInt(missing) = %d, want 0", got)
	}
}

// TestExecutionContext_ArgBool tests boolean argument retrieval
func TestExecutionContext_ArgBool(t *testing.T) {
	// Given: Command with bool argument
	cmd := planfmt.Command{
		Decorator: "test",
		Args: []planfmt.Arg{
			{Key: "enabled", Val: planfmt.Value{Kind: planfmt.ValueBool, Bool: true}},
		},
	}

	// When: Creating execution context
	ctx := newExecutionContext(cmd, nil, context.Background())

	// Then: Can retrieve bool argument
	if got := ctx.ArgBool("enabled"); got != true {
		t.Errorf("ArgBool(enabled) = %v, want true", got)
	}

	// And: Missing argument returns false
	if got := ctx.ArgBool("missing"); got != false {
		t.Errorf("ArgBool(missing) = %v, want false", got)
	}
}

// TestExecutionContext_Args tests snapshot of all arguments
func TestExecutionContext_Args(t *testing.T) {
	// Given: Command with multiple argument types
	cmd := planfmt.Command{
		Decorator: "test",
		Args: []planfmt.Arg{
			{Key: "name", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "test"}},
			{Key: "count", Val: planfmt.Value{Kind: planfmt.ValueInt, Int: 42}},
			{Key: "enabled", Val: planfmt.Value{Kind: planfmt.ValueBool, Bool: true}},
		},
	}

	// When: Creating execution context and getting args snapshot
	ctx := newExecutionContext(cmd, nil, context.Background())
	args := ctx.Args()

	// Then: Snapshot contains all arguments
	if args["name"] != "test" {
		t.Errorf("Args()[name] = %v, want test", args["name"])
	}
	if args["count"] != int64(42) {
		t.Errorf("Args()[count] = %v, want 42", args["count"])
	}
	if args["enabled"] != true {
		t.Errorf("Args()[enabled] = %v, want true", args["enabled"])
	}
}
