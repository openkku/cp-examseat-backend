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

## Architecture — MVC

```
HTTP request
    │
    ▼
routes/            URL → controller action, middleware (logger, recoverer, CORS, compression)
    │
    ▼
controllers/  (C)  parse + validate the request, call a service, pick a view
    │         ▲
    ▼         │
services/     (M)  application logic, in-memory state loaded at startup (rounds, rooms, stats)
    │
    ▼
models/       (M)  domain entities + rules (Seat, Round, Room, Dashboard, ID normalization,
    │              round ordering, analytics), repository interfaces
    ▼
repositories/ (M)  persistence: sqlite/ (exams.db), filesystem/ (room/*.json)

views/        (V)  representations: JSON payloads, error bodies, iCalendar (.ics),
                   cached pre-compressed responses
```

Rules the layers follow:

- **Controllers** never touch the database or format bodies themselves; they
  only talk to services and views.
- **Models** know nothing about HTTP or storage engines. `models.ExamRepository`
  is the persistence contract; `repositories/sqlite` implements it.
- **Views** own the wire format. For example `views.ExamSchedule` is the public
  shape of a seat, independent of the `models.Seat` entity.
- **`internal/app`** is the composition root that wires everything together;
  there is no global mutable state.

### Layout

```
cmd/
  server/          API server (port 8080)
  migrate/         CLI importing seating files into exams.db
  room-config/     local room layout editor (Go API + embedded Vite UI, port 8081)
internal/
  app/             composition root + end-to-end API tests
  config/          environment configuration
  routes/          router definitions
  middleware/      compression, CORS
  controllers/     exam, explore, calendar, room, round, stats, room-config
  services/        exam, round, room, stats, ingest
  models/          entities, domain rules, repository contract
  repositories/    sqlite/ and filesystem/ implementations
  views/           JSON / ICS rendering and response cache
  compression/     content-encoding negotiation and codecs
  importers/       pdf/, xlsx/, jsonfile/ seat extractors
script/            Python helpers (xlsx scraper, enrollment parser)
docs/              API and environment reference
```

## Operating it

| Task | How |
|------|-----|
| Import a round | Admin page (`/admin` on the frontend), or `migrate` CLI then reload |
| Reload after changing data or room files | Admin page "Reload", `POST /api/admin/reload`, or `kill -HUP <pid>` / `docker compose kill -s HUP backend` |
| Back up | Automatic with `BACKUP_DIR`; on demand from the admin page or `go run ./cmd/migrate backup <file>` |
| Deploy | Every push to `main` publishes `ghcr.io/openkku/cp-examseat-backend:latest` (and `:sha-…`, `:X.Y.Z` for `vX.Y.Z` tags); `docker compose pull && docker compose up -d` |

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
