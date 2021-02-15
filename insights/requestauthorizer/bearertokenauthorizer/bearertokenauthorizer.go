package bearertokenauthorizer

import (
	"fmt"
	"net/http"
)

// BearerTokenAuthorizer A standard implementation for bearer token authorization
type BearerTokenAuthorizer struct {
	token string
}

// New Initialize a new bearer token authorizer object
func New(token string) *BearerTokenAuthorizer {
	return &BearerTokenAuthorizer{
		token: token,
	}
}

// SetAuthorization Sets the authorization header for bearer token auth
func (b *BearerTokenAuthorizer) SetAuthorization(req *http.Request) {
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", b.token))
}
