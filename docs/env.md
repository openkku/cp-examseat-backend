# Environment Variables Reference

This document describes the environment variables supported by the CP Exam Seat backend.

| Variable | Default | Used by |
|----------|---------|---------|
| `PORT` | `8080` | `cmd/server` |
| `DATA_DIR` | `data` | `cmd/server`, `cmd/migrate`, `cmd/room-config` |
| `IMAGE_BASE_URL` | *(empty)* | `cmd/server` |
| `CORS_ALLOWED_ORIGINS` | *(empty)* | `cmd/server` |
| `ROOM_CONFIG_ADDR` | `127.0.0.1:8081` | `cmd/room-config` |
| `RATE_LIMIT_RPS` | `10` | `cmd/server` |
| `RATE_LIMIT_BURST` | `40` | `cmd/server` |
| `TRUSTED_PROXIES` | `loopback,private` | `cmd/server` |
| `ADMIN_TOKEN` | *(empty)* | `cmd/server` |
| `MAX_UPLOAD_MB` | `20` | `cmd/server` |
| `BACKUP_DIR` | *(empty)* | `cmd/server` |
| `BACKUP_INTERVAL` | `24h` | `cmd/server` |
| `BACKUP_KEEP` | `7` | `cmd/server` |
| `SCRAPLER_URL` | `http://127.0.0.1:8090` | `cmd/server` |
| `CHECKEX_ROUNDS` | *(empty)* | `cmd/server` |
| `CHECKEX_ROUND_LABELS` | *(derived)* | `cmd/server` |
| `CHECKEX_ID_PREFIXES` | `68,69` | `cmd/server` |
| `CHECKEX_NOTFOUND_TTL` | `6h` | `cmd/server` |
| `CHECKEX_NOTFOUND_PER_IP` | `5` | `cmd/server` |
| `CHECKEX_NOTFOUND_WINDOW` | `1h` | `cmd/server` |

---

## 1. `PORT`

The TCP port the API server listens on.

---

## 2. `DATA_DIR`

The directory path where all runtime dynamic configurations, sqlite database files, and local image assets are stored.

- **Default Value**: `data` (resolves relative to the server's working directory).
- **Purpose**: Decouples the application code from user/school-specific runtime data. It also simplifies volume mapping inside Docker containers.

### Expected Directory Structure

```
$DATA_DIR/
├── exams.db           # Private SQLite database containing student schedules
└── room/
    ├── metadata.json  # Catalog of room properties, blueprints, and reference photos
    ├── map/           # Seating map layout grids (JSON files)
    │   ├── CP9127.json
    │   └── SC1101.json
    └── image/         # Static layout drawings & photos (served locally if IMAGE_BASE_URL is unset)
        ├── CP.9127.jpg
        └── SC.1101.jpg
```

---

## 3. `IMAGE_BASE_URL`

The base URL prefix for layout blueprints and photos when serving them from an external CDN or static asset storage (e.g., Cloudflare R2, AWS S3, or Google Cloud Storage) instead of the Go application server.

- **Default Value**: Empty (unset).
- **Purpose**: Offloads heavy static image downloads from the Go backend directly to your CDN.

### Routing Behavior

- **When Unset (Local serving)**: The server returns relative URL paths (e.g. `/room/image/CP.9127.jpg`) in the API responses. The client requests the image from the Go server (through the frontend's `/room/image/*` proxy), which serves it from `$DATA_DIR/room/image/CP.9127.jpg`.
- **When Set (CDN / External serving)**: The server prepends `IMAGE_BASE_URL` to all layout image and reference photo URLs returned by the API (e.g. `https://cdn.example.com/room/image/CP.9127.jpg`). The client requests the image directly from the CDN.

---

## 4. `CORS_ALLOWED_ORIGINS`

Comma-separated list of origins allowed to call the API from a browser on another origin, or `*` for any origin.

- **Default Value**: Empty — no CORS headers are sent.
- **When to set it**: Only when the browser calls the backend directly (the frontend's `NEXT_PUBLIC_API_BASE_URL` points at the backend). The default frontend setup proxies `/api/*` through Next.js on the same origin and does not need CORS.

```bash
CORS_ALLOWED_ORIGINS="https://exam.example.com,https://staging.example.com"
```

---

## 5. `ROOM_CONFIG_ADDR`

Listen address of the room-config editor. It defaults to loopback because the
tool writes files and has no authentication; it also refuses requests whose
`Host` is not a loopback name (DNS rebinding) and cross-site writes (CSRF).

---

## 6. Rate limiting: `RATE_LIMIT_RPS`, `RATE_LIMIT_BURST`, `TRUSTED_PROXIES`

Each client IP gets a token bucket refilled at `RATE_LIMIT_RPS` requests per
second holding up to `RATE_LIMIT_BURST` requests; beyond that the API answers
`429 Too Many Requests` with `Retry-After`. `RATE_LIMIT_RPS=0` disables it.
`/healthz` is never limited.

Behind the Next.js frontend every request arrives from the frontend's
address, so the client is read from `X-Forwarded-For` — but only when the
direct peer is listed in `TRUSTED_PROXIES` (IPs, CIDRs, or the shorthands
`loopback` and `private`). A client talking to the backend directly cannot
spoof its address. If the frontend reaches the backend over a public
address, add that address here.

Next.js forwards the `X-Forwarded-For` header it receives without appending
the client address, so the frontend must itself sit behind a reverse proxy
that sets the header (see the frontend README, "Deployment"). At most
100,000 client buckets are tracked; past that, new clients share one bucket
so a flood of spoofed addresses cannot exhaust memory.

---

## 7. Admin API: `ADMIN_TOKEN`, `MAX_UPLOAD_MB`

Setting `ADMIN_TOKEN` enables `/api/admin/*` (see [api.md](api.md#admin-api)),
used by the frontend's `/admin` page. Requests must send
`Authorization: Bearer <ADMIN_TOKEN>`. Use a long random value, e.g.
`openssl rand -hex 32`. Uploads are capped at `MAX_UPLOAD_MB` megabytes.

---

## 8. Backups: `BACKUP_DIR`, `BACKUP_INTERVAL`, `BACKUP_KEEP`

With `BACKUP_DIR` set the server writes a consistent snapshot of `exams.db`
(`VACUUM INTO`) every `BACKUP_INTERVAL` (Go duration, e.g. `6h`) and before
every admin import or delete, keeping the newest `BACKUP_KEEP` files named
`exams-YYYYMMDD-HHMMSS.mmm.db`. Put it on a different volume from `DATA_DIR`.
To restore, stop the server and copy a backup over `exams.db`.

---

## 9. On-demand lookup from checkex: `CHECKEX_*`, `SCRAPLER_URL`

From 1/2569 the faculty publishes seats only on
`checkex.computing.kku.ac.th`, which is reachable only through KKU's VPN and
looks up one student ID at a time. For rounds listed in `CHECKEX_ROUNDS`, the
backend fetches a student's seats the first time someone looks them up,
stores them in `exams.db` and serves them from the database after that.

The scraping itself runs in a separate service, **scrapler**
([openkku/akhsakov-scrapler](https://github.com/openkku/akhsakov-scrapler)),
which sits in the VPN's network namespace. scrapler owns the upstream pacing,
the lookup queue and the back-off; the backend owns the database cache, the
round list, the ID policy and the per-visitor not-found limit. The whole
stack (VPN, scrapler, backend, frontend) is in
[openkku/cp-examseat-deployment](https://github.com/openkku/cp-examseat-deployment).

| Variable | Default | Purpose |
|---|---|---|
| `CHECKEX_ROUNDS` | empty (feature off) | Comma-separated round IDs served by checkex, e.g. `final_1_2569`. Other rounds never go upstream. |
| `CHECKEX_ROUND_LABELS` | derived (`final_1_2569` → `ปลายภาค 1/2569`) | Optional `id=label;id=label` overrides. An existing label (e.g. renamed in `/admin`) is kept. |
| `CHECKEX_ID_PREFIXES` | `68,69` | Only 10-digit IDs starting with these are sent upstream. **Bump to `69,70` for 2570** (see `TODO(2570)` in `internal/config/config.go`). |
| `CHECKEX_NOTFOUND_TTL` | `6h` | How long a "not found" answer is trusted before that ID may be asked again. |
| `CHECKEX_NOTFOUND_PER_IP` | `5` | "Not found" answers one visitor can cause per window. `0` turns the limit off. |
| `CHECKEX_NOTFOUND_WINDOW` | `1h` | Window of the limit above. The budget refills evenly over it (one slot every 12 minutes by default). |
| `SCRAPLER_URL` | `http://127.0.0.1:8090` | scrapler base URL. |

Behaviour:

- **Found:** fetched once per (round, student) and never refreshed. To force
  a refetch, delete the round (admin page) or purge it; the lookups recorded
  in `upstream_lookups` are cleared with it.
- **Already imported:** students with in-schedule seats imported another way
  (Excel, JSON) are never asked upstream. Custom (out-of-schedule) datasets
  don't count, since upstream still has the regular exams.
- **Queue:** a search waits for its turn in scrapler (up to scrapler's
  `SCRAPLER_MAX_WAIT`, 20s by default). If it isn't done by then,
  `/api/exam` answers `202` with `{"queued": true, "position": N,
  "retry_after": 5}`; the lookup keeps running in scrapler and the retry
  stores its answer. Identical concurrent lookups share one call.
- **Per-visitor limit:** real students look up IDs that exist, while scripts
  walking ID ranges mostly get "not found". After `CHECKEX_NOTFOUND_PER_IP`
  not-found answers in the window, a visitor's searches for IDs not stored
  yet get `429` and aren't sent to scrapler. IDs already in the database
  still work for them. The visitor is the client IP from section 6, so
  `TRUSTED_PROXIES` must be right, or every visitor shares one budget.
- **Errors** (scrapler or VPN down, unexpected page) are never cached, so
  the next request retries. The search shows "no data" meanwhile, and the
  calendar feed is served without caching.
- **Stats** (`/api/stats`) include fetched students after the next reload.

---

## Quick Configuration Examples

### Local Development (Default)
Run the API without any environment variables. It loads the database and configuration files relative to `./data/`:
```bash
go run ./cmd/server
```

### Local Development with Custom Data Directory
If you want to configure layouts in a separate local directory (e.g. a separate checkout of your private layouts repository):
```bash
DATA_DIR="../my-layouts-repo" go run ./cmd/server
```

### Production with CDN Offloading
Deploy the server in a Docker container mounting the data folder, and point it to your CDN storage:
```bash
docker run -d \
  -p 8080:8080 \
  -v ./my-local-data:/app/data \
  -e IMAGE_BASE_URL="https://cdn.yourdomain.com" \
  cp-examseat-backend:latest
```
