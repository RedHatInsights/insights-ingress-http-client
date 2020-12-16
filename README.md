# insights-ingress-http-client

A golang http client that streamlines authentication of openshift 4 clusters
with cloud.redhat.com ingress by providing a configured http client that can
authenticate using credentials associated with an individual openshift cluster.

Authentication for openshift 4 clusters is facilitated by using the
cluster-id and pull secret as credentials.

# How to use the client

The _client_ is actually a controller that periodically sends data. To get
started you must create a `Summarizer` and a new controller.

The job of a summarizer is to provide a blob of data to be uploaded when requested.
The controller will take care of submitting the data, tracking configuration changes, emitting metrics and logging information.

# Goals

This library aims to provide an easy-to-use interface to the cloud.redhat.com
ingress service. The expected uses of this library include batch payload
delivery of gathered data from Openshift 4 clusters.

The library provides:
* a configured http client
* automatic error handling
* credential fetching from the openshift cluster
* automatic proxy configuration inherited from the cluster
* metrics relative to the posting of data

# Non-Goals

This library currently does not aim to manage disconnected environments.

# How to Test

`make test` Will execute the unit test suite.

# How to Tag a Release

This project follows the [official guidelines](https://blog.golang.org/publishing-go-modules) for module
versioning provided by the golang project.  That means that git tags are created for each version and the format of the tag follows [semantic versioning](https://semver.org/) rules.

The following tag command will mark version `1.0.0`:
```
git tag v1.0.0
```

# Generate Cluster Role

This library needs to collect the pull secret information from the openshift-config namespace.
A template for the cluster role and role binding exist in `templates/cluster_role.yaml`.
You must replace the `OPERATOR_SERVICE_ACCOUNT`, `OPERATOR_NAMESPACE`, and `OPERATOR_PREFIX` for the role binding.
`make gen-cluster-role SERVICE_ACCOUNT=myoperator NAMESPACE=default OPERATOR_PREFIX=myprefix` will create a `cluster_role.yaml`
file in the `manifests` directory replacing the placeholders with the supplied values.
