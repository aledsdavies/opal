package decorator

import (
	"bytes"
	"fmt"
	"io/fs"
	"net"
	"os"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// SSHSession implements Session for remote command execution over SSH.
type SSHSession struct {
	client *ssh.Client
	host   string
}

// NewSSHSession creates a new SSH session from connection parameters.
func NewSSHSession(params map[string]any) (*SSHSession, error) {
	host, ok := params["host"].(string)
	if !ok {
		return nil, fmt.Errorf("host parameter required")
	}

	user, ok := params["user"].(string)
	if !ok {
		user = os.Getenv("USER")
	}

	port, ok := params["port"].(int)
	if !ok {
		port = 22
	}

	// Create SSH client config
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			sshAgentAuth(),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: Proper host key verification
	}

	// Connect
	addr := fmt.Sprintf("%s:%d", host, port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("ssh dial failed: %w", err)
	}

	return &SSHSession{
		client: client,
		host:   host,
	}, nil
}

// Run executes a command on the remote host.
func (s *SSHSession) Run(argv []string, opts RunOpts) (Result, error) {
	session, err := s.client.NewSession()
	if err != nil {
		return Result{}, fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	// Build command string
	cmd := shellEscape(argv)

	// Wire up I/O
	if opts.Stdin != nil {
		session.Stdin = bytes.NewReader(opts.Stdin)
	}

	var stdout, stderr bytes.Buffer
	if opts.Stdout != nil {
		session.Stdout = opts.Stdout
	} else {
		session.Stdout = &stdout
	}
	if opts.Stderr != nil {
		session.Stderr = opts.Stderr
	} else {
		session.Stderr = &stderr
	}

	// Execute
	err = session.Run(cmd)
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*ssh.ExitError); ok {
			exitCode = exitErr.ExitStatus()
		} else {
			exitCode = 1
		}
	}

	return Result{
		ExitCode: exitCode,
		Stdout:   stdout.Bytes(),
		Stderr:   stderr.Bytes(),
	}, nil
}

// Put writes data to a file on the remote host.
func (s *SSHSession) Put(data []byte, path string, mode fs.FileMode) error {
	session, err := s.client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	// Use cat to write file
	cmd := fmt.Sprintf("cat > %s && chmod %o %s", shellQuote(path), mode, shellQuote(path))
	session.Stdin = bytes.NewReader(data)

	return session.Run(cmd)
}

// Get reads data from a file on the remote host.
func (s *SSHSession) Get(path string) ([]byte, error) {
	session, err := s.client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	var stdout bytes.Buffer
	session.Stdout = &stdout

	cmd := fmt.Sprintf("cat %s", shellQuote(path))
	if err := session.Run(cmd); err != nil {
		return nil, err
	}

	return stdout.Bytes(), nil
}

// Env returns the remote environment variables.
func (s *SSHSession) Env() map[string]string {
	session, err := s.client.NewSession()
	if err != nil {
		return make(map[string]string)
	}
	defer session.Close()

	var stdout bytes.Buffer
	session.Stdout = &stdout

	// Run env command
	if err := session.Run("env"); err != nil {
		return make(map[string]string)
	}

	// Parse env output
	return parseEnv(stdout.String())
}

// WithEnv returns a new Session with environment delta applied.
// For SSH, this creates a wrapper that sets env vars before commands.
func (s *SSHSession) WithEnv(delta map[string]string) Session {
	return &SSHSessionWithEnv{
		base:  s,
		delta: delta,
	}
}

// Cwd returns the current working directory on the remote host.
func (s *SSHSession) Cwd() string {
	session, err := s.client.NewSession()
	if err != nil {
		return ""
	}
	defer session.Close()

	var stdout bytes.Buffer
	session.Stdout = &stdout

	if err := session.Run("pwd"); err != nil {
		return ""
	}

	return strings.TrimSpace(stdout.String())
}

// Close closes the SSH connection.
func (s *SSHSession) Close() error {
	return s.client.Close()
}

// SSHSessionWithEnv wraps SSHSession to inject environment variables.
type SSHSessionWithEnv struct {
	base  *SSHSession
	delta map[string]string
}

func (s *SSHSessionWithEnv) Run(argv []string, opts RunOpts) (Result, error) {
	// Prepend env vars to command
	var envPrefix []string
	for k, v := range s.delta {
		envPrefix = append(envPrefix, fmt.Sprintf("%s=%s", k, shellQuote(v)))
	}

	// Build command: VAR1=val1 VAR2=val2 original_command
	cmd := append(envPrefix, argv...)
	return s.base.Run(cmd, opts)
}

func (s *SSHSessionWithEnv) Put(data []byte, path string, mode fs.FileMode) error {
	return s.base.Put(data, path, mode)
}

func (s *SSHSessionWithEnv) Get(path string) ([]byte, error) {
	return s.base.Get(path)
}

func (s *SSHSessionWithEnv) Env() map[string]string {
	// Merge base env with delta
	env := s.base.Env()
	for k, v := range s.delta {
		env[k] = v
	}
	return env
}

func (s *SSHSessionWithEnv) WithEnv(delta map[string]string) Session {
	// Merge deltas
	merged := make(map[string]string)
	for k, v := range s.delta {
		merged[k] = v
	}
	for k, v := range delta {
		merged[k] = v
	}
	return &SSHSessionWithEnv{
		base:  s.base,
		delta: merged,
	}
}

func (s *SSHSessionWithEnv) Cwd() string {
	return s.base.Cwd()
}

func (s *SSHSessionWithEnv) Close() error {
	return s.base.Close()
}

// SSHTransport implements Transport for SSH connections.
type SSHTransport struct{}

func (t *SSHTransport) Descriptor() Descriptor {
	return Descriptor{
		Path: "ssh",
	}
}

func (t *SSHTransport) Open(parent Session, params map[string]any) (Session, error) {
	return NewSSHSession(params)
}

func (t *SSHTransport) Wrap(next ExecNode, params map[string]any) ExecNode {
	// TODO: Implement execution wrapping
	return next
}

// Helper functions

func sshAgentAuth() ssh.AuthMethod {
	// Connect to SSH agent
	socket := os.Getenv("SSH_AUTH_SOCK")
	if socket == "" {
		return nil
	}

	conn, err := net.Dial("unix", socket)
	if err != nil {
		return nil
	}

	agentClient := agent.NewClient(conn)
	return ssh.PublicKeysCallback(agentClient.Signers)
}

func parseEnv(output string) map[string]string {
	env := make(map[string]string)
	for _, line := range strings.Split(output, "\n") {
		if idx := strings.IndexByte(line, '='); idx > 0 {
			env[line[:idx]] = line[idx+1:]
		}
	}
	return env
}

func shellEscape(argv []string) string {
	escaped := make([]string, len(argv))
	for i, arg := range argv {
		escaped[i] = shellQuote(arg)
	}
	return strings.Join(escaped, " ")
}

func shellQuote(s string) string {
	// Simple quoting - wrap in single quotes and escape single quotes
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
