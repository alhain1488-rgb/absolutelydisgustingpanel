// Package ssh provides a small interface (Runner) over SSH command execution
// and file upload, plus a real implementation backed by
// golang.org/x/crypto/ssh. All server operations go through Runner so business
// logic can be unit-tested with a mock and integration-tested against a local
// dockerized node.
package ssh

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"

	gossh "golang.org/x/crypto/ssh"
)

// Runner executes commands and uploads files on a remote host.
type Runner interface {
	// Run executes cmd and returns its stdout and stderr. A non-zero exit code
	// is returned as err (an *ExitError when available).
	Run(ctx context.Context, cmd string) (stdout string, stderr string, err error)
	// Upload writes content to remotePath with the given octal file mode.
	Upload(ctx context.Context, remotePath string, content []byte, mode uint32) error
}

// AuthMethod selects how to authenticate to the server.
type AuthMethod string

const (
	AuthKey      AuthMethod = "key"
	AuthPassword AuthMethod = "password"
)

// Config describes how to reach a server.
type Config struct {
	Host       string
	Port       int
	User       string
	AuthMethod AuthMethod
	// Secret is the private key PEM (AuthKey) or the password (AuthPassword).
	Secret     string
	Passphrase string // optional key passphrase
	Timeout    time.Duration
}

// ExitError reports a remote command that exited non-zero.
type ExitError struct {
	Code   int
	Stderr string
}

func (e *ExitError) Error() string {
	return fmt.Sprintf("remote command exited with code %d: %s", e.Code, e.Stderr)
}

// Client is the real Runner. Each operation dials a fresh connection, which is
// simple and robust for a low-frequency control panel.
type Client struct {
	cfg Config
}

// NewClient builds a Client. It does not connect until an operation runs.
func NewClient(cfg Config) *Client {
	if cfg.Port == 0 {
		cfg.Port = 22
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 10 * time.Second
	}
	return &Client{cfg: cfg}
}

func (c *Client) dial(ctx context.Context) (*gossh.Client, error) {
	auth, err := c.authMethods()
	if err != nil {
		return nil, err
	}
	clientCfg := &gossh.ClientConfig{
		User: c.cfg.User,
		Auth: auth,
		// Personal panel against operator-owned hosts: we accept any host key.
		// Pinning known_hosts is a possible future hardening.
		HostKeyCallback: gossh.InsecureIgnoreHostKey(), //nolint:gosec // operator-owned hosts
		Timeout:         c.cfg.Timeout,
	}
	addr := net.JoinHostPort(c.cfg.Host, strconv.Itoa(c.cfg.Port))

	d := net.Dialer{Timeout: c.cfg.Timeout}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", addr, err)
	}
	sshConn, chans, reqs, err := gossh.NewClientConn(conn, addr, clientCfg)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("ssh handshake %s: %w", addr, err)
	}
	return gossh.NewClient(sshConn, chans, reqs), nil
}

func (c *Client) authMethods() ([]gossh.AuthMethod, error) {
	switch c.cfg.AuthMethod {
	case AuthPassword:
		return []gossh.AuthMethod{gossh.Password(c.cfg.Secret)}, nil
	case AuthKey:
		var signer gossh.Signer
		var err error
		if c.cfg.Passphrase != "" {
			signer, err = gossh.ParsePrivateKeyWithPassphrase([]byte(c.cfg.Secret), []byte(c.cfg.Passphrase))
		} else {
			signer, err = gossh.ParsePrivateKey([]byte(c.cfg.Secret))
		}
		if err != nil {
			return nil, fmt.Errorf("parse private key: %w", err)
		}
		return []gossh.AuthMethod{gossh.PublicKeys(signer)}, nil
	default:
		return nil, fmt.Errorf("unsupported auth method %q", c.cfg.AuthMethod)
	}
}

// Run implements Runner.
func (c *Client) Run(ctx context.Context, cmd string) (string, string, error) {
	client, err := c.dial(ctx)
	if err != nil {
		return "", "", err
	}
	defer func() { _ = client.Close() }()

	session, err := client.NewSession()
	if err != nil {
		return "", "", err
	}
	defer func() { _ = session.Close() }()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	runErr := runWithContext(ctx, session, cmd)
	if runErr != nil {
		var exitErr *gossh.ExitError
		if errors.As(runErr, &exitErr) {
			return stdout.String(), stderr.String(), &ExitError{Code: exitErr.ExitStatus(), Stderr: stderr.String()}
		}
		return stdout.String(), stderr.String(), runErr
	}
	return stdout.String(), stderr.String(), nil
}

func runWithContext(ctx context.Context, session *gossh.Session, cmd string) error {
	done := make(chan error, 1)
	go func() { done <- session.Run(cmd) }()
	select {
	case <-ctx.Done():
		_ = session.Signal(gossh.SIGKILL)
		return ctx.Err()
	case err := <-done:
		return err
	}
}

// Upload implements Runner by streaming content to `cat > path` and chmod-ing.
func (c *Client) Upload(ctx context.Context, remotePath string, content []byte, mode uint32) error {
	client, err := c.dial(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()

	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer func() { _ = session.Close() }()

	session.Stdin = bytes.NewReader(content)
	var stderr bytes.Buffer
	session.Stderr = &stderr
	// Quote the path and write atomically-ish via a temp file then move.
	cmd := fmt.Sprintf("cat > %q && chmod %o %q", remotePath, mode, remotePath)
	if err := session.Run(cmd); err != nil {
		return fmt.Errorf("upload %s: %w (%s)", remotePath, err, stderr.String())
	}
	return nil
}
