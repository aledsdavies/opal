package generator

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"

	devcmdParser "github.com/aledsdavies/devcmd/pkgs/parser"
)

func TestPreprocessCommands(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectedData func(*TemplateData) bool
		expectError  bool
	}{
		{
			name:  "simple command",
			input: "build: go build ./...;",
			expectedData: func(data *TemplateData) bool {
				return len(data.Commands) == 1 &&
					data.Commands[0].Name == "build" &&
					data.Commands[0].FunctionName == "runBuild" &&
					data.Commands[0].Type == "regular" &&
					data.Commands[0].ShellCommand == "go build ./..." &&
					!data.HasProcessMgmt
			},
		},
		{
			name:  "watch command",
			input: "watch server: npm start;",
			expectedData: func(data *TemplateData) bool {
				return len(data.Commands) == 1 &&
					data.Commands[0].Name == "server" &&
					(data.Commands[0].Type == "watch-only" || data.Commands[0].Type == "watch") &&
					data.Commands[0].IsBackground &&
					data.HasProcessMgmt
			},
		},
		{
			name:  "stop command",
			input: "stop server: pkill node;",
			expectedData: func(data *TemplateData) bool {
				return len(data.Commands) == 1 &&
					data.Commands[0].Name == "server" &&
					(data.Commands[0].Type == "stop-only" || data.Commands[0].Type == "stop") &&
					!data.HasProcessMgmt // stop alone doesn't need process mgmt
			},
		},
		{
			name:  "hyphenated command name",
			input: "check-deps: which go;",
			expectedData: func(data *TemplateData) bool {
				return len(data.Commands) == 1 &&
					data.Commands[0].Name == "check-deps" &&
					data.Commands[0].FunctionName == "runCheckDeps"
			},
		},
		{
			name:  "watch-stop pair",
			input: "watch server: npm start;\nstop server: pkill node;",
			expectedData: func(data *TemplateData) bool {
				return len(data.Commands) == 1 &&
					data.Commands[0].Name == "server" &&
					(data.Commands[0].Type == "watch-stop" || strings.Contains(data.Commands[0].Type, "watch")) &&
					data.Commands[0].IsBackground &&
					data.HasProcessMgmt
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the input
			cf, err := devcmdParser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			// Preprocess commands
			data, err := PreprocessCommands(cf)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("PreprocessCommands error: %v", err)
			}

			if !tt.expectedData(data) {
				t.Errorf("Data validation failed for %s", tt.name)
				t.Logf("Commands: %+v", data.Commands)
				t.Logf("HasProcessMgmt: %v", data.HasProcessMgmt)
			}
		})
	}
}

func TestSanitizeFunctionName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"build", "runBuild"},
		{"check-deps", "runCheckDeps"},
		{"run-all", "runRunAll"},
		{"test_coverage", "runTestCoverage"},
		{"api-server-dev", "runApiServerDev"},
		{"", "runCommand"},
		{"kebab-case-command", "runKebabCaseCommand"},
		{"snake_case_command", "runSnakeCaseCommand"},
		{"CamelCase", "runCamelcase"},
		{"123-numeric", "run123Numeric"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeFunctionName(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeFunctionName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestBuildShellCommand(t *testing.T) {
	tests := []struct {
		name     string
		input    devcmdParser.Command
		expected string
	}{
		{
			name: "simple command",
			input: devcmdParser.Command{
				Command: "echo hello",
			},
			expected: "echo hello",
		},
		{
			name: "block command",
			input: devcmdParser.Command{
				IsBlock: true,
				Block: []devcmdParser.BlockStatement{
					{Command: "npm install", Background: false},
					{Command: "npm start", Background: true},
					{Command: "echo done", Background: false},
				},
			},
			expected: "npm install; npm start &; echo done",
		},
		{
			name: "block with all background",
			input: devcmdParser.Command{
				IsBlock: true,
				Block: []devcmdParser.BlockStatement{
					{Command: "server", Background: true},
					{Command: "client", Background: true},
				},
			},
			expected: "server &; client &",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildShellCommand(tt.input)
			if result != tt.expected {
				t.Errorf("buildShellCommand() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestGenerateGo_BasicCommands(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedInCode []string
		notInCode      []string
	}{
		{
			name:  "simple command",
			input: "build: go build ./...;",
			expectedInCode: []string{
				"func (c *CLI) runBuild(args []string)",
				`go build ./...`,
				`case "build":`,
				"c.runBuild(args)",
				"// Regular command",
			},
			notInCode: []string{
				"ProcessRegistry",
				"runInBackground",
				"syscall", // Should not import syscall for regular commands
			},
		},
		{
			name:  "command with POSIX parentheses",
			input: "check: (which go && echo \"found\") || echo \"not found\";",
			expectedInCode: []string{
				"func (c *CLI) runCheck(args []string)",
				`(which go && echo "found") || echo "not found"`,
				`case "check":`,
			},
			notInCode: []string{
				"syscall",
				"ProcessRegistry",
			},
		},
		{
			name:  "command with watch/stop keywords in text",
			input: "monitor: watch -n 1 \"ps aux\" && echo \"stop with Ctrl+C\";",
			expectedInCode: []string{
				"func (c *CLI) runMonitor(args []string)",
				`watch -n 1 "ps aux" && echo "stop with Ctrl+C"`,
			},
			notInCode: []string{
				"syscall",
				"ProcessRegistry",
			},
		},
		{
			name:  "hyphenated command name",
			input: "check-deps: which go || echo missing;",
			expectedInCode: []string{
				"func (c *CLI) runCheckDeps(args []string)",
				`case "check-deps":`, // Case should use original name
				"c.runCheckDeps(args)",
			},
			notInCode: []string{
				"syscall",
				"ProcessRegistry",
			},
		},
		{
			name:  "command with POSIX find and braces",
			input: "cleanup: find . -name \"*.tmp\" -exec rm {} \\;;",
			expectedInCode: []string{
				"func (c *CLI) runCleanup(args []string)",
				`find . -name "*.tmp" -exec rm {} \;`,
				`case "cleanup":`,
			},
			notInCode: []string{
				"syscall",
				"ProcessRegistry",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the input
			cf, err := devcmdParser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			// Generate Go code
			generated, err := GenerateGo(cf)
			if err != nil {
				t.Fatalf("GenerateGo error: %v", err)
			}

			// Verify generated code is valid Go
			if !isValidGoCode(t, generated) {
				t.Errorf("Generated code is not valid Go")
				t.Logf("Generated code:\n%s", generated)
				return
			}

			// Check expected content
			for _, expected := range tt.expectedInCode {
				if !strings.Contains(generated, expected) {
					t.Errorf("Generated code missing expected content: %q", expected)
				}
			}

			// Check that unwanted content is not present
			for _, notExpected := range tt.notInCode {
				if strings.Contains(generated, notExpected) {
					t.Errorf("Generated code contains unwanted content: %q", notExpected)
				}
			}
		})
	}
}

func TestGenerateGo_WatchStopCommands(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedInCode []string
		notInCode      []string
	}{
		{
			name:  "simple watch command",
			input: "watch server: npm start;",
			expectedInCode: []string{
				"ProcessRegistry",
				"runInBackground",
				"func (c *CLI) runServer(args []string)",
				`npm start`,
				`case "server":`,
				"syscall", // Watch commands should include syscall
			},
		},
		{
			name:  "simple stop command",
			input: "stop server: pkill node;",
			expectedInCode: []string{
				"func (c *CLI) runServer(args []string)",
				`pkill node`,
			},
			notInCode: []string{
				"ProcessRegistry", // No watch commands means no process management
				"syscall",         // Stop-only commands don't need syscall
			},
		},
		{
			name:  "watch and stop pair",
			input: "watch api: go run main.go;\nstop api: pkill -f main.go;",
			expectedInCode: []string{
				"ProcessRegistry", // Should have ProcessRegistry due to watch command
				"func (c *CLI) runApi(args []string)",
				"go run main.go",
				"pkill -f main.go",
				"syscall", // Watch/stop pairs need syscall
			},
		},
		{
			name:  "watch command with parentheses",
			input: "watch dev: (cd src && npm start);",
			expectedInCode: []string{
				"ProcessRegistry",
				"runInBackground",
				`(cd src && npm start)`,
				"syscall",
			},
		},
		{
			name:  "watch command with POSIX find and braces",
			input: "watch cleanup: find . -name \"*.tmp\" -exec rm {} \\;;",
			expectedInCode: []string{
				"ProcessRegistry",
				"runInBackground",
				`find . -name "*.tmp" -exec rm {} \;`,
				"syscall",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the input
			cf, err := devcmdParser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			// Generate Go code
			generated, err := GenerateGo(cf)
			if err != nil {
				t.Fatalf("GenerateGo error: %v", err)
			}

			// Verify generated code is valid Go
			if !isValidGoCode(t, generated) {
				t.Errorf("Generated code is not valid Go")
				t.Logf("Generated code:\n%s", generated)
				return
			}

			// Check expected content
			for _, expected := range tt.expectedInCode {
				if !strings.Contains(generated, expected) {
					t.Errorf("Generated code missing expected content: %q", expected)
				}
			}

			// Check that unwanted content is not present
			for _, notInCode := range tt.notInCode {
				if strings.Contains(generated, notInCode) {
					t.Errorf("Generated code contains unwanted content: %q", notInCode)
				}
			}
		})
	}
}

func TestGenerateGo_BlockCommands(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedInCode []string
		notInCode      []string
	}{
		{
			name:  "simple block command",
			input: "setup: { npm install; go mod tidy; echo done }",
			expectedInCode: []string{
				"func (c *CLI) runSetup(args []string)",
				"npm install; go mod tidy; echo done",
			},
			notInCode: []string{
				"syscall",
				"ProcessRegistry",
			},
		},
		{
			name:  "block with background processes",
			input: "run-all: { server &; client &; monitor }",
			expectedInCode: []string{
				"func (c *CLI) runRunAll(args []string)",
				"server &; client &; monitor",
			},
			notInCode: []string{
				"syscall",
				"ProcessRegistry",
			},
		},
		{
			name:  "watch block command",
			input: "watch services: { server &; worker &; echo \"started\" }",
			expectedInCode: []string{
				"ProcessRegistry",
				"server &; worker &; echo \"started\"",
				"runInBackground",
				"syscall",
			},
		},
		{
			name:  "block with parentheses and complex syntax",
			input: "parallel: { (task1 && echo \"done1\") &; (task2 || echo \"failed2\") }",
			expectedInCode: []string{
				`(task1 && echo "done1") &; (task2 || echo "failed2")`,
			},
			notInCode: []string{
				"syscall",
				"ProcessRegistry",
			},
		},
		{
			name:  "block with POSIX find and braces",
			input: "cleanup: { find . -name \"*.tmp\" -exec rm {} \\;; echo \"cleanup done\" }",
			expectedInCode: []string{
				`find . -name "*.tmp" -exec rm {} \;`,
				`echo "cleanup done"`,
			},
			notInCode: []string{
				"syscall",
				"ProcessRegistry",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the input
			cf, err := devcmdParser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			// Generate Go code
			generated, err := GenerateGo(cf)
			if err != nil {
				t.Fatalf("GenerateGo error: %v", err)
			}

			// Verify generated code is valid Go
			if !isValidGoCode(t, generated) {
				t.Errorf("Generated code is not valid Go")
				t.Logf("Generated code:\n%s", generated)
				return
			}

			// Check expected content
			for _, expected := range tt.expectedInCode {
				if !strings.Contains(generated, expected) {
					t.Errorf("Generated code missing expected content: %q", expected)
				}
			}

			// Check that unwanted content is not present
			for _, notInCode := range tt.notInCode {
				if strings.Contains(generated, notInCode) {
					t.Errorf("Generated code contains unwanted content: %q", notInCode)
				}
			}
		})
	}
}

func TestGenerateGo_VariableHandling(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedInCode []string
		notInCode      []string
	}{
		{
			name: "commands with variables",
			input: `def SRC = ./src;
def PORT = 8080;
build: cd $(SRC) && go build;
start: go run $(SRC) --port=$(PORT);`,
			expectedInCode: []string{
				"func (c *CLI) runBuild(args []string)",
				"func (c *CLI) runStart(args []string)",
				"cd ./src && go build",
				"go run ./src --port=8080",
			},
			notInCode: []string{
				"syscall",
				"ProcessRegistry",
			},
		},
		{
			name: "variables with parentheses",
			input: `def CHECK = (which go || echo "missing");
validate: $(CHECK) && echo "ok";`,
			expectedInCode: []string{
				`(which go || echo "missing") && echo "ok"`,
			},
			notInCode: []string{
				"syscall",
				"ProcessRegistry",
			},
		},
		{
			name: "variables with POSIX find and braces",
			input: `def PATTERN = "*.tmp";
cleanup: find . -name $(PATTERN) -exec rm {} \;;`,
			expectedInCode: []string{
				`find . -name "*.tmp" -exec rm {} \;`,
			},
			notInCode: []string{
				"syscall",
				"ProcessRegistry",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the input
			cf, err := devcmdParser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			// Expand variables
			err = cf.ExpandVariables()
			if err != nil {
				t.Fatalf("ExpandVariables error: %v", err)
			}

			// Generate Go code
			generated, err := GenerateGo(cf)
			if err != nil {
				t.Fatalf("GenerateGo error: %v", err)
			}

			// Verify generated code is valid Go
			if !isValidGoCode(t, generated) {
				t.Errorf("Generated code is not valid Go")
				t.Logf("Generated code:\n%s", generated)
				return
			}

			// Check expected content
			for _, expected := range tt.expectedInCode {
				if !strings.Contains(generated, expected) {
					t.Errorf("Generated code missing expected content: %q", expected)
				}
			}

			// Check that unwanted content is not present
			for _, notInCode := range tt.notInCode {
				if strings.Contains(generated, notInCode) {
					t.Errorf("Generated code contains unwanted content: %q", notInCode)
				}
			}
		})
	}
}

func TestBasicDevExample_NoSyscall(t *testing.T) {
	// This tests the specific case mentioned by the user - basicDev shouldn't get syscall
	basicDevCommands := `
# Basic development commands
def SRC = ./src;
def BUILD_DIR = ./build;

build: {
  echo "Building project...";
  mkdir -p $(BUILD_DIR);
  (cd $(SRC) && make) || echo "No Makefile found"
}

test: {
  echo "Running tests...";
  (cd $(SRC) && make test) || go test ./... || npm test || echo "No tests found"
}

clean: {
  echo "Cleaning build artifacts...";
  rm -rf $(BUILD_DIR);
  find . -name "*.tmp" -delete;
  echo "Clean complete"
}

lint: {
  echo "Running linters...";
  (which golangci-lint && golangci-lint run) || echo "No Go linter";
  (which eslint && eslint .) || echo "No JS linter";
  echo "Linting complete"
}

deps: {
  echo "Installing dependencies...";
  (test -f go.mod && go mod download) || echo "No Go modules";
  (test -f package.json && npm install) || echo "No NPM packages";
  (test -f requirements.txt && pip install -r requirements.txt) || echo "No Python packages";
  echo "Dependencies installed"
}
`

	// Parse the input
	cf, err := devcmdParser.Parse(basicDevCommands)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	// Expand variables
	err = cf.ExpandVariables()
	if err != nil {
		t.Fatalf("ExpandVariables error: %v", err)
	}

	// Generate Go code
	generated, err := GenerateGo(cf)
	if err != nil {
		t.Fatalf("GenerateGo error: %v", err)
	}

	// Verify generated code is valid Go - this is the main compile check
	if !isValidGoCode(t, generated) {
		t.Errorf("Generated code is not valid Go")
		t.Logf("Generated code:\n%s", generated)
		return
	}

	// These should be present (basic functionality)
	expectedContent := []string{
		"func (c *CLI) runBuild(args []string)",
		"func (c *CLI) runTest(args []string)",
		"func (c *CLI) runClean(args []string)",
		"func (c *CLI) runLint(args []string)",
		"func (c *CLI) runDeps(args []string)",
		`"fmt"`,
		`"os"`,
		`"os/exec"`,
		// Variable expansions
		"./src",
		"./build",
	}

	for _, expected := range expectedContent {
		if !strings.Contains(generated, expected) {
			t.Errorf("Generated code missing expected content: %q", expected)
		}
	}

	// These should NOT be present (no watch commands)
	unwantedContent := []string{
		`"syscall"`,
		`"encoding/json"`,
		`"os/signal"`,
		`"time"`,
		"ProcessRegistry",
		"runInBackground",
		"gracefulStop",
	}

	for _, unwanted := range unwantedContent {
		if strings.Contains(generated, unwanted) {
			t.Errorf("Generated code contains unwanted content: %q", unwanted)
		}
	}
}

func TestImportHandling(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		shouldHave    []string
		shouldNotHave []string
	}{
		{
			name:  "regular commands only - minimal imports",
			input: "build: go build;\ntest: go test;\nclean: rm -rf dist;",
			shouldHave: []string{
				`"fmt"`,
				`"os"`,
				`"os/exec"`,
			},
			shouldNotHave: []string{
				`"syscall"`,
				`"encoding/json"`,
				`"os/signal"`,
				`"time"`,
				"ProcessRegistry",
			},
		},
		{
			name:  "watch commands - full imports",
			input: "watch server: npm start;",
			shouldHave: []string{
				`"fmt"`,
				`"os"`,
				`"os/exec"`,
				`"syscall"`,
				"ProcessRegistry",
			},
			shouldNotHave: []string{}, // All imports should be present
		},
		{
			name:  "mixed commands - full imports due to watch",
			input: "build: go build;\nwatch dev: npm start;",
			shouldHave: []string{
				`"fmt"`,
				`"os"`,
				`"os/exec"`,
				`"syscall"`,
				"ProcessRegistry",
			},
			shouldNotHave: []string{}, // All imports should be present due to watch command
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the input
			cf, err := devcmdParser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			// Generate Go code
			generated, err := GenerateGo(cf)
			if err != nil {
				t.Fatalf("GenerateGo error: %v", err)
			}

			// Verify generated code is valid Go - MAIN COMPILE CHECK
			if !isValidGoCode(t, generated) {
				t.Errorf("Generated code is not valid Go")
				t.Logf("Generated code:\n%s", generated)
				return
			}

			// Check expected imports/features
			for _, expected := range tt.shouldHave {
				if !strings.Contains(generated, expected) {
					t.Errorf("Generated code missing expected import/feature: %q", expected)
				}
			}

			// Check that unwanted imports/features are not present
			for _, notExpected := range tt.shouldNotHave {
				if strings.Contains(generated, notExpected) {
					t.Errorf("Generated code contains unwanted import/feature: %q", notExpected)
				}
			}
		})
	}
}

func TestGenerateGo_ErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		input       *devcmdParser.CommandFile
		template    *string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "nil command file",
			input:       nil,
			expectError: true,
			errorMsg:    "command file cannot be nil",
		},
		{
			name:        "empty template string",
			input:       &devcmdParser.CommandFile{},
			template:    stringPtr(""),
			expectError: true,
			errorMsg:    "template string cannot be empty",
		},
		{
			name:        "whitespace-only template",
			input:       &devcmdParser.CommandFile{},
			template:    stringPtr("   \n\t  "),
			expectError: true,
			errorMsg:    "template string cannot be empty",
		},
		{
			name: "invalid template syntax",
			input: &devcmdParser.CommandFile{
				Commands: []devcmdParser.Command{
					{Name: "test", Command: "echo test"},
				},
			},
			template:    stringPtr("{{.InvalidField"),
			expectError: true,
			errorMsg:    "failed to parse template",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error

			if tt.template != nil {
				_, err = GenerateGoWithTemplate(tt.input, *tt.template)
			} else {
				_, err = GenerateGo(tt.input)
			}

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestGenerateGo_EdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "empty command file",
			input: "",
		},
		{
			name:  "only definitions",
			input: "def VAR = value;",
		},
		{
			name:  "command with special characters",
			input: `special: echo "quotes" && echo 'single' && echo \$escaped;`,
		},
		{
			name:  "command with unicode",
			input: "unicode: echo \"Hello 世界\";",
		},
		{
			name:  "command with POSIX find and braces",
			input: "cleanup: find . -name \"*.tmp\" -exec rm {} \\;;",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the input
			cf, err := devcmdParser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			// Generate Go code
			generated, err := GenerateGo(cf)
			if err != nil {
				t.Fatalf("GenerateGo error: %v", err)
			}

			// Verify generated code is valid Go - MAIN COMPILE CHECK
			if !isValidGoCode(t, generated) {
				t.Errorf("Generated code is not valid Go")
				t.Logf("Generated code:\n%s", generated)
				return
			}

			// Basic structure should always be present
			expectedStructure := []string{
				"package main",
				"func main()",
				"cli := NewCLI()",
				"cli.Execute()",
			}

			for _, expected := range expectedStructure {
				if !strings.Contains(generated, expected) {
					t.Errorf("Generated code missing expected structure: %q", expected)
				}
			}
		})
	}
}

// Helper function to check if generated code is valid Go - THE KEY FUNCTION
func isValidGoCode(t *testing.T, code string) bool {
	fset := token.NewFileSet()
	_, err := parser.ParseFile(fset, "generated.go", code, parser.ParseComments)
	if err != nil {
		t.Logf("Go parsing error: %v", err)
		return false
	}
	return true
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}

// Benchmark tests for performance
func BenchmarkGenerateGo_SimpleCommand(b *testing.B) {
	cf := &devcmdParser.CommandFile{
		Commands: []devcmdParser.Command{
			{Name: "build", Command: "go build ./..."},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := GenerateGo(cf)
		if err != nil {
			b.Fatalf("GenerateGo error: %v", err)
		}
	}
}

func BenchmarkPreprocessCommands(b *testing.B) {
	cf := &devcmdParser.CommandFile{
		Commands: []devcmdParser.Command{
			{Name: "build", Command: "go build"},
			{Name: "test", Command: "go test"},
			{Name: "watch-server", Command: "npm start", IsWatch: true},
			{Name: "stop-server", Command: "pkill node", IsStop: true},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := PreprocessCommands(cf)
		if err != nil {
			b.Fatalf("PreprocessCommands error: %v", err)
		}
	}
}
