package parser

import (
	"testing"
)

// Test that @ symbols in email addresses are treated as regular text, not decorators
func TestAtSymbolInEmailAddresses(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "email in echo command",
			Input: "notify: echo 'Build failed' | mail admin@company.com",
			Expected: Program(
				Cmd("notify", Simple(Text("echo 'Build failed' | mail admin@company.com"))),
			),
		},
		{
			Name:  "email in git command",
			Input: "commit: git log --author=john@company.com",
			Expected: Program(
				Cmd("commit", Simple(Text("git log --author=john@company.com"))),
			),
		},
		{
			Name:  "multiple emails in command",
			Input: "notify-all: mail admin@company.com,dev@company.com < report.txt",
			Expected: Program(
				Cmd("notify-all", Simple(Text("mail admin@company.com,dev@company.com < report.txt"))),
			),
		},
		{
			Name:  "email with special characters",
			Input: "send: sendmail test+user@example.org",
			Expected: Program(
				Cmd("send", Simple(Text("sendmail test+user@example.org"))),
			),
		},
		{
			Name:  "email with subdomain",
			Input: "alert: echo 'Error' | mail ops@api.company.com",
			Expected: Program(
				Cmd("alert", Simple(Text("echo 'Error' | mail ops@api.company.com"))),
			),
		},
		{
			Name:  "email with numbers",
			Input: "notify: echo 'Build' | mail admin123@company123.com",
			Expected: Program(
				Cmd("notify", Simple(Text("echo 'Build' | mail admin123@company123.com"))),
			),
		},
		{
			Name:  "email with underscores and hyphens",
			Input: "send: mail test_user@company-name.org",
			Expected: Program(
				Cmd("send", Simple(Text("mail test_user@company-name.org"))),
			),
		},
		{
			Name:  "email in quoted string",
			Input: "notify: echo \"Send to admin@company.com for help\"",
			Expected: Program(
				Cmd("notify", Simple(Text("echo \"Send to admin@company.com for help\""))),
			),
		},
		{
			Name:  "email in single quoted string",
			Input: "notify: echo 'Contact admin@company.com'",
			Expected: Program(
				Cmd("notify", Simple(Text("echo 'Contact admin@company.com'"))),
			),
		},
		{
			Name:  "multiple emails in one command",
			Input: "notify: echo 'Send to admin@company.com and dev@company.com'",
			Expected: Program(
				Cmd("notify", Simple(Text("echo 'Send to admin@company.com and dev@company.com'"))),
			),
		},
	}

	for _, tc := range testCases {
		RunTestCase(t, tc)
	}
}

// Test that @ symbols in SSH commands are treated as regular text
func TestAtSymbolInSSHCommands(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "ssh user@host",
			Input: "deploy: ssh deploy@server.com 'systemctl restart api'",
			Expected: Program(
				Cmd("deploy", Simple(Text("ssh deploy@server.com 'systemctl restart api'"))),
			),
		},
		{
			Name:  "scp with user@host",
			Input: "upload: scp ./app user@remote.com:/opt/app/",
			Expected: Program(
				Cmd("upload", Simple(Text("scp ./app user@remote.com:/opt/app/"))),
			),
		},
		{
			Name:  "rsync with user@host",
			Input: "sync: rsync -av ./ backup@storage.com:/backups/",
			Expected: Program(
				Cmd("sync", Simple(Text("rsync -av ./ backup@storage.com:/backups/"))),
			),
		},
		{
			Name:  "ssh with port specification",
			Input: "connect: ssh -p 2222 user@remote.example.com",
			Expected: Program(
				Cmd("connect", Simple(Text("ssh -p 2222 user@remote.example.com"))),
			),
		},
		{
			Name:  "scp with specific port",
			Input: "secure-copy: scp -P 2222 file.txt user@server.com:/path/",
			Expected: Program(
				Cmd("secure-copy", Simple(Text("scp -P 2222 file.txt user@server.com:/path/"))),
			),
		},
		{
			Name:  "ssh with complex command",
			Input: "remote-build: ssh build@ci.company.com 'cd /builds && make clean && make all'",
			Expected: Program(
				Cmd("remote-build", Simple(Text("ssh build@ci.company.com 'cd /builds && make clean && make all'"))),
			),
		},
		{
			Name:  "ssh tunnel",
			Input: "tunnel: ssh -L 8080:localhost:8080 user@gateway.com",
			Expected: Program(
				Cmd("tunnel", Simple(Text("ssh -L 8080:localhost:8080 user@gateway.com"))),
			),
		},
		{
			Name:  "ssh with key file",
			Input: "secure-connect: ssh -i ~/.ssh/key user@secure.server.com",
			Expected: Program(
				Cmd("secure-connect", Simple(Text("ssh -i ~/.ssh/key user@secure.server.com"))),
			),
		},
		{
			Name:  "rsync with ssh options",
			Input: "backup: rsync -av -e 'ssh -p 2222' ./ user@backup.com:/data/",
			Expected: Program(
				Cmd("backup", Simple(Text("rsync -av -e 'ssh -p 2222' ./ user@backup.com:/data/"))),
			),
		},
		{
			Name:  "multiple ssh commands",
			Input: "multi-deploy: ssh app@server1.com 'restart-app' && ssh app@server2.com 'restart-app'",
			Expected: Program(
				Cmd("multi-deploy", Simple(Text("ssh app@server1.com 'restart-app' && ssh app@server2.com 'restart-app'"))),
			),
		},
	}

	for _, tc := range testCases {
		RunTestCase(t, tc)
	}
}

// Test @ symbols in shell command substitution patterns
func TestAtSymbolInShellSubstitution(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "shell command substitution with @",
			Input: "permissions: docker run --user @(id -u):@(id -g) ubuntu",
			Expected: Program(
				Cmd("permissions", Simple(Text("docker run --user @(id -u):@(id -g) ubuntu"))),
			),
		},
		{
			Name:  "shell parameter expansion with @",
			Input: "array: echo @{array[@]}",
			Expected: Program(
				Cmd("array", Simple(Text("echo @{array[@]}"))),
			),
		},
		{
			Name:  "bash array substitution",
			Input: "process-all: for item in @{items[@]}; do process $item; done",
			Expected: Program(
				Cmd("process-all", Simple(Text("for item in @{items[@]}; do process $item; done"))),
			),
		},
		{
			Name:  "complex shell substitution",
			Input: "check: test @(echo $USER) = @{EXPECTED_USER:-admin}",
			Expected: Program(
				Cmd("check", Simple(Text("test @(echo $USER) = @{EXPECTED_USER:-admin}"))),
			),
		},
		{
			Name:  "nested shell substitution",
			Input: "complex: echo @(echo @(date +%Y) is current year)",
			Expected: Program(
				Cmd("complex", Simple(Text("echo @(echo @(date +%Y) is current year)"))),
			),
		},
		{
			Name:  "arithmetic expansion with @",
			Input: "math: echo Result is @((2 + 3 * 4))",
			Expected: Program(
				Cmd("math", Simple(Text("echo Result is @((2 + 3 * 4))"))),
			),
		},
		{
			Name:  "process substitution with @",
			Input: "diff-dirs: diff @(ls dir1) @(ls dir2)",
			Expected: Program(
				Cmd("diff-dirs", Simple(Text("diff @(ls dir1) @(ls dir2)"))),
			),
		},
		{
			Name:  "command substitution in string",
			Input: "info: echo \"Current time is @(date) and user is @(whoami)\"",
			Expected: Program(
				Cmd("info", Simple(Text("echo \"Current time is @(date) and user is @(whoami)\""))),
			),
		},
	}

	for _, tc := range testCases {
		RunTestCase(t, tc)
	}
}

// Test @ symbols in various other contexts that should NOT be decorators
func TestAtSymbolInOtherContexts(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "at symbol in URL",
			Input: "download: curl https://api@service.com/data",
			Expected: Program(
				Cmd("download", Simple(Text("curl https://api@service.com/data"))),
			),
		},
		{
			Name:  "at symbol in timestamp or ID",
			Input: "tag: git tag v1.0@$(date +%s)",
			Expected: Program(
				Cmd("tag", Simple(Text("git tag v1.0@$(date +%s)"))),
			),
		},
		{
			Name:  "at symbol in file path or reference",
			Input: "checkout: git show HEAD@{2}",
			Expected: Program(
				Cmd("checkout", Simple(Text("git show HEAD@{2}"))),
			),
		},
		{
			Name:  "at symbol in literal strings with emails - but @var should still work",
			Input: "test: echo 'Contact @var(SUPPORT_USER) @ support@company.com'",
			Expected: Program(
				Cmd("test", Simple(
					Text("echo 'Contact "),
					At("var", "SUPPORT_USER"),
					Text(" @ support@company.com'"),
				)),
			),
		},
		{
			Name:  "at symbol without parentheses or braces",
			Input: "script: ./run.sh @ production",
			Expected: Program(
				Cmd("script", Simple(Text("./run.sh @ production"))),
			),
		},
		{
			Name:  "shell variables should work alongside @var",
			Input: "mixed: echo \"User: $USER, Project: @var(PROJECT), Home: $HOME\"",
			Expected: Program(
				Cmd("mixed", Simple(
					Text("echo \"User: $USER, Project: "),
					At("var", "PROJECT"),
					Text(", Home: $HOME\""),
				)),
			),
		},
		{
			Name:  "shell command substitution should work alongside @var",
			Input: "commands: echo \"Time: $(date), Path: @var(SRC), Files: $(ls | wc -l)\"",
			Expected: Program(
				Cmd("commands", Simple(
					Text("echo \"Time: $(date), Path: "),
					At("var", "SRC"),
					Text(", Files: $(ls | wc -l)\""),
				)),
			),
		},
		{
			Name:  "at symbol in git references",
			Input: "revert: git reset --hard HEAD@{1}",
			Expected: Program(
				Cmd("revert", Simple(Text("git reset --hard HEAD@{1}"))),
			),
		},
		{
			Name:  "at symbol in URLs with auth",
			Input: "api-call: curl https://user:pass@api.service.com/endpoint",
			Expected: Program(
				Cmd("api-call", Simple(Text("curl https://user:pass@api.service.com/endpoint"))),
			),
		},
		{
			Name:  "at symbol in database connection strings",
			Input: "connect: psql postgresql://user:pass@localhost:5432/dbname",
			Expected: Program(
				Cmd("connect", Simple(Text("psql postgresql://user:pass@localhost:5432/dbname"))),
			),
		},
		{
			Name:  "at symbol in docker registry URLs",
			Input: "pull: docker pull registry@sha256:abc123def456",
			Expected: Program(
				Cmd("pull", Simple(Text("docker pull registry@sha256:abc123def456"))),
			),
		},
		{
			Name:  "at symbol in time specifications",
			Input: "schedule: at 15:30@monday echo 'Weekly reminder'",
			Expected: Program(
				Cmd("schedule", Simple(Text("at 15:30@monday echo 'Weekly reminder'"))),
			),
		},
		{
			Name:  "at symbol in network addresses",
			Input: "ping: ping host@192.168.1.100",
			Expected: Program(
				Cmd("ping", Simple(Text("ping host@192.168.1.100"))),
			),
		},
		{
			Name:  "at symbol in version tags",
			Input: "release: git tag release@v1.2.3",
			Expected: Program(
				Cmd("release", Simple(Text("git tag release@v1.2.3"))),
			),
		},
		{
			Name:  "at symbol in file names",
			Input: "backup: cp important.txt important@$(date +%Y%m%d).txt",
			Expected: Program(
				Cmd("backup", Simple(Text("cp important.txt important@$(date +%Y%m%d).txt"))),
			),
		},
	}

	for _, tc := range testCases {
		RunTestCase(t, tc)
	}
}

// Test complex mixed scenarios with both decorators and non-decorator @ symbols
func TestMixedAtSymbolScenarios(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "email and decorator in same command",
			Input: "notify: @sh(echo \"Build complete\" | mail admin@company.com)",
			Expected: Program(
				Cmd("notify", Simple(
					At("sh", "echo \"Build complete\" | mail admin@company.com"),
				)),
			),
		},
		{
			Name:  "ssh and @var decorator",
			Input: "deploy: ssh @var(DEPLOY_USER)@server.com 'restart-app'",
			Expected: Program(
				Cmd("deploy", Simple(
					Text("ssh "),
					At("var", "DEPLOY_USER"),
					Text("@server.com 'restart-app'"),
				)),
			),
		},
		{
			Name:  "database URL with @var replacement",
			Input: "connect: psql postgresql://@var(DB_USER):@var(DB_PASS)@localhost/@var(DB_NAME)",
			Expected: Program(
				Cmd("connect", Simple(
					Text("psql postgresql://"),
					At("var", "DB_USER"),
					Text(":"),
					At("var", "DB_PASS"),
					Text("@localhost/"),
					At("var", "DB_NAME"),
				)),
			),
		},
		{
			Name:  "git with email author and @var tag",
			Input: "commit: git commit --author=\"@var(AUTHOR_NAME) <author@company.com>\" -m \"@var(COMMIT_MSG)\"",
			Expected: Program(
				Cmd("commit", Simple(
					Text("git commit --author=\""),
					At("var", "AUTHOR_NAME"),
					Text(" <author@company.com>\" -m \""),
					At("var", "COMMIT_MSG"),
					Text("\""),
				)),
			),
		},
		{
			Name:  "docker run with @var user and email notification",
			Input: "run-container: docker run --user @var(USER_ID) myapp && echo 'Started' | mail admin@company.com",
			Expected: Program(
				Cmd("run-container", Simple(
					Text("docker run --user "),
					At("var", "USER_ID"),
					Text(" myapp && echo 'Started' | mail admin@company.com"),
				)),
			),
		},
		{
			Name:  "ssh tunnel with @var ports and email alert",
			Input: "secure-tunnel: ssh -L @var(LOCAL_PORT):localhost:@var(REMOTE_PORT) user@gateway.com || echo 'Tunnel failed' | mail ops@company.com",
			Expected: Program(
				Cmd("secure-tunnel", Simple(
					Text("ssh -L "),
					At("var", "LOCAL_PORT"),
					Text(":localhost:"),
					At("var", "REMOTE_PORT"),
					Text(" user@gateway.com || echo 'Tunnel failed' | mail ops@company.com"),
				)),
			),
		},
		{
			Name:  "curl with auth URL and @var token",
			Input: "api-test: curl -H \"Authorization: Bearer @var(API_TOKEN)\" https://user:pass@api.service.com/test",
			Expected: Program(
				Cmd("api-test", Simple(
					Text("curl -H \"Authorization: Bearer "),
					At("var", "API_TOKEN"),
					Text("\" https://user:pass@api.service.com/test"),
				)),
			),
		},
	}

	for _, tc := range testCases {
		RunTestCase(t, tc)
	}
}

// Test @ symbols that look like decorators but have invalid syntax patterns
func TestAtSymbolEdgeCases(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "multiple consecutive @ symbols",
			Input: "weird: echo '@@@@'",
			Expected: Program(
				Cmd("weird", Simple(Text("echo '@@@@'"))),
			),
		},
		{
			Name:  "at symbol at end of line",
			Input: "suffix: echo hello@",
			Expected: Program(
				Cmd("suffix", Simple(Text("echo hello@"))),
			),
		},
		{
			Name:  "at symbol with invalid decorator syntax - starts with number",
			Input: "invalid: echo @123invalid",
			Expected: Program(
				Cmd("invalid", Simple(Text("echo @123invalid"))),
			),
		},
		{
			Name:  "at symbol followed by special characters",
			Input: "special: echo @$#%!",
			Expected: Program(
				Cmd("special", Simple(Text("echo @$#%!"))),
			),
		},
		{
			Name:  "at symbol with incomplete decorator syntax - missing closing paren",
			Input: "incomplete: echo @partial(unclosed",
			Expected: Program(
				Cmd("incomplete", Simple(Text("echo @partial(unclosed"))),
			),
		},
		{
			Name:  "at symbol with space after @",
			Input: "spaced: echo @ variable",
			Expected: Program(
				Cmd("spaced", Simple(Text("echo @ variable"))),
			),
		},
		{
			Name:  "at symbol followed by invalid characters for decorator name",
			Input: "invalid-chars: echo @-invalid @.invalid @/invalid",
			Expected: Program(
				Cmd("invalid-chars", Simple(Text("echo @-invalid @.invalid @/invalid"))),
			),
		},
		{
			Name:  "at symbol in quoted context - @var should still work as decorator",
			Input: "quoted: echo 'Building @var(PROJECT) version @var(VERSION)'",
			Expected: Program(
				Cmd("quoted", Simple(
					Text("echo 'Building "),
					At("var", "PROJECT"),
					Text(" version "),
					At("var", "VERSION"),
					Text("'"),
				)),
			),
		},
		{
			Name:  "at symbol that looks like block decorator but missing opening brace",
			Input: "no-brace: @parallel server",
			Expected: Program(
				Cmd("no-brace", Simple(Text("@parallel server"))),
			),
		},
		{
			Name:  "at symbol with mismatched braces",
			Input: "mismatched: echo @{unclosed",
			Expected: Program(
				Cmd("mismatched", Simple(Text("echo @{unclosed"))),
			),
		},
		{
			Name:  "at symbol with empty parentheses",
			Input: "empty-parens: echo @()",
			Expected: Program(
				Cmd("empty-parens", Simple(Text("echo @()"))),
			),
		},
		{
			Name:  "at symbol with nested parentheses but invalid decorator",
			Input: "nested-invalid: echo @(echo @(date))",
			Expected: Program(
				Cmd("nested-invalid", Simple(Text("echo @(echo @(date))"))),
			),
		},
		{
			Name:  "at symbol in complex shell expression",
			Input: "complex-shell: for f in *.@{ext}; do echo $f@$(date); done",
			Expected: Program(
				Cmd("complex-shell", Simple(Text("for f in *.@{ext}; do echo $f@$(date); done"))),
			),
		},
		{
			Name:  "at symbol with unicode characters",
			Input: "unicode: echo 用户@domain.中国",
			Expected: Program(
				Cmd("unicode", Simple(Text("echo 用户@domain.中国"))),
			),
		},
		{
			Name:  "at symbol in regex patterns",
			Input: "regex: grep '@[a-zA-Z]+@[a-zA-Z.]+' emails.txt",
			Expected: Program(
				Cmd("regex", Simple(Text("grep '@[a-zA-Z]+@[a-zA-Z.]+' emails.txt"))),
			),
		},
	}

	for _, tc := range testCases {
		RunTestCase(t, tc)
	}
}
