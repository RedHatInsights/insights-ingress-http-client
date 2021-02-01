package summarize

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"k8s.io/klog"
)

// ReportingType describes how report uploads will be handled.
type ReportingType string

const (
	// Latest only sends the latest payload
	Latest ReportingType = "latest"

	// AllSinceLastReporting Sends all files since last successful report
	AllSinceLastReporting ReportingType = "all"
)

// BasicDiskSummarizer provides basic implementation for Summarizer interface
type BasicDiskSummarizer struct {
	basePath            string
	filePrefix          string
	fileSuffix          string
	recentFiles         []string
	processCounter      int
	reportProcessorType ReportingType
}

// New constructor for BasicDiskSummarizer
func New(basePath string, filePrefix string, fileSuffix string, reportProcessorType ReportingType) *BasicDiskSummarizer {
	return &BasicDiskSummarizer{
		basePath:       basePath,
		filePrefix:     filePrefix,
		fileSuffix:     fileSuffix,
		processCounter: 1,
	}
}

// getRecentFiles returns a list of the recent filtered files on disk
func (b *BasicDiskSummarizer) getRecentFiles(ctx context.Context, since time.Time) ([]string, bool, error) {
	files, err := ioutil.ReadDir(b.basePath)
	if err != nil {
		return nil, false, err
	}
	if len(files) == 0 {
		return nil, false, nil
	}
	recentFiles := make([]string, 0, len(files))
	for _, file := range files {
		if file.IsDir() || !strings.HasPrefix(file.Name(), b.filePrefix) || !strings.HasSuffix(file.Name(), b.fileSuffix) {
			continue
		}
		if !file.ModTime().After(since) {
			continue
		}
		recentFiles = append(recentFiles, file.Name())
	}
	if len(recentFiles) == 0 {
		return nil, false, nil
	}
	b.recentFiles = recentFiles
	return recentFiles, true, nil
}

// getRecentFiles returns a list of the recent filtered files on disk
func (b *BasicDiskSummarizer) getNextFile() (string, error) {
	if b.reportProcessorType == Latest && b.processCounter > 1 {
		return "", errors.New("Latest file was already processed")
	}
	if len(b.recentFiles)-b.processCounter < 0 {
		return "", errors.New("No files left to process")
	}
	nextFile := b.recentFiles[len(b.recentFiles)-b.processCounter]
	b.processCounter = b.processCounter + 1
	return nextFile, nil
}

// Summary provides the content of the most recent file
func (b *BasicDiskSummarizer) Summary(ctx context.Context, since time.Time) (io.ReadCloser, bool, error) {
	if len(b.recentFiles) == 0 {
		_, content, err := b.getRecentFiles(ctx, since)
		if !content {
			return nil, content, err
		}
	}

	summaryFile, err := b.getNextFile()
	if err != nil {
		return nil, false, nil
	}
	klog.V(4).Infof("Found files to send: %v", summaryFile)
	f, err := os.Open(filepath.Join(b.basePath, summaryFile))
	if err != nil {
		return nil, false, nil
	}
	return f, true, nil
}
