package insightsclient

import (
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
	"net/url"
	"os"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/version"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/transport"

	"k8s.io/klog"

	configv1 "github.com/openshift/api/config/v1"
	configv1client "github.com/openshift/client-go/config/clientset/versioned/typed/config/v1"
	apimachineryversion "k8s.io/apimachinery/pkg/version"

	"github.com/redhatinsights/insights-ingress-http-client/authorizer"
)

const (
	responseBodyLogLen = 1024
)

// Client The client structure for making requests to cloud.redhat.com
type Client struct {
	client       *http.Client
	maxBytes     int64
	operatorName string
	mimeType     string

	authorizer     Authorizer
	kubeConfig     *rest.Config
	clusterVersion *configv1.ClusterVersion
}

// Authorizer An interface for adding authorization and proxy information the request
type Authorizer interface {
	Authorize(req *http.Request) error
	NewSystemOrConfiguredProxy() func(*http.Request) (*url.URL, error)
}

// Source An object for storing data of multiple types
type Source struct {
	ID       string
	Type     string
	Contents io.Reader
}

// ErrWaitingForVersion An error due to cluster version responding slowly
var ErrWaitingForVersion = fmt.Errorf("waiting for the cluster version to be loaded")

// ErrObtainingForVersion An error due to cluster version client collection
var ErrObtainingForVersion = fmt.Errorf("waiting for the cluster version to be loaded")

// New Initialize a new client object
func New(client *http.Client, maxBytes int64, operatorName string, mimeType string, authorizer Authorizer, kubeConfig *rest.Config) *Client {
	if client == nil {
		client = &http.Client{}
	}
	if maxBytes == 0 {
		maxBytes = 10 * 1024 * 1024
	}
	return &Client{
		client:       client,
		maxBytes:     maxBytes,
		operatorName: operatorName,
		mimeType:     mimeType,
		authorizer:   authorizer,
		kubeConfig:   kubeConfig,
	}
}

// Get Cluster Version via API
func (c *Client) getClusterVersion() (*configv1.ClusterVersion, error) {
	if c.clusterVersion != nil {
		return c.clusterVersion, nil
	}
	ctx := context.Background()
	client, err := configv1client.NewForConfig(c.kubeConfig)
	if err != nil {
		return nil, err
	}
	cv, err := client.ClusterVersions().Get(ctx, "version", metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	c.clusterVersion = cv
	return cv, nil
}

func getTrustedCABundle() (*x509.CertPool, error) {
	caBytes, err := ioutil.ReadFile("/var/run/configmaps/trusted-ca-bundle/ca-bundle.crt") //Should this be parameterized?
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
func clientTransport(authorizer Authorizer) http.RoundTripper {
	clientTransport := &http.Transport{
		Proxy: authorizer.NewSystemOrConfiguredProxy(),
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout: 10 * time.Second,
		DisableKeepAlives:   true,
	}

	// get the cluster proxy trusted CA bundle in case the proxy need it
	rootCAs, err := getTrustedCABundle()
	if err != nil {
		klog.Errorf("Failed to get proxy trusted CA: %v", err)
	}
	if rootCAs != nil {
		clientTransport.TLSClientConfig = &tls.Config{}
		clientTransport.TLSClientConfig.RootCAs = rootCAs
	}

	return transport.DebugWrappers(clientTransport)
}

func userAgent(operatorName string, releaseVersionEnv string, v apimachineryversion.Info, cv *configv1.ClusterVersion) string {
	gitVersion := v.GitVersion
	// If the RELEASE_VERSION is set in pod, use it
	if releaseVersionEnv != "" {
		gitVersion = releaseVersionEnv
	}
	gitVersion = fmt.Sprintf("%s-%s", gitVersion, v.GitCommit)
	return fmt.Sprintf("%s/%s cluster/%s", operatorName, gitVersion, cv.Spec.ClusterID)
}

// GetMimeType Accessor for the client mimeType
func (c *Client) GetMimeType() string {
	return c.mimeType
}

// Send Posts source data to an endpoint
func (c *Client) Send(ctx context.Context, endpoint string, source Source) error {
	cv, err := c.getClusterVersion()
	if err != nil {
		return ErrObtainingForVersion
	}
	if cv == nil {
		return ErrWaitingForVersion
	}

	req, err := http.NewRequest("POST", endpoint, nil)
	if err != nil {
		return err
	}

	if req.Header == nil {
		req.Header = make(http.Header)
	}
	releaseVersionEnv := os.Getenv("RELEASE_VERSION")
	ua := userAgent(c.operatorName, releaseVersionEnv, version.Get(), cv)
	req.Header.Set("User-Agent", ua)
	if err := c.authorizer.Authorize(req); err != nil {
		return err
	}

	var bytesRead int64
	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	go func() {
		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition", fmt.Sprintf(`form-data; name=%q; filename=%q`, "file", "payload.tar.gz"))
		h.Set("Content-Type", source.Type)
		fw, err := mw.CreatePart(h)
		if err != nil {
			pw.CloseWithError(err)
			return
		}
		r := &LimitedReader{R: source.Contents, N: c.maxBytes}
		n, err := io.Copy(fw, r)
		bytesRead = n
		if err != nil {
			pw.CloseWithError(err)
		}
		pw.CloseWithError(mw.Close())
	}()

	req = req.WithContext(ctx)
	req.Body = pr

	// dynamically set the proxy environment
	c.client.Transport = clientTransport(c.authorizer)

	klog.V(4).Infof("Uploading %s to %s", source.Type, req.URL.String())
	resp, err := c.client.Do(req)
	if err != nil {
		klog.V(4).Infof("Unable to build a request, possible invalid token: %v", err)
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
		klog.V(2).Infof("Successfully reported id=%s x-rh-insights-request-id=%s, wrote=%d", source.ID, requestID, bytesRead)
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
