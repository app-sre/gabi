## GABI /query vs /streamquery benchmark

- **When:** 2026-04-08T16:08:23Z (UTC)
- **GABI:** `http://127.0.0.1:8080`
- **Query file:** `/home/tcarvalh/workspace/commercial/fork/gabi/scripts/bench.json`
- **Runs per endpoint:** 10
- **TTFB:** curl `time_starttransfer` (first byte of HTTP response).
- **PostgreSQL metrics:** `pg_stat_database`, `pg_stat_wal` (PG14+).
- **Not included:** OS RSS, cgroup memory, or RDS CloudWatch — only what PostgreSQL exposes in catalogs/views.

### Interpretation (metadata, not full “memory pressure”)

- **shared_buffers / effective_cache_size / work_mem** — configured capacity from `pg_settings`.
- **Buffer cache effectiveness** — `pg_stat_database` `blks_hit` vs `blks_read` (sampled during each request).
- **Spill / temp pressure proxy** — `temp_files`, `temp_bytes` peaks and deltas while the query runs.
- **WAL / wal_buffers** — `pg_stat_wal` (PG14+): `wal_bytes`, `wal_buffers_full` (buffer saturation signal).
- **Database size on disk** — `pg_database_size` (not RAM).

### Memory-related settings (`pg_settings`)

| name | setting | unit | description |
|------|---------|------|-------------|
| autovacuum_work_mem | 65536 | kB | Sets the maximum memory to be used by each autovacuum worker process. |
| dynamic_shared_memory_type | posix | Selects the dynamic shared memory implementation used. |  |
| effective_cache_size | 110994 | 8kB | Sets the planner's assumption about the total size of the data caches. |
| huge_pages | off | Use of huge pages on Linux or Windows. |  |
| maintenance_work_mem | 65536 | kB | Sets the maximum memory to be used for maintenance operations. |
| max_connections | 190 | Sets the maximum number of concurrent connections. |  |
| shared_buffers | 55497 | 8kB | Sets the number of shared memory buffers used by the server. |
| temp_buffers | 1024 | 8kB | Sets the maximum number of temporary buffers used by each session. |
| wal_buffers | 1734 | 8kB | Sets the number of disk-page buffers in shared memory for WAL. |
| work_mem | 4096 | kB | Sets the maximum memory to be used for query workspaces. |

### Per-run table

| Mode | Run | HTTP | TTFB (s) | Total (s) | Bytes downloaded | PG sample summary |
|------|-----|------|----------|-----------|------------------|-------------------|
| non-streaming | 1 | 200 | 1.716738 | 1.720506 | 10888926 | samples=4 hit_pct[min,max]=[75.7151,75.7164] temp_bytes_peak=261932416 temp_files_peak=8 Δblks_read=0 Δblks_hit=1498 Δtemp_bytes=14000000 Δtemp_files=1 Δwal_bytes=0 Δwal_buffers_full=0 wal_buffers_full_peak=39017429 |
| non-streaming | 2 | 200 | 0.948370 | 0.951261 | 10888926 | samples=3 hit_pct[min,max]=[75.7168,75.7176] temp_bytes_peak=275932416 temp_files_peak=9 Δblks_read=0 Δblks_hit=966 Δtemp_bytes=14000000 Δtemp_files=1 Δwal_bytes=0 Δwal_buffers_full=0 wal_buffers_full_peak=39017429 |
| non-streaming | 3 | 200 | 1.032158 | 1.037643 | 10888926 | samples=3 hit_pct[min,max]=[75.7180,75.7189] temp_bytes_peak=289932416 temp_files_peak=10 Δblks_read=0 Δblks_hit=1049 Δtemp_bytes=14000000 Δtemp_files=1 Δwal_bytes=0 Δwal_buffers_full=0 wal_buffers_full_peak=39017429 |
| non-streaming | 4 | 200 | 1.113088 | 1.114873 | 10888926 | samples=3 hit_pct[min,max]=[75.7193,75.7201] temp_bytes_peak=303932416 temp_files_peak=11 Δblks_read=0 Δblks_hit=966 Δtemp_bytes=14000000 Δtemp_files=1 Δwal_bytes=0 Δwal_buffers_full=0 wal_buffers_full_peak=39017429 |
| non-streaming | 5 | 200 | 1.317605 | 1.320210 | 10888926 | samples=3 hit_pct[min,max]=[75.7205,75.7214] temp_bytes_peak=317932416 temp_files_peak=12 Δblks_read=0 Δblks_hit=1081 Δtemp_bytes=14000000 Δtemp_files=1 Δwal_bytes=4299 Δwal_buffers_full=0 wal_buffers_full_peak=39017429 |
| non-streaming | 6 | 200 | 1.404831 | 1.408864 | 10888926 | samples=3 hit_pct[min,max]=[75.7218,75.7226] temp_bytes_peak=331932416 temp_files_peak=13 Δblks_read=0 Δblks_hit=966 Δtemp_bytes=14000000 Δtemp_files=1 Δwal_bytes=0 Δwal_buffers_full=0 wal_buffers_full_peak=39017429 |
| non-streaming | 7 | 200 | 1.222307 | 1.224244 | 10888926 | samples=3 hit_pct[min,max]=[75.7231,75.7239] temp_bytes_peak=345932416 temp_files_peak=14 Δblks_read=0 Δblks_hit=966 Δtemp_bytes=14000000 Δtemp_files=1 Δwal_bytes=0 Δwal_buffers_full=0 wal_buffers_full_peak=39017429 |
| non-streaming | 8 | 200 | 1.197539 | 1.200311 | 10888926 | samples=3 hit_pct[min,max]=[75.7243,75.7251] temp_bytes_peak=359932416 temp_files_peak=15 Δblks_read=0 Δblks_hit=966 Δtemp_bytes=14000000 Δtemp_files=1 Δwal_bytes=0 Δwal_buffers_full=0 wal_buffers_full_peak=39017429 |
| non-streaming | 9 | 200 | 1.172120 | 1.177701 | 10888926 | samples=3 hit_pct[min,max]=[75.7255,75.7264] temp_bytes_peak=373932416 temp_files_peak=16 Δblks_read=0 Δblks_hit=1049 Δtemp_bytes=14000000 Δtemp_files=1 Δwal_bytes=50 Δwal_buffers_full=0 wal_buffers_full_peak=39017429 |
| non-streaming | 10 | 200 | 1.157742 | 1.160722 | 10888926 | samples=3 hit_pct[min,max]=[75.7268,75.7276] temp_bytes_peak=387932416 temp_files_peak=17 Δblks_read=0 Δblks_hit=966 Δtemp_bytes=14000000 Δtemp_files=1 Δwal_bytes=0 Δwal_buffers_full=0 wal_buffers_full_peak=39017429 |
| streaming | 1 | 200 | 1.056512 | 1.059229 | 11888927 | samples=3 hit_pct[min,max]=[75.7280,75.7288] temp_bytes_peak=401932416 temp_files_peak=18 Δblks_read=0 Δblks_hit=966 Δtemp_bytes=14000000 Δtemp_files=1 Δwal_bytes=0 Δwal_buffers_full=0 wal_buffers_full_peak=39017429 |
| streaming | 2 | 200 | 1.057351 | 1.060792 | 11888927 | samples=3 hit_pct[min,max]=[75.7292,75.7301] temp_bytes_peak=415932416 temp_files_peak=19 Δblks_read=0 Δblks_hit=1049 Δtemp_bytes=14000000 Δtemp_files=1 Δwal_bytes=0 Δwal_buffers_full=0 wal_buffers_full_peak=39017429 |
| streaming | 3 | 200 | 1.070886 | 1.073534 | 11888927 | samples=3 hit_pct[min,max]=[75.7305,75.7313] temp_bytes_peak=429932416 temp_files_peak=20 Δblks_read=0 Δblks_hit=966 Δtemp_bytes=14000000 Δtemp_files=1 Δwal_bytes=0 Δwal_buffers_full=0 wal_buffers_full_peak=39017429 |
| streaming | 4 | 200 | 1.085199 | 1.087411 | 11888927 | samples=3 hit_pct[min,max]=[75.7317,75.7325] temp_bytes_peak=443932416 temp_files_peak=21 Δblks_read=0 Δblks_hit=966 Δtemp_bytes=14000000 Δtemp_files=1 Δwal_bytes=1376 Δwal_buffers_full=0 wal_buffers_full_peak=39017429 |
| streaming | 5 | 200 | 1.063168 | 1.065070 | 11888927 | samples=3 hit_pct[min,max]=[75.7329,75.7337] temp_bytes_peak=457932416 temp_files_peak=22 Δblks_read=0 Δblks_hit=966 Δtemp_bytes=14000000 Δtemp_files=1 Δwal_bytes=0 Δwal_buffers_full=0 wal_buffers_full_peak=39023886 |
| streaming | 6 | 200 | 1.077537 | 1.079964 | 11888927 | samples=3 hit_pct[min,max]=[75.7341,75.7350] temp_bytes_peak=471932416 temp_files_peak=23 Δblks_read=0 Δblks_hit=1049 Δtemp_bytes=14000000 Δtemp_files=1 Δwal_bytes=50 Δwal_buffers_full=0 wal_buffers_full_peak=39023886 |
| streaming | 7 | 200 | 1.536203 | 1.540798 | 11888927 | samples=3 hit_pct[min,max]=[75.7354,75.7363] temp_bytes_peak=485932416 temp_files_peak=24 Δblks_read=0 Δblks_hit=1049 Δtemp_bytes=14000000 Δtemp_files=1 Δwal_bytes=0 Δwal_buffers_full=0 wal_buffers_full_peak=39023886 |
| streaming | 8 | 200 | 1.167236 | 1.184654 | 11888927 | samples=3 hit_pct[min,max]=[75.7367,75.7375] temp_bytes_peak=499932416 temp_files_peak=25 Δblks_read=0 Δblks_hit=966 Δtemp_bytes=14000000 Δtemp_files=1 Δwal_bytes=0 Δwal_buffers_full=0 wal_buffers_full_peak=39023886 |
| streaming | 9 | 200 | 1.164034 | 1.167205 | 11888927 | samples=3 hit_pct[min,max]=[75.7379,75.7387] temp_bytes_peak=513932416 temp_files_peak=26 Δblks_read=0 Δblks_hit=1049 Δtemp_bytes=14000000 Δtemp_files=1 Δwal_bytes=0 Δwal_buffers_full=0 wal_buffers_full_peak=39023886 |
| streaming | 10 | 200 | 1.102200 | 1.105290 | 11888927 | samples=3 hit_pct[min,max]=[75.7392,75.7400] temp_bytes_peak=527932416 temp_files_peak=27 Δblks_read=0 Δblks_hit=966 Δtemp_bytes=14000000 Δtemp_files=1 Δwal_bytes=0 Δwal_buffers_full=0 wal_buffers_full_peak=39023886 |

### Summary (successful HTTP 200 only)

non-streaming TTFB: n=10 mean=1.228250 min=0.948370 max=1.716738
non-streaming total: n=10 mean=1.231634 min=0.951261 max=1.720506
streaming TTFB: n=10 mean=1.138033 min=1.056512 max=1.536203
streaming total: n=10 mean=1.142395 min=1.059229 max=1.540798

### Comparison

TTFB streaming − non-streaming (s): n=10 mean=-0.090217 min=-0.660226 max=0.313896 (negative ⇒ streaming first byte faster)

Per-run PostgreSQL TSV samples were under `/tmp/tmp.isbzeJTveD` (removed after this run).
