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
  cd @var(SRC) && go run main.go --port=@var(PORT)
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
  @sh(cd @var(SRC) && go build -o @var(DIST)/app) && rsync -av @var(DIST)/ server:/opt/app/
}`,
			Expected: Program(
				Var("SRC", "./src"),
				Var("DIST", "./dist"),
				Var("PORT", "8080"),
				// Grouped variables
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
				CmdBlock("serve",
					Decorator("timeout", At("var", "TIMEOUT")),
					Text("cd "),
					At("var", "SRC"),
					Text(" && go run main.go --port="),
					At("var", "PORT"),
				),
				WatchBlock("dev",
					Decorator("parallel"),
					At("sh", "cd @var(SRC) && go run main.go"),
					At("sh", "cd frontend && npm start"),
				),
				StopBlock("dev",
					Text("pkill -f \"go run\""),
					Text("pkill -f \"npm start\""),
				),
				CmdBlock("deploy",
					Decorator("confirm", "Deploy to production?"),
					At("sh", "cd @var(SRC) && go build -o @var(DIST)/app"),
					Text(" && rsync -av "),
					At("var", "DIST"),
					Text("/ server:/opt/app/"),
				),
			),
		},
		{
			Name: "realistic development workflow",
			Input: `var (
  SRC = ./src
  DIST = ./dist
  PORT = 3000
  ENV = development
)

install: npm install

clean: rm -rf @var(DIST) node_modules

build: @timeout(2m) {
  echo "Building project..." && npm run build && echo "Build complete"
}

dev: @parallel {
  @sh(cd @var(SRC) && npm run dev)
  @sh(echo "Server starting on port @var(PORT)")
}

test: @retry(3) {
  npm run test && npm run lint
}

watch test: @debounce(500ms) {
  npm run test:watch
}

deploy: @confirm("Deploy to production?") @timeout(5m) {
  npm run build && @sh(rsync -av @var(DIST)/ server:/var/www/app/) && echo "Deployment complete"
}

stop test: pkill -f "npm.*test"`,
			Expected: Program(
				Var("SRC", "./src"),
				Var("DIST", "./dist"),
				Var("PORT", "3000"),
				Var("ENV", "development"),
				Cmd("install", "npm install"),
				Cmd("clean", Simple(
					Text("rm -rf "),
					At("var", "DIST"),
					Text(" node_modules"),
				)),
				CmdBlock("build",
					Decorator("timeout", "2m"),
					Text("echo \"Building project...\" && npm run build && echo \"Build complete\""),
				),
				CmdBlock("dev",
					Decorator("parallel"),
					At("sh", "cd @var(SRC) && npm run dev"),
					At("sh", "echo \"Server starting on port @var(PORT)\""),
				),
				CmdBlock("test",
					Decorator("retry", "3"),
					Text("npm run test && npm run lint"),
				),
				WatchBlock("test",
					Decorator("debounce", "500ms"),
					Text("npm run test:watch"),
				),
				CmdBlock("deploy",
					Decorator("confirm", "Deploy to production?"),
					Decorator("timeout", "5m"),
					Text("npm run build && "),
					At("sh", "rsync -av @var(DIST)/ server:/var/www/app/"),
					Text(" && echo \"Deployment complete\""),
				),
				Stop("test", "pkill -f \"npm.*test\""),
			),
		},
		{
			Name: "complex mixed content example",
			Input: `var API_URL = https://api.example.com
var TOKEN = abc123
var PROJECT = myproject

api-test: {
  echo "Testing API at @var(API_URL)" && @sh(curl -H "Authorization: Bearer @var(TOKEN)" @var(API_URL)/health) && echo "API test complete"
}

backup: @confirm("Create backup?") {
  echo "Starting backup..." && @sh(DATE=$(date +%Y%m%d); echo "Backup date: $DATE") && rsync -av /data/ backup@server.com:/backups/@var(PROJECT)/ && echo "Backup complete"
}`,
			Expected: Program(
				Var("API_URL", "https://api.example.com"),
				Var("TOKEN", "abc123"),
				Var("PROJECT", "myproject"),
				CmdBlock("api-test",
					Text("echo \"Testing API at "),
					At("var", "API_URL"),
					Text("\" && "),
					At("sh", "curl -H \"Authorization: Bearer @var(TOKEN)\" @var(API_URL)/health"),
					Text(" && echo \"API test complete\""),
				),
				CmdBlock("backup",
					Decorator("confirm", "Create backup?"),
					Text("echo \"Starting backup...\" && "),
					At("sh", "DATE=$(date +%Y%m%d); echo \"Backup date: $DATE\""),
					Text(" && rsync -av /data/ backup@server.com:/backups/"),
					At("var", "PROJECT"),
					Text("/ && echo \"Backup complete\""),
				),
			),
		},
		{
			Name: "simple commands with function decorators",
			Input: `var HOST = localhost
var PORT = 8080

ping: curl http://@var(HOST):@var(PORT)/health
status: echo "Server running at @var(HOST):@var(PORT)"
info: @sh(echo "Host: @var(HOST), Port: @var(PORT)")`,
			Expected: Program(
				Var("HOST", "localhost"),
				Var("PORT", "8080"),
				Cmd("ping", Simple(
					Text("curl http://"),
					At("var", "HOST"),
					Text(":"),
					At("var", "PORT"),
					Text("/health"),
				)),
				Cmd("status", Simple(
					Text("echo \"Server running at "),
					At("var", "HOST"),
					Text(":"),
					At("var", "PORT"),
					Text("\""),
				)),
				Cmd("info", Simple(
					At("sh", "echo \"Host: @var(HOST), Port: @var(PORT)\""),
				)),
			),
		},
		{
			Name: "nested decorators with explicit blocks",
			Input: `var RETRIES = 3
var TIMEOUT = 30s

complex: @timeout(@var(TIMEOUT)) {
  @retry(@var(RETRIES)) {
    echo "Attempting operation..." && ./run-operation.sh
  }
}`,
			Expected: Program(
				Var("RETRIES", "3"),
				Var("TIMEOUT", "30s"),
				CmdBlock("complex",
					Decorator("timeout", At("var", "TIMEOUT")),
					Decorator("retry", At("var", "RETRIES")),
					Text("echo \"Attempting operation...\" && ./run-operation.sh"),
				),
			),
		},
		{
			Name: "edge case - empty commands and blocks",
			Input: `var EMPTY =

empty:
block: {}
decorated: @parallel {}`,
			Expected: Program(
				Var("EMPTY", ""),
				Cmd("empty", ""),
				CmdBlock("block"),
				CmdBlock("decorated", Decorator("parallel")),
			),
		},
	}

	for _, tc := range testCases {
		RunTestCase(t, tc)
	}
}
