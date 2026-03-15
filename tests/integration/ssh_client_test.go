//go:build integration

package integration_test

import (
	"context"
	"os"
	"os/exec"
	"pa/internal/config"
	"pa/internal/ssh"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const sshTestPort = 2222

// TestSSHClient_Exec_Close_Integration runs against an SSH server in Docker: generates keys, starts container, runs test, stops container.
// Requires Docker and ssh-keygen/ssh-keyscan on PATH. Covers AC-006, ssh.Client Exec/Close.
func TestSSHClient_Exec_Close_Integration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	dir := t.TempDir()
	privateKeyPath := filepath.Join(dir, "id_ed25519")
	knownHostsPath := filepath.Join(dir, "known_hosts")
	allowlistPath := filepath.Join(dir, "allowlist.txt")

	setupSSHKeys(t, ctx, dir, privateKeyPath, allowlistPath)
	containerID := startSSHContainer(t, ctx, dir)
	defer stopSSHContainer(ctx, containerID)
	writeKnownHosts(t, ctx, knownHostsPath)

	cfg := &config.Config{
		Paths: config.Paths{SSHKnownHostsPath: knownHostsPath},
		Nodes: map[string]config.Node{
			"testnode": {
				Host:                 "127.0.0.1",
				Port:                 sshTestPort,
				DedicatedUser:        "test",
				Auth:                 config.NodeAuth{PrivateKeyPath: privateKeyPath},
				CommandAllowlistPath: allowlistPath,
			},
		},
	}

	client, err := ssh.NewClient(ctx, cfg, "testnode")
	if err != nil {
		t.Fatalf("SSH connect: %v", err)
	}
	defer func() {
		if closeErr := client.Close(); closeErr != nil {
			t.Logf("Close: %v", closeErr)
		}
	}()

	stdout, stderr, err := client.Exec(ctx, "echo ok")
	if err != nil {
		t.Fatalf("Exec: %v", err)
	}
	if want := "ok"; !strings.Contains(string(stdout), want) {
		t.Errorf("Exec stdout = %q, want to contain %q (stderr=%q)", stdout, want, stderr)
	}
}

func setupSSHKeys(t *testing.T, ctx context.Context, dir, privateKeyPath, allowlistPath string) {
	t.Helper()
	cmd := exec.CommandContext(ctx, "ssh-keygen", "-t", "ed25519", "-f", privateKeyPath, "-N", "")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("ssh-keygen: %v\n%s", err, out)
	}
	pub, err := os.ReadFile(privateKeyPath + ".pub")
	if err != nil {
		t.Fatalf("read public key: %v", err)
	}
	authKeys := filepath.Join(dir, "authorized_keys")
	if err := os.WriteFile(authKeys, pub, 0o600); err != nil {
		t.Fatalf("write authorized_keys: %v", err)
	}
	if err := os.WriteFile(allowlistPath, []byte("echo\n"), 0o600); err != nil {
		t.Fatalf("write allowlist: %v", err)
	}
}

func startSSHContainer(t *testing.T, ctx context.Context, dir string) string {
	t.Helper()
	repoRoot := findRepoRoot(t)
	dockerfile := filepath.Join(repoRoot, "tests", "integration", "testdata", "ssh", "Dockerfile")
	buildCmd := exec.CommandContext(ctx, "docker", "build", "-t", "pa-ssh-test", "-f", dockerfile, repoRoot)
	buildCmd.Dir = repoRoot
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("docker build: %v\n%s", err, out)
	}
	runCmd := exec.CommandContext(ctx, "docker", "run", "-d",
		"-p", "2222:22",
		"-v", dir+":/home/test/.ssh:ro",
		"pa-ssh-test",
	)
	runOut, err := runCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("docker run: %v\n%s", err, runOut)
	}
	return strings.TrimSpace(string(runOut))
}

func stopSSHContainer(ctx context.Context, containerID string) {
	_ = exec.CommandContext(ctx, "docker", "rm", "-f", containerID).Run()
}

func writeKnownHosts(t *testing.T, ctx context.Context, knownHostsPath string) {
	t.Helper()
	for i := 0; i < 30; i++ {
		time.Sleep(500 * time.Millisecond)
		scanCmd := exec.CommandContext(ctx, "ssh-keyscan", "-p", "2222", "-T", "2", "127.0.0.1")
		scanOut, scanErr := scanCmd.CombinedOutput()
		if scanErr == nil && len(scanOut) > 0 {
			if err := os.WriteFile(knownHostsPath, scanOut, 0o600); err != nil {
				t.Fatalf("write known_hosts: %v", err)
			}
			return
		}
		if i == 29 {
			t.Fatalf("ssh-keyscan did not get host key in time (is sshd up?)\nlast output: %s", scanOut)
		}
	}
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find repo root (go.mod)")
		}
		dir = parent
	}
}
