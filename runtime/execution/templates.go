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
	CommandChain               []ChainElement
	BaseName                   string
	WorkingDir                 string // Working directory for command execution
	NeedsShellCommandWithInput bool   // Whether executeShellCommandWithInput is needed
	NeedsShellCommand          bool   // Whether executeShellCommand is needed
	NeedsAppendToFile          bool   // Whether appendToFile is needed
	NeedsLastResult            bool   // Whether lastResult variable is needed
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
		
		// Create buffers to capture output while streaming to terminal
		var {{.BaseName}}Stdout, {{.BaseName}}Stderr bytes.Buffer
		
		// Use MultiWriter to stream to terminal AND capture for CommandResult
		{{.ExecVarName}}.Stdout = io.MultiWriter(os.Stdout, &{{.BaseName}}Stdout)
		{{.ExecVarName}}.Stderr = io.MultiWriter(os.Stderr, &{{.BaseName}}Stderr)
		{{.ExecVarName}}.Stdin = os.Stdin
		
		{{.BaseName}}Err := {{.ExecVarName}}.Run()
		{{.BaseName}}ExitCode := 0
		if {{.BaseName}}Err != nil {
			if exitError, ok := {{.BaseName}}Err.(*exec.ExitError); ok {
				{{.BaseName}}ExitCode = exitError.ExitCode()
			} else {
				{{.BaseName}}ExitCode = 1
			}
		}
		
		return CommandResult{Stdout: {{.BaseName}}Stdout.String(), Stderr: {{.BaseName}}Stderr.String(), ExitCode: {{.BaseName}}ExitCode}`

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

	// Analyze command chain to determine which helper functions are needed
	needsShellCommandWithInput := false
	needsShellCommand := false
	needsAppendToFile := false
	needsLastResult := false
	hasNonCmdActions := false

	for _, element := range commandChain {
		if element.Type == "text" {
			if element.IsPipeTarget {
				needsShellCommandWithInput = true
				needsLastResult = true
			} else if element.IsFileTarget {
				needsAppendToFile = true
				needsLastResult = true
			} else {
				needsShellCommand = true
				needsLastResult = true
			}
		} else if element.Type == "operator" {
			needsLastResult = true
		} else if element.Type == "action" && element.ActionName != "cmd" {
			hasNonCmdActions = true
			needsLastResult = true
		}
	}

	// For pure @cmd chains without operators, we don't need lastResult
	if !needsLastResult && !hasNonCmdActions {
		for _, element := range commandChain {
			if element.Type == "operator" {
				needsLastResult = true
				break
			}
		}
	}

	// Create template data
	templateData := ActionChainTemplateData{
		CommandChain:               commandChain,
		BaseName:                   b.getBaseName(),
		WorkingDir:                 b.context.WorkingDir,
		NeedsShellCommandWithInput: needsShellCommandWithInput,
		NeedsShellCommand:          needsShellCommand,
		NeedsAppendToFile:          needsAppendToFile,
		NeedsLastResult:            needsLastResult,
	}

	// Return the action chain template
	const actionChainTemplate = `// ActionDecorator command chain with Go-native operators
		{{if .NeedsShellCommandWithInput}}
		// Helper function for executing shell commands with piped input
		executeShellCommandWithInput := func(ctx context.Context, command, input string) CommandResult {
			cmd := exec.CommandContext(ctx, "sh", "-c", command){{if .WorkingDir}}
			cmd.Dir = {{printf "%q" .WorkingDir}}{{end}}
			cmd.Stdin = strings.NewReader(input)
			
			var stdout, stderr bytes.Buffer
			cmd.Stdout = io.MultiWriter(os.Stdout, &stdout)
			cmd.Stderr = io.MultiWriter(os.Stderr, &stderr)
			
			err := cmd.Run()
			exitCode := 0
			if err != nil {
				if exitError, ok := err.(*exec.ExitError); ok {
					exitCode = exitError.ExitCode()
				} else {
					exitCode = 1
				}
			}
			
			return CommandResult{
				Stdout:   stdout.String(),
				Stderr:   stderr.String(),
				ExitCode: exitCode,
			}
		}
		{{end}}
		{{if .NeedsShellCommand}}
		// Helper function for executing shell commands
		executeShellCommand := func(ctx context.Context, command string) CommandResult {
			cmd := exec.CommandContext(ctx, "sh", "-c", command){{if .WorkingDir}}
			cmd.Dir = {{printf "%q" .WorkingDir}}{{end}}
			
			var stdout, stderr bytes.Buffer
			cmd.Stdout = io.MultiWriter(os.Stdout, &stdout)
			cmd.Stderr = io.MultiWriter(os.Stderr, &stderr)
			
			err := cmd.Run()
			exitCode := 0
			if err != nil {
				if exitError, ok := err.(*exec.ExitError); ok {
					exitCode = exitError.ExitCode()
				} else {
					exitCode = 1
				}
			}
			
			return CommandResult{
				Stdout:   stdout.String(),
				Stderr:   stderr.String(),
				ExitCode: exitCode,
			}
		}
		{{end}}
		{{if .NeedsAppendToFile}}
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
		{{end}}
		{{if .NeedsLastResult}}
		var lastResult CommandResult
		{{end}}
{{range $i, $element := .CommandChain}}
{{- if eq $element.Type "action"}}
{{- if eq $element.ActionName "cmd"}}
		// @cmd decorator - call the referenced command function directly
		{{$element.VariableName}} := {{cmdFunctionName $element.ActionArgs}}()
		{{if $.NeedsLastResult}}lastResult = {{$element.VariableName}}{{end}}
		if {{$element.VariableName}}.Failed() {
			return {{$element.VariableName}}
		}
{{- else}}
		{{$element.VariableName}} := execute{{title $element.ActionName}}Decorator(ctx, {{formatParams $element.ActionArgs}})
		{{if $.NeedsLastResult}}lastResult = {{$element.VariableName}}{{end}}
		if {{$element.VariableName}}.Failed() {
			return {{$element.VariableName}}
		}
{{- end}}
{{- else if eq $element.Type "operator"}}
		// {{$element.Operator}} operator - conditional execution logic
{{- if eq $element.Operator "&&"}}
		// AND: next command runs only if previous succeeded
		if lastResult.Failed() {
			return lastResult
		}
{{- else if eq $element.Operator "||"}}
		// OR: next command runs only if previous failed
		if !lastResult.Failed() {
			return lastResult // Skip remaining commands in chain
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
			return {{$element.VariableName}}
		}
{{- else if $element.IsFileTarget}}
		if err := appendToFile({{printf "%q" $element.Text}}, lastResult.Stdout); err != nil {
			return CommandResult{Stdout: "", Stderr: err.Error(), ExitCode: 1}
		}
		// Set lastResult to indicate successful file operation
		lastResult = CommandResult{Stdout: "", Stderr: "", ExitCode: 0}
{{- else}}
		{{$element.VariableName}} := executeShellCommand(ctx, {{printf "%q" $element.Text}})
		lastResult = {{$element.VariableName}}
		if {{$element.VariableName}}.Failed() {
			return {{$element.VariableName}}
		}
{{- end}}
{{- end}}
{{end}}

		return CommandResult{Stdout: "", Stderr: "", ExitCode: 0}`

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
	// Look up the block decorator using the dependency injection lookup to avoid circular imports
	if b.context.blockDecoratorLookup == nil {
		return "", fmt.Errorf("block decorator lookup not available (engine not properly initialized)")
	}
	
	decoratorInterface, exists := b.context.blockDecoratorLookup(blockDecorator.Name)
	if !exists {
		return "", fmt.Errorf("block decorator @%s not found", blockDecorator.Name)
	}

	// Cast to the expected interface type
	decorator, ok := decoratorInterface.(interface {
		Execute(ctx *ExecutionContext, params []ast.NamedParameter, content []ast.CommandContent) *ExecutionResult
	})
	if !ok {
		return "", fmt.Errorf("block decorator @%s does not implement expected Execute method", blockDecorator.Name)
	}

	// Execute the decorator in generator mode to get the generated Go code
	// Create a child context in generator mode to ensure proper code generation
	generatorCtx := b.context.WithMode(GeneratorMode)
	result := decorator.Execute(generatorCtx, blockDecorator.Args, blockDecorator.Content)
	
	if result.Error != nil {
		return "", fmt.Errorf("failed to generate code for @%s decorator: %w", blockDecorator.Name, result.Error)
	}
	
	// The result should contain the generated Go code as a string
	if generatedCode, ok := result.Data.(string); ok {
		return generatedCode, nil
	}
	
	return "", fmt.Errorf("@%s decorator returned unexpected data type for generator mode: %T", blockDecorator.Name, result.Data)
}

// generatePatternDecoratorTemplate creates template string for executing a pattern decorator
func (b *ShellCodeBuilder) generatePatternDecoratorTemplate(patternDecorator *ast.PatternDecorator) (string, error) {
	// Look up the pattern decorator using the dependency injection lookup to avoid circular imports
	if b.context.patternDecoratorLookup == nil {
		return "", fmt.Errorf("pattern decorator lookup not available (engine not properly initialized)")
	}
	
	decoratorInterface, exists := b.context.patternDecoratorLookup(patternDecorator.Name)
	if !exists {
		return "", fmt.Errorf("pattern decorator @%s not found", patternDecorator.Name)
	}

	// Cast to the expected interface type
	decorator, ok := decoratorInterface.(interface {
		Execute(ctx *ExecutionContext, params []ast.NamedParameter, patterns []ast.PatternBranch) *ExecutionResult
	})
	if !ok {
		return "", fmt.Errorf("pattern decorator @%s does not implement expected Execute method", patternDecorator.Name)
	}

	// Execute the decorator in generator mode to get the generated Go code
	// Create a child context in generator mode to ensure proper code generation
	generatorCtx := b.context.WithMode(GeneratorMode)
	result := decorator.Execute(generatorCtx, patternDecorator.Args, patternDecorator.Patterns)
	
	if result.Error != nil {
		return "", fmt.Errorf("failed to generate code for @%s decorator: %w", patternDecorator.Name, result.Error)
	}
	
	// The result should contain the generated Go code as a string
	if generatedCode, ok := result.Data.(string); ok {
		return generatedCode, nil
	}
	
	return "", fmt.Errorf("@%s decorator returned unexpected data type for generator mode: %T", patternDecorator.Name, result.Data)
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

// getBaseName returns the base name for variable generation with descriptive naming
func (b *ShellCodeBuilder) getBaseName() string {
	b.context.shellCounter++
	
	// Create a descriptive base name using camelCase convention
	baseName := "command"
	if b.context.currentCommand != "" {
		// Convert to proper camelCase handling hyphens, underscores, and spaces
		baseName = b.toCamelCase(b.context.currentCommand)
	}
	
	// Use descriptive naming instead of just numbers
	if b.context.shellCounter == 1 {
		return baseName
	}
	return fmt.Sprintf("%sStep%d", baseName, b.context.shellCounter)
}

// formatParams formats parameters for Go code generation
func (b *ShellCodeBuilder) formatParams(params []ast.NamedParameter) string {
	if len(params) == 0 {
		return "nil"
	}
	// For now, return simple representation - this needs to be expanded
	return "[]ast.NamedParameter{}"
}

// toCamelCase converts a command name to camelCase for function naming
func (b *ShellCodeBuilder) toCamelCase(name string) string {
	// Handle different separators: hyphens, underscores, and spaces
	parts := strings.FieldsFunc(name, func(c rune) bool {
		return c == '-' || c == '_' || c == ' '
	})
	
	if len(parts) == 0 {
		return name
	}
	
	// First part stays lowercase, subsequent parts get title case
	result := strings.ToLower(parts[0])
	for i := 1; i < len(parts); i++ {
		result += strings.Title(strings.ToLower(parts[i]))
	}
	
	return result
}

// GetTemplateFunctions returns the template functions that should be available to all templates
func (b *ShellCodeBuilder) GetTemplateFunctions() template.FuncMap {
	return template.FuncMap{
		"generateShellCode": func(cmd ast.CommandContent) (string, error) {
			// Use the existing context which should already be in the correct mode
			// Don't create a new child context - this preserves all context state including working directory
			return b.GenerateShellCode(cmd)
		},
		"formatParams": b.formatParams,
		"title":        strings.Title,
		"cmdFunctionName": func(args []ast.NamedParameter) string {
			// Extract command name from @cmd arguments and convert to function name
			if len(args) == 0 {
				return "unknownCommand"
			}
			// Get the first argument (should be the command name)
			nameParam := args[0]
			if ident, ok := nameParam.Value.(*ast.Identifier); ok {
				return "execute" + strings.Title(b.toCamelCase(ident.Name))
			}
			return "unknownCommand"
		},
	}
}