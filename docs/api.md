# API Reference

All endpoints are `GET` and public. JSON errors have the shape
`{"error": "<message>"}`. Responses are compressed (zstd, br, gzip or
deflate) according to `Accept-Encoding`.

| Endpoint | Controller | Description |
|----------|------------|-------------|
| `GET /healthz` | `controllers.Health` | Liveness probe |
| `GET /api/rounds` | `RoundController.Index` | Exam rounds, newest first |
| `GET /api/exam` | `ExamController.Show` | A student's exam schedule |
| `GET /api/calendar/{id}` | `CalendarController.Show` | iCalendar feed of a student |
| `GET /api/explore` | `ExploreController.Index` | Roster of one room and exam slot |
| `GET /api/options` | `ExploreController.Options` | Cascading explorer filter options |
| `GET /api/stats` | `StatsController.Index` | Analytics dashboard |
| `GET /api/room` | `RoomController.Index` | Room layouts and images |
| `GET /room/image/*` | `RoomController.Image` | Room images from `$DATA_DIR/room/image` |
| `/api/admin/*` | `AdminController` | Import and data management (see [Admin API](#admin-api)) |

Every route except `/healthz` is rate limited per client; over the limit
the API answers `429 {"error": "Too many requests, please slow down"}` with a
`Retry-After` header (seconds).

---

## `GET /api/rounds`

```json
[{"id": "mid_1_2569", "label": "กลางภาค 1/2569"}]
```

## `GET /api/exam?id={studentId}&round={roundId}`

`id` may contain a dash (`653380123-4`); non-digits are ignored.

| Status | Body |
|--------|------|
| 200 | `ExamSchedule[]` ordered by date and start time |
| 400 | `Student ID and Round are required` |
| 404 | `No exam schedules found` |

`ExamSchedule`:

```json
{
  "sheet": "S1",
  "date": "2026-09-01",
  "time": "08.30-11.30",
  "room": "CP.9127",
  "subject": "CP1001",
  "subject_name": "Intro",
  "section": "1",
  "student_id": "6533801234",
  "seat": "A1",
  "note": "",
  "branch": "CP-CS",
  "labels": ["LAB", "Lab"],
  "room_layout": "CP9127",
  "custom_id": "LAB_X"
}
```

`labels`, `room_layout` and `custom_id` are omitted when empty.

## `GET /api/calendar/{id}` · `GET /api/calendar/{id}.ics`

Returns `text/calendar` as an attachment named `exams-{id}.ics` covering every
round, refreshed by calendar clients every 12 hours. Exams without a date or
start time are skipped.

| Status | Body |
|--------|------|
| 400 | `Student ID is required` |
| 404 | `No exam schedules found for this student` |

## `GET /api/explore?round=&room=&date=&time=[&seat=]`

`time` matches either the start time (`13.00`) or the full range (`08.30-11.30`).

| Status | Body |
|--------|------|
| 200 | `ExamSchedule[]` ordered by seat |
| 400 | `Round, Room, Date, and Time parameters are required` |
| 404 | `no exams found` |

## `GET /api/options?type={dates|times|rooms}&round=[&date=&time=]`

Lists the distinct values for the explorer filters. When room layouts are
configured, only rooms that have a layout are considered.

| `type` | Extra parameters | Example |
|--------|------------------|---------|
| `dates` | — | `["2026-09-01", "2026-09-03"]` |
| `times` | `date` | `["08.30-11.30", "13.00-16.00"]` |
| `rooms` | `date`, `time` (optional) | `["CP.9127"]` |

| Status | Body |
|--------|------|
| 400 | `Round parameter is required` · `invalid mode` |
| 404 | `No options found` |

## `GET /api/stats`

```json
{
  "options": [{"id": "global", "label": "Global View (All Rounds)"}, {"id": "mid_1_2569", "label": "กลางภาค 1/2569"}],
  "stats": {
    "global": {
      "student_count": 2, "room_count": 2, "occupancy": 0, "total_seatings": 3,
      "avg_exams_per_student": 1.5, "back_to_back_count": 0,
      "top_subjects": [{"code": "CP1001", "name": "Intro", "count": 2}],
      "year_distribution": [{"year": "65", "count": 2}],
      "timeslot_distribution": [{"time": "08.30-11.30", "count": 2}],
      "room_utilization": [{"room": "CP.9127", "seat_count": 2, "days_active": 1, "subjects": 1}],
      "department_breakdown": [{"department": "CP (Computer)", "seatings": 2, "subjects": 1}],
      "peak_day": {"date": "2026-09-01", "count": 2, "students": 2, "rooms": 1}
    }
  }
}
```

Computed once at startup; `503 Stats not ready` if that failed.

## `GET /api/room[?room=A,B|?room=A&room=B][&no_layout=true]`

Without `room`, every room with a loaded layout is returned.

```json
{
  "CP.9127": {
    "i_layout": "/room/image/CP.9127.jpg",
    "i_map": "https://maps.example/cp9127",
    "i_images": ["/room/image/CP.9127-1.jpg"],
    "layout": [{"type": "column", "label": "A", "items": [{"type": "seats", "char": "A", "count": 30}]}],
    "frontLabel": "Whiteboard",
    "backLabel": "Door"
  }
}
```

`title`, `description`, `lat` and `lng` are included when set in
`metadata.json`. `layout`, `frontLabel` and `backLabel` are omitted with `no_layout=true`
(`frontLabel`/`backLabel` also when the layout file has none).

| Status | Body |
|--------|------|
| 404 | `Rooms not found` |
| 405 | `Method Not Allowed` — more rooms requested than are configured |

## `GET /room/image/{file}`

Serves `$DATA_DIR/room/image/{file}`; `404 Image not found` otherwise. Unused
when `IMAGE_BASE_URL` points image URLs at a CDN.

---

## Admin API

Enabled only when `ADMIN_TOKEN` is set; otherwise every `/api/admin` path is
404. Every request needs `Authorization: Bearer <ADMIN_TOKEN>` (401
otherwise). Changes are applied immediately: each write reloads rounds,
rooms and statistics and clears the response caches. With `BACKUP_DIR` set,
imports and deletes take a safety backup first.

| Endpoint | Description |
|----------|-------------|
| `GET /api/admin/status` | `{rooms, rooms_configured, last_reload, backups_enabled, backups: [{name, size, created_at}], max_upload_mb}` |
| `GET /api/admin/rounds` | `[{id, label, seats, students, in_schedule_seats, custom_datasets: [{id, seats}]}]` |
| `PATCH /api/admin/rounds/{round}` | Rename: `{"label": "..."}` |
| `DELETE /api/admin/rounds/{round}` | Delete a round and everything in it |
| `DELETE /api/admin/rounds/{round}?custom_id=ID` | Delete one out-of-schedule dataset |
| `POST /api/admin/import` | `multipart/form-data`: `file` (.xlsx, .xls, .pdf, .json), `round`, optional `display`, `labels` (comma-separated), `room_layout`, `custom_id`. Same replace semantics as `cmd/migrate`. Returns `{round, seats, reloaded_at}` |
| `POST /api/admin/reload` | Reload data and room files from disk |
| `POST /api/admin/backups` | Write a backup into `BACKUP_DIR` (409 if not configured) |
| `GET /api/admin/backup` | Download a fresh snapshot of `exams.db` |

IDs (`round`, `custom_id`, `room_layout`) may contain letters, digits, `_`
and `-` (max 64). Import errors return 422 with the extractor's message;
uploads over `MAX_UPLOAD_MB` return 413.
