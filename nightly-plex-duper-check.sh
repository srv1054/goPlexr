#!/usr/bin/env bash
# Wrapper script for goPlexr to loop through all our plex users and execute the dupe report tool
#   Puts outputted HTML files into ${DEST} variable
#   v0.1.3 @srv1054 github.com/srv1054/goPlexr

set -euo pipefail

DPATH="/opt/goplexr"
DEST="/opt/plexyland/static/reports"
TIMEOUT_SECS="${TIMEOUT_SECS:-45}"   # configurable: per-server timeout
RETRY_COUNT="${RETRY_COUNT:-0}"      # configurable: how many retries after a failure (0 = none)
RETRY_DELAY_SECS="${RETRY_DELAY_SECS:-3}"

mkdir -p "$DEST"

now() { date +'%Y-%m-%d %H:%M:%S%z'; }
log() { printf '%s  %s\n' "$(now)" "$*"; }  

# Bracket IPv6; include :port only if present
build_url() {
  local ip="$1" port="${2:-}"
  if [[ -z "$ip" ]]; then
    printf ''
    return 0
  fi
  if [[ "$ip" == *:* && "$ip" != \[*\] ]]; then
    ip="[$ip]"
  fi
  if [[ -n "$port" ]]; then
    printf 'http://%s:%s' "$ip" "$port"
  else
    printf 'http://%s' "$ip"
  fi
}

SQL='
  WITH ranked AS (
    SELECT
      u.alias,
      u.apikey,
      u.ip,
      u.port,
      u.poll,
      ROW_NUMBER() OVER (PARTITION BY u.alias ORDER BY u.id DESC) AS rn
    FROM `user` u
    WHERE COALESCE(u.poll, 0) = 1
  )
  SELECT
    alias,
    COALESCE(apikey, "") AS apikey,
    COALESCE(ip, "")     AS ip,
    COALESCE(port, "")   AS port,
    COALESCE(poll, 0)    AS poll
  FROM ranked
  WHERE rn = 1;
'

log "------ Running ------"

# Runs goPlexr with a timeout and optional retries; returns 0 on success, non-zero otherwise.
run_goplexr() {
  local url="$1" token="$2" html_out="$3" json_out="$4"
  local attempt=0 rc=0

  while :; do
    # --foreground lets Ctrl-C be delivered if you run this interactively
    if timeout --foreground -k 5s "${TIMEOUT_SECS}s" \
      "${DPATH}/goPlexr" \
        -url "$url" \
        -token "$token" \
        -deep \
        -include-shows \
        -verify \
        -quiet \
        -ignore-extras \
        -html-out "$html_out" \
        -json-out "$json_out"
    then
      return 0
    fi

    rc=$?
    (( attempt++ ))
    if (( attempt > RETRY_COUNT )); then
      return "$rc"
    fi
    log "WARN: attempt ${attempt}/${RETRY_COUNT} failed for ${url} (rc=${rc}); retrying in ${RETRY_DELAY_SECS}s…"
    sleep "${RETRY_DELAY_SECS}"
  done
}

while IFS=$'\t' read -r alias apikey ip port poll; do
  url="$(build_url "$ip" "$port")"
  out="${DEST}/${alias}.html"
  ouj="${DEST}/${alias}.json"

  if [[ -z "$apikey" || -z "$url" ]]; then
    log "SKIP: \"$alias\" missing token or IP/URL (ip='${ip}', port='${port}')."
    continue
  fi

  log "Running dupe report for \"$alias\" @ ${ip}:${port}"

  # Guard the call so set -e doesn’t kill the whole loop on failure.
  if ! run_goplexr "$url" "$apikey" "$out" "$ouj"; then
    rc=$?
    case "$rc" in
      124)
        log "ERROR: \"$alias\" @ ${url} timed out after ${TIMEOUT_SECS}s. Moving on."
        ;;
      137)
        log "ERROR: \"$alias\" @ ${url} killed (SIGKILL/oom? rc=${rc}). Moving on."
        ;;
      *)
        log "ERROR: \"$alias\" @ ${url} failed (rc=${rc}). Moving on."
        ;;
    esac
    continue
  fi

  log "Completed for \"$alias\" -> ${out}"
done < <(
  /usr/bin/mysql \
    --defaults-extra-file="${DPATH}/.mysql" \
    --batch --raw --silent --skip-column-names --skip-reconnect \
    --database=plex \
    -e "$SQL"
)

log "------ Done ------"

