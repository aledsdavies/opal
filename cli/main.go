package main

import (
	"bytes"
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
	// CRITICAL: Lock down stdout/stderr at CLI entry point
	// This ensures even lexer/parser/planner cannot leak secrets
	var outputBuf bytes.Buffer
	scrubber := executor.NewSecretScrubber(&outputBuf)

	// Redirect all stdout/stderr through scrubber
	restore := executor.LockDownStdStreams(&executor.LockdownConfig{
		Scrubber: scrubber,
	})

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
			exitCode, err := runCommand(cmd, args, file, dryRun, debug, noColor, scrubber, &outputBuf)
			if err != nil {
				return err
			}
			if exitCode != 0 {
				// Store exit code for later (can't os.Exit here - skips defers)
				return fmt.Errorf("command failed with exit code %d", exitCode)
			}
			return nil
		},
	}

	// Add flags
	rootCmd.PersistentFlags().StringVarP(&file, "file", "f", "commands.opl", "Path to command definitions file")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Show execution plan without running commands")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug output")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable colored output")

	// Execute command and capture exit code
	exitCode := 0
	if err := rootCmd.Execute(); err != nil {
		// Error messages go through scrubber
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		exitCode = 1
	}

	// CRITICAL: Restore streams BEFORE writing to real stdout
	restore()

	// Now write captured (and scrubbed) output to real stdout
	os.Stdout.Write(outputBuf.Bytes())

	// Exit with proper code (after all cleanup)
	if exitCode != 0 {
		os.Exit(exitCode)
	}
}

func runCommand(cmd *cobra.Command, args []string, file string, dryRun, debug, noColor bool, scrubber *executor.SecretScrubber, outputBuf *bytes.Buffer) (int, error) {
	commandName := args[0]

	// Get input reader based on file options
	reader, closeFunc, err := getInputReader(file)
	if err != nil {
		return 1, err
	}
	defer func() { _ = closeFunc() }()

	// Read source
	source, err := io.ReadAll(reader)
	if err != nil {
		return 1, fmt.Errorf("error reading input: %w", err)
	}

	// Lex
	l := lexer.NewLexer()
	l.Init(source)
	tokens := l.GetTokens()

	// Parse
	tree := parser.Parse(source)
	if len(tree.Errors) > 0 {
		for _, parseErr := range tree.Errors {
			fmt.Fprintf(os.Stderr, "Parse error: %v\n", parseErr)
		}
		return 1, fmt.Errorf("parse errors encountered")
	}

	// Plan
	debugLevel := planner.DebugOff
	if debug {
		debugLevel = planner.DebugDetailed
	}

	plan, err := planner.Plan(tree.Events, tokens, planner.Config{
		Target: commandName,
		Debug:  debugLevel,
	})
	if err != nil {
		return 1, fmt.Errorf("planning failed: %w", err)
	}

	// Register all secrets with scrubber (ALL value decorator results are secrets)
	for _, secret := range plan.Secrets {
		// Use DisplayID as placeholder (e.g., "opal:secret:3J98t56A")
		scrubber.RegisterSecret(secret.RuntimeValue, secret.DisplayID)
	}

	// Dry-run mode: just show the plan
	if dryRun {
		fmt.Printf("Plan for command '%s':\n", commandName)
		fmt.Printf("  Steps: %d\n", len(plan.Steps))
		return 0, nil
	}

	// Execute (lockdown already active from main())
	execDebug := executor.DebugOff
	if debug {
		execDebug = executor.DebugDetailed
	}

	result, err := executor.Execute(plan, executor.Config{
		Debug:              execDebug,
		Telemetry:          executor.TelemetryBasic,
		LockdownStdStreams: false, // Already locked down at CLI level
	})
	if err != nil {
		return 1, fmt.Errorf("execution failed: %w", err)
	}

	// Print execution summary if debug enabled
	if debug {
		fmt.Fprintf(os.Stderr, "\nExecution summary:\n")
		fmt.Fprintf(os.Stderr, "  Steps run: %d/%d\n", result.StepsRun, len(plan.Steps))
		fmt.Fprintf(os.Stderr, "  Duration: %v\n", result.Duration)
		fmt.Fprintf(os.Stderr, "  Exit code: %d\n", result.ExitCode)
	}

	// Return exit code to main (don't call os.Exit - skips defers!)
	return result.ExitCode, nil
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
