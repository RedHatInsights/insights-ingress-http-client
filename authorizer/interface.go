package authorizer

// Error structure for the authorizer interface
type Error struct {
	Err error
}

// Error Obtains the error string from the error object
func (e Error) Error() string {
	return e.Err.Error()
}

// IsAuthorizationError Based on the error type it returns true
// if it is an authorization error and false otherwise
func IsAuthorizationError(err error) bool {
	switch err.(type) {
	case Error:
		return true
	default:
		return false
	}
}
