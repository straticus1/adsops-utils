.PHONY: build install clean test deps

# Binary name
BINARY=hostctl

# Build the binary
build:
	@echo "Building ${BINARY}..."
	@go build -o ${BINARY} .
	@echo "Build complete: ${BINARY}"

# Install to /usr/local/bin
install: build
	@echo "Installing ${BINARY} to /usr/local/bin..."
	@sudo mv ${BINARY} /usr/local/bin/
	@echo "Installation complete"

# Install to ~/bin (no sudo required)
install-user: build
	@echo "Installing ${BINARY} to ~/bin..."
	@mkdir -p ~/bin
	@mv ${BINARY} ~/bin/
	@echo "Installation complete. Make sure ~/bin is in your PATH"

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy
	@echo "Dependencies updated"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -f ${BINARY}
	@go clean
	@echo "Clean complete"

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Build for multiple platforms
build-all:
	@echo "Building for multiple platforms..."
	@GOOS=linux GOARCH=amd64 go build -o ${BINARY}-linux-amd64 .
	@GOOS=darwin GOARCH=amd64 go build -o ${BINARY}-darwin-amd64 .
	@GOOS=darwin GOARCH=arm64 go build -o ${BINARY}-darwin-arm64 .
	@echo "Multi-platform build complete"

# Show help
help:
	@echo "Available targets:"
	@echo "  build        - Build the binary"
	@echo "  install      - Build and install to /usr/local/bin (requires sudo)"
	@echo "  install-user - Build and install to ~/bin (no sudo)"
	@echo "  deps         - Download and tidy dependencies"
	@echo "  clean        - Remove build artifacts"
	@echo "  test         - Run tests"
	@echo "  build-all    - Build for multiple platforms"
	@echo "  help         - Show this help message"
