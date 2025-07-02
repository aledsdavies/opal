package parser

import (
	"testing"
)

func TestCompleteParserIntegration(t *testing.T) {
	testCases := []TestCase{
		{
			Name: "complete devcmd file",
			Input: `# Complete devcmd example
var SRC = ./src
var DIST = ./dist
var PORT = 8080

var (
  NODE_ENV = development
  TIMEOUT = 30s
  DEBUG = true
)

build: go build -o @var(DIST)/app @var(SRC)/main.go

serve: @timeout(@var(TIMEOUT)) {
  cd @var(SRC)
  go run main.go --port=@var(PORT)
}

watch dev: @parallel {
  @sh(cd @var(SRC) && go run main.go)
  @sh(cd frontend && npm start)
}

stop dev: {
  pkill -f "go run"
  pkill -f "npm start"
}

deploy: @confirm("Deploy to production?") {
  @sh(cd @var(SRC) && go build -o @var(DIST)/app)
  rsync -av @var(DIST)/ server:/opt/app/
}`,
			Expected: Program(
				Var("SRC", "./src"),
				Var("DIST", "./dist"),
				Var("PORT", "8080"),
				Var("NODE_ENV", "development"),
				Var("TIMEOUT", "30s"),
				Var("DEBUG", "true"),
				Cmd("build", Simple(
					Text("go build -o "),
					At("var", "DIST"),
					Text("/app "),
					At("var", "SRC"),
					Text("/main.go"),
				)),
				CmdWith(At("timeout", "@var(TIMEOUT)"), "serve", Block(
					Statement(Text("cd "), At("var", "SRC")),
					Statement(Text("go run main.go --port="), At("var", "PORT")),
				)),
				WatchWith(At("parallel"), "dev", Block(
					At("sh", "cd @var(SRC) && go run main.go"),
					At("sh", "cd frontend && npm start"),
				)),
				Stop("dev", Block(
					"pkill -f \"go run\"",
					"pkill -f \"npm start\"",
				)),
				CmdWith(At("confirm", "Deploy to production?"), "deploy", Block(
					At("sh", "cd @var(SRC) && go build -o @var(DIST)/app"),
					Statement(Text("rsync -av "), At("var", "DIST"), Text("/ server:/opt/app/")),
				)),
			),
		},
	}

	for _, tc := range testCases {
		RunTestCase(t, tc)
	}
}
