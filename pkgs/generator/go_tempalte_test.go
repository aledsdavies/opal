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
			input: "build: go build ./...",
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
			input: "watch server: npm start",
			expectedData: func(data *TemplateData) bool {
				return len(data.Commands) == 1 &&
					data.Commands[0].Name == "server" &&
					data.Commands[0].Type == "watch" &&
					data.Commands[0].IsBackground &&
					data.HasProcessMgmt
			},
		},
		{
			name:  "stop command",
			input: "stop server: pkill node",
			expectedData: func(data *TemplateData) bool {
				return len(data.Commands) == 1 &&
					data.Commands[0].Name == "server" &&
					data.Commands[0].Type == "stop" &&
					data.Commands[0].BaseName == "server" &&
					!data.HasProcessMgmt // stop alone doesn't need process mgmt
			},
		},
		{
			name:  "hyphenated command name",
			input: "check-deps: which go",
			expectedData: func(data *TemplateData) bool {
				return len(data.Commands) == 1 &&
					data.Commands[0].Name == "check-deps" &&
					data.Commands[0].FunctionName == "runCheckDeps"
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

func TestExtractBaseName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"server", "server"},
		{"stop-server", "server"},
		{"stop_api", "api"},
		{"api-stop", "api-stop"}, // doesn't start with stop
		{"stopwatch", "watch"},   // edge case
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := extractBaseName(tt.input)
			if result != tt.expected {
				t.Errorf("extractBaseName(%q) = %q, want %q", tt.input, result, tt.expected)
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
			input: "build: go build ./...",
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
				"HasProcessMgmt", // This shouldn't appear in generated code
			},
		},
		{
			name:  "command with POSIX parentheses",
			input: "check: (which go && echo \"found\") || echo \"not found\"",
			expectedInCode: []string{
				"func (c *CLI) runCheck(args []string)",
				`(which go && echo "found") || echo "not found"`,
				`case "check":`,
			},
		},
		{
			name:  "command with watch/stop keywords in text",
			input: "monitor: watch -n 1 \"ps aux\" && echo \"stop with Ctrl+C\"",
			expectedInCode: []string{
				"func (c *CLI) runMonitor(args []string)",
				`watch -n 1 "ps aux" && echo "stop with Ctrl+C"`,
			},
		},
		{
			name:  "hyphenated command name",
			input: "check-deps: which go || echo missing",
			expectedInCode: []string{
				"func (c *CLI) runCheckDeps(args []string)",
				`case "check-deps":`, // Case should use original name
				"c.runCheckDeps(args)",
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
			input: "watch server: npm start",
			expectedInCode: []string{
				"ProcessRegistry",
				"runInBackground",
				"func (c *CLI) runServer(args []string)",
				"// Watch command - run in background",
				`npm start`,
				`case "server":`,
			},
		},
		{
			name:  "simple stop command",
			input: "stop server: pkill node",
			expectedInCode: []string{
				"func (c *CLI) runServer(args []string)",
				"// Stop command - terminate associated processes",
				`pkill node`,
				`fmt.Printf("No background process named '%s' to stop\n", baseName)`,
			},
			notInCode: []string{
				"ProcessRegistry", // No watch commands means no process management
			},
		},
		{
			name:  "watch and stop pair",
			input: "watch api: go run main.go\nstop api: pkill -f main.go",
			expectedInCode: []string{
				"ProcessRegistry", // Should have ProcessRegistry due to watch command
				"func (c *CLI) runApi(args []string)",
				"// Watch command - run in background",
				"// Stop command - terminate associated processes",
				"go run main.go",
				"pkill -f main.go",
				"c.stopCommand(baseName)", // Should have this due to HasProcessMgmt
			},
		},
		{
			name:  "watch command with parentheses",
			input: "watch dev: (cd src && npm start)",
			expectedInCode: []string{
				"ProcessRegistry",
				"runInBackground",
				`(cd src && npm start)`,
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
	}{
		{
			name:  "simple block command",
			input: "setup: { npm install; go mod tidy; echo done }",
			expectedInCode: []string{
				"func (c *CLI) runSetup(args []string)",
				"npm install; go mod tidy; echo done",
			},
		},
		{
			name:  "block with background processes",
			input: "run-all: { server &; client &; monitor }",
			expectedInCode: []string{
				"func (c *CLI) runRunAll(args []string)",
				"server &; client &; monitor",
			},
		},
		{
			name:  "watch block command",
			input: "watch services: { server &; worker &; echo \"started\" }",
			expectedInCode: []string{
				"ProcessRegistry",
				"// Watch command - run in background",
				"server &; worker &; echo \"started\"",
				"runInBackground",
			},
		},
		{
			name:  "block with parentheses and complex syntax",
			input: "parallel: { (task1 && echo \"done1\") &; (task2 || echo \"failed2\") }",
			expectedInCode: []string{
				`(task1 && echo "done1") &; (task2 || echo "failed2")`,
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
			}

			// Check expected content
			for _, expected := range tt.expectedInCode {
				if !strings.Contains(generated, expected) {
					t.Errorf("Generated code missing expected content: %q", expected)
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
	}{
		{
			name: "commands with variables",
			input: `def SRC = ./src;
def PORT = 8080;
build: cd $(SRC) && go build
start: go run $(SRC) --port=$(PORT)`,
			expectedInCode: []string{
				"func (c *CLI) runBuild(args []string)",
				"func (c *CLI) runStart(args []string)",
				"cd ./src && go build",
				"go run ./src --port=8080",
			},
		},
		{
			name: "variables with parentheses",
			input: `def CHECK = (which go || echo "missing");
validate: $(CHECK) && echo "ok"`,
			expectedInCode: []string{
				`(which go || echo "missing") && echo "ok"`,
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
			}

			// Check expected content
			for _, expected := range tt.expectedInCode {
				if !strings.Contains(generated, expected) {
					t.Errorf("Generated code missing expected content: %q", expected)
				}
			}
		})
	}
}

func TestGenerateGo_ComplexScenarios(t *testing.T) {
	complexInput := `
# Complex devcmd file with all features
def SRC = ./src;
def PORT = 8080;

# Simple command with parentheses
check-deps: (which go && echo "Go found") || (echo "Go missing" && exit 1)

# Watch command with variables and background processes
watch server: {
  cd $(SRC);
  go run main.go --port=$(PORT) &;
  echo "Server started on port $(PORT)"
}

# Stop command with complex cleanup
stop server: {
  pkill -f "main.go" || echo "No server running";
  echo "Cleanup done"
}

# Block command with mixed syntax
deploy: {
  echo "Building...";
  (cd $(SRC) && go build -o ../build/app) &;
  wait;
  echo "Deployment ready"
}
`

	t.Run("complex scenario", func(t *testing.T) {
		// Parse the input
		cf, err := devcmdParser.Parse(complexInput)
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
		}

		expectedContent := []string{
			// Process management for watch commands
			"ProcessRegistry",
			"runInBackground",

			// All command functions (sanitized names)
			"func (c *CLI) runCheckDeps(args []string)",
			"func (c *CLI) runServer(args []string)",
			"func (c *CLI) runDeploy(args []string)",

			// Expanded variables
			"cd ./src",
			"--port=8080",

			// Complex POSIX syntax preservation
			`(which go && echo "Go found") || (echo "Go missing" && exit 1)`,
			`(cd ./src && go build -o ../build/app) &`,

			// Command type handling
			"// Watch command - run in background",
			"// Stop command - terminate associated processes",

			// Case statements (original names)
			`case "check-deps":`,
			`case "server":`,
			`case "deploy":`,
		}

		for _, expected := range expectedContent {
			if !strings.Contains(generated, expected) {
				t.Errorf("Generated code missing expected content: %q", expected)
			}
		}
	})
}

func TestGenerateGo_ErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		input       *devcmdParser.CommandFile
		template    *string // Use pointer to distinguish between nil and empty string
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
			template:    stringPtr(""), // Empty string
			expectError: true,
			errorMsg:    "template string cannot be empty",
		},
		{
			name:        "whitespace-only template",
			input:       &devcmdParser.CommandFile{},
			template:    stringPtr("   \n\t  "), // Whitespace only
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

func TestGenerateGo_CustomTemplate(t *testing.T) {
	customTemplate := `package {{.PackageName}}

import "fmt"

func main() {
{{range .Commands}}	fmt.Println("Command: {{.Name}}")
{{end}}}
`

	cf := &devcmdParser.CommandFile{
		Commands: []devcmdParser.Command{
			{Name: "build", Command: "go build"},
			{Name: "test", Command: "go test"},
		},
	}

	generated, err := GenerateGoWithTemplate(cf, customTemplate)
	if err != nil {
		t.Fatalf("GenerateGoWithTemplate error: %v", err)
	}

	expectedContent := []string{
		"package main",
		`fmt.Println("Command: build")`,
		`fmt.Println("Command: test")`,
	}

	for _, expected := range expectedContent {
		if !strings.Contains(generated, expected) {
			t.Errorf("Generated code missing expected content: %q", expected)
		}
	}

	// Verify it's valid Go
	if !isValidGoCode(t, generated) {
		t.Errorf("Generated code is not valid Go")
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
			input: `special: echo "quotes" && echo 'single' && echo \$escaped`,
		},
		{
			name:  "command with unicode",
			input: "unicode: echo \"Hello 世界\"",
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

// Helper function to check if generated code is valid Go
func isValidGoCode(t *testing.T, code string) bool {
	fset := token.NewFileSet()
	_, err := parser.ParseFile(fset, "generated.go", code, parser.ParseComments)
	if err != nil {
		t.Logf("Go parsing error: %v", err)
		t.Logf("Generated code:\n%s", code)
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
