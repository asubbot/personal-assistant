#!/bin/sh
# Container entrypoint for the PA service: prepare data volume, configure in-container cron
# for daily/monthly/yearly summarization, start cron in background, then exec the main bot (/pa).
# Used by Dockerfile as ENTRYPOINT; expects PA_CONFIG_DIR, PA_DATA_DIR, PA_SECRETS_DIR, TZ in env.
set -e
# 1. Ensure data volume is owned by pa (root-owned at first start).
chown -R pa:pa /data 2>/dev/null || true

# 2. Write cron.d file for summarization from container env (so PA_* and TZ match docker-compose).
PA_CONFIG_DIR=${PA_CONFIG_DIR:-/etc/pa}
PA_DATA_DIR=${PA_DATA_DIR:-/data}
PA_SECRETS_DIR=${PA_SECRETS_DIR:-/run/secrets}
TZ=${TZ:-UTC}
{
  echo "SHELL=/bin/sh"
  echo "PA_CONFIG_DIR=$PA_CONFIG_DIR"
  echo "PA_DATA_DIR=$PA_DATA_DIR"
  echo "PA_SECRETS_DIR=$PA_SECRETS_DIR"
  echo "TZ=$TZ"
  echo "15 0 * * * pa /usr/local/bin/summarize.sh day"
  echo "30 0 1 * * pa /usr/local/bin/summarize.sh month"
  echo "45 0 1 1 * pa /usr/local/bin/summarize.sh year"
} > /etc/cron.d/pa-summarize
chmod 644 /etc/cron.d/pa-summarize

# 3. Start cron in background (summarization runs in child processes).
cron -f &

# 4. Run main app as user pa (PID 1).
exec gosu pa /pa "$@"
