package generator

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"text/template"
	"unicode"

	"github.com/aledsdavies/devcmd/pkgs/parser"
)

// TemplateData represents preprocessed data for template generation
type TemplateData struct {
	PackageName      string
	Imports          []string
	HasProcessMgmt   bool
	Commands         []TemplateCommand
	ProcessMgmtFuncs []string
}

// TemplateCommand represents a command ready for template generation
type TemplateCommand struct {
	Name            string // Original command name
	FunctionName    string // Sanitized Go function name
	GoCase          string // Case statement value
	Type            string // "regular", "watch", "stop"
	ShellCommand    string // The actual shell command to execute
	IsBackground    bool   // For watch commands
	BaseName        string // For stop commands
	HelpDescription string // Description for help text
}

// PreprocessCommands converts parser commands into template-ready data
func PreprocessCommands(cf *parser.CommandFile) (*TemplateData, error) {
	if cf == nil {
		return nil, fmt.Errorf("command file cannot be nil")
	}

	data := &TemplateData{
		PackageName: "main",
		Imports:     []string{},
		Commands:    []TemplateCommand{},
	}

	// Determine what features we need
	hasWatchCommands := false
	for _, cmd := range cf.Commands {
		if cmd.IsWatch {
			hasWatchCommands = true
			break
		}
	}
	data.HasProcessMgmt = hasWatchCommands

	// Set up imports based on features needed
	data.Imports = []string{
		"fmt",
		"os",
		"os/exec",
		"strings",
		"syscall",
	}

	if hasWatchCommands {
		additionalImports := []string{
			"bufio",
			"context",
			"encoding/json",
			"io",
			"os/signal",
			"path/filepath",
			"strconv",
			"time",
		}
		data.Imports = append(data.Imports, additionalImports...)
	}

	// Sort imports for consistent output
	sort.Strings(data.Imports)

	// Process commands
	for _, cmd := range cf.Commands {
		templateCmd, err := processCommand(cmd)
		if err != nil {
			return nil, fmt.Errorf("failed to process command %s: %w", cmd.Name, err)
		}
		data.Commands = append(data.Commands, templateCmd)
	}

	// Add process management functions if needed
	if hasWatchCommands {
		data.ProcessMgmtFuncs = []string{
			"showStatus",
			"showLogs",
			"stopCommand",
			"runInBackground",
		}
	}

	return data, nil
}

// processCommand converts a parser command to a template command
func processCommand(cmd parser.Command) (TemplateCommand, error) {
	templateCmd := TemplateCommand{
		Name:         cmd.Name,
		FunctionName: sanitizeFunctionName(cmd.Name),
		GoCase:       cmd.Name, // Keep original name for case statements
	}

	// Determine command type and generate shell command
	if cmd.IsWatch {
		templateCmd.Type = "watch"
		templateCmd.IsBackground = true
		templateCmd.HelpDescription = fmt.Sprintf("%s (watch)", cmd.Name)
		templateCmd.ShellCommand = buildShellCommand(cmd)
	} else if cmd.IsStop {
		templateCmd.Type = "stop"
		templateCmd.BaseName = extractBaseName(cmd.Name)
		templateCmd.HelpDescription = fmt.Sprintf("%s (stop)", cmd.Name)
		templateCmd.ShellCommand = buildShellCommand(cmd)
	} else {
		templateCmd.Type = "regular"
		templateCmd.HelpDescription = cmd.Name
		templateCmd.ShellCommand = buildShellCommand(cmd)
	}

	return templateCmd, nil
}

// sanitizeFunctionName converts command names to valid Go function names
func sanitizeFunctionName(name string) string {
	// Capitalize first letter of each word
	parts := strings.FieldsFunc(name, func(r rune) bool {
		return (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9')
	})

	var result strings.Builder
	for _, part := range parts {
		if len(part) > 0 {
			// Simple capitalize: uppercase first rune, lowercase rest
			runes := []rune(strings.ToLower(part))
			if len(runes) > 0 {
				runes[0] = unicode.ToUpper(runes[0])
			}
			result.WriteString(string(runes))
		}
	}

	funcName := result.String()
	if funcName == "" {
		funcName = "Command"
	}

	return "run" + funcName
}

// extractBaseName extracts the base name from stop commands
func extractBaseName(stopName string) string {
	// Handle "stop-service" -> "service" or "stop service" -> "service"
	if strings.HasPrefix(stopName, "stop") {
		base := strings.TrimPrefix(stopName, "stop")
		base = strings.TrimPrefix(base, "-")
		base = strings.TrimPrefix(base, "_")
		base = strings.TrimSpace(base)
		if base != "" {
			return base
		}
	}
	return stopName
}

// buildShellCommand constructs the shell command string from parser command
func buildShellCommand(cmd parser.Command) string {
	if cmd.IsBlock {
		var parts []string
		for _, stmt := range cmd.Block {
			part := stmt.Command
			if stmt.Background {
				part += " &"
			}
			parts = append(parts, part)
		}
		return strings.Join(parts, "; ")
	}
	return cmd.Command
}

// Template for generating Go CLI
const cleanGoTemplate = `package {{.PackageName}}

import (
{{range .Imports}}	"{{.}}"
{{end}})

{{if .HasProcessMgmt}}
// ProcessInfo represents a managed background process
type ProcessInfo struct {
	Name      string    ` + "`json:\"name\"`" + `
	PID       int       ` + "`json:\"pid\"`" + `
	Command   string    ` + "`json:\"command\"`" + `
	StartTime time.Time ` + "`json:\"start_time\"`" + `
	LogFile   string    ` + "`json:\"log_file\"`" + `
	Status    string    ` + "`json:\"status\"`" + `
}

// ProcessRegistry manages background processes
type ProcessRegistry struct {
	dir       string
	processes map[string]*ProcessInfo
}

// NewProcessRegistry creates a new process registry
func NewProcessRegistry() *ProcessRegistry {
	dir := ".devcmd"
	os.MkdirAll(dir, 0755)

	pr := &ProcessRegistry{
		dir:       dir,
		processes: make(map[string]*ProcessInfo),
	}
	pr.loadProcesses()
	return pr
}

// loadProcesses loads existing processes from registry file
func (pr *ProcessRegistry) loadProcesses() {
	registryFile := filepath.Join(pr.dir, "registry.json")
	data, err := os.ReadFile(registryFile)
	if err != nil {
		return // File doesn't exist or can't be read
	}

	var processes map[string]*ProcessInfo
	if err := json.Unmarshal(data, &processes); err != nil {
		return
	}

	// Verify processes are still running
	for name, proc := range processes {
		if pr.isProcessRunning(proc.PID) {
			proc.Status = "running"
			pr.processes[name] = proc
		}
	}
	pr.saveProcesses()
}

// saveProcesses saves current processes to registry file
func (pr *ProcessRegistry) saveProcesses() {
	registryFile := filepath.Join(pr.dir, "registry.json")
	data, err := json.MarshalIndent(pr.processes, "", "  ")
	if err != nil {
		return
	}
	os.WriteFile(registryFile, data, 0644)
}

// isProcessRunning checks if a process is still running
func (pr *ProcessRegistry) isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// addProcess adds a process to the registry
func (pr *ProcessRegistry) addProcess(name string, pid int, command string, logFile string) {
	pr.processes[name] = &ProcessInfo{
		Name:      name,
		PID:       pid,
		Command:   command,
		StartTime: time.Now(),
		LogFile:   logFile,
		Status:    "running",
	}
	pr.saveProcesses()
}

// removeProcess removes a process from the registry
func (pr *ProcessRegistry) removeProcess(name string) {
	delete(pr.processes, name)
	pr.saveProcesses()
}

// getProcess gets a process by name
func (pr *ProcessRegistry) getProcess(name string) (*ProcessInfo, bool) {
	proc, exists := pr.processes[name]
	return proc, exists
}

// listProcesses returns all processes
func (pr *ProcessRegistry) listProcesses() []*ProcessInfo {
	var procs []*ProcessInfo
	for _, proc := range pr.processes {
		procs = append(procs, proc)
	}
	return procs
}
{{end}}

// CLI represents the command line interface
type CLI struct {
{{if .HasProcessMgmt}}	registry *ProcessRegistry{{end}}
}

// NewCLI creates a new CLI instance
func NewCLI() *CLI {
	return &CLI{
{{if .HasProcessMgmt}}		registry: NewProcessRegistry(),{{end}}
	}
}

// Execute runs the CLI with given arguments
func (c *CLI) Execute() {
	if len(os.Args) < 2 {
		c.showHelp()
		return
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch command {
{{if .HasProcessMgmt}}	case "status":
		c.showStatus()
	case "logs":
		if len(args) == 0 {
			fmt.Fprintf(os.Stderr, "Usage: %s logs <command-name>\n", os.Args[0])
			os.Exit(1)
		}
		c.showLogs(args[0])
	case "stop":
		if len(args) == 0 {
			fmt.Fprintf(os.Stderr, "Usage: %s stop <command-name>\n", os.Args[0])
			os.Exit(1)
		}
		c.stopCommand(args[0])
{{end}}{{range .Commands}}	case "{{.GoCase}}":
		c.{{.FunctionName}}(args)
{{end}}	case "help", "--help", "-h":
		c.showHelp()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		c.showHelp()
		os.Exit(1)
	}
}

// showHelp displays available commands
func (c *CLI) showHelp() {
	fmt.Println("Available commands:")
{{if .HasProcessMgmt}}	fmt.Println("  status          - Show running background processes")
	fmt.Println("  logs <name>     - Show logs for a background process")
	fmt.Println("  stop <name>     - Stop a background process")
{{end}}{{range .Commands}}	fmt.Println("  {{.HelpDescription}}")
{{end}}}

{{if .HasProcessMgmt}}
// showStatus displays running processes
func (c *CLI) showStatus() {
	processes := c.registry.listProcesses()
	if len(processes) == 0 {
		fmt.Println("No background processes running")
		return
	}

	fmt.Printf("%-15s %-8s %-10s %-20s %s\n", "NAME", "PID", "STATUS", "STARTED", "COMMAND")
	fmt.Println(strings.Repeat("-", 80))

	for _, proc := range processes {
		// Verify process is still running
		if !c.registry.isProcessRunning(proc.PID) {
			proc.Status = "stopped"
		}

		startTime := proc.StartTime.Format("15:04:05")
		command := proc.Command
		if len(command) > 30 {
			command = command[:27] + "..."
		}

		fmt.Printf("%-15s %-8d %-10s %-20s %s\n",
			proc.Name, proc.PID, proc.Status, startTime, command)
	}
}

// showLogs displays logs for a process
func (c *CLI) showLogs(name string) {
	proc, exists := c.registry.getProcess(name)
	if !exists {
		fmt.Fprintf(os.Stderr, "No process named '%s' found\n", name)
		os.Exit(1)
	}

	if _, err := os.Stat(proc.LogFile); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Log file not found: %s\n", proc.LogFile)
		os.Exit(1)
	}

	// Stream log file
	file, err := os.Open(proc.LogFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening log file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	io.Copy(os.Stdout, file)
}

// stopCommand stops a background process
func (c *CLI) stopCommand(name string) {
	proc, exists := c.registry.getProcess(name)
	if !exists {
		fmt.Fprintf(os.Stderr, "No process named '%s' found\n", name)
		os.Exit(1)
	}

	// Try to terminate gracefully
	process, err := os.FindProcess(proc.PID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Process not found: %v\n", err)
		c.registry.removeProcess(name)
		return
	}

	fmt.Printf("Stopping process %s (PID: %d)...\n", name, proc.PID)

	// Send SIGTERM
	process.Signal(syscall.SIGTERM)

	// Wait up to 5 seconds for graceful shutdown
	timeout := time.After(5 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			// Force kill
			fmt.Printf("Force killing process %s...\n", name)
			process.Signal(syscall.SIGKILL)
			c.registry.removeProcess(name)
			return
		case <-ticker.C:
			if !c.registry.isProcessRunning(proc.PID) {
				fmt.Printf("Process %s stopped successfully\n", name)
				c.registry.removeProcess(name)
				return
			}
		}
	}
}

// runInBackground starts a command in background with logging
func (c *CLI) runInBackground(name, command string) error {
	logFile := filepath.Join(c.registry.dir, name+".log")

	// Create or truncate log file
	logFileHandle, err := os.Create(logFile)
	if err != nil {
		return fmt.Errorf("failed to create log file: %v", err)
	}

	// Start command
	cmd := exec.Command("sh", "-c", command)
	cmd.Stdout = logFileHandle
	cmd.Stderr = logFileHandle

	if err := cmd.Start(); err != nil {
		logFileHandle.Close()
		return fmt.Errorf("failed to start command: %v", err)
	}

	// Register process
	c.registry.addProcess(name, cmd.Process.Pid, command, logFile)

	fmt.Printf("Started %s in background (PID: %d)\n", name, cmd.Process.Pid)

	// Monitor process in goroutine
	go func() {
		defer logFileHandle.Close()
		cmd.Wait()
		c.registry.removeProcess(name)
	}()

	return nil
}
{{end}}

// Command implementations
{{range .Commands}}
func (c *CLI) {{.FunctionName}}(args []string) {
{{if eq .Type "watch"}}	// Watch command - run in background
	command := ` + "`{{.ShellCommand}}`" + `
	if err := c.runInBackground("{{.Name}}", command); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting {{.Name}}: %v\n", err)
		os.Exit(1)
	}
{{else if eq .Type "stop"}}	// Stop command - terminate associated processes
	baseName := "{{.BaseName}}"
{{if $.HasProcessMgmt}}	c.stopCommand(baseName){{else}}	fmt.Printf("No background process named '%s' to stop\n", baseName){{end}}

	// Also run user-defined stop commands
	cmd := exec.Command("sh", "-c", ` + "`{{.ShellCommand}}`" + `)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Stop command failed: %v\n", err)
	}
{{else}}	// Regular command
	cmd := exec.Command("sh", "-c", ` + "`{{.ShellCommand}}`" + `)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			os.Exit(exitError.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "Command failed: %v\n", err)
		os.Exit(1)
	}
{{end}}}
{{end}}

func main() {
	cli := NewCLI()
{{if .HasProcessMgmt}}
	// Handle interrupt signals gracefully
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\nShutting down...")
		os.Exit(0)
	}()
{{end}}
	cli.Execute()
}
`

// GenerateGo creates a Go CLI from a CommandFile using the new preprocessing approach
func GenerateGo(cf *parser.CommandFile) (string, error) {
	// Preprocess the command file into template-ready data
	data, err := PreprocessCommands(cf)
	if err != nil {
		return "", fmt.Errorf("failed to preprocess commands: %w", err)
	}

	// Parse and execute template
	tmpl, err := template.New("go-cli").Parse(cleanGoTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	result := buf.String()
	if len(result) == 0 {
		return "", fmt.Errorf("generated empty Go code")
	}

	return result, nil
}

// GenerateGoWithTemplate creates a Go CLI with a custom template (for testing)
func GenerateGoWithTemplate(cf *parser.CommandFile, templateStr string) (string, error) {
	if len(strings.TrimSpace(templateStr)) == 0 {
		return "", fmt.Errorf("template string cannot be empty")
	}

	// Preprocess the command file
	data, err := PreprocessCommands(cf)
	if err != nil {
		return "", fmt.Errorf("failed to preprocess commands: %w", err)
	}

	// Parse and execute custom template
	tmpl, err := template.New("custom").Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}
