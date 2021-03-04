package insightsuploader

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"time"

	"k8s.io/klog"

	"github.com/redhatinsights/insights-ingress-http-client/authorizer"
	"github.com/redhatinsights/insights-ingress-http-client/config"
	"github.com/redhatinsights/insights-ingress-http-client/controllerstatus"
	"github.com/redhatinsights/insights-ingress-http-client/insights/insightsclient"
	"github.com/redhatinsights/insights-ingress-http-client/insights/source"
)

// Controller An object for processing an upload
type Controller struct {
	controllerstatus.Simple

	client       *insightsclient.Client
	configurator config.Configurator
}

// New Initialize a new Controller object
func New(client *insightsclient.Client, configurator config.Configurator) *Controller {
	return &Controller{
		Simple:       controllerstatus.Simple{Name: "insightsuploader"},
		configurator: configurator,
		client:       client,
	}
}

// Upload Execute the payload upload
func (c *Controller) Upload(ctx context.Context, data io.ReadCloser, mimeType string) {
	c.Simple.UpdateStatus(controllerstatus.Summary{Healthy: true})

	if c.client == nil {
		klog.Infof("No reporting possible without a configured client")
		return
	}

	enabled := c.configurator.IsEnabled()
	endpoint := c.configurator.GetEndpoint()

	if data == nil {
		klog.V(4).Infof("Nothing to report")
		return
	}
	defer data.Close()

	if enabled && len(endpoint) > 0 {
		// send the results
		start := time.Now()
		id := start.Format(time.RFC3339)
		klog.V(4).Infof("Uploading report at %s", start.Format(time.RFC3339))
		if err := c.client.Send(ctx, endpoint, source.Source{
			ID:       id,
			Type:     mimeType,
			Contents: data,
		}); err != nil {
			klog.V(2).Infof("Unable to upload report after %s: %v", time.Now().Sub(start).Truncate(time.Second/100), err)
			versionError := err == insightsclient.ErrWaitingForVersion || err == insightsclient.ErrObtainingForVersion
			if versionError {
				return
			}
			if authorizer.IsAuthorizationError(err) {
				c.Simple.UpdateStatus(controllerstatus.Summary{Operation: controllerstatus.Uploading,
					Reason: "NotAuthorized", Message: fmt.Sprintf("Reporting was not allowed: %v", err)})
				return
			}
			c.Simple.UpdateStatus(controllerstatus.Summary{Operation: controllerstatus.Uploading,
				Reason: "UploadFailed", Message: fmt.Sprintf("Unable to report: %v", err)})
			return
		}
		klog.V(4).Infof("Uploaded report successfully in %s", time.Now().Sub(start))
		c.Simple.UpdateStatus(controllerstatus.Summary{Healthy: true})
	} else {
		klog.V(4).Info("Display report that would be sent")
		// display what would have been sent (to ensure we always exercise source processing)
		if err := reportToLogs(data, klog.V(4)); err != nil {
			klog.Errorf("Unable to log upload: %v", err)
		}
		// we didn't actually report logs, so don't advance the report date
	}
}

func reportToLogs(source io.Reader, klog klog.Verbose) error {
	if !klog {
		return nil
	}
	gr, err := gzip.NewReader(source)
	if err != nil {
		return err
	}
	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		klog.Infof("Dry-run: %s %7d %s", hdr.ModTime.Format(time.RFC3339), hdr.Size, hdr.Name)
	}
	return nil
}
