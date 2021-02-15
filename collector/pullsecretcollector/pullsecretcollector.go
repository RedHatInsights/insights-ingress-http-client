package pullsecretcollector

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	openShiftConfigNamespace = "openshift-config"
	pullSecretName           = "pull-secret"
	pullSecretDataKey        = ".dockerconfigjson"
	pullSecretAuthKey        = "cloud.openshift.com"
)

type serializedAuthMap struct {
	Auths map[string]serializedAuth `json:"auths"`
}

type serializedAuth struct {
	Auth string `json:"auth"`
}

// PullSecretCollector The structure for obtaining cluster pull secret
type PullSecretCollector struct {
	clientset *kubernetes.Clientset
}

// New Initialize a new pull secret collector object
func New(clientset *kubernetes.Clientset) *PullSecretCollector {
	return &PullSecretCollector{
		clientset: clientset,
	}
}

// GetPullSecret Obtain the pull secret in the openshift-config namespace
func (p *PullSecretCollector) GetPullSecret() (*corev1.Secret, error) {
	ctx := context.Background()
	secret, err := p.clientset.CoreV1().Secrets(openShiftConfigNamespace).Get(ctx, pullSecretName, metav1.GetOptions{})
	return secret, err
}

// GetPullSecretToken Obtain the bearer token string from the pull secret in the openshift-config namespace
func (p *PullSecretCollector) GetPullSecretToken() (string, error) {
	secret, err := p.GetPullSecret()
	if err != nil {
		return "", err
	}

	encodedPullSecret := secret.Data[pullSecretDataKey]
	if len(encodedPullSecret) <= 0 {
		return "", fmt.Errorf("cluster authorization secret did not have data")
	}
	var pullSecret serializedAuthMap
	if err := json.Unmarshal(encodedPullSecret, &pullSecret); err != nil {
		return "", err
	}
	if auth, ok := pullSecret.Auths[pullSecretAuthKey]; ok {
		token := strings.TrimSpace(auth.Auth)
		if strings.Contains(token, "\n") || strings.Contains(token, "\r") {
			return "", fmt.Errorf("cluster authorization token is not valid: contains newlines")
		}
		if len(token) > 0 {
			return token, nil
		}
		return "", fmt.Errorf("cluster authorization token is not found")
	}
	return "", fmt.Errorf("cluster authorization token is not found")
}
