package parser

import (
	"testing"
)

func TestParseErrors(t *testing.T) {
	testCases := []TestCase{
		{
			Name:        "invalid variable syntax",
			Input:       "var = invalid",
			WantErr:     true,
			ErrorSubstr: "expected variable name",
		},
		{
			Name:        "unclosed block",
			Input:       "test: { echo hello",
			WantErr:     true,
			ErrorSubstr: "unclosed block",
		},
		{
			Name:        "invalid command name",
			Input:       "123invalid: echo hello",
			WantErr:     true,
			ErrorSubstr: "invalid command name",
		},
		{
			Name:        "missing colon",
			Input:       "build echo hello",
			WantErr:     true,
			ErrorSubstr: "expected ':'",
		},
		{
			Name:        "unclosed decorator parentheses",
			Input:       "test: @sh(echo hello",
			WantErr:     true,
			ErrorSubstr: "unclosed",
		},
		{
			Name:        "invalid decorator name",
			Input:       "test: @123invalid",
			WantErr:     true,
			ErrorSubstr: "invalid",
		},
		{
			Name:        "duplicate variable names",
			Input:       "var SRC = ./src\nvar SRC = ./other",
			WantErr:     true,
			ErrorSubstr: "duplicate",
		},
		{
			Name:        "invalid variable name starting with number",
			Input:       "var 123VAR = value",
			WantErr:     true,
			ErrorSubstr: "invalid variable name",
		},
		{
			Name:        "missing variable value",
			Input:       "var SRC =",
			WantErr:     true,
			ErrorSubstr: "missing variable value",
		},
		{
			Name:        "unclosed variable group",
			Input:       "var (\n  SRC = ./src",
			WantErr:     true,
			ErrorSubstr: "unclosed variable group",
		},
	}

	for _, tc := range testCases {
		RunTestCase(t, tc)
	}
}
