package execution

import (
	"fmt"
	"strings"
	"text/template"

	"github.com/aledsdavies/devcmd/core/ast"
)

// ShellCodeBuilder provides unified shell code generation for templates
type ShellCodeBuilder struct {
	context *ExecutionContext
}

// NewShellCodeBuilder creates a new shell code builder
func NewShellCodeBuilder(ctx *ExecutionContext) *ShellCodeBuilder {
	return &ShellCodeBuilder{
		context: ctx,
	}
}

// ShellTemplateData holds data for shell execution templates
type ShellTemplateData struct {
	VarDeclarations []string
	FormatString    string
	FormatArgs      []string
	HasFormatArgs   bool
	CmdVarName      string
	ExecVarName     string
	BaseName        string
	CommandString   string
	WorkingDir      string // Working directory for command execution
}

// ActionChainTemplateData holds data for action decorator chain templates
type ActionChainTemplateData struct {
	CommandChain []ChainElement
	BaseName     string
	WorkingDir   string // Working directory for command execution
}

// GenerateShellCode converts AST CommandContent to template string for Go shell execution
// This is the main template function that block decorators use
func (b *ShellCodeBuilder) GenerateShellCode(cmd ast.CommandContent) (string, error) {
	switch c := cmd.(type) {
	case *ast.ShellContent:
		return b.GenerateShellExecutionTemplate(c)
	case *ast.BlockDecorator:
		return b.generateBlockDecoratorTemplate(c)
	case *ast.PatternDecorator:
		return b.generatePatternDecoratorTemplate(c)
	default:
		return "", fmt.Errorf("unsupported command content type for shell generation: %T", cmd)
	}
}

// GenerateShellExecutionTemplate creates template string for executing shell content
func (b *ShellCodeBuilder) GenerateShellExecutionTemplate(content *ast.ShellContent) (string, error) {
	var formatParts []string
	var formatArgs []string
	var hasActionDecorators bool

	// First pass: check for ActionDecorators or shell operators
	var hasShellOperators bool
	for _, part := range content.Parts {
		if _, ok := part.(*ast.ActionDecorator); ok {
			hasActionDecorators = true
			break
		}
		if textPart, ok := part.(*ast.TextPart); ok {
			// Check if text contains shell operators
			text := strings.TrimSpace(textPart.Text)
			if strings.Contains(text, "&&") || strings.Contains(text, "||") || 
			   strings.Contains(text, "|") || strings.Contains(text, ">>") {
				hasShellOperators = true
			}
		}
	}

	if hasActionDecorators || hasShellOperators {
		// Generate direct ActionDecorator execution template
		return b.GenerateDirectActionTemplate(content)
	}

	// Build format string and arguments for shell command
	for _, part := range content.Parts {
		switch p := part.(type) {
		case *ast.TextPart:
			formatParts = append(formatParts, p.Text)
		case *ast.ValueDecorator:
			formatParts = append(formatParts, "%s")
			// Call the ValueDecorator to get the proper generated code
			result, err := b.context.processValueDecoratorUnified(p)
			if err != nil {
				return "", fmt.Errorf("failed to process value decorator @%s: %w", p.Name, err)
			}
			if code, ok := result.(string); ok {
				formatArgs = append(formatArgs, code)
			} else {
				return "", fmt.Errorf("value decorator @%s returned non-string result: %T", p.Name, result)
			}
		default:
			return "", fmt.Errorf("unsupported shell part type: %T", part)
		}
	}

	// Get base name for variables
	baseName := b.getBaseName()

	// Create template data
	templateData := ShellTemplateData{
		FormatString:  strings.Join(formatParts, ""),
		FormatArgs:    formatArgs,
		HasFormatArgs: len(formatArgs) > 0,
		CmdVarName:    fmt.Sprintf("%sCmdStr", baseName),
		ExecVarName:   fmt.Sprintf("%sExecCmd", baseName),
		BaseName:      baseName,
		WorkingDir:    b.context.WorkingDir, // Include working directory from context
	}

	// Return the shell execution template
	const shellExecTemplate = `{{if .HasFormatArgs}}{{.CmdVarName}} := fmt.Sprintf({{printf "%q" .FormatString}}, {{range $i, $arg := .FormatArgs}}{{if $i}}, {{end}}{{$arg}}{{end}}){{else}}{{.CmdVarName}} := {{printf "%q" .FormatString}}{{end}}
		{{.ExecVarName}} := exec.CommandContext(ctx, "sh", "-c", {{.CmdVarName}}){{if .WorkingDir}}
		{{.ExecVarName}}.Dir = {{printf "%q" .WorkingDir}} // Set working directory{{end}}
		{{.ExecVarName}}.Stdout = os.Stdout
		{{.ExecVarName}}.Stderr = os.Stderr
		{{.ExecVarName}}.Stdin = os.Stdin
		if err := {{.ExecVarName}}.Run(); err != nil {
			return fmt.Errorf("command failed: %v", err)
		}`

	// Execute the template with our data
	tmpl, err := template.New("shellExec").Parse(shellExecTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse shell execution template: %w", err)
	}

	var result strings.Builder
	if err := tmpl.Execute(&result, templateData); err != nil {
		return "", fmt.Errorf("failed to execute shell execution template: %w", err)
	}

	return result.String(), nil
}

// GenerateDirectActionTemplate creates template string for ActionDecorator direct execution
func (b *ShellCodeBuilder) GenerateDirectActionTemplate(content *ast.ShellContent) (string, error) {
	// Parse the shell content into a sequence of commands and operators
	commandChain, err := b.parseActionDecoratorChain(content)
	if err != nil {
		return "", fmt.Errorf("failed to parse ActionDecorator chain: %w", err)
	}

	// Create template data
	templateData := ActionChainTemplateData{
		CommandChain: commandChain,
		BaseName:     b.getBaseName(),
		WorkingDir:   b.context.WorkingDir,
	}

	// Return the action chain template
	const actionChainTemplate = `// ActionDecorator command chain with Go-native operators
		
		// Helper function for executing shell commands with piped input
		executeShellCommandWithInput := func(ctx context.Context, command, input string) execution.CommandResult {
			cmd := exec.CommandContext(ctx, "sh", "-c", command){{if .WorkingDir}}
			cmd.Dir = {{printf "%q" .WorkingDir}}{{end}}
			cmd.Stdin = strings.NewReader(input)
			
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			
			err := cmd.Run()
			exitCode := 0
			if err != nil {
				if exitError, ok := err.(*exec.ExitError); ok {
					exitCode = exitError.ExitCode()
				} else {
					exitCode = 1
				}
			}
			
			return execution.CommandResult{
				Stdout:   stdout.String(),
				Stderr:   stderr.String(),
				ExitCode: exitCode,
			}
		}
		
		// Helper function for executing shell commands
		executeShellCommand := func(ctx context.Context, command string) execution.CommandResult {
			cmd := exec.CommandContext(ctx, "sh", "-c", command){{if .WorkingDir}}
			cmd.Dir = {{printf "%q" .WorkingDir}}{{end}}
			
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			
			err := cmd.Run()
			exitCode := 0
			if err != nil {
				if exitError, ok := err.(*exec.ExitError); ok {
					exitCode = exitError.ExitCode()
				} else {
					exitCode = 1
				}
			}
			
			return execution.CommandResult{
				Stdout:   stdout.String(),
				Stderr:   stderr.String(),
				ExitCode: exitCode,
			}
		}
		
		// Helper function for appending content to files
		appendToFile := func(filename, content string) error {
			file, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
			if err != nil {
				return fmt.Errorf("failed to open file %s: %v", filename, err)
			}
			defer file.Close()
			
			_, err = file.WriteString(content)
			if err != nil {
				return fmt.Errorf("failed to write to file %s: %v", filename, err)
			}
			
			return nil
		}
		
		var lastResult execution.CommandResult
{{range $i, $element := .CommandChain}}
{{- if eq $element.Type "action"}}
		{{$element.VariableName}} := execute{{title $element.ActionName}}Decorator(ctx, {{formatParams $element.ActionArgs}})
		lastResult = {{$element.VariableName}}
		if {{$element.VariableName}}.Failed() {
			return fmt.Errorf("@{{$element.ActionName}} failed: %v", {{$element.VariableName}}.Error())
		}
{{- else if eq $element.Type "operator"}}
		// {{$element.Operator}} operator - conditional execution logic
{{- if eq $element.Operator "&&"}}
		// AND: next command runs only if previous succeeded
		if lastResult.Failed() {
			return fmt.Errorf("previous command failed")
		}
{{- else if eq $element.Operator "||"}}
		// OR: next command runs only if previous failed
		if lastResult.Success() {
			return nil // Skip remaining commands in chain
		}
{{- else if eq $element.Operator "|"}}
		// PIPE: stdout of previous feeds to next command
		// Next command will use executeShellCommandWithInput
{{- else if eq $element.Operator ">>"}}
		// APPEND: stdout appends to file
		// Next element should be filename for file operation
{{- end}}
{{- else if eq $element.Type "text"}}
{{- if $element.IsPipeTarget}}
		{{$element.VariableName}} := executeShellCommandWithInput(ctx, {{printf "%q" $element.Text}}, lastResult.Stdout)
		lastResult = {{$element.VariableName}}
		if {{$element.VariableName}}.Failed() {
			return fmt.Errorf("piped command failed: %v", {{$element.VariableName}}.Error())
		}
{{- else if $element.IsFileTarget}}
		if err := appendToFile({{printf "%q" $element.Text}}, lastResult.Stdout); err != nil {
			return fmt.Errorf("file append failed: %v", err)
		}
		// Set lastResult to indicate successful file operation
		lastResult = execution.CommandResult{Stdout: "", Stderr: "", ExitCode: 0}
{{- else}}
		{{$element.VariableName}} := executeShellCommand(ctx, {{printf "%q" $element.Text}})
		lastResult = {{$element.VariableName}}
		if {{$element.VariableName}}.Failed() {
			return fmt.Errorf("shell command failed: %v", {{$element.VariableName}}.Error())
		}
{{- end}}
{{- end}}
{{end}}`

	// Execute the template with our data
	tmpl, err := template.New("actionChain").Funcs(b.GetTemplateFunctions()).Parse(actionChainTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse action chain template: %w", err)
	}

	var result strings.Builder
	if err := tmpl.Execute(&result, templateData); err != nil {
		return "", fmt.Errorf("failed to execute action chain template: %w", err)
	}

	return result.String(), nil
}

// generateBlockDecoratorTemplate creates template string for executing a block decorator
func (b *ShellCodeBuilder) generateBlockDecoratorTemplate(blockDecorator *ast.BlockDecorator) (string, error) {
	// Generate a template function call to the block decorator's execution function
	functionName := fmt.Sprintf("execute%sDecorator", strings.Title(blockDecorator.Name))
	templateStr := fmt.Sprintf(`if err := %s(ctx, {{formatParams .Params}}, content); err != nil {
			return fmt.Errorf("@%s decorator failed: %%v", err)
		}`, functionName, blockDecorator.Name)
	
	// Create template data with parameters
	templateData := struct {
		Params []ast.NamedParameter
	}{
		Params: blockDecorator.Args,
	}

	// Execute the template
	tmpl, err := template.New("blockDecorator").Funcs(b.GetTemplateFunctions()).Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse block decorator template: %w", err)
	}

	var result strings.Builder
	if err := tmpl.Execute(&result, templateData); err != nil {
		return "", fmt.Errorf("failed to execute block decorator template: %w", err)
	}

	return result.String(), nil
}

// generatePatternDecoratorTemplate creates template string for executing a pattern decorator
func (b *ShellCodeBuilder) generatePatternDecoratorTemplate(patternDecorator *ast.PatternDecorator) (string, error) {
	// Generate a template function call to the pattern decorator's execution function
	functionName := fmt.Sprintf("execute%sDecorator", strings.Title(patternDecorator.Name))
	templateStr := fmt.Sprintf(`if err := %s(ctx, {{formatParams .Params}}, patterns); err != nil {
			return fmt.Errorf("@%s decorator failed: %%v", err)
		}`, functionName, patternDecorator.Name)
	
	// Create template data with parameters
	templateData := struct {
		Params []ast.NamedParameter
	}{
		Params: patternDecorator.Args,
	}

	// Execute the template
	tmpl, err := template.New("patternDecorator").Funcs(b.GetTemplateFunctions()).Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse pattern decorator template: %w", err)
	}

	var result strings.Builder
	if err := tmpl.Execute(&result, templateData); err != nil {
		return "", fmt.Errorf("failed to execute pattern decorator template: %w", err)
	}

	return result.String(), nil
}

// parseActionDecoratorChain parses shell content into a chain of commands and operators
func (b *ShellCodeBuilder) parseActionDecoratorChain(content *ast.ShellContent) ([]ChainElement, error) {
	var chain []ChainElement
	var currentIndex int

	for _, part := range content.Parts {
		switch p := part.(type) {
		case *ast.ActionDecorator:
			element := ChainElement{
				Type:         "action",
				ActionName:   p.Name,
				ActionArgs:   p.Args,
				VariableName: fmt.Sprintf("%sResult%d", b.getBaseName(), currentIndex),
			}
			chain = append(chain, element)
			currentIndex++

		case *ast.TextPart:
			text := strings.TrimSpace(p.Text)
			if text == "" {
				continue
			}

			// Parse shell text for operators (&&, ||, |, >>)
			parsed, err := b.parseShellOperators(text)
			if err != nil {
				return nil, fmt.Errorf("failed to parse shell operators: %w", err)
			}

			// Add parsed elements to chain
			for _, element := range parsed {
				if element.Type == "text" && element.Text != "" {
					element.VariableName = fmt.Sprintf("%sShell%d", b.getBaseName(), currentIndex)
					currentIndex++
				}
				chain = append(chain, element)
			}

		case *ast.ValueDecorator:
			// ValueDecorators in ActionDecorator context should be resolved to values
			if value, exists := b.context.GetVariable(p.Name); exists {
				element := ChainElement{
					Type: "text",
					Text: value,
				}
				chain = append(chain, element)
			} else {
				return nil, fmt.Errorf("undefined variable in ActionDecorator chain: %s", p.Name)
			}
		}
	}

	// Mark pipe targets and file targets
	for i := 0; i < len(chain); i++ {
		if chain[i].Type == "operator" {
			if chain[i].Operator == "|" && i+1 < len(chain) && chain[i+1].Type == "text" {
				// Next element is a pipe target
				chain[i+1].IsPipeTarget = true
			} else if chain[i].Operator == ">>" && i+1 < len(chain) && chain[i+1].Type == "text" {
				// Next element is a file target
				chain[i+1].IsFileTarget = true
			}
		}
	}

	return chain, nil
}

// parseShellOperators parses shell text and splits it on operators (&&, ||, |, >>)
// Returns a sequence of ChainElements representing commands and operators
func (b *ShellCodeBuilder) parseShellOperators(text string) ([]ChainElement, error) {
	var elements []ChainElement
	var current strings.Builder
	inQuotes := false
	var quoteChar rune
	
	i := 0
	for i < len(text) {
		char := rune(text[i])
		
		// Handle quotes
		if char == '"' || char == '\'' {
			if !inQuotes {
				inQuotes = true
				quoteChar = char
			} else if char == quoteChar {
				inQuotes = false
			}
			current.WriteRune(char)
			i++
			continue
		}
		
		// If we're in quotes, just add the character
		if inQuotes {
			current.WriteRune(char)
			i++
			continue
		}
		
		// Check for operators
		if i < len(text)-1 {
			twoChar := text[i:i+2]
			switch twoChar {
			case "&&", "||", ">>":
				// Add current command if not empty
				cmd := strings.TrimSpace(current.String())
				if cmd != "" {
					elements = append(elements, ChainElement{
						Type: "text",
						Text: cmd,
					})
				}
				current.Reset()
				
				// Add operator
				elements = append(elements, ChainElement{
					Type:     "operator", 
					Operator: twoChar,
				})
				
				i += 2
				// Skip whitespace after operator
				for i < len(text) && (text[i] == ' ' || text[i] == '\t') {
					i++
				}
				continue
			}
		}
		
		// Check for single character operators
		if char == '|' {
			// Add current command if not empty
			cmd := strings.TrimSpace(current.String())
			if cmd != "" {
				elements = append(elements, ChainElement{
					Type: "text",
					Text: cmd,
				})
			}
			current.Reset()
			
			// Add pipe operator
			elements = append(elements, ChainElement{
				Type:     "operator",
				Operator: "|",
			})
			
			i++
			// Skip whitespace after operator
			for i < len(text) && (text[i] == ' ' || text[i] == '\t') {
				i++
			}
			continue
		}
		
		// Regular character
		current.WriteRune(char)
		i++
	}
	
	// Add final command if not empty
	cmd := strings.TrimSpace(current.String())
	if cmd != "" {
		elements = append(elements, ChainElement{
			Type: "text",
			Text: cmd,
		})
	}
	
	// Validate the chain
	if err := b.validateChain(elements); err != nil {
		return nil, err
	}
	
	return elements, nil
}

// validateChain validates that the parsed chain is well-formed
func (b *ShellCodeBuilder) validateChain(elements []ChainElement) error {
	if len(elements) == 0 {
		return nil
	}
	
	// Chain should start with a command, not an operator
	if elements[0].Type == "operator" {
		return fmt.Errorf("chain cannot start with operator %s", elements[0].Operator)
	}
	
	// Chain should end with a command, not an operator
	if elements[len(elements)-1].Type == "operator" {
		return fmt.Errorf("chain cannot end with operator %s", elements[len(elements)-1].Operator)
	}
	
	// Operators and commands should alternate
	for i := 0; i < len(elements)-1; i++ {
		current := elements[i]
		next := elements[i+1]
		
		if current.Type == "operator" && next.Type == "operator" {
			return fmt.Errorf("consecutive operators not allowed: %s %s", current.Operator, next.Operator)
		}
		if current.Type == "text" && next.Type == "text" {
			return fmt.Errorf("consecutive commands without operator: %s | %s", current.Text, next.Text)
		}
	}
	
	return nil
}

// getBaseName returns the base name for variable generation
func (b *ShellCodeBuilder) getBaseName() string {
	if b.context.currentCommand != "" {
		return strings.Title(b.context.currentCommand)
	}
	return "Action"
}

// formatParams formats parameters for Go code generation
func (b *ShellCodeBuilder) formatParams(params []ast.NamedParameter) string {
	if len(params) == 0 {
		return "nil"
	}
	// For now, return simple representation - this needs to be expanded
	return "[]ast.NamedParameter{}"
}

// GetTemplateFunctions returns the template functions that should be available to all templates
func (b *ShellCodeBuilder) GetTemplateFunctions() template.FuncMap {
	return template.FuncMap{
		"generateShellCode": func(cmd ast.CommandContent) (string, error) {
			return b.GenerateShellCode(cmd)
		},
		"formatParams": b.formatParams,
		"title":        strings.Title,
	}
}