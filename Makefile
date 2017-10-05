test:
	@go test -v

cover:
	@go test -coverprofile=coverage.out -covermode=count
	@go tool cover -html=coverage.out -o coverage.html
