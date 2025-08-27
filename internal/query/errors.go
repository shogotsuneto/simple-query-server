package query

import "fmt"

// ClientError represents an error caused by invalid client input (should return 400)
type ClientError struct {
	message string
}

func (e *ClientError) Error() string {
	return e.message
}

// NewClientError creates a new client error
func NewClientError(message string) *ClientError {
	return &ClientError{message: message}
}

// NewClientErrorf creates a new client error with formatted message
func NewClientErrorf(format string, args ...interface{}) *ClientError {
	return &ClientError{message: fmt.Sprintf(format, args...)}
}

// IsClientError checks if an error is a client error
func IsClientError(err error) bool {
	_, ok := err.(*ClientError)
	return ok
}
