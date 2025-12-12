.PHONY: build run test lint clean

# Build the project
# Formats code and compiles the binary
build:
	go fmt ./...
	go build -o helloworld .

# Run the compiled binary
run:
	./helloworld

# Run tests with coverage
test:
	go test -v -cover ./...

# Run linter
lint:
	golangci-lint run

# Remove build artifacts
clean:
	rm -f helloworld
