package ini

import (
	"fmt"
)

// ErrDelimiterNotFound indicates the error type of no delimiter is found which there should be one.
type ErrDelimiterNotFound struct {
	Line string
}

// IsErrDelimiterNotFound returns true if the given error is an instance of ErrDelimiterNotFound.
func IsErrDelimiterNotFound(err error) bool {
	_, ok := err.(ErrDelimiterNotFound)
	return ok
}

func (err ErrDelimiterNotFound) Error() string {
	return fmt.Sprintf("key-value delimiter not found: %s", err.Line)
}
