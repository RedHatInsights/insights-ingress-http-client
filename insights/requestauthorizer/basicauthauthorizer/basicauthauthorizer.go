package basicauthauthorizer

import "net/http"

// BasicAuthAuthorizer A standard implementation for basic authorization
type BasicAuthAuthorizer struct {
	username string
	password string
}

// New Initialize a new basic auth authorizer object
func New(username string, password string) *BasicAuthAuthorizer {
	return &BasicAuthAuthorizer{
		username: username,
		password: password,
	}
}

// SetAuthorization Sets the authorization header for basic auth
func (b *BasicAuthAuthorizer) SetAuthorization(req *http.Request) {
	req.SetBasicAuth(b.username, b.password)
}
