#!/bin/sh
# Wrapper for in-container cron: given day|month|year, computes the target date in TZ,
# runs /pa -summarize=<date> with up to 3 attempts and 60s pause on non-zero exit.
# Used by /etc/cron.d/pa-summarize; env (TZ, PA_CONFIG_DIR, PA_DATA_DIR, PA_SECRETS_DIR) is set by cron.
# Usage: summarize.sh day | month | year

TZ=${TZ:-UTC}
export TZ
PA_CONFIG_DIR=${PA_CONFIG_DIR:-/etc/pa}
PA_DATA_DIR=${PA_DATA_DIR:-/data}
PA_SECRETS_DIR=${PA_SECRETS_DIR:-/run/secrets}
export PA_CONFIG_DIR PA_DATA_DIR PA_SECRETS_DIR

case "$1" in
  day)   DATE=$(date +%Y-%m-%d -d "yesterday") ;;
  month) DATE=$(date +%Y-%m -d "last month") ;;
  year)  DATE=$(date +%Y -d "last year") ;;
  *)     echo "Usage: $0 day|month|year" >&2; exit 1 ;;
esac

MAX_ATTEMPTS=3
SLEEP_SEC=60
attempt=1
code=0

while [ "$attempt" -le "$MAX_ATTEMPTS" ]; do
  if /pa -summarize="$DATE"; then
    exit 0
  fi
  code=$?
  if [ "$attempt" -lt "$MAX_ATTEMPTS" ]; then
    sleep $SLEEP_SEC
  fi
  attempt=$((attempt + 1))
done
exit $code
