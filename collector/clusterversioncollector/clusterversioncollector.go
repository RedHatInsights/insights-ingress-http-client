package clusterversioncollector

import (
	"context"
	"fmt"

	configv1 "github.com/openshift/api/config/v1"
	configv1client "github.com/openshift/client-go/config/clientset/versioned/typed/config/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

// ClusterVersionCollector The structure for obtaining cluster version information
type ClusterVersionCollector struct {
	kubeConfig     *rest.Config
	clusterVersion *configv1.ClusterVersion
	clusterID      string
}

// New Initialize a new cluster version collector object
func New(kubeConfig *rest.Config) *ClusterVersionCollector {
	return &ClusterVersionCollector{
		kubeConfig: kubeConfig,
	}
}

// GetClusterVersion Get Cluster Version via API
func (c *ClusterVersionCollector) GetClusterVersion() (*configv1.ClusterVersion, error) {
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

// GetClusterID Get Cluster ID from the ClusterVersion
func (c *ClusterVersionCollector) GetClusterID() (string, error) {
	if c.clusterID != "" {
		return c.clusterID, nil
	}
	cv, err := c.GetClusterVersion()
	if err != nil {
		return "", err
	}
	clusterID := ""
	if cv.Spec.ClusterID != "" {
		clusterID = string(cv.Spec.ClusterID)
	} else {
		err = fmt.Errorf("No cluster ID found in ClusterVersion Spec")
	}
	return clusterID, err
}
