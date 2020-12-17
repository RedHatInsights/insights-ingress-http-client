
UNAME_S := $(shell uname -s)
OS_SED :=
ifeq ($(UNAME_S),Darwin)
	OS_SED += ""
endif

test:
	go test $$(go list ./... | grep -v /test/) $(TEST_OPTIONS)
.PHONY: test


vet:
	go vet $$(go list ./... | grep -v /vendor/)

lint:
	golint $$(go list ./... | grep -v /vendor/)

gen-cluster-role:
	@cp templates/cluster_role.yaml manifests/cluster_role.yaml
	@sed -i ${OS_SED} 's/OPERATOR_SERVICE_ACCOUNT/$(shell printf "$(shell echo $(SERVICE_ACCOUNT))")/g' manifests/cluster_role.yaml
	@sed -i ${OS_SED} 's/OPERATOR_NAMESPACE/$(shell printf "$(shell echo $(NAMESPACE))")/g' manifests/cluster_role.yaml
	@sed -i ${OS_SED} 's/OPERATOR_PREFIX/$(shell printf "$(shell echo $(OPERATOR_PREFIX))")/g' manifests/cluster_role.yaml

vendor:
	go mod tidy
	go mod vendor
	go mod verify
.PHONY: vendor
