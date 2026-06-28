BINARY     := gomaat
CMD        := ./cmd/gomaat/
BUILD_DIR  := ./bin

GO         := $(shell which go 2>/dev/null || echo /usr/local/go/bin/go)

.PHONY: all build install fmt vet test lint clean tidy check

all: fmt vet test build

## build: compile the binary to ./bin/gomaat
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

## test: run all tests
test:
	$(GO) test ./...

## test-verbose: run all tests with verbose output
test-verbose:
	$(GO) test -v ./...

## watchtest: re-run tests on any .go file change (requires entr)
watchtest:
	find . -name '*.go' | entr -c $(GO) test ./...

## lint: run golangci-lint
lint:
	golangci-lint run

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
