module github.com/redhatinsights/insights-ingress-http-client

go 1.14

require (
	github.com/openshift/api v0.0.0-20201214114959-164a2fb63b5f
	github.com/openshift/client-go v0.0.0-00010101000000-000000000000
	golang.org/x/net v0.0.0-20201209123823-ac852fbbde11
	k8s.io/api v0.20.0
	k8s.io/apiextensions-apiserver v0.20.0
	k8s.io/apimachinery v0.20.0
	k8s.io/client-go v11.0.0+incompatible
	k8s.io/component-base v0.20.0
	k8s.io/klog v1.0.0
	k8s.io/klog/v2 v2.4.0
)

replace (
	github.com/openshift/api => github.com/openshift/api v0.0.0-20201214114959-164a2fb63b5f
	github.com/openshift/client-go => github.com/openshift/client-go v0.0.0-20201214125552-e615e336eb49
	k8s.io/api => k8s.io/api v0.20.0
	k8s.io/apimachinery => k8s.io/apimachinery v0.20.0
	k8s.io/apiserver => k8s.io/apiserver v0.20.0
	k8s.io/client-go => k8s.io/client-go v0.20.0
)
