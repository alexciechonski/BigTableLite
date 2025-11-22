.PHONY: proto build docker-build docker-run test clean

# Generate protobuf code
proto:
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/bigtablelite.proto

# Build the application
build: proto
	go build -o bigtablelite .

# Build Docker image
docker-build:
	docker build -t bigtablelite:latest .

# Run locally (requires Redis on localhost:6379)
run: build
	./bigtablelite -redis-addr localhost:6379

# Run tests
test:
	go test -v -race -cover ./...

# Clean build artifacts
clean:
	rm -f bigtablelite
	rm -rf proto/*.pb.go

# Install dependencies
deps:
	go mod download
	go mod tidy

