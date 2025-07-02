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

// Cmd creates a simple command: NAME: BODY
func Cmd(name string, body interface{}) ExpectedCommand {
	return ExpectedCommand{
		Name:       name,
		Type:       ast.Command,
		Decorators: []ExpectedDecorator{},
		Body:       toCommandBody(body),
	}
}

// CmdWith creates a command with decorators: @decorator NAME: BODY
func CmdWith(decorators interface{}, name string, body interface{}) ExpectedCommand {
	return ExpectedCommand{
		Name:       name,
		Type:       ast.Command,
		Decorators: toDecorators(decorators),
		Body:       toCommandBody(body),
	}
}

// Watch creates a watch command: watch NAME: BODY
func Watch(name string, body interface{}) ExpectedCommand {
	return ExpectedCommand{
		Name:       name,
		Type:       ast.WatchCommand,
		Decorators: []ExpectedDecorator{},
		Body:       toCommandBody(body),
	}
}

// WatchWith creates a watch command with decorators
func WatchWith(decorators interface{}, name string, body interface{}) ExpectedCommand {
	return ExpectedCommand{
		Name:       name,
		Type:       ast.WatchCommand,
		Decorators: toDecorators(decorators),
		Body:       toCommandBody(body),
	}
}

// Stop creates a stop command: stop NAME: BODY
func Stop(name string, body interface{}) ExpectedCommand {
	return ExpectedCommand{
		Name:       name,
		Type:       ast.StopCommand,
		Decorators: []ExpectedDecorator{},
		Body:       toCommandBody(body),
	}
}

// StopWith creates a stop command with decorators
func StopWith(decorators interface{}, name string, body interface{}) ExpectedCommand {
	return ExpectedCommand{
		Name:       name,
		Type:       ast.StopCommand,
		Decorators: toDecorators(decorators),
		Body:       toCommandBody(body),
	}
}

// Block creates a block command body: { stmt1; stmt2; ... }
func Block(statements ...interface{}) ExpectedCommandBody {
	var stmts []ExpectedStatement
	for _, stmt := range statements {
		stmts = append(stmts, toStatement(stmt))
	}
	return ExpectedCommandBody{
		IsBlock:    true,
		Statements: stmts,
	}
}

// Simple creates a simple command body (single line)
func Simple(elements ...interface{}) ExpectedCommandBody {
	var elems []ExpectedCommandElement
	for _, elem := range elements {
		elems = append(elems, toElement(elem))
	}
	return ExpectedCommandBody{
		IsBlock:  false,
		Elements: elems,
	}
}

// Text creates a text element
func Text(text string) ExpectedCommandElement {
	return ExpectedCommandElement{
		Type: "text",
		Text: text,
	}
}

// Helper functions for backward compatibility with old test patterns
func StringExpr(value string) string                  { return value }
func NumberExpr(value string) string                  { return value }
func DurationExpr(value string) ExpectedExpression {
	return ExpectedExpression{Type: "duration", Value: value}
}
func IdentifierExpr(value string) string              { return value }
func Statement(elements ...interface{}) []interface{} { return elements }

// At creates a decorator (@decorator or @decorator(args) or @decorator{block})
func At(name string, args ...interface{}) interface{} {
	if len(args) == 0 {
		// Simple decorator: @name
		return ExpectedDecorator{Name: name, Args: []ExpectedExpression{}}
	}

	// Check if last argument is a block
	lastArg := args[len(args)-1]
	if body, ok := lastArg.(ExpectedCommandBody); ok && body.IsBlock {
		// Decorator with block: @name(args...) { block }
		var decoratorArgs []ExpectedExpression
		for _, arg := range args[:len(args)-1] {
			decoratorArgs = append(decoratorArgs, toExpression(arg))
		}
		return ExpectedDecorator{
			Name:  name,
			Args:  decoratorArgs,
			Block: &body,
		}
	}

	// For @var decorator specifically, handle the variable reference
	if name == "var" && len(args) == 1 {
		if argStr, ok := args[0].(string); ok {
			return ExpectedDecorator{
				Name: name,
				Args: []ExpectedExpression{
					{Type: "identifier", Value: argStr},
				},
			}
		}
	}

	// Handle special @var(TIMEOUT) pattern in decorator arguments
	if len(args) == 1 {
		if argStr, ok := args[0].(string); ok && strings.HasPrefix(argStr, "@var(") && strings.HasSuffix(argStr, ")") {
			// Extract variable name from @var(NAME)
			varName := argStr[5 : len(argStr)-1] // Remove "@var(" and ")"
			return ExpectedExpression{
				Type:          "decorator",
				Value:         argStr,
				DecoratorName: "var",
				DecoratorArgs: []ExpectedExpression{
					{Type: "identifier", Value: varName},
				},
			}
		}
	}

	// Decorator with arguments: @name(args...)
	var decoratorArgs []ExpectedExpression
	for _, arg := range args {
		decoratorArgs = append(decoratorArgs, toExpression(arg))
	}

	// Check if this should be a command element or expression
	return ExpectedDecorator{Name: name, Args: decoratorArgs}
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
	case ExpectedDecorator:
		return ExpectedExpression{
			Type:          "decorator",
			Value:         "@" + val.Name,
			DecoratorName: val.Name,
			DecoratorArgs: val.Args,
		}
	default:
		// Try to convert to string and handle as identifier
		str := fmt.Sprintf("%v", val)
		return ExpectedExpression{Type: "identifier", Value: strings.Trim(str, "\"")}
	}
}

// isDurationString checks if a string looks like a duration (30s, 5m, etc.)
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

// isNumericString checks if a string represents a number
func isNumericString(s string) bool {
	if _, err := strconv.ParseInt(s, 10, 64); err == nil {
		return true
	}
	if _, err := strconv.ParseFloat(s, 64); err == nil {
		return true
	}
	return false
}

func toCommandBody(v interface{}) ExpectedCommandBody {
	switch val := v.(type) {
	case ExpectedCommandBody:
		return val
	case string:
		return Simple(Text(val))
	case []interface{}:
		return Simple(val...)
	default:
		return ExpectedCommandBody{}
	}
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

func toStatement(v interface{}) ExpectedStatement {
	switch val := v.(type) {
	case ExpectedStatement:
		return val
	case string:
		return ExpectedStatement{Elements: []ExpectedCommandElement{Text(val)}}
	case []interface{}:
		var elements []ExpectedCommandElement
		for _, elem := range val {
			elements = append(elements, toElement(elem))
		}
		return ExpectedStatement{Elements: elements}
	case ExpectedCommandElement:
		return ExpectedStatement{Elements: []ExpectedCommandElement{val}}
	case ExpectedDecorator:
		return ExpectedStatement{Elements: []ExpectedCommandElement{
			{Type: "decorator", Decorator: &val},
		}}
	default:
		return ExpectedStatement{}
	}
}

func toElement(v interface{}) ExpectedCommandElement {
	switch val := v.(type) {
	case ExpectedCommandElement:
		return val
	case string:
		return Text(val)
	case ExpectedDecorator:
		return ExpectedCommandElement{
			Type:      "decorator",
			Decorator: &val,
		}
	default:
		return Text("")
	}
}

// Test helper types (kept for internal use)
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
	Type       ast.CommandType
	Decorators []ExpectedDecorator
	Body       ExpectedCommandBody
}

type ExpectedCommandBody struct {
	IsBlock    bool
	Elements   []ExpectedCommandElement
	Statements []ExpectedStatement
}

type ExpectedStatement struct {
	Elements []ExpectedCommandElement
}

type ExpectedCommandElement struct {
	Type      string
	Text      string
	Decorator *ExpectedDecorator
}

type ExpectedDecorator struct {
	Name  string
	Args  []ExpectedExpression
	Block *ExpectedCommandBody // Add support for decorator blocks
}

type ExpectedExpression struct {
	Type          string
	Value         string
	DecoratorName string
	DecoratorArgs []ExpectedExpression
}

// Test case structure
type TestCase struct {
	Name        string
	Input       string
	WantErr     bool
	ErrorSubstr string
	Expected    ExpectedProgram
}

// Comparison helpers (updated to handle duration properly)
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
		// Keep duration literals as duration type
		return map[string]interface{}{
			"Type":  "duration",
			"Value": e.Value,
		}
	case *ast.Identifier:
		return map[string]interface{}{
			"Type":  "identifier",
			"Value": e.Name,
		}
	case *ast.Decorator:
		args := make([]interface{}, len(e.Args))
		for i, arg := range e.Args {
			args[i] = expressionToComparable(arg)
		}
		return map[string]interface{}{
			"Type":  "decorator",
			"Value": "@" + e.Name, // Add the Value field to match expected format
			"Decorator": map[string]interface{}{
				"Name": e.Name,
				"Args": args,
			},
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

	if expr.Type == "decorator" {
		args := make([]interface{}, len(expr.DecoratorArgs))
		for i, arg := range expr.DecoratorArgs {
			args[i] = expectedExpressionToComparable(arg)
		}
		result["Decorator"] = map[string]interface{}{
			"Name": expr.DecoratorName,
			"Args": args,
		}
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
		decoratorMap := map[string]interface{}{
			"Name": e.Name,
			"Args": args,
		}
		// Handle decorator block if present
		if e.Block != nil {
			// Convert DecoratorBlock to CommandBody format for comparison
			blockBody := ast.CommandBody{
				Statements: e.Block.Statements,
				IsBlock:    true,
			}
			decoratorMap["Block"] = commandBodyToComparable(blockBody)
		}
		return map[string]interface{}{
			"Type":      "decorator",
			"Decorator": decoratorMap,
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
			decoratorMap := map[string]interface{}{
				"Name": elem.Decorator.Name,
				"Args": args,
			}
			// Handle decorator block if present
			if elem.Decorator.Block != nil {
				decoratorMap["Block"] = expectedCommandBodyToComparable(*elem.Decorator.Block)
			}
			result["Decorator"] = decoratorMap
		}
	}

	return result
}

func commandBodyToComparable(body ast.CommandBody) interface{} {
	result := map[string]interface{}{
		"IsBlock": body.IsBlock,
	}

	if body.IsBlock {
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
