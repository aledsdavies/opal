package parser

import "testing"


func TestErrorHandling(t *testing.T) {
	testCases := []TestCase{
		{
			Name:        "missing colon",
			Input:       "build echo hello", // Missing colon
			WantErr:     true,
			ErrorSubstr: "expected ':'",
		},
		{
			Name:        "unclosed brace",
			Input:       "test: { echo hello",
			WantErr:     true,
			ErrorSubstr: "unclosed",
		},
		{
			Name:        "invalid command name starting with number",
			Input:       "123invalid: echo hello",
			WantErr:     true,
			ErrorSubstr: "invalid command name",
		},
		{
			Name:        "duplicate variable",
			Input:       "var SRC = ./src\nvar SRC = ./bin",
			WantErr:     true,
			ErrorSubstr: "duplicate variable", // Updated to match new error format
		},
		{
			Name:        "duplicate command of same type",
			Input:       "build: echo hello\nbuild: echo world",
			WantErr:     true,
			ErrorSubstr: "duplicate command", // Updated to match new error format
		},
		{
			Name:        "invalid variable name starting with number",
			Input:       "var 123INVALID = value",
			WantErr:     true,
			ErrorSubstr: "invalid variable name", // Updated to match actual error
		},
		{
			Name:        "unclosed string in variable",
			Input:       "var MSG = \"unclosed string",
			WantErr:     true,
			ErrorSubstr: "unclosed", // This should be caught by lexer
		},
		{
			Name:        "unclosed string in command",
			Input:       "test: echo \"unclosed string",
			WantErr:     true,
			ErrorSubstr: "unclosed", // This should be caught by lexer
		},
		{
			Name:        "invalid decorator syntax",
			Input:       "test: @invalid-decorator-name(arg)",
			WantErr:     true,
			ErrorSubstr: "invalid decorator",
		},
		{
			Name:        "unclosed decorator arguments",
			Input:       "test: @timeout(30s",
			WantErr:     true,
			ErrorSubstr: "unclosed",
		},
		{
			Name:        "missing variable name after var",
			Input:       "var = value",
			WantErr:     true,
			ErrorSubstr: "expected variable name",
		},
		{
			Name:        "missing equals in variable declaration",
			Input:       "var SRC value",
			WantErr:     true,
			ErrorSubstr: "expected '='",
		},
		{
			Name:        "undefined variable reference",
			Input:       "build: cd @var(UNDEFINED)",
			WantErr:     true,
			ErrorSubstr: "undefined variable",
		},
	}

	for _, tc := range testCases {
		runTestCase(t, tc)
	}
}
