# insights-ingress-http-client

A golang http client that streamlines authentication of openshift 4 clusters
with cloud.redhat.com ingress by providing a configured http client that can
authenticate using credentials associated with an individual openshift cluster.

Authentication for openshift 4 clusters is facilitated by using the cluster-id and pull secret as credentials.
