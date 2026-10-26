# AGENTS.md

Outbound data-source connections and the built-in demo source. Go standards
live in [server/core/AGENTS.md](../../AGENTS.md).

## The demo data sources

Every organization is seeded, unconditionally, with the Prometheus data
source "Demo Metrics" at `demo://engineering`, and the welcome document whose
charts query it. The Loki "Demo Logs" at the same URL is ready but stays
commented out in `NewDemoDataSources` until the web app shows loki data
sources.

- `demo/` synthesises both in-process. `runner.ensurePrepared` picks it for
  exactly that URL under either type; any other `demo://` is
  `demo.ErrUnknownSource`. The types stay `prometheus` and `loki`, so
  nothing downstream can tell.
- One `demo.Client` lives on `datasource.Manager` for the whole process,
  because its registry walks cache replayed history; a runner is built per
  call.
- The timeline is a pure function of the tick (fixed epoch, one-minute
  tick, per-series seed), so history never rewrites and any window
  backfills. `walk.checkpoint` caches one value per day-long segment.
- Metric queries run through the official `promql` engine over a
  `storage.Queryable` synthesising only the requested window, at the
  query's step.
- `registry.go` declares every metric family. `Test_welcomeDocumentQueries`
  runs every query in `internal/document/files/available_metrics.json`
  against it.
- Loki's LogQL code cannot be imported (its go.mod needs 8 `replace`
  directives); `logql.go` answers the subset of selector, line filters and
  label filters.

## Processor tests

`processor/` starts throwaway MariaDB, Postgres and Loki (oldest supported
and newest) containers (gnomock) to cover real connection, query and
read-only-check paths; docker is a unit-test dependency here, as in
`internal/db`. Loki holds a 6000-line corpus built to break paging (one
timestamp holding more lines than a page); its config keeps backfilled
lines in the ingesters, which otherwise flush them out of query reach.
