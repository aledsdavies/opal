package executor

import (
	"os"
	"testing"
	"time"

	"github.com/aledsdavies/opal/core/planfmt"
	_ "github.com/aledsdavies/opal/runtime/decorators" // Register built-in decorators
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper to create a simple shell command tree
func shellCmd(cmd string) *planfmt.CommandNode {
	return &planfmt.CommandNode{
		Decorator: "@shell",
		Args: []planfmt.Arg{
			{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: cmd}},
		},
	}
}

// TestExecuteSimpleShellCommand tests executing a single echo command
func TestExecuteSimpleShellCommand(t *testing.T) {
	plan := &planfmt.Plan{
		Target: "hello",
		Steps: []planfmt.Step{
			{
				ID:   1,
				Tree: shellCmd("echo 'Hello, World!'"),
			},
		},
	}

	steps := planfmt.ToSDKSteps(plan.Steps)
	result, err := Execute(steps, Config{})
	require.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
	assert.Equal(t, 1, result.StepsRun)
	assert.Greater(t, result.Duration, time.Duration(0))
}

// TestExecuteMultipleCommands tests executing multiple sequential commands
func TestExecuteMultipleCommands(t *testing.T) {
	plan := &planfmt.Plan{
		Target: "multi",
		Steps: []planfmt.Step{
			{
				ID:   1,
				Tree: shellCmd("echo 'First'"),
			},
			{
				ID:   2,
				Tree: shellCmd("echo 'Second'"),
			},
		},
	}

	steps := planfmt.ToSDKSteps(plan.Steps)
	result, err := Execute(steps, Config{})
	require.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
	assert.Equal(t, 2, result.StepsRun)
}

// TestExecuteFailingCommand tests that non-zero exit codes are returned
func TestExecuteFailingCommand(t *testing.T) {
	plan := &planfmt.Plan{
		Target: "fail",
		Steps: []planfmt.Step{
			{
				ID:   1,
				Tree: shellCmd("exit 42"),
			},
		},
	}

	steps := planfmt.ToSDKSteps(plan.Steps)
	result, err := Execute(steps, Config{})
	require.NoError(t, err) // Execute returns result, not error
	assert.Equal(t, 42, result.ExitCode)
	assert.Equal(t, 1, result.StepsRun)
}

// TestExecuteStopOnFirstFailure tests fail-fast behavior
func TestExecuteStopOnFirstFailure(t *testing.T) {
	plan := &planfmt.Plan{
		Target: "failfast",
		Steps: []planfmt.Step{
			{
				ID:   1,
				Tree: shellCmd("echo 'First'"),
			},
			{
				ID:   2,
				Tree: shellCmd("exit 1"),
			},
			{
				ID:   3,
				Tree: shellCmd("echo 'Should not run'"),
			},
		},
	}

	steps := planfmt.ToSDKSteps(plan.Steps)
	result, err := Execute(steps, Config{})
	require.NoError(t, err)
	assert.Equal(t, 1, result.ExitCode)
	assert.Equal(t, 2, result.StepsRun) // Only first two steps run
}

// TestExecuteEmptyPlan tests that empty plans succeed
func TestExecuteEmptyPlan(t *testing.T) {
	plan := &planfmt.Plan{
		Target: "empty",
		Steps:  []planfmt.Step{},
	}

	steps := planfmt.ToSDKSteps(plan.Steps)
	result, err := Execute(steps, Config{})
	require.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
	assert.Equal(t, 0, result.StepsRun)
}

// TestExecuteTelemetryBasic tests basic telemetry collection
func TestExecuteTelemetryBasic(t *testing.T) {
	plan := &planfmt.Plan{
		Target: "telemetry",
		Steps: []planfmt.Step{
			{
				ID:   1,
				Tree: shellCmd("echo 'test'"),
			},
		},
	}

	steps := planfmt.ToSDKSteps(plan.Steps)
	result, err := Execute(steps, Config{Telemetry: TelemetryBasic})
	require.NoError(t, err)
	require.NotNil(t, result.Telemetry)
	assert.Equal(t, 1, result.Telemetry.StepCount)
	assert.Equal(t, 1, result.Telemetry.StepsRun)
	assert.Nil(t, result.Telemetry.FailedStep)
}

// TestExecuteTelemetryTiming tests timing telemetry collection
func TestExecuteTelemetryTiming(t *testing.T) {
	plan := &planfmt.Plan{
		Target: "timing",
		Steps: []planfmt.Step{
			{
				ID:   1,
				Tree: shellCmd("echo 'test'"),
			},
		},
	}

	steps := planfmt.ToSDKSteps(plan.Steps)
	result, err := Execute(steps, Config{Telemetry: TelemetryTiming})
	require.NoError(t, err)
	require.NotNil(t, result.Telemetry)
	require.NotNil(t, result.Telemetry.StepTimings)
	assert.Len(t, result.Telemetry.StepTimings, 1)
	assert.Equal(t, uint64(1), result.Telemetry.StepTimings[0].StepID)
	assert.Greater(t, result.Telemetry.StepTimings[0].Duration, time.Duration(0))
	assert.Equal(t, 0, result.Telemetry.StepTimings[0].ExitCode)
}

// TestExecuteTelemetryFailedStep tests failed step tracking
func TestExecuteTelemetryFailedStep(t *testing.T) {
	plan := &planfmt.Plan{
		Target: "fail",
		Steps: []planfmt.Step{
			{
				ID:   1,
				Tree: shellCmd("exit 1"),
			},
		},
	}

	steps := planfmt.ToSDKSteps(plan.Steps)
	result, err := Execute(steps, Config{Telemetry: TelemetryBasic})
	require.NoError(t, err)
	require.NotNil(t, result.Telemetry)
	require.NotNil(t, result.Telemetry.FailedStep)
	assert.Equal(t, uint64(1), *result.Telemetry.FailedStep)
}

// TestExecuteDebugPaths tests path-level debug tracing
func TestExecuteDebugPaths(t *testing.T) {
	plan := &planfmt.Plan{
		Target: "debug",
		Steps: []planfmt.Step{
			{
				ID:   1,
				Tree: shellCmd("echo 'test'"),
			},
		},
	}

	steps := planfmt.ToSDKSteps(plan.Steps)
	result, err := Execute(steps, Config{Debug: DebugPaths})
	require.NoError(t, err)
	require.NotNil(t, result.DebugEvents)
	assert.Greater(t, len(result.DebugEvents), 0)

	// Should have enter_execute and exit_execute events
	events := make(map[string]bool)
	for _, e := range result.DebugEvents {
		events[e.Event] = true
	}
	assert.True(t, events["enter_execute"])
	assert.True(t, events["exit_execute"])
}

// TestExecuteDebugDetailed tests detailed debug tracing
func TestExecuteDebugDetailed(t *testing.T) {
	plan := &planfmt.Plan{
		Target: "debug",
		Steps: []planfmt.Step{
			{
				ID:   1,
				Tree: shellCmd("echo 'test'"),
			},
		},
	}

	steps := planfmt.ToSDKSteps(plan.Steps)
	result, err := Execute(steps, Config{Debug: DebugDetailed})
	require.NoError(t, err)
	require.NotNil(t, result.DebugEvents)

	// Should have step-level events
	events := make(map[string]bool)
	for _, e := range result.DebugEvents {
		events[e.Event] = true
	}
	assert.True(t, events["step_start"])
	assert.True(t, events["step_complete"])
}

// TestInvariantNilPlan tests that nil plan causes panic
func TestInvariantNilPlan(t *testing.T) {
	assert.Panics(t, func() {
		_, _ = Execute(nil, Config{})
	})
}

// TestExecuteRedirectOverwrite tests output redirection with > operator
func TestExecuteRedirectOverwrite(t *testing.T) {
	// Create temp file for testing
	tmpFile := t.TempDir() + "/output.txt"

	plan := &planfmt.Plan{
		Target: "redirect",
		Steps: []planfmt.Step{
			{
				ID: 1,
				Tree: &planfmt.RedirectNode{
					Source: shellCmd("echo 'Hello, World!'"),
					Target: *shellCmd(tmpFile),
					Mode:   planfmt.RedirectOverwrite,
				},
			},
		},
	}

	steps := planfmt.ToSDKSteps(plan.Steps)
	result, err := Execute(steps, Config{})
	require.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
	assert.Equal(t, 1, result.StepsRun)

	// Verify file contents
	content, err := os.ReadFile(tmpFile)
	require.NoError(t, err)
	assert.Equal(t, "Hello, World!\n", string(content))
}

// TestExecuteRedirectAppend tests output redirection with >> operator
func TestExecuteRedirectAppend(t *testing.T) {
	// Create temp file with initial content
	tmpFile := t.TempDir() + "/output.txt"
	err := os.WriteFile(tmpFile, []byte("Line 1\n"), 0o644)
	require.NoError(t, err)

	plan := &planfmt.Plan{
		Target: "redirect",
		Steps: []planfmt.Step{
			{
				ID: 1,
				Tree: &planfmt.RedirectNode{
					Source: shellCmd("echo 'Line 2'"),
					Target: *shellCmd(tmpFile),
					Mode:   planfmt.RedirectAppend,
				},
			},
		},
	}

	steps := planfmt.ToSDKSteps(plan.Steps)
	result, err := Execute(steps, Config{})
	require.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
	assert.Equal(t, 1, result.StepsRun)

	// Verify file contents (should have both lines)
	content, err := os.ReadFile(tmpFile)
	require.NoError(t, err)
	assert.Equal(t, "Line 1\nLine 2\n", string(content))
}

// TestExecuteRedirectWithPipeline tests redirect with pipeline source
func TestExecuteRedirectWithPipeline(t *testing.T) {
	tmpFile := t.TempDir() + "/output.txt"

	plan := &planfmt.Plan{
		Target: "redirect-pipeline",
		Steps: []planfmt.Step{
			{
				ID: 1,
				Tree: &planfmt.RedirectNode{
					Source: &planfmt.PipelineNode{
						Commands: []planfmt.CommandNode{
							{
								Decorator: "@shell",
								Args: []planfmt.Arg{
									{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo 'hello world'"}},
								},
							},
							{
								Decorator: "@shell",
								Args: []planfmt.Arg{
									{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "grep hello"}},
								},
							},
						},
					},
					Target: *shellCmd(tmpFile),
					Mode:   planfmt.RedirectOverwrite,
				},
			},
		},
	}

	steps := planfmt.ToSDKSteps(plan.Steps)
	result, err := Execute(steps, Config{})
	require.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)

	// Verify file contains grep output
	content, err := os.ReadFile(tmpFile)
	require.NoError(t, err)
	assert.Equal(t, "hello world\n", string(content))
}

// TestExecuteRedirectWithAndOperator tests redirect with && operator
func TestExecuteRedirectWithAndOperator(t *testing.T) {
	tmpFile := t.TempDir() + "/output.txt"

	plan := &planfmt.Plan{
		Target: "redirect-and",
		Steps: []planfmt.Step{
			{
				ID: 1,
				Tree: &planfmt.RedirectNode{
					Source: &planfmt.AndNode{
						Left:  shellCmd("echo 'first'"),
						Right: shellCmd("echo 'second'"),
					},
					Target: *shellCmd(tmpFile),
					Mode:   planfmt.RedirectOverwrite,
				},
			},
		},
	}

	steps := planfmt.ToSDKSteps(plan.Steps)
	result, err := Execute(steps, Config{})
	require.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)

	// Verify file contains both commands' output (bash subshell semantics)
	content, err := os.ReadFile(tmpFile)
	require.NoError(t, err)
	assert.Equal(t, "first\nsecond\n", string(content))
}

// TestExecuteRedirectWithOrOperator tests redirect with || operator
func TestExecuteRedirectWithOrOperator(t *testing.T) {
	tmpFile := t.TempDir() + "/output.txt"

	plan := &planfmt.Plan{
		Target: "redirect-or",
		Steps: []planfmt.Step{
			{
				ID: 1,
				Tree: &planfmt.RedirectNode{
					Source: &planfmt.OrNode{
						Left:  shellCmd("exit 1"),          // Fails
						Right: shellCmd("echo 'fallback'"), // Runs
					},
					Target: *shellCmd(tmpFile),
					Mode:   planfmt.RedirectOverwrite,
				},
			},
		},
	}

	steps := planfmt.ToSDKSteps(plan.Steps)
	result, err := Execute(steps, Config{})
	require.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)

	// Verify file contains fallback output
	content, err := os.ReadFile(tmpFile)
	require.NoError(t, err)
	assert.Equal(t, "fallback\n", string(content))
}

// TestExecuteRedirectWithSequence tests redirect with semicolon operator
func TestExecuteRedirectWithSequence(t *testing.T) {
	tmpFile := t.TempDir() + "/output.txt"

	plan := &planfmt.Plan{
		Target: "redirect-sequence",
		Steps: []planfmt.Step{
			{
				ID: 1,
				Tree: &planfmt.RedirectNode{
					Source: &planfmt.SequenceNode{
						Nodes: []planfmt.ExecutionNode{
							shellCmd("echo 'first'"),
							shellCmd("echo 'second'"),
							shellCmd("echo 'third'"),
						},
					},
					Target: *shellCmd(tmpFile),
					Mode:   planfmt.RedirectOverwrite,
				},
			},
		},
	}

	steps := planfmt.ToSDKSteps(plan.Steps)
	result, err := Execute(steps, Config{})
	require.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)

	// Verify file contains all commands' output (bash subshell semantics)
	content, err := os.ReadFile(tmpFile)
	require.NoError(t, err)
	assert.Equal(t, "first\nsecond\nthird\n", string(content))
}

// TestOperatorSemicolon tests semicolon operator (always execute next)
func TestOperatorSemicolon(t *testing.T) {
	tests := []struct {
		name     string
		tree     planfmt.ExecutionNode
		wantExit int
	}{
		{
			name: "all succeed",
			tree: &planfmt.SequenceNode{
				Nodes: []planfmt.ExecutionNode{
					shellCmd("echo 'a'"),
					shellCmd("echo 'b'"),
					shellCmd("echo 'c'"),
				},
			},
			wantExit: 0,
		},
		{
			name: "first fails, rest run",
			tree: &planfmt.SequenceNode{
				Nodes: []planfmt.ExecutionNode{
					shellCmd("exit 1"),
					shellCmd("echo 'still runs'"),
					shellCmd("exit 0"),
				},
			},
			wantExit: 0, // Last command succeeds
		},
		{
			name: "last fails",
			tree: &planfmt.SequenceNode{
				Nodes: []planfmt.ExecutionNode{
					shellCmd("echo 'a'"),
					shellCmd("exit 42"),
				},
			},
			wantExit: 42, // Last command's exit code
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := &planfmt.Plan{
				Target: "test",
				Steps: []planfmt.Step{
					{
						ID:   1,
						Tree: tt.tree,
					},
				},
			}

			steps := planfmt.ToSDKSteps(plan.Steps)
			result, err := Execute(steps, Config{})
			require.NoError(t, err)
			assert.Equal(t, tt.wantExit, result.ExitCode)
			assert.Equal(t, 1, result.StepsRun)
		})
	}
}

// TestOperatorAND tests AND operator (execute next only if previous succeeded)
func TestOperatorAND(t *testing.T) {
	tests := []struct {
		name     string
		tree     planfmt.ExecutionNode
		wantExit int
	}{
		{
			name: "both succeed",
			tree: &planfmt.AndNode{
				Left:  shellCmd("echo 'a'"),
				Right: shellCmd("echo 'b'"),
			},
			wantExit: 0,
		},
		{
			name: "first fails, second skipped",
			tree: &planfmt.AndNode{
				Left:  shellCmd("exit 1"),
				Right: shellCmd("echo 'should not run'"),
			},
			wantExit: 1, // First command's exit code
		},
		{
			name: "chain of ANDs all succeed",
			tree: &planfmt.AndNode{
				Left: &planfmt.AndNode{
					Left:  shellCmd("echo 'a'"),
					Right: shellCmd("echo 'b'"),
				},
				Right: shellCmd("echo 'c'"),
			},
			wantExit: 0,
		},
		{
			name: "chain of ANDs, middle fails",
			tree: &planfmt.AndNode{
				Left: &planfmt.AndNode{
					Left:  shellCmd("echo 'a'"),
					Right: shellCmd("exit 42"),
				},
				Right: shellCmd("echo 'should not run'"),
			},
			wantExit: 42, // Middle command's exit code
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := &planfmt.Plan{
				Target: "test",
				Steps: []planfmt.Step{
					{
						ID:   1,
						Tree: tt.tree,
					},
				},
			}

			steps := planfmt.ToSDKSteps(plan.Steps)
			result, err := Execute(steps, Config{})
			require.NoError(t, err)
			assert.Equal(t, tt.wantExit, result.ExitCode)
			assert.Equal(t, 1, result.StepsRun)
		})
	}
}

// TestOperatorOR tests OR operator (execute next only if previous failed)
func TestOperatorOR(t *testing.T) {
	tests := []struct {
		name     string
		tree     planfmt.ExecutionNode
		wantExit int
	}{
		{
			name: "first succeeds, second skipped",
			tree: &planfmt.OrNode{
				Left:  shellCmd("echo 'success'"),
				Right: shellCmd("echo 'should not run'"),
			},
			wantExit: 0,
		},
		{
			name: "first fails, second runs and succeeds",
			tree: &planfmt.OrNode{
				Left:  shellCmd("exit 1"),
				Right: shellCmd("echo 'fallback'"),
			},
			wantExit: 0, // Fallback succeeds
		},
		{
			name: "first fails, second fails too",
			tree: &planfmt.OrNode{
				Left:  shellCmd("exit 1"),
				Right: shellCmd("exit 2"),
			},
			wantExit: 2, // Second command's exit code
		},
		{
			name: "chain of ORs, first succeeds",
			tree: &planfmt.OrNode{
				Left: &planfmt.OrNode{
					Left:  shellCmd("echo 'success'"),
					Right: shellCmd("echo 'fallback1'"),
				},
				Right: shellCmd("echo 'fallback2'"),
			},
			wantExit: 0, // First succeeds, rest skipped
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := &planfmt.Plan{
				Target: "test",
				Steps: []planfmt.Step{
					{
						ID:   1,
						Tree: tt.tree,
					},
				},
			}

			steps := planfmt.ToSDKSteps(plan.Steps)
			result, err := Execute(steps, Config{})
			require.NoError(t, err)
			assert.Equal(t, tt.wantExit, result.ExitCode)
			assert.Equal(t, 1, result.StepsRun)
		})
	}
}
