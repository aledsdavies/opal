package parser

import (
	"testing"
)

func TestComplexShellCommands(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "simple shell command substitution",
			Input: `test-simple: echo "$(date)"`,
			Expected: Program(
				Cmd("test-simple", Simple(
					Text(`echo "$(date)"`),
				)),
			),
		},
		{
			Name:  "shell command with test and command substitution",
			Input: `test-condition: if [ "$(echo test)" = "test" ]; then echo ok; fi`,
			Expected: Program(
				Cmd("test-condition", Simple(
					Text(`if [ "$(echo test)" = "test" ]; then echo ok; fi`),
				)),
			),
		},
		{
			Name:  "command with @var and shell substitution",
			Input: `test-mixed: cd @var(SRC) && echo "files: $(ls | wc -l)"`,
			Expected: Program(
				Cmd("test-mixed", Simple(
					Text("cd "),
					At("var", "SRC"),
					Text(` && echo "files: $(ls | wc -l)"`),
				)),
			),
		},
		{
			Name:  "simplified version of failing command",
			Input: `test-format: if [ "$(gofumpt -l . | wc -l)" -gt 0 ]; then echo "issues"; fi`,
			Expected: Program(
				Cmd("test-format", Simple(
					Text(`if [ "$(gofumpt -l . | wc -l)" -gt 0 ]; then echo "issues"; fi`),
				)),
			),
		},
		{
			Name: "backup command with shell substitution and @var",
			Input: `backup: {
        echo "Creating backup..."
        DATE=$(date +%Y%m%d-%H%M%S); echo "Backup timestamp: $DATE"
        (which kubectl && kubectl exec deployment/database -n @var(KUBE_NAMESPACE) -- pg_dump myapp > backup-$(date +%Y%m%d-%H%M%S).sql) || echo "No database"
      }`,
			Expected: Program(
				CmdBlock("backup",
					// Lexer preserves the entire content as a single text block with embedded decorators
					Text(`echo "Creating backup..."
        DATE=$(date +%Y%m%d-%H%M%S); echo "Backup timestamp: $DATE"
        (which kubectl && kubectl exec deployment/database -n `),
					At("var", "KUBE_NAMESPACE"),
					Text(` -- pg_dump myapp > backup-$(date +%Y%m%d-%H%M%S).sql) || echo "No database"`),
				),
			),
		},
		{
			Name: "exact command from real commands.cli file",
			Input: `test-quick: {
    echo "⚡ Running quick checks..."
    echo "🔍 Checking Go formatting..."
    if command -v gofumpt >/dev/null 2>&1; then if [ "$(gofumpt -l . | wc -l)" -gt 0 ]; then echo "❌ Go formatting issues:"; gofumpt -l .; exit 1; fi; else if [ "$(gofmt -l . | wc -l)" -gt 0 ]; then echo "❌ Go formatting issues:"; gofmt -l .; exit 1; fi; fi
    echo "🔍 Checking Nix formatting..."
    if command -v nixpkgs-fmt >/dev/null 2>&1; then nixpkgs-fmt --check . || (echo "❌ Run 'dev format' to fix"; exit 1); else echo "⚠️  nixpkgs-fmt not available, skipping Nix format check"; fi
    dev lint
    echo "✅ Quick checks passed!"
}`,
			Expected: Program(
				CmdBlock("test-quick",
					// Lexer preserves all content as single concatenated text (trust the lexer philosophy)
					Text(`echo "⚡ Running quick checks..."
    echo "🔍 Checking Go formatting..."
    if command -v gofumpt >/dev/null 2>&1; then if [ "$(gofumpt -l . | wc -l)" -gt 0 ]; then echo "❌ Go formatting issues:"; gofumpt -l .; exit 1; fi; else if [ "$(gofmt -l . | wc -l)" -gt 0 ]; then echo "❌ Go formatting issues:"; gofmt -l .; exit 1; fi; fi
    echo "🔍 Checking Nix formatting..."
    if command -v nixpkgs-fmt >/dev/null 2>&1; then nixpkgs-fmt --check . || (echo "❌ Run 'dev format' to fix"; exit 1); else echo "⚠️  nixpkgs-fmt not available, skipping Nix format check"; fi
    dev lint
    echo "✅ Quick checks passed!"`),
				),
			),
		},
		{
			Name:  "shell with here document",
			Input: "heredoc: {\ncat <<EOF\nLine 1\nLine 2\nEOF\n}",
			Expected: Program(
				CmdBlock("heredoc",
					Text("cat <<EOF\nLine 1\nLine 2\nEOF\n"),
				),
			),
		},
	}

	for _, tc := range testCases {
		RunTestCase(t, tc)
	}
}

func TestVarInShellCommands(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "simple @var in shell command",
			Input: "test-var: cd @var(DIR)",
			Expected: Program(
				Cmd("test-var", Simple(
					Text("cd "),
					At("var", "DIR"),
				)),
			),
		},
		{
			Name:  "@var with shell command substitution",
			Input: `test-var-cmd: cd @var(DIR) && echo "$(pwd)"`,
			Expected: Program(
				Cmd("test-var-cmd", Simple(
					Text("cd "),
					At("var", "DIR"),
					Text(` && echo "$(pwd)"`),
				)),
			),
		},
		{
			Name:  "multiple @var with complex shell",
			Input: `test-multi-var: if [ -d @var(SRC) ] && [ "$(ls @var(SRC) | wc -l)" -gt 0 ]; then echo "Source dir has files"; fi`,
			Expected: Program(
				Cmd("test-multi-var", Simple(
					Text("if [ -d "),
					At("var", "SRC"),
					Text(` ] && [ "$(ls `),
					At("var", "SRC"),
					Text(` | wc -l)" -gt 0 ]; then echo "Source dir has files"; fi`),
				)),
			),
		},
		{
			Name:  "@var in shell array",
			Input: "array-var: FILES=(@var(FILE1) @var(FILE2) @var(FILE3)); echo ${FILES[@]}",
			Expected: Program(
				Cmd("array-var", Simple(
					Text("FILES=("),
					At("var", "FILE1"),
					Text(" "),
					At("var", "FILE2"),
					Text(" "),
					At("var", "FILE3"),
					Text("); echo ${FILES[@]}"),
				)),
			),
		},
		{
			Name:  "@var in shell case statement",
			Input: "case-var: case @var(ENV) in prod) echo production;; dev) echo development;; esac",
			Expected: Program(
				Cmd("case-var", Simple(
					Text("case "),
					At("var", "ENV"),
					Text(" in prod) echo production;; dev) echo development;; esac"),
				)),
			),
		},
		{
			Name:  "@var in shell parameter expansion",
			Input: "param-expansion: echo ${@var(VAR):-@var(DEFAULT)}",
			Expected: Program(
				Cmd("param-expansion", Simple(
					Text("echo ${"),
					At("var", "VAR"),
					Text(":-"),
					At("var", "DEFAULT"),
					Text("}"),
				)),
			),
		},
	}

	for _, tc := range testCases {
		RunTestCase(t, tc)
	}
}

func TestLineContinuationEdgeCases(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "continuation at end of line with no following content",
			Input: "test: echo hello \\\n",
			Expected: Program(
				Cmd("test", Simple(Text("echo hello "))),
			),
		},
		{
			Name:  "continuation with only whitespace on next line",
			Input: "test: echo hello \\\n   ",
			Expected: Program(
				Cmd("test", Simple(Text("echo hello "))),
			),
		},
		{
			Name:  "continuation with tab characters",
			Input: "test: echo hello \\\n\tworld",
			Expected: Program(
				Cmd("test", Simple(Text("echo hello world"))),
			),
		},
		{
			Name:  "continuation in middle of quoted string",
			Input: `test: echo "hello \` + "\n" + `world"`,
			Expected: Program(
				Cmd("test", Simple(Text(`echo "hello world"`))),
			),
		},
		{
			Name:  "continuation in single quotes (should be literal)",
			Input: "test: echo 'hello \\\nworld'",
			Expected: Program(
				Cmd("test", Simple(Text("echo 'hello \\\nworld'"))),
			),
		},
		{
			Name:  "continuation with @var across lines",
			Input: "test: echo \\\n@var(NAME) \\\nis here",
			Expected: Program(
				Cmd("test", Simple(
					Text("echo "),
					At("var", "NAME"),
					Text(" is here"),
				)),
			),
		},
	}

	for _, tc := range testCases {
		RunTestCase(t, tc)
	}
}

func TestContinuationLines(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "simple continuation",
			Input: "build: echo hello \\\nworld",
			Expected: Program(
				Cmd("build", Simple(Text("echo hello world"))),
			),
		},
		{
			Name:  "continuation with @var()",
			Input: "build: cd @var(DIR) \\\n&& make",
			Expected: Program(
				Cmd("build", Simple(
					Text("cd "),
					At("var", "DIR"),
					Text(" && make"),
				)),
			),
		},
		{
			Name:  "multiple line continuations",
			Input: "complex: echo start \\\n&& echo middle \\\n&& echo end",
			Expected: Program(
				Cmd("complex", Simple(Text("echo start && echo middle && echo end"))),
			),
		},
		{
			Name:  "continuation in block",
			Input: "block: { echo hello \\\nworld; echo done }",
			Expected: Program(
				CmdBlock("block",
					Text("echo hello world; echo done"),
				),
			),
		},
		{
			Name:  "continuation with mixed content",
			Input: "mixed: docker run \\\n@var(IMAGE) \\\n--port=@var(PORT)",
			Expected: Program(
				Cmd("mixed", Simple(
					Text("docker run "),
					At("var", "IMAGE"),
					Text(" --port="),
					At("var", "PORT"),
				)),
			),
		},
		{
			Name:  "continuation with trailing spaces",
			Input: "trailing: echo hello \\\n   world",
			Expected: Program(
				Cmd("trailing", Simple(Text("echo hello world"))),
			),
		},
		{
			Name:  "continuation breaking long docker command",
			Input: "docker: docker run \\\n--name myapp \\\n--port 8080:8080 \\\n--env NODE_ENV=production \\\nmyimage:latest",
			Expected: Program(
				Cmd("docker", Simple(Text("docker run --name myapp --port 8080:8080 --env NODE_ENV=production myimage:latest"))),
			),
		},
	}

	for _, tc := range testCases {
		RunTestCase(t, tc)
	}
}
