PROJECT_NAME := professor
BINARY_NAME := $(PROJECT_NAME)
DOCKER_IMAGE := h4i-umd/$(PROJECT_NAME)
VERSION ?= $(shell git describe --tags --always --dirty)-$(shell git rev-parse --short HEAD)

.PHONY: build test docker-build docker-run clean

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

build: ## Build the Go application natively
	@echo "Building Go application..."
	go build -o $(BINARY_NAME)

test: ## Run unit tests
	@echo "Running tests..."
	go test -v ./...

docker-build: build ## Build the Docker image
	@echo "Building Docker image..."
	docker build --rm --tag $(DOCKER_IMAGE):$(VERSION) .
	@echo "Docker image built: $(DOCKER_IMAGE):$(VERSION)"

docker-run: docker-build ## Build and run the Docker container
	@echo "Running Docker container..."
	docker run --rm -p 8000:8000 --env-file ./.env --name $(PROJECT_NAME)-container $(DOCKER_IMAGE):$(VERSION)

clean: ## Clean up build artifacts
	@echo "Cleaning up..."
	rm -f $(BINARY_NAME)
	docker rmi $(DOCKER_IMAGE):$(VERSION) || true
