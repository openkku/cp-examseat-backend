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
go run ./cmd/room-config
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

## Documentation

- [API reference](docs/api.md)
- [Environment variables](docs/env.md)

## License

MIT — see [LICENSE](LICENSE).
