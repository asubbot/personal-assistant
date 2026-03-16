#!/bin/sh
# Copy mounted keys into each user's .ssh and chown so sshd StrictModes accepts them.
cp -r /auth/user_a/. /home/user_a/.ssh/ 2>/dev/null || true
cp -r /auth/user_b/. /home/user_b/.ssh/ 2>/dev/null || true
chown -R user_a:user_a /home/user_a/.ssh
chown -R user_b:user_b /home/user_b/.ssh
chmod 700 /home/user_a/.ssh /home/user_b/.ssh
chmod 600 /home/user_a/.ssh/authorized_keys /home/user_b/.ssh/authorized_keys 2>/dev/null || true
exec /usr/sbin/sshd -D
