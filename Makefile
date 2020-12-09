

test:
	go test $$(go list ./... | grep -v /test/) $(TEST_OPTIONS)
.PHONY: test


vet:
	go vet $$(go list ./... | grep -v /vendor/)

lint:
	golint $$(go list ./... | grep -v /vendor/)

gen-cluster-role:
	@cp templates/cluster_role.yaml manifests/cluster_role.yaml
	@sed -i 's/OPERATOR_SERVICE_ACCOUNT/$(shell printf "$(shell echo $(SERVICE_ACCOUNT))")/g' manifests/cluster_role.yaml
	@sed -i 's/OPERATOR_NAMESPACE/$(shell printf "$(shell echo $(NAMESPACE))")/g' manifests/cluster_role.yaml

vendor:
	go mod tidy
	go mod vendor
	go mod verify
.PHONY: vendor
