package engine

import (
	"bytes"
	"strconv"
	"text/template"
)

// Template constants for code generation
const (
	packageTemplate = `package main

`

	importsTemplate = `import (
	"fmt"{{if .HasCommands}}
	"context"
	"os"
	"os/exec"
	"os/signal"
	"syscall"{{end}}{{range .StandardImports}}
	"{{.}}"{{end}}{{if .ThirdPartyImports}}
{{range .ThirdPartyImports}}
	"{{.}}"{{end}}{{end}}
)

`

	mainStartTemplate = `func main() {
`

	signalHandlingTemplate = `	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle Ctrl+C gracefully
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		<-c
		cancel()
	}()

`

	variableTemplate = `	{{.Name}} := {{.Value}}
`

	mainEndTemplate = `}
`
)

// TemplateData holds data for template execution
type TemplateData struct {
	HasCommands       bool
	StandardImports   []string
	ThirdPartyImports []string
	Name              string
	Value             string
}

// executeTemplate executes a template with given data
func executeTemplate(tmplStr string, data interface{}) (string, error) {
	tmpl, err := template.New("template").Parse(tmplStr)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// renderImports renders the imports section
func renderImports(hasCommands bool, standardImports, thirdPartyImports []string) (string, error) {
	// Filter out base imports that are handled by the template
	filteredStandard := make([]string, 0)
	baseImports := map[string]bool{
		"fmt": true, "context": true, "os": true, "os/exec": true,
		"os/signal": true, "syscall": true,
	}

	for _, imp := range standardImports {
		if !baseImports[imp] {
			filteredStandard = append(filteredStandard, imp)
		}
	}

	data := TemplateData{
		HasCommands:       hasCommands,
		StandardImports:   filteredStandard,
		ThirdPartyImports: thirdPartyImports,
	}
	return executeTemplate(importsTemplate, data)
}

// renderVariable renders a variable declaration
func renderVariable(name, value string) (string, error) {
	data := TemplateData{
		Name:  name,
		Value: strconv.Quote(value),
	}
	return executeTemplate(variableTemplate, data)
}
