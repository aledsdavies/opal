package parser

import (
	"testing"
)

// Helper to create an identifier expression for clarity in tests
func identifier(val string) ExpectedExpression {
	return ExpectedExpression{Type: "identifier", Value: val}
}

func TestVariableDefinitions(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "simple variable",
			Input: "var SRC = ./src",
			Expected: Program(
				Var("SRC", identifier("./src")),
			),
		},
		{
			Name:  "variable with complex value",
			Input: "var CMD = go test -v ./...",
			Expected: Program(
				Var("CMD", identifier("go test -v ./...")),
			),
		},
		{
			Name:  "multiple variables",
			Input: "var SRC = ./src\nvar BIN = ./bin",
			Expected: Program(
				Var("SRC", identifier("./src")),
				Var("BIN", identifier("./bin")),
			),
		},
		{
			Name:  "grouped variables",
			Input: "var (\n  SRC = ./src\n  BIN = ./bin\n)",
			Expected: Program(
				Var("SRC", identifier("./src")),
				Var("BIN", identifier("./bin")),
			),
		},
		{
			Name:  "variable with number value",
			Input: "var PORT = 8080",
			Expected: Program(
				Var("PORT", "8080"),
			),
		},
		{
			Name:  "variable with duration value",
			Input: "var TIMEOUT = 30s",
			Expected: Program(
				Var("TIMEOUT", DurationExpr("30s")),
			),
		},
		{
			Name:  "variable with quoted string",
			Input: `var MESSAGE = "Hello, World!"`,
			Expected: Program(
				Var("MESSAGE", "Hello, World!"),
			),
		},
		{
			Name:  "variable with special characters",
			Input: "var API_URL = https://api.example.com/v1",
			Expected: Program(
				Var("API_URL", identifier("https://api.example.com/v1")),
			),
		},
		{
			Name:  "mixed variable types in group",
			Input: "var (\n  SRC = ./src\n  PORT = 3000\n  TIMEOUT = 5m\n  DEBUG = true\n)",
			Expected: Program(
				Var("SRC", identifier("./src")),
				Var("PORT", "3000"),
				Var("TIMEOUT", DurationExpr("5m")),
				Var("DEBUG", identifier("true")),
			),
		},
		{
			Name:  "variable with environment-style name",
			Input: "var NODE_ENV = production",
			Expected: Program(
				Var("NODE_ENV", identifier("production")),
			),
		},
		{
			Name:  "variable with URL containing query params",
			Input: "var API_URL = https://api.example.com/v1?key=abc123",
			Expected: Program(
				Var("API_URL", identifier("https://api.example.com/v1?key=abc123")),
			),
		},
		{
			Name:  "variable with boolean-like value",
			Input: "var DEBUG = true",
			Expected: Program(
				Var("DEBUG", identifier("true")),
			),
		},
		{
			Name:  "variable with path containing spaces",
			Input: `var PROJECT_PATH = "/path/with spaces/project"`,
			Expected: Program(
				Var("PROJECT_PATH", "/path/with spaces/project"),
			),
		},
		{
			Name:  "variable with empty string value",
			Input: `var EMPTY = ""`,
			Expected: Program(
				Var("EMPTY", ""),
			),
		},
		{
			Name:  "variable with numeric string",
			Input: `var VERSION = "1.2.3"`,
			Expected: Program(
				Var("VERSION", "1.2.3"),
			),
		},
		{
			Name:  "variable with complex file path",
			Input: "var CONFIG_FILE = /etc/myapp/config.json",
			Expected: Program(
				Var("CONFIG_FILE", identifier("/etc/myapp/config.json")),
			),
		},
		{
			Name:  "variable with URL containing port (quoted to avoid colon parsing issues)",
			Input: `var DATABASE_URL = "postgresql://user:pass@localhost:5432/dbname"`,
			Expected: Program(
				Var("DATABASE_URL", "postgresql://user:pass@localhost:5432/dbname"),
			),
		},
		{
			Name:  "variable with floating point duration",
			Input: "var TIMEOUT = 2.5s",
			Expected: Program(
				Var("TIMEOUT", DurationExpr("2.5s")),
			),
		},
		{
			Name:  "multiple variables with mixed types",
			Input: "var PORT = 3000\nvar HOST = localhost\nvar TIMEOUT = 30s\nvar DEBUG = true",
			Expected: Program(
				Var("PORT", "3000"),
				Var("HOST", identifier("localhost")),
				Var("TIMEOUT", DurationExpr("30s")),
				Var("DEBUG", identifier("true")),
			),
		},
		{
			Name:  "variable with quoted identifier value",
			Input: `var MODE = "production"`,
			Expected: Program(
				Var("MODE", "production"),
			),
		},
		{
			Name:  "variable with underscores and URL",
			Input: `var API_BASE_URL = "https://api.example.com"`,
			Expected: Program(
				Var("API_BASE_URL", "https://api.example.com"),
			),
		},
	}

	for _, tc := range testCases {
		RunTestCase(t, tc)
	}
}

func TestVariableUsageInCommands(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "variables with command usage - requires explicit block",
			Input: "var SRC = ./src\nvar DEST = ./dist\nbuild: { cp -r @var(SRC)/* @var(DEST)/ }",
			Expected: Program(
				Var("SRC", identifier("./src")),
				Var("DEST", identifier("./dist")),
				CmdBlock("build",
					Text("cp -r "),
					At("var", "SRC"),
					Text("/* "),
					At("var", "DEST"),
					Text("/"),
				),
			),
		},
		{
			Name:  "grouped variables with usage - requires explicit block",
			Input: "var (\n  PORT = 8080\n  HOST = localhost\n)\nserve: { go run main.go --port=@var(PORT) --host=@var(HOST) }",
			Expected: Program(
				Var("PORT", "8080"),
				Var("HOST", identifier("localhost")),
				CmdBlock("serve",
					Text("go run main.go --port="),
					At("var", "PORT"),
					Text(" --host="),
					At("var", "HOST"),
				),
			),
		},
		{
			Name:  "variables in block commands",
			Input: "var SRC = ./src\ndeploy: { cd @var(SRC); make clean; make install }",
			Expected: Program(
				Var("SRC", identifier("./src")),
				CmdBlock("deploy",
					Text("cd "),
					At("var", "SRC"),
					Text("; make clean; make install"),
				),
			),
		},
		{
			Name:  "variables in decorator arguments",
			Input: "var TIMEOUT = 30s\ntest: @timeout(@var(TIMEOUT)) { npm test }",
			Expected: Program(
				Var("TIMEOUT", DurationExpr("30s")),
				CmdBlock("test",
					Decorator("timeout", FuncDecorator("var", "TIMEOUT")),
					Text("npm test"),
				),
			),
		},
		{
			Name:  "complex variable usage with multiple decorators",
			Input: "var ENV = production\nvar TIME = 5m\ndeploy: @env(NODE_ENV=@var(ENV)) @timeout(@var(TIME)) { npm run deploy }",
			Expected: Program(
				Var("ENV", identifier("production")),
				Var("TIME", DurationExpr("5m")),
				CmdBlock("deploy",
					Decorator("env", identifier("NODE_ENV=@var(ENV)")), // This is parsed as a single identifier
					Decorator("timeout", FuncDecorator("var", "TIME")),
					Text("npm run deploy"),
				),
			),
		},
		{
			Name:  "variables in watch commands",
			Input: `var SRC = ./src
watch build: @debounce(500ms) { echo "Building @var(SRC)" }`,
			Expected: Program(
				Var("SRC", identifier("./src")),
				WatchBlock("build",
					Decorator("debounce", "500ms"),
					Text(`echo "Building `),
					At("var", "SRC"),
					Text(`"`),
				),
			),
		},
		{
			Name:  "variables in stop commands - simple command gets syntax sugar",
			Input: "var PROCESS = myapp\nstop server: { pkill -f @var(PROCESS) }",
			Expected: Program(
				Var("PROCESS", identifier("myapp")),
				StopBlock("server",
					Text("pkill -f "),
					At("var", "PROCESS"),
				),
			),
		},
		{
			Name:  "variables with file counting command - requires explicit block",
			Input: `var SRC = ./src
build: { echo "Files: $(ls @var(SRC) | wc -l)" }`,
			Expected: Program(
				Var("SRC", identifier("./src")),
				CmdBlock("build",
					Text(`echo "Files: $(ls `),
					At("var", "SRC"),
					Text(` | wc -l)"`),
				),
			),
		},
		{
			Name:  "variables with nested shell content - requires explicit block",
			Input: "var HOST = server.com\nvar PORT = 22\nconnect: { ssh -p @var(PORT) user@@var(HOST) }",
			Expected: Program(
				Var("HOST", identifier("server.com")),
				Var("PORT", "22"),
				CmdBlock("connect",
					Text("ssh -p "),
					At("var", "PORT"),
					Text(" user@"),
					At("var", "HOST"),
				),
			),
		},
		{
			Name:  "variables in complex command chains - requires explicit block",
			Input: "var SRC = ./src\nvar DEST = ./dist\nvar ENV = prod\nbuild: { cd @var(SRC) && npm run build:@var(ENV) && cp -r dist/* @var(DEST)/ }",
			Expected: Program(
				Var("SRC", identifier("./src")),
				Var("DEST", identifier("./dist")),
				Var("ENV", identifier("prod")),
				CmdBlock("build",
					Text("cd "),
					At("var", "SRC"),
					Text(" && npm run build:"),
					At("var", "ENV"),
					Text(" && cp -r dist/* "),
					At("var", "DEST"),
					Text("/"),
				),
			),
		},
		{
			Name:  "variables in conditional expressions - requires explicit block",
			Input: `var ENV = production
check: { test "@var(ENV)" = "production" && echo "prod mode" || echo "dev mode" }`,
			Expected: Program(
				Var("ENV", identifier("production")),
				CmdBlock("check",
					Text(`test "`),
					At("var", "ENV"),
					Text(`" = "production" && echo "prod mode" || echo "dev mode"`),
				),
			),
		},
	}

	for _, tc := range testCases {
		RunTestCase(t, tc)
	}
}

func TestVariableEdgeCases(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "variable with special characters in name",
			Input: "var API_BASE_URL_V2 = https://api.example.com/v2",
			Expected: Program(
				Var("API_BASE_URL_V2", identifier("https://api.example.com/v2")),
			),
		},
		{
			Name:  "variable with mixed case",
			Input: "var NodeEnv = development",
			Expected: Program(
				Var("NodeEnv", identifier("development")),
			),
		},
		{
			Name:  "variable with numbers in name",
			Input: "var API_V2_URL = https://api.example.com/v2",
			Expected: Program(
				Var("API_V2_URL", identifier("https://api.example.com/v2")),
			),
		},
		{
			Name:  "variable with very long value",
			Input: "var LONG_VALUE = this-is-a-very-long-value-that-spans-multiple-words-and-contains-hyphens",
			Expected: Program(
				Var("LONG_VALUE", identifier("this-is-a-very-long-value-that-spans-multiple-words-and-contains-hyphens")),
			),
		},
		{
			Name:  "variable with value containing equals (quoted)",
			Input: `var QUERY = "name=value&other=data"`,
			Expected: Program(
				Var("QUERY", "name=value&other=data"),
			),
		},
		{
			Name:  `variable with quoted value containing spaces`,
			Input: `var MESSAGE = "Hello World from Devcmd"`,
			Expected: Program(
				Var("MESSAGE", "Hello World from Devcmd"),
			),
		},
		{
			Name:  "variables with similar names",
			Input: `var API_URL = "https://api.com"
var API_URL_V2 = "https://api.com/v2"`,
			Expected: Program(
				Var("API_URL", "https://api.com"),
				Var("API_URL_V2", "https://api.com/v2"),
			),
		},
		{
			Name:  "variable usage in quoted strings - requires explicit block",
			Input: `var NAME = World
greet: { echo "Hello @var(NAME)!" }`,
			Expected: Program(
				Var("NAME", identifier("World")),
				CmdBlock("greet",
					Text(`echo "Hello `),
					At("var", "NAME"),
					Text(`!"`),
				),
			),
		},
		{
			Name:  "variable usage with shell operators - requires explicit block",
			Input: "var FILE = data.txt\nprocess: { cat @var(FILE) | grep pattern | sort }",
			Expected: Program(
				Var("FILE", identifier("data.txt")),
				CmdBlock("process",
					Text("cat "),
					At("var", "FILE"),
					Text(" | grep pattern | sort"),
				),
			),
		},
		{
			Name:  "variable usage in file paths - requires explicit block",
			Input: "var HOME = /home/user\nbackup: { cp important.txt @var(HOME)/backup/ }",
			Expected: Program(
				Var("HOME", identifier("/home/user")),
				CmdBlock("backup",
					Text("cp important.txt "),
					At("var", "HOME"),
					Text("/backup/"),
				),
			),
		},
	}

	for _, tc := range testCases {
		RunTestCase(t, tc)
	}
}
