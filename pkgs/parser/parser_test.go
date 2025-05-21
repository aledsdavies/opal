package parser

import (
	"strings"
	"testing"
)

func TestBasicParsing(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantCommand string
		wantName    string
		wantErr     bool
	}{
		{
			name:        "simple command",
			input:       "build: echo hello",
			wantCommand: "echo hello",
			wantName:    "build",
			wantErr:     false,
		},
		{
			name:        "command with arguments",
			input:       "test: go test -v ./...",
			wantCommand: "go test -v ./...",
			wantName:    "test",
			wantErr:     false,
		},
		{
			name:        "command with special characters",
			input:       "run: echo 'Hello, World!'",
			wantCommand: "echo 'Hello, World!'",
			wantName:    "run",
			wantErr:     false,
		},
		{
			name:        "command with empty content",
			input:       "noop: ",
			wantCommand: "",
			wantName:    "noop",
			wantErr:     false,
		},
		{
			name:        "command with trailing space",
			input:       "build: make all   ",
			wantCommand: "make all",
			wantName:    "build",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Parse(tt.input)

			// Check error expectation
			if (err != nil) != tt.wantErr {
				t.Fatalf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			// Ensure we have exactly one command
			if len(result.Commands) != 1 {
				t.Fatalf("Expected 1 command, got %d", len(result.Commands))
			}

			// Check command properties
			cmd := result.Commands[0]
			if cmd.Name != tt.wantName {
				t.Errorf("Command name = %q, want %q", cmd.Name, tt.wantName)
			}

			if cmd.Command != tt.wantCommand {
				t.Errorf("Command text = %q, want %q", cmd.Command, tt.wantCommand)
			}
		})
	}
}

func TestDefinitions(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantName  string
		wantValue string
		wantErr   bool
	}{
		{
			name:      "simple definition",
			input:     "def SRC = ./src;",
			wantName:  "SRC",
			wantValue: "./src",
			wantErr:   false,
		},
		{
			name:      "definition with complex value",
			input:     "def CMD = go test -v ./...;",
			wantName:  "CMD",
			wantValue: "go test -v ./...",
			wantErr:   false,
		},
		{
			name:      "definition with special chars",
			input:     "def PATH = /usr/local/bin:$PATH;",
			wantName:  "PATH",
			wantValue: "/usr/local/bin:$PATH",
			wantErr:   false,
		},
		{
			name:      "definition with quotes",
			input:     `def MSG = "Hello, World!";`,
			wantName:  "MSG",
			wantValue: `"Hello, World!"`,
			wantErr:   false,
		},
		{
			name:      "definition with empty value",
			input:     "def EMPTY = ;",
			wantName:  "EMPTY",
			wantValue: "",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Parse(tt.input)

			// Check error expectation
			if (err != nil) != tt.wantErr {
				t.Fatalf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			// Ensure we have exactly one definition
			if len(result.Definitions) != 1 {
				t.Fatalf("Expected 1 definition, got %d", len(result.Definitions))
			}

			// Check definition properties
			def := result.Definitions[0]
			if def.Name != tt.wantName {
				t.Errorf("Definition name = %q, want %q", def.Name, tt.wantName)
			}

			if def.Value != tt.wantValue {
				t.Errorf("Definition value = %q, want %q", def.Value, tt.wantValue)
			}
		})
	}
}

func TestBlockCommands(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		wantName       string
		wantBlockSize  int
		wantCommands   []string
		wantBackground []bool
		wantErr        bool
	}{
		{
			name:           "empty block",
			input:          "setup: { }",
			wantName:       "setup",
			wantBlockSize:  0,
			wantCommands:   []string{},
			wantBackground: []bool{},
			wantErr:        false,
		},
		{
			name:           "single statement block",
			input:          "setup: { npm install }",
			wantName:       "setup",
			wantBlockSize:  1,
			wantCommands:   []string{"npm install"},
			wantBackground: []bool{false},
			wantErr:        false,
		},
		{
			name:           "multiple statements",
			input:          "setup: { npm install; go mod tidy; echo done }",
			wantName:       "setup",
			wantBlockSize:  3,
			wantCommands:   []string{"npm install", "go mod tidy", "echo done"},
			wantBackground: []bool{false, false, false},
			wantErr:        false,
		},
		{
			name:           "multiline block",
			input:          "setup: {\n  npm install;\n  go mod tidy;\n  echo done\n}",
			wantName:       "setup",
			wantBlockSize:  3,
			wantCommands:   []string{"npm install", "go mod tidy", "echo done"},
			wantBackground: []bool{false, false, false},
			wantErr:        false,
		},
		{
			name:           "background processes",
			input:          "run-all: { server &; client &; db & }",
			wantName:       "run-all",
			wantBlockSize:  3,
			wantCommands:   []string{"server", "client", "db"},
			wantBackground: []bool{true, true, true},
			wantErr:        false,
		},
		{
			name:           "mixed background and foreground",
			input:          "run: { setup; server &; monitor }",
			wantName:       "run",
			wantBlockSize:  3,
			wantCommands:   []string{"setup", "server", "monitor"},
			wantBackground: []bool{false, true, false},
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Parse(tt.input)

			// Check error expectation
			if (err != nil) != tt.wantErr {
				t.Fatalf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			// Ensure we have exactly one command
			if len(result.Commands) != 1 {
				t.Fatalf("Expected 1 command, got %d", len(result.Commands))
			}

			// Check command properties
			cmd := result.Commands[0]
			if cmd.Name != tt.wantName {
				t.Errorf("Command name = %q, want %q", cmd.Name, tt.wantName)
			}

			if !cmd.IsBlock {
				t.Errorf("Expected IsBlock to be true")
			}

			if len(cmd.Block) != tt.wantBlockSize {
				t.Fatalf("Block size = %d, want %d", len(cmd.Block), tt.wantBlockSize)
			}

			// Check each statement in the block
			for i := 0; i < tt.wantBlockSize; i++ {
				if i >= len(cmd.Block) {
					t.Fatalf("Missing block statement %d", i)
				}

				stmt := cmd.Block[i]
				if stmt.Command != tt.wantCommands[i] {
					t.Errorf("Block[%d].Command = %q, want %q", i, stmt.Command, tt.wantCommands[i])
				}

				if stmt.Background != tt.wantBackground[i] {
					t.Errorf("Block[%d].Background = %v, want %v", i, stmt.Background, tt.wantBackground[i])
				}
			}
		})
	}
}

func TestWatchStopCommands(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantName  string
		wantWatch bool
		wantStop  bool
		wantText  string
		wantBlock bool
		wantErr   bool
	}{
		{
			name:      "simple watch command",
			input:     "watch server: npm start",
			wantName:  "server",
			wantWatch: true,
			wantStop:  false,
			wantText:  "npm start",
			wantBlock: false,
			wantErr:   false,
		},
		{
			name:      "simple stop command",
			input:     "stop server: pkill node",
			wantName:  "server",
			wantWatch: false,
			wantStop:  true,
			wantText:  "pkill node",
			wantBlock: false,
			wantErr:   false,
		},
		{
			name:      "watch command with block",
			input:     "watch dev: {\nnpm start &;\ngo run main.go &\n}",
			wantName:  "dev",
			wantWatch: true,
			wantStop:  false,
			wantText:  "",
			wantBlock: true,
			wantErr:   false,
		},
		{
			name:      "stop command with block",
			input:     "stop dev: {\npkill node;\npkill go\n}",
			wantName:  "dev",
			wantWatch: false,
			wantStop:  true,
			wantText:  "",
			wantBlock: true,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Parse(tt.input)

			// Check error expectation
			if (err != nil) != tt.wantErr {
				t.Fatalf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			// Ensure we have exactly one command
			if len(result.Commands) != 1 {
				t.Fatalf("Expected 1 command, got %d", len(result.Commands))
			}

			// Check command properties
			cmd := result.Commands[0]
			if cmd.Name != tt.wantName {
				t.Errorf("Command name = %q, want %q", cmd.Name, tt.wantName)
			}

			if cmd.IsWatch != tt.wantWatch {
				t.Errorf("IsWatch = %v, want %v", cmd.IsWatch, tt.wantWatch)
			}

			if cmd.IsStop != tt.wantStop {
				t.Errorf("IsStop = %v, want %v", cmd.IsStop, tt.wantStop)
			}

			if cmd.IsBlock != tt.wantBlock {
				t.Errorf("IsBlock = %v, want %v", cmd.IsBlock, tt.wantBlock)
			}

			// For simple commands, check the command text
			if !tt.wantBlock && cmd.Command != tt.wantText {
				t.Errorf("Command text = %q, want %q", cmd.Command, tt.wantText)
			}
		})
	}
}

func TestVariableReferences(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantExpanded string
		wantErr      bool
	}{
		{
			name:         "simple variable reference",
			input:        "def SRC = ./src;\nbuild: cd $(SRC) && make",
			wantExpanded: "cd ./src && make",
			wantErr:      false,
		},
		{
			name:         "multiple variable references",
			input:        "def SRC = ./src;\ndef BIN = ./bin;\nbuild: cp $(SRC)/main $(BIN)/app",
			wantExpanded: "cp ./src/main ./bin/app",
			wantErr:      false,
		},
		{
			name:         "variable in block command",
			input:        "def SRC = ./src;\nsetup: { cd $(SRC); make all }",
			wantExpanded: "cd ./src", // Check just first statement
			wantErr:      false,
		},
		{
			name:         "escaped dollar sign",
			input:        "def PATH = /bin;\necho: echo \\$PATH is $(PATH)",
			wantExpanded: "echo $PATH is /bin",
			wantErr:      false,
		},
		{
			name:         "undefined variable",
			input:        "build: echo $(UNDEFINED)",
			wantExpanded: "",
			wantErr:      true, // Should fail during ExpandVariables
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the input
			result, err := Parse(tt.input)
			if err != nil {
				if !tt.wantErr {
					t.Fatalf("Parse() error = %v", err)
				}
				return
			}

			// Try to expand variables
			err = result.ExpandVariables()
			if (err != nil) != tt.wantErr {
				t.Fatalf("ExpandVariables() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			// Check the expanded command
			if len(result.Commands) == 0 {
				t.Fatalf("No commands found")
			}

			cmd := result.Commands[0]
			var expandedText string

			if cmd.IsBlock {
				if len(cmd.Block) == 0 {
					t.Fatalf("No block statements found")
				}
				expandedText = cmd.Block[0].Command
			} else {
				expandedText = cmd.Command
			}

			if expandedText != tt.wantExpanded {
				t.Errorf("Expanded text = %q, want %q", expandedText, tt.wantExpanded)
			}
		})
	}
}

func TestContinuationLines(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantCommand string
		wantErr     bool
	}{
		{
			name:        "simple continuation",
			input:       "build: echo hello \\\nworld",
			wantCommand: "echo hello world",
			wantErr:     false,
		},
		{
			name:        "multiple continuations",
			input:       "build: echo hello \\\nworld \\\nuniverse",
			wantCommand: "echo hello world universe",
			wantErr:     false,
		},
		{
			name:        "continuation with variables",
			input:       "def DIR = src;\nbuild: cd $(DIR) \\\n&& make",
			wantCommand: "cd $(DIR) && make",
			wantErr:     false,
		},
		{
			name:        "continuation with indentation",
			input:       "build: echo hello \\\n    world",
			wantCommand: "echo hello world",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the input
			result, err := Parse(tt.input)

			// Check error expectation
			if (err != nil) != tt.wantErr {
				t.Fatalf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			// Find the actual command (might not be the first one in some tests)
			var cmd *Command
			for i := range result.Commands {
				if strings.HasPrefix(result.Commands[i].Command, "echo") ||
					strings.HasPrefix(result.Commands[i].Command, "cd") {
					cmd = &result.Commands[i]
					break
				}
			}

			if cmd == nil {
				t.Fatalf("Command not found in result")
			}

			// Check the command text
			if cmd.Command != tt.wantCommand {
				t.Errorf("Command text = %q, want %q", cmd.Command, tt.wantCommand)
			}
		})
	}
}

func TestErrorHandling(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		wantErrSubstr string
	}{
		{
			name:          "duplicate command",
			input:         "build: echo hello\nbuild: echo world",
			wantErrSubstr: "duplicate command",
		},
		{
			name:          "duplicate definition",
			input:         "def VAR = value1;\ndef VAR = value2;",
			wantErrSubstr: "duplicate definition",
		},
		{
			name:          "syntax error in command",
			input:         "build echo hello",  // Missing colon
			wantErrSubstr: "missing ':'",      // Updated to match actual error
		},
		{
			name:          "unclosed block",
			input:         "build: { echo hello",
			wantErrSubstr: "missing '}'",      // Updated to match actual error
		},
		{
			name:          "bad variable expansion",
			input:         "build: echo $(missingVar)",
			wantErrSubstr: "undefined variable",
		},
		{
			name:          "missing semicolon in definition",
			input:         "def VAR = value\nbuild: echo hello",
			wantErrSubstr: "missing ';'",      // Updated to match actual error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse and possibly expand variables
			result, err := Parse(tt.input)

			// If no syntax error, try expanding variables to catch semantic errors
			if err == nil && strings.Contains(tt.input, "$(") {
				err = result.ExpandVariables()
			}

			// Expect an error
			if err == nil {
				t.Fatalf("Expected error containing %q, got nil", tt.wantErrSubstr)
			}

			// Check that the error contains the expected substring
			if !strings.Contains(err.Error(), tt.wantErrSubstr) {
				t.Errorf("Error = %q, want substring %q", err.Error(), tt.wantErrSubstr)
			}
		})
	}
}

func TestCompleteFile(t *testing.T) {
	input := `
# Development commands
def SRC = ./src;
def BIN = ./bin;

# Build commands
build: cd $(SRC) && make all

# Run commands
watch server: {
  cd $(SRC);
  ./server --port=8080 &;
  ./worker --queue=jobs &
}

stop server: pkill -f "server|worker"
`

	result, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Verify definitions
	if len(result.Definitions) != 2 {
		t.Errorf("Expected 2 definitions, got %d", len(result.Definitions))
	} else {
		defNames := map[string]string{
			result.Definitions[0].Name: result.Definitions[0].Value,
			result.Definitions[1].Name: result.Definitions[1].Value,
		}

		if defNames["SRC"] != "./src" {
			t.Errorf("Definition SRC = %q, want %q", defNames["SRC"], "./src")
		}

		if defNames["BIN"] != "./bin" {
			t.Errorf("Definition BIN = %q, want %q", defNames["BIN"], "./bin")
		}
	}

	// Verify commands
	if len(result.Commands) != 3 {
		t.Errorf("Expected 3 commands, got %d", len(result.Commands))
	} else {
		// Collect commands by name
		cmdMap := make(map[string]*Command)
		for i := range result.Commands {
			cmdMap[result.Commands[i].Name] = &result.Commands[i]
		}

		// Check build command
		buildCmd, ok := cmdMap["build"]
		if !ok {
			t.Errorf("Missing 'build' command")
		} else if buildCmd.Command != "cd $(SRC) && make all" {
			t.Errorf("build command = %q, want %q", buildCmd.Command, "cd $(SRC) && make all")
		}

		// Check server watch command
		serverCmd, ok := cmdMap["server"]
		if !ok {
			t.Errorf("Missing 'server' command")
		} else {
			if !serverCmd.IsWatch {
				t.Errorf("Expected server command to be a watch command")
			}

			if !serverCmd.IsBlock {
				t.Errorf("Expected server command to be a block command")
			}

			if len(serverCmd.Block) != 3 {
				t.Errorf("Expected 3 block statements in server command, got %d", len(serverCmd.Block))
			} else {
				// Check for background statements
				backgroundCount := 0
				for _, stmt := range serverCmd.Block {
					if stmt.Background {
						backgroundCount++
					}
				}

				if backgroundCount != 2 {
					t.Errorf("Expected 2 background statements, got %d", backgroundCount)
				}
			}
		}

		// Verify variable expansion
		err = result.ExpandVariables()
		if err != nil {
			t.Fatalf("ExpandVariables() error = %v", err)
		}

		// Check that variables were expanded
		if buildCmd, ok := cmdMap["build"]; ok {
			if buildCmd.Command != "cd ./src && make all" {
				t.Errorf("Expanded build command = %q, want %q", buildCmd.Command, "cd ./src && make all")
			}
		}
	}
}
