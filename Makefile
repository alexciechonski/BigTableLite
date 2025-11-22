.PHONY: proto build docker-build docker-run test clean cpp-lib

# Build C++ SSTable library
cpp-lib:
	$(MAKE) -C cpp

# Generate protobuf code
proto:
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/bigtablelite.proto

# Build the application
build: cpp-lib proto
	CGO_ENABLED=1 go build -o bigtablelite ./cmd/server

# Build Docker image
docker-build:
	docker build -t bigtablelite:latest .

# Run locally with SSTable backend (default)
run: build
	./bigtablelite

# Run locally with Redis backend
run-redis: build
	./bigtablelite -use-redis -redis-addr localhost:6379

# Run tests
test:
	go test -v -race -cover ./...

# Clean build artifacts
clean:
	rm -f bigtablelite
	# Note: proto/*.pb.go files are not removed - regenerate with 'make proto' if needed
	$(MAKE) -C cpp clean

# Install dependencies
deps:
	go mod download
	go mod tidy

