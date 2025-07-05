package parser

import (
	"testing"
)

func TestVariableDefinitions(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "simple variable with quoted string",
			Input: `var SRC = "./src"`,
			Expected: Program(
				Var("SRC", "./src"),
			),
		},
		{
			Name:  "variable with complex quoted value",
			Input: `var CMD = "go test -v ./..."`,
			Expected: Program(
				Var("CMD", "go test -v ./..."),
			),
		},
		{
			Name:  "multiple variables with quoted strings",
			Input: `var SRC = "./src"
var BIN = "./bin"`,
			Expected: Program(
				Var("SRC", "./src"),
				Var("BIN", "./bin"),
			),
		},
		{
			Name:  "grouped variables with quoted strings",
			Input: `var (
  SRC = "./src"
  BIN = "./bin"
)`,
			Expected: Program(
				Var("SRC", "./src"),
				Var("BIN", "./bin"),
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
			Name:  "variable with quoted string containing special chars",
			Input: `var MESSAGE = "Hello, World!"`,
			Expected: Program(
				Var("MESSAGE", "Hello, World!"),
			),
		},
		{
			Name:  "variable with quoted URL",
			Input: `var API_URL = "https://api.example.com/v1"`,
			Expected: Program(
				Var("API_URL", "https://api.example.com/v1"),
			),
		},
		{
			Name:  "mixed variable types in group",
			Input: `var (
  SRC = "./src"
  PORT = 3000
  TIMEOUT = 5m
  DEBUG = true
)`,
			Expected: Program(
				Var("SRC", "./src"),
				Var("PORT", "3000"),
				Var("TIMEOUT", DurationExpr("5m")),
				Var("DEBUG", BooleanExpr(true)),
			),
		},
		{
			Name:  "variable with environment-style name",
			Input: `var NODE_ENV = "production"`,
			Expected: Program(
				Var("NODE_ENV", "production"),
			),
		},
		{
			Name:  "variable with URL containing query params",
			Input: `var API_URL = "https://api.example.com/v1?key=abc123"`,
			Expected: Program(
				Var("API_URL", "https://api.example.com/v1?key=abc123"),
			),
		},
		{
			Name:  "variable with boolean value true",
			Input: "var DEBUG = true",
			Expected: Program(
				Var("DEBUG", BooleanExpr(true)),
			),
		},
		{
			Name:  "variable with boolean value false",
			Input: "var PRODUCTION = false",
			Expected: Program(
				Var("PRODUCTION", BooleanExpr(false)),
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
			Name:  "variable with quoted file path",
			Input: `var CONFIG_FILE = "/etc/myapp/config.json"`,
			Expected: Program(
				Var("CONFIG_FILE", "/etc/myapp/config.json"),
			),
		},
		{
			Name:  "variable with URL containing port",
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
			Input: `var PORT = 3000
var HOST = "localhost"
var TIMEOUT = 30s
var DEBUG = true`,
			Expected: Program(
				Var("PORT", "3000"),
				Var("HOST", "localhost"),
				Var("TIMEOUT", DurationExpr("30s")),
				Var("DEBUG", BooleanExpr(true)),
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
		{
			Name:  "variable with single quotes",
			Input: `var NAME = 'John Doe'`,
			Expected: Program(
				Var("NAME", "John Doe"),
			),
		},
		{
			Name:  "variable with backticks",
			Input: "var TEMPLATE = `Hello ${name}`",
			Expected: Program(
				Var("TEMPLATE", "Hello ${name}"),
			),
		},
		{
			Name:  "variable with negative number",
			Input: "var OFFSET = -100",
			Expected: Program(
				Var("OFFSET", "-100"),
			),
		},
		{
			Name:  "variable with floating point number",
			Input: "var RATIO = 3.14159",
			Expected: Program(
				Var("RATIO", "3.14159"),
			),
		},
		// Error cases - variables must use literal values
		{
			Name:        "error: unquoted string value",
			Input:       "var SRC = ./src",
			WantErr:     true,
			ErrorSubstr: "variable value must be a quoted string, number, duration, or boolean literal",
		},
		{
			Name:        "error: unquoted URL value",
			Input:       "var URL = https://example.com",
			WantErr:     true,
			ErrorSubstr: "variable value must be a quoted string, number, duration, or boolean literal",
		},
		{
			Name:        "error: unquoted path value",
			Input:       "var PATH = /usr/local/bin",
			WantErr:     true,
			ErrorSubstr: "variable value must be a quoted string, number, duration, or boolean literal",
		},
		{
			Name:        "error: unquoted complex value",
			Input:       "var CMD = go test ./...",
			WantErr:     true,
			ErrorSubstr: "variable value must be a quoted string, number, duration, or boolean literal",
		},
		{
			Name:        "error: grouped variable with unquoted value",
			Input:       "var (\n  SRC = ./src\n)",
			WantErr:     true,
			ErrorSubstr: "variable value must be a quoted string, number, duration, or boolean literal",
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
			Input: `var SRC = "./src"
var DEST = "./dist"
build: { cp -r @var(SRC)/* @var(DEST)/ }`,
			Expected: Program(
				Var("SRC", "./src"),
				Var("DEST", "./dist"),
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
			Input: `var (
  PORT = 8080
  HOST = "localhost"
)
serve: { go run main.go --port=@var(PORT) --host=@var(HOST) }`,
			Expected: Program(
				Var("PORT", "8080"),
				Var("HOST", "localhost"),
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
			Input: `var SRC = "./src"
deploy: { cd @var(SRC); make clean; make install }`,
			Expected: Program(
				Var("SRC", "./src"),
				CmdBlock("deploy",
					Text("cd "),
					At("var", "SRC"),
					Text("; make clean; make install"),
				),
			),
		},
		{
			Name:  "variables in decorator arguments",
			Input: `var TIMEOUT = 30s
test: @timeout(@var(TIMEOUT)) { npm test }`,
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
			Input: `var ENV = "production"
var TIME = 5m
deploy: @env(NODE_ENV=@var(ENV)) @timeout(@var(TIME)) { npm run deploy }`,
			Expected: Program(
				Var("ENV", "production"),
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
			Input: `var SRC = "./src"
watch build: @debounce(500ms) { echo "Building @var(SRC)" }`,
			Expected: Program(
				Var("SRC", "./src"),
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
			Input: `var PROCESS = "myapp"
stop server: { pkill -f @var(PROCESS) }`,
			Expected: Program(
				Var("PROCESS", "myapp"),
				StopBlock("server",
					Text("pkill -f "),
					At("var", "PROCESS"),
				),
			),
		},
		{
			Name:  "variables with file counting command - requires explicit block",
			Input: `var SRC = "./src"
build: { echo "Files: $(ls @var(SRC) | wc -l)" }`,
			Expected: Program(
				Var("SRC", "./src"),
				CmdBlock("build",
					Text(`echo "Files: $(ls `),
					At("var", "SRC"),
					Text(` | wc -l)"`),
				),
			),
		},
		{
			Name:  "variables with nested shell content - requires explicit block",
			Input: `var HOST = "server.com"
var PORT = 22
connect: { ssh -p @var(PORT) user@@var(HOST) }`,
			Expected: Program(
				Var("HOST", "server.com"),
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
			Input: `var SRC = "./src"
var DEST = "./dist"
var ENV = "prod"
build: { cd @var(SRC) && npm run build:@var(ENV) && cp -r dist/* @var(DEST)/ }`,
			Expected: Program(
				Var("SRC", "./src"),
				Var("DEST", "./dist"),
				Var("ENV", "prod"),
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
			Input: `var ENV = "production"
check: { test "@var(ENV)" = "production" && echo "prod mode" || echo "dev mode" }`,
			Expected: Program(
				Var("ENV", "production"),
				CmdBlock("check",
					Text(`test "`),
					At("var", "ENV"),
					Text(`" = "production" && echo "prod mode" || echo "dev mode"`),
				),
			),
		},
		{
			Name:  "boolean variable usage in commands",
			Input: `var DEBUG = true
run: { if [ "@var(DEBUG)" = "true" ]; then echo "Debug mode"; fi }`,
			Expected: Program(
				Var("DEBUG", BooleanExpr(true)),
				CmdBlock("run",
					Text(`if [ "`),
					At("var", "DEBUG"),
					Text(`" = "true" ]; then echo "Debug mode"; fi`),
				),
			),
		},
		{
			Name:  "number variable usage in commands",
			Input: `var MAX_WORKERS = 4
start: { node app.js --workers=@var(MAX_WORKERS) }`,
			Expected: Program(
				Var("MAX_WORKERS", "4"),
				CmdBlock("start",
					Text("node app.js --workers="),
					At("var", "MAX_WORKERS"),
				),
			),
		},
		{
			Name:  "duration variable usage in commands",
			Input: `var TIMEOUT = 30s
test: { npm test -- --timeout=@var(TIMEOUT) }`,
			Expected: Program(
				Var("TIMEOUT", DurationExpr("30s")),
				CmdBlock("test",
					Text("npm test -- --timeout="),
					At("var", "TIMEOUT"),
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
			Input: `var API_BASE_URL_V2 = "https://api.example.com/v2"`,
			Expected: Program(
				Var("API_BASE_URL_V2", "https://api.example.com/v2"),
			),
		},
		{
			Name:  "variable with mixed case",
			Input: `var NodeEnv = "development"`,
			Expected: Program(
				Var("NodeEnv", "development"),
			),
		},
		{
			Name:  "variable with numbers in name",
			Input: `var API_V2_URL = "https://api.example.com/v2"`,
			Expected: Program(
				Var("API_V2_URL", "https://api.example.com/v2"),
			),
		},
		{
			Name:  "variable with very long value",
			Input: `var LONG_VALUE = "this-is-a-very-long-value-that-spans-multiple-words-and-contains-hyphens"`,
			Expected: Program(
				Var("LONG_VALUE", "this-is-a-very-long-value-that-spans-multiple-words-and-contains-hyphens"),
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
			Input: `var NAME = "World"
greet: { echo "Hello @var(NAME)!" }`,
			Expected: Program(
				Var("NAME", "World"),
				CmdBlock("greet",
					Text(`echo "Hello `),
					At("var", "NAME"),
					Text(`!"`),
				),
			),
		},
		{
			Name:  "variable usage with shell operators - requires explicit block",
			Input: `var FILE = "data.txt"
process: { cat @var(FILE) | grep pattern | sort }`,
			Expected: Program(
				Var("FILE", "data.txt"),
				CmdBlock("process",
					Text("cat "),
					At("var", "FILE"),
					Text(" | grep pattern | sort"),
				),
			),
		},
		{
			Name:  "variable usage in file paths - requires explicit block",
			Input: `var HOME = "/home/user"
backup: { cp important.txt @var(HOME)/backup/ }`,
			Expected: Program(
				Var("HOME", "/home/user"),
				CmdBlock("backup",
					Text("cp important.txt "),
					At("var", "HOME"),
					Text("/backup/"),
				),
			),
		},
		{
			Name:  "variable with escaped quotes in string",
			Input: `var MSG = "He said \"Hello\""`,
			Expected: Program(
				Var("MSG", `He said "Hello"`),
			),
		},
		{
			Name:  "variable with newline in string",
			Input: `var MULTILINE = "Line 1\nLine 2"`,
			Expected: Program(
				Var("MULTILINE", "Line 1\nLine 2"),
			),
		},
		{
			Name:  "variable with tab in string",
			Input: `var TABBED = "Col1\tCol2"`,
			Expected: Program(
				Var("TABBED", "Col1\tCol2"),
			),
		},
		{
			Name:  "multiple durations with different units",
			Input: `var (
  NANO = 500ns
  MICRO = 250us
  MILLI = 100ms
  SEC = 30s
  MIN = 5m
  HOUR = 2h
)`,
			Expected: Program(
				Var("NANO", DurationExpr("500ns")),
				Var("MICRO", DurationExpr("250us")),
				Var("MILLI", DurationExpr("100ms")),
				Var("SEC", DurationExpr("30s")),
				Var("MIN", DurationExpr("5m")),
				Var("HOUR", DurationExpr("2h")),
			),
		},
		{
			Name:  "zero values",
			Input: `var (
  ZERO_NUM = 0
  ZERO_DUR = 0s
  EMPTY_STR = ""
  FALSE_BOOL = false
)`,
			Expected: Program(
				Var("ZERO_NUM", "0"),
				Var("ZERO_DUR", DurationExpr("0s")),
				Var("EMPTY_STR", ""),
				Var("FALSE_BOOL", BooleanExpr(false)),
			),
		},
		{
			Name:  "variable with JSON string",
			Input: `var CONFIG = '{"host": "localhost", "port": 8080}'`,
			Expected: Program(
				Var("CONFIG", `{"host": "localhost", "port": 8080}`),
			),
		},
		{
			Name:  "variable with regex pattern",
			Input: `var PATTERN = "^[a-zA-Z0-9]+$"`,
			Expected: Program(
				Var("PATTERN", "^[a-zA-Z0-9]+$"),
			),
		},
		{
			Name:  "variable with shell special characters",
			Input: `var SHELL_CMD = "echo $HOME && ls -la | grep .txt"`,
			Expected: Program(
				Var("SHELL_CMD", "echo $HOME && ls -la | grep .txt"),
			),
		},
		// More error cases
		{
			Name:        "error: variable with decorator as value",
			Input:       "var FUNC = @var(OTHER)",
			WantErr:     true,
			ErrorSubstr: "variable value must be a quoted string, number, duration, or boolean literal",
		},
		{
			Name:        "error: variable with shell command as value",
			Input:       "var OUTPUT = $(echo hello)",
			WantErr:     true,
			ErrorSubstr: "variable value must be a quoted string, number, duration, or boolean literal",
		},
		{
			Name:        "error: variable with arithmetic expression",
			Input:       "var CALC = 1 + 2",
			WantErr:     true,
			ErrorSubstr: "variable value must be a quoted string, number, duration, or boolean literal",
		},
		{
			Name:        "error: variable with concatenation",
			Input:       "var CONCAT = hello + world",
			WantErr:     true,
			ErrorSubstr: "variable value must be a quoted string, number, duration, or boolean literal",
		},
		{
			Name:        "error: variable with environment variable reference",
			Input:       "var HOME_DIR = $HOME",
			WantErr:     true,
			ErrorSubstr: "variable value must be a quoted string, number, duration, or boolean literal",
		},
		// Edge cases that should work
		{
			Name:  "variable names that look like keywords",
			Input: `var (
  var = "variable"
  watch = "watcher"
  stop = "stopper"
  when = "whenever"
  try = "trying"
)`,
			Expected: Program(
				Var("var", "variable"),
				Var("watch", "watcher"),
				Var("stop", "stopper"),
				Var("when", "whenever"),
				Var("try", "trying"),
			),
		},
		{
			Name:  "very large number",
			Input: "var BIG_NUM = 9223372036854775807",
			Expected: Program(
				Var("BIG_NUM", "9223372036854775807"),
			),
		},
		{
			Name:  "scientific notation number",
			Input: "var SCI_NUM = 1.23e-4",
			Expected: Program(
				Var("SCI_NUM", "1.23e-4"),
			),
		},
		{
			Name:  "hexadecimal number (as string)",
			Input: `var HEX = "0xFF00"`,
			Expected: Program(
				Var("HEX", "0xFF00"),
			),
		},
		{
			Name:  "string that looks like boolean",
			Input: `var NOT_BOOL = "true story"`,
			Expected: Program(
				Var("NOT_BOOL", "true story"),
			),
		},
		{
			Name:  "string that looks like number",
			Input: `var NOT_NUM = "123 Main St"`,
			Expected: Program(
				Var("NOT_NUM", "123 Main St"),
			),
		},
		{
			Name:  "string that looks like duration",
			Input: `var NOT_DUR = "30 seconds"`,
			Expected: Program(
				Var("NOT_DUR", "30 seconds"),
			),
		},
		{
			Name:  "variable with unicode in name",
			Input: `var HELLO_世界 = "Hello World"`,
			Expected: Program(
				Var("HELLO_世界", "Hello World"),
			),
		},
		{
			Name:  "variable with unicode in value",
			Input: `var GREETING = "Hello 世界"`,
			Expected: Program(
				Var("GREETING", "Hello 世界"),
			),
		},
		{
			Name:  "variable with emoji",
			Input: `var EMOJI = "🚀 Launch"`,
			Expected: Program(
				Var("EMOJI", "🚀 Launch"),
			),
		},
	}

	for _, tc := range testCases {
		RunTestCase(t, tc)
	}
}

// Helper to create an identifier expression for clarity in tests
func identifier(val string) ExpectedExpression {
	return ExpectedExpression{Type: "identifier", Value: val}
}
