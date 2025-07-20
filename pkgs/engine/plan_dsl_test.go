package engine

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/aledsdavies/devcmd/pkgs/ast"
	"github.com/aledsdavies/devcmd/pkgs/decorators"
	"github.com/aledsdavies/devcmd/pkgs/plan"
)

func TestPlanDSL_BasicStructure(t *testing.T) {
	tests := []struct {
		name     string
		elements []plan.PlanElement
		validate func(t *testing.T, ep *plan.ExecutionPlan)
	}{
		{
			name: "simple command plan",
			elements: []plan.PlanElement{
				plan.Command("echo hello").WithDescription("Say hello"),
				plan.Command("ls -la").WithDescription("List files"),
			},
			validate: func(t *testing.T, ep *plan.ExecutionPlan) {
				if len(ep.Steps) != 2 {
					t.Errorf("Expected 2 steps, got %d", len(ep.Steps))
				}

				if ep.Steps[0].Type != plan.StepShell {
					t.Errorf("Expected first step to be shell, got %s", ep.Steps[0].Type)
				}

				if ep.Steps[0].Command != "echo hello" {
					t.Errorf("Expected first command to be 'echo hello', got %s", ep.Steps[0].Command)
				}

				if ep.Summary.ShellCommands != 2 {
					t.Errorf("Expected 2 shell commands in summary, got %d", ep.Summary.ShellCommands)
				}
			},
		},
		{
			name: "decorator with parameters",
			elements: []plan.PlanElement{
				plan.Decorator("timeout").
					WithType("block").
					WithTimeout(30*time.Second).
					WithParameter("duration", "30s").
					WithDescription("Execute with timeout"),
			},
			validate: func(t *testing.T, ep *plan.ExecutionPlan) {
				if len(ep.Steps) != 1 {
					t.Errorf("Expected 1 step, got %d", len(ep.Steps))
				}

				step := ep.Steps[0]
				if step.Type != plan.StepTimeout {
					t.Errorf("Expected timeout step, got %s", step.Type)
				}

				if step.Decorator == nil {
					t.Fatal("Expected decorator info to be set")
				}

				if step.Decorator.Name != "timeout" {
					t.Errorf("Expected decorator name 'timeout', got %s", step.Decorator.Name)
				}

				if step.Decorator.Type != "block" {
					t.Errorf("Expected decorator type 'block', got %s", step.Decorator.Type)
				}

				duration, exists := step.Decorator.Parameters["duration"]
				if !exists {
					t.Error("Expected duration parameter to exist")
				}
				if duration != "30s" {
					t.Errorf("Expected duration '30s', got %v", duration)
				}

				if step.Timing == nil {
					t.Fatal("Expected timing info to be set")
				}

				if step.Timing.Timeout == nil || *step.Timing.Timeout != 30*time.Second {
					t.Errorf("Expected timeout of 30s, got %v", step.Timing.Timeout)
				}

				if len(ep.Summary.DecoratorsUsed) != 1 || ep.Summary.DecoratorsUsed[0] != "timeout" {
					t.Errorf("Expected decorators used to contain 'timeout', got %v", ep.Summary.DecoratorsUsed)
				}
			},
		},
		{
			name: "nested decorator with children",
			elements: []plan.PlanElement{
				plan.Decorator("parallel").
					WithType("block").
					WithConcurrency(3).
					WithParameter("concurrency", 3).
					WithDescription("Execute in parallel").
					WithChildren(
						plan.Command("npm run build").WithDescription("Build frontend"),
						plan.Command("npm run test").WithDescription("Run tests"),
					),
			},
			validate: func(t *testing.T, ep *plan.ExecutionPlan) {
				if len(ep.Steps) != 1 {
					t.Errorf("Expected 1 step, got %d", len(ep.Steps))
				}

				step := ep.Steps[0]
				if step.Type != plan.StepParallel {
					t.Errorf("Expected parallel step, got %s", step.Type)
				}

				if len(step.Children) != 2 {
					t.Errorf("Expected 2 children, got %d", len(step.Children))
				}

				// Validate first child
				child1 := step.Children[0]
				if child1.Type != plan.StepShell {
					t.Errorf("Expected first child to be shell, got %s", child1.Type)
				}
				if child1.Command != "npm run build" {
					t.Errorf("Expected first child command 'npm run build', got %s", child1.Command)
				}

				// Validate second child
				child2 := step.Children[1]
				if child2.Type != plan.StepShell {
					t.Errorf("Expected second child to be shell, got %s", child2.Type)
				}
				if child2.Command != "npm run test" {
					t.Errorf("Expected second child command 'npm run test', got %s", child2.Command)
				}

				// Validate timing
				if step.Timing == nil {
					t.Fatal("Expected timing info to be set")
				}
				if step.Timing.ConcurrencyLimit != 3 {
					t.Errorf("Expected concurrency limit 3, got %d", step.Timing.ConcurrencyLimit)
				}

				// Validate summary
				if ep.Summary.ParallelSections != 1 {
					t.Errorf("Expected 1 parallel section, got %d", ep.Summary.ParallelSections)
				}
				if ep.Summary.ShellCommands != 2 {
					t.Errorf("Expected 2 shell commands, got %d", ep.Summary.ShellCommands)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			planBuilder := plan.NewPlan()
			for _, element := range tt.elements {
				planBuilder.Add(element)
			}

			executionPlan := planBuilder.Build()
			tt.validate(t, executionPlan)
		})
	}
}

func TestPlanDSL_TreeVisualization(t *testing.T) {
	// Build a complex nested plan
	executionPlan := plan.NewPlan().
		Add(plan.Command("echo 'Starting'").WithDescription("Initialize")).
		Add(
			plan.Decorator("timeout").
				WithType("block").
				WithTimeout(5*time.Minute).
				WithParameter("duration", "5m").
				WithDescription("Deploy with timeout").
				WithChildren(
					plan.Decorator("parallel").
						WithType("block").
						WithConcurrency(2).
						WithParameter("concurrency", 2).
						WithDescription("Build in parallel").
						WithChildren(
							plan.Command("npm run build").WithDescription("Build frontend"),
							plan.Command("npm run test").WithDescription("Run tests"),
						),
					plan.Command("docker build").WithDescription("Build container"),
				),
		).
		Add(plan.Command("echo 'Done'").WithDescription("Finalize")).
		Build()

	// Test the string representation contains proper structure
	output := executionPlan.String()

	// Should contain step information
	if !containsStr(output, "[shell] Initialize: echo 'Starting'") {
		t.Error("Expected first shell command in output")
	}

	if !containsStr(output, "[timeout] Deploy with timeout") {
		t.Error("Expected timeout decorator in output")
	}

	if !containsStr(output, "🔧 @timeout(duration=5m)") {
		t.Error("Expected timeout decorator info with emoji")
	}

	if !containsStr(output, "⏱️  timeout=5m0s") {
		t.Error("Expected timing information with emoji")
	}

	if !containsStr(output, "[parallel] Build in parallel") {
		t.Error("Expected parallel decorator in output")
	}

	if !containsStr(output, "[shell] Build frontend: npm run build") {
		t.Error("Expected nested build command")
	}

	if !containsStr(output, "[shell] Run tests: npm run test") {
		t.Error("Expected nested test command")
	}

	// Test summary information (recursive counting includes all nested steps)
	if executionPlan.Summary.TotalSteps != 7 {
		t.Errorf("Expected 7 total steps, got %d", executionPlan.Summary.TotalSteps)
	}

	if executionPlan.Summary.ShellCommands != 5 {
		t.Errorf("Expected 5 shell commands, got %d", executionPlan.Summary.ShellCommands)
	}

	if executionPlan.Summary.ParallelSections != 1 {
		t.Errorf("Expected 1 parallel section, got %d", executionPlan.Summary.ParallelSections)
	}

	expectedDecorators := []string{"timeout", "parallel"}
	if len(executionPlan.Summary.DecoratorsUsed) != len(expectedDecorators) {
		t.Errorf("Expected %d decorators, got %d", len(expectedDecorators), len(executionPlan.Summary.DecoratorsUsed))
	}
}

func TestPlanDSL_DecoratorIntegration(t *testing.T) {
	// Test that decorators properly create plan elements
	program := &ast.Program{
		Variables: []ast.VariableDecl{
			{Name: "USER", Value: &ast.StringLiteral{Value: "admin"}},
		},
	}

	ctx := decorators.NewExecutionContext(context.Background(), program)
	ctx.SetVariable("USER", "admin")

	tests := []struct {
		name            string
		decoratorName   string
		decoratorType   string
		params          []ast.NamedParameter
		content         []ast.CommandContent
		validateElement func(t *testing.T, element plan.PlanElement)
	}{
		{
			name:          "var decorator plan",
			decoratorName: "var",
			decoratorType: "function",
			params: []ast.NamedParameter{
				{Value: &ast.Identifier{Name: "USER"}},
			},
			validateElement: func(t *testing.T, element plan.PlanElement) {
				step := element.Build()

				if step.Decorator == nil {
					t.Fatal("Expected decorator info")
				}

				if step.Decorator.Name != "var" {
					t.Errorf("Expected decorator name 'var', got %s", step.Decorator.Name)
				}

				if step.Decorator.Type != "function" {
					t.Errorf("Expected decorator type 'function', got %s", step.Decorator.Type)
				}

				name, exists := step.Decorator.Parameters["name"]
				if !exists || name != "USER" {
					t.Errorf("Expected name parameter 'USER', got %v", name)
				}

				if !containsStr(step.Description, "Variable resolution: ${USER}") {
					t.Errorf("Expected variable resolution in description, got %s", step.Description)
				}
			},
		},
		{
			name:          "timeout decorator plan",
			decoratorName: "timeout",
			decoratorType: "block",
			params: []ast.NamedParameter{
				{Name: "duration", Value: &ast.DurationLiteral{Value: "30s"}},
			},
			content: []ast.CommandContent{
				&ast.ShellContent{Parts: []ast.ShellPart{&ast.TextPart{Text: "echo test"}}},
			},
			validateElement: func(t *testing.T, element plan.PlanElement) {
				step := element.Build()

				if step.Type != plan.StepTimeout {
					t.Errorf("Expected timeout step type, got %s", step.Type)
				}

				if step.Decorator == nil {
					t.Fatal("Expected decorator info")
				}

				if step.Decorator.Name != "timeout" {
					t.Errorf("Expected decorator name 'timeout', got %s", step.Decorator.Name)
				}

				duration, exists := step.Decorator.Parameters["duration"]
				if !exists || duration != "30s" {
					t.Errorf("Expected duration parameter '30s', got %v", duration)
				}

				if step.Timing == nil {
					t.Fatal("Expected timing info")
				}

				if step.Timing.Timeout == nil || *step.Timing.Timeout != 30*time.Second {
					t.Errorf("Expected timeout of 30s, got %v", step.Timing.Timeout)
				}

				if !containsStr(step.Description, "Execute 1 commands with 30s timeout") {
					t.Errorf("Expected timeout description, got %s", step.Description)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var element plan.PlanElement
			var err error

			switch tt.decoratorType {
			case "function":
				decorator, getErr := decorators.GetFunction(tt.decoratorName)
				if getErr != nil {
					t.Fatalf("Failed to get function decorator: %v", getErr)
				}
				element, err = decorator.Plan(ctx, tt.params)
			case "block":
				decorator, getErr := decorators.GetBlock(tt.decoratorName)
				if getErr != nil {
					t.Fatalf("Failed to get block decorator: %v", getErr)
				}
				element, err = decorator.Plan(ctx, tt.params, tt.content)
			}

			if err != nil {
				t.Fatalf("Failed to create plan element: %v", err)
			}

			tt.validateElement(t, element)
		})
	}
}

func TestPlanDSL_ConditionalElements(t *testing.T) {
	// Test conditional plan elements
	conditional := plan.Conditional("NODE_ENV", "production", "production").
		WithReason("Environment matches production").
		AddBranch("production", "Production deployment", true).
		AddBranch("staging", "Staging deployment", false).
		AddBranch("development", "Development deployment", false).
		WithChildren(
			plan.Command("kubectl apply -f prod.yaml").WithDescription("Deploy to production"),
		)

	step := conditional.Build()

	if step.Type != plan.StepConditional {
		t.Errorf("Expected conditional step type, got %s", step.Type)
	}

	if step.Condition == nil {
		t.Fatal("Expected condition info")
	}

	if step.Condition.Variable != "NODE_ENV" {
		t.Errorf("Expected variable 'NODE_ENV', got %s", step.Condition.Variable)
	}

	if step.Condition.Evaluation.CurrentValue != "production" {
		t.Errorf("Expected current value 'production', got %s", step.Condition.Evaluation.CurrentValue)
	}

	if step.Condition.Evaluation.SelectedBranch != "production" {
		t.Errorf("Expected selected branch 'production', got %s", step.Condition.Evaluation.SelectedBranch)
	}

	if len(step.Condition.Branches) != 3 {
		t.Errorf("Expected 3 branches, got %d", len(step.Condition.Branches))
	}

	// Check that the production branch is marked as will execute
	prodBranch := step.Condition.Branches[0]
	if prodBranch.Pattern != "production" || !prodBranch.WillExecute {
		t.Errorf("Expected production branch to be marked as will execute")
	}

	if len(step.Children) != 1 {
		t.Errorf("Expected 1 child, got %d", len(step.Children))
	}
}

// Helper function to check if a string contains a substring
func containsStr(s, substr string) bool {
	return strings.Contains(s, substr)
}
