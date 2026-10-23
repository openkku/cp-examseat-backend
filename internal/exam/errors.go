package exam

import (
	"errors"

	"github.com/openkku/cp-examseat-backend/internal/checkex"
)

// ErrInvalidOptionMode is returned for an OptionsQuery with an unknown Mode.
var ErrInvalidOptionMode = errors.New("invalid mode")

// RetryLater reports whether err is an upstream answer the visitor should
// retry later rather than a failure.
func RetryLater(err error) bool {
	var queued *checkex.LookupQueuedError
	return errors.As(err, &queued) || errors.Is(err, checkex.ErrLookupLimited) || errors.Is(err, checkex.ErrLookupBusy)
}
