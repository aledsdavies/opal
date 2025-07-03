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

func TestBlockDecorators(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "valid @timeout block decorator",
			Input: "deploy: @timeout(30s) { echo deploying }",
			Expected: Program(
				CmdWith(Decorator("timeout", "30s"), "deploy", Simple(
					Text("echo deploying"),
				)),
			),
		},
		{
			Name:  "valid @env decorator with argument",
			Input: "setup: @env(NODE_ENV=production) { npm start }",
			Expected: Program(
				CmdWith(Decorator("env", "NODE_ENV=production"), "setup", Simple(
					Text("npm start"),
				)),
			),
		},
		{
			Name:  "valid @confirm decorator",
			Input: "dangerous: @confirm(\"Are you sure?\") { rm -rf /tmp/* }",
			Expected: Program(
				CmdWith(Decorator("confirm", "Are you sure?"), "dangerous", Simple(
					Text("rm -rf /tmp/*"),
				)),
			),
		},
		{
			Name:  "valid @debounce decorator",
			Input: "watch-changes: @debounce(500ms) { npm run build }",
			Expected: Program(
				CmdWith(Decorator("debounce", "500ms"), "watch-changes", Simple(
					Text("npm run build"),
				)),
			),
		},
		{
			Name:  "valid @cwd decorator",
			Input: "build-lib: @cwd(./lib) { make all }",
			Expected: Program(
				CmdWith(Decorator("cwd", "./lib"), "build-lib", Simple(
					Text("make all"),
				)),
			),
		},
		{
			Name:  "valid @parallel block decorator with multiple statements",
			Input: "services: @parallel { server; client }",
			Expected: Program(
				CmdWith(Decorator("parallel"), "services", Block(
					Text("server"),
					Text("client"),
				)),
			),
		},
		{
			Name:  "valid @retry block decorator with multiple statements",
			Input: "flaky-test: @retry { npm test; echo 'done' }",
			Expected: Program(
				CmdWith(Decorator("retry"), "flaky-test", Block(
					Text("npm test"),
					Text("echo 'done'"),
				)),
			),
		},
		{
			Name:  "valid @watch-files block decorator with multiple statements",
			Input: "monitor: @watch-files { echo 'checking'; sleep 1 }",
			Expected: Program(
				CmdWith(Decorator("watch-files"), "monitor", Block(
					Text("echo 'checking'"),
					Text("sleep 1"),
				)),
			),
		},
		{
			Name:  "empty block with decorators",
			Input: "parallel-empty: @parallel { }",
			Expected: Program(
				CmdWith(Decorator("parallel"), "parallel-empty", Simple()),
			),
		},
	}

	for _, tc := range testCases {
		RunTestCase(t, tc)
	}
}

func TestFunctionDecorators(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "function decorator @sh() within shell content",
			Input: "build: echo start && @sh(cd @var(SRC)) && echo done",
			Expected: Program(
				Cmd("build", Simple(
					Text("echo start && "),
					At("sh", "cd @var(SRC)"),
					Text(" && echo done"),
				)),
			),
		},
		{
			Name:  "function decorator @sh() with complex command",
			Input: "deploy: @sh(deploy.sh @var(ENV) && echo success || echo failed)",
			Expected: Program(
				Cmd("deploy", Simple(
					At("sh", "deploy.sh @var(ENV) && echo success || echo failed"),
				)),
			),
		},
		{
			Name:  "mixed function decorators and text",
			Input: "info: echo \"Date: @sh(date)\" and \"User: @sh(whoami)\"",
			Expected: Program(
				Cmd("info", Simple(
					Text("echo \"Date: "),
					At("sh", "date"),
					Text("\" and \"User: "),
					At("sh", "whoami"),
					Text("\""),
				)),
			),
		},
		{
			Name:  "function decorator with @var argument",
			Input: "test: @sh(test -f @var(CONFIG_FILE) && echo exists || echo missing)",
			Expected: Program(
				Cmd("test", Simple(
					At("sh", "test -f @var(CONFIG_FILE) && echo exists || echo missing"),
				)),
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
			Name:  "block decorator with function decorator inside",
			Input: "deploy: @timeout(30s) { @sh(deploy.sh @var(ENV)) }",
			Expected: Program(
				CmdWith(Decorator("timeout", "30s"), "deploy", Simple(
					At("sh", "deploy.sh @var(ENV)"),
				)),
			),
		},
		{
			Name:  "parallel with mixed content",
			Input: "multi: @parallel { echo @var(MSG1); echo @var(MSG2) }",
			Expected: Program(
				CmdWith(Decorator("parallel"), "multi", Block(
					Text("echo "),
					At("var", "MSG1"),
					Text("echo "),
					At("var", "MSG2"),
				)),
			),
		},
		{
			Name:  "decorator with @var in argument",
			Input: "setup: @env(PATH=@var(CUSTOM_PATH)) { which custom-tool }",
			Expected: Program(
				CmdWith(Decorator("env", "PATH=@var(CUSTOM_PATH)"), "setup", Simple(
					Text("which custom-tool"),
				)),
			),
		},
		{
			Name:  "multiple decorators",
			Input: "complex: @timeout(30s) @env(NODE_ENV=@var(ENV)) { npm start }",
			Expected: Program(
				CmdWith([]ExpectedDecorator{
					Decorator("timeout", "30s"),
					Decorator("env", "NODE_ENV=@var(ENV)"),
				}, "complex", Simple(
					Text("npm start"),
				)),
			),
		},
		{
			Name:  "decorator with @var as argument",
			Input: "build: @cwd(@var(BUILD_DIR)) { make clean && make all }",
			Expected: Program(
				CmdWith(Decorator("cwd", At("var", "BUILD_DIR")), "build", Simple(
					Text("make clean && make all"),
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
				CmdWith(Decorator("parallel"), "sync", Block(
					Text("task1"),
					Text("task2"),
				)),
			),
		},
		{
			Name:  "decorator with single string argument",
			Input: "ask: @confirm(\"Are you sure?\") { rm -rf /tmp/* }",
			Expected: Program(
				CmdWith(Decorator("confirm", "Are you sure?"), "ask", Simple(
					Text("rm -rf /tmp/*"),
				)),
			),
		},
		{
			Name:  "decorator with duration argument",
			Input: "slow: @timeout(5m) { sleep 300 }",
			Expected: Program(
				CmdWith(Decorator("timeout", "5m"), "slow", Simple(
					Text("sleep 300"),
				)),
			),
		},
		{
			Name:  "decorator with number argument",
			Input: "retry-task: @retry(3) { flaky-command }",
			Expected: Program(
				CmdWith(Decorator("retry", "3"), "retry-task", Simple(
					Text("flaky-command"),
				)),
			),
		},
		{
			Name:  "decorator with multiple arguments",
			Input: "watch-files: @debounce(500ms, \"src/**/*\") { npm run build }",
			Expected: Program(
				CmdWith(Decorator("debounce", "500ms", "src/**/*"), "watch-files", Simple(
					Text("npm run build"),
				)),
			),
		},
		{
			Name:  "multiple decorators on one command",
			Input: "complex: @timeout(30s) @retry(3) @env(NODE_ENV=test) { npm test }",
			Expected: Program(
				CmdWith([]ExpectedDecorator{
					Decorator("timeout", "30s"),
					Decorator("retry", "3"),
					Decorator("env", "NODE_ENV=test"),
				}, "complex", Simple(
					Text("npm test"),
				)),
			),
		},
		{
			Name:  "decorator with @var argument",
			Input: "deploy: @cwd(@var(BUILD_DIR)) { make install }",
			Expected: Program(
				CmdWith(Decorator("cwd", At("var", "BUILD_DIR")), "deploy", Simple(
					Text("make install"),
				)),
			),
		},
		{
			Name:  "decorator with mixed argument types",
			Input: "advanced: @watch-files(@var(PATTERN), 1s, true) { rebuild }",
			Expected: Program(
				CmdWith(Decorator("watch-files", At("var", "PATTERN"), "1s", "true"), "advanced", Simple(
					Text("rebuild"),
				)),
			),
		},
		{
			Name:  "decorator with string containing @ symbol",
			Input: "email: @sh(echo \"Contact us @ support@company.com\")",
			Expected: Program(
				Cmd("email", Simple(
					At("sh", "echo \"Contact us @ support@company.com\""),
				)),
			),
		},
		{
			Name:  "decorator with boolean argument",
			Input: "deploy: @confirm(true) { ./deploy.sh }",
			Expected: Program(
				CmdWith(Decorator("confirm", "true"), "deploy", Simple(
					Text("./deploy.sh"),
				)),
			),
		},
		{
			Name:  "decorator with negative number",
			Input: "adjust: @offset(-5) { process }",
			Expected: Program(
				CmdWith(Decorator("offset", "-5"), "adjust", Simple(
					Text("process"),
				)),
			),
		},
		{
			Name:  "decorator with decimal number",
			Input: "scale: @factor(1.5) { scale-service }",
			Expected: Program(
				CmdWith(Decorator("factor", "1.5"), "scale", Simple(
					Text("scale-service"),
				)),
			),
		},
		{
			Name:  "multiple decorators with different argument types",
			Input: "complex: @timeout(30s) @retry(3) @env(\"NODE_ENV=test\") @confirm(false) { test-suite }",
			Expected: Program(
				CmdWith([]ExpectedDecorator{
					Decorator("timeout", "30s"),
					Decorator("retry", "3"),
					Decorator("env", "NODE_ENV=test"),
					Decorator("confirm", "false"),
				}, "complex", Simple(
					Text("test-suite"),
				)),
			),
		},
		{
			Name:  "decorator with no arguments but parentheses",
			Input: "test: @parallel() { task1; task2 }",
			Expected: Program(
				CmdWith(Decorator("parallel"), "test", Block(
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
