package generator

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/aledsdavies/devcmd/pkgs/parser"
)

const goTemplate = `package main

import ({{if .HasWatchCommands}}
	"bufio"
	"context"
	"encoding/json"{{end}}
	"fmt"{{if .HasWatchCommands}}
	"io"{{end}}
	"os"
	"os/exec"{{if .HasWatchCommands}}
	"os/signal"
	"path/filepath"
	"strconv"{{end}}
	"strings"
	"syscall"{{if .HasWatchCommands}}
	"time"{{end}}
)

{{if .HasWatchCommands}}
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

// Main CLI struct
type CLI struct {
	registry *ProcessRegistry
}

// NewCLI creates a new CLI instance
func NewCLI() *CLI {
	return &CLI{
		registry: NewProcessRegistry(),
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
	case "status":
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
{{range .Commands}}
	case "{{.Name}}":
		c.run{{.Name | title}}(args)
{{end}}
	case "help", "--help", "-h":
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
	fmt.Println("  status          - Show running background processes")
	fmt.Println("  logs <name>     - Show logs for a background process")
	fmt.Println("  stop <name>     - Stop a background process")
{{range .Commands}}
	fmt.Println("  {{.Name}}{{if .IsWatch}} (watch){{else if .IsStop}} (stop){{end}}")
{{end}}
}

{{if .HasWatchCommands}}
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
func (c *CLI) run{{.Name | title}}(args []string) {
{{if .IsWatch}}
	// Watch command - run in background{{if $.HasWatchCommands}}
	command := ` + "`" + `{{if .IsBlock}}{{range .Block}}{{.Command}}{{if .Background}} &{{end}}; {{end}}{{else}}{{.Command}}{{end}}` + "`" + `
	if err := c.runInBackground("{{.Name}}", command); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting {{.Name}}: %v\n", err)
		os.Exit(1)
	}{{else}}
	fmt.Fprintf(os.Stderr, "Watch commands not supported in this build\n")
	os.Exit(1){{end}}
{{else if .IsStop}}
	// Stop command - terminate associated processes
	baseName := "{{.BaseName}}"
	if baseName == "" {
		baseName = "{{.Name}}"
	}{{if $.HasWatchCommands}}
	c.stopCommand(baseName){{else}}
	fmt.Printf("No background process named '%s' to stop\n", baseName){{end}}

	// Also run user-defined stop commands
	cmd := exec.Command("sh", "-c", ` + "`" + `{{if .IsBlock}}{{range .Block}}{{.Command}}{{if .Background}} &{{end}}; {{end}}{{else}}{{.Command}}{{end}}` + "`" + `)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Stop command failed: %v\n", err)
	}
{{else}}
	// Regular command
	cmd := exec.Command("sh", "-c", ` + "`" + `{{if .IsBlock}}{{range .Block}}{{.Command}}{{if .Background}} &{{end}}; {{end}}{{else}}{{.Command}}{{end}}` + "`" + `)
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
{{end}}
}
{{end}}

func main() {
	cli := NewCLI()
	{{if .HasWatchCommands}}
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

// GenerateGo creates a Go CLI from a CommandFile
func GenerateGo(cf *parser.CommandFile) (string, error) {
	return generateGo(cf, goTemplate)
}

// GenerateGoWithTemplate creates a Go CLI with a custom template
func GenerateGoWithTemplate(cf *parser.CommandFile, templateStr string) (string, error) {
	return generateGo(cf, templateStr)
}

// generateGo creates Go CLI with the given template
func generateGo(cf *parser.CommandFile, templateStr string) (string, error) {
	if cf == nil {
		return "", fmt.Errorf("command file cannot be nil")
	}
	if len(templateStr) == 0 {
		return "", fmt.Errorf("template string cannot be empty")
	}

	// Enhance commands with additional properties
	enhancedCommands := enhanceCommands(cf.Commands)

	// Check if there are any watch commands to determine if we need process management
	hasWatchCommands := false
	for _, cmd := range enhancedCommands {
		if cmd.IsWatch {
			hasWatchCommands = true
			break
		}
	}

	// Template functions
	funcMap := template.FuncMap{
		"title": strings.Title,
	}

	// Parse template with functions
	tmpl, err := template.New("go").Funcs(funcMap).Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	// Execute template
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, map[string]interface{}{
		"Commands":         enhancedCommands,
		"HasWatchCommands": hasWatchCommands,
	})
	if err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	result := buf.String()
	if len(result) == 0 {
		return "", fmt.Errorf("generated empty Go code")
	}

	return result, nil
}

// EnhancedCommand adds additional properties needed for template generation
// It wraps the original parser.Command with template-specific metadata
type EnhancedCommand struct {
	parser.Command                    // Embedded original command struct
	BaseName      string             // For stop commands, the base name without 'stop' prefix
	HasBackground bool               // Whether any block statements have Background=true
}

// TemplateData holds data for template execution with pre-built command list
type TemplateData struct {
	Commands    []EnhancedCommand    // All commands with enhanced metadata
	CommandList string              // Space-separated list of command names for completion
}

// enhanceCommands transforms parser commands into template-ready enhanced commands
// This function adds template-specific metadata that the generators need
func enhanceCommands(commands []parser.Command) []EnhancedCommand {
	var enhanced []EnhancedCommand

	for _, cmd := range commands {
		// Start with the base command
		enh := EnhancedCommand{
			Command: cmd,
		}

		// For stop commands, extract the base name
		// Example: "stop server" -> BaseName: "server"
		// Example: "stop-api" -> BaseName: "api"
		if cmd.IsStop {
			enh.BaseName = strings.TrimPrefix(cmd.Name, "stop")
			// Handle both "stop server" and "stop-server" formats
			if strings.HasPrefix(enh.BaseName, "-") {
				enh.BaseName = enh.BaseName[1:]
			}
			// If no suffix, use the full name
			if enh.BaseName == "" {
				enh.BaseName = cmd.Name
			}
		}

		// For block commands, check if any statements run in background
		// This helps templates decide whether to add 'wait' commands
		if cmd.IsBlock {
			for _, stmt := range cmd.Block {
				if stmt.Background {
					enh.HasBackground = true
					break
				}
			}
		}

		enhanced = append(enhanced, enh)
	}

	return enhanced
}

// buildCommandList creates a space-separated string of all command names
// Used for shell completion and help text
func buildCommandList(commands []EnhancedCommand) string {
	var cmdNames []string

	for _, cmd := range commands {
		cmdNames = append(cmdNames, cmd.Name)
	}

	// Add built-in commands
	cmdNames = append(cmdNames, "help", "status", "logs")

	return strings.Join(cmdNames, " ")
}

// Common template functions available to all generators
func getTemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"title": strings.Title,
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
		"join":  strings.Join,
		"contains": strings.Contains,
		"hasPrefix": strings.HasPrefix,
		"hasSuffix": strings.HasSuffix,
		"trimPrefix": strings.TrimPrefix,
		"trimSuffix": strings.TrimSuffix,
	}
}
