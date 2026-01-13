# Makefile for blackout tool

BINARY_NAME=blackout
INSTALL_PATH=/usr/local/bin
GO=go
GOFLAGS=-ldflags="-s -w"

.PHONY: all build install clean test deps run help

all: build

## build: Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	$(GO) build $(GOFLAGS) -o $(BINARY_NAME)
	@echo "Build complete: ./$(BINARY_NAME)"

## install: Build and install to $(INSTALL_PATH)
install: build
	@echo "Installing $(BINARY_NAME) to $(INSTALL_PATH)..."
	sudo cp $(BINARY_NAME) $(INSTALL_PATH)/$(BINARY_NAME)
	@echo "Installed: $(INSTALL_PATH)/$(BINARY_NAME)"

## deps: Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GO) mod download
	$(GO) mod tidy

## test: Run tests
test:
	@echo "Running tests..."
	$(GO) test -v ./...

## clean: Remove build artifacts
clean:
	@echo "Cleaning..."
	rm -f $(BINARY_NAME)
	@echo "Clean complete"

## run: Build and run with example arguments
run: build
	@echo "Running $(BINARY_NAME) help..."
	./$(BINARY_NAME) help

## schema: Display SQL schema
schema:
	@cat schema.sql

## uninstall: Remove installed binary
uninstall:
	@echo "Removing $(INSTALL_PATH)/$(BINARY_NAME)..."
	sudo rm -f $(INSTALL_PATH)/$(BINARY_NAME)
	@echo "Uninstalled"

## help: Display this help message
help:
	@echo "Blackout Tool - Makefile commands:"
	@echo ""
	@grep -E '^##' Makefile | sed 's/## /  /'
	@echo ""
	@echo "Example usage:"
	@echo "  make build                    # Build the binary"
	@echo "  make install                  # Build and install to /usr/local/bin"
	@echo "  make run                      # Build and run help"
	@echo "  make clean                    # Remove build artifacts"
	@echo ""
