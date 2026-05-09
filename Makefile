BINARY     := godemaat
CMD        := ./cmd/godemaat/
BUILD_DIR  := ./bin

GO         := $(shell which go 2>/dev/null || echo /usr/local/go/bin/go)
GOLINT     := golangci-lint

.PHONY: all build install fmt vet lint test clean tidy check

all: fmt vet lint test build

## build: compile the binary to ./bin/godemaat
build:
	$(GO) build -o $(BUILD_DIR)/$(BINARY) $(CMD)

## install: install the binary to $GOPATH/bin
install:
	$(GO) install $(CMD)

## fmt: format all Go source files
fmt:
	$(GO) fmt ./...

## vet: run go vet on all packages
vet:
	$(GO) vet ./...

## lint: run golangci-lint (install from https://golangci-lint.run/usage/install/)
lint:
	$(GOLINT) run ./...

## test: run all tests
test:
	$(GO) test ./...

## test-verbose: run all tests with verbose output
test-verbose:
	$(GO) test -v ./...

## tidy: tidy and verify go.mod / go.sum
tidy:
	$(GO) mod tidy
	$(GO) mod verify

## check: fmt + vet + lint + test (CI-style, no build artifact)
check: fmt vet lint test

## clean: remove the build directory
clean:
	rm -rf $(BUILD_DIR)

## help: print this help message
help:
	@echo "Usage: make <target>"
	@echo ""
	@sed -n 's/^## //p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/  /'
