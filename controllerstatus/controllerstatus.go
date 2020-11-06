package controllerstatus

import (
	"sync"
	"time"

	"k8s.io/klog"
)

// Interface an interface for controller status
type Interface interface {
	CurrentStatus() (summary Summary, ready bool)
}

// Operation describing the operation
type Operation string

const (
	// Uploading specific flag for summary related to uploading process.
	Uploading Operation = "Uploading"
)

// Summary a structure describing the health, time, and count of an operation
type Summary struct {
	Operation          Operation
	Healthy            bool
	Reason             string
	Message            string
	LastTransitionTime time.Time
	Count              int
}

// Simple a structure for tracking a named summary
type Simple struct {
	Name string

	lock    sync.Mutex
	summary Summary
}

// UpdateStatus A method for updating the tracking of a summary
func (s *Simple) UpdateStatus(summary Summary) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.summary.Healthy != summary.Healthy {
		klog.V(2).Infof("name=%s healthy=%t reason=%s message=%s", s.Name, summary.Healthy, summary.Reason, summary.Message)
		if summary.LastTransitionTime.IsZero() {
			summary.LastTransitionTime = time.Now()
		}

		s.summary = summary
		s.summary.Count = 1
		return
	}

	s.summary.Count++
	if summary.Healthy {
		return
	}
	if s.summary.Message != summary.Message || s.summary.Reason != summary.Reason {
		klog.V(2).Infof("name=%s healthy=%t reason=%s message=%s", s.Name, summary.Healthy, summary.Reason, summary.Message)
		s.summary.Reason = summary.Reason
		s.summary.Message = summary.Message
		return
	}
}

// CurrentStatus A method for obtaining the status of a summary object and whether it previously existed
func (s *Simple) CurrentStatus() (Summary, bool) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.summary.Count == 0 {
		return Summary{}, false
	}
	return s.summary, true
}
