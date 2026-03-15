package ssh

import (
	"context"
	"fmt"
	"net"
	"os"
	"pa/internal/config"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

const defaultSSHPort = "22"

// Client holds an SSH connection to a single node. Use only the dedicated user for that node (REQ-013).
// Call Close when done.
type Client struct {
	client *ssh.Client
	nodeID string
}

// NewClient connects to the node using credentials from config only (AC-006, REQ-004, REQ-013).
// Paths in config (e.g. private_key_path) are relative to project root (CWD at startup) when not absolute.
func NewClient(ctx context.Context, cfg *config.Config, nodeID string) (*Client, error) {
	node, ok := cfg.Nodes[nodeID]
	if !ok {
		return nil, fmt.Errorf("ssh: node %q not found in config", nodeID)
	}
	if node.DedicatedUser == "" {
		return nil, fmt.Errorf("ssh: node %q has no dedicated_user", nodeID)
	}
	if node.Auth.PrivateKeyPath == "" {
		return nil, fmt.Errorf("ssh: node %q has no auth.private_key_path", nodeID)
	}
	if cfg.Paths.SSHKnownHostsPath == "" {
		return nil, fmt.Errorf("ssh: ssh_known_hosts_path is required when using nodes")
	}

	hostKeyCallback, err := knownhosts.New(cfg.Paths.SSHKnownHostsPath)
	if err != nil {
		return nil, fmt.Errorf("ssh: known_hosts %s: %w", cfg.Paths.SSHKnownHostsPath, err)
	}

	keyPath := node.Auth.PrivateKeyPath
	keyBytes, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("ssh: read private key %s: %w", keyPath, err)
	}
	signer, err := ssh.ParsePrivateKey(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("ssh: parse private key: %w", err)
	}

	sshConfig := &ssh.ClientConfig{
		User:            node.DedicatedUser,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: hostKeyCallback,
		Timeout:         10 * time.Second,
	}

	port := defaultSSHPort
	if node.Port != 0 {
		port = fmt.Sprintf("%d", node.Port)
	}
	addr := net.JoinHostPort(node.Host, port)
	dialer := &net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("ssh: dial %s: %w", addr, err)
	}
	sshConn, chans, reqs, err := ssh.NewClientConn(conn, addr, sshConfig)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("ssh: handshake %s: %w", addr, err)
	}
	client := ssh.NewClient(sshConn, chans, reqs)
	return &Client{client: client, nodeID: nodeID}, nil
}

// Exec runs the command on the node. Command is executed exec-style (no shell on our side).
// The remote server may run it in a shell; allowlist must restrict what is allowed (enforced by caller).
func (c *Client) Exec(ctx context.Context, command string) (stdout, stderr []byte, err error) {
	session, err := c.client.NewSession()
	if err != nil {
		return nil, nil, fmt.Errorf("ssh: new session: %w", err)
	}
	defer func() { _ = session.Close() }()

	var outBuf, errBuf buffer
	session.Stdout = &outBuf
	session.Stderr = &errBuf

	// Run in goroutine so we can cancel on ctx.Done()
	done := make(chan error, 1)
	go func() {
		done <- session.Run(command)
	}()
	select {
	case <-ctx.Done():
		_ = session.Signal(ssh.SIGKILL)
		<-done
		return outBuf.Bytes(), errBuf.Bytes(), ctx.Err()
	case err := <-done:
		if err != nil {
			return outBuf.Bytes(), errBuf.Bytes(), fmt.Errorf("ssh: exec: %w", err)
		}
		return outBuf.Bytes(), errBuf.Bytes(), nil
	}
}

// Close closes the SSH connection.
func (c *Client) Close() error {
	return c.client.Close()
}

type buffer struct {
	mu  sync.Mutex
	buf []byte
}

func (b *buffer) Write(p []byte) (n int, err error) {
	b.mu.Lock()
	b.buf = append(b.buf, p...)
	b.mu.Unlock()
	return len(p), nil
}

func (b *buffer) Bytes() []byte {
	b.mu.Lock()
	out := make([]byte, len(b.buf))
	copy(out, b.buf)
	b.mu.Unlock()
	return out
}
