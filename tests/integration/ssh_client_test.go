//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"pa/internal/config"
	"pa/internal/ssh"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const (
	sshTestPort         = 2222
	sshTestPortTwoUsers = 2224 // avoid conflict with single-user test (2222) or other use (2223)
)

// integrationTmpDir returns a subdir under tests/integration/tmp (under repo root so bind mounts work in Colima/Lima).
// Cleans the subdir first so ssh-keygen and other steps see an empty dir on every run.
func integrationTmpDir(t *testing.T, subdir string) string {
	t.Helper()
	root := findRepoRoot(t)
	dir := filepath.Join(root, "tests", "integration", "tmp", subdir)
	if err := os.RemoveAll(dir); err != nil {
		t.Logf("RemoveAll %s: %v", dir, err)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("MkdirAll %s: %v", dir, err)
	}
	return dir
}

// TestSSHClient_Exec_Close_Integration runs against an SSH server in Docker: generates keys, starts container, runs test, stops container.
// Requires Docker and ssh-keygen/ssh-keyscan on PATH. Covers AC-01.006, ssh.Client Exec/Close.
func TestSSHClient_Exec_Close_Integration(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	dir := integrationTmpDir(t, "ssh-oneuser")
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

// Covers AC-01.010 (US-04): two nodes, same host — each connection uses dedicated user (whoami per node).
// Requires Docker and the two-user SSH image (Dockerfile.twousers).
func TestSSHClient_twoNodes_dedicatedUserPerNode(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	dir := integrationTmpDir(t, "ssh-twousers")
	knownHostsPath := filepath.Join(dir, "known_hosts")
	setupSSHKeysTwoUsers(t, ctx, dir)
	containerID := startSSHContainerTwoUsers(t, ctx, dir)
	defer stopSSHContainer(ctx, containerID)
	writeKnownHostsForPort(t, ctx, knownHostsPath, sshTestPortTwoUsers)

	allowlistPath := filepath.Join(dir, "allowlist.txt")
	keyPathA := filepath.Join(dir, "id_ed25519_a")
	keyPathB := filepath.Join(dir, "id_ed25519_b")
	cfg := &config.Config{
		Paths: config.Paths{SSHKnownHostsPath: knownHostsPath},
		Nodes: map[string]config.Node{
			"node_a": {
				Host:                 "127.0.0.1",
				Port:                 sshTestPortTwoUsers,
				DedicatedUser:        "user_a",
				Auth:                 config.NodeAuth{PrivateKeyPath: keyPathA},
				CommandAllowlistPath: allowlistPath,
			},
			"node_b": {
				Host:                 "127.0.0.1",
				Port:                 sshTestPortTwoUsers,
				DedicatedUser:        "user_b",
				Auth:                 config.NodeAuth{PrivateKeyPath: keyPathB},
				CommandAllowlistPath: allowlistPath,
			},
		},
	}

	if _, err := os.Stat(keyPathA); err != nil {
		t.Fatalf("key file A missing: %v", err)
	}
	if _, err := os.Stat(keyPathB); err != nil {
		t.Fatalf("key file B missing: %v", err)
	}
	clientA, err := ssh.NewClient(ctx, cfg, "node_a")
	if err != nil {
		t.Fatalf("SSH connect node_a: %v", err)
	}
	defer func() {
		if closeErr := clientA.Close(); closeErr != nil {
			t.Logf("clientA.Close: %v", closeErr)
		}
	}()
	outA, _, err := clientA.Exec(ctx, "whoami")
	if err != nil {
		t.Fatalf("Exec whoami node_a: %v", err)
	}
	if want := "user_a"; !strings.Contains(string(outA), want) {
		t.Errorf("node_a whoami = %q, want to contain %q", outA, want)
	}

	clientB, err := ssh.NewClient(ctx, cfg, "node_b")
	if err != nil {
		t.Fatalf("SSH connect node_b: %v", err)
	}
	defer func() {
		if closeErr := clientB.Close(); closeErr != nil {
			t.Logf("clientB.Close: %v", closeErr)
		}
	}()
	outB, _, err := clientB.Exec(ctx, "whoami")
	if err != nil {
		t.Fatalf("Exec whoami node_b: %v", err)
	}
	if want := "user_b"; !strings.Contains(string(outB), want) {
		t.Errorf("node_b whoami = %q, want to contain %q", outB, want)
	}
}

func setupSSHKeysTwoUsers(t *testing.T, ctx context.Context, dir string) {
	t.Helper()
	keyA := filepath.Join(dir, "id_ed25519_a")
	keyB := filepath.Join(dir, "id_ed25519_b")
	for _, keyPath := range []string{keyA, keyB} {
		cmd := exec.CommandContext(ctx, "ssh-keygen", "-t", "ed25519", "-f", keyPath, "-N", "")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("ssh-keygen %s: %v\n%s", keyPath, err, out)
		}
	}
	pubA, err := os.ReadFile(keyA + ".pub")
	if err != nil {
		t.Fatalf("read pub A: %v", err)
	}
	pubB, err := os.ReadFile(keyB + ".pub")
	if err != nil {
		t.Fatalf("read pub B: %v", err)
	}
	userADir := filepath.Join(dir, "user_a")
	userBDir := filepath.Join(dir, "user_b")
	if err := os.MkdirAll(userADir, 0o700); err != nil {
		t.Fatalf("mkdir user_a: %v", err)
	}
	if err := os.MkdirAll(userBDir, 0o700); err != nil {
		t.Fatalf("mkdir user_b: %v", err)
	}
	if err := os.WriteFile(filepath.Join(userADir, "authorized_keys"), pubA, 0o600); err != nil {
		t.Fatalf("write user_a authorized_keys: %v", err)
	}
	if err := os.WriteFile(filepath.Join(userBDir, "authorized_keys"), pubB, 0o600); err != nil {
		t.Fatalf("write user_b authorized_keys: %v", err)
	}
	allowlistPath := filepath.Join(dir, "allowlist.txt")
	if err := os.WriteFile(allowlistPath, []byte("whoami\n"), 0o600); err != nil {
		t.Fatalf("write allowlist: %v", err)
	}
}

func startSSHContainerTwoUsers(t *testing.T, ctx context.Context, dir string) string {
	t.Helper()
	stopContainersUsingPort(t, ctx, sshTestPortTwoUsers)
	repoRoot := findRepoRoot(t)
	dockerfile := filepath.Join(repoRoot, "tests", "integration", "testdata", "ssh", "Dockerfile.twousers")
	buildCmd := exec.CommandContext(ctx, "docker", "build", "-t", "pa-ssh-test-twousers", "-f", dockerfile, repoRoot)
	buildCmd.Dir = repoRoot
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("docker build twousers: %v\n%s", err, out)
	}
	userA := filepath.Join(dir, "user_a")
	userB := filepath.Join(dir, "user_b")
	runCmd := exec.CommandContext(ctx, "docker", "run", "-d",
		"-p", fmt.Sprintf("%d:22", sshTestPortTwoUsers),
		"-v", userA+":/auth/user_a:ro",
		"-v", userB+":/auth/user_b:ro",
		"pa-ssh-test-twousers",
	)
	runOut, err := runCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("docker run twousers: %v\n%s", err, runOut)
	}
	return strings.TrimSpace(string(runOut))
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

// stopContainersUsingPort removes any running container that publishes the given host port (e.g. leftover from a previous test run).
func stopContainersUsingPort(t *testing.T, ctx context.Context, port int) {
	t.Helper()
	portStr := fmt.Sprintf("%d", port)
	listCmd := exec.CommandContext(ctx, "docker", "ps", "-q", "--filter", "publish="+portStr)
	out, err := listCmd.CombinedOutput()
	if err != nil {
		t.Logf("docker ps (port %s): %v", portStr, err)
		return
	}
	ids := strings.TrimSpace(string(out))
	if ids == "" {
		return
	}
	for _, id := range strings.Split(ids, "\n") {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		rmCmd := exec.CommandContext(ctx, "docker", "rm", "-f", id)
		if out, err := rmCmd.CombinedOutput(); err != nil {
			t.Logf("docker rm -f %s: %v %s", id, err, out)
		}
	}
}

func startSSHContainer(t *testing.T, ctx context.Context, dir string) string {
	t.Helper()
	stopContainersUsingPort(t, ctx, sshTestPort)
	repoRoot := findRepoRoot(t)
	dockerfile := filepath.Join(repoRoot, "tests", "integration", "testdata", "ssh", "Dockerfile")
	buildCmd := exec.CommandContext(ctx, "docker", "build", "-t", "pa-ssh-test", "-f", dockerfile, repoRoot)
	buildCmd.Dir = repoRoot
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("docker build: %v\n%s", err, out)
	}
	runCmd := exec.CommandContext(ctx, "docker", "run", "-d",
		"-p", fmt.Sprintf("%d:22", sshTestPort),
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
	writeKnownHostsForPort(t, ctx, knownHostsPath, sshTestPort)
}

// writeKnownHostsForPort runs ssh-keyscan for the given port and writes known_hosts. port 0 means default test port 2222.
func writeKnownHostsForPort(t *testing.T, ctx context.Context, knownHostsPath string, port int) {
	t.Helper()
	portStr := "2222"
	if port != 0 {
		portStr = fmt.Sprintf("%d", port)
	}
	for i := 0; i < 30; i++ {
		time.Sleep(500 * time.Millisecond)
		scanCmd := exec.CommandContext(ctx, "ssh-keyscan", "-p", portStr, "-T", "2", "127.0.0.1")
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
