package requestdecorator

import (
	"fmt"
	"net/http"

	"github.com/redhatinsights/insights-ingress-http-client/insights/requestauthorizer"
)

// RequestConfig An interface for handling the configuration request headers
type RequestConfig interface {
	GetUserAgent() string
}

// BasicRequestConfig A standard implementation for configuring request headers
type BasicRequestConfig struct {
	OperatorName   string
	OperatorCommit string
	ClusterID      string
}

// GetUserAgent Get the string for the user agent header string
func (b BasicRequestConfig) GetUserAgent() string {
	return fmt.Sprintf("%s/%s cluster/%s", b.OperatorName, b.OperatorCommit, b.ClusterID)
}

// RequestDecorator An object for updating the headers of a request
type RequestDecorator struct {
	config     *RequestConfig
	authorizer *requestauthorizer.RequestAuthorizer
}

// New Initialize a new request decorator object
func New(config *RequestConfig, authorizer *requestauthorizer.RequestAuthorizer) *RequestDecorator {
	return &RequestDecorator{
		config:     config,
		authorizer: authorizer,
	}
}

// UpdateHeaders Adds user agent and content type headers to a request
func (rd *RequestDecorator) UpdateHeaders(req *http.Request, contentType string) {
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if rd.config != nil {
		config := *(rd.config)
		useragent := config.GetUserAgent()
		if useragent != "" {
			req.Header.Set("User-Agent", useragent)
		}
	}
	if rd.authorizer != nil {
		authorizer := *(rd.authorizer)
		authorizer.SetAuthorization(req)
	}
}
