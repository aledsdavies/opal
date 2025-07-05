package parser

import (
	"testing"
)

func TestVarDecorators(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "simple @var() reference - requires explicit block",
			Input: "build: { cd @var(SRC) }",
			Expected: Program(
				CmdBlock("build", Text("cd "), At("var", "SRC")),
			),
		},
		{
			Name:  "multiple @var() references - requires explicit block",
			Input: "deploy: { docker build -t @var(IMAGE):@var(TAG) }",
			Expected: Program(
				CmdBlock("deploy",
					Text("docker build -t "),
					At("var", "IMAGE"),
					Text(":"),
					At("var", "TAG"),
				),
			),
		},
		{
			Name:  "@var() in quoted string - requires explicit block",
			Input: "echo: { echo \"Building @var(PROJECT) version @var(VERSION)\" }",
			Expected: Program(
				CmdBlock("echo",
					Text("echo \"Building "),
					At("var", "PROJECT"),
					Text(" version "),
					At("var", "VERSION"),
					Text("\""),
				),
			),
		},
		{
			Name:  "mixed @var() and shell variables - requires explicit block",
			Input: "info: { echo \"Project: @var(NAME), User: $USER\" }",
			Expected: Program(
				CmdBlock("info",
					Text("echo \"Project: "),
					At("var", "NAME"),
					Text(", User: $USER\""),
				),
			),
		},
		{
			Name:  "@var() in file paths - requires explicit block",
			Input: "copy: { cp @var(SRC)/*.go @var(DEST)/ }",
			Expected: Program(
				CmdBlock("copy",
					Text("cp "),
					At("var", "SRC"),
					Text("/*.go "),
					At("var", "DEST"),
					Text("/"),
				),
			),
		},
		{
			Name:  "@var() in command arguments - requires explicit block",
			Input: "serve: { go run main.go --port=@var(PORT) --host=@var(HOST) }",
			Expected: Program(
				CmdBlock("serve",
					Text("go run main.go --port="),
					At("var", "PORT"),
					Text(" --host="),
					At("var", "HOST"),
				),
			),
		},
		{
			Name:  "@var() with special characters in value - requires explicit block",
			Input: "url: { curl \"@var(API_URL)/users?filter=@var(FILTER)\" }",
			Expected: Program(
				CmdBlock("url",
					Text("curl \""),
					At("var", "API_URL"),
					Text("/users?filter="),
					At("var", "FILTER"),
					Text("\""),
				),
			),
		},
		{
			Name:  "@var() in conditional expressions - requires explicit block",
			Input: "check: { test \"@var(ENV)\" = \"production\" && echo prod || echo dev }",
			Expected: Program(
				CmdBlock("check",
					Text("test \""),
					At("var", "ENV"),
					Text("\" = \"production\" && echo prod || echo dev"),
				),
			),
		},
		{
			Name:  "@var() in loops - requires explicit block",
			Input: "process: { for file in @var(SRC)/*.txt; do process $file; done }",
			Expected: Program(
				CmdBlock("process",
					Text("for file in "),
					At("var", "SRC"),
					Text("/*.txt; do process $file; done"),
				),
			),
		},
		{
			Name:  "string with escaped quotes and @var - requires explicit block",
			Input: "msg: { echo \"He said \\\"Hello @var(NAME)\\\" to everyone\" }",
			Expected: Program(
				CmdBlock("msg",
					Text("echo \"He said \\\"Hello "),
					At("var", "NAME"),
					Text("\\\" to everyone\""),
				),
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
				CmdBlock("deploy",
					Decorator("timeout", "30s"),
					Text("echo deploying"),
				),
			),
		},
		{
			Name:  "valid @env decorator with argument",
			Input: "setup: @env(NODE_ENV=production) { npm start }",
			Expected: Program(
				CmdBlock("setup",
					Decorator("env", "NODE_ENV=production"),
					Text("npm start"),
				),
			),
		},
		{
			Name:  "valid @confirm decorator",
			Input: "dangerous: @confirm(\"Are you sure?\") { rm -rf /tmp/* }",
			Expected: Program(
				CmdBlock("dangerous",
					Decorator("confirm", "Are you sure?"),
					Text("rm -rf /tmp/*"),
				),
			),
		},
		{
			Name:  "valid @debounce decorator",
			Input: "watch-changes: @debounce(500ms) { npm run build }",
			Expected: Program(
				CmdBlock("watch-changes",
					Decorator("debounce", "500ms"),
					Text("npm run build"),
				),
			),
		},
		{
			Name:  "valid @cwd decorator",
			Input: "build-lib: @cwd(./lib) { make all }",
			Expected: Program(
				CmdBlock("build-lib",
					Decorator("cwd", "./lib"),
					Text("make all"),
				),
			),
		},
		{
			Name:  "valid @parallel block decorator with multiple statements",
			Input: "services: @parallel { server; client }",
			Expected: Program(
				CmdBlock("services",
					Decorator("parallel"),
					Text("server; client"),
				),
			),
		},
		{
			Name:  "valid @retry block decorator with multiple statements",
			Input: "flaky-test: @retry { npm test; echo 'done' }",
			Expected: Program(
				CmdBlock("flaky-test",
					Decorator("retry"),
					Text("npm test; echo 'done'"),
				),
			),
		},
		{
			Name:  "valid @watch-files block decorator with multiple statements",
			Input: "monitor: @watch-files { echo 'checking'; sleep 1 }",
			Expected: Program(
				CmdBlock("monitor",
					Decorator("watch-files"),
					Text("echo 'checking'; sleep 1"),
				),
			),
		},
		{
			Name:  "empty block with decorators",
			Input: "parallel-empty: @parallel { }",
			Expected: Program(
				CmdBlock("parallel-empty",
					Decorator("parallel"),
				),
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
			Name:  "function decorator @sh() within shell content - requires explicit block",
			Input: "build: { echo start && @sh(cd) && echo done }",
			Expected: Program(
				CmdBlock("build",
					Text("echo start && "),
					At("sh", "cd"),
					Text(" && echo done"),
				),
			),
		},
		{
			Name:  "function decorator @sh() with simple command - requires explicit block",
			Input: "deploy: { @sh(deploy.sh) }",
			Expected: Program(
				CmdBlock("deploy", At("sh", "deploy.sh")),
			),
		},
		{
			Name:  "mixed function decorators and text - requires explicit block",
			Input: "info: { echo \"Date: @sh(date)\" and \"User: @sh(whoami)\" }",
			Expected: Program(
				CmdBlock("info",
					Text("echo \"Date: "),
					At("sh", "date"),
					Text("\" and \"User: "),
					At("sh", "whoami"),
					Text("\""),
				),
			),
		},
		{
			Name:  "function decorator with simple argument - requires explicit block",
			Input: "test: { @sh(test) && echo success }",
			Expected: Program(
				CmdBlock("test",
					At("sh", "test"),
					Text(" && echo success"),
				),
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
			Input: "deploy: @timeout(30s) { @sh(deploy.sh) }",
			Expected: Program(
				CmdBlock("deploy",
					Decorator("timeout", "30s"),
					At("sh", "deploy.sh"),
				),
			),
		},
		{
			Name:  "parallel with mixed content",
			Input: "multi: @parallel { echo start; echo end }",
			Expected: Program(
				CmdBlock("multi",
					Decorator("parallel"),
					Text("echo start; echo end"),
				),
			),
		},
		{
			Name:  "decorator with simple argument",
			Input: "setup: @env(PATH=/usr/bin) { which tool }",
			Expected: Program(
				CmdBlock("setup",
					Decorator("env", "PATH=/usr/bin"),
					Text("which tool"),
				),
			),
		},
		{
			Name:  "single timeout decorator",
			Input: "build: @timeout(30s) { npm test }",
			Expected: Program(
				CmdBlock("build",
					Decorator("timeout", "30s"),
					Text("npm test"),
				),
			),
		},
		{
			Name:  "decorator with @var as argument",
			Input: "build: @cwd(@var(BUILD_DIR)) { make clean && make all }",
			Expected: Program(
				CmdBlock("build",
					Decorator("cwd", At("var", "BUILD_DIR")),
					Text("make clean && make all"),
				),
			),
		},
		{
			Name:  "explicitly nested decorators",
			Input: "complex: @timeout(30s) { @retry(2) { npm run integration-tests } }",
			Expected: Program(
				CmdBlock("complex",
					Decorator("timeout", "30s"),
					Decorator("retry", "2"),
					Text("npm run integration-tests"),
				),
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
				CmdBlock("sync",
					Decorator("parallel"),
					Text("task1; task2"),
				),
			),
		},
		{
			Name:  "decorator with single string argument",
			Input: "ask: @confirm(\"Are you sure?\") { rm -rf /tmp/* }",
			Expected: Program(
				CmdBlock("ask",
					Decorator("confirm", "Are you sure?"),
					Text("rm -rf /tmp/*"),
				),
			),
		},
		{
			Name:  "decorator with duration argument",
			Input: "slow: @timeout(5m) { sleep 300 }",
			Expected: Program(
				CmdBlock("slow",
					Decorator("timeout", "5m"),
					Text("sleep 300"),
				),
			),
		},
		{
			Name:  "decorator with number argument",
			Input: "retry-task: @retry(3) { flaky-command }",
			Expected: Program(
				CmdBlock("retry-task",
					Decorator("retry", "3"),
					Text("flaky-command"),
				),
			),
		},
		{
			Name:  "decorator with single argument",
			Input: "watch-files: @debounce(500ms) { npm run build }",
			Expected: Program(
				CmdBlock("watch-files",
					Decorator("debounce", "500ms"),
					Text("npm run build"),
				),
			),
		},
		{
			Name:  "decorator with @var argument",
			Input: "deploy: @cwd(@var(BUILD_DIR)) { make install }",
			Expected: Program(
				CmdBlock("deploy",
					Decorator("cwd", At("var", "BUILD_DIR")),
					Text("make install"),
				),
			),
		},
		{
			Name:  "decorator with @var pattern argument",
			Input: "advanced: @watch-files(@var(PATTERN)) { rebuild }",
			Expected: Program(
				CmdBlock("advanced",
					Decorator("watch-files", At("var", "PATTERN")),
					Text("rebuild"),
				),
			),
		},
		{
			Name:  "decorator with boolean argument",
			Input: "deploy: @confirm(true) { ./deploy.sh }",
			Expected: Program(
				CmdBlock("deploy",
					Decorator("confirm", "true"),
					Text("./deploy.sh"),
				),
			),
		},
		{
			Name:  "decorator with negative number",
			Input: "adjust: @offset(-5) { process }",
			Expected: Program(
				CmdBlock("adjust",
					Decorator("offset", "-5"),
					Text("process"),
				),
			),
		},
		{
			Name:  "decorator with decimal number",
			Input: "scale: @factor(1.5) { scale-service }",
			Expected: Program(
				CmdBlock("scale",
					Decorator("factor", "1.5"),
					Text("scale-service"),
				),
			),
		},
		{
			Name:  "decorator with no arguments but parentheses",
			Input: "test: @parallel() { task1; task2 }",
			Expected: Program(
				CmdBlock("test",
					Decorator("parallel"),
					Text("task1; task2"),
				),
			),
		},
	}

	for _, tc := range testCases {
		RunTestCase(t, tc)
	}
}
