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
					"npm start",
					"go run main.go",
				)),
			),
		},
		{
			Name:  "watch with decorators",
			Input: "watch api: @env(NODE_ENV=development) { npm run dev }",
			Expected: Program(
				WatchWith(At("env", "NODE_ENV=development"), "api", Block(
					"npm run dev",
				)),
			),
		},
		{
			Name:  "stop with graceful shutdown",
			Input: "stop api: @sh(curl -X POST localhost:3000/shutdown || pkill node)",
			Expected: Program(
				StopWith(At("sh", "curl -X POST localhost:3000/shutdown || pkill node"), "api", Block()),
			),
		},
		{
			Name:  "watch with multiple services",
			Input: "watch services: @parallel { npm run api; npm run worker; npm run scheduler }",
			Expected: Program(
				WatchWith(At("parallel"), "services", Block(
					"npm run api",
					"npm run worker",
					"npm run scheduler",
				)),
			),
		},
		{
			Name:  "stop with cleanup block",
			Input: "stop services: { pkill -f node; docker stop $(docker ps -q); echo cleaned }",
			Expected: Program(
				Stop("services", Block(
					"pkill -f node",
					"docker stop $(docker ps -q)",
					"echo cleaned",
				)),
			),
		},
		{
			Name:  "watch with timeout",
			Input: "watch build: @timeout(60s) { npm run watch:build }",
			Expected: Program(
				WatchWith(At("timeout", "60s"), "build", Block(
					"npm run watch:build",
				)),
			),
		},
		{
			Name:  "stop with confirmation",
			Input: "stop production: @confirm(\"Really stop production?\") { systemctl stop myapp }",
			Expected: Program(
				StopWith(At("confirm", "Really stop production?"), "production", Block(
					"systemctl stop myapp",
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
				Cmd("setup", Block("npm install")),
			),
		},
		{
			Name:  "multiple statements",
			Input: "setup: { npm install; go mod tidy; echo done }",
			Expected: Program(
				Cmd("setup", Block(
					"npm install",
					"go mod tidy",
					"echo done",
				)),
			),
		},
		{
			Name:  "block with @var() references",
			Input: "build: { cd @var(SRC); make @var(TARGET) }",
			Expected: Program(
				Cmd("build", Block(
					Statement(Text("cd "), At("var", "SRC")),
					Statement(Text("make "), At("var", "TARGET")),
				)),
			),
		},
		{
			Name:  "block with decorators",
			Input: "services: { @parallel { server; client } }",
			Expected: Program(
				Cmd("services", Block(
					At("parallel"),
				)),
			),
		},
		{
			Name:  "nested blocks with decorators",
			Input: "complex: { @timeout(30s) { @sh(long-running-task); echo done } }",
			Expected: Program(
				Cmd("complex", Block(
					At("timeout", "30s"),
				)),
			),
		},
		{
			Name:  "block with mixed statements and decorators",
			Input: "deploy: { echo starting; @sh(deploy.sh); @parallel { service1; service2 }; echo finished }",
			Expected: Program(
				Cmd("deploy", Block(
					"echo starting",
					At("sh", "deploy.sh"),
					At("parallel"),
					"echo finished",
				)),
			),
		},
		{
			Name:  "block with complex shell statements",
			Input: "test: { echo start; for i in {1..3}; do echo $i; done; echo end }",
			Expected: Program(
				Cmd("test", Block(
					"echo start",
					"for i in {1..3}; do echo $i; done",
					"echo end",
				)),
			),
		},
		{
			Name:  "block with conditional statements",
			Input: "conditional: { test -f file.txt && echo exists || echo missing; echo checked }",
			Expected: Program(
				Cmd("conditional", Block(
					"test -f file.txt && echo exists || echo missing",
					"echo checked",
				)),
			),
		},
		{
			Name:  "block with background processes",
			Input: "background: { server &; client &; wait }",
			Expected: Program(
				Cmd("background", Block(
					"server &",
					"client &",
					"wait",
				)),
			),
		},
	}

	for _, tc := range testCases {
		RunTestCase(t, tc)
	}
}
