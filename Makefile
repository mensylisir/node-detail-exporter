# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
DOCKER_BUILD=docker build

# Binary name
BINARY_NAME=node-prober

# Build variables
VERSION ?= $(shell git describe --tags --always --dirty)
BUILD_DATE ?= $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')

.PHONY: all build test docker-build clean

all: build

build:
	@echo "Building $(BINARY_NAME) version $(VERSION)..."
	@$(GOBUILD) -o $(BINARY_NAME) -ldflags="-X 'main.Version=$(VERSION)' -X 'main.BuildDate=$(BUILD_DATE)'" .

test:
	@echo "Running tests..."
	@$(GOTEST) -v ./...

docker-build:
	@echo "Building Docker image for $(BINARY_NAME)..."
	@$(DOCKER_BUILD) -t $(BINARY_NAME):$(VERSION) .

clean:
	@echo "Cleaning up..."
	@rm -f $(BINARY_NAME)
