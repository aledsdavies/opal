package parser

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/aledsdavies/devcmd/pkgs/ast"
	"github.com/google/go-cmp/cmp"
)

// DSL for building expected test results using natural language

// Program creates an expected program
func Program(items ...interface{}) ExpectedProgram {
	var variables []ExpectedVariable
	var commands []ExpectedCommand

	for _, item := range items {
		switch v := item.(type) {
		case ExpectedVariable:
			variables = append(variables, v)
		case ExpectedCommand:
			commands = append(commands, v)
		}
	}

	return ExpectedProgram{
		Variables: variables,
		Commands:  commands,
	}
}

// Var creates a variable declaration: var NAME = VALUE
func Var(name string, value interface{}) ExpectedVariable {
	return ExpectedVariable{
		Name:  name,
		Value: toExpression(value),
	}
}

// DurationExpr creates a duration expression for test expectations
func DurationExpr(value string) ExpectedExpression {
	return ExpectedExpression{
		Type:  "duration",
		Value: value,
	}
}

// Cmd creates a simple command: NAME: BODY
func Cmd(name string, body interface{}) ExpectedCommand {
	return ExpectedCommand{
		Name: name,
		Type: ast.Command,
		Body: toCommandBody(body),
	}
}

// CmdWith creates a command with decorators: @decorator NAME: BODY
// This is syntax sugar for: NAME: { @decorator ... }
func CmdWith(decorators interface{}, name string, body interface{}) ExpectedCommand {
	return ExpectedCommand{
		Name: name,
		Type: ast.Command,
		Body: ExpectedCommandBody{
			IsBlock: true,
			Content: toCommandContentWithDecorators(decorators, body),
		},
	}
}

// Watch creates a watch command: watch NAME: BODY
func Watch(name string, body interface{}) ExpectedCommand {
	return ExpectedCommand{
		Name: name,
		Type: ast.WatchCommand,
		Body: toCommandBody(body),
	}
}

// WatchWith creates a watch command with decorators
func WatchWith(decorators interface{}, name string, body interface{}) ExpectedCommand {
	return ExpectedCommand{
		Name: name,
		Type: ast.WatchCommand,
		Body: ExpectedCommandBody{
			IsBlock: true,
			Content: toCommandContentWithDecorators(decorators, body),
		},
	}
}

// Stop creates a stop command: stop NAME: BODY
func Stop(name string, body interface{}) ExpectedCommand {
	return ExpectedCommand{
		Name: name,
		Type: ast.StopCommand,
		Body: toCommandBody(body),
	}
}

// StopWith creates a stop command with decorators
func StopWith(decorators interface{}, name string, body interface{}) ExpectedCommand {
	return ExpectedCommand{
		Name: name,
		Type: ast.StopCommand,
		Body: ExpectedCommandBody{
			IsBlock: true,
			Content: toCommandContentWithDecorators(decorators, body),
		},
	}
}

// Block creates a block command body: { content }
func Block(content ...interface{}) ExpectedCommandBody {
	return ExpectedCommandBody{
		IsBlock: true,
		Content: toCommandContent(content...),
	}
}

// Simple creates a simple command body (single line)
func Simple(parts ...interface{}) ExpectedCommandBody {
	return ExpectedCommandBody{
		IsBlock: false,
		Content: ExpectedShellContent{
			Parts: toShellParts(parts...),
		},
	}
}

// Text creates a text part
func Text(text string) ExpectedShellPart {
	return ExpectedShellPart{
		Type: "text",
		Text: text,
	}
}

// At creates a function decorator within shell content: @var(NAME)
func At(name string, args ...interface{}) ExpectedShellPart {
	var decoratorArgs []ExpectedExpression
	for _, arg := range args {
		decoratorArgs = append(decoratorArgs, toExpression(arg))
	}

	return ExpectedShellPart{
		Type: "function_decorator",
		FunctionDecorator: &ExpectedFunctionDecorator{
			Name: name,
			Args: decoratorArgs,
		},
	}
}

// Decorator creates a block decorator: @timeout(30s)
func Decorator(name string, args ...interface{}) ExpectedDecorator {
	var decoratorArgs []ExpectedExpression
	for _, arg := range args {
		decoratorArgs = append(decoratorArgs, toExpression(arg))
	}

	return ExpectedDecorator{
		Name: name,
		Args: decoratorArgs,
	}
}

// Helper conversion functions
func toExpression(v interface{}) ExpectedExpression {
	switch val := v.(type) {
	case string:
		if strings.HasPrefix(val, "@") {
			return ExpectedExpression{Type: "decorator", Value: val}
		}
		// Check if it's a duration pattern and return as duration
		if isDurationString(val) {
			return ExpectedExpression{Type: "duration", Value: val}
		}
		// Check if it's a numeric string
		if isNumericString(val) {
			return ExpectedExpression{Type: "number", Value: val}
		}
		return ExpectedExpression{Type: "string", Value: val}
	case int:
		return ExpectedExpression{Type: "number", Value: strconv.Itoa(val)}
	case ExpectedExpression:
		return val
	case ExpectedFunctionDecorator:
		return ExpectedExpression{Type: "function_decorator", Value: "@" + val.Name}
	default:
		// Try to convert to string and handle as identifier
		str := fmt.Sprintf("%v", val)
		return ExpectedExpression{Type: "identifier", Value: strings.Trim(str, "\"")}
	}
}

func toCommandBody(v interface{}) ExpectedCommandBody {
	switch val := v.(type) {
	case ExpectedCommandBody:
		return val
	case string:
		return Simple(Text(val))
	default:
		return ExpectedCommandBody{
			IsBlock: false,
			Content: ExpectedShellContent{
				Parts: []ExpectedShellPart{Text("")},
			},
		}
	}
}

func toCommandContent(items ...interface{}) ExpectedCommandContent {
	if len(items) == 0 {
		return ExpectedShellContent{Parts: []ExpectedShellPart{}}
	}

	// Check if first item is a decorator
	if dec, ok := items[0].(ExpectedDecorator); ok {
		// This is decorated content
		decorators := []ExpectedDecorator{dec}

		// Look for more decorators
		contentStart := 1
		for i := 1; i < len(items); i++ {
			if nextDec, ok := items[i].(ExpectedDecorator); ok {
				decorators = append(decorators, nextDec)
				contentStart = i + 1
			} else {
				break
			}
		}

		// Rest is the content
		var content ExpectedCommandContent
		if contentStart < len(items) {
			content = toCommandContent(items[contentStart:]...)
		} else {
			content = ExpectedShellContent{Parts: []ExpectedShellPart{}}
		}

		return ExpectedDecoratedContent{
			Decorators: decorators,
			Content:    content,
		}
	}

	// This is shell content
	return ExpectedShellContent{
		Parts: toShellParts(items...),
	}
}

func toCommandContentWithDecorators(decorators interface{}, body interface{}) ExpectedCommandContent {
	decoratorList := toDecorators(decorators)

	// Convert body to content
	var content ExpectedCommandContent
	switch v := body.(type) {
	case ExpectedCommandBody:
		content = v.Content
	case string:
		content = ExpectedShellContent{
			Parts: []ExpectedShellPart{Text(v)},
		}
	default:
		content = ExpectedShellContent{
			Parts: []ExpectedShellPart{},
		}
	}

	return ExpectedDecoratedContent{
		Decorators: decoratorList,
		Content:    content,
	}
}

func toShellParts(items ...interface{}) []ExpectedShellPart {
	var parts []ExpectedShellPart
	for _, item := range items {
		switch v := item.(type) {
		case ExpectedShellPart:
			parts = append(parts, v)
		case string:
			parts = append(parts, Text(v))
		case ExpectedFunctionDecorator:
			parts = append(parts, ExpectedShellPart{
				Type: "function_decorator",
				FunctionDecorator: &v,
			})
		default:
			parts = append(parts, Text(fmt.Sprintf("%v", v)))
		}
	}
	return parts
}

func toDecorators(v interface{}) []ExpectedDecorator {
	switch val := v.(type) {
	case ExpectedDecorator:
		return []ExpectedDecorator{val}
	case []ExpectedDecorator:
		return val
	case []interface{}:
		var decorators []ExpectedDecorator
		for _, item := range val {
			if dec, ok := item.(ExpectedDecorator); ok {
				decorators = append(decorators, dec)
			}
		}
		return decorators
	default:
		return []ExpectedDecorator{}
	}
}

// Helper functions for string validation
func isDurationString(s string) bool {
	if len(s) < 2 {
		return false
	}

	suffixes := []string{"ns", "us", "μs", "ms", "s", "m", "h"}
	for _, suffix := range suffixes {
		if strings.HasSuffix(s, suffix) {
			prefix := s[:len(s)-len(suffix)]
			if _, err := strconv.ParseFloat(prefix, 64); err == nil {
				return true
			}
		}
	}
	return false
}

func isNumericString(s string) bool {
	if _, err := strconv.ParseInt(s, 10, 64); err == nil {
		return true
	}
	if _, err := strconv.ParseFloat(s, 64); err == nil {
		return true
	}
	return false
}

// Test helper types for the new CST structure
type ExpectedProgram struct {
	Variables []ExpectedVariable
	Commands  []ExpectedCommand
}

type ExpectedVariable struct {
	Name  string
	Value ExpectedExpression
}

type ExpectedCommand struct {
	Name string
	Type ast.CommandType
	Body ExpectedCommandBody
}

type ExpectedCommandBody struct {
	IsBlock bool
	Content ExpectedCommandContent
}

type ExpectedCommandContent interface {
	IsExpectedCommandContent() bool
}

type ExpectedShellContent struct {
	Parts []ExpectedShellPart
}

func (s ExpectedShellContent) IsExpectedCommandContent() bool { return true }

type ExpectedDecoratedContent struct {
	Decorators []ExpectedDecorator
	Content    ExpectedCommandContent
}

func (d ExpectedDecoratedContent) IsExpectedCommandContent() bool { return true }

type ExpectedShellPart struct {
	Type              string
	Text              string
	FunctionDecorator *ExpectedFunctionDecorator
}

type ExpectedDecorator struct {
	Name string
	Args []ExpectedExpression
}

type ExpectedFunctionDecorator struct {
	Name string
	Args []ExpectedExpression
}

type ExpectedExpression struct {
	Type  string
	Value string
}

// Test case structure
type TestCase struct {
	Name        string
	Input       string
	WantErr     bool
	ErrorSubstr string
	Expected    ExpectedProgram
}

// Comparison helpers for the new CST structure
func expressionToComparable(expr ast.Expression) interface{} {
	switch e := expr.(type) {
	case *ast.StringLiteral:
		return map[string]interface{}{
			"Type":  "string",
			"Value": e.Value,
		}
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
	case *ast.Identifier:
		return map[string]interface{}{
			"Type":  "identifier",
			"Value": e.Name,
		}
	case *ast.FunctionDecorator:
		args := make([]interface{}, len(e.Args))
		for i, arg := range e.Args {
			args[i] = expressionToComparable(arg)
		}
		return map[string]interface{}{
			"Type":  "function_decorator",
			"Name":  e.Name,
			"Args":  args,
		}
	default:
		return map[string]interface{}{
			"Type":  "unknown",
			"Value": expr.String(),
		}
	}
}

func expectedExpressionToComparable(expr ExpectedExpression) interface{} {
	return map[string]interface{}{
		"Type":  expr.Type,
		"Value": expr.Value,
	}
}

func shellPartToComparable(part ast.ShellPart) interface{} {
	switch p := part.(type) {
	case *ast.TextPart:
		return map[string]interface{}{
			"Type": "text",
			"Text": p.Text,
		}
	case *ast.FunctionDecorator:
		args := make([]interface{}, len(p.Args))
		for i, arg := range p.Args {
			args[i] = expressionToComparable(arg)
		}
		return map[string]interface{}{
			"Type": "function_decorator",
			"FunctionDecorator": map[string]interface{}{
				"Name": p.Name,
				"Args": args,
			},
		}
	default:
		return map[string]interface{}{
			"Type": "unknown",
			"Text": part.String(),
		}
	}
}

func expectedShellPartToComparable(part ExpectedShellPart) interface{} {
	result := map[string]interface{}{
		"Type": part.Type,
	}

	switch part.Type {
	case "text":
		result["Text"] = part.Text
	case "function_decorator":
		if part.FunctionDecorator != nil {
			args := make([]interface{}, len(part.FunctionDecorator.Args))
			for i, arg := range part.FunctionDecorator.Args {
				args[i] = expectedExpressionToComparable(arg)
			}
			result["FunctionDecorator"] = map[string]interface{}{
				"Name": part.FunctionDecorator.Name,
				"Args": args,
			}
		}
	}

	return result
}

func commandContentToComparable(content ast.CommandContent) interface{} {
	switch c := content.(type) {
	case *ast.ShellContent:
		parts := make([]interface{}, len(c.Parts))
		for i, part := range c.Parts {
			parts[i] = shellPartToComparable(part)
		}
		return map[string]interface{}{
			"Type":  "shell",
			"Parts": parts,
		}
	case *ast.DecoratedContent:
		decorators := make([]interface{}, len(c.Decorators))
		for i, decorator := range c.Decorators {
			args := make([]interface{}, len(decorator.Args))
			for j, arg := range decorator.Args {
				args[j] = expressionToComparable(arg)
			}
			decorators[i] = map[string]interface{}{
				"Name": decorator.Name,
				"Args": args,
			}
		}
		return map[string]interface{}{
			"Type":       "decorated",
			"Decorators": decorators,
			"Content":    commandContentToComparable(c.Content),
		}
	default:
		return map[string]interface{}{
			"Type": "unknown",
		}
	}
}

func expectedCommandContentToComparable(content ExpectedCommandContent) interface{} {
	switch c := content.(type) {
	case ExpectedShellContent:
		parts := make([]interface{}, len(c.Parts))
		for i, part := range c.Parts {
			parts[i] = expectedShellPartToComparable(part)
		}
		return map[string]interface{}{
			"Type":  "shell",
			"Parts": parts,
		}
	case ExpectedDecoratedContent:
		decorators := make([]interface{}, len(c.Decorators))
		for i, decorator := range c.Decorators {
			args := make([]interface{}, len(decorator.Args))
			for j, arg := range decorator.Args {
				args[j] = expectedExpressionToComparable(arg)
			}
			decorators[i] = map[string]interface{}{
				"Name": decorator.Name,
				"Args": args,
			}
		}
		return map[string]interface{}{
			"Type":       "decorated",
			"Decorators": decorators,
			"Content":    expectedCommandContentToComparable(c.Content),
		}
	default:
		return map[string]interface{}{
			"Type": "unknown",
		}
	}
}

func commandBodyToComparable(body ast.CommandBody) interface{} {
	return map[string]interface{}{
		"IsBlock": body.IsBlock,
		"Content": commandContentToComparable(body.Content),
	}
}

func expectedCommandBodyToComparable(body ExpectedCommandBody) interface{} {
	return map[string]interface{}{
		"IsBlock": body.IsBlock,
		"Content": expectedCommandContentToComparable(body.Content),
	}
}

func RunTestCase(t *testing.T, tc TestCase) {
	t.Run(tc.Name, func(t *testing.T) {
		program, err := Parse(tc.Input)

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

				actualComparable := map[string]interface{}{
					"Name": actualCmd.Name,
					"Type": actualCmd.Type,
					"Body": commandBodyToComparable(actualCmd.Body),
				}

				expectedComparable := map[string]interface{}{
					"Name": expectedCmd.Name,
					"Type": expectedCmd.Type,
					"Body": expectedCommandBodyToComparable(expectedCmd.Body),
				}

				if diff := cmp.Diff(expectedComparable, actualComparable); diff != "" {
					t.Errorf("Command[%d] mismatch (-expected +actual):\n%s", i, diff)
				}
			}
		}
	})
}
