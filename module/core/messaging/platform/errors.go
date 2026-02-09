package platform

import "errors"

var (
	// ErrNonRetriable marks handler errors that should not be retried.
	ErrNonRetriable = errors.New("messaging non-retriable error")
)

// NonRetriable wraps an error as non-retriable.
func NonRetriable(err error) error {
	if err == nil {
		return ErrNonRetriable
	}

	return errors.Join(ErrNonRetriable, err)
}

// IsNonRetriable returns true when an error is marked as non-retriable.
func IsNonRetriable(err error) bool {
	return errors.Is(err, ErrNonRetriable)
}
