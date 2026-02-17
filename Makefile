.PHONY: build push buildx test lint all


GH_USER = ?
GH_REPO = stackctl

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

install-cli:
	@go install ./cmd/stackctl

multapps:
	@echo "🔍 Checking if Multipass is installed..."
	@which multipass > /dev/null 2>&1 || { echo "❌ Multipass is not installed. Please install it first."; exit 1; }

	@echo "🚀 Launching Multipass instance 'dev' with 4 CPUs and 4GB RAM..."
	@if multipass info dev >/dev/null 2>&1; then \
		echo "⚙️  Instance 'dev' already exists. Skipping creation..."; \
	else \
		multipass launch -n dev --cpus 4 --memory 4G --disk 20G --mount $$(pwd):/home/ubuntu/workdir --cloud-init example/multipass-init.yaml; \
	fi

	@echo "🔗 Connecting to the 'dev' instance..."
	multipass shell dev

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
