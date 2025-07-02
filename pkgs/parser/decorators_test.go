package parser

import (
	"testing"
)

func TestVarDecorators(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "simple @var() reference",
			Input: "build: cd @var(SRC)",
			Expected: Program(
				Cmd("build", Simple(Text("cd "), At("var", "SRC"))),
			),
		},
		{
			Name:  "multiple @var() references",
			Input: "deploy: docker build -t @var(IMAGE):@var(TAG)",
			Expected: Program(
				Cmd("deploy", Simple(
					Text("docker build -t "),
					At("var", "IMAGE"),
					Text(":"),
					At("var", "TAG"),
				)),
			),
		},
		{
			Name:  "@var() in quoted string",
			Input: "echo: echo \"Building @var(PROJECT) version @var(VERSION)\"",
			Expected: Program(
				Cmd("echo", Simple(
					Text("echo \"Building "),
					At("var", "PROJECT"),
					Text(" version "),
					At("var", "VERSION"),
					Text("\""),
				)),
			),
		},
		{
			Name:  "mixed @var() and shell variables",
			Input: "info: echo \"Project: @var(NAME), User: $USER\"",
			Expected: Program(
				Cmd("info", Simple(
					Text("echo \"Project: "),
					At("var", "NAME"),
					Text(", User: $USER\""),
				)),
			),
		},
		{
			Name:  "@var() in file paths",
			Input: "copy: cp @var(SRC)/*.go @var(DEST)/",
			Expected: Program(
				Cmd("copy", Simple(
					Text("cp "),
					At("var", "SRC"),
					Text("/*.go "),
					At("var", "DEST"),
					Text("/"),
				)),
			),
		},
		{
			Name:  "@var() in command arguments",
			Input: "serve: go run main.go --port=@var(PORT) --host=@var(HOST)",
			Expected: Program(
				Cmd("serve", Simple(
					Text("go run main.go --port="),
					At("var", "PORT"),
					Text(" --host="),
					At("var", "HOST"),
				)),
			),
		},
		{
			Name:  "@var() with special characters in value",
			Input: "url: curl \"@var(API_URL)/users?filter=@var(FILTER)\"",
			Expected: Program(
				Cmd("url", Simple(
					Text("curl \""),
					At("var", "API_URL"),
					Text("/users?filter="),
					At("var", "FILTER"),
					Text("\""),
				)),
			),
		},
		{
			Name:  "@var() in conditional expressions",
			Input: "check: test \"@var(ENV)\" = \"production\" && echo prod || echo dev",
			Expected: Program(
				Cmd("check", Simple(
					Text("test \""),
					At("var", "ENV"),
					Text("\" = \"production\" && echo prod || echo dev"),
				)),
			),
		},
		{
			Name:  "@var() in loops",
			Input: "process: for file in @var(SRC)/*.txt; do process $file; done",
			Expected: Program(
				Cmd("process", Simple(
					Text("for file in "),
					At("var", "SRC"),
					Text("/*.txt; do process $file; done"),
				)),
			),
		},
		{
			Name:  "string with escaped quotes and @var",
			Input: "msg: echo \"He said \\\"Hello @var(NAME)\\\" to everyone\"",
			Expected: Program(
				Cmd("msg", Simple(
					Text("echo \"He said \\\"Hello "),
					At("var", "NAME"),
					Text("\\\" to everyone\""),
				)),
			),
		},
	}

	for _, tc := range testCases {
		RunTestCase(t, tc)
	}
}

func TestDecoratorsWithArguments(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "valid @sh function decorator",
			Input: "test: @sh(echo hello)",
			Expected: Program(
				CmdWith(At("sh", "echo hello"), "test", Block()),
			),
		},
		{
			Name:  "valid @timeout function decorator",
			Input: "deploy: @timeout(30s) { echo deploying }",
			Expected: Program(
				CmdWith(At("timeout", "30s"), "deploy", Block(
					Text("echo deploying"),
				)),
			),
		},
		{
			Name:  "valid @env decorator with argument",
			Input: "setup: @env(NODE_ENV=production) { npm start }",
			Expected: Program(
				CmdWith(At("env", "NODE_ENV=production"), "setup", Block(
					Text("npm start"),
				)),
			),
		},
		{
			Name:  "valid @confirm decorator",
			Input: "dangerous: @confirm(\"Are you sure?\") { rm -rf /tmp/* }",
			Expected: Program(
				CmdWith(At("confirm", "Are you sure?"), "dangerous", Block(
					Text("rm -rf /tmp/*"),
				)),
			),
		},
		{
			Name:  "valid @debounce decorator",
			Input: "watch-changes: @debounce(500ms) { npm run build }",
			Expected: Program(
				CmdWith(At("debounce", "500ms"), "watch-changes", Block(
					Text("npm run build"),
				)),
			),
		},
		{
			Name:  "valid @cwd decorator",
			Input: "build-lib: @cwd(./lib) { make all }",
			Expected: Program(
				CmdWith(At("cwd", "./lib"), "build-lib", Block(
					Text("make all"),
				)),
			),
		},
	}

	for _, tc := range testCases {
		RunTestCase(t, tc)
	}
}

func TestDecoratorsWithBlocks(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "valid @parallel block decorator",
			Input: "services: @parallel { server; client }",
			Expected: Program(
				CmdWith(At("parallel"), "services", Block(
					Text("server"),
					Text("client"),
				)),
			),
		},
		{
			Name:  "valid @retry block decorator",
			Input: "flaky-test: @retry { npm test; echo 'done' }",
			Expected: Program(
				CmdWith(At("retry"), "flaky-test", Block(
					Text("npm test"),
					Text("echo 'done'"),
				)),
			),
		},
		{
			Name:  "valid @watch-files block decorator",
			Input: "monitor: @watch-files { echo 'checking'; sleep 1 }",
			Expected: Program(
				CmdWith(At("watch-files"), "monitor", Block(
					Text("echo 'checking'"),
					Text("sleep 1"),
				)),
			),
		},
		{
			Name:  "empty block with decorators",
			Input: "parallel-empty: @parallel { }",
			Expected: Program(
				CmdWith(At("parallel"), "parallel-empty", Block()),
			),
		},
	}

	for _, tc := range testCases {
		RunTestCase(t, tc)
	}
}

func TestNestedDecorators(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "@sh() with @var()",
			Input: "build: @sh(cd @var(SRC))",
			Expected: Program(
				CmdWith(At("sh", "cd @var(SRC)"), "build", Block()),
			),
		},
		{
			Name:  "@timeout with @sh nested",
			Input: "deploy: @timeout(30s) { @sh(deploy.sh @var(ENV)) }",
			Expected: Program(
				CmdWith(At("timeout", "30s"), "deploy", Block(
					At("sh", "deploy.sh @var(ENV)"),
				)),
			),
		},
		{
			Name:  "@parallel with @var() in statements",
			Input: "multi: @parallel { echo @var(MSG1); echo @var(MSG2) }",
			Expected: Program(
				CmdWith(At("parallel"), "multi", Block(
					Simple(Text("echo "), At("var", "MSG1")),
					Simple(Text("echo "), At("var", "MSG2")),
				)),
			),
		},
		{
			Name:  "@env with @var() values",
			Input: "setup: @env(PATH=@var(CUSTOM_PATH)) { which custom-tool }",
			Expected: Program(
				CmdWith(At("env", "PATH=@var(CUSTOM_PATH)"), "setup", Block(
					Text("which custom-tool"),
				)),
			),
		},
		{
			Name:  "multiple decorators with @var",
			Input: "complex: @timeout(30s) @env(NODE_ENV=@var(ENV)) { npm start }",
			Expected: Program(
				CmdWith([]interface{}{
					At("timeout", "30s"),
					At("env", "NODE_ENV=@var(ENV)"),
				}, "complex", Block(
					Text("npm start"),
				)),
			),
		},
		{
			Name:  "@cwd with @var path and @sh command",
			Input: "build: @cwd(@var(BUILD_DIR)) { @sh(make clean && make all) }",
			Expected: Program(
				CmdWith(At("cwd", At("var", "BUILD_DIR")), "build", Block(
					At("sh", "make clean && make all"),
				)),
			),
		},
	}

	for _, tc := range testCases {
		RunTestCase(t, tc)
	}
}

func TestDecoratorVariations(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "decorator with no arguments",
			Input: "sync: @parallel { task1; task2 }",
			Expected: Program(
				CmdWith(At("parallel"), "sync", Block(
					Text("task1"),
					Text("task2"),
				)),
			),
		},
		{
			Name:  "decorator with single string argument",
			Input: "ask: @confirm(\"Are you sure?\") { rm -rf /tmp/* }",
			Expected: Program(
				CmdWith(At("confirm", "Are you sure?"), "ask", Block(
					Text("rm -rf /tmp/*"),
				)),
			),
		},
		{
			Name:  "decorator with duration argument",
			Input: "slow: @timeout(5m) { sleep 300 }",
			Expected: Program(
				CmdWith(At("timeout", "5m"), "slow", Block(
					Text("sleep 300"),
				)),
			),
		},
		{
			Name:  "decorator with number argument",
			Input: "retry-task: @retry(3) { flaky-command }",
			Expected: Program(
				CmdWith(At("retry", "3"), "retry-task", Block(
					Text("flaky-command"),
				)),
			),
		},
		{
			Name:  "decorator with multiple arguments",
			Input: "watch-files: @debounce(500ms, \"src/**/*\") { npm run build }",
			Expected: Program(
				CmdWith(At("debounce", "500ms", "src/**/*"), "watch-files", Block(
					Text("npm run build"),
				)),
			),
		},
		{
			Name:  "multiple decorators on one command",
			Input: "complex: @timeout(30s) @retry(3) @env(NODE_ENV=test) { npm test }",
			Expected: Program(
				CmdWith([]interface{}{
					At("timeout", "30s"),
					At("retry", "3"),
					At("env", "NODE_ENV=test"),
				}, "complex", Block(
					Text("npm test"),
				)),
			),
		},
		{
			Name:  "decorator with @var argument",
			Input: "deploy: @cwd(@var(BUILD_DIR)) { make install }",
			Expected: Program(
				CmdWith(At("cwd", At("var", "BUILD_DIR")), "deploy", Block(
					Text("make install"),
				)),
			),
		},
		{
			Name:  "decorator with mixed argument types",
			Input: "advanced: @watch-files(@var(PATTERN), 1s, true) { rebuild }",
			Expected: Program(
				CmdWith(At("watch-files", At("var", "PATTERN"), "1s", "true"), "advanced", Block(
					Text("rebuild"),
				)),
			),
		},
		{
			Name:  "decorator on simple command",
			Input: "quick: @sh(echo hello && sleep 1)",
			Expected: Program(
				CmdWith(At("sh", "echo hello && sleep 1"), "quick", Block()),
			),
		},
		{
			Name:  "nested decorators in block",
			Input: "nested: { @timeout(10s) { @sh(long-task); echo done } }",
			Expected: Program(
				Cmd("nested", Block(
					At("timeout", "10s"),
				)),
			),
		},
		{
			Name:  "decorator with string containing @ symbol",
			Input: "email: @sh(echo \"Contact us @ support@company.com\")",
			Expected: Program(
				CmdWith(At("sh", "echo \"Contact us @ support@company.com\""), "email", Block()),
			),
		},
		{
			Name:  "decorator with boolean argument",
			Input: "deploy: @confirm(true) { ./deploy.sh }",
			Expected: Program(
				CmdWith(At("confirm", "true"), "deploy", Block(
					Text("./deploy.sh"),
				)),
			),
		},
		{
			Name:  "decorator with negative number",
			Input: "adjust: @offset(-5) { process }",
			Expected: Program(
				CmdWith(At("offset", "-5"), "adjust", Block(
					Text("process"),
				)),
			),
		},
		{
			Name:  "decorator with decimal number",
			Input: "scale: @factor(1.5) { scale-service }",
			Expected: Program(
				CmdWith(At("factor", "1.5"), "scale", Block(
					Text("scale-service"),
				)),
			),
		},
		{
			Name:  "multiple decorators with different argument types",
			Input: "complex: @timeout(30s) @retry(3) @env(\"NODE_ENV=test\") @confirm(false) { test-suite }",
			Expected: Program(
				CmdWith([]interface{}{
					At("timeout", "30s"),
					At("retry", "3"),
					At("env", "NODE_ENV=test"),
					At("confirm", "false"),
				}, "complex", Block(
					Text("test-suite"),
				)),
			),
		},
		{
			Name:  "decorator with no arguments but parentheses",
			Input: "test: @parallel() { task1; task2 }",
			Expected: Program(
				CmdWith(At("parallel"), "test", Block(
					Text("task1"),
					Text("task2"),
				)),
			),
		},
	}

	for _, tc := range testCases {
		RunTestCase(t, tc)
	}
}
