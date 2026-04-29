.PHONY: build push buildx test lint all multipass multipass-delete multipass-recreate multipass-shell multipass-test binary install clean help

BINARY_NAME=stackctl
BUILD_DIR=bin
GO=go
GOFLAGS=-v
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DATE=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS=-ldflags "-s -w \
	-X github.com/eliasmeireles/stackctl/cmd/stackctl/cmd.Version=$(VERSION) \
	-X github.com/eliasmeireles/stackctl/cmd/stackctl/cmd.BuildDate=$(BUILD_DATE) \
	-X github.com/eliasmeireles/stackctl/cmd/stackctl/cmd.Commit=$(COMMIT)"

GH_USER = ?
GH_REPO = stackctl

help:
	@echo "Available targets:"
	@echo "  binary         - Build the CLI binary and copy to .dev/multipass/.volumes"
	@echo "  install        - Install the binary to GOPATH/bin"
	@echo "  lint           - Run linters"
	@echo "  test           - Run tests"
	@echo "  multipass      - Setup Multipass instance"
	@echo "  multipass-test - Run tests inside Multipass"
	@echo "  build          - Build and push Docker image"
	@echo "  clean          - Remove build artifacts"

binary:
	@echo "Building $(BINARY_NAME) $(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/stackctl
	@mkdir -p .dev/multipass/.volumes
	@cp $(BUILD_DIR)/$(BINARY_NAME) .dev/multipass/.volumes/
	@echo "✅ Binary built and copied to .dev/multipass/.volumes/"
	./.dev/multipass/.volumes/stackctl -v

clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@$(GO) clean

lint:
	@golangci-lint run --timeout=5m

test:
	# https://github.com/gotestyourself/gotestsum
	# go install gotest.tools/gotestsum@latest
	@gotestsum --format testname

update:
	@go mod tidy

# Setup Buildx builder
buildx:
	@docker buildx create --name buildxBuilder --use
	@docker buildx inspect buildxBuilder --bootstrap


build:
	@read -p "Enter the tag version: " TAG; \
	 docker buildx build --platform linux/amd64,linux/arm64 -t ghcr.io/$${GH_USER}/$${GH_REPO}:$$TAG --push .

# Push the Docker image
push:
	@read -p "Enter the tag version: " TAG; \
	 docker push ghcr.io/$${GH_USER}/$${GH_REPO}:$$TAG


# CLI Docker targets
build-cli:
	@read -p "Enter the tag version: " TAG; \
	 docker buildx build --platform linux/amd64,linux/arm64 -f ./Dockerfile.cli -t ghcr.io/$${GH_USER}/$${GH_REPO}:$$TAG --push .
	 # If tag is not latest, push latest
	 if [ "$$TAG" != "latest" ]; then \
		 docker push ghcr.io/$${GH_USER}/$${GH_REPO}:latest; \
	 fi

test-build-cli:
	 docker buildx build -f ./Dockerfile.cli -t ghcr.io/$${GH_USER}/$${GH_REPO}:latest --load . && \
	 docker run --rm --privileged --entrypoint /bin/bash ghcr.io/$${GH_USER}/$${GH_REPO}:latest \
	 -c "nohup netbird service run > /dev/null 2>&1 & sleep 5 && stackctl vault fetch --resource-name home-lab --with-netbird && echo '✅ Fetch complete, listing pods...' && kubectl get pods -n kube-system"

test-cli:
	 docker run --rm --privileged --entrypoint /bin/bash ghcr.io/$${GH_USER}/$${GH_REPO}:latest \
	 -c "nohup netbird service run > /dev/null 2>&1 & sleep 5 && stackctl vault fetch --resource-name home-lab --with-netbird && echo '✅ Fetch complete, listing pods...' && kubectl get pods -n kube-system"

install: binary
	@echo "Installing $(BINARY_NAME) to $(GOPATH)/bin..."
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/
	@chmod +x $(GOPATH)/bin/$(BINARY_NAME)
	@echo "✅ Installed successfully to $(GOPATH)/bin/$(BINARY_NAME)"

multipass:
	@bash .dev/multipass/setup-new.sh

multipass-shell:
	@multipass shell stackctl

multipass-delete:
	@echo "🗑️  Deleting Multipass instance 'stackctl'..."
	@multipass delete stackctl
	@multipass purge
	@echo "✅ Instance deleted."

multipass-recreate: multipass-delete multipass

multipass-test:
	@echo "🧪 Running stackctl validation tests inside Multipass instance..."
	@multipass exec stackctl -- bash /home/ubuntu/workdir/test-stackctl.sh

multipass-test-homelab-rbac: binary
	@echo "🧪 Running homelab-rbac end-to-end test inside Multipass..."
	@cp .dev/multipass/test-homelab-rbac.sh .dev/multipass/.volumes/test-homelab-rbac.sh
	@chmod +x .dev/multipass/.volumes/test-homelab-rbac.sh
	@cp example/homelab-rbac.yaml .dev/multipass/.volumes/homelab-rbac.yaml
	@multipass exec stackctl -- sudo cp /home/ubuntu/workdir/stackctl /usr/local/bin/stackctl
	@multipass exec stackctl -- sudo chmod +x /usr/local/bin/stackctl
	@multipass exec stackctl -- bash /home/ubuntu/workdir/test-homelab-rbac.sh

# Multi-arch build variables
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64

# Build for all platforms
build-all:
	@echo "🏗️  Building artifacts..."
	@for platform in $(PLATFORMS); do \
		GOOS=$${platform%/*} ; \
		GOARCH=$${platform#*/} ; \
		OUTPUT_BIN="bin/$$GOOS-$$GOARCH/stackctl" ; \
		if [ "$$GOOS" = "windows" ]; then OUTPUT_BIN="$${OUTPUT_BIN}.exe"; fi ; \
		echo "🚀 Building for $$GOOS/$$GOARCH..." ; \
		GOOS=$$GOOS GOARCH=$$GOARCH go build -o $$OUTPUT_BIN ./cmd/stackctl ; \
	done
	@echo "✅ Build complete for all platforms!"
