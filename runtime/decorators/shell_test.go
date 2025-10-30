package decorators

import (
	"bytes"
	"context"
	"io"
	"os"
	"time"
)

import (
	"testing"

	"github.com/aledsdavies/opal/core/decorator"
)

// TestShellDecorator_NewArch_SimpleCommand tests basic command execution with new architecture
func TestShellDecorator_NewArch_SimpleCommand(t *testing.T) {
	// Create decorator instance
	shell := &ShellDecorator{}

	// Verify descriptor
	desc := shell.Descriptor()
	if desc.Path != "shell" {
		t.Errorf("expected path 'shell', got %q", desc.Path)
	}

	// Create execution node
	params := map[string]any{
		"command": "echo hello",
	}
	node := shell.Wrap(nil, params)

	// Create execution context with local session
	session := decorator.NewLocalSession()
	defer session.Close()

	ctx := decorator.ExecContext{
		Session: session,
		Context: context.Background(), // No deadline

		Trace: nil, // No tracing for tests
	}

	// Execute
	result, err := node.Execute(ctx)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got: %d", result.ExitCode)
	}
}

// TestShellDecorator_NewArch_FailingCommand tests non-zero exit codes
func TestShellDecorator_NewArch_FailingCommand(t *testing.T) {
	shell := &ShellDecorator{}

	params := map[string]any{
		"command": "exit 42",
	}
	node := shell.Wrap(nil, params)

	session := decorator.NewLocalSession()
	defer session.Close()

	ctx := decorator.ExecContext{
		Session: session,
		Context: context.Background(),

		Trace: nil, // No tracing for tests
	}

	result, err := node.Execute(ctx)
	// Exit code should be 42, no error
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
	if result.ExitCode != 42 {
		t.Errorf("expected exit code 42, got: %d", result.ExitCode)
	}
}

// TestShellDecorator_NewArch_MissingCommandArg tests error when command arg is missing
func TestShellDecorator_NewArch_MissingCommandArg(t *testing.T) {
	shell := &ShellDecorator{}

	params := map[string]any{} // No command param
	node := shell.Wrap(nil, params)

	session := decorator.NewLocalSession()
	defer session.Close()

	ctx := decorator.ExecContext{
		Session: session,
		Context: context.Background(),

		Trace: nil, // No tracing for tests
	}

	result, err := node.Execute(ctx)
	// Should return error
	if err == nil {
		t.Error("expected error for missing command arg, got nil")
	}
	if result.ExitCode != 127 {
		t.Errorf("expected exit code 127 for missing command, got: %d", result.ExitCode)
	}
}

// TestShellDecorator_NewArch_UsesSessionWorkdir tests that session workdir is used
func TestShellDecorator_NewArch_UsesSessionWorkdir(t *testing.T) {
	shell := &ShellDecorator{}

	// Create temp directory
	tmpDir := t.TempDir()

	params := map[string]any{
		"command": "pwd",
	}
	node := shell.Wrap(nil, params)

	// Create session with custom workdir
	session := decorator.NewLocalSession().WithWorkdir(tmpDir)
	defer session.Close()

	ctx := decorator.ExecContext{
		Session: session,
		Context: context.Background(),

		Trace: nil, // No tracing for tests
	}

	result, err := node.Execute(ctx)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got: %d", result.ExitCode)
	}

	// Verify output contains tmpDir
	output := string(result.Stdout)
	if output != tmpDir+"\n" {
		t.Errorf("expected pwd output %q, got %q", tmpDir+"\n", output)
	}
}

// TestShellDecorator_NewArch_UsesSessionEnviron tests that session environ is used
func TestShellDecorator_NewArch_UsesSessionEnviron(t *testing.T) {
	shell := &ShellDecorator{}

	params := map[string]any{
		"command": "echo $TEST_SHELL_VAR",
	}
	node := shell.Wrap(nil, params)

	// Create session with custom env
	session := decorator.NewLocalSession().WithEnv(map[string]string{
		"TEST_SHELL_VAR": "from_session",
	})
	defer session.Close()

	ctx := decorator.ExecContext{
		Session: session,
		Context: context.Background(),

		Trace: nil, // No tracing for tests
	}

	result, err := node.Execute(ctx)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got: %d", result.ExitCode)
	}

	// Verify output contains env var value
	output := string(result.Stdout)
	if output != "from_session\n" {
		t.Errorf("expected output 'from_session\\n', got %q", output)
	}
}

// TestShellDecorator_NewArch_Registered tests that @shell is registered in new registry
func TestShellDecorator_NewArch_Registered(t *testing.T) {
	// Verify @shell is registered in new registry
	entry, exists := decorator.Global().Lookup("shell")
	if !exists {
		t.Fatal("@shell should be registered in new registry")
	}

	// Verify it implements Exec interface
	_, ok := entry.Impl.(decorator.Exec)
	if !ok {
		t.Error("@shell should implement Exec interface")
	}

	// Verify descriptor
	desc := entry.Impl.Descriptor()
	if desc.Path != "shell" {
		t.Errorf("expected path 'shell', got %q", desc.Path)
	}
}

// TestShellDecorator_NewArch_Timeout tests deadline enforcement
func TestShellDecorator_NewArch_Timeout(t *testing.T) {
	shell := &ShellDecorator{}

	params := map[string]any{
		"command": "sleep 5", // Long-running command
	}
	node := shell.Wrap(nil, params)

	session := decorator.NewLocalSession()
	defer session.Close()

	// Create context with very short deadline (100ms)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	shellCtx := decorator.ExecContext{
		Session: session,
		Context: ctx,
		Trace:   nil,
	}

	// Execute should fail due to timeout
	result, err := node.Execute(shellCtx)
	if err == nil {
		t.Error("expected error due to timeout, got nil")
	}
	// Exit code should be -1 (canceled) when context deadline exceeded
	if result.ExitCode != decorator.ExitCanceled {
		t.Errorf("expected exit code %d (canceled), got: %d", decorator.ExitCanceled, result.ExitCode)
	}
}

// TestShellDecorator_NewArch_WithPipedStdin verifies @shell reads from piped stdin
func TestShellDecorator_NewArch_WithPipedStdin(t *testing.T) {
	shell := &ShellDecorator{}

	params := map[string]any{
		"command": "grep hello",
	}
	node := shell.Wrap(nil, params)

	session := decorator.NewLocalSession()
	defer session.Close()

	// Provide stdin data
	stdinData := []byte("hello world")

	ctx := decorator.ExecContext{
		Session: session,
		Context: context.Background(),

		Stdin:  bytes.NewReader(stdinData), // Piped input
		Stdout: nil,
		Trace:  nil,
	}

	result, err := node.Execute(ctx)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0 (grep finds 'hello'), got: %d", result.ExitCode)
	}
}

// TestShellDecorator_NewArch_WithPipedStdout verifies @shell writes to piped stdout
func TestShellDecorator_NewArch_WithPipedStdout(t *testing.T) {
	shell := &ShellDecorator{}

	params := map[string]any{
		"command": "echo test",
	}
	node := shell.Wrap(nil, params)

	session := decorator.NewLocalSession()
	defer session.Close()

	// Capture stdout
	var stdout bytes.Buffer

	ctx := decorator.ExecContext{
		Session: session,
		Context: context.Background(),

		Stdin:  nil,
		Stdout: &stdout, // Piped output
		Trace:  nil,
	}

	result, err := node.Execute(ctx)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got: %d", result.ExitCode)
	}
	if stdout.String() != "test\n" {
		t.Errorf("expected stdout 'test\\n', got: %q", stdout.String())
	}
}

// TestShellDecorator_NewArch_WithBothPipes verifies @shell works with both stdin and stdout piped
func TestShellDecorator_NewArch_WithBothPipes(t *testing.T) {
	shell := &ShellDecorator{}

	params := map[string]any{
		"command": "grep hello",
	}
	node := shell.Wrap(nil, params)

	session := decorator.NewLocalSession()
	defer session.Close()

	// Provide stdin and capture stdout
	stdinData := []byte("hello world")
	var stdout bytes.Buffer

	ctx := decorator.ExecContext{
		Session: session,
		Context: context.Background(),

		Stdin:  bytes.NewReader(stdinData), // Piped input
		Stdout: &stdout,                    // Piped output
		Trace:  nil,
	}

	result, err := node.Execute(ctx)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got: %d", result.ExitCode)
	}
	if stdout.String() != "hello world\n" {
		t.Errorf("expected stdout 'hello world\\n', got: %q", stdout.String())
	}
}

// TestShellDecorator_NewArch_PipedStdinNoMatch verifies grep fails when no match
func TestShellDecorator_NewArch_PipedStdinNoMatch(t *testing.T) {
	shell := &ShellDecorator{}

	params := map[string]any{
		"command": "grep nomatch",
	}
	node := shell.Wrap(nil, params)

	session := decorator.NewLocalSession()
	defer session.Close()

	// Provide stdin data that won't match
	stdinData := []byte("hello world")

	ctx := decorator.ExecContext{
		Session: session,
		Context: context.Background(),

		Stdin:  bytes.NewReader(stdinData),
		Stdout: nil,
		Trace:  nil,
	}

	result, err := node.Execute(ctx)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
	if result.ExitCode != 1 {
		t.Errorf("expected exit code 1 (grep no match), got: %d", result.ExitCode)
	}
}

// TestShellDecorator_NewArch_EndpointWrite tests @shell as file write endpoint
func TestShellDecorator_NewArch_EndpointWrite(t *testing.T) {
	// Create temp file path
	tmpFile := t.TempDir() + "/test_output.txt"

	// Create decorator instance with params
	shell := &ShellDecorator{
		params: map[string]any{
			"command": tmpFile,
		},
	}

	session := decorator.NewLocalSession()
	defer session.Close()

	ctx := decorator.ExecContext{
		Session: session,
		Context: context.Background(),
		Trace:   nil,
	}

	// Open as write endpoint
	writer, err := shell.Open(ctx, decorator.IOWrite)
	if err != nil {
		t.Fatalf("expected no error opening endpoint, got: %v", err)
	}
	defer writer.Close()

	// Write data
	data := []byte("test data\n")
	n, err := writer.Write(data)
	if err != nil {
		t.Errorf("expected no error writing, got: %v", err)
	}
	if n != len(data) {
		t.Errorf("expected to write %d bytes, wrote %d", len(data), n)
	}

	// Close to flush
	if err := writer.Close(); err != nil {
		t.Errorf("expected no error closing, got: %v", err)
	}

	// Verify file contents
	content, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("expected file to exist: %v", err)
	}
	if string(content) != "test data\n" {
		t.Errorf("expected file content 'test data\\n', got %q", string(content))
	}
}

// TestShellDecorator_NewArch_EndpointRead tests @shell as file read endpoint
func TestShellDecorator_NewArch_EndpointRead(t *testing.T) {
	// Create temp file with content
	tmpFile := t.TempDir() + "/test_input.txt"
	if err := os.WriteFile(tmpFile, []byte("input data\n"), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create decorator instance with params
	shell := &ShellDecorator{
		params: map[string]any{
			"command": tmpFile,
		},
	}

	session := decorator.NewLocalSession()
	defer session.Close()

	ctx := decorator.ExecContext{
		Session: session,
		Context: context.Background(),
		Trace:   nil,
	}

	// Open as read endpoint
	reader, err := shell.Open(ctx, decorator.IORead)
	if err != nil {
		t.Fatalf("expected no error opening endpoint, got: %v", err)
	}
	defer reader.Close()

	// Read data
	data := make([]byte, 100)
	n, err := reader.Read(data)
	if err != nil && err != io.EOF {
		t.Errorf("expected no error reading, got: %v", err)
	}
	if string(data[:n]) != "input data\n" {
		t.Errorf("expected to read 'input data\\n', got %q", string(data[:n]))
	}
}

// TestShellDecorator_NewArch_MultiRole tests that @shell implements both Exec and Endpoint
func TestShellDecorator_NewArch_MultiRole(t *testing.T) {
	shell := &ShellDecorator{}

	// Verify it implements Exec
	_, ok := interface{}(shell).(decorator.Exec)
	if !ok {
		t.Error("@shell should implement Exec interface")
	}

	// Verify it implements Endpoint
	_, ok = interface{}(shell).(decorator.Endpoint)
	if !ok {
		t.Error("@shell should implement Endpoint interface")
	}

	// Verify descriptor shows both roles
	desc := shell.Descriptor()
	hasExec := false
	hasEndpoint := false
	for _, role := range desc.Roles {
		if role == decorator.RoleWrapper {
			hasExec = true
		}
		if role == decorator.RoleEndpoint {
			hasEndpoint = true
		}
	}
	if !hasExec {
		t.Error("@shell descriptor should include RoleWrapper")
	}
	if !hasEndpoint {
		t.Error("@shell descriptor should include RoleEndpoint")
	}
}

// TestShellDecorator_NewArch_StreamingPipe tests that stdin streams without buffering
// This reproduces the issue: yes | head -n1 should complete quickly, not hang
func TestShellDecorator_NewArch_StreamingPipe(t *testing.T) {
	shell := &ShellDecorator{}

	params := map[string]any{
		"command": "head -n1", // Read only 1 line from stdin
	}
	node := shell.Wrap(nil, params)

	session := decorator.NewLocalSession()
	defer session.Close()

	// Create a pipe that produces infinite output (simulates "yes")
	pr, pw := io.Pipe()
	defer pr.Close()

	// Producer goroutine: write infinite "y\n" lines
	go func() {
		defer pw.Close()
		for i := 0; i < 1000000; i++ { // Large but not truly infinite for test safety
			if _, err := pw.Write([]byte("y\n")); err != nil {
				return // Pipe closed, stop producing
			}
		}
	}()

	// Convert pipe reader to bytes for current interface
	// This is where the bug is - we read ALL data before execution
	stdinData, err := io.ReadAll(pr)
	if err != nil {
		t.Fatalf("failed to read stdin: %v", err)
	}

	ctx := decorator.ExecContext{
		Session: session,
		Context: context.Background(),
		Stdin:   bytes.NewReader(stdinData), // This will hang trying to read all 1M lines
		Trace:   nil,
	}

	// Set a timeout to detect the hang
	done := make(chan bool)
	go func() {
		result, err := node.Execute(ctx)
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
		if result.ExitCode != 0 {
			t.Errorf("expected exit code 0, got: %d", result.ExitCode)
		}
		// Should only read 1 line
		if string(result.Stdout) != "y\n" {
			t.Errorf("expected output 'y\\n', got %q", string(result.Stdout))
		}
		done <- true
	}()

	// Should complete in <100ms with streaming, but will hang with buffering
	select {
	case <-done:
		// Success - streaming worked
	case <-time.After(2 * time.Second):
		t.Fatal("Test timed out - stdin is being buffered instead of streamed")
	}
}
