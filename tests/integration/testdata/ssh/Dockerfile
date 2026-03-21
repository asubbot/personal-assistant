# Minimal SSH server for integration tests. Run with:
#   docker build -t pa-ssh-test -f tests/integration/testdata/ssh/Dockerfile .
#   docker run -d -p 2222:22 -v /path/to/authdir:/auth:ro pa-ssh-test
# authdir must contain authorized_keys (one line per public key). Mount as /auth; entrypoint copies into ~test/.ssh with chown (StrictModes vs host bind-mount UIDs).
FROM alpine:3.19
RUN apk add --no-cache openssh-server && \
    ssh-keygen -A && \
    mkdir -p /run/sshd /home/test/.ssh && \
    adduser -D test && \
    chown -R test:test /home/test && \
    echo "test:test" | chpasswd
# Allow key auth; optional password for debug
RUN sed -i 's/#PubkeyAuthentication yes/PubkeyAuthentication yes/' /etc/ssh/sshd_config && \
    sed -i 's/#AuthorizedKeysFile/AuthorizedKeysFile/' /etc/ssh/sshd_config
COPY tests/integration/testdata/ssh/entrypoint.oneuser.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh
EXPOSE 22
ENTRYPOINT ["/entrypoint.sh"]
