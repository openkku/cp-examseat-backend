package sqlite_test

import (
	admindb "github.com/openkku/cp-examseat-backend/internal/admin/database"
	checkexdb "github.com/openkku/cp-examseat-backend/internal/checkex/database"
	examdb "github.com/openkku/cp-examseat-backend/internal/exam/database"
	ingestdb "github.com/openkku/cp-examseat-backend/internal/ingest/database"
	platformdb "github.com/openkku/cp-examseat-backend/internal/platform/database"
	"github.com/openkku/cp-examseat-backend/internal/platform/database/sqlite"
	rounddb "github.com/openkku/cp-examseat-backend/internal/round/database"
	statsdb "github.com/openkku/cp-examseat-backend/internal/stats/database"
)

// These tests run every slice's SQL against the SQLite engine.

type (
	adminRepo   = admindb.Repo
	checkexRepo = checkexdb.Repo
	examRepo    = examdb.Repo
	ingestRepo  = ingestdb.Repo
	roundRepo   = rounddb.Repo
	statsRepo   = statsdb.Repo
)

// testDB is a SQLite database with the repository of every slice.
type testDB struct {
	*platformdb.DB
	*adminRepo
	*checkexRepo
	*examRepo
	*ingestRepo
	*roundRepo
	*statsRepo
}

// open opens (creating) the SQLite database at path.
func open(path string) (*testDB, error) {
	db, err := sqlite.Open(path)
	if err != nil {
		return nil, err
	}
	return &testDB{
		DB:          db,
		adminRepo:   admindb.New(db),
		checkexRepo: checkexdb.New(db),
		examRepo:    examdb.New(db),
		ingestRepo:  ingestdb.New(db),
		roundRepo:   rounddb.New(db),
		statsRepo:   statsdb.New(db),
	}, nil
}
