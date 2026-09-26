# Environment Variables Reference

This document describes the environment variables supported by the CP Exam Seat backend.

| Variable | Default | Used by |
|----------|---------|---------|
| `PORT` | `8080` | `cmd/server` |
| `DATA_DIR` | `data` | `cmd/server`, `cmd/migrate`, `cmd/room-config` |
| `IMAGE_BASE_URL` | *(empty)* | `cmd/server` |
| `CORS_ALLOWED_ORIGINS` | *(empty)* | `cmd/server` |

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
