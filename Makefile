

test:
	go test $$(go list ./... | grep -v /test/) $(TEST_OPTIONS)
.PHONY: test


vet:
	go vet $$(go list ./... | grep -v /vendor/)

lint:
	golint $$(go list ./... | grep -v /vendor/)


vendor:
	go mod tidy
	go mod vendor
	go mod verify
.PHONY: vendor
