#!/usr/bin/env bash
# Compare POST /query vs POST /streamquery using a large PostgreSQL result set.
#
# - curl %{time_starttransfer} = time-to-first-byte (TTFB) at the HTTP client.
# - Memory / buffer / WAL signals come ONLY from PostgreSQL catalogs (psql), not OS/proc.
#
# Prerequisites: curl, awk, psql
# PG14+: includes pg_stat_wal. Earlier: WAL columns shown as empty / n/a.
#
# Environment:
#   GABI_BASE_URL          default http://127.0.0.1:8080
#   X_FORWARDED_USER       GABI auth header
#   QUERY_JSON_FILE        default: scripts/bench.json (1M-row query)
#   RUNS                   default 10
#   PSQL                   default psql
#   ENV_FILE               default: repo root env.template (same as GABI local config)
#   SOURCE_ENV_FILE        default 1; set 0 to skip sourcing ENV_FILE
#   DB_HOST DB_PORT DB_USER DB_PASS DB_NAME  (GABI / env.template — mapped to libpq for psql)
#   DB_SSLMODE             optional; copied to PGSSLMODE if PGSSLMODE unset (e.g. require for RDS)
#   PGHOST PGPORT PGUSER PGPASSWORD PGDATABASE PGSSLMODE  (libpq; override DB_* when set)
#   CURL_MAX_TIME          0 = none
#   MEM_POLL_INTERVAL_SEC  poll pg_stat_* while request runs (default 0.25)
#   REPORT_FILE            default ./gabi-bench-report.md
#
# Example (same vars as running GABI from env.template):
#   # optional: set -a; source env.template; set +a
#   ./scripts/bench-query-endpoints.sh
#
# Example (explicit libpq):
#   export PGHOST=... PGPORT=5432 PGUSER=postgres PGPASSWORD=... PGDATABASE=postgres PGSSLMODE=require
#   SOURCE_ENV_FILE=0 ./scripts/bench-query-endpoints.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

GABI_BASE_URL="${GABI_BASE_URL:-http://127.0.0.1:8080}"
X_FORWARDED_USER="${X_FORWARDED_USER:-$GABI_USER}"
RUNS="${RUNS:-10}"
QUERY_JSON_FILE="${QUERY_JSON_FILE:-${SCRIPT_DIR}/bench.json}"
PSQL_CMD="${PSQL:-psql}"
CURL_MAX_TIME="${CURL_MAX_TIME:-0}"
MEM_POLL_INTERVAL_SEC="${MEM_POLL_INTERVAL_SEC:-0.25}"
REPORT_FILE="${REPORT_FILE:-${REPO_ROOT}/gabi-bench-report.md}"

# psql uses libpq; GABI uses DB_* (see env.template). Source ENV_FILE then map DB_* → PG* unless already set.
ENV_FILE="${ENV_FILE:-${REPO_ROOT}/env.template}"
SOURCE_ENV_FILE="${SOURCE_ENV_FILE:-1}"
if [[ "$SOURCE_ENV_FILE" == "1" || "$SOURCE_ENV_FILE" == "yes" || "$SOURCE_ENV_FILE" == "true" ]] && [[ -f "$ENV_FILE" ]]; then
	set +u
	# shellcheck source=/dev/null
	. "$ENV_FILE"
	set -u
fi

export PGHOST="${PGHOST:-${DB_HOST:-}}"
export PGPORT="${PGPORT:-${DB_PORT:-5432}}"
export PGUSER="${PGUSER:-${DB_USER:-}}"
export PGPASSWORD="${PGPASSWORD:-${DB_PASS:-}}"
export PGDATABASE="${PGDATABASE:-${DB_NAME:-}}"
if [[ -n "${PGSSLMODE:-}" || -n "${DB_SSLMODE:-}" ]]; then
	export PGSSLMODE="${PGSSLMODE:-${DB_SSLMODE:-}}"
fi

usage() {
	sed -n '1,42p' "$0" | tail -n +2
	exit "${1:-0}"
}

[[ "${1:-}" == "-h" || "${1:-}" == "--help" ]] && usage 0

for cmd in curl awk "$PSQL_CMD"; do
	command -v "$cmd" >/dev/null 2>&1 || {
		echo "error: required command not found: $cmd" >&2
		exit 1
	}
done

[[ -f "$QUERY_JSON_FILE" ]] || {
	echo "error: QUERY_JSON_FILE not found: $QUERY_JSON_FILE" >&2
	exit 1
}

PG_MAJOR="$("$PSQL_CMD" -v ON_ERROR_STOP=1 -At -c "SHOW server_version_num;" 2>/dev/null | head -1 | cut -c1-2)" || true
if [[ -z "${PG_MAJOR:-}" || ! "$PG_MAJOR" =~ ^[0-9]+$ ]]; then
	echo "error: cannot read server_version_num via psql; set DB_HOST/DB_USER/DB_PASS/DB_NAME in ${ENV_FILE} (or SOURCE_ENV_FILE=0 and set PGHOST PGUSER PGPASSWORD PGDATABASE)" >&2
	exit 1
fi

pg_metrics_tsv() {
	if [[ "$PG_MAJOR" -ge 14 ]]; then
		"$PSQL_CMD" -v ON_ERROR_STOP=1 -At -F $'\t' -c "
		SELECT
			extract(epoch from clock_timestamp())::text,
			d.blks_read::text,
			d.blks_hit::text,
			(case when (d.blks_hit + d.blks_read) > 0
				then round(100.0 * d.blks_hit / (d.blks_hit + d.blks_read), 6)::text
				else '' end),
			d.temp_files::text,
			d.temp_bytes::text,
			pg_database_size(d.datid)::text,
			w.wal_records::text,
			w.wal_bytes::text,
			w.wal_buffers_full::text
		FROM pg_stat_database d
		CROSS JOIN pg_stat_wal w
		WHERE d.datname = current_database();"
	else
		"$PSQL_CMD" -v ON_ERROR_STOP=1 -At -F $'\t' -c "
		SELECT
			extract(epoch from clock_timestamp())::text,
			d.blks_read::text,
			d.blks_hit::text,
			(case when (d.blks_hit + d.blks_read) > 0
				then round(100.0 * d.blks_hit / (d.blks_hit + d.blks_read), 6)::text
				else '' end),
			d.temp_files::text,
			d.temp_bytes::text,
			pg_database_size(d.datid)::text,
			'', '', '';"
	fi
}

pg_memory_settings_md() {
	"$PSQL_CMD" -v ON_ERROR_STOP=1 -At -F $'\t' -c "
	SELECT name, setting, COALESCE(unit, ''), short_desc
	FROM pg_settings
	WHERE name IN (
		'shared_buffers', 'effective_cache_size', 'work_mem', 'maintenance_work_mem',
		'temp_buffers', 'wal_buffers', 'max_connections', 'autovacuum_work_mem',
		'huge_pages', 'dynamic_shared_memory_type'
	)
	ORDER BY name;"
}

curl_common=( -sS -X POST -H 'Content-Type: application/json' -H "X-Forwarded-User: ${X_FORWARDED_USER}" --data-binary "@${QUERY_JSON_FILE}" )
[[ "$CURL_MAX_TIME" != "0" ]] && curl_common+=( --max-time "$CURL_MAX_TIME" )

CURL_FMT=$'%{http_code}\t%{time_starttransfer}\t%{time_total}\t%{size_download}'

poll_pg_while() {
	local watch_pid="$1"
	local out_file="$2"
	while kill -0 "$watch_pid" 2>/dev/null; do
		pg_metrics_tsv >>"$out_file" || true
		sleep "$MEM_POLL_INTERVAL_SEC"
	done
	pg_metrics_tsv >>"$out_file" || true
}

summarize_pg_samples() {
	local f="$1"
	[[ -s "$f" ]] || {
		echo "no samples"
		return
	}
	awk -F'\t' '
	BEGIN { first=1 }
	{
		if (first) {
			r0=$2+0; h0=$3+0; tf0=$5+0; tb0=$6+0
			wrec0=$8; wb0=$9; wbf0=$10
			first=0
		}
		if ($4!="") {
			ch=$4+0
			if (cmin=="" || ch<cmin) cmin=ch
			if (cmax=="" || ch>cmax) cmax=ch
		}
		tb=$6+0; if (tbmax=="" || tb>tbmax) tbmax=tb
		tf=$5+0; if (tfmax=="" || tf>tfmax) tfmax=tf
		if ($10!="") { w=$10+0; if (wbfmax=="" || w>wbfmax) wbfmax=w }
		if ($9!="") { wb=$9+0; if (wbmax=="" || wb>wbmax) wbmax=wb }
		lr=$2+0; lh=$3+0; ltb=$6+0; ltf=$5+0
		lwrec=$8; lwb=$9; lwbf=$10
	}
	END {
		dbr=lr-r0; dbh=lh-h0; dtb=ltb-tb0; dtf=ltf-tf0
		hitmm=(cmin==""||cmax=="") ? "n/a,n/a" : sprintf("%.4f,%.4f", cmin+0, cmax+0)
		printf "samples=%d hit_pct[min,max]=[%s] temp_bytes_peak=%s temp_files_peak=%s Δblks_read=%s Δblks_hit=%s Δtemp_bytes=%s Δtemp_files=%s",
			NR, hitmm, tbmax+0, tfmax+0, dbr+0, dbh+0, dtb+0, dtf+0
		if (wb0!="" && lwb!="") printf " Δwal_bytes=%s", (lwb+0)-(wb0+0)
		if (wbf0!="" && lwbf!="") printf " Δwal_buffers_full=%s", (lwbf+0)-(wbf0+0)
		if (wbfmax!="") printf " wal_buffers_full_peak=%s", wbfmax+0
		else printf " wal_buffers_full_peak=n/a"
	}
	' "$f"
}

run_one_curl_poll() {
	local url="$1"
	local poll_file="$2"
	local curl_out="$3"

	: >"$poll_file"
	pg_metrics_tsv >>"$poll_file" || true

	set +e
	curl "${curl_common[@]}" -o /dev/null -w "$CURL_FMT" "$url" >"$curl_out" &
	local cpid=$!
	set -e

	poll_pg_while "$cpid" "$poll_file"
	wait "$cpid"
	local ec=$?
	echo "$ec"
}

stats_mean_min_max() {
	awk -v label="$1" '
	{
		x=$1+0; sum+=x; n++; if(n==1||x<min)min=x; if(n==1||x>max)max=x
	}
	END {
		if(n<1){print label ": n=0"; exit}
		avg=sum/n
		printf "%s: n=%d mean=%.6f min=%.6f max=%.6f\n", label, n, avg, min, max
	}' "$2"
}

WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT

TS="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

{
	echo "## GABI /query vs /streamquery benchmark"
	echo
	echo "- **When:** $TS (UTC)"
	echo "- **GABI:** \`$GABI_BASE_URL\`"
	echo "- **Query file:** \`$QUERY_JSON_FILE\`"
	echo "- **Runs per endpoint:** $RUNS"
	echo "- **TTFB:** curl \`time_starttransfer\` (first byte of HTTP response)."
	if [[ "$PG_MAJOR" -ge 14 ]]; then
		echo "- **PostgreSQL metrics:** \`pg_stat_database\`, \`pg_stat_wal\` (PG14+)."
	else
		echo "- **PostgreSQL metrics:** \`pg_stat_database\` only (no \`pg_stat_wal\` before PG14)."
	fi
	echo "- **Not included:** OS RSS, cgroup memory, or RDS CloudWatch — only what PostgreSQL exposes in catalogs/views."
	echo
	echo "### Interpretation (metadata, not full “memory pressure”)"
	echo
	echo "- **shared_buffers / effective_cache_size / work_mem** — configured capacity from \`pg_settings\`."
	echo "- **Buffer cache effectiveness** — \`pg_stat_database\` \`blks_hit\` vs \`blks_read\` (sampled during each request)."
	echo "- **Spill / temp pressure proxy** — \`temp_files\`, \`temp_bytes\` peaks and deltas while the query runs."
	echo "- **WAL / wal_buffers** — \`pg_stat_wal\` (PG14+): \`wal_bytes\`, \`wal_buffers_full\` (buffer saturation signal)."
	echo "- **Database size on disk** — \`pg_database_size\` (not RAM)."
	echo
	echo "### Memory-related settings (\`pg_settings\`)"
	echo
	echo '| name | setting | unit | description |'
	echo '|------|---------|------|-------------|'
	while IFS=$'\t' read -r name setting unit desc; do
		[[ -z "${name:-}" ]] && continue
		echo "| $name | $setting | $unit | ${desc//|/\\|} |"
	done < <(pg_memory_settings_md)
	echo
	echo "### Per-run table"
	echo
	echo '| Mode | Run | HTTP | TTFB (s) | Total (s) | Bytes downloaded | PG sample summary |'
	echo '|------|-----|------|----------|-----------|------------------|-------------------|'
} >"$REPORT_FILE"

collect_endpoint() {
	local label="$1"
	local path="$2"
	local url="${GABI_BASE_URL%/}${path}"
	local ttfb_file="$WORKDIR/${label}.ttfb"
	local tot_file="$WORKDIR/${label}.tot"
	: >"$ttfb_file"
	: >"$tot_file"

	local i
	for ((i = 1; i <= RUNS; i++)); do
		local pf="$WORKDIR/${label}-$i.pg.tsv"
		local cf="$WORKDIR/${label}-$i.curl"
		local ec
		ec="$(run_one_curl_poll "$url" "$pf" "$cf")"
		IFS=$'\t' read -r code ttfb tot dl <"$cf" || true
		local summ
		summ="$(summarize_pg_samples "$pf")"
		if [[ "$ec" != "0" ]]; then
			echo "| $label | $i | curl_err_$ec | — | — | — | $summ |" >>"$REPORT_FILE"
			continue
		fi
		if [[ "$code" != "200" ]]; then
			echo "| $label | $i | $code | — | — | $dl | $summ |" >>"$REPORT_FILE"
			continue
		fi
		echo "$ttfb" >>"$ttfb_file"
		echo "$tot" >>"$tot_file"
		echo "| $label | $i | $code | $ttfb | $tot | $dl | $summ |" >>"$REPORT_FILE"
	done
}

collect_endpoint "non-streaming" "/query"
collect_endpoint "streaming" "/streamquery"

{
	echo
	echo "### Summary (successful HTTP 200 only)"
	echo
	if [[ -s "$WORKDIR/non-streaming.ttfb" ]]; then
		stats_mean_min_max "non-streaming TTFB" "$WORKDIR/non-streaming.ttfb"
		stats_mean_min_max "non-streaming total" "$WORKDIR/non-streaming.tot"
	else
		echo "non-streaming: no successful runs"
	fi
	if [[ -s "$WORKDIR/streaming.ttfb" ]]; then
		stats_mean_min_max "streaming TTFB" "$WORKDIR/streaming.ttfb"
		stats_mean_min_max "streaming total" "$WORKDIR/streaming.tot"
	else
		echo "streaming: no successful runs"
	fi
	echo
	echo "### Comparison"
	echo
	if [[ -s "$WORKDIR/non-streaming.ttfb" && -s "$WORKDIR/streaming.ttfb" ]]; then
		paste "$WORKDIR/non-streaming.ttfb" "$WORKDIR/streaming.ttfb" | awk -F'\t' '
		{
			n++; d=$2-$1; sum+=d; if(n==1||d<min)min=d; if(n==1||d>max)max=d
		}
		END {
			if(n>0) printf "TTFB streaming − non-streaming (s): n=%d mean=%.6f min=%.6f max=%.6f (negative ⇒ streaming first byte faster)\n", n, sum/n, min, max
		}'
	fi
	echo
	echo "Per-run PostgreSQL TSV samples were under \`$WORKDIR\` (removed after this run)."
} >>"$REPORT_FILE"

echo "Report written: $REPORT_FILE"
cat "$REPORT_FILE"
