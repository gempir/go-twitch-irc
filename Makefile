test:
	@go test -v
	
bench:
	@go test -bench=. -benchtime=5x -run=^a

cover:
	@go test -coverprofile=coverage.out -covermode=count
	@go tool cover -html=coverage.out -o coverage.html
