package admin

// InvalidInputError is a request the admin must correct; Message says how.
type InvalidInputError struct {
	Message string
}

func (e *InvalidInputError) Error() string { return e.Message }

func invalid(message string) error { return &InvalidInputError{Message: message} }

// ReloadError means a change was saved but reloading the in-memory state failed.
type ReloadError struct {
	Err error
}

func (e *ReloadError) Error() string { return e.Err.Error() }

func (e *ReloadError) Unwrap() error { return e.Err }
