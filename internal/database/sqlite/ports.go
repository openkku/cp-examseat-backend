package sqlite

import (
	"github.com/openkku/cp-examseat-backend/internal/admin"
	"github.com/openkku/cp-examseat-backend/internal/checkex"
	"github.com/openkku/cp-examseat-backend/internal/exam"
	"github.com/openkku/cp-examseat-backend/internal/ingest"
	"github.com/openkku/cp-examseat-backend/internal/round"
	"github.com/openkku/cp-examseat-backend/internal/stats"
)

// The ports DB implements. Another engine implements the same list.
var (
	_ admin.Repository    = (*DB)(nil)
	_ admin.DatasetPurger = (*DB)(nil)
	_ checkex.Repository  = (*DB)(nil)
	_ exam.Repository     = (*DB)(nil)
	_ ingest.Repository   = (*DB)(nil)
	_ round.Repository    = (*DB)(nil)
	_ stats.Repository    = (*DB)(nil)
)
