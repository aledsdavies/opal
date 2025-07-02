package parser

import "testing"



func TestDecoratorVariations(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "decorator with no arguments",
			Input: "sync: @parallel { task1; task2 }",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("sync", []ExpectedDecorator{
						{Name: "parallel", Args: []ExpectedExpression{}},
					}, BlockCommandBody(
						Statement(TextElement("task1")),
						Statement(TextElement("task2")))),
				},
			},
		},
		{
			Name:  "decorator with single string argument",
			Input: "ask: @confirm(\"Are you sure?\") { rm -rf /tmp/* }",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("ask", []ExpectedDecorator{
						{Name: "confirm", Args: []ExpectedExpression{
							StringExpr("Are you sure?"),
						}},
					}, BlockCommandBody(
						Statement(TextElement("rm -rf /tmp/*")))),
				},
			},
		},
		{
			Name:  "decorator with duration argument",
			Input: "slow: @timeout(5m) { sleep 300 }",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("slow", []ExpectedDecorator{
						{Name: "timeout", Args: []ExpectedExpression{
							DurationExpr("5m"),
						}},
					}, BlockCommandBody(
						Statement(TextElement("sleep 300")))),
				},
			},
		},
		{
			Name:  "decorator with number argument",
			Input: "retry-task: @retry(3) { flaky-command }",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("retry-task", []ExpectedDecorator{
						{Name: "retry", Args: []ExpectedExpression{
							NumberExpr("3"),
						}},
					}, BlockCommandBody(
						Statement(TextElement("flaky-command")))),
				},
			},
		},
		{
			Name:  "decorator with multiple arguments",
			Input: "watch-files: @debounce(500ms, \"src/**/*\") { npm run build }",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("watch-files", []ExpectedDecorator{
						{Name: "debounce", Args: []ExpectedExpression{
							DurationExpr("500ms"),
							StringExpr("src/**/*"),
						}},
					}, BlockCommandBody(
						Statement(TextElement("npm run build")))),
				},
			},
		},
		{
			Name:  "multiple decorators on one command",
			Input: "complex: @timeout(30s) @retry(3) @env(NODE_ENV=test) { npm test }",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("complex", []ExpectedDecorator{
						{Name: "timeout", Args: []ExpectedExpression{DurationExpr("30s")}},
						{Name: "retry", Args: []ExpectedExpression{NumberExpr("3")}},
						{Name: "env", Args: []ExpectedExpression{IdentifierExpr("NODE_ENV=test")}},
					}, BlockCommandBody(
						Statement(TextElement("npm test")))),
				},
			},
		},
		{
			Name:  "decorator with @var argument",
			Input: "deploy: @cwd(@var(BUILD_DIR)) { make install }",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("deploy", []ExpectedDecorator{
						{Name: "cwd", Args: []ExpectedExpression{
							VarRefExpr("BUILD_DIR"),
						}},
					}, BlockCommandBody(
						Statement(TextElement("make install")))),
				},
			},
		},
		{
			Name:  "decorator with mixed argument types",
			Input: "advanced: @watch-files(@var(PATTERN), 1s, true) { rebuild }",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("advanced", []ExpectedDecorator{
						{Name: "watch-files", Args: []ExpectedExpression{
							VarRefExpr("PATTERN"),
							DurationExpr("1s"),
							IdentifierExpr("true"),
						}},
					}, BlockCommandBody(
						Statement(TextElement("rebuild")))),
				},
			},
		},
		{
			Name:  "decorator on simple command",
			Input: "quick: @sh(echo hello && sleep 1)",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("quick", []ExpectedDecorator{
						{Name: "sh", Args: []ExpectedExpression{
							StringExpr("echo hello && sleep 1"),
						}},
					}, BlockCommandBody()),
				},
			},
		},
		{
			Name:  "nested decorators in block",
			Input: "nested: { @timeout(10s) { @sh(long-task); echo done } }",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("nested", nil, BlockCommandBody(
						Statement(DecoratorElement("timeout", DurationExpr("10s"))))),
				},
			},
		},
		{
			Name:  "decorator with string containing @ symbol",
			Input: "email: @sh(echo \"Contact us @ support@company.com\")",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("email", []ExpectedDecorator{
						{Name: "sh", Args: []ExpectedExpression{
							StringExpr("echo \"Contact us @ support@company.com\""),
						}},
					}, BlockCommandBody()),
				},
			},
		},
		{
			Name:  "decorator with boolean argument",
			Input: "deploy: @confirm(true) { ./deploy.sh }",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("deploy", []ExpectedDecorator{
						{Name: "confirm", Args: []ExpectedExpression{
							IdentifierExpr("true"),
						}},
					}, BlockCommandBody(
						Statement(TextElement("./deploy.sh")))),
				},
			},
		},
		{
			Name:  "decorator with negative number",
			Input: "adjust: @offset(-5) { process }",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("adjust", []ExpectedDecorator{
						{Name: "offset", Args: []ExpectedExpression{
							NumberExpr("-5"),
						}},
					}, BlockCommandBody(
						Statement(TextElement("process")))),
				},
			},
		},
		{
			Name:  "decorator with decimal number",
			Input: "scale: @factor(1.5) { scale-service }",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("scale", []ExpectedDecorator{
						{Name: "factor", Args: []ExpectedExpression{
							NumberExpr("1.5"),
						}},
					}, BlockCommandBody(
						Statement(TextElement("scale-service")))),
				},
			},
		},
		{
			Name:  "multiple decorators with different argument types",
			Input: "complex: @timeout(30s) @retry(3) @env(\"NODE_ENV=test\") @confirm(false) { test-suite }",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("complex", []ExpectedDecorator{
						{Name: "timeout", Args: []ExpectedExpression{DurationExpr("30s")}},
						{Name: "retry", Args: []ExpectedExpression{NumberExpr("3")}},
						{Name: "env", Args: []ExpectedExpression{StringExpr("NODE_ENV=test")}},
						{Name: "confirm", Args: []ExpectedExpression{IdentifierExpr("false")}},
					}, BlockCommandBody(
						Statement(TextElement("test-suite")))),
				},
			},
		},
		{
			Name:  "decorator with no arguments but parentheses",
			Input: "test: @parallel() { task1; task2 }",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("test", []ExpectedDecorator{
						{Name: "parallel", Args: []ExpectedExpression{}},
					}, BlockCommandBody(
						Statement(TextElement("task1")),
						Statement(TextElement("task2")))),
				},
			},
		},
	}

	for _, tc := range testCases {
		runTestCase(t, tc)
	}
}

