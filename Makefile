# DumbProtocol Service Makefile

.PHONY: help dev build tidy tag docker docker_publish test clean

IMAGE_NAME ?= dumbprotocol
REGISTRY ?= 

# Determine current git version tag or default to v0.1.0
CURRENT_TAG := $(shell git describe --tags --abbrev=0 2>/dev/null)
VERSION ?= $(if $(CURRENT_TAG),$(CURRENT_TAG),v0.1.0)

# Calculate next patch version tag if not explicitly set
NEXT_VERSION ?= $(shell \
	if [ -z "$(CURRENT_TAG)" ]; then \
		echo "v0.1.0"; \
	else \
		BASE=$${CURRENT_TAG\#v}; \
		MAJOR=$$(echo $$BASE | cut -d. -f1); \
		MINOR=$$(echo $$BASE | cut -d. -f2); \
		PATCH=$$(echo $$BASE | cut -d. -f3); \
		NEW_PATCH=$$((PATCH + 1)); \
		echo "v$${MAJOR}.$${MINOR}.$${NEW_PATCH}"; \
	fi \
)

TAG_NAME ?= $(NEXT_VERSION)

FULL_IMAGE := $(if $(REGISTRY),$(REGISTRY)/$(IMAGE_NAME),$(IMAGE_NAME))

help: ## Display available commands
	@echo "Usage: make [target] [ARGS...]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

dev: ## Run the local service for dev testing (loads .env and accepts optional args)
	@echo "Starting local dev server (Version: $(VERSION))..."
	@if [ -f .env ]; then \
		echo "Loading configuration from .env file..."; \
		set -a; source .env; set +a; go run main.go $(ARGS); \
	else \
		go run main.go $(ARGS); \
	fi

build: ## Build the Go application binary into bin/dumbprotocol
	@echo "Building Go binary..."
	@mkdir -p bin
	@go build -o bin/dumbprotocol main.go
	@echo "Binary created at bin/dumbprotocol"

tidy: ## Clean and vendor Go dependencies (go mod tidy & go mod vendor)
	@echo "Tidying and vendoring Go modules..."
	@go mod tidy
	@go mod vendor
	@echo "Go modules updated successfully."

tag: ## Create a new git tag based on the previous version (override with TAG_NAME=vX.Y.Z)
	@echo "Current Git Tag: $(if $(CURRENT_TAG),$(CURRENT_TAG),None)"
	@echo "Creating New Git Tag: $(TAG_NAME)"
	@git tag -a $(TAG_NAME) -m "Release $(TAG_NAME)"
	@echo "Successfully tagged commit with $(TAG_NAME)"
	@echo "To push tag upstream run: git push origin $(TAG_NAME)"

docker: ## Build Docker image tagged with current git version
	@echo "Building Docker image $(FULL_IMAGE):$(VERSION)..."
	@docker build -t $(FULL_IMAGE):$(VERSION) -t $(FULL_IMAGE):latest .
	@echo "Docker image built: $(FULL_IMAGE):$(VERSION)"

docker_publish: docker ## Build and publish/push the Docker image to registry
	@echo "Publishing Docker image $(FULL_IMAGE):$(VERSION)..."
	@docker push $(FULL_IMAGE):$(VERSION)
	@docker push $(FULL_IMAGE):latest
	@echo "Successfully published $(FULL_IMAGE):$(VERSION) and $(FULL_IMAGE):latest"

test: ## Run unit and integration tests
	@go test -v ./...

clean: ## Clean built binaries and temp artifacts
	@rm -rf bin/ dumbprotocol
	@echo "Cleaned build artifacts."
