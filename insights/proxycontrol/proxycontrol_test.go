package proxycontrol

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"golang.org/x/net/http/httpproxy"
)

// nonCachedProxyFromEnvironment creates Proxier if Proxy is set. It uses always fresh Env
func nonCachedProxyFromEnvironment() func(*http.Request) (*url.URL, error) {
	return func(req *http.Request) (*url.URL, error) {
		return httpproxy.FromEnvironment().ProxyFunc()(req.URL)
	}
}

func TestProxy(tt *testing.T) {
	testCases := []struct {
		Name       string
		EnvValues  map[string]interface{}
		RequestURL string
		ProxyURL   string
	}{
		{
			Name:       "No env set, no specific proxy",
			EnvValues:  map[string]interface{}{"HTTP_PROXY": nil},
			RequestURL: "http://google.com",
			ProxyURL:   "",
		},
		{
			Name:       "Env set, no specific proxy",
			EnvValues:  map[string]interface{}{"HTTP_PROXY": "proxy.to"},
			RequestURL: "http://google.com",
			ProxyURL:   "http://proxy.to",
		},
		{
			Name:       "Env set with HTTPS, no specific proxy",
			EnvValues:  map[string]interface{}{"HTTPS_PROXY": "secproxy.to"},
			RequestURL: "https://google.com",
			ProxyURL:   "http://secproxy.to",
		},
	}
	for _, tcase := range testCases {
		tc := tcase
		tt.Run(tc.Name, func(t *testing.T) {
			for k, v := range tc.EnvValues {
				defer SafeRestoreEnv(k)()
				// nil will indicate the need to unset Env
				if v != nil {
					vv := v.(string)
					os.Setenv(k, vv)
				} else {
					os.Unsetenv(k)
				}
			}

			b := BasicProxyControl{proxyFromEnvironment: nonCachedProxyFromEnvironment()}
			p := b.NewSystemOrConfiguredProxy()
			req := httptest.NewRequest("GET", tc.RequestURL, nil)
			url, err := p(req)

			if err != nil {
				t.Fatalf("unexpected err %s", err)
			}
			if (tc.ProxyURL == "" && url != nil) ||
				(len(tc.ProxyURL) > 0 && (url == nil || tc.ProxyURL != url.String())) {
				t.Fatalf("Unexpected value of Proxy Url. Test %s Expected Url %s Received Url %s", tc.Name, tc.ProxyURL, url)
			}
		})
	}
}

func SafeRestoreEnv(key string) func() {
	originalVal, wasSet := os.LookupEnv(key)
	return func() {
		if !wasSet {
			fmt.Printf("unsetting key %s", key)
			os.Unsetenv(key)
		} else {
			fmt.Printf("restoring key %s", key)
			os.Setenv(key, originalVal)
		}
	}
}
