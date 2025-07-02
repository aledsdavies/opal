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
				CmdWith(At("sh", "echo \"$(date)\""), "test-simple", Block()),
			),
		},
		{
			Name:  "shell command with test and command substitution",
			Input: "test-condition: @sh(if [ \"$(echo test)\" = \"test\" ]; then echo ok; fi)",
			Expected: Program(
				CmdWith(At("sh", "if [ \"$(echo test)\" = \"test\" ]; then echo ok; fi"), "test-condition", Block()),
			),
		},
		{
			Name:  "command with @var and shell substitution",
			Input: "test-mixed: @sh(cd @var(SRC) && echo \"files: $(ls | wc -l)\")",
			Expected: Program(
				CmdWith(At("sh", "cd @var(SRC) && echo \"files: $(ls | wc -l)\""), "test-mixed", Block()),
			),
		},
		{
			Name:  "simplified version of failing command",
			Input: "test-format: @sh(if [ \"$(gofumpt -l . | wc -l)\" -gt 0 ]; then echo \"issues\"; fi)",
			Expected: Program(
				CmdWith(At("sh", "if [ \"$(gofumpt -l . | wc -l)\" -gt 0 ]; then echo \"issues\"; fi"), "test-format", Block()),
			),
		},
		{
			Name:  "even simpler - just the command substitution in quotes",
			Input: "test-basic: @sh(\"$(gofumpt -l . | wc -l)\")",
			Expected: Program(
				CmdWith(At("sh", "\"$(gofumpt -l . | wc -l)\""), "test-basic", Block()),
			),
		},
		{
			Name:  "debug - minimal parentheses in quotes",
			Input: "test-debug: @sh(\"()\")",
			Expected: Program(
				CmdWith(At("sh", "\"()\""), "test-debug", Block()),
			),
		},
		{
			Name:  "debug - single command substitution",
			Input: "test-debug2: @sh($(echo test))",
			Expected: Program(
				CmdWith(At("sh", "$(echo test)"), "test-debug2", Block()),
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
					"echo \"Creating backup...\"",
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
					"echo \"⚡ Running quick checks...\"",
					"echo \"🔍 Checking Go formatting...\"",
					At("sh", "if command -v gofumpt >/dev/null 2>&1; then if [ \"$(gofumpt -l . | wc -l)\" -gt 0 ]; then echo \"❌ Go formatting issues:\"; gofumpt -l .; exit 1; fi; else if [ \"$(gofmt -l . | wc -l)\" -gt 0 ]; then echo \"❌ Go formatting issues:\"; gofmt -l .; exit 1; fi; fi"),
					"echo \"🔍 Checking Nix formatting...\"",
					At("sh", "if command -v nixpkgs-fmt >/dev/null 2>&1; then nixpkgs-fmt --check . || (echo \"❌ Run 'dev format' to fix\"; exit 1); else echo \"⚠️  nixpkgs-fmt not available, skipping Nix format check\"; fi"),
					"dev lint",
					"echo \"✅ Quick checks passed!\"",
				)),
			),
		},
		{
			Name:  "shell with arithmetic expansion",
			Input: "math: @sh(echo $((2 + 3 * 4)))",
			Expected: Program(
				CmdWith(At("sh", "echo $((2 + 3 * 4))"), "math", Block()),
			),
		},
		{
			Name:  "shell with process substitution",
			Input: "diff: @sh(diff <(sort file1) <(sort file2))",
			Expected: Program(
				CmdWith(At("sh", "diff <(sort file1) <(sort file2)"), "diff", Block()),
			),
		},
		{
			Name:  "shell with parameter expansion",
			Input: "param: @sh(echo ${VAR:-default})",
			Expected: Program(
				CmdWith(At("sh", "echo ${VAR:-default}"), "param", Block()),
			),
		},
		{
			Name:  "shell with complex conditionals",
			Input: "conditional: @sh([[ -f file.txt && -r file.txt ]] && echo readable || echo not-readable)",
			Expected: Program(
				CmdWith(At("sh", "[[ -f file.txt && -r file.txt ]] && echo readable || echo not-readable"), "conditional", Block()),
			),
		},
		{
			Name:  "shell with here document",
			Input: "heredoc: @sh(cat <<EOF\nLine 1\nLine 2\nEOF)",
			Expected: Program(
				CmdWith(At("sh", "cat <<EOF\nLine 1\nLine 2\nEOF"), "heredoc", Block()),
			),
		},
		{
			Name:  "shell with case statement",
			Input: "case-test: @sh(case $1 in start) echo starting;; stop) echo stopping;; esac)",
			Expected: Program(
				CmdWith(At("sh", "case $1 in start) echo starting;; stop) echo stopping;; esac"), "case-test", Block()),
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
				CmdWith(At("sh", "echo \"He said \\\"hello\\\" to me\""), "test-escaped", Block()),
			),
		},
		{
			Name:  "mixed quotes with parentheses",
			Input: "test-mixed-quotes: @sh(echo 'test \"$(date)\" done' && echo \"test '$(whoami)' done\")",
			Expected: Program(
				CmdWith(At("sh", "echo 'test \"$(date)\" done' && echo \"test '$(whoami)' done\""), "test-mixed-quotes", Block()),
			),
		},
		{
			Name:  "backticks with parentheses",
			Input: "test-backticks: @sh(echo `date` and $(whoami))",
			Expected: Program(
				CmdWith(At("sh", "echo `date` and $(whoami)"), "test-backticks", Block()),
			),
		},
		{
			Name:  "nested quotes with special characters",
			Input: "special-chars: @sh(echo \"Path: '$HOME' and size: $(du -sh .)\")",
			Expected: Program(
				CmdWith(At("sh", "echo \"Path: '$HOME' and size: $(du -sh .)\""), "special-chars", Block()),
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
				CmdWith(At("sh", "(cd /tmp && ls) > /dev/null"), "parens", Block()),
			),
		},
		{
			Name:  "nested parentheses with command substitution",
			Input: "nested: @sh(echo $(echo $(date +%Y)))",
			Expected: Program(
				CmdWith(At("sh", "echo $(echo $(date +%Y))"), "nested", Block()),
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
				CmdWith(At("sh", "cd @var(DIR)"), "test-var", Block()),
			),
		},
		{
			Name:  "@var with shell command substitution",
			Input: "test-var-cmd: @sh(cd @var(DIR) && echo \"$(pwd)\")",
			Expected: Program(
				CmdWith(At("sh", "cd @var(DIR) && echo \"$(pwd)\""), "test-var-cmd", Block()),
			),
		},
		{
			Name:  "multiple @var with complex shell",
			Input: "test-multi-var: @sh(if [ -d @var(SRC) ] && [ \"$(ls @var(SRC) | wc -l)\" -gt 0 ]; then echo \"Source dir has files\"; fi)",
			Expected: Program(
				CmdWith(At("sh", "if [ -d @var(SRC) ] && [ \"$(ls @var(SRC) | wc -l)\" -gt 0 ]; then echo \"Source dir has files\"; fi"), "test-multi-var", Block()),
			),
		},
		{
			Name:  "@var in shell function definition",
			Input: "func-def: @sh(deploy() { rsync -av @var(SRC)/ deploy@server.com:@var(DEST)/; }; deploy)",
			Expected: Program(
				CmdWith(At("sh", "deploy() { rsync -av @var(SRC)/ deploy@server.com:@var(DEST)/; }; deploy"), "func-def", Block()),
			),
		},
		{
			Name:  "@var in shell array",
			Input: "array-var: @sh(FILES=(@var(FILE1) @var(FILE2) @var(FILE3)); echo ${FILES[@]})",
			Expected: Program(
				CmdWith(At("sh", "FILES=(@var(FILE1) @var(FILE2) @var(FILE3)); echo ${FILES[@]}"), "array-var", Block()),
			),
		},
		{
			Name:  "@var in shell case statement",
			Input: "case-var: @sh(case @var(ENV) in prod) echo production;; dev) echo development;; esac)",
			Expected: Program(
				CmdWith(At("sh", "case @var(ENV) in prod) echo production;; dev) echo development;; esac"), "case-var", Block()),
			),
		},
		{
			Name:  "@var in shell parameter expansion",
			Input: "param-expansion: @sh(echo ${@var(VAR):-@var(DEFAULT)})",
			Expected: Program(
				CmdWith(At("sh", "echo ${@var(VAR):-@var(DEFAULT)}"), "param-expansion", Block()),
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
					"echo hello world",
					"echo done",
				)),
			),
		},
		{
			Name:  "continuation with decorators",
			Input: "deploy: @sh(docker build \\\n-t myapp \\\n.)",
			Expected: Program(
				CmdWith(At("sh", "docker build -t myapp ."), "deploy", Block()),
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
	}

	for _, tc := range testCases {
		RunTestCase(t, tc)
	}
}
