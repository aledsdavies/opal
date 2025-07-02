package parser

import (
	"testing"
)

func TestVariableDefinitions(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "simple variable",
			Input: "var SRC = ./src",
			Expected: Program(
				Var("SRC", "./src"),
			),
		},
		{
			Name:  "variable with complex value",
			Input: "var CMD = go test -v ./...",
			Expected: Program(
				Var("CMD", "go test -v ./..."),
			),
		},
		{
			Name:  "multiple variables",
			Input: "var SRC = ./src\nvar BIN = ./bin",
			Expected: Program(
				Var("SRC", "./src"),
				Var("BIN", "./bin"),
			),
		},
		{
			Name:  "grouped variables",
			Input: "var (\n  SRC = ./src\n  BIN = ./bin\n)",
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
			Name:  "variable with quoted string",
			Input: "var MESSAGE = \"Hello, World!\"",
			Expected: Program(
				Var("MESSAGE", "Hello, World!"),
			),
		},
		{
			Name:  "variable with special characters",
			Input: "var API_URL = https://api.example.com/v1",
			Expected: Program(
				Var("API_URL", "https://api.example.com/v1"),
			),
		},
		{
			Name:  "mixed variable types in group",
			Input: "var (\n  SRC = ./src\n  PORT = 3000\n  TIMEOUT = 5m\n  DEBUG = true\n)",
			Expected: Program(
				Var("SRC", "./src"),
				Var("PORT", "3000"),
				Var("TIMEOUT", DurationExpr("5m")),
				Var("DEBUG", "true"),
			),
		},
		{
			Name:  "variable with environment-style name",
			Input: "var NODE_ENV = production",
			Expected: Program(
				Var("NODE_ENV", "production"),
			),
		},
		{
			Name:  "variable with special characters in value",
			Input: "var API_URL = https://api.example.com/v1?key=abc123",
			Expected: Program(
				Var("API_URL", "https://api.example.com/v1?key=abc123"),
			),
		},
		{
			Name:  "variable with boolean-like value",
			Input: "var DEBUG = true",
			Expected: Program(
				Var("DEBUG", "true"),
			),
		},
		{
			Name:  "variable with path containing spaces",
			Input: "var PROJECT_PATH = \"/path/with spaces/project\"",
			Expected: Program(
				Var("PROJECT_PATH", "/path/with spaces/project"),
			),
		},
		{
			Name:  "variable with empty string value",
			Input: "var EMPTY = \"\"",
			Expected: Program(
				Var("EMPTY", ""),
			),
		},
		{
			Name:  "variable with numeric string",
			Input: "var VERSION = \"1.2.3\"",
			Expected: Program(
				Var("VERSION", "1.2.3"),
			),
		},
		{
			Name:  "variable with complex file path",
			Input: "var CONFIG_FILE = /etc/myapp/config.json",
			Expected: Program(
				Var("CONFIG_FILE", "/etc/myapp/config.json"),
			),
		},
		{
			Name:  "variable with URL containing port",
			Input: "var DATABASE_URL = postgresql://user:pass@localhost:5432/dbname",
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
				Var("HOST", "localhost"),
				Var("TIMEOUT", DurationExpr("30s")),
				Var("DEBUG", "true"),
			),
		},
		{
			Name:  "variable with quoted identifier value",
			Input: "var MODE = \"production\"",
			Expected: Program(
				Var("MODE", "production"),
			),
		},
		{
			Name:  "variable with underscores",
			Input: "var API_BASE_URL = https://api.example.com",
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
			Name:  "variables with command usage",
			Input: "var SRC = ./src\nvar DEST = ./dist\nbuild: cp -r @var(SRC)/* @var(DEST)/",
			Expected: Program(
				Var("SRC", "./src"),
				Var("DEST", "./dist"),
				Cmd("build", Simple(
					Text("cp -r "),
					At("var", "SRC"),
					Text("/* "),
					At("var", "DEST"),
					Text("/"),
				)),
			),
		},
		{
			Name:  "grouped variables with usage",
			Input: "var (\n  PORT = 8080\n  HOST = localhost\n)\nserve: go run main.go --port=@var(PORT) --host=@var(HOST)",
			Expected: Program(
				Var("PORT", "8080"),
				Var("HOST", "localhost"),
				Cmd("serve", Simple(
					Text("go run main.go --port="),
					At("var", "PORT"),
					Text(" --host="),
					At("var", "HOST"),
				)),
			),
		},
		{
			Name:  "variables in block commands",
			Input: "var SRC = ./src\ndeploy: { cd @var(SRC); make clean; make install }",
			Expected: Program(
				Var("SRC", "./src"),
				Cmd("deploy", Block(
					Statement(Text("cd "), At("var", "SRC"), Text("; make clean; make install")),
				)),
			),
		},
		{
			Name:  "variables in decorator arguments",
			Input: "var TIMEOUT = 30s\ntest: @timeout(@var(TIMEOUT)) { npm test }",
			Expected: Program(
				Var("TIMEOUT", DurationExpr("30s")),
				// Command body is an implicit block containing the decorator
				// The decorator owns the { npm test } block
				Cmd("test", Block(
					At("timeout", At("var", "TIMEOUT"), Block("npm test")),
				)),
			),
		},
	}

	for _, tc := range testCases {
		RunTestCase(t, tc)
	}
}

