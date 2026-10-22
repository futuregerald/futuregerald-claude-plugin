#!/usr/bin/env bash
if [ "${BREAKER_DISABLED:-0}" = "1" ]; then
  exit 0
fi
exec python3 "$(dirname "$0")/autonomous_breaker.py" "$@"
