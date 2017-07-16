PACKAGES = $(shell go list ./... | grep -v /vendor/)

default: install build

install:
	go get github.com/stretchr/testify/assert

build:
	go build

test:
	go test -v $(PACKAGES)

cover:
	echo "mode: count" > coverage-all.out
	$(foreach pkg,$(PACKAGES),\
		go test -coverprofile=coverage.out -covermode=count $(pkg);\
		tail -n +2 coverage.out >> coverage-all.out;)
	go tool cover -html=coverage-all.out