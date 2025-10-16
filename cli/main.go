package main

import (
	"fmt"
	"io"
	"os"

	"github.com/aledsdavies/opal/runtime/executor"
	"github.com/aledsdavies/opal/runtime/lexer"
	"github.com/aledsdavies/opal/runtime/parser"
	"github.com/aledsdavies/opal/runtime/planner"
	"github.com/spf13/cobra"
)

func main() {
	var (
		file    string
		dryRun  bool
		debug   bool
		noColor bool
	)

	rootCmd := &cobra.Command{
		Use:   "opal [command]",
		Short: "Execute commands defined in opal files",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, file, dryRun, debug, noColor)
		},
	}

	// Add flags
	rootCmd.PersistentFlags().StringVarP(&file, "file", "f", "commands.opl", "Path to command definitions file")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Show execution plan without running commands")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug output")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable colored output")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runCommand(cmd *cobra.Command, args []string, file string, dryRun, debug, noColor bool) error {
	commandName := args[0]

	// Get input reader based on file options
	reader, closeFunc, err := getInputReader(file)
	if err != nil {
		return err
	}
	defer func() { _ = closeFunc() }()

	// Read source
	source, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("error reading input: %w", err)
	}

	// Lex
	tokens := lexer.Lex(source)

	// Parse
	tree := parser.Parse(source, tokens)
	if len(tree.Errors) > 0 {
		for _, parseErr := range tree.Errors {
			fmt.Fprintf(os.Stderr, "Parse error: %v\n", parseErr)
		}
		return fmt.Errorf("parse errors encountered")
	}

	// Plan
	debugLevel := planner.DebugOff
	if debug {
		debugLevel = planner.DebugDetailed
	}

	planResult, err := planner.Plan(tree.Events, tokens, planner.Config{
		Target: commandName,
		Debug:  debugLevel,
	})
	if err != nil {
		return fmt.Errorf("planning failed: %w", err)
	}

	// Dry-run mode: just show the plan
	if dryRun {
		fmt.Printf("Plan for command '%s':\n", commandName)
		fmt.Printf("  Steps: %d\n", len(planResult.Plan.Steps))
		fmt.Printf("  Planning time: %v\n", planResult.PlanTime)
		return nil
	}

	// Execute
	execDebug := executor.DebugOff
	if debug {
		execDebug = executor.DebugDetailed
	}

	result, err := executor.Execute(planResult.Plan, executor.Config{
		Debug:     execDebug,
		Telemetry: executor.TelemetryBasic,
	})
	if err != nil {
		return fmt.Errorf("execution failed: %w", err)
	}

	// Print execution summary if debug enabled
	if debug {
		fmt.Fprintf(os.Stderr, "\nExecution summary:\n")
		fmt.Fprintf(os.Stderr, "  Steps run: %d/%d\n", result.StepsRun, len(planResult.Plan.Steps))
		fmt.Fprintf(os.Stderr, "  Duration: %v\n", result.Duration)
		fmt.Fprintf(os.Stderr, "  Exit code: %d\n", result.ExitCode)
	}

	// Exit with the command's exit code
	if result.ExitCode != 0 {
		os.Exit(result.ExitCode)
	}

	return nil
}

// getInputReader handles the 3 modes of input:
// 1. Explicit stdin with -f -
// 2. Piped input (auto-detected when using default file)
// 3. File input (specific file or default commands.opl)
func getInputReader(file string) (io.Reader, func() error, error) {
	// Mode 1: Explicit stdin
	if file == "-" {
		return os.Stdin, func() error { return nil }, nil
	}

	// Mode 2: Check for piped input when using default file
	if file == "commands.opl" {
		if hasPipedInput() {
			return os.Stdin, func() error { return nil }, nil
		}
	}

	// Mode 3: File input
	f, err := os.Open(file)
	if err != nil {
		return nil, nil, fmt.Errorf("error opening file %s: %w", file, err)
	}

	closeFunc := func() error {
		return f.Close()
	}

	return f, closeFunc, nil
}

// hasPipedInput detects if there's data piped to stdin
func hasPipedInput() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}

	// Check if stdin is not a character device (i.e., it's piped)
	return (stat.Mode() & os.ModeCharDevice) == 0
}
