package insightsuploader

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"strconv"
	"time"

	"k8s.io/klog"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/redhatinsights/insights-ingress-http-client/authorizer"
	"github.com/redhatinsights/insights-ingress-http-client/config"
	"github.com/redhatinsights/insights-ingress-http-client/controllerstatus"
	"github.com/redhatinsights/insights-ingress-http-client/insights/insightsclient"
)

// Configurator An interface for managing the configuration data
type Configurator interface {
	Config() *config.Configuration
	ConfigChanged() (<-chan struct{}, func())
}

// Authorizer An interface for determining if an error is related to authorization
type Authorizer interface {
	IsAuthorizationError(error) bool
}

// Summarizer An interface for peforming an summary
type Summarizer interface {
	Summary(ctx context.Context, since time.Time) (io.ReadCloser, bool, error)
}

// StatusReporter An interface for providing stats around reporting
type StatusReporter interface {
	LastReportedTime() time.Time
	SetLastReportedTime(time.Time)
	SafeInitialStart() bool
	SetSafeInitialStart(s bool)
}

// Controller An object for processing a summary
type Controller struct {
	controllerstatus.Simple

	summarizer   Summarizer
	client       *insightsclient.Client
	configurator Configurator
	reporter     StatusReporter
}

// New Initialize a new Controller object
func New(summarizer Summarizer, client *insightsclient.Client, configurator Configurator, statusReporter StatusReporter) *Controller {
	return &Controller{
		Simple: controllerstatus.Simple{Name: "insightsuploader"},

		summarizer:   summarizer,
		configurator: configurator,
		client:       client,
		reporter:     statusReporter,
	}
}

func (c *Controller) upload(ctx context.Context, cfg *config.Configuration, start time.Time, sourceIndex int, source io.ReadCloser) error {
	endpoint := cfg.Endpoint
	id := start.Format(time.RFC3339) + "-" + strconv.Itoa(sourceIndex)
	if err := c.client.Send(ctx, endpoint, insightsclient.Source{
		ID:       id,
		Type:     c.client.GetMimeType(),
		Contents: source,
	}); err != nil {
		klog.V(2).Infof("Unable to upload report after %s: %v", time.Now().Sub(start).Truncate(time.Second/100), err)
		versionError := err == insightsclient.ErrWaitingForVersion || err == insightsclient.ErrObtainingForVersion
		if versionError {
			return err
		}
		c.reporter.SetSafeInitialStart(false)
		if authorizer.IsAuthorizationError(err) {
			c.Simple.UpdateStatus(controllerstatus.Summary{Operation: controllerstatus.Uploading,
				Reason: "NotAuthorized", Message: fmt.Sprintf("Reporting was not allowed: %v", err)})
			return err
		}
		c.Simple.UpdateStatus(controllerstatus.Summary{Operation: controllerstatus.Uploading,
			Reason: "UploadFailed", Message: fmt.Sprintf("Unable to report: %v", err)})
		return err
	}
	return nil
}

// Run Execute the payload upload
func (c *Controller) Run(ctx context.Context) {
	c.Simple.UpdateStatus(controllerstatus.Summary{Healthy: true})

	if c.client == nil {
		klog.Infof("No reporting possible without a configured client")
		return
	}

	// the controller periodically uploads results to the remote insights endpoint
	cfg := c.configurator.Config()
	_, cancelFn := c.configurator.ConfigChanged()
	defer cancelFn()

	enabled := cfg.Report
	endpoint := cfg.Endpoint
	lastReported := c.reporter.LastReportedTime()

	wait.Until(func() {
		// attempt to get a summary to send to the server
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()

		if len(endpoint) > 0 {
			start := time.Now()
			sourceIndex := 0
			var source io.ReadCloser
			var ok bool
			var err error
			for source, ok, err = c.summarizer.Summary(ctx, lastReported); err == nil && ok; source, ok, err = c.summarizer.Summary(ctx, lastReported) {
				defer source.Close()
				if enabled {
					// send the results
					klog.V(4).Infof("Uploading latest report since %s", lastReported.Format(time.RFC3339))
					err = c.upload(ctx, cfg, start, sourceIndex, source)
					if err != nil {
						return
					}
					c.reporter.SetSafeInitialStart(false)
					klog.V(4).Infof("Uploaded report successfully in %s", time.Now().Sub(start))
				} else {
					klog.V(4).Info("Display report that would be sent")
					// display what would have been sent (to ensure we always exercise source processing)
					if err := reportToLogs(source, klog.V(4)); err != nil {
						klog.Errorf("Unable to log upload: %v", err)
					}
					// we didn't actually report logs, so don't advance the report date
				}
				sourceIndex = sourceIndex + 1
			}
			if err != nil {
				c.Simple.UpdateStatus(controllerstatus.Summary{Reason: "SummaryFailed", Message: fmt.Sprintf("Unable to retrieve local data: %v", err)})
				return
			}
			if !ok {
				klog.V(4).Infof("Nothing to report since %s", lastReported.Format(time.RFC3339))
				return
			}
			if enabled && err == nil {
				lastReported = start.UTC()
				c.Simple.UpdateStatus(controllerstatus.Summary{Healthy: true})
			}
		}

		c.reporter.SetLastReportedTime(lastReported)
	}, 15*time.Second, ctx.Done())
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
