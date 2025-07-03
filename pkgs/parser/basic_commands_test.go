package parser

import (
	"testing"
)

func TestBasicCommands(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "simple command",
			Input: "build: echo hello",
			Expected: Program(
				Cmd("build", Simple(Text("echo hello"))),
			),
		},
		{
			Name:  "command with special characters",
			Input: "run: echo 'Hello, World!'",
			Expected: Program(
				Cmd("run", Simple(Text("echo 'Hello, World!'"))),
			),
		},
		{
			Name:  "empty command",
			Input: "noop:",
			Expected: Program(
				Cmd("noop", Simple()),
			),
		},
		{
			Name:  "command with parentheses",
			Input: "check: (echo test)",
			Expected: Program(
				Cmd("check", Simple(Text("(echo test)"))),
			),
		},
		{
			Name:  "command with pipes",
			Input: "process: echo hello | grep hello",
			Expected: Program(
				Cmd("process", Simple(Text("echo hello | grep hello"))),
			),
		},
		{
			Name:  "command with redirection",
			Input: "save: echo hello > output.txt",
			Expected: Program(
				Cmd("save", Simple(Text("echo hello > output.txt"))),
			),
		},
		{
			Name:  "command with background process",
			Input: "background: sleep 10 &",
			Expected: Program(
				Cmd("background", Simple(Text("sleep 10 &"))),
			),
		},
		{
			Name:  "command with logical operators",
			Input: "conditional: test -f file.txt && echo exists || echo missing",
			Expected: Program(
				Cmd("conditional", Simple(Text("test -f file.txt && echo exists || echo missing"))),
			),
		},
		{
			Name:  "command with environment variables",
			Input: "env-test: NODE_ENV=production npm start",
			Expected: Program(
				Cmd("env-test", Simple(Text("NODE_ENV=production npm start"))),
			),
		},
		{
			Name:  "command with complex shell syntax",
			Input: "complex: for i in {1..5}; do echo $i; done",
			Expected: Program(
				Cmd("complex", Simple(Text("for i in {1..5}; do echo $i; done"))),
			),
		},
		{
			Name:  "command with tabs and mixed whitespace",
			Input: "build:\t\techo\t\"building\" \t&& \tmake",
			Expected: Program(
				Cmd("build", Simple(Text("echo\t\"building\" \t&& \tmake"))),
			),
		},
		{
			Name:  "command name with underscores and hyphens",
			Input: "test_command-name: echo hello",
			Expected: Program(
				Cmd("test_command-name", Simple(Text("echo hello"))),
			),
		},
	}

	for _, tc := range testCases {
		RunTestCase(t, tc)
	}
}

func TestWatchStopCommands(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "simple watch command",
			Input: "watch server: npm start",
			Expected: Program(
				Watch("server", Simple(Text("npm start"))),
			),
		},
		{
			Name:  "simple stop command",
			Input: "stop server: pkill node",
			Expected: Program(
				Stop("server", Simple(Text("pkill node"))),
			),
		},
		{
			Name:  "watch command with @var()",
			Input: "watch server: go run @var(MAIN_FILE) --port=@var(PORT)",
			Expected: Program(
				Watch("server", Simple(
					Text("go run "),
					At("var", "MAIN_FILE"),
					Text(" --port="),
					At("var", "PORT"),
				)),
			),
		},
		{
			Name:  "watch block command",
			Input: "watch dev: { npm start; go run main.go }",
			Expected: Program(
				Watch("dev", Block(
					Text("npm start"),
					Text("go run main.go"),
				)),
			),
		},
		{
			Name:  "watch with timeout decorator",
			Input: "watch build: @timeout(60s) { npm run watch:build }",
			Expected: Program(
				WatchWith(Decorator("timeout", "60s"), "build", Simple(
					Text("npm run watch:build"),
				)),
			),
		},
		{
			Name:  "watch with parallel decorator",
			Input: "watch services: @parallel { npm run api; npm run worker; npm run scheduler }",
			Expected: Program(
				WatchWith(Decorator("parallel"), "services", Block(
					Text("npm run api"),
					Text("npm run worker"),
					Text("npm run scheduler"),
				)),
			),
		},
		{
			Name:  "stop with cleanup block",
			Input: "stop services: { pkill -f node; docker stop $(docker ps -q); echo cleaned }",
			Expected: Program(
				Stop("services", Block(
					Text("pkill -f node"),
					Text("docker stop $(docker ps -q)"),
					Text("echo cleaned"),
				)),
			),
		},
		{
			Name:  "watch and stop with same name should be allowed",
			Input: "watch server: npm start\nstop server: pkill node",
			Expected: Program(
				Watch("server", Simple(Text("npm start"))),
				Stop("server", Simple(Text("pkill node"))),
			),
		},
	}

	for _, tc := range testCases {
		RunTestCase(t, tc)
	}
}

func TestBlockCommands(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "empty block",
			Input: "setup: { }",
			Expected: Program(
				Cmd("setup", Block()),
			),
		},
		{
			Name:  "single statement block",
			Input: "setup: { npm install }",
			Expected: Program(
				Cmd("setup", Block(Text("npm install"))),
			),
		},
		{
			Name:  "multiple statements",
			Input: "setup: { npm install; go mod tidy; echo done }",
			Expected: Program(
				Cmd("setup", Block(
					Text("npm install"),
					Text("go mod tidy"),
					Text("echo done"),
				)),
			),
		},
		{
			Name:  "block with @var() references",
			Input: "build: { cd @var(SRC); make @var(TARGET) }",
			Expected: Program(
				Cmd("build", Block(
					Text("cd "),
					At("var", "SRC"),
					Text("make "),
					At("var", "TARGET"),
				)),
			),
		},
		{
			Name:  "block with complex shell statements",
			Input: "test: { echo start; for i in {1..3}; do echo $i; done; echo end }",
			Expected: Program(
				Cmd("test", Block(
					Text("echo start"),
					Text("for i in {1..3}; do echo $i; done"),
					Text("echo end"),
				)),
			),
		},
		{
			Name:  "block with conditional statements",
			Input: "conditional: { test -f file.txt && echo exists || echo missing; echo checked }",
			Expected: Program(
				Cmd("conditional", Block(
					Text("test -f file.txt && echo exists || echo missing"),
					Text("echo checked"),
				)),
			),
		},
		{
			Name:  "block with background processes",
			Input: "background: { server &; client &; wait }",
			Expected: Program(
				Cmd("background", Block(
					Text("server &"),
					Text("client &"),
					Text("wait"),
				)),
			),
		},
		{
			Name:  "block with mixed @var() and shell text",
			Input: "deploy: { echo \"Deploying @var(APP_NAME) to @var(ENVIRONMENT)\"; kubectl apply -f @var(CONFIG_FILE) }",
			Expected: Program(
				Cmd("deploy", Block(
					Text("echo \"Deploying "),
					At("var", "APP_NAME"),
					Text(" to "),
					At("var", "ENVIRONMENT"),
					Text("\""),
					Text("kubectl apply -f "),
					At("var", "CONFIG_FILE"),
				)),
			),
		},
		{
			Name:  "block with decorator",
			Input: "services: @parallel { server; client }",
			Expected: Program(
				CmdWith(Decorator("parallel"), "services", Block(
					Text("server"),
					Text("client"),
				)),
			),
		},
		{
			Name:  "block with timeout decorator",
			Input: "deploy: @timeout(5m) { npm run build; npm run deploy }",
			Expected: Program(
				CmdWith(Decorator("timeout", "5m"), "deploy", Block(
					Text("npm run build"),
					Text("npm run deploy"),
				)),
			),
		},
		{
			Name:  "block with retry decorator",
			Input: "flaky-task: @retry(3) { npm test }",
			Expected: Program(
				CmdWith(Decorator("retry", "3"), "flaky-task", Simple(
					Text("npm test"),
				)),
			),
		},
		{
			Name:  "block with multiple decorators",
			Input: "complex: @timeout(30s) @retry(2) { npm run integration-tests }",
			Expected: Program(
				CmdWith([]ExpectedDecorator{
					Decorator("timeout", "30s"),
					Decorator("retry", "2"),
				}, "complex", Simple(
					Text("npm run integration-tests"),
				)),
			),
		},
	}

	for _, tc := range testCases {
		RunTestCase(t, tc)
	}
}

func TestCommandsWithVariables(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "simple variable usage",
			Input: "build: echo @var(MESSAGE)",
			Expected: Program(
				Cmd("build", Simple(
					Text("echo "),
					At("var", "MESSAGE"),
				)),
			),
		},
		{
			Name:  "multiple variables in command",
			Input: "deploy: docker run --name @var(CONTAINER) -p @var(PORT):@var(PORT) @var(IMAGE)",
			Expected: Program(
				Cmd("deploy", Simple(
					Text("docker run --name "),
					At("var", "CONTAINER"),
					Text(" -p "),
					At("var", "PORT"),
					Text(":"),
					At("var", "PORT"),
					Text(" "),
					At("var", "IMAGE"),
				)),
			),
		},
		{
			Name:  "variable in quoted string",
			Input: "msg: echo \"Hello @var(NAME), welcome to @var(APP)!\"",
			Expected: Program(
				Cmd("msg", Simple(
					Text("echo \"Hello "),
					At("var", "NAME"),
					Text(", welcome to "),
					At("var", "APP"),
					Text("!\""),
				)),
			),
		},
		{
			Name:  "variable with file paths",
			Input: "copy: cp @var(SRC)/* @var(DEST)/",
			Expected: Program(
				Cmd("copy", Simple(
					Text("cp "),
					At("var", "SRC"),
					Text("/* "),
					At("var", "DEST"),
					Text("/"),
				)),
			),
		},
		{
			Name:  "variable in complex shell command",
			Input: "check: test -f @var(CONFIG_FILE) && echo \"Config exists\" || echo \"Missing config\"",
			Expected: Program(
				Cmd("check", Simple(
					Text("test -f "),
					At("var", "CONFIG_FILE"),
					Text(" && echo \"Config exists\" || echo \"Missing config\""),
				)),
			),
		},
		{
			Name:  "variable with email-like text",
			Input: "notify: echo \"Build @var(STATUS)\" | mail admin@company.com",
			Expected: Program(
				Cmd("notify", Simple(
					Text("echo \"Build "),
					At("var", "STATUS"),
					Text("\" | mail admin@company.com"),
				)),
			),
		},
		{
			Name:  "variable in environment setting",
			Input: "serve: NODE_ENV=@var(ENV) npm start",
			Expected: Program(
				Cmd("serve", Simple(
					Text("NODE_ENV="),
					At("var", "ENV"),
					Text(" npm start"),
				)),
			),
		},
		{
			Name:  "variable in URL",
			Input: "api-call: curl https://api.example.com/@var(ENDPOINT)?token=@var(TOKEN)",
			Expected: Program(
				Cmd("api-call", Simple(
					Text("curl https://api.example.com/"),
					At("var", "ENDPOINT"),
					Text("?token="),
					At("var", "TOKEN"),
				)),
			),
		},
	}

	for _, tc := range testCases {
		RunTestCase(t, tc)
	}
}
