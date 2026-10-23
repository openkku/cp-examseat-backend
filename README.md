# CP Exam Seat — Backend

JSON API for **CP Exam Seat**, the exam seat lookup of the College of Computing,
Khon Kaen University. It serves student exam schedules, room rosters, seating
layouts, analytics and iCalendar feeds.

This repository was split out of the monorepo
[openkku/cp-examseat](https://github.com/openkku/cp-examseat). The web UI now
lives in [openkku/cp-examseat-frontend](https://github.com/openkku/cp-examseat-frontend)
(Next.js). The monorepo itself is unchanged.

**Stack** (same as the monorepo): Go 1.26 · [chi](https://github.com/go-chi/chi) ·
SQLite ([modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite), no CGO) ·
[otter](https://github.com/maypok86/otter) in-memory caches · zstd/brotli/gzip compression.

## Architecture — clean architecture, one slice per feature

The code is grouped by feature. Each slice has a **core** (domain types, use
cases and the interfaces, or *ports*, it needs) and **adapters** in
subpackages that plug the core into HTTP, files or other services. The
database is the one adapter shared by every slice: a single package per
engine (see [Database engines](#database-engines)).

```
               HTTP request
                    │
                    ▼
   internal/app ── router (chi, middleware) ── composition root: wires cores to adapters
                    │
        ┌───────────┴─────────────────────────────────────────────┐
        ▼                                                         ▼
  <slice>/handler   parse + validate the request,           <slice>/presenter  wire format
                    call the core, map errors to statuses                      (JSON, .ics)
        │
        ▼
  <slice>           core: entities, rules, use cases (Service) and ports
        ▲           — imports only the standard library, shared/domain and other cores
        │ implements a port
  database/sqlite    SQL + schema for every slice (one package per engine)
  <slice>/filesystem room JSON files
  checkex/scrapler   checkex lookups over HTTP
  ingest/importer    pdf / xlsx / json seat extractors
```

The dependency rule: **adapters depend on cores, never the other way round.**
The compiler enforces it, because a core package can't import its own
subpackages without an import cycle. In practice:

- A core never imports `net/http`, `database/sql`, chi or an adapter package.
  Each port is declared in the core that uses it, always in
  **`<slice>/ports.go`**, and each adapter has a compile-time check that it
  implements the port. Database ports are always called `Repository`.
- Every core uses the same files:

  | File | Holds |
  |------|-------|
  | `<slice>.go` | package doc and the slice's data types (queries, options, results) |
  | `errors.go` | sentinel errors and error types |
  | `ports.go` | the interfaces the core needs (its ports) |
  | `service.go` | `Service`, its constructor and the use cases |

  A slice without data types of its own (`round`, `roomconfig`) keeps its
  package doc in `service.go`. Larger helpers get their own file
  (`checkex/limiter.go`, `stats/builder.go`).
- Handlers depend on small interfaces of their own, not on `*Service`, so
  they're easy to test with fakes.
- **`internal/app`** is the composition root. It's the only place that knows
  which adapter implements which port. There is no global mutable state.

### Ports

| Slice | Port (`<slice>/ports.go`) | Implemented by |
|-------|---------------------------|----------------|
| `exam` | `Repository` | `database/sqlite` |
| | `Upstream` | `checkex.Service` (`exam.NoUpstream` when checkex is off) |
| | `RoomDirectory` | `room.Service` |
| `checkex` | `Repository` | `database/sqlite` |
| | `Source` | `checkex/scrapler.Client` |
| `ingest` | `Repository` | `database/sqlite` |
| | `Extractor` | `ingest/importer.Registry` |
| `admin` | `Repository`, `DatasetPurger` | `database/sqlite` |
| | `Importer` | `ingest.Service` |
| | `Reloader` | `platform/reloadsignal.Notifier` |
| `round` | `Repository` | `database/sqlite` |
| `stats` | `Repository` | `database/sqlite` |
| `room` | `CatalogLoader` | `room/filesystem.Loader` |
| `roomconfig` | `Store` | `roomconfig/filesystem.Store` |
| | `Publisher` | `platform/reloadsignal.Notifier` |

`internal/database/sqlite/ports.go` lists every port the database implements.

### Database engines

All SQL lives in one package per engine. Today that is `internal/database/sqlite`:
the modernc driver, DSN pragmas, schema migrations, and one file per slice
(`exam.go`, `checkex.go`, `ingest.go`, …) with that slice's queries. A single
`*sqlite.DB` implements every slice's repository port, and `ports.go` lists
them as compile-time checks. Only `internal/app`, `cmd/migrate` and `cmd/operator` import it.
The tests next to it run every query against SQLite.

To add another engine (for example `internal/database/mariadb`), add a
sibling package with an `Open`, its own schema, and methods for every port in
`ports.go`, then choose it in `internal/app`. The SQLite queries use these
dialect features, which the new engine needs its own version of:

- `?` placeholders
- `INSERT OR IGNORE` and `ON CONFLICT … DO UPDATE` (ingest, checkex)
- `GROUP_CONCAT(… ORDER BY …)` (`seatSelect` in `seats.go`); labels can't contain commas, which ingest enforces
- `res.LastInsertId()` (ingest, checkex)
- the `rowid` ordering of rounds (`rounds` in `seats.go`)
- `IS ?` comparisons (checkex)

### Layout

```
cmd/
  server/          API server (port 8080)
  migrate/         CLI: import seating files, rename and delete rounds
  operator/        operator app: terminal UI, or `operator web` (embedded Vite UI, port 8081)
internal/
  app/             composition root, router + end-to-end API tests
  config/          environment configuration
  shared/
    domain/        shared kernel: Seat, Round, labels, student IDs, Thai dates
    httpx/         JSON/error bodies, cached pre-compressed responses, parameter limits
  database/
    sqlite/        SQLite: driver, settings, migrations, every slice's queries (+ SQL tests)
  platform/
    middleware/    compression, CORS, rate limiting, security headers, local-tool guard
    compression/   content-encoding negotiation and codecs
    reloadsignal/  $DATA_DIR/.reload: tools ask a running server to reload
  operator/        operator app web API (router) and tui/ (terminal UI)
  exam/            student schedules, room explorer, calendar feeds
                   handler/ presenter/
  checkex/         on-demand seat lookups from checkex   scrapler/
  room/            room catalog (layouts, images, maps)  filesystem/ handler/ presenter/
  roomconfig/      room editing (operator app)            filesystem/ handler/
  round/           cached round list                      handler/
  stats/           analytics dashboard                    handler/
  ingest/          seating file imports                   importer/{pdf,xlsx,jsonfile}/
  admin/           round management (operator app, CLI)   handler/
script/            Python helpers (xlsx scraper, enrollment parser)
docs/              API and environment reference
```

## Operating it

| Task | How |
|------|-----|
| Import a round, rename or delete one | Operator app (`operator` in a terminal, or `operator web` through an SSH tunnel), or `migrate` |
| Edit rooms and seating layouts | Operator app: Rooms tab (`operator web` has the drag-and-drop editor) |
| Reload after a change | Automatic: the operator app and `migrate` touch `$DATA_DIR/.reload` and the server reloads within 2s. After editing files by hand, `touch $DATA_DIR/.reload` or `kill -HUP <pid>` |
| Back up | Not done by the app. Back up `$DATA_DIR/exams.db` at the infrastructure level, e.g. snapshot the data volume or run `sqlite3 exams.db ".backup /backups/exams-$(date +%F).db"` from the host on a schedule |
| Deploy | Every push to `main` publishes `ghcr.io/openkku/cp-examseat-backend:latest` (and `:sha-…`, `:X.Y.Z` for `vX.Y.Z` tags). [openkku/cp-examseat-deployment](https://github.com/openkku/cp-examseat-deployment) runs it with the frontend, scrapler and the KKU VPN behind HTTPS |
| Serve a checkex round | Set `CHECKEX_ROUNDS` (e.g. `final_1_2569`) and `SCRAPLER_URL`; seats are fetched on first lookup through [scrapler](https://github.com/openkku/akhsakov-scrapler). See [docs/env.md](docs/env.md#9-on-demand-lookup-from-checkex-checkex_-scrapler_url) |

The server needs no restart after a change: rounds, rooms, statistics and
all response caches are refreshed on reload. There is no admin API: data is
only changed by tools run on the server itself.

## Getting started

```bash
# Run the API (reads ./data by default)
go run ./cmd/server

# Import an exam round
go run ./cmd/migrate data/source/final_2_2568.xlsx final_2_2568 "ปลายภาค 2/2568"

# Import an out-of-schedule dataset with labels and a custom layout
go run ./cmd/migrate custom --file data/source/custom/lab.pdf --round final_2_2568 \
  --custom-id FINAL_2_OOP_LAB_2026 --labels LAB,Lab --room-layout CP9421_LAB

# Rename or delete a round (or one custom dataset)
go run ./cmd/migrate rename final_2_2568 "ปลายภาค 2/2568"
go run ./cmd/migrate delete final_2_2568 --custom-id FINAL_2_OOP_LAB_2026

# Operator app in the terminal: rounds (import, rename, delete) and rooms
go run ./cmd/operator

# ...or in the browser with visual layout editing, import preview, and health checks
(cd cmd/operator/frontend && npm ci && npm run build)   # once
go run ./cmd/operator web         # http://localhost:8081, loopback only; see OPERATOR_ADDR
```

PDF imports need `pdftotext` (poppler-utils). A running server picks up every
change within 2 seconds.

### Tests

```bash
pip install openpyxl   # used by the xlsx equivalence tests
go vet ./...
go test ./...
```

### Docker

```bash
docker compose up -d --build
# import data inside the container
docker compose exec backend ./migrate /app/data/source/final_2_2568.xlsx final_2_2568
# or the operator app's terminal UI
docker compose exec -it backend ./operator
```

The image contains `server`, `migrate` and `operator`; mount your data
directory at `/app/data`.

## Security

- SQL is fully parameterized; `internal/app/security_test.go` exercises
  injection payloads against every query parameter.
- Query values are capped at 64 characters (student IDs at 20 digits) and
  request headers at 16 KiB, because values become in-memory cache keys.
- Every response carries `nosniff`, `X-Frame-Options: DENY`,
  `Referrer-Policy: no-referrer` and a `default-src 'none'` CSP.
- Room images are served through `http.Dir`, which cannot leave `room/image/`.
- The public server cannot change data: there is no admin API. Data changes
  only through the operator app and `migrate`, run on the server.
- The operator app's terminal UI opens no port. `operator web` listens on
  loopback only and rejects non-loopback `Host` headers, cross-site writes,
  non-JSON writes without `X-Operator` (so a cross-site form can't upload),
  and bodies over 1 MiB (uploads over `MAX_UPLOAD_MB`).
- Every client IP is rate limited (default 10 req/s, burst 40); the client
  address comes from `X-Forwarded-For` only via `TRUSTED_PROXIES`.
- CI runs `govulncheck`, `gosec`, the race detector and fuzz tests for the
  parsers that handle untrusted input.

## Documentation

- [API reference](docs/api.md)
- [Environment variables](docs/env.md)

## License

MIT — see [LICENSE](LICENSE).
