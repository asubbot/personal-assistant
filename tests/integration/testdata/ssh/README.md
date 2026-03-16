# SSH integration test (Docker)

`TestSSHClient_Exec_Close_Integration` runs an SSH server in Docker for the duration of the test. No manual testdata is required.

**Requirements:** Docker daemon running, `ssh-keygen` and `ssh-keyscan` on PATH (usually with OpenSSH client).

**What the test does:** generates an ed25519 key, builds the image from this Dockerfile, runs a container with the key in `authorized_keys`, runs `ssh-keyscan` to get the host key, then connects with `pa/internal/ssh` and runs `echo ok`. The container is stopped and removed after the test.

**Ports:** the single-user test uses host port 2222; the two-user test (AC-010, `TestSSHClient_twoNodes_dedicatedUserPerNode`) uses port 2224 and the image built from `Dockerfile.twousers`. If a port is in use, the test will fail; stop the process using that port or run on another machine.

**Images:** the single-user server is Alpine (`Dockerfile`); the two-user server is **Debian** (`Dockerfile.twousers`, base `debian:bookworm-slim`) for reliable key auth and bind mounts on common Docker hosts (e.g. Colima, Rancher Desktop).
