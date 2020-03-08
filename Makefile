test:
	@go test -v
	
bench:
	@go test -bench=. -run=^a

cover:
	@go test -coverprofile=coverage.out -covermode=count
	@go tool cover -html=coverage.out -o coverage.html

lint:
	@golangci-lint run --new-from-rev=e0a5614e47d349897~0

lint-full:
	@golangci-lint run
