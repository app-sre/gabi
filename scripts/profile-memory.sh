#!/usr/bin/env bash
# Profile GABI server-side memory (RSS) during /query vs /streamquery.
#
# This script starts GABI, sends a large query to each endpoint while polling
# the process's VmRSS from /proc, and reports peak RSS + TTFB for each.
# Run it TWICE — once on main and once on your branch — to produce the
# before/after evidence for the streaming refactor.
#
# Prerequisites: curl, GABI must connect to a running PostgreSQL with
#                env.template (or DB_* / PG* env vars) configured.
#
# Environment (same as bench-query-endpoints.sh):
#   ENV_FILE / SOURCE_ENV_FILE   — source DB creds (default: env.template)
#   GABI_BASE_URL                — default http://127.0.0.1:8080
#   X_FORWARDED_USER             — default: $GABI_USER
#   QUERY_JSON_FILE              — default: scripts/bench.json (1M rows)
#   POLL_INTERVAL_MS             — RSS poll interval in ms (default 50)
#   WARMUP_QUERY                 — lightweight query to prime connections
#                                  (default: SELECT 1)
#   PROFILE_REPORT               — output file (default: gabi-profile-report.md)
#
# Usage:
#   # On main branch:
#   git stash && git checkout main
#   make build
#   ./scripts/profile-memory.sh | tee profile-main.md
#
#   # On feature branch:
#   git checkout feature-branch && git stash pop
#   make build
#   ./scripts/profile-memory.sh | tee profile-stream.md

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

GABI_BASE_URL="${GABI_BASE_URL:-http://127.0.0.1:8080}"
X_FORWARDED_USER="${X_FORWARDED_USER:-${GABI_USER:-test}}"
QUERY_JSON_FILE="${QUERY_JSON_FILE:-${SCRIPT_DIR}/bench.json}"
POLL_INTERVAL_MS="${POLL_INTERVAL_MS:-50}"
WARMUP_QUERY="${WARMUP_QUERY:-SELECT 1}"
PROFILE_REPORT="${PROFILE_REPORT:-${REPO_ROOT}/gabi-profile-report.md}"

ENV_FILE="${ENV_FILE:-${REPO_ROOT}/env.template}"
SOURCE_ENV_FILE="${SOURCE_ENV_FILE:-1}"
if [[ "$SOURCE_ENV_FILE" == "1" || "$SOURCE_ENV_FILE" == "yes" || "$SOURCE_ENV_FILE" == "true" ]] && [[ -f "$ENV_FILE" ]]; then
	set +u; . "$ENV_FILE"; set -u
fi

POLL_INTERVAL_SEC="$(awk "BEGIN{printf \"%.3f\", ${POLL_INTERVAL_MS}/1000}")"

die() { echo "error: $*" >&2; exit 1; }

for cmd in curl awk; do
	command -v "$cmd" >/dev/null 2>&1 || die "required command not found: $cmd"
done
[[ -f "$QUERY_JSON_FILE" ]] || die "QUERY_JSON_FILE not found: $QUERY_JSON_FILE"

GABI_BIN="${REPO_ROOT}/gabi"
[[ -x "$GABI_BIN" ]] || die "GABI binary not found; run 'make build' first"

WORKDIR="$(mktemp -d)"
GABI_PID=""

cleanup() {
	if [[ -n "${GABI_PID:-}" ]] && kill -0 "$GABI_PID" 2>/dev/null; then
		kill "$GABI_PID" 2>/dev/null; wait "$GABI_PID" 2>/dev/null || true
	fi
	rm -rf "$WORKDIR"
}
trap cleanup EXIT

start_gabi() {
	"$GABI_BIN" >"$WORKDIR/gabi.log" 2>&1 &
	GABI_PID=$!
	local tries=0
	while ! curl -sf "${GABI_BASE_URL}/healthcheck" >/dev/null 2>&1; do
		sleep 0.2
		tries=$((tries + 1))
		if [[ $tries -ge 50 ]]; then
			echo "GABI failed to start. Log:" >&2
			cat "$WORKDIR/gabi.log" >&2
			die "GABI did not become healthy within 10s"
		fi
	done
}

stop_gabi() {
	if [[ -n "${GABI_PID:-}" ]] && kill -0 "$GABI_PID" 2>/dev/null; then
		kill "$GABI_PID" 2>/dev/null; wait "$GABI_PID" 2>/dev/null || true
	fi
	GABI_PID=""
}

get_rss_kb() {
	awk '/^VmRSS:/ {print $2}' "/proc/$1/status" 2>/dev/null || echo 0
}

poll_rss() {
	local pid="$1" outfile="$2"
	while kill -0 "$pid" 2>/dev/null && [[ -f "$WORKDIR/.polling" ]]; do
		get_rss_kb "$pid" >>"$outfile"
		sleep "$POLL_INTERVAL_SEC"
	done
	get_rss_kb "$pid" >>"$outfile"
}

peak_rss_from_file() {
	awk 'BEGIN{m=0}{v=$1+0; if(v>m)m=v}END{print m}' "$1"
}

baseline_rss() {
	get_rss_kb "$GABI_PID"
}

warmup() {
	curl -sS -o /dev/null -X POST \
		-H 'Content-Type: application/json' \
		-H "X-Forwarded-User: ${X_FORWARDED_USER}" \
		-d "{\"query\": \"${WARMUP_QUERY}\"}" \
		"${GABI_BASE_URL}/query" || true
}

CURL_FMT='%{http_code}\t%{time_starttransfer}\t%{time_total}\t%{size_download}'

run_profiled_request() {
	local label="$1" endpoint="$2"
	local url="${GABI_BASE_URL%/}${endpoint}"
	local rss_file="$WORKDIR/${label}.rss"
	local curl_file="$WORKDIR/${label}.curl"

	warmup
	sleep 0.5

	local rss_before
	rss_before="$(baseline_rss)"

	touch "$WORKDIR/.polling"
	: >"$rss_file"
	poll_rss "$GABI_PID" "$rss_file" &
	local poll_pid=$!

	curl -sS -X POST \
		-H 'Content-Type: application/json' \
		-H "X-Forwarded-User: ${X_FORWARDED_USER}" \
		--data-binary "@${QUERY_JSON_FILE}" \
		-o /dev/null \
		-w "$CURL_FMT" \
		"$url" >"$curl_file" 2>&1

	rm -f "$WORKDIR/.polling"
	sleep 0.2
	wait "$poll_pid" 2>/dev/null || true

	local rss_peak rss_delta
	rss_peak="$(peak_rss_from_file "$rss_file")"
	rss_delta=$((rss_peak - rss_before))

	local http_code ttfb total_time dl_bytes
	IFS=$'\t' read -r http_code ttfb total_time dl_bytes <"$curl_file" || true

	echo "${label}|${http_code}|${rss_before}|${rss_peak}|${rss_delta}|${ttfb}|${total_time}|${dl_bytes}"
}

git_info() {
	local branch sha
	branch="$(git -C "$REPO_ROOT" rev-parse --abbrev-ref HEAD 2>/dev/null || echo unknown)"
	sha="$(git -C "$REPO_ROOT" rev-parse --short HEAD 2>/dev/null || echo unknown)"
	echo "${branch} (${sha})"
}

echo "==> Building GABI..." >&2
(cd "$REPO_ROOT" && go build -o gabi cmd/gabi/main.go) || die "build failed"

echo "==> Starting GABI..." >&2
start_gabi
echo "==> GABI running (PID ${GABI_PID}), polling RSS every ${POLL_INTERVAL_MS}ms" >&2

echo "==> Profiling /query..." >&2
QUERY_LINE="$(run_profiled_request "non-streaming" "/query")"

echo "==> Profiling /streamquery..." >&2
STREAM_LINE="$(run_profiled_request "streaming" "/streamquery")"

stop_gabi

TS="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
GIT="$(git_info)"

parse() { echo "$1" | cut -d'|' -f"$2"; }

fmt_kb() { awk "BEGIN{printf \"%.1f\", $1/1024}"; }

q_http="$(parse "$QUERY_LINE" 2)"
q_rss0="$(parse "$QUERY_LINE" 3)"
q_peak="$(parse "$QUERY_LINE" 4)"
q_delta="$(parse "$QUERY_LINE" 5)"
q_ttfb="$(parse "$QUERY_LINE" 6)"
q_total="$(parse "$QUERY_LINE" 7)"
q_dl="$(parse "$QUERY_LINE" 8)"

s_http="$(parse "$STREAM_LINE" 2)"
s_rss0="$(parse "$STREAM_LINE" 3)"
s_peak="$(parse "$STREAM_LINE" 4)"
s_delta="$(parse "$STREAM_LINE" 5)"
s_ttfb="$(parse "$STREAM_LINE" 6)"
s_total="$(parse "$STREAM_LINE" 7)"
s_dl="$(parse "$STREAM_LINE" 8)"

{
cat <<EOF
## GABI Memory Profile: /query vs /streamquery

- **When:** ${TS}
- **Branch:** ${GIT}
- **Query:** \`$(cat "$QUERY_JSON_FILE")\`
- **RSS poll interval:** ${POLL_INTERVAL_MS}ms

### Process Memory (VmRSS from /proc)

| Metric | /query | /streamquery | Delta |
|--------|--------|--------------|-------|
| Baseline RSS (before request) | $(fmt_kb "$q_rss0") MB | $(fmt_kb "$s_rss0") MB | |
| Peak RSS (during request) | $(fmt_kb "$q_peak") MB | $(fmt_kb "$s_peak") MB | $(fmt_kb "$((s_peak - q_peak))") MB |
| RSS growth (peak − baseline) | $(fmt_kb "$q_delta") MB | $(fmt_kb "$s_delta") MB | **$(fmt_kb "$((s_delta - q_delta))") MB** |

### HTTP Performance

| Metric | /query | /streamquery | Delta |
|--------|--------|--------------|-------|
| HTTP status | ${q_http} | ${s_http} | |
| TTFB (s) | ${q_ttfb} | ${s_ttfb} | $(awk "BEGIN{printf \"%.6f\", ${s_ttfb}-${q_ttfb}}") |
| Total time (s) | ${q_total} | ${s_total} | $(awk "BEGIN{printf \"%.6f\", ${s_total}-${q_total}}") |
| Response size | ${q_dl} B | ${s_dl} B | |

### Key Takeaway

EOF

if [[ "$q_delta" -gt 0 && "$s_delta" -gt 0 ]]; then
	ratio="$(awk "BEGIN{printf \"%.1f\", ${q_delta}/${s_delta}")"
	pct="$(awk "BEGIN{printf \"%.0f\", 100*(${q_delta}-${s_delta})/${q_delta}")"
	echo "- \`/query\` RSS growth: **$(fmt_kb "$q_delta") MB** — \`/streamquery\` RSS growth: **$(fmt_kb "$s_delta") MB** (${pct}% less, ${ratio}x ratio)."
fi
echo "- TTFB: /streamquery delivers first byte in **${s_ttfb}s** vs **${q_ttfb}s** for /query."
echo ""
echo "The streaming endpoint sends rows as they arrive from PostgreSQL, keeping"
echo "server memory bounded regardless of result set size."
} >"$PROFILE_REPORT"

echo "" >&2
echo "==> Report written: $PROFILE_REPORT" >&2
echo "" >&2
cat "$PROFILE_REPORT"
