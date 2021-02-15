package insightsclient

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net"
	"net/http"
	"net/textproto"
	"os"
	"strconv"
	"time"

	"k8s.io/client-go/transport"
	"k8s.io/component-base/metrics"
	"k8s.io/component-base/metrics/legacyregistry"

	"k8s.io/klog"

	"github.com/redhatinsights/insights-ingress-http-client/authorizer"
	"github.com/redhatinsights/insights-ingress-http-client/insights/proxycontrol"
	"github.com/redhatinsights/insights-ingress-http-client/insights/requestdecorator"
	"github.com/redhatinsights/insights-ingress-http-client/insights/source"
)

const (
	responseBodyLogLen = 1024
)

// Client The client structure for making requests to cloud.redhat.com
type Client struct {
	client       *http.Client
	maxBytes     int64
	certPath     string
	metricsName  string
	proxyCtrl    *proxycontrol.ProxyControl
	reqDecorator *requestdecorator.RequestDecorator
}

// ErrWaitingForVersion An error due to cluster version responding slowly
var ErrWaitingForVersion = fmt.Errorf("waiting for the cluster version to be loaded")

// ErrObtainingForVersion An error due to cluster version client collection
var ErrObtainingForVersion = fmt.Errorf("waiting for the cluster version to be loaded")

// New Initialize a new client object
func New(client *http.Client, maxBytes int64, certPath string, metricsName string, proxyCtrl *proxycontrol.ProxyControl, reqDecorator *requestdecorator.RequestDecorator) *Client {
	if client == nil {
		client = &http.Client{}
	}
	if maxBytes == 0 {
		maxBytes = 10 * 1024 * 1024
	}
	return &Client{
		client:       client,
		maxBytes:     maxBytes,
		certPath:     certPath,
		metricsName:  metricsName,
		proxyCtrl:    proxyCtrl,
		reqDecorator: reqDecorator,
	}
}

func (c *Client) getTrustedCABundle() (*x509.CertPool, error) {
	caBytes, err := ioutil.ReadFile(c.certPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if len(caBytes) == 0 {
		return nil, nil
	}
	certs := x509.NewCertPool()
	if ok := certs.AppendCertsFromPEM(caBytes); !ok {
		return nil, errors.New("error loading cert pool from ca data")
	}
	return certs, nil
}

// clientTransport creates new http.Transport with either system or configured Proxy
func (c *Client) clientTransport() http.RoundTripper {
	prxy := *(c.proxyCtrl)
	clientTransport := &http.Transport{
		Proxy: prxy.NewSystemOrConfiguredProxy(),
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout: 10 * time.Second,
		DisableKeepAlives:   true,
	}

	// get the cluster proxy trusted CA bundle in case the proxy need it
	rootCAs, err := c.getTrustedCABundle()
	if err != nil {
		klog.Errorf("Failed to get proxy trusted CA: %v", err)
	}
	if rootCAs != nil {
		clientTransport.TLSClientConfig = &tls.Config{}
		clientTransport.TLSClientConfig.RootCAs = rootCAs
	}

	return transport.DebugWrappers(clientTransport)
}

// SetupRequest creates a new request, adds headers to request object for communication, and returns the request
func (c *Client) SetupRequest(ctx context.Context, method, uri string, body *bytes.Buffer, contentType string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, uri, body)
	if err != nil {
		return nil, fmt.Errorf("could not create request: %v", err)
	}

	if req.Header == nil {
		req.Header = make(http.Header)
	}
	c.reqDecorator.UpdateHeaders(req, contentType)

	// dynamically set the proxy environment
	c.client.Transport = c.clientTransport()

	return req, nil
}

// GetMultiPartBodyAndHeaders Get multi-part body and headers for upload
func (c *Client) GetMultiPartBodyAndHeaders(req *http.Request, data source.Source) int64 {
	// set the content and content type
	var bytesRead int64
	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)
	filename := "payload.tar.gz"
	if data.Filename != "" {
		filename = data.Filename
	}
	go func() {
		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition", fmt.Sprintf(`form-data; name=%q; filename=%q`, "file", filename))
		h.Set("Content-Type", data.Type)
		fw, err := mw.CreatePart(h)
		if err != nil {
			pw.CloseWithError(err)
			return
		}
		r := &LimitedReader{R: data.Contents, N: c.maxBytes}
		n, err := io.Copy(fw, r)
		bytesRead = n
		if err != nil {
			pw.CloseWithError(err)
		}
		pw.CloseWithError(mw.Close())
	}()
	req.Body = pr
	return bytesRead
}

// Send Posts source data to an endpoint
func (c *Client) Send(ctx context.Context, endpoint string, data source.Source) error {
	req, err := c.SetupRequest(ctx, "POST", endpoint, nil, data.Type)
	if err != nil {
		return err
	}
	bytesRead := c.GetMultiPartBodyAndHeaders(req, data)
	klog.V(4).Infof("Uploading %s to %s", data.Type, req.URL.String())
	resp, err := c.client.Do(req)
	if err != nil {
		klog.V(4).Infof("Unable to build a request, possible invalid token: %v", err)
		// if the request is not build, for example because of invalid endpoint,(maybe some problem with DNS), we want to have record about it in metrics as well.
		counterRequestSend.WithLabelValues(c.metricsName, "0").Inc()
		return fmt.Errorf("unable to build request to connect to Insights server: %v", err)
	}

	requestID := resp.Header.Get("x-rh-insights-request-id")

	defer func() {
		if _, err := io.Copy(ioutil.Discard, resp.Body); err != nil {
			klog.Warningf("error copying body: %v", err)
		}
		if err := resp.Body.Close(); err != nil {
			klog.Warningf("Failed to close response body: %v", err)
		}
	}()

	counterRequestSend.WithLabelValues(c.metricsName, strconv.Itoa(resp.StatusCode)).Inc()

	if resp.StatusCode == http.StatusUnauthorized {
		klog.V(2).Infof("gateway server %s returned 401, x-rh-insights-request-id=%s", resp.Request.URL, requestID)
		return authorizer.Error{Err: fmt.Errorf("your Red Hat account is not enabled for remote support or your token has expired: %s", responseBody(resp))}
	}

	if resp.StatusCode == http.StatusForbidden {
		klog.V(2).Infof("gateway server %s returned 403, x-rh-insights-request-id=%s", resp.Request.URL, requestID)
		return authorizer.Error{Err: fmt.Errorf("your Red Hat account is not enabled for remote support")}
	}

	if resp.StatusCode == http.StatusBadRequest {
		return fmt.Errorf("gateway server bad request: %s (request=%s): %s", resp.Request.URL, requestID, responseBody(resp))
	}

	if resp.StatusCode >= 300 || resp.StatusCode < 200 {
		return fmt.Errorf("gateway server reported unexpected error code: %d (request=%s): %s", resp.StatusCode, requestID, responseBody(resp))
	}

	if len(requestID) > 0 {
		klog.V(2).Infof("Successfully reported id=%s x-rh-insights-request-id=%s, wrote=%d", data.ID, requestID, bytesRead)
	}

	return nil
}

func responseBody(r *http.Response) string {
	if r == nil {
		return ""
	}
	body, _ := ioutil.ReadAll(r.Body)
	if len(body) > responseBodyLogLen {
		body = body[:responseBodyLogLen]
	}
	return string(body)
}

var (
	counterRequestSend = metrics.NewCounterVec(&metrics.CounterOpts{
		Name: "insightsclient_request_send_total",
		Help: "Tracks the number of metrics sends",
	}, []string{"client", "status_code"})
)

func init() {
	err := legacyregistry.Register(
		counterRequestSend,
	)
	if err != nil {
		fmt.Println(err)
	}

}
