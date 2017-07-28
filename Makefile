default: install build

install:
	go get github.com/stretchr/testify/assert

build:
	go build

test:
	@go test -v

cover:
	@go test -coverprofile=coverage.out -covermode=count
	@go tool cover -html=coverage.out
    