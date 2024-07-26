# 默认值
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
CGO_ENABLED ?= 0

BINARY_NAME=icmp_hijacker

VERSION ?= $(shell git describe --tags --always --dirty)
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%I:%M:%S%p')

MAIN_FILE=main.go

LDFLAGS=-ldflags "-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}"

.PHONY: all build clean run

all: build

build:
	@echo "Building for $(GOOS)/$(GOARCH)..."
	@GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=$(CGO_ENABLED) go build $(LDFLAGS) -o $(BINARY_NAME)-$(GOOS)-$(GOARCH) $(MAIN_FILE)
	@echo "Build complete: $(BINARY_NAME)-$(GOOS)-$(GOARCH)"

clean:
	@echo "Cleaning..."
	@go clean
	@rm -f $(BINARY_NAME)-*
	@echo "Clean complete"

run: build
	@echo "Running $(BINARY_NAME)-$(GOOS)-$(GOARCH)..."
	@./$(BINARY_NAME)-$(GOOS)-$(GOARCH)
