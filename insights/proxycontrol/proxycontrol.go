package proxycontrol

import (
	"net/http"
	"net/url"

	knet "k8s.io/apimachinery/pkg/util/net"
)

// ProxyControl An interface for adding proxy information the request
type ProxyControl interface {
	NewSystemOrConfiguredProxy() func(*http.Request) (*url.URL, error)
}

// BasicProxyControl Used for request setup
type BasicProxyControl struct {
	// exposed for tests
	proxyFromEnvironment func(*http.Request) (*url.URL, error)
}

// NewSystemOrConfiguredProxy Used to setup request proxy
func (b BasicProxyControl) NewSystemOrConfiguredProxy() func(*http.Request) (*url.URL, error) {
	// defautl system proxy
	if b.proxyFromEnvironment == nil {
		b.proxyFromEnvironment = http.ProxyFromEnvironment
	}
	return knet.NewProxierWithNoProxyCIDR(b.proxyFromEnvironment)
}
