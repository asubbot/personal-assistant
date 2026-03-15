# SSH integration test (Docker)

`TestSSHClient_Exec_Close_Integration` runs an SSH server in Docker for the duration of the test. No manual testdata is required.

**Requirements:** Docker daemon running, `ssh-keygen` and `ssh-keyscan` on PATH (usually with OpenSSH client).

**What the test does:** generates an ed25519 key, builds the image from this Dockerfile, runs a container with the key in `authorized_keys`, runs `ssh-keyscan` to get the host key, then connects with `pa/internal/ssh` and runs `echo ok`. The container is stopped and removed after the test.

**Port:** the test uses host port 2222. If it is in use, the test will fail; use another machine or stop the process using 2222.
