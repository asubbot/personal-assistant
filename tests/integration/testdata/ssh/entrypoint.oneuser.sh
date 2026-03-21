#!/bin/sh
# Copy authorized_keys from bind mount and chown for sshd StrictModes (host UID != container "test" on Linux CI).
set -e
mkdir -p /home/test/.ssh
cp /auth/authorized_keys /home/test/.ssh/authorized_keys
chown -R test:test /home/test/.ssh
chmod 700 /home/test/.ssh
chmod 600 /home/test/.ssh/authorized_keys
exec /usr/sbin/sshd -D
