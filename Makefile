test:
	openssl ecparam -genkey -name secp384r1 -out server.key
	openssl req -new -x509 -sha256 -key server.key -out server.crt -days 3650 -subj "/C=GB/ST=London/L=London/O=Global Security/OU=IT Department/CN=example.com"
	@go test -v
	rm server.key server.crt

cover:
	@go test -coverprofile=coverage.out -covermode=count
	@go tool cover -html=coverage.out -o coverage.html
