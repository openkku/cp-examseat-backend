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
subpackages that plug the core into HTTP, the database, files or other
services:

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
  <slice>/database   SQL through database/sql (engine-neutral)
  <slice>/filesystem room JSON files
  checkex/scrapler   checkex lookups over HTTP
  ingest/importer    pdf / xlsx / json seat extractors
```

The dependency rule: **adapters depend on cores, never the other way round.**
The compiler enforces it, because a core package can't import its own
subpackages without an import cycle. In practice:

- A core never imports `net/http`, `database/sql`, chi or an adapter package.
  Each port is declared in the core that uses it (`exam.Repository`,
  `checkex.Source`, `ingest.Extractor`, …), and each adapter has a
  compile-time check that it implements the port.
- Handlers depend on small interfaces of their own, not on `*Service`, so
  they're easy to test with fakes.
- **`internal/app`** is the composition root. It's the only place that knows
  which adapter implements which port. There is no global mutable state.

### Database engines

Slice adapters are named `database`. They talk to the database only through
`database/sql` and `platform/database`, and never import a driver. Everything
SQLite-specific (the modernc driver, DSN pragmas, schema migrations) lives in
`platform/database/sqlite`. Only `internal/app` and `cmd/migrate` import it.
The SQL tests in `platform/database/sqlite` run every slice's queries against
SQLite.

To add another engine (for example `platform/database/postgres`), add a
sibling package with an `Open` that returns a `*database.DB` and its own
schema, and choose it in `internal/app`. The slice queries still use these
SQLite/MySQL-isms, which a new engine needs to handle:

- `?` placeholders
- `INSERT OR IGNORE` (ingest, checkex)
- `GROUP_CONCAT` (`platform/database.SeatSelect`)
- `res.LastInsertId()` (ingest, checkex)
- the `rowid` ordering of rounds (`platform/database.Rounds`)
- `IS ?` comparisons (checkex)

### Layout

```
cmd/
  server/          API server (port 8080)
  migrate/         CLI importing seating files into exams.db
  room-config/     local room layout editor (Go API + embedded Vite UI, port 8081)
internal/
  app/             composition root, router + end-to-end API tests
  config/          environment configuration
  shared/
    domain/        shared kernel: Seat, Round, labels, student IDs, Thai dates
    httpx/         JSON/error bodies, cached pre-compressed responses, parameter limits
  platform/
    database/      engine-neutral DB handle and shared seat/round row helpers
      sqlite/      SQLite engine: driver, connection settings, migrations (+ SQL tests)
    middleware/    compression, CORS, rate limiting, security headers, auth
    compression/   content-encoding negotiation and codecs
  exam/            student schedules, room explorer, calendar feeds
                   database/ handler/ presenter/
  checkex/         on-demand seat lookups from checkex   database/ scrapler/
  room/            room catalog (layouts, images, maps)  filesystem/ handler/ presenter/
  roomconfig/      room-config tool use cases             filesystem/ handler/
  round/           cached round list                      database/ handler/
  stats/           analytics dashboard                    database/ handler/
  ingest/          seating file imports                   database/ importer/{pdf,xlsx,jsonfile}/
  admin/           admin API use cases                    database/ handler/
script/            Python helpers (xlsx scraper, enrollment parser)
docs/              API and environment reference
```

## Operating it

| Task | How |
|------|-----|
| Import a round | Admin page (`/admin` on the frontend), or `migrate` CLI then reload |
| Reload after changing data or room files | Admin page "Reload", `POST /api/admin/reload`, or `kill -HUP <pid>` / `docker compose kill -s HUP backend` |
| Back up | Not done by the app. Back up `$DATA_DIR/exams.db` at the infrastructure level, e.g. snapshot the data volume or run `sqlite3 exams.db ".backup /backups/exams-$(date +%F).db"` from the host on a schedule |
| Deploy | Every push to `main` publishes `ghcr.io/openkku/cp-examseat-backend:latest` (and `:sha-…`, `:X.Y.Z` for `vX.Y.Z` tags). [openkku/cp-examseat-deployment](https://github.com/openkku/cp-examseat-deployment) runs it with the frontend, scrapler and the KKU VPN behind HTTPS |
| Serve a checkex round | Set `CHECKEX_ROUNDS` (e.g. `final_1_2569`) and `SCRAPLER_URL`; seats are fetched on first lookup through [scrapler](https://github.com/openkku/akhsakov-scrapler). See [docs/env.md](docs/env.md#9-on-demand-lookup-from-checkex-checkex_-scrapler_url) |

The server needs no restart after an import: rounds, rooms, statistics and
all response caches are refreshed on reload.

## Getting started

```bash
# Run the API (reads ./data by default)
go run ./cmd/server

# Import an exam round
go run ./cmd/migrate data/source/final_2_2568.xlsx final_2_2568 "ปลายภาค 2/2568"

# Import an out-of-schedule dataset with labels and a custom layout
go run ./cmd/migrate custom --file data/source/custom/lab.pdf --round final_2_2568 \
  --custom-id FINAL_2_OOP_LAB_2026 --labels LAB,Lab --room-layout CP9421_LAB

# Edit room layouts (build the UI once, then open http://localhost:8081)
(cd cmd/room-config/frontend && npm ci && npm run build)
go run ./cmd/room-config          # loopback only; see ROOM_CONFIG_ADDR
```

PDF imports need `pdftotext` (poppler-utils). The server re-reads rounds,
rooms and statistics at startup, so restart it after importing data.

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
```

The image contains `server` and `migrate`; mount your data directory at `/app/data`.

## Security

- SQL is fully parameterized; `internal/app/security_test.go` exercises
  injection payloads against every query parameter.
- Query values are capped at 64 characters (student IDs at 20 digits) and
  request headers at 16 KiB, because values become in-memory cache keys.
- Every response carries `nosniff`, `X-Frame-Options: DENY`,
  `Referrer-Policy: no-referrer` and a `default-src 'none'` CSP.
- Room images are served through `http.Dir`, which cannot leave `room/image/`.
- The room-config tool listens on loopback only, rejects non-loopback `Host`
  headers, cross-site and non-JSON writes, and bodies over 1 MiB.
- Every client IP is rate limited (default 10 req/s, burst 40); the client
  address comes from `X-Forwarded-For` only via `TRUSTED_PROXIES`.
- The admin API is off unless `ADMIN_TOKEN` is set, compares tokens in
  constant time, and validates uploaded file types, sizes and IDs.
- CI runs `govulncheck`, `gosec`, the race detector and fuzz tests for the
  parsers that handle untrusted input.

## Documentation

- [API reference](docs/api.md)
- [Environment variables](docs/env.md)

## License

MIT — see [LICENSE](LICENSE).
