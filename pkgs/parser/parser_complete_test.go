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
				// Grouped variables (these would be handled differently in real parser)
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
				CmdWith(Decorator("timeout", At("var", "TIMEOUT")), "serve", Block(
					Text("cd "),
					At("var", "SRC"),
					Text("go run main.go --port="),
					At("var", "PORT"),
				)),
				WatchWith(Decorator("parallel"), "dev", Block(
					At("sh", "cd @var(SRC) && go run main.go"),
					At("sh", "cd frontend && npm start"),
				)),
				Stop("dev", Block(
					Text("pkill -f \"go run\""),
					Text("pkill -f \"npm start\""),
				)),
				CmdWith(Decorator("confirm", "Deploy to production?"), "deploy", Block(
					At("sh", "cd @var(SRC) && go build -o @var(DIST)/app"),
					Text("rsync -av "),
					At("var", "DIST"),
					Text("/ server:/opt/app/"),
				)),
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
  echo "Building project..."
  npm run build
  echo "Build complete"
}

dev: @parallel {
  @sh(cd @var(SRC) && npm run dev)
  @sh(echo "Server starting on port @var(PORT)")
}

test: @retry(3) {
  npm run test
  npm run lint
}

watch test: @debounce(500ms) {
  npm run test:watch
}

deploy: @confirm("Deploy to production?") @timeout(5m) {
  npm run build
  @sh(rsync -av @var(DIST)/ server:/var/www/app/)
  echo "Deployment complete"
}

stop test: pkill -f "npm.*test"`,
			Expected: Program(
				Var("SRC", "./src"),
				Var("DIST", "./dist"),
				Var("PORT", "3000"),
				Var("ENV", "development"),
				Cmd("install", Simple(Text("npm install"))),
				Cmd("clean", Simple(
					Text("rm -rf "),
					At("var", "DIST"),
					Text(" node_modules"),
				)),
				CmdWith(Decorator("timeout", "2m"), "build", Block(
					Text("echo \"Building project...\""),
					Text("npm run build"),
					Text("echo \"Build complete\""),
				)),
				CmdWith(Decorator("parallel"), "dev", Block(
					At("sh", "cd @var(SRC) && npm run dev"),
					At("sh", "echo \"Server starting on port @var(PORT)\""),
				)),
				CmdWith(Decorator("retry", "3"), "test", Block(
					Text("npm run test"),
					Text("npm run lint"),
				)),
				WatchWith(Decorator("debounce", "500ms"), "test", Block(
					Text("npm run test:watch"),
				)),
				CmdWith([]ExpectedDecorator{
					Decorator("confirm", "Deploy to production?"),
					Decorator("timeout", "5m"),
				}, "deploy", Block(
					Text("npm run build"),
					At("sh", "rsync -av @var(DIST)/ server:/var/www/app/"),
					Text("echo \"Deployment complete\""),
				)),
				Stop("test", Simple(Text("pkill -f \"npm.*test\""))),
			),
		},
		{
			Name: "complex mixed content example",
			Input: `var API_URL = https://api.example.com
var TOKEN = abc123

api-test: {
  echo "Testing API at @var(API_URL)"
  @sh(curl -H "Authorization: Bearer @var(TOKEN)" @var(API_URL)/health)
  echo "API test complete"
}

backup: @confirm("Create backup?") {
  echo "Starting backup..."
  @sh(DATE=$(date +%Y%m%d); echo "Backup date: $DATE")
  rsync -av /data/ backup@server.com:/backups/@var(PROJECT)/
  echo "Backup complete"
}`,
			Expected: Program(
				Var("API_URL", "https://api.example.com"),
				Var("TOKEN", "abc123"),
				Cmd("api-test", Block(
					Text("echo \"Testing API at "),
					At("var", "API_URL"),
					Text("\""),
					At("sh", "curl -H \"Authorization: Bearer @var(TOKEN)\" @var(API_URL)/health"),
					Text("echo \"API test complete\""),
				)),
				CmdWith(Decorator("confirm", "Create backup?"), "backup", Block(
					Text("echo \"Starting backup...\""),
					At("sh", "DATE=$(date +%Y%m%d); echo \"Backup date: $DATE\""),
					Text("rsync -av /data/ backup@server.com:/backups/"),
					At("var", "PROJECT"),
					Text("/"),
					Text("echo \"Backup complete\""),
				)),
			),
		},
	}

	for _, tc := range testCases {
		RunTestCase(t, tc)
	}
}

func TestErrorHandling(t *testing.T) {
	testCases := []TestCase{
		{
			Name:        "missing closing brace",
			Input:       "test: { echo hello",
			WantErr:     true,
			ErrorSubstr: "missing closing brace",
		},
		{
			Name:        "missing closing parenthesis in decorator",
			Input:       "test: @timeout(30s { echo hello }",
			WantErr:     true,
			ErrorSubstr: "missing closing parenthesis",
		},
		{
			Name:        "invalid decorator syntax",
			Input:       "test: @123invalid { echo hello }",
			WantErr:     true,
			ErrorSubstr: "invalid decorator name",
		},
		{
			Name:        "missing command name",
			Input:       ": echo hello",
			WantErr:     true,
			ErrorSubstr: "missing command name",
		},
		{
			Name:        "missing colon after command name",
			Input:       "test echo hello",
			WantErr:     true,
			ErrorSubstr: "expected colon",
		},
		{
			Name:        "invalid variable name",
			Input:       "var 123invalid = value",
			WantErr:     true,
			ErrorSubstr: "invalid variable name",
		},
		{
			Name:        "missing variable value",
			Input:       "var NAME =",
			WantErr:     true,
			ErrorSubstr: "missing variable value",
		},
		{
			Name:        "nested commands not allowed",
			Input:       "outer: { inner: echo hello }",
			WantErr:     true,
			ErrorSubstr: "nested commands not allowed",
		},
		{
			Name:        "decorator without block when required",
			Input:       "test: @timeout(30s) echo hello",
			WantErr:     true,
			ErrorSubstr: "decorator requires block",
		},
		{
			Name:        "multiple decorators without proper syntax",
			Input:       "test: @timeout(30s) @parallel echo hello",
			WantErr:     true,
			ErrorSubstr: "multiple decorators require block",
		},
	}

	for _, tc := range testCases {
		RunTestCase(t, tc)
	}
}
