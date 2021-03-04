package requestauthorizer

import "net/http"

// RequestAuthorizer An interface for handling the authorization request header
type RequestAuthorizer interface {
	SetAuthorization(req *http.Request)
}
