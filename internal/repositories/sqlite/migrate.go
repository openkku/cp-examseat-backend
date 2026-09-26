package sqlite

import (
	"context"
	"fmt"
	"log"
)

// migrate creates or upgrades the schema. It runs on one pinned connection
// because the legacy upgrade toggles foreign_keys, a per-connection setting.
func (d *Database) migrate(ctx context.Context) error {
	conn, err := d.db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire connection for migration: %w", err)
	}
	defer conn.Close()

	// Drop legacy flat exams table if it exists
	conn.ExecContext(ctx, "DROP TABLE IF EXISTS exams;")

	// Check if exam_sessions has 'date' column. If not (old date_start schema), drop tables to recreate cleanly.
	var hasDateCol bool
	rows, err := conn.QueryContext(ctx, "PRAGMA table_info(exam_sessions);")
	if err == nil {
		for rows.Next() {
			var cid int
			var name, ctype string
			var notnull, pk int
			var dfltVal any
			if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltVal, &pk); err == nil {
				if name == "date" {
					hasDateCol = true
				}
			}
		}
		rows.Close()
		if !hasDateCol {
			conn.ExecContext(ctx, "DROP TABLE IF EXISTS exam_seats;")
			conn.ExecContext(ctx, "DROP TABLE IF EXISTS session_labels;")
			conn.ExecContext(ctx, "DROP TABLE IF EXISTS exam_sessions;")
		}
	}

	schema := `
    CREATE TABLE IF NOT EXISTS round_info (
        id TEXT PRIMARY KEY,
        label TEXT
    );

    CREATE TABLE IF NOT EXISTS subjects (
        id TEXT,
        exam_round TEXT NOT NULL,
        name TEXT,
        PRIMARY KEY (id, exam_round),
        FOREIGN KEY (exam_round) REFERENCES round_info(id) ON DELETE CASCADE
    );
    CREATE INDEX IF NOT EXISTS idx_subjects_round ON subjects(exam_round);

    CREATE TABLE IF NOT EXISTS students (
        id TEXT PRIMARY KEY,
        branch TEXT
    );

    CREATE TABLE IF NOT EXISTS exam_sessions (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        exam_round TEXT NOT NULL,
        category TEXT NOT NULL DEFAULT 'IN_SCHEDULE',
        custom_id TEXT DEFAULT '',
        sheet TEXT NOT NULL,
        date TEXT NOT NULL,
        time_start TEXT NOT NULL,
        time_end TEXT,
        room TEXT NOT NULL,
        subject_id TEXT NOT NULL,
        section TEXT NOT NULL,
        note TEXT DEFAULT '',
        room_layout TEXT DEFAULT '',
        FOREIGN KEY (exam_round) REFERENCES round_info(id) ON DELETE CASCADE
    );
    CREATE INDEX IF NOT EXISTS idx_sessions_lookup ON exam_sessions(exam_round, room, date, time_start);
    CREATE INDEX IF NOT EXISTS idx_sessions_category ON exam_sessions(exam_round, category);
    CREATE INDEX IF NOT EXISTS idx_sessions_custom ON exam_sessions(exam_round, custom_id);

    CREATE TABLE IF NOT EXISTS session_labels (
        session_id INTEGER NOT NULL,
        label TEXT NOT NULL,
        PRIMARY KEY (session_id, label),
        FOREIGN KEY (session_id) REFERENCES exam_sessions(id) ON DELETE CASCADE
    );
    CREATE INDEX IF NOT EXISTS idx_session_labels_label ON session_labels(label);

    CREATE TABLE IF NOT EXISTS exam_seats (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        session_id INTEGER NOT NULL,
        student_id TEXT NOT NULL,
        seat TEXT NOT NULL,
        FOREIGN KEY (session_id) REFERENCES exam_sessions(id) ON DELETE CASCADE,
        FOREIGN KEY (student_id) REFERENCES students(id)
    );
    CREATE INDEX IF NOT EXISTS idx_seats_student ON exam_seats(student_id);
    CREATE INDEX IF NOT EXISTS idx_seats_session ON exam_seats(session_id);
    `

	if _, err := conn.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("error creating schema: %w", err)
	}

	// Check if existing exam_sessions table has FK to round_info. If not, perform schema migration.
	var hasFKToRoundInfo bool
	fkRows, err := conn.QueryContext(ctx, "PRAGMA foreign_key_list(exam_sessions);")
	if err == nil {
		for fkRows.Next() {
			var id, seq int
			var table, from, to, onUpdate, onDelete, match string
			if err := fkRows.Scan(&id, &seq, &table, &from, &to, &onUpdate, &onDelete, &match); err == nil {
				if table == "round_info" {
					hasFKToRoundInfo = true
				}
			}
		}
		fkRows.Close()
	}

	if !hasFKToRoundInfo {
		// Populate round_info for any existing orphan exam_rounds if any
		conn.ExecContext(ctx, "INSERT OR IGNORE INTO round_info (id, label) SELECT DISTINCT exam_round, exam_round FROM exam_sessions WHERE exam_round NOT IN (SELECT id FROM round_info);")
		conn.ExecContext(ctx, "INSERT OR IGNORE INTO round_info (id, label) SELECT DISTINCT exam_round, exam_round FROM subjects WHERE exam_round NOT IN (SELECT id FROM round_info);")

		conn.ExecContext(ctx, "PRAGMA foreign_keys = OFF;")

		migSQL := `
			CREATE TABLE IF NOT EXISTS exam_sessions_new (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				exam_round TEXT NOT NULL,
				category TEXT NOT NULL DEFAULT 'IN_SCHEDULE',
				custom_id TEXT DEFAULT '',
				sheet TEXT NOT NULL,
				date TEXT NOT NULL,
				time_start TEXT NOT NULL,
				time_end TEXT,
				room TEXT NOT NULL,
				subject_id TEXT NOT NULL,
				section TEXT NOT NULL,
				note TEXT DEFAULT '',
				room_layout TEXT DEFAULT '',
				FOREIGN KEY (exam_round) REFERENCES round_info(id) ON DELETE CASCADE
			);
			INSERT INTO exam_sessions_new SELECT id, exam_round, category, custom_id, sheet, date, time_start, time_end, room, subject_id, section, note, room_layout FROM exam_sessions;
			DROP TABLE exam_sessions;
			ALTER TABLE exam_sessions_new RENAME TO exam_sessions;
			CREATE INDEX IF NOT EXISTS idx_sessions_lookup ON exam_sessions(exam_round, room, date, time_start);
			CREATE INDEX IF NOT EXISTS idx_sessions_category ON exam_sessions(exam_round, category);
			CREATE INDEX IF NOT EXISTS idx_sessions_custom ON exam_sessions(exam_round, custom_id);

			CREATE TABLE IF NOT EXISTS subjects_new (
				id TEXT,
				exam_round TEXT NOT NULL,
				name TEXT,
				PRIMARY KEY (id, exam_round),
				FOREIGN KEY (exam_round) REFERENCES round_info(id) ON DELETE CASCADE
			);
			INSERT INTO subjects_new SELECT id, exam_round, name FROM subjects;
			DROP TABLE subjects;
			ALTER TABLE subjects_new RENAME TO subjects;
			CREATE INDEX IF NOT EXISTS idx_subjects_round ON subjects(exam_round);
		`
		if _, err := conn.ExecContext(ctx, migSQL); err != nil {
			log.Printf("⚠️ Schema migration error: %v", err)
		}
		conn.ExecContext(ctx, "PRAGMA foreign_keys = ON;")
	}

	return nil
}
