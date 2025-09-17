#!/usr/bin/env bash
# Wrapper script for goPlexr to loop through plex users in the db and execute the dupe report tool
# This is unique to my environment BUT you might find useful to modify for your own awesomeness. YMMV. not a guarantee.
#   Puts outputted HTML files into ${DEST} variable
#   Expects ./goplexr binary in ${DPATH}
#   v0.1.2 @srv1054 github.com/srv1054/goPlexr/scripts

set -euo pipefail

DPATH="/opt/goplexr"
DEST="/web/reports"

mkdir -p "$DEST"

now() { date +'%Y-%m-%d %H:%M:%S%z'; }

echo "------ Running $(now) ------"

# Bracket IPv6; include :port only if present
build_url() {
  local ip="$1" port="${2:-}"
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

while IFS=$'\t' read -r alias apikey ip port poll; do
  printf '%s  Running dupe report for "%s" @ %s:%s\n' "$(now)" "$alias" "$ip" "$port"

  url="$(build_url "$ip" "$port")"
  out="${DEST}/${alias}.html"

  "${DPATH}/goPlexr" \
    -url "$url" \
    -token "$apikey" \
    -deep \
    -include-shows \
    -verify \
    -quiet \
    -html-out "$out"

  printf '%s  Completed for "%s" -> %s\n' "$(now)" "$alias" "$out"
done < <(
  /usr/bin/mysql \
    --defaults-extra-file="${DPATH}/.mysql" \
    --batch --raw --silent --skip-column-names --skip-reconnect \
    --database=plex \
    -e "$SQL"
)

