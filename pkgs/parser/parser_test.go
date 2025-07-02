package parser

import (
	"strings"
	"testing"

	"github.com/aledsdavies/devcmd/pkgs/ast"
	"github.com/google/go-cmp/cmp"
)

// Test helper types for cleaner test definitions using new unified AST
type ExpectedProgram struct {
	Variables []ExpectedVariable
	Commands  []ExpectedCommand
}

type ExpectedVariable struct {
	Name  string
	Value ExpectedExpression
}

type ExpectedCommand struct {
	Name       string
	Type       ast.CommandType // VarCommand, WatchCommand, StopCommand
	Decorators []ExpectedDecorator
	Body       ExpectedCommandBody
}

type ExpectedCommandBody struct {
	IsBlock    bool
	Elements   []ExpectedCommandElement // For simple commands
	Statements []ExpectedStatement      // For block commands
}

type ExpectedStatement struct {
	Elements []ExpectedCommandElement
}

type ExpectedCommandElement struct {
	Type      string // "text", "decorator", "variable_ref"
	Text      string // for text elements
	Decorator *ExpectedDecorator
	VarRef    *ExpectedVariableRef
}

type ExpectedDecorator struct {
	Name string
	Args []ExpectedExpression
}

type ExpectedVariableRef struct {
	Name string
}

type ExpectedExpression struct {
	Type     string // "string", "number", "duration", "variable_ref", "identifier"
	Value    string
	VarRef   *ExpectedVariableRef // for variable references
	HasVars  bool                 // for strings with variable interpolation
	VarRefs  []ExpectedVariableRef // for string interpolation
}

// Helper functions for creating expected elements
func TextElement(text string) ExpectedCommandElement {
	return ExpectedCommandElement{
		Type: "text",
		Text: text,
	}
}

func VarRefElement(name string) ExpectedCommandElement {
	return ExpectedCommandElement{
		Type:   "variable_ref",
		VarRef: &ExpectedVariableRef{Name: name},
	}
}

func DecoratorElement(name string, args ...ExpectedExpression) ExpectedCommandElement {
	return ExpectedCommandElement{
		Type: "decorator",
		Decorator: &ExpectedDecorator{
			Name: name,
			Args: args,
		},
	}
}

func StringExpr(value string, varRefs ...ExpectedVariableRef) ExpectedExpression {
	return ExpectedExpression{
		Type:    "string",
		Value:   value,
		HasVars: len(varRefs) > 0,
		VarRefs: varRefs,
	}
}

func NumberExpr(value string) ExpectedExpression {
	return ExpectedExpression{
		Type:  "number",
		Value: value,
	}
}

func DurationExpr(value string) ExpectedExpression {
	return ExpectedExpression{
		Type:  "duration",
		Value: value,
	}
}

func IdentifierExpr(value string) ExpectedExpression {
	return ExpectedExpression{
		Type:  "identifier",
		Value: value,
	}
}

func VarRefExpr(name string) ExpectedExpression {
	return ExpectedExpression{
		Type:   "variable_ref",
		VarRef: &ExpectedVariableRef{Name: name},
	}
}

func SimpleCommandBody(elements ...ExpectedCommandElement) ExpectedCommandBody {
	return ExpectedCommandBody{
		IsBlock:  false,
		Elements: elements,
	}
}

func BlockCommandBody(statements ...ExpectedStatement) ExpectedCommandBody {
	return ExpectedCommandBody{
		IsBlock:    true,
		Statements: statements,
	}
}

func Statement(elements ...ExpectedCommandElement) ExpectedStatement {
	return ExpectedStatement{
		Elements: elements,
	}
}

func Variable(name string, value ExpectedExpression) ExpectedVariable {
	return ExpectedVariable{
		Name:  name,
		Value: value,
	}
}

func SimpleCommand(name string, decorators []ExpectedDecorator, body ExpectedCommandBody) ExpectedCommand {
	return ExpectedCommand{
		Name:       name,
		Type:       ast.Command,
		Decorators: decorators,
		Body:       body,
	}
}

func WatchCommand(name string, decorators []ExpectedDecorator, body ExpectedCommandBody) ExpectedCommand {
	return ExpectedCommand{
		Name:       name,
		Type:       ast.WatchCommand,
		Decorators: decorators,
		Body:       body,
	}
}

func StopCommand(name string, decorators []ExpectedDecorator, body ExpectedCommandBody) ExpectedCommand {
	return ExpectedCommand{
		Name:       name,
		Type:       ast.StopCommand,
		Decorators: decorators,
		Body:       body,
	}
}

// Test case structure
type TestCase struct {
	Name        string
	Input       string
	WantErr     bool
	ErrorSubstr string
	Expected    ExpectedProgram
}

// Comparison helpers
func expressionToComparable(expr ast.Expression) interface{} {
	switch e := expr.(type) {
	case *ast.StringLiteral:
		result := map[string]interface{}{
			"Type":  "string",
			"Value": e.Value,
		}
		if e.HasVariables {
			result["HasVars"] = true
			varRefs := make([]map[string]interface{}, len(e.Variables))
			for i, vr := range e.Variables {
				varRefs[i] = map[string]interface{}{
					"Name": vr.Name,
				}
			}
			result["VarRefs"] = varRefs
		}
		return result
	case *ast.NumberLiteral:
		return map[string]interface{}{
			"Type":  "number",
			"Value": e.Value,
		}
	case *ast.DurationLiteral:
		return map[string]interface{}{
			"Type":  "duration",
			"Value": e.Value,
		}
	case *ast.VariableRef:
		return map[string]interface{}{
			"Type":   "variable_ref",
			"VarRef": map[string]interface{}{"Name": e.Name},
		}
	case *ast.Identifier:
		return map[string]interface{}{
			"Type":  "identifier",
			"Value": e.Name,
		}
	default:
		return map[string]interface{}{
			"Type":  "unknown",
			"Value": expr.String(),
		}
	}
}

func expectedExpressionToComparable(expr ExpectedExpression) interface{} {
	result := map[string]interface{}{
		"Type":  expr.Type,
		"Value": expr.Value,
	}
	if expr.VarRef != nil {
		result["VarRef"] = map[string]interface{}{"Name": expr.VarRef.Name}
	}
	if expr.HasVars {
		result["HasVars"] = true
		varRefs := make([]map[string]interface{}, len(expr.VarRefs))
		for i, vr := range expr.VarRefs {
			varRefs[i] = map[string]interface{}{
				"Name": vr.Name,
			}
		}
		result["VarRefs"] = varRefs
	}
	return result
}

func commandElementToComparable(elem ast.CommandElement) interface{} {
	switch e := elem.(type) {
	case *ast.TextElement:
		return map[string]interface{}{
			"Type": "text",
			"Text": e.Text,
		}
	case *ast.Decorator:
		args := make([]interface{}, len(e.Args))
		for i, arg := range e.Args {
			args[i] = expressionToComparable(arg)
		}
		return map[string]interface{}{
			"Type": "decorator",
			"Decorator": map[string]interface{}{
				"Name": e.Name,
				"Args": args,
			},
		}
	case *ast.VariableRef:
		return map[string]interface{}{
			"Type":   "variable_ref",
			"VarRef": map[string]interface{}{"Name": e.Name},
		}
	default:
		return map[string]interface{}{
			"Type": "unknown",
			"Text": elem.String(),
		}
	}
}

func expectedCommandElementToComparable(elem ExpectedCommandElement) interface{} {
	result := map[string]interface{}{
		"Type": elem.Type,
	}

	switch elem.Type {
	case "text":
		result["Text"] = elem.Text
	case "decorator":
		if elem.Decorator != nil {
			args := make([]interface{}, len(elem.Decorator.Args))
			for i, arg := range elem.Decorator.Args {
				args[i] = expectedExpressionToComparable(arg)
			}
			result["Decorator"] = map[string]interface{}{
				"Name": elem.Decorator.Name,
				"Args": args,
			}
		}
	case "variable_ref":
		if elem.VarRef != nil {
			result["VarRef"] = map[string]interface{}{"Name": elem.VarRef.Name}
		}
	}

	return result
}

// Updated function to work with unified CommandBody structure
func commandBodyToComparable(body ast.CommandBody) interface{} {
	result := map[string]interface{}{
		"IsBlock": body.IsBlock,
	}

	if body.IsBlock {
		// For block commands, use Statements
		statements := make([]interface{}, len(body.Statements))
		for i, stmt := range body.Statements {
			if shell, ok := stmt.(*ast.ShellStatement); ok {
				elements := make([]interface{}, len(shell.Elements))
				for j, elem := range shell.Elements {
					elements[j] = commandElementToComparable(elem)
				}
				statements[i] = map[string]interface{}{
					"Elements": elements,
				}
			}
		}
		result["Statements"] = statements
	} else {
		// For simple commands, extract elements from the first (and only) statement
		if len(body.Statements) > 0 {
			if shell, ok := body.Statements[0].(*ast.ShellStatement); ok {
				elements := make([]interface{}, len(shell.Elements))
				for i, elem := range shell.Elements {
					elements[i] = commandElementToComparable(elem)
				}
				result["Elements"] = elements
			}
		} else {
			result["Elements"] = []interface{}{}
		}
	}

	return result
}

func expectedCommandBodyToComparable(body ExpectedCommandBody) interface{} {
	result := map[string]interface{}{
		"IsBlock": body.IsBlock,
	}

	if body.IsBlock {
		statements := make([]interface{}, len(body.Statements))
		for i, stmt := range body.Statements {
			elements := make([]interface{}, len(stmt.Elements))
			for j, elem := range stmt.Elements {
				elements[j] = expectedCommandElementToComparable(elem)
			}
			statements[i] = map[string]interface{}{
				"Elements": elements,
			}
		}
		result["Statements"] = statements
	} else {
		elements := make([]interface{}, len(body.Elements))
		for i, elem := range body.Elements {
			elements[i] = expectedCommandElementToComparable(elem)
		}
		result["Elements"] = elements
	}

	return result
}

func runTestCase(t *testing.T, tc TestCase) {
	t.Run(tc.Name, func(t *testing.T) {
		// Parse the input using your parser
		program, err := Parse(tc.Input) // Assuming Parse returns *ast.Program and error

		// Check error expectations
		if tc.WantErr {
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if tc.ErrorSubstr != "" && !strings.Contains(err.Error(), tc.ErrorSubstr) {
				t.Errorf("expected error containing %q, got %q", tc.ErrorSubstr, err.Error())
			}
			return
		}

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify variables
		if len(program.Variables) != len(tc.Expected.Variables) {
			t.Errorf("expected %d variables, got %d", len(tc.Expected.Variables), len(program.Variables))
		} else {
			for i, expectedVar := range tc.Expected.Variables {
				actualVar := program.Variables[i]

				actualComparable := map[string]interface{}{
					"Name":  actualVar.Name,
					"Value": expressionToComparable(actualVar.Value),
				}

				expectedComparable := map[string]interface{}{
					"Name":  expectedVar.Name,
					"Value": expectedExpressionToComparable(expectedVar.Value),
				}

				if diff := cmp.Diff(expectedComparable, actualComparable); diff != "" {
					t.Errorf("Variable[%d] mismatch (-expected +actual):\n%s", i, diff)
				}
			}
		}

		// Verify commands
		if len(program.Commands) != len(tc.Expected.Commands) {
			t.Errorf("expected %d commands, got %d", len(tc.Expected.Commands), len(program.Commands))
		} else {
			for i, expectedCmd := range tc.Expected.Commands {
				actualCmd := program.Commands[i]

				// Convert decorators
				actualDecorators := make([]interface{}, len(actualCmd.Decorators))
				for j, decorator := range actualCmd.Decorators {
					args := make([]interface{}, len(decorator.Args))
					for k, arg := range decorator.Args {
						args[k] = expressionToComparable(arg)
					}
					actualDecorators[j] = map[string]interface{}{
						"Name": decorator.Name,
						"Args": args,
					}
				}

				expectedDecorators := make([]interface{}, len(expectedCmd.Decorators))
				for j, decorator := range expectedCmd.Decorators {
					args := make([]interface{}, len(decorator.Args))
					for k, arg := range decorator.Args {
						args[k] = expectedExpressionToComparable(arg)
					}
					expectedDecorators[j] = map[string]interface{}{
						"Name": decorator.Name,
						"Args": args,
					}
				}

				actualComparable := map[string]interface{}{
					"Name":       actualCmd.Name,
					"Type":       actualCmd.Type,
					"Decorators": actualDecorators,
					"Body":       commandBodyToComparable(actualCmd.Body),
				}

				expectedComparable := map[string]interface{}{
					"Name":       expectedCmd.Name,
					"Type":       expectedCmd.Type,
					"Decorators": expectedDecorators,
					"Body":       expectedCommandBodyToComparable(expectedCmd.Body),
				}

				if diff := cmp.Diff(expectedComparable, actualComparable); diff != "" {
					t.Errorf("Command[%d] mismatch (-expected +actual):\n%s", i, diff)
				}
			}
		}
	})
}

// TESTS FOR @ SYMBOL CONTEXT-AWARE PARSING

// Test that @ symbols in email addresses are treated as regular text, not decorators
func TestAtSymbolInEmailAddresses(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "email in echo command",
			Input: "notify: echo 'Build failed' | mail admin@company.com",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("notify", nil, SimpleCommandBody(
						TextElement("echo 'Build failed' | mail admin@company.com"))),
				},
			},
		},
		{
			Name:  "email in git command",
			Input: "commit: git log --author=john@company.com",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("commit", nil, SimpleCommandBody(
						TextElement("git log --author=john@company.com"))),
				},
			},
		},
		{
			Name:  "multiple emails in command",
			Input: "notify-all: mail admin@company.com,dev@company.com < report.txt",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("notify-all", nil, SimpleCommandBody(
						TextElement("mail admin@company.com,dev@company.com < report.txt"))),
				},
			},
		},
		{
			Name:  "email with special characters",
			Input: "send: sendmail test+user@example.org",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("send", nil, SimpleCommandBody(
						TextElement("sendmail test+user@example.org"))),
				},
			},
		},
		{
			Name:  "email with subdomain",
			Input: "alert: echo 'Error' | mail ops@api.company.com",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("alert", nil, SimpleCommandBody(
						TextElement("echo 'Error' | mail ops@api.company.com"))),
				},
			},
		},
		{
			Name:  "email with numbers",
			Input: "notify: echo 'Build' | mail admin123@company123.com",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("notify", nil, SimpleCommandBody(
						TextElement("echo 'Build' | mail admin123@company123.com"))),
				},
			},
		},
		{
			Name:  "email with underscores and hyphens",
			Input: "send: mail test_user@company-name.org",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("send", nil, SimpleCommandBody(
						TextElement("mail test_user@company-name.org"))),
				},
			},
		},
		{
			Name:  "email in quoted string",
			Input: "notify: echo \"Send to admin@company.com for help\"",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("notify", nil, SimpleCommandBody(
						TextElement("echo \"Send to admin@company.com for help\""))),
				},
			},
		},
		{
			Name:  "email in single quoted string",
			Input: "notify: echo 'Contact admin@company.com'",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("notify", nil, SimpleCommandBody(
						TextElement("echo 'Contact admin@company.com'"))),
				},
			},
		},
		{
			Name:  "multiple emails in one command",
			Input: "notify: echo 'Send to admin@company.com and dev@company.com'",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("notify", nil, SimpleCommandBody(
						TextElement("echo 'Send to admin@company.com and dev@company.com'"))),
				},
			},
		},
	}

	for _, tc := range testCases {
		runTestCase(t, tc)
	}
}

// Test that @ symbols in SSH commands are treated as regular text
func TestAtSymbolInSSHCommands(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "ssh user@host",
			Input: "deploy: ssh deploy@server.com 'systemctl restart api'",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("deploy", nil, SimpleCommandBody(
						TextElement("ssh deploy@server.com 'systemctl restart api'"))),
				},
			},
		},
		{
			Name:  "scp with user@host",
			Input: "upload: scp ./app user@remote.com:/opt/app/",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("upload", nil, SimpleCommandBody(
						TextElement("scp ./app user@remote.com:/opt/app/"))),
				},
			},
		},
		{
			Name:  "rsync with user@host",
			Input: "sync: rsync -av ./ backup@storage.com:/backups/",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("sync", nil, SimpleCommandBody(
						TextElement("rsync -av ./ backup@storage.com:/backups/"))),
				},
			},
		},
		{
			Name:  "ssh with port specification",
			Input: "connect: ssh -p 2222 user@remote.example.com",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("connect", nil, SimpleCommandBody(
						TextElement("ssh -p 2222 user@remote.example.com"))),
				},
			},
		},
		{
			Name:  "scp with specific port",
			Input: "secure-copy: scp -P 2222 file.txt user@server.com:/path/",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("secure-copy", nil, SimpleCommandBody(
						TextElement("scp -P 2222 file.txt user@server.com:/path/"))),
				},
			},
		},
		{
			Name:  "ssh with complex command",
			Input: "remote-build: ssh build@ci.company.com 'cd /builds && make clean && make all'",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("remote-build", nil, SimpleCommandBody(
						TextElement("ssh build@ci.company.com 'cd /builds && make clean && make all'"))),
				},
			},
		},
		{
			Name:  "ssh tunnel",
			Input: "tunnel: ssh -L 8080:localhost:8080 user@gateway.com",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("tunnel", nil, SimpleCommandBody(
						TextElement("ssh -L 8080:localhost:8080 user@gateway.com"))),
				},
			},
		},
		{
			Name:  "ssh with key file",
			Input: "secure-connect: ssh -i ~/.ssh/key user@secure.server.com",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("secure-connect", nil, SimpleCommandBody(
						TextElement("ssh -i ~/.ssh/key user@secure.server.com"))),
				},
			},
		},
		{
			Name:  "rsync with ssh options",
			Input: "backup: rsync -av -e 'ssh -p 2222' ./ user@backup.com:/data/",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("backup", nil, SimpleCommandBody(
						TextElement("rsync -av -e 'ssh -p 2222' ./ user@backup.com:/data/"))),
				},
			},
		},
		{
			Name:  "multiple ssh commands",
			Input: "multi-deploy: ssh app@server1.com 'restart-app' && ssh app@server2.com 'restart-app'",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("multi-deploy", nil, SimpleCommandBody(
						TextElement("ssh app@server1.com 'restart-app' && ssh app@server2.com 'restart-app'"))),
				},
			},
		},
	}

	for _, tc := range testCases {
		runTestCase(t, tc)
	}
}

// Test @ symbols in shell command substitution patterns
func TestAtSymbolInShellSubstitution(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "shell command substitution with @",
			Input: "permissions: docker run --user @(id -u):@(id -g) ubuntu",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("permissions", nil, SimpleCommandBody(
						TextElement("docker run --user @(id -u):@(id -g) ubuntu"))),
				},
			},
		},
		{
			Name:  "shell parameter expansion with @",
			Input: "array: echo @{array[@]}",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("array", nil, SimpleCommandBody(
						TextElement("echo @{array[@]}"))),
				},
			},
		},
		{
			Name:  "bash array substitution",
			Input: "process-all: for item in @{items[@]}; do process $item; done",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("process-all", nil, SimpleCommandBody(
						TextElement("for item in @{items[@]}; do process $item; done"))),
				},
			},
		},
		{
			Name:  "complex shell substitution",
			Input: "check: test @(echo $USER) = @{EXPECTED_USER:-admin}",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("check", nil, SimpleCommandBody(
						TextElement("test @(echo $USER) = @{EXPECTED_USER:-admin}"))),
				},
			},
		},
		{
			Name:  "nested shell substitution",
			Input: "complex: echo @(echo @(date +%Y) is current year)",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("complex", nil, SimpleCommandBody(
						TextElement("echo @(echo @(date +%Y) is current year)"))),
				},
			},
		},
		{
			Name:  "arithmetic expansion with @",
			Input: "math: echo Result is @((2 + 3 * 4))",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("math", nil, SimpleCommandBody(
						TextElement("echo Result is @((2 + 3 * 4))"))),
				},
			},
		},
		{
			Name:  "process substitution with @",
			Input: "diff-dirs: diff @(ls dir1) @(ls dir2)",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("diff-dirs", nil, SimpleCommandBody(
						TextElement("diff @(ls dir1) @(ls dir2)"))),
				},
			},
		},
		{
			Name:  "command substitution in string",
			Input: "info: echo \"Current time is @(date) and user is @(whoami)\"",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("info", nil, SimpleCommandBody(
						TextElement("echo \"Current time is @(date) and user is @(whoami)\""))),
				},
			},
		},
	}

	for _, tc := range testCases {
		runTestCase(t, tc)
	}
}

// Test @ symbols in various other contexts that should NOT be decorators
func TestAtSymbolInOtherContexts(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "at symbol in URL",
			Input: "download: curl https://api@service.com/data",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("download", nil, SimpleCommandBody(
						TextElement("curl https://api@service.com/data"))),
				},
			},
		},
		{
			Name:  "at symbol in timestamp or ID",
			Input: "tag: git tag v1.0@$(date +%s)",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("tag", nil, SimpleCommandBody(
						TextElement("git tag v1.0@$(date +%s)"))),
				},
			},
		},
		{
			Name:  "at symbol in file path or reference",
			Input: "checkout: git show HEAD@{2}",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("checkout", nil, SimpleCommandBody(
						TextElement("git show HEAD@{2}"))),
				},
			},
		},
		{
			Name:  "at symbol in literal strings with emails - but @var should still work",
			Input: "test: echo 'Contact @var(SUPPORT_USER) @ support@company.com'",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("test", nil, SimpleCommandBody(
						TextElement("echo 'Contact "),
						VarRefElement("SUPPORT_USER"),
						TextElement(" @ support@company.com'"))),
				},
			},
		},
		{
			Name:  "at symbol without parentheses or braces",
			Input: "script: ./run.sh @ production",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("script", nil, SimpleCommandBody(
						TextElement("./run.sh @ production"))),
				},
			},
		},
		{
			Name:  "shell variables should work alongside @var",
			Input: "mixed: echo \"User: $USER, Project: @var(PROJECT), Home: $HOME\"",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("mixed", nil, SimpleCommandBody(
						TextElement("echo \"User: $USER, Project: "),
						VarRefElement("PROJECT"),
						TextElement(", Home: $HOME\""))),
				},
			},
		},
		{
			Name:  "shell command substitution should work alongside @var",
			Input: "commands: echo \"Time: $(date), Path: @var(SRC), Files: $(ls | wc -l)\"",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("commands", nil, SimpleCommandBody(
						TextElement("echo \"Time: $(date), Path: "),
						VarRefElement("SRC"),
						TextElement(", Files: $(ls | wc -l)\""))),
				},
			},
		},
		{
			Name:  "at symbol in git references",
			Input: "revert: git reset --hard HEAD@{1}",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("revert", nil, SimpleCommandBody(
						TextElement("git reset --hard HEAD@{1}"))),
				},
			},
		},
		{
			Name:  "at symbol in URLs with auth",
			Input: "api-call: curl https://user:pass@api.service.com/endpoint",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("api-call", nil, SimpleCommandBody(
						TextElement("curl https://user:pass@api.service.com/endpoint"))),
				},
			},
		},
		{
			Name:  "at symbol in database connection strings",
			Input: "connect: psql postgresql://user:pass@localhost:5432/dbname",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("connect", nil, SimpleCommandBody(
						TextElement("psql postgresql://user:pass@localhost:5432/dbname"))),
				},
			},
		},
		{
			Name:  "at symbol in docker registry URLs",
			Input: "pull: docker pull registry@sha256:abc123def456",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("pull", nil, SimpleCommandBody(
						TextElement("docker pull registry@sha256:abc123def456"))),
				},
			},
		},
		{
			Name:  "at symbol in time specifications",
			Input: "schedule: at 15:30@monday echo 'Weekly reminder'",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("schedule", nil, SimpleCommandBody(
						TextElement("at 15:30@monday echo 'Weekly reminder'"))),
				},
			},
		},
		{
			Name:  "at symbol in network addresses",
			Input: "ping: ping host@192.168.1.100",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("ping", nil, SimpleCommandBody(
						TextElement("ping host@192.168.1.100"))),
				},
			},
		},
		{
			Name:  "at symbol in version tags",
			Input: "release: git tag release@v1.2.3",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("release", nil, SimpleCommandBody(
						TextElement("git tag release@v1.2.3"))),
				},
			},
		},
		{
			Name:  "at symbol in file names",
			Input: "backup: cp important.txt important@$(date +%Y%m%d).txt",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("backup", nil, SimpleCommandBody(
						TextElement("cp important.txt important@$(date +%Y%m%d).txt"))),
				},
			},
		},
	}

	for _, tc := range testCases {
		runTestCase(t, tc)
	}
}

// Test that valid decorators are still properly parsed as decorators
func TestValidDecoratorsStillWork(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "valid @var decorator",
			Input: "build: cd @var(SRC)",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("build", nil, SimpleCommandBody(
						TextElement("cd "),
						VarRefElement("SRC"))),
				},
			},
		},
		{
			Name:  "valid @sh function decorator",
			Input: "test: @sh(echo hello)",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("test", []ExpectedDecorator{
						{Name: "sh", Args: []ExpectedExpression{StringExpr("echo hello")}},
					}, BlockCommandBody()),
				},
			},
		},
		{
			Name:  "valid @parallel block decorator",
			Input: "services: @parallel { server; client }",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("services", []ExpectedDecorator{
						{Name: "parallel", Args: []ExpectedExpression{}},
					}, BlockCommandBody(
						Statement(TextElement("server")),
						Statement(TextElement("client")))),
				},
			},
		},
		{
			Name:  "valid @timeout function decorator",
			Input: "deploy: @timeout(30s) { echo deploying }",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("deploy", []ExpectedDecorator{
						{Name: "timeout", Args: []ExpectedExpression{DurationExpr("30s")}},
					}, BlockCommandBody(
						Statement(TextElement("echo deploying")))),
				},
			},
		},
		{
			Name:  "valid @retry block decorator",
			Input: "flaky-test: @retry { npm test; echo 'done' }",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("flaky-test", []ExpectedDecorator{
						{Name: "retry", Args: []ExpectedExpression{}},
					}, BlockCommandBody(
						Statement(TextElement("npm test")),
						Statement(TextElement("echo 'done'")))),
				},
			},
		},
		{
			Name:  "valid @watch-files block decorator",
			Input: "monitor: @watch-files { echo 'checking'; sleep 1 }",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("monitor", []ExpectedDecorator{
						{Name: "watch-files", Args: []ExpectedExpression{}},
					}, BlockCommandBody(
						Statement(TextElement("echo 'checking'")),
						Statement(TextElement("sleep 1")))),
				},
			},
		},
		{
			Name:  "valid @env decorator with argument",
			Input: "setup: @env(NODE_ENV=production) { npm start }",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("setup", []ExpectedDecorator{
						{Name: "env", Args: []ExpectedExpression{IdentifierExpr("NODE_ENV=production")}},
					}, BlockCommandBody(
						Statement(TextElement("npm start")))),
				},
			},
		},
		{
			Name:  "valid @confirm decorator",
			Input: "dangerous: @confirm(\"Are you sure?\") { rm -rf /tmp/* }",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("dangerous", []ExpectedDecorator{
						{Name: "confirm", Args: []ExpectedExpression{StringExpr("Are you sure?")}},
					}, BlockCommandBody(
						Statement(TextElement("rm -rf /tmp/*")))),
				},
			},
		},
		{
			Name:  "valid @debounce decorator",
			Input: "watch-changes: @debounce(500ms) { npm run build }",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("watch-changes", []ExpectedDecorator{
						{Name: "debounce", Args: []ExpectedExpression{DurationExpr("500ms")}},
					}, BlockCommandBody(
						Statement(TextElement("npm run build")))),
				},
			},
		},
		{
			Name:  "valid @cwd decorator",
			Input: "build-lib: @cwd(./lib) { make all }",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("build-lib", []ExpectedDecorator{
						{Name: "cwd", Args: []ExpectedExpression{StringExpr("./lib")}},
					}, BlockCommandBody(
						Statement(TextElement("make all")))),
				},
			},
		},
	}

	for _, tc := range testCases {
		runTestCase(t, tc)
	}
}

// Test complex mixed scenarios with both decorators and non-decorator @ symbols
func TestMixedAtSymbolScenarios(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "email and decorator in same command",
			Input: "notify: @sh(echo \"Build complete\" | mail admin@company.com)",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("notify", []ExpectedDecorator{
						{Name: "sh", Args: []ExpectedExpression{
							StringExpr("echo \"Build complete\" | mail admin@company.com"),
						}},
					}, BlockCommandBody()),
				},
			},
		},
		{
			Name:  "ssh and @var decorator",
			Input: "deploy: ssh @var(DEPLOY_USER)@server.com 'restart-app'",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("deploy", nil, SimpleCommandBody(
						TextElement("ssh "),
						VarRefElement("DEPLOY_USER"),
						TextElement("@server.com 'restart-app'"))),
				},
			},
		},
		{
			Name: "block with mixed @ usage including block decorators",
			Input: `complex: {
        echo "Starting deployment..."
        ssh deploy@server.com 'mkdir -p @var(APP_DIR)'
        @sh(rsync -av ./ deploy@server.com:@var(APP_DIR)/)
        @parallel {
          echo "Process 1"
          echo "Process 2"
        }
        echo "Deployed to user@server.com"
      }`,
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("complex", nil, BlockCommandBody(
						Statement(TextElement("echo \"Starting deployment...\"")),
						Statement(TextElement("ssh deploy@server.com 'mkdir -p "), VarRefElement("APP_DIR"), TextElement("'")),
						Statement(DecoratorElement("sh", StringExpr("rsync -av ./ deploy@server.com:", ExpectedVariableRef{Name: "APP_DIR"}))),
						Statement(DecoratorElement("parallel")),
						Statement(TextElement("echo \"Deployed to user@server.com\"")))),
				},
			},
		},
		{
			Name:  "database URL with @var replacement",
			Input: "connect: psql postgresql://@var(DB_USER):@var(DB_PASS)@localhost/@var(DB_NAME)",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("connect", nil, SimpleCommandBody(
						TextElement("psql postgresql://"),
						VarRefElement("DB_USER"),
						TextElement(":"),
						VarRefElement("DB_PASS"),
						TextElement("@localhost/"),
						VarRefElement("DB_NAME"))),
				},
			},
		},
		{
			Name:  "git with email author and @var tag",
			Input: "commit: git commit --author=\"@var(AUTHOR_NAME) <author@company.com>\" -m \"@var(COMMIT_MSG)\"",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("commit", nil, SimpleCommandBody(
						TextElement("git commit --author=\""),
						VarRefElement("AUTHOR_NAME"),
						TextElement(" <author@company.com>\" -m \""),
						VarRefElement("COMMIT_MSG"),
						TextElement("\""))),
				},
			},
		},
		{
			Name:  "docker run with @var user and email notification",
			Input: "run-container: docker run --user @var(USER_ID) myapp && echo 'Started' | mail admin@company.com",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("run-container", nil, SimpleCommandBody(
						TextElement("docker run --user "),
						VarRefElement("USER_ID"),
						TextElement(" myapp && echo 'Started' | mail admin@company.com"))),
				},
			},
		},
		{
			Name:  "ssh tunnel with @var ports and email alert",
			Input: "secure-tunnel: ssh -L @var(LOCAL_PORT):localhost:@var(REMOTE_PORT) user@gateway.com || echo 'Tunnel failed' | mail ops@company.com",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("secure-tunnel", nil, SimpleCommandBody(
						TextElement("ssh -L "),
						VarRefElement("LOCAL_PORT"),
						TextElement(":localhost:"),
						VarRefElement("REMOTE_PORT"),
						TextElement(" user@gateway.com || echo 'Tunnel failed' | mail ops@company.com"))),
				},
			},
		},
		{
			Name:  "curl with auth URL and @var token",
			Input: "api-test: curl -H \"Authorization: Bearer @var(API_TOKEN)\" https://user:pass@api.service.com/test",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("api-test", nil, SimpleCommandBody(
						TextElement("curl -H \"Authorization: Bearer "),
						VarRefElement("API_TOKEN"),
						TextElement("\" https://user:pass@api.service.com/test"))),
				},
			},
		},
	}

	for _, tc := range testCases {
		runTestCase(t, tc)
	}
}

// Test @ symbols that look like decorators but have invalid syntax patterns
func TestAtSymbolEdgeCases(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "multiple consecutive @ symbols",
			Input: "weird: echo '@@@@'",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("weird", nil, SimpleCommandBody(
						TextElement("echo '@@@@'"))),
				},
			},
		},
		{
			Name:  "at symbol at end of line",
			Input: "suffix: echo hello@",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("suffix", nil, SimpleCommandBody(
						TextElement("echo hello@"))),
				},
			},
		},
		{
			Name:  "at symbol with invalid decorator syntax - starts with number",
			Input: "invalid: echo @123invalid",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("invalid", nil, SimpleCommandBody(
						TextElement("echo @123invalid"))),
				},
			},
		},
		{
			Name:  "at symbol followed by special characters",
			Input: "special: echo @$#%!",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("special", nil, SimpleCommandBody(
						TextElement("echo @$#%!"))),
				},
			},
		},
		{
			Name:  "at symbol with incomplete decorator syntax - missing closing paren",
			Input: "incomplete: echo @partial(unclosed",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("incomplete", nil, SimpleCommandBody(
						TextElement("echo @partial(unclosed"))),
				},
			},
		},
		{
			Name:  "at symbol with space after @",
			Input: "spaced: echo @ variable",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("spaced", nil, SimpleCommandBody(
						TextElement("echo @ variable"))),
				},
			},
		},
		{
			Name:  "at symbol followed by invalid characters for decorator name",
			Input: "invalid-chars: echo @-invalid @.invalid @/invalid",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("invalid-chars", nil, SimpleCommandBody(
						TextElement("echo @-invalid @.invalid @/invalid"))),
				},
			},
		},
		{
			Name:  "at symbol in quoted context - @var should still work as decorator",
			Input: "quoted: echo 'Building @var(PROJECT) version @var(VERSION)'",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("quoted", nil, SimpleCommandBody(
						TextElement("echo 'Building "),
						VarRefElement("PROJECT"),
						TextElement(" version "),
						VarRefElement("VERSION"),
						TextElement("'"))),
				},
			},
		},
		{
			Name:  "at symbol that looks like block decorator but missing opening brace",
			Input: "no-brace: @parallel server",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("no-brace", nil, SimpleCommandBody(
						TextElement("@parallel server"))),
				},
			},
		},
		{
			Name:  "at symbol with mismatched braces",
			Input: "mismatched: echo @{unclosed",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("mismatched", nil, SimpleCommandBody(
						TextElement("echo @{unclosed"))),
				},
			},
		},
		{
			Name:  "at symbol with empty parentheses",
			Input: "empty-parens: echo @()",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("empty-parens", nil, SimpleCommandBody(
						TextElement("echo @()"))),
				},
			},
		},
		{
			Name:  "at symbol with nested parentheses but invalid decorator",
			Input: "nested-invalid: echo @(echo @(date))",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("nested-invalid", nil, SimpleCommandBody(
						TextElement("echo @(echo @(date))"))),
				},
			},
		},
		{
			Name:  "at symbol in complex shell expression",
			Input: "complex-shell: for f in *.@{ext}; do echo $f@$(date); done",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("complex-shell", nil, SimpleCommandBody(
						TextElement("for f in *.@{ext}; do echo $f@$(date); done"))),
				},
			},
		},
		{
			Name:  "at symbol with unicode characters",
			Input: "unicode: echo 用户@domain.中国",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("unicode", nil, SimpleCommandBody(
						TextElement("echo 用户@domain.中国"))),
				},
			},
		},
		{
			Name:  "at symbol in regex patterns",
			Input: "regex: grep '@[a-zA-Z]+@[a-zA-Z.]+' emails.txt",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("regex", nil, SimpleCommandBody(
						TextElement("grep '@[a-zA-Z]+@[a-zA-Z.]+' emails.txt"))),
				},
			},
		},
	}

	for _, tc := range testCases {
		runTestCase(t, tc)
	}
}

// EXISTING TESTS UPDATED FOR NEW AST

// Main test functions with updated expectations for the new parser
func TestBasicCommands(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "simple command",
			Input: "build: echo hello",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("build", nil, SimpleCommandBody(
						TextElement("echo hello"))),
				},
			},
		},
		{
			Name:  "command with special characters",
			Input: "run: echo 'Hello, World!'",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("run", nil, SimpleCommandBody(
						TextElement("echo 'Hello, World!'"))),
				},
			},
		},
		{
			Name:  "empty command",
			Input: "noop:",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("noop", nil, SimpleCommandBody()),
				},
			},
		},
		{
			Name:  "command with parentheses",
			Input: "check: (echo test)",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("check", nil, SimpleCommandBody(
						TextElement("(echo test)"))),
				},
			},
		},
		{
			Name:  "command with pipes",
			Input: "process: echo hello | grep hello",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("process", nil, SimpleCommandBody(
						TextElement("echo hello | grep hello"))),
				},
			},
		},
		{
			Name:  "command with redirection",
			Input: "save: echo hello > output.txt",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("save", nil, SimpleCommandBody(
						TextElement("echo hello > output.txt"))),
				},
			},
		},
		{
			Name:  "command with background process",
			Input: "background: sleep 10 &",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("background", nil, SimpleCommandBody(
						TextElement("sleep 10 &"))),
				},
			},
		},
		{
			Name:  "command with logical operators",
			Input: "conditional: test -f file.txt && echo exists || echo missing",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("conditional", nil, SimpleCommandBody(
						TextElement("test -f file.txt && echo exists || echo missing"))),
				},
			},
		},
		{
			Name:  "command with environment variables",
			Input: "env-test: NODE_ENV=production npm start",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("env-test", nil, SimpleCommandBody(
						TextElement("NODE_ENV=production npm start"))),
				},
			},
		},
		{
			Name:  "command with complex shell syntax",
			Input: "complex: for i in {1..5}; do echo $i; done",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("complex", nil, SimpleCommandBody(
						TextElement("for i in {1..5}; do echo $i; done"))),
				},
			},
		},
		{
			Name:  "command with tabs and mixed whitespace",
			Input: "build:\t\techo\t\"building\" \t&& \tmake",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("build", nil, SimpleCommandBody(
						TextElement("echo\t\"building\" \t&& \tmake"))),
				},
			},
		},
		{
			Name:  "command name with underscores and hyphens",
			Input: "test_command-name: echo hello",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("test_command-name", nil, SimpleCommandBody(
						TextElement("echo hello"))),
				},
			},
		},
	}

	for _, tc := range testCases {
		runTestCase(t, tc)
	}
}

func TestVarDecorators(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "simple @var() reference",
			Input: "build: cd @var(SRC)",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("build", nil, SimpleCommandBody(
						TextElement("cd "),
						VarRefElement("SRC"))),
				},
			},
		},
		{
			Name:  "multiple @var() references",
			Input: "deploy: docker build -t @var(IMAGE):@var(TAG)",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("deploy", nil, SimpleCommandBody(
						TextElement("docker build -t "),
						VarRefElement("IMAGE"),
						TextElement(":"),
						VarRefElement("TAG"))),
				},
			},
		},
		{
			Name:  "@var() in quoted string",
			Input: "echo: echo \"Building @var(PROJECT) version @var(VERSION)\"",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("echo", nil, SimpleCommandBody(
						TextElement("echo \"Building "),
						VarRefElement("PROJECT"),
						TextElement(" version "),
						VarRefElement("VERSION"),
						TextElement("\""))),
				},
			},
		},
		{
			Name:  "mixed @var() and shell variables",
			Input: "info: echo \"Project: @var(NAME), User: $USER\"",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("info", nil, SimpleCommandBody(
						TextElement("echo \"Project: "),
						VarRefElement("NAME"),
						TextElement(", User: $USER\""))),
				},
			},
		},
		{
			Name:  "@var() in file paths",
			Input: "copy: cp @var(SRC)/*.go @var(DEST)/",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("copy", nil, SimpleCommandBody(
						TextElement("cp "),
						VarRefElement("SRC"),
						TextElement("/*.go "),
						VarRefElement("DEST"),
						TextElement("/"))),
				},
			},
		},
		{
			Name:  "@var() in command arguments",
			Input: "serve: go run main.go --port=@var(PORT) --host=@var(HOST)",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("serve", nil, SimpleCommandBody(
						TextElement("go run main.go --port="),
						VarRefElement("PORT"),
						TextElement(" --host="),
						VarRefElement("HOST"))),
				},
			},
		},
		{
			Name:  "@var() with special characters in value",
			Input: "url: curl \"@var(API_URL)/users?filter=@var(FILTER)\"",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("url", nil, SimpleCommandBody(
						TextElement("curl \""),
						VarRefElement("API_URL"),
						TextElement("/users?filter="),
						VarRefElement("FILTER"),
						TextElement("\""))),
				},
			},
		},
		{
			Name:  "@var() in conditional expressions",
			Input: "check: test \"@var(ENV)\" = \"production\" && echo prod || echo dev",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("check", nil, SimpleCommandBody(
						TextElement("test \""),
						VarRefElement("ENV"),
						TextElement("\" = \"production\" && echo prod || echo dev"))),
				},
			},
		},
		{
			Name:  "@var() in loops",
			Input: "process: for file in @var(SRC)/*.txt; do process $file; done",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("process", nil, SimpleCommandBody(
						TextElement("for file in "),
						VarRefElement("SRC"),
						TextElement("/*.txt; do process $file; done"))),
				},
			},
		},
		{
			Name: "debug backup - DATE assignment with semicolon",
			Input: `backup-debug2: {
        @sh(DATE=$(date +%Y%m%d-%H%M%S); echo "test")
      }`,
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("backup-debug2", nil, BlockCommandBody(
						Statement(DecoratorElement("sh", StringExpr("DATE=$(date +%Y%m%d-%H%M%S); echo \"test\""))))),
				},
			},
		},
		{
			Name:  "string with escaped quotes and @var",
			Input: "msg: echo \"He said \\\"Hello @var(NAME)\\\" to everyone\"",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("msg", nil, SimpleCommandBody(
						TextElement("echo \"He said \\\"Hello "),
						VarRefElement("NAME"),
						TextElement("\\\" to everyone\""))),
				},
			},
		},
	}

	for _, tc := range testCases {
		runTestCase(t, tc)
	}
}

func TestNestedDecorators(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "@sh() with @var()",
			Input: "build: @sh(cd @var(SRC))",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("build", []ExpectedDecorator{
						{Name: "sh", Args: []ExpectedExpression{
							StringExpr("cd ", ExpectedVariableRef{Name: "SRC"}),
						}},
					}, BlockCommandBody()),
				},
			},
		},
		{
			Name:  "@sh() with multiple @var()",
			Input: "server: @sh(go run @var(MAIN_FILE) --port=@var(PORT))",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("server", []ExpectedDecorator{
						{Name: "sh", Args: []ExpectedExpression{
							StringExpr("go run ",
								ExpectedVariableRef{Name: "MAIN_FILE"},
								ExpectedVariableRef{Name: "PORT"}),
						}},
					}, BlockCommandBody()),
				},
			},
		},
		{
			Name:  "complex @sh() with parentheses and @var()",
			Input: "check: @sh((cd @var(SRC) && make) || echo \"failed\")",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("check", []ExpectedDecorator{
						{Name: "sh", Args: []ExpectedExpression{
							StringExpr("(cd ", ExpectedVariableRef{Name: "SRC"}),
						}},
					}, BlockCommandBody()),
				},
			},
		},
		{
			Name:  "@timeout with @sh nested",
			Input: "deploy: @timeout(30s) { @sh(deploy.sh @var(ENV)) }",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("deploy", []ExpectedDecorator{
						{Name: "timeout", Args: []ExpectedExpression{DurationExpr("30s")}},
					}, BlockCommandBody(
						Statement(DecoratorElement("sh", StringExpr("deploy.sh ", ExpectedVariableRef{Name: "ENV"}))))),
				},
			},
		},
		{
			Name:  "@parallel with @var() in statements",
			Input: "multi: @parallel { echo @var(MSG1); echo @var(MSG2) }",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("multi", []ExpectedDecorator{
						{Name: "parallel", Args: []ExpectedExpression{}},
					}, BlockCommandBody(
						Statement(TextElement("echo "), VarRefElement("MSG1")),
						Statement(TextElement("echo "), VarRefElement("MSG2")))),
				},
			},
		},
		{
			Name:  "@env with @var() values",
			Input: "setup: @env(PATH=@var(CUSTOM_PATH)) { which custom-tool }",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("setup", []ExpectedDecorator{
						{Name: "env", Args: []ExpectedExpression{
							StringExpr("PATH=", ExpectedVariableRef{Name: "CUSTOM_PATH"}),
						}},
					}, BlockCommandBody(
						Statement(TextElement("which custom-tool")))),
				},
			},
		},
		{
			Name:  "multiple decorators with @var",
			Input: "complex: @timeout(30s) @env(NODE_ENV=@var(ENV)) { npm start }",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("complex", []ExpectedDecorator{
						{Name: "timeout", Args: []ExpectedExpression{DurationExpr("30s")}},
						{Name: "env", Args: []ExpectedExpression{
							StringExpr("NODE_ENV=", ExpectedVariableRef{Name: "ENV"}),
						}},
					}, BlockCommandBody(
						Statement(TextElement("npm start")))),
				},
			},
		},
		{
			Name:  "@cwd with @var path and @sh command",
			Input: "build: @cwd(@var(BUILD_DIR)) { @sh(make clean && make all) }",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("build", []ExpectedDecorator{
						{Name: "cwd", Args: []ExpectedExpression{VarRefExpr("BUILD_DIR")}},
					}, BlockCommandBody(
						Statement(DecoratorElement("sh", StringExpr("make clean && make all"))))),
				},
			},
		},
	}

	for _, tc := range testCases {
		runTestCase(t, tc)
	}
}

// Test cases specifically targeting the failing shell command structure
func TestComplexShellCommands(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "simple shell command substitution",
			Input: "test-simple: @sh(echo \"$(date)\")",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("test-simple", []ExpectedDecorator{
						{Name: "sh", Args: []ExpectedExpression{
							StringExpr("echo \"$(date)\""),
						}},
					}, BlockCommandBody()),
				},
			},
		},
		{
			Name:  "shell command with test and command substitution",
			Input: "test-condition: @sh(if [ \"$(echo test)\" = \"test\" ]; then echo ok; fi)",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("test-condition", []ExpectedDecorator{
						{Name: "sh", Args: []ExpectedExpression{
							StringExpr("if [ \"$(echo test)\" = \"test\" ]; then echo ok; fi"),
						}},
					}, BlockCommandBody()),
				},
			},
		},
		{
			Name:  "command with @var and shell substitution",
			Input: "test-mixed: @sh(cd @var(SRC) && echo \"files: $(ls | wc -l)\")",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("test-mixed", []ExpectedDecorator{
						{Name: "sh", Args: []ExpectedExpression{
							StringExpr("cd ", ExpectedVariableRef{Name: "SRC"}),
						}},
					}, BlockCommandBody()),
				},
			},
		},
		{
			Name: "the actual failing command from commands.cli",
			Input: `test-quick: {
    echo "⚡ Running quick checks..."
    @sh(if command -v gofumpt >/dev/null 2>&1; then if [ "$(gofumpt -l . | wc -l)" -gt 0 ]; then echo "❌ Go formatting issues:"; gofumpt -l .; exit 1; fi; else if [ "$(gofmt -l . | wc -l)" -gt 0 ]; then echo "❌ Go formatting issues:"; gofumpt -l .; exit 1; fi; fi)
}`,
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("test-quick", nil, BlockCommandBody(
						Statement(TextElement("echo \"⚡ Running quick checks...\"")),
						Statement(DecoratorElement("sh", StringExpr("if command -v gofumpt >/dev/null 2>&1; then if [ \"$(gofumpt -l . | wc -l)\" -gt 0 ]; then echo \"❌ Go formatting issues:\"; gofumpt -l .; exit 1; fi; else if [ \"$(gofmt -l . | wc -l)\" -gt 0 ]; then echo \"❌ Go formatting issues:\"; gofumpt -l .; exit 1; fi; fi"))))),
				},
			},
		},
		{
			Name:  "simplified version of failing command",
			Input: "test-format: @sh(if [ \"$(gofumpt -l . | wc -l)\" -gt 0 ]; then echo \"issues\"; fi)",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("test-format", []ExpectedDecorator{
						{Name: "sh", Args: []ExpectedExpression{
							StringExpr("if [ \"$(gofumpt -l . | wc -l)\" -gt 0 ]; then echo \"issues\"; fi"),
						}},
					}, BlockCommandBody()),
				},
			},
		},
		{
			Name:  "even simpler - just the command substitution in quotes",
			Input: "test-basic: @sh(\"$(gofumpt -l . | wc -l)\")",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("test-basic", []ExpectedDecorator{
						{Name: "sh", Args: []ExpectedExpression{
							StringExpr("\"$(gofumpt -l . | wc -l)\""),
						}},
					}, BlockCommandBody()),
				},
			},
		},
		{
			Name:  "debug - minimal parentheses in quotes",
			Input: "test-debug: @sh(\"()\")",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("test-debug", []ExpectedDecorator{
						{Name: "sh", Args: []ExpectedExpression{
							StringExpr("\"()\""),
						}},
					}, BlockCommandBody()),
				},
			},
		},
		{
			Name:  "debug - single command substitution",
			Input: "test-debug2: @sh($(echo test))",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("test-debug2", []ExpectedDecorator{
						{Name: "sh", Args: []ExpectedExpression{
							StringExpr("$(echo test)"),
						}},
					}, BlockCommandBody()),
				},
			},
		},
		// Test case for backup command with complex shell substitution and @var()
		{
			Name: "backup command with shell substitution and @var",
			Input: `backup: {
        echo "Creating backup..."
        # Shell command substitution uses regular $() syntax in @sh()
        @sh(DATE=$(date +%Y%m%d-%H%M%S); echo "Backup timestamp: $DATE")
        @sh((which kubectl && kubectl exec deployment/database -n @var(KUBE_NAMESPACE) -- pg_dump myapp > backup-$(date +%Y%m%d-%H%M%S).sql) || echo "No database")
      }`,
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("backup", nil, BlockCommandBody(
						Statement(TextElement("echo \"Creating backup...\"")),
						Statement(DecoratorElement("sh", StringExpr("DATE=$(date +%Y%m%d-%H%M%S); echo \"Backup timestamp: $DATE\""))),
						Statement(DecoratorElement("sh", StringExpr("(which kubectl && kubectl exec deployment/database -n ", ExpectedVariableRef{Name: "KUBE_NAMESPACE"}))))),
				},
			},
		},
		// Add this test case at the end of the existing testCases slice:
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
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("test-quick", nil, BlockCommandBody(
						Statement(TextElement("echo \"⚡ Running quick checks...\"")),
						Statement(TextElement("echo \"🔍 Checking Go formatting...\"")),
						Statement(DecoratorElement("sh", StringExpr("if command -v gofumpt >/dev/null 2>&1; then if [ \"$(gofumpt -l . | wc -l)\" -gt 0 ]; then echo \"❌ Go formatting issues:\"; gofumpt -l .; exit 1; fi; else if [ \"$(gofmt -l . | wc -l)\" -gt 0 ]; then echo \"❌ Go formatting issues:\"; gofmt -l .; exit 1; fi; fi"))),
						Statement(TextElement("echo \"🔍 Checking Nix formatting...\"")),
						Statement(DecoratorElement("sh", StringExpr("if command -v nixpkgs-fmt >/dev/null 2>&1; then nixpkgs-fmt --check . || (echo \"❌ Run 'dev format' to fix\"; exit 1); else echo \"⚠️  nixpkgs-fmt not available, skipping Nix format check\"; fi"))),
						Statement(TextElement("dev lint")),
						Statement(TextElement("echo \"✅ Quick checks passed!\"")))),
				},
			},
		},
		{
			Name:  "shell with arithmetic expansion",
			Input: "math: @sh(echo $((2 + 3 * 4)))",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("math", []ExpectedDecorator{
						{Name: "sh", Args: []ExpectedExpression{
							StringExpr("echo $((2 + 3 * 4))"),
						}},
					}, BlockCommandBody()),
				},
			},
		},
		{
			Name:  "shell with process substitution",
			Input: "diff: @sh(diff <(sort file1) <(sort file2))",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("diff", []ExpectedDecorator{
						{Name: "sh", Args: []ExpectedExpression{
							StringExpr("diff <(sort file1) <(sort file2)"),
						}},
					}, BlockCommandBody()),
				},
			},
		},
		{
			Name:  "shell with parameter expansion",
			Input: "param: @sh(echo ${VAR:-default})",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("param", []ExpectedDecorator{
						{Name: "sh", Args: []ExpectedExpression{
							StringExpr("echo ${VAR:-default}"),
						}},
					}, BlockCommandBody()),
				},
			},
		},
		{
			Name:  "shell with complex conditionals",
			Input: "conditional: @sh([[ -f file.txt && -r file.txt ]] && echo readable || echo not-readable)",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("conditional", []ExpectedDecorator{
						{Name: "sh", Args: []ExpectedExpression{
							StringExpr("[[ -f file.txt && -r file.txt ]] && echo readable || echo not-readable"),
						}},
					}, BlockCommandBody()),
				},
			},
		},
		{
			Name:  "shell with here document",
			Input: "heredoc: @sh(cat <<EOF\nLine 1\nLine 2\nEOF)",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("heredoc", []ExpectedDecorator{
						{Name: "sh", Args: []ExpectedExpression{
							StringExpr("cat <<EOF\nLine 1\nLine 2\nEOF"),
						}},
					}, BlockCommandBody()),
				},
			},
		},
		{
			Name:  "shell with case statement",
			Input: "case-test: @sh(case $1 in start) echo starting;; stop) echo stopping;; esac)",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("case-test", []ExpectedDecorator{
						{Name: "sh", Args: []ExpectedExpression{
							StringExpr("case $1 in start) echo starting;; stop) echo stopping;; esac"),
						}},
					}, BlockCommandBody()),
				},
			},
		},
	}

	for _, tc := range testCases {
		runTestCase(t, tc)
	}
}

// Test cases for edge cases in quote and parentheses handling
func TestQuoteAndParenthesesEdgeCases(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "escaped quotes in shell command",
			Input: "test-escaped: @sh(echo \"He said \\\"hello\\\" to me\")",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("test-escaped", []ExpectedDecorator{
						{Name: "sh", Args: []ExpectedExpression{
							StringExpr("echo \"He said \\\"hello\\\" to me\""),
						}},
					}, BlockCommandBody()),
				},
			},
		},
		{
			Name:  "mixed quotes with parentheses",
			Input: "test-mixed-quotes: @sh(echo 'test \"$(date)\" done' && echo \"test '$(whoami)' done\")",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("test-mixed-quotes", []ExpectedDecorator{
						{Name: "sh", Args: []ExpectedExpression{
							StringExpr("echo 'test \"$(date)\" done' && echo \"test '$(whoami)' done\""),
						}},
					}, BlockCommandBody()),
				},
			},
		},
		{
			Name:  "backticks with parentheses",
			Input: "test-backticks: @sh(echo `date` and $(whoami))",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("test-backticks", []ExpectedDecorator{
						{Name: "sh", Args: []ExpectedExpression{
							StringExpr("echo `date` and $(whoami)"),
						}},
					}, BlockCommandBody()),
				},
			},
		},
		{
			Name:  "nested quotes with special characters",
			Input: "special-chars: @sh(echo \"Path: '$HOME' and size: $(du -sh .)\")",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("special-chars", []ExpectedDecorator{
						{Name: "sh", Args: []ExpectedExpression{
							StringExpr("echo \"Path: '$HOME' and size: $(du -sh .)\""),
						}},
					}, BlockCommandBody()),
				},
			},
		},
		{
			Name:  "quotes within @var context",
			Input: "var-quotes: echo \"Config file: '@var(CONFIG_FILE)'\"",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("var-quotes", nil, SimpleCommandBody(
						TextElement("echo \"Config file: '"),
						VarRefElement("CONFIG_FILE"),
						TextElement("'\""))),
				},
			},
		},
		{
			Name:  "parentheses in shell without command substitution",
			Input: "parens: @sh((cd /tmp && ls) > /dev/null)",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("parens", []ExpectedDecorator{
						{Name: "sh", Args: []ExpectedExpression{
							StringExpr("(cd /tmp && ls) > /dev/null"),
						}},
					}, BlockCommandBody()),
				},
			},
		},
		{
			Name:  "nested parentheses with command substitution",
			Input: "nested: @sh(echo $(echo $(date +%Y)))",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("nested", []ExpectedDecorator{
						{Name: "sh", Args: []ExpectedExpression{
							StringExpr("echo $(echo $(date +%Y))"),
						}},
					}, BlockCommandBody()),
				},
			},
		},
		{
			Name:  "quotes with regex patterns",
			Input: "regex: grep \"pattern[0-9]+\" file.txt",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("regex", nil, SimpleCommandBody(
						TextElement("grep \"pattern[0-9]+\" file.txt"))),
				},
			},
		},
		{
			Name:  "single quotes preserving literals",
			Input: "literal: echo 'Variables like $HOME and $(date) are not expanded'",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("literal", nil, SimpleCommandBody(
						TextElement("echo 'Variables like $HOME and $(date) are not expanded'"))),
				},
			},
		},
		{
			Name:  "mixed quote styles in one command",
			Input: "mixed: echo 'Single quotes' and \"double quotes\" and `backticks`",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("mixed", nil, SimpleCommandBody(
						TextElement("echo 'Single quotes' and \"double quotes\" and `backticks`"))),
				},
			},
		},
	}

	for _, tc := range testCases {
		runTestCase(t, tc)
	}
}

// Test cases for @var() within shell commands
func TestVarInShellCommands(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "simple @var in shell command",
			Input: "test-var: @sh(cd @var(DIR))",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("test-var", []ExpectedDecorator{
						{Name: "sh", Args: []ExpectedExpression{
							StringExpr("cd ", ExpectedVariableRef{Name: "DIR"}),
						}},
					}, BlockCommandBody()),
				},
			},
		},
		{
			Name:  "@var with shell command substitution",
			Input: "test-var-cmd: @sh(cd @var(DIR) && echo \"$(pwd)\")",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("test-var-cmd", []ExpectedDecorator{
						{Name: "sh", Args: []ExpectedExpression{
							StringExpr("cd ", ExpectedVariableRef{Name: "DIR"}),
						}},
					}, BlockCommandBody()),
				},
			},
		},
		{
			Name:  "multiple @var with complex shell",
			Input: "test-multi-var: @sh(if [ -d @var(SRC) ] && [ \"$(ls @var(SRC) | wc -l)\" -gt 0 ]; then echo \"Source dir has files\"; fi)",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("test-multi-var", []ExpectedDecorator{
						{Name: "sh", Args: []ExpectedExpression{
							StringExpr("if [ -d ",
								ExpectedVariableRef{Name: "SRC"},
								ExpectedVariableRef{Name: "SRC"}),
						}},
					}, BlockCommandBody()),
				},
			},
		},
		{
			Name:  "@var in shell function definition",
			Input: "func-def: @sh(deploy() { rsync -av @var(SRC)/ deploy@server.com:@var(DEST)/; }; deploy)",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("func-def", []ExpectedDecorator{
						{Name: "sh", Args: []ExpectedExpression{
							StringExpr("deploy() { rsync -av ",
								ExpectedVariableRef{Name: "SRC"},
								ExpectedVariableRef{Name: "DEST"}),
						}},
					}, BlockCommandBody()),
				},
			},
		},
		{
			Name:  "@var in shell array",
			Input: "array-var: @sh(FILES=(@var(FILE1) @var(FILE2) @var(FILE3)); echo ${FILES[@]})",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("array-var", []ExpectedDecorator{
						{Name: "sh", Args: []ExpectedExpression{
							StringExpr("FILES=(",
								ExpectedVariableRef{Name: "FILE1"},
								ExpectedVariableRef{Name: "FILE2"},
								ExpectedVariableRef{Name: "FILE3"}),
						}},
					}, BlockCommandBody()),
				},
			},
		},
		{
			Name:  "@var in shell case statement",
			Input: "case-var: @sh(case @var(ENV) in prod) echo production;; dev) echo development;; esac)",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("case-var", []ExpectedDecorator{
						{Name: "sh", Args: []ExpectedExpression{
							StringExpr("case ", ExpectedVariableRef{Name: "ENV"}),
						}},
					}, BlockCommandBody()),
				},
			},
		},
		{
			Name:  "@var in shell parameter expansion",
			Input: "param-expansion: @sh(echo ${@var(VAR):-@var(DEFAULT)})",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("param-expansion", []ExpectedDecorator{
						{Name: "sh", Args: []ExpectedExpression{
							StringExpr("echo ${",
								ExpectedVariableRef{Name: "VAR"},
								ExpectedVariableRef{Name: "DEFAULT"}),
						}},
					}, BlockCommandBody()),
				},
			},
		},
	}

	for _, tc := range testCases {
		runTestCase(t, tc)
	}
}

func TestBlockCommands(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "empty block",
			Input: "setup: { }",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("setup", nil, BlockCommandBody()),
				},
			},
		},
		{
			Name:  "single statement block",
			Input: "setup: { npm install }",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("setup", nil, BlockCommandBody(
						Statement(TextElement("npm install")))),
				},
			},
		},
		{
			Name:  "multiple statements",
			Input: "setup: { npm install; go mod tidy; echo done }",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("setup", nil, BlockCommandBody(
						Statement(TextElement("npm install")),
						Statement(TextElement("go mod tidy")),
						Statement(TextElement("echo done")))),
				},
			},
		},
		{
			Name:  "block with @var() references",
			Input: "build: { cd @var(SRC); make @var(TARGET) }",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("build", nil, BlockCommandBody(
						Statement(TextElement("cd "), VarRefElement("SRC")),
						Statement(TextElement("make "), VarRefElement("TARGET")))),
				},
			},
		},
		{
			Name:  "block with decorators",
			Input: "services: { @parallel { server; client } }",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("services", nil, BlockCommandBody(
						Statement(DecoratorElement("parallel")))),
				},
			},
		},
		{
			Name:  "nested blocks with decorators",
			Input: "complex: { @timeout(30s) { @sh(long-running-task); echo done } }",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("complex", nil, BlockCommandBody(
						Statement(DecoratorElement("timeout", DurationExpr("30s"))))),
				},
			},
		},
		{
			Name:  "block with mixed statements and decorators",
			Input: "deploy: { echo starting; @sh(deploy.sh); @parallel { service1; service2 }; echo finished }",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("deploy", nil, BlockCommandBody(
						Statement(TextElement("echo starting")),
						Statement(DecoratorElement("sh", StringExpr("deploy.sh"))),
						Statement(DecoratorElement("parallel")),
						Statement(TextElement("echo finished")))),
				},
			},
		},
		{
			Name:  "block with complex shell statements",
			Input: "test: { echo start; for i in {1..3}; do echo $i; done; echo end }",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("test", nil, BlockCommandBody(
						Statement(TextElement("echo start")),
						Statement(TextElement("for i in {1..3}; do echo $i; done")),
						Statement(TextElement("echo end")))),
				},
			},
		},
		{
			Name:  "block with conditional statements",
			Input: "conditional: { test -f file.txt && echo exists || echo missing; echo checked }",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("conditional", nil, BlockCommandBody(
						Statement(TextElement("test -f file.txt && echo exists || echo missing")),
						Statement(TextElement("echo checked")))),
				},
			},
		},
		{
			Name:  "block with background processes",
			Input: "background: { server &; client &; wait }",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("background", nil, BlockCommandBody(
						Statement(TextElement("server &")),
						Statement(TextElement("client &")),
						Statement(TextElement("wait")))),
				},
			},
		},
		{
			Name:  "deeply nested block with mixed content",
			Input: "deploy: { echo start; @parallel { @timeout(10s) { service1 }; service2 }; echo done }",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("deploy", nil, BlockCommandBody(
						Statement(TextElement("echo start")),
						Statement(DecoratorElement("parallel")),
						Statement(TextElement("echo done")))),
				},
			},
		},
		{
			Name:  "block with variable references and decorators",
			Input: "build: { cd @var(SRC); @sh(make clean); make @var(TARGET); echo \"Built @var(TARGET)\" }",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("build", nil, BlockCommandBody(
						Statement(TextElement("cd "), VarRefElement("SRC")),
						Statement(DecoratorElement("sh", StringExpr("make clean"))),
						Statement(TextElement("make "), VarRefElement("TARGET")),
						Statement(TextElement("echo \"Built "), VarRefElement("TARGET"), TextElement("\"")))),
				},
			},
		},
		{
			Name:  "block with conditional shell commands",
			Input: "check: { test -f @var(CONFIG) && echo \"Config exists\" || echo \"Missing config\" }",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("check", nil, BlockCommandBody(
						Statement(TextElement("test -f "), VarRefElement("CONFIG"), TextElement(" && echo \"Config exists\" || echo \"Missing config\"")))),
				},
			},
		},
		{
			Name:  "empty block with decorators",
			Input: "parallel-empty: @parallel { }",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("parallel-empty", []ExpectedDecorator{
						{Name: "parallel", Args: []ExpectedExpression{}},
					}, BlockCommandBody()),
				},
			},
		},
	}

	for _, tc := range testCases {
		runTestCase(t, tc)
	}
}

func TestWatchStopCommands(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "simple watch command",
			Input: "watch server: npm start",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					WatchCommand("server", nil, SimpleCommandBody(
						TextElement("npm start"))),
				},
			},
		},
		{
			Name:  "simple stop command",
			Input: "stop server: pkill node",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					StopCommand("server", nil, SimpleCommandBody(
						TextElement("pkill node"))),
				},
			},
		},
		{
			Name:  "watch command with @var()",
			Input: "watch server: go run @var(MAIN_FILE) --port=@var(PORT)",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					WatchCommand("server", nil, SimpleCommandBody(
						TextElement("go run "),
						VarRefElement("MAIN_FILE"),
						TextElement(" --port="),
						VarRefElement("PORT"))),
				},
			},
		},
		{
			Name:  "watch block command",
			Input: "watch dev: { npm start; go run main.go }",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					WatchCommand("dev", nil, BlockCommandBody(
						Statement(TextElement("npm start")),
						Statement(TextElement("go run main.go")))),
				},
			},
		},
		{
			Name:  "watch with decorators",
			Input: "watch api: @env(NODE_ENV=development) { npm run dev }",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					WatchCommand("api", []ExpectedDecorator{
						{Name: "env", Args: []ExpectedExpression{IdentifierExpr("NODE_ENV=development")}},
					}, BlockCommandBody(
						Statement(TextElement("npm run dev")))),
				},
			},
		},
		{
			Name:  "stop with graceful shutdown",
			Input: "stop api: @sh(curl -X POST localhost:3000/shutdown || pkill node)",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					StopCommand("api", []ExpectedDecorator{
						{Name: "sh", Args: []ExpectedExpression{
							StringExpr("curl -X POST localhost:3000/shutdown || pkill node"),
						}},
					}, BlockCommandBody()),
				},
			},
		},
		{
			Name:  "watch with multiple services",
			Input: "watch services: @parallel { npm run api; npm run worker; npm run scheduler }",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					WatchCommand("services", []ExpectedDecorator{
						{Name: "parallel", Args: []ExpectedExpression{}},
					}, BlockCommandBody(
						Statement(TextElement("npm run api")),
						Statement(TextElement("npm run worker")),
						Statement(TextElement("npm run scheduler")))),
				},
			},
		},
		{
			Name:  "stop with cleanup block",
			Input: "stop services: { pkill -f node; docker stop $(docker ps -q); echo cleaned }",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					StopCommand("services", nil, BlockCommandBody(
						Statement(TextElement("pkill -f node")),
						Statement(TextElement("docker stop $(docker ps -q)")),
						Statement(TextElement("echo cleaned")))),
				},
			},
		},
		{
			Name:  "watch with timeout",
			Input: "watch build: @timeout(60s) { npm run watch:build }",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					WatchCommand("build", []ExpectedDecorator{
						{Name: "timeout", Args: []ExpectedExpression{DurationExpr("60s")}},
					}, BlockCommandBody(
						Statement(TextElement("npm run watch:build")))),
				},
			},
		},
		{
			Name:  "stop with confirmation",
			Input: "stop production: @confirm(\"Really stop production?\") { systemctl stop myapp }",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					StopCommand("production", []ExpectedDecorator{
						{Name: "confirm", Args: []ExpectedExpression{
							StringExpr("Really stop production?"),
						}},
					}, BlockCommandBody(
						Statement(TextElement("systemctl stop myapp")))),
				},
			},
		},
	}

	for _, tc := range testCases {
		runTestCase(t, tc)
	}
}

func TestContinuationLines(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "simple continuation",
			Input: "build: echo hello \\\nworld",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("build", nil, SimpleCommandBody(
						TextElement("echo hello world"))),
				},
			},
		},
		{
			Name:  "continuation with @var()",
			Input: "build: cd @var(DIR) \\\n&& make",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("build", nil, SimpleCommandBody(
						TextElement("cd "),
						VarRefElement("DIR"),
						TextElement(" && make"))),
				},
			},
		},
		{
			Name:  "multiple line continuations",
			Input: "complex: echo start \\\n&& echo middle \\\n&& echo end",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("complex", nil, SimpleCommandBody(
						TextElement("echo start && echo middle && echo end"))),
				},
			},
		},
		{
			Name:  "continuation in block",
			Input: "block: { echo hello \\\nworld; echo done }",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("block", nil, BlockCommandBody(
						Statement(TextElement("echo hello world")),
						Statement(TextElement("echo done")))),
				},
			},
		},
		{
			Name:  "continuation with decorators",
			Input: "deploy: @sh(docker build \\\n-t myapp \\\n.)",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("deploy", []ExpectedDecorator{
						{Name: "sh", Args: []ExpectedExpression{
							StringExpr("docker build -t myapp ."),
						}},
					}, BlockCommandBody()),
				},
			},
		},
		{
			Name:  "continuation with mixed content",
			Input: "mixed: docker run \\\n@var(IMAGE) \\\n--port=@var(PORT)",
			Expected: ExpectedProgram{
				Commands: []ExpectedCommand{
					SimpleCommand("mixed", nil, SimpleCommandBody(
						TextElement("docker run "),
						VarRefElement("IMAGE"),
						TextElement(" --port="),
						VarRefElement("PORT"))),
				},
			},
		},
	}

	for _, tc := range testCases {
		runTestCase(t, tc)
	}
}

func TestVariableDefinitions(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "simple variable",
			Input: "var SRC = ./src",
			Expected: ExpectedProgram{
				Variables: []ExpectedVariable{
					Variable("SRC", StringExpr("./src")),
				},
			},
		},
		{
			Name:  "variable with complex value",
			Input: "var CMD = go test -v ./...",
			Expected: ExpectedProgram{
				Variables: []ExpectedVariable{
					Variable("CMD", StringExpr("go test -v ./...")),
				},
			},
		},
		{
			Name:  "multiple variables",
			Input: "var SRC = ./src\nvar BIN = ./bin",
			Expected: ExpectedProgram{
				Variables: []ExpectedVariable{
					Variable("SRC", StringExpr("./src")),
					Variable("BIN", StringExpr("./bin")),
				},
			},
		},
		{
			Name:  "grouped variables",
			Input: "var (\n  SRC = ./src\n  BIN = ./bin\n)",
			Expected: ExpectedProgram{
				Variables: []ExpectedVariable{
					Variable("SRC", StringExpr("./src")),
					Variable("BIN", StringExpr("./bin")),
				},
			},
		},
		{
			Name:  "variable with number value",
			Input: "var PORT = 8080",
			Expected: ExpectedProgram{
				Variables: []ExpectedVariable{
					Variable("PORT", NumberExpr("8080")),
				},
			},
		},
		{
			Name:  "variable with duration value",
			Input: "var TIMEOUT = 30s",
			Expected: ExpectedProgram{
				Variables: []ExpectedVariable{
					Variable("TIMEOUT", DurationExpr("30s")),
				},
			},
		},
		{
			Name:  "variable with quoted string",
			Input: "var MESSAGE = \"Hello, World!\"",
			Expected: ExpectedProgram{
				Variables: []ExpectedVariable{
					Variable("MESSAGE", StringExpr("Hello, World!")),
				},
			},
		},
		{
			Name:  "variable with special characters",
			Input: "var API_URL = https://api.example.com/v1",
			Expected: ExpectedProgram{
				Variables: []ExpectedVariable{
					Variable("API_URL", StringExpr("https://api.example.com/v1")),
				},
			},
		},
		{
			Name:  "mixed variable types in group",
			Input: "var (\n  SRC = ./src\n  PORT = 3000\n  TIMEOUT = 5m\n  DEBUG = true\n)",
			Expected: ExpectedProgram{
				Variables: []ExpectedVariable{
					Variable("SRC", StringExpr("./src")),
					Variable("PORT", NumberExpr("3000")),
					Variable("TIMEOUT", DurationExpr("5m")),
					Variable("DEBUG", IdentifierExpr("true")),
				},
			},
		},
		{
			Name:  "variable with environment-style name",
			Input: "var NODE_ENV = production",
			Expected: ExpectedProgram{
				Variables: []ExpectedVariable{
					Variable("NODE_ENV", IdentifierExpr("production")),
				},
			},
		},
		{
			Name:  "variable with special characters in value",
			Input: "var API_URL = https://api.example.com/v1?key=abc123",
			Expected: ExpectedProgram{
				Variables: []ExpectedVariable{
					Variable("API_URL", StringExpr("https://api.example.com/v1?key=abc123")),
				},
			},
		},
		{
			Name:  "variable with boolean-like value",
			Input: "var DEBUG = true",
			Expected: ExpectedProgram{
				Variables: []ExpectedVariable{
					Variable("DEBUG", IdentifierExpr("true")),
				},
			},
		},
		{
			Name:  "variable with path containing spaces",
			Input: "var PROJECT_PATH = \"/path/with spaces/project\"",
			Expected: ExpectedProgram{
				Variables: []ExpectedVariable{
					Variable("PROJECT_PATH", StringExpr("/path/with spaces/project")),
				},
			},
		},
		{
			Name:  "variable with empty string value",
			Input: "var EMPTY = \"\"",
			Expected: ExpectedProgram{
				Variables: []ExpectedVariable{
					Variable("EMPTY", StringExpr("")),
				},
			},
		},
		{
			Name:  "variable with numeric string",
			Input: "var VERSION = \"1.2.3\"",
			Expected: ExpectedProgram{
				Variables: []ExpectedVariable{
					Variable("VERSION", StringExpr("1.2.3")),
				},
			},
		},
		{
			Name:  "variable with complex file path",
			Input: "var CONFIG_FILE = /etc/myapp/config.json",
			Expected: ExpectedProgram{
				Variables: []ExpectedVariable{
					Variable("CONFIG_FILE", StringExpr("/etc/myapp/config.json")),
				},
			},
		},
		{
			Name:  "variable with URL containing port",
			Input: "var DATABASE_URL = postgresql://user:pass@localhost:5432/dbname",
			Expected: ExpectedProgram{
				Variables: []ExpectedVariable{
					Variable("DATABASE_URL", StringExpr("postgresql://user:pass@localhost:5432/dbname")),
				},
			},
		},
		{
			Name:  "variable with floating point duration",
			Input: "var TIMEOUT = 2.5s",
			Expected: ExpectedProgram{
				Variables: []ExpectedVariable{
					Variable("TIMEOUT", DurationExpr("2.5s")),
				},
			},
		},
		{
			Name:  "multiple variables with mixed types",
			Input: "var PORT = 3000\nvar HOST = localhost\nvar TIMEOUT = 30s\nvar DEBUG = true",
			Expected: ExpectedProgram{
				Variables: []ExpectedVariable{
					Variable("PORT", NumberExpr("3000")),
					Variable("HOST", IdentifierExpr("localhost")),
					Variable("TIMEOUT", DurationExpr("30s")),
					Variable("DEBUG", IdentifierExpr("true")),
				},
			},
		},
		{
			Name:  "variable with quoted identifier value",
			Input: "var MODE = \"production\"",
			Expected: ExpectedProgram{
				Variables: []ExpectedVariable{
					Variable("MODE", StringExpr("production")),
				},
			},
		},
		{
			Name:  "variable with underscores",
			Input: "var API_BASE_URL = https://api.example.com",
			Expected: ExpectedProgram{
				Variables: []ExpectedVariable{
					Variable("API_BASE_URL", StringExpr("https://api.example.com")),
				},
			},
		},
	}

	for _, tc := range testCases {
		runTestCase(t, tc)
	}
}


func TestWatchStopSameName(t *testing.T) {
	input := `
watch server: npm start
stop server: pkill node
`

	tc := TestCase{
		Name:  "watch and stop with same name should be allowed",
		Input: input,
		Expected: ExpectedProgram{
			Commands: []ExpectedCommand{
				WatchCommand("server", nil, SimpleCommandBody(
					TextElement("npm start"))),
				StopCommand("server", nil, SimpleCommandBody(
					TextElement("pkill node"))),
			},
		},
	}

	runTestCase(t, tc)
}

