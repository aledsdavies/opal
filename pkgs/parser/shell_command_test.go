package parser

import (
	"testing"
)

func TestComplexShellCommands(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "simple shell command substitution",
			Input: "test-simple: @sh(echo \"$(date)\")",
			Expected: Program(
				Cmd("test-simple", Simple(
					At("sh", "echo \"$(date)\""),
				)),
			),
		},
		{
			Name:  "shell command with test and command substitution",
			Input: "test-condition: @sh(if [ \"$(echo test)\" = \"test\" ]; then echo ok; fi)",
			Expected: Program(
				Cmd("test-condition", Simple(
					At("sh", "if [ \"$(echo test)\" = \"test\" ]; then echo ok; fi"),
				)),
			),
		},
		{
			Name:  "command with @var and shell substitution",
			Input: "test-mixed: @sh(cd @var(SRC) && echo \"files: $(ls | wc -l)\")",
			Expected: Program(
				Cmd("test-mixed", Simple(
					At("sh", "cd @var(SRC) && echo \"files: $(ls | wc -l)\""),
				)),
			),
		},
		{
			Name:  "simplified version of failing command",
			Input: "test-format: @sh(if [ \"$(gofumpt -l . | wc -l)\" -gt 0 ]; then echo \"issues\"; fi)",
			Expected: Program(
				Cmd("test-format", Simple(
					At("sh", "if [ \"$(gofumpt -l . | wc -l)\" -gt 0 ]; then echo \"issues\"; fi"),
				)),
			),
		},
		{
			Name:  "even simpler - just the command substitution in quotes",
			Input: "test-basic: @sh(\"$(gofumpt -l . | wc -l)\")",
			Expected: Program(
				Cmd("test-basic", Simple(
					At("sh", "\"$(gofumpt -l . | wc -l)\""),
				)),
			),
		},
		{
			Name:  "debug - minimal parentheses in quotes",
			Input: "test-debug: @sh(\"()\")",
			Expected: Program(
				Cmd("test-debug", Simple(
					At("sh", "\"()\""),
				)),
			),
		},
		{
			Name:  "debug - single command substitution",
			Input: "test-debug2: @sh($(echo test))",
			Expected: Program(
				Cmd("test-debug2", Simple(
					At("sh", "$(echo test)"),
				)),
			),
		},
		{
			Name: "backup command with shell substitution and @var",
			Input: `backup: {
        echo "Creating backup..."
        @sh(DATE=$(date +%Y%m%d-%H%M%S); echo "Backup timestamp: $DATE")
        @sh((which kubectl && kubectl exec deployment/database -n @var(KUBE_NAMESPACE) -- pg_dump myapp > backup-$(date +%Y%m%d-%H%M%S).sql) || echo "No database")
      }`,
			Expected: Program(
				Cmd("backup", Block(
					Text("echo \"Creating backup...\""),
					At("sh", "DATE=$(date +%Y%m%d-%H%M%S); echo \"Backup timestamp: $DATE\""),
					At("sh", "(which kubectl && kubectl exec deployment/database -n @var(KUBE_NAMESPACE) -- pg_dump myapp > backup-$(date +%Y%m%d-%H%M%S).sql) || echo \"No database\""),
				)),
			),
		},
		{
			Name: "exact command from real commands.cli file",
			Input: `test-quick: {
    echo "⚡ Running quick checks..."
    echo "🔍 Checking Go formatting..."
    @sh(if command -v gofumpt >/dev/null 2>&1; then if [ "$(gofumpt -l . | wc -l)" -gt 0 ]; then echo "❌ Go formatting issues:"; gofumpt -l .; exit 1; fi; else if [ "$(gofmt -l . | wc -l)" -gt 0 ]; then echo "❌ Go formatting issues:"; gofmt -l .; exit 1; fi; fi)
    echo "🔍 Checking Nix formatting..."
    @sh(if command -v nixpkgs-fmt >/dev/null 2>&1; then nixpkgs-fmt --check . || (echo "❌ Run 'dev format' to fix"; exit 1); else echo "⚠️  nixpkgs-fmt not available, skipping Nix format check"; fi)
    dev lint
    echo "✅ Quick checks passed!"
}`,
			Expected: Program(
				Cmd("test-quick", Block(
					Text("echo \"⚡ Running quick checks...\""),
					Text("echo \"🔍 Checking Go formatting...\""),
					At("sh", "if command -v gofumpt >/dev/null 2>&1; then if [ \"$(gofumpt -l . | wc -l)\" -gt 0 ]; then echo \"❌ Go formatting issues:\"; gofumpt -l .; exit 1; fi; else if [ \"$(gofmt -l . | wc -l)\" -gt 0 ]; then echo \"❌ Go formatting issues:\"; gofmt -l .; exit 1; fi; fi"),
					Text("echo \"🔍 Checking Nix formatting...\""),
					At("sh", "if command -v nixpkgs-fmt >/dev/null 2>&1; then nixpkgs-fmt --check . || (echo \"❌ Run 'dev format' to fix\"; exit 1); else echo \"⚠️  nixpkgs-fmt not available, skipping Nix format check\"; fi"),
					Text("dev lint"),
					Text("echo \"✅ Quick checks passed!\""),
				)),
			),
		},
		{
			Name:  "shell with arithmetic expansion",
			Input: "math: @sh(echo $((2 + 3 * 4)))",
			Expected: Program(
				Cmd("math", Simple(
					At("sh", "echo $((2 + 3 * 4))"),
				)),
			),
		},
		{
			Name:  "shell with process substitution",
			Input: "diff: @sh(diff <(sort file1) <(sort file2))",
			Expected: Program(
				Cmd("diff", Simple(
					At("sh", "diff <(sort file1) <(sort file2)"),
				)),
			),
		},
		{
			Name:  "shell with parameter expansion",
			Input: "param: @sh(echo ${VAR:-default})",
			Expected: Program(
				Cmd("param", Simple(
					At("sh", "echo ${VAR:-default}"),
				)),
			),
		},
		{
			Name:  "shell with complex conditionals",
			Input: "conditional: @sh([[ -f file.txt && -r file.txt ]] && echo readable || echo not-readable)",
			Expected: Program(
				Cmd("conditional", Simple(
					At("sh", "[[ -f file.txt && -r file.txt ]] && echo readable || echo not-readable"),
				)),
			),
		},
		{
			Name:  "shell with here document",
			Input: "heredoc: @sh(cat <<EOF\nLine 1\nLine 2\nEOF)",
			Expected: Program(
				Cmd("heredoc", Simple(
					At("sh", "cat <<EOF\nLine 1\nLine 2\nEOF"),
				)),
			),
		},
		{
			Name:  "shell with case statement",
			Input: "case-test: @sh(case $1 in start) echo starting;; stop) echo stopping;; esac)",
			Expected: Program(
				Cmd("case-test", Simple(
					At("sh", "case $1 in start) echo starting;; stop) echo stopping;; esac"),
				)),
			),
		},
	}

	for _, tc := range testCases {
		RunTestCase(t, tc)
	}
}

func TestQuoteAndParenthesesEdgeCases(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "escaped quotes in shell command",
			Input: "test-escaped: @sh(echo \"He said \\\"hello\\\" to me\")",
			Expected: Program(
				Cmd("test-escaped", Simple(
					At("sh", "echo \"He said \\\"hello\\\" to me\""),
				)),
			),
		},
		{
			Name:  "mixed quotes with parentheses",
			Input: "test-mixed-quotes: @sh(echo 'test \"$(date)\" done' && echo \"test '$(whoami)' done\")",
			Expected: Program(
				Cmd("test-mixed-quotes", Simple(
					At("sh", "echo 'test \"$(date)\" done' && echo \"test '$(whoami)' done\""),
				)),
			),
		},
		{
			Name:  "backticks with parentheses",
			Input: "test-backticks: @sh(echo `date` and $(whoami))",
			Expected: Program(
				Cmd("test-backticks", Simple(
					At("sh", "echo `date` and $(whoami)"),
				)),
			),
		},
		{
			Name:  "nested quotes with special characters",
			Input: "special-chars: @sh(echo \"Path: '$HOME' and size: $(du -sh .)\")",
			Expected: Program(
				Cmd("special-chars", Simple(
					At("sh", "echo \"Path: '$HOME' and size: $(du -sh .)\""),
				)),
			),
		},
		{
			Name:  "quotes within @var context",
			Input: "var-quotes: echo \"Config file: '@var(CONFIG_FILE)'\"",
			Expected: Program(
				Cmd("var-quotes", Simple(
					Text("echo \"Config file: '"),
					At("var", "CONFIG_FILE"),
					Text("'\""),
				)),
			),
		},
		{
			Name:  "parentheses in shell without command substitution",
			Input: "parens: @sh((cd /tmp && ls) > /dev/null)",
			Expected: Program(
				Cmd("parens", Simple(
					At("sh", "(cd /tmp && ls) > /dev/null"),
				)),
			),
		},
		{
			Name:  "nested parentheses with command substitution",
			Input: "nested: @sh(echo $(echo $(date +%Y)))",
			Expected: Program(
				Cmd("nested", Simple(
					At("sh", "echo $(echo $(date +%Y))"),
				)),
			),
		},
		{
			Name:  "quotes with regex patterns",
			Input: "regex: grep \"pattern[0-9]+\" file.txt",
			Expected: Program(
				Cmd("regex", Simple(Text("grep \"pattern[0-9]+\" file.txt"))),
			),
		},
		{
			Name:  "single quotes preserving literals",
			Input: "literal: echo 'Variables like $HOME and $(date) are not expanded'",
			Expected: Program(
				Cmd("literal", Simple(Text("echo 'Variables like $HOME and $(date) are not expanded'"))),
			),
		},
		{
			Name:  "mixed quote styles in one command",
			Input: "mixed: echo 'Single quotes' and \"double quotes\" and `backticks`",
			Expected: Program(
				Cmd("mixed", Simple(Text("echo 'Single quotes' and \"double quotes\" and `backticks`"))),
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
			Input: "test-var: @sh(cd @var(DIR))",
			Expected: Program(
				Cmd("test-var", Simple(
					At("sh", "cd @var(DIR)"),
				)),
			),
		},
		{
			Name:  "@var with shell command substitution",
			Input: "test-var-cmd: @sh(cd @var(DIR) && echo \"$(pwd)\")",
			Expected: Program(
				Cmd("test-var-cmd", Simple(
					At("sh", "cd @var(DIR) && echo \"$(pwd)\""),
				)),
			),
		},
		{
			Name:  "multiple @var with complex shell",
			Input: "test-multi-var: @sh(if [ -d @var(SRC) ] && [ \"$(ls @var(SRC) | wc -l)\" -gt 0 ]; then echo \"Source dir has files\"; fi)",
			Expected: Program(
				Cmd("test-multi-var", Simple(
					At("sh", "if [ -d @var(SRC) ] && [ \"$(ls @var(SRC) | wc -l)\" -gt 0 ]; then echo \"Source dir has files\"; fi"),
				)),
			),
		},
		{
			Name:  "@var in shell function definition",
			Input: "func-def: @sh(deploy() { rsync -av @var(SRC)/ deploy@server.com:@var(DEST)/; }; deploy)",
			Expected: Program(
				Cmd("func-def", Simple(
					At("sh", "deploy() { rsync -av @var(SRC)/ deploy@server.com:@var(DEST)/; }; deploy"),
				)),
			),
		},
		{
			Name:  "@var in shell array",
			Input: "array-var: @sh(FILES=(@var(FILE1) @var(FILE2) @var(FILE3)); echo ${FILES[@]})",
			Expected: Program(
				Cmd("array-var", Simple(
					At("sh", "FILES=(@var(FILE1) @var(FILE2) @var(FILE3)); echo ${FILES[@]}"),
				)),
			),
		},
		{
			Name:  "@var in shell case statement",
			Input: "case-var: @sh(case @var(ENV) in prod) echo production;; dev) echo development;; esac)",
			Expected: Program(
				Cmd("case-var", Simple(
					At("sh", "case @var(ENV) in prod) echo production;; dev) echo development;; esac"),
				)),
			),
		},
		{
			Name:  "@var in shell parameter expansion",
			Input: "param-expansion: @sh(echo ${@var(VAR):-@var(DEFAULT)})",
			Expected: Program(
				Cmd("param-expansion", Simple(
					At("sh", "echo ${@var(VAR):-@var(DEFAULT)}"),
				)),
			),
		},
		{
			Name:  "mixing @var in shell content with regular shell content",
			Input: "mixed-content: echo start && @sh(cd @var(DIR)) && echo done",
			Expected: Program(
				Cmd("mixed-content", Simple(
					Text("echo start && "),
					At("sh", "cd @var(DIR)"),
					Text(" && echo done"),
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
				Cmd("test", Simple(Text("echo hello"))),
			),
		},
		{
			Name:  "continuation with only whitespace on next line",
			Input: "test: echo hello \\\n   ",
			Expected: Program(
				Cmd("test", Simple(Text("echo hello"))),
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
			Input: "test: echo \"hello \\\nworld\"",
			Expected: Program(
				Cmd("test", Simple(Text("echo \"hello world\""))),
			),
		},
		{
			Name:  "continuation in single quotes (should not join)",
			Input: "test: echo 'hello \\\nworld'",
			Expected: Program(
				Cmd("test", Simple(Text("echo 'hello \\\nworld'"))),
			),
		},
		{
			Name:  "backslash without newline (not a continuation)",
			Input: "test: echo hello\\world",
			Expected: Program(
				Cmd("test", Simple(Text("echo hello\\world"))),
			),
		},
		{
			Name:  "multiple backslashes before newline",
			Input: "test: echo hello\\\\\\\nworld",
			Expected: Program(
				Cmd("test", Simple(Text("echo hello\\\\ world"))),
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
				Cmd("block", Block(
					Text("echo hello world"),
					Text("echo done"),
				)),
			),
		},
		{
			Name:  "continuation with decorators",
			Input: "deploy: @sh(docker build \\\n-t myapp \\\n.)",
			Expected: Program(
				Cmd("deploy", Simple(
					At("sh", "docker build -t myapp ."),
				)),
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
			Name:  "continuation preserves spaces correctly",
			Input: "spaced: echo hello\\\n world",
			Expected: Program(
				Cmd("spaced", Simple(Text("echo hello world"))),
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
			Name:  "multiple continuations in decorator argument",
			Input: "long-arg: @sh(very long command \\\nwith multiple \\\nlines here)",
			Expected: Program(
				Cmd("long-arg", Simple(
					At("sh", "very long command with multiple lines here"),
				)),
			),
		},
		{
			Name:  "continuation with shell operators",
			Input: "operators: cmd1 \\\n&& cmd2 \\\n|| cmd3",
			Expected: Program(
				Cmd("operators", Simple(Text("cmd1 && cmd2 || cmd3"))),
			),
		},
		{
			Name:  "continuation with pipes",
			Input: "pipes: cat file.txt \\\n| grep pattern \\\n| sort",
			Expected: Program(
				Cmd("pipes", Simple(Text("cat file.txt | grep pattern | sort"))),
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
