package parser

import "testing"

// Test complete file integration
func TestCompleteFile(t *testing.T) {
	input := `
# Development commands
var SRC = ./src
var BIN = ./bin

# Build commands
build: cd @var(SRC) && make all

# Run commands with parallel execution
watch server: {
  cd @var(SRC)
  @parallel {
    ./server --port=8080
    ./worker --queue=jobs
  }
}

stop server: pkill -f "server|worker"

# POSIX shell commands with braces using @sh()
cleanup: @sh(find . -name "*.tmp" -exec rm {} \;)
`

	tc := TestCase{
		Name:  "complete file integration test",
		Input: input,
		Expected: ExpectedProgram{
			Variables: []ExpectedVariable{
				Variable("SRC", StringExpr("./src")),
				Variable("BIN", StringExpr("./bin")),
			},
			Commands: []ExpectedCommand{
				SimpleCommand("build", nil, SimpleCommandBody(
					TextElement("cd "),
					VarRefElement("SRC"),
					TextElement(" && make all"))),
				WatchCommand("server", nil, BlockCommandBody(
					Statement(TextElement("cd "), VarRefElement("SRC")),
					Statement(DecoratorElement("parallel"),
						TextElement("./server --port=8080"),
						TextElement("./worker --queue=jobs")))),
				StopCommand("server", nil, SimpleCommandBody(
					TextElement("pkill -f \"server|worker\""))),
				SimpleCommand("cleanup", []ExpectedDecorator{
					{Name: "sh", Args: []ExpectedExpression{
						StringExpr("find . -name \"*.tmp\" -exec rm {} \\;"),
					}},
				}, BlockCommandBody()),
			},
		},
	}

	runTestCase(t, tc)
}
