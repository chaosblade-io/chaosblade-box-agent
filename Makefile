# Copyright 2025 The ChaosBlade Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

.PHONY: build clean
export AGENT_VERSION = 1.1.0
export BLADE_VERSION = 1.8.0

BLADE_SRC_ROOT=$(shell pwd)

BUILD_BINARY_MUSL_PATH=build/binary_musl
BUILD_BINARY_ARM64_PATH=build/binary_arm
BUILD_IMAGE_MUSL_PATH=build/image_musl
BUILD_IMAGE_ARM64_PATH=build/image_arm


BUILD_HELM_PATH_AMD64=build/helm3/chaos-agent/
BUILD_HELM_PATH_ARM64=build/helm3/chaos-agent-arm/
BUILD_BINARY_PATH=build
BUILD_PACKAGE_PATH=build/package

# Auto-detect ARCH if not specified
ifeq ($(ARCH),)
  MACHINE_ARCH := $(shell uname -m)
  ifeq ($(MACHINE_ARCH),x86_64)
    ARCH := amd64
  else ifeq ($(MACHINE_ARCH),aarch64)
    ARCH := arm64
  else ifeq ($(MACHINE_ARCH),arm64)
    ARCH := arm64
  else
    ARCH := unknown
  endif
endif

VERSION_PKG=github.com/chaosblade-io/chaos-agent/version

GO_ENV=CGO_ENABLED=1
GO_MODULE=GO111MODULE=on
#GO_PROXY=GOPROXY=https://mirrors.aliyun.com/goproxy/
GO=env $(GO_ENV) $(GO_MODULE) go
#GO_X_FLAGS=-X ${VERSION_PKG}.AgentVersion=$(AGENT_VERSION) -X '${VERSION_PKG}.Env=`uname -mv`' -X '${VERSION_PKG}.BuildTime=`date`'
GO_FLAGS=-ldflags="-s -w"

ifeq ($(GOOS), linux)
	GO_FLAGS=-ldflags="-linkmode external -extldflags -static -s -w"
endif

build: build_binary

build_binary: cmd/chaos_agent.go
	$(GO) build $(GO_FLAGS) -o $(BUILD_BINARY_PATH)/agent $<

build_amd64:
	@echo "Building for linux/amd64..."
	@MACHINE_ARCH=$$(uname -m); \
	MACHINE_OS=$$(uname -s | tr '[:upper:]' '[:lower:]'); \
	if [ "$$MACHINE_OS" = "linux" ] && [ "$$MACHINE_ARCH" = "x86_64" ]; then \
		echo "Detected native linux/amd64 system, using direct build..."; \
		GOOS=linux GOARCH=amd64 $(MAKE) build_binary; \
	else \
		echo "Using Docker to build for linux/amd64..."; \
		docker build -f $(BUILD_BINARY_MUSL_PATH)/Dockerfile -t agent-build-musl:latest $(BUILD_BINARY_MUSL_PATH) && \
		docker run --rm \
			-v $(shell echo -n ${GOPATH}):/go \
			-w /chaos-agent \
			-v $(BLADE_SRC_ROOT):/chaos-agent \
			agent-build-musl:latest; \
	fi

build_arm64:
	@echo "Building for linux/arm64..."
	@MACHINE_ARCH=$$(uname -m); \
	MACHINE_OS=$$(uname -s | tr '[:upper:]' '[:lower:]'); \
	if [ "$$MACHINE_OS" = "linux" ] && ([ "$$MACHINE_ARCH" = "aarch64" ] || [ "$$MACHINE_ARCH" = "arm64" ]); then \
		echo "Detected native linux/arm64 system, using direct build..."; \
		GOOS=linux GOARCH=arm64 $(MAKE) build_binary; \
	else \
		echo "Using Docker to build for linux/arm64..."; \
		docker build -f $(BUILD_BINARY_ARM64_PATH)/Dockerfile -t agent-build-arm:latest $(BUILD_BINARY_ARM64_PATH) && \
		docker run --rm \
			-v $(shell echo -n ${GOPATH}):/go \
			-w /chaos-agent \
			-v $(BLADE_SRC_ROOT):/chaos-agent \
			agent-build-arm:latest; \
	fi

build_chart_amd64:
	helm package $(BUILD_HELM_PATH_AMD64)

build_chart_arm64:
	helm package $(BUILD_HELM_PATH_ARM64)

build_image_amd64:
	rm -rf $(BUILD_IMAGE_MUSL_PATH)/agent
	cp $(BUILD_BINARY_PATH)/agent $(BUILD_IMAGE_MUSL_PATH)
	docker build --pull --build-arg BLADE_VERSION=${BLADE_VERSION} -f $(BUILD_IMAGE_MUSL_PATH)/Dockerfile \
		-t chaosbladeio/chaosblade-agent:$(AGENT_VERSION) $(BLADE_SRC_ROOT)/$(BUILD_IMAGE_MUSL_PATH)

build_image_arm64:
	rm -rf $(BUILD_IMAGE_ARM64_PATH)/agent
	cp $(BUILD_BINARY_PATH)/agent $(BUILD_IMAGE_ARM64_PATH)
	docker build --pull --build-arg BLADE_VERSION=${BLADE_VERSION} -f $(BUILD_IMAGE_ARM64_PATH)/Dockerfile \
		-t chaosbladeio/chaosblade-agent-arm64:$(AGENT_VERSION) $(BLADE_SRC_ROOT)/$(BUILD_IMAGE_ARM64_PATH)

.PHONY: package
package: package_prepare package_scripts package_chaosblade package_binary package_tar

package_prepare:
	@echo "Preparing package directory..."
	@if [ "$(ARCH)" = "unknown" ]; then \
		echo "Error: Unsupported architecture: $(shell uname -m)"; \
		echo "Please specify ARCH explicitly: make package ARCH=amd64 or ARCH=arm64"; \
		exit 1; \
	fi
	@if [ -z "$(ARCH)" ]; then \
		echo "Error: Failed to determine architecture"; \
		exit 1; \
	fi
	@echo "Using architecture: $(ARCH)"
	@rm -rf $(BUILD_PACKAGE_PATH)
	@mkdir -p $(BUILD_PACKAGE_PATH)/chaos/chaosblade

package_scripts:
	@echo "Copying scripts..."
	@cp $(BLADE_SRC_ROOT)/build/cmd/chaosctl.sh $(BUILD_PACKAGE_PATH)/chaos/
	@cp $(BLADE_SRC_ROOT)/build/cmd/chaosrcv.sh $(BUILD_PACKAGE_PATH)/chaos/
	@chmod +x $(BUILD_PACKAGE_PATH)/chaos/*.sh

package_chaosblade:
	@echo "Downloading chaosblade tool..."
	@cd $(BUILD_PACKAGE_PATH)/chaos && \
	curl -L https://chaosblade.oss-cn-hangzhou.aliyuncs.com/agent/github/$(BLADE_VERSION)/chaosblade-$(BLADE_VERSION)-linux_$(ARCH).tar.gz | tar xz && \
	EXTRACTED_DIR="chaosblade-$(BLADE_VERSION)-linux_$(ARCH)"; \
	if [ -d "$$EXTRACTED_DIR" ]; then \
		mv $$EXTRACTED_DIR/* chaosblade/ && \
		rmdir $$EXTRACTED_DIR || true; \
		echo "✅ ChaosBlade tool extracted to chaosblade directory"; \
	else \
		echo "Warning: Expected directory $$EXTRACTED_DIR not found after extraction"; \
		ls -la; \
	fi

package_binary:
	@echo "Copying agent binary..."
	@if [ ! -f "$(BUILD_BINARY_PATH)/agent" ]; then \
		echo "Error: Binary file not found at $(BUILD_BINARY_PATH)/agent"; \
		echo "Please run 'make build_amd64' (for amd64) or 'make build_arm64' (for arm64) first"; \
		exit 1; \
	fi
	@cp $(BUILD_BINARY_PATH)/agent $(BUILD_PACKAGE_PATH)/chaos/
	@chmod +x $(BUILD_PACKAGE_PATH)/chaos/agent
	@echo "✅ Agent binary copied"
	@echo "Package structure:"
	@echo "  chaos/"
	@echo "    ├── chaosctl.sh"
	@echo "    ├── chaosrcv.sh"
	@echo "    ├── agent"
	@echo "    └── chaosblade/"
	@echo "        └── [chaosblade tool files]"

package_tar:
	@echo "Creating tar.gz package..."
	@cd $(BUILD_PACKAGE_PATH) && \
	tar -czf $(BLADE_SRC_ROOT)/$(BUILD_BINARY_PATH)/chaosagent-$(AGENT_VERSION)-linux_$(ARCH).tar.gz chaos
	@echo "✅ Package created: $(BUILD_BINARY_PATH)/chaosagent-$(AGENT_VERSION)-linux_$(ARCH).tar.gz"
	@ls -lh $(BLADE_SRC_ROOT)/$(BUILD_BINARY_PATH)/chaosagent-$(AGENT_VERSION)-linux_$(ARCH).tar.gz

.PHONY: format
format: license-format
	@echo "Running goimports and gofumpt to format Go code..."
	@./hack/update-imports.sh
	@./hack/update-gofmt.sh

.PHONY: verify
verify:
	@echo "Verifying Go code formatting and import order..."
	@./hack/verify-gofmt.sh
	@./hack/verify-imports.sh

.PHONY: license-check
license-check:
	@echo "Checking license headers..."
	@if command -v hawkeye > /dev/null 2>&1; then \
		hawkeye check; \
	else \
		echo "Using Docker to run hawkeye..."; \
		docker run -it --rm -v $(shell pwd):/github/workspace ghcr.io/korandoru/hawkeye check; \
	fi

.PHONY: license-format
license-format:
	@echo "Formatting license headers..."
	@if command -v hawkeye > /dev/null 2>&1; then \
		hawkeye format; \
	else \
		echo "Using Docker to run hawkeye..."; \
		docker run -it --rm -v $(shell pwd):/github/workspace ghcr.io/korandoru/hawkeye format; \
	fi

.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_PACKAGE_PATH)
	@rm -f $(BUILD_BINARY_PATH)/chaosagent-*.tar.gz

.PHONY: help
help:
	@echo "Makefile commands:"
	@echo "  build             - Build the chaos agent binary"
	@echo "  build_amd64      - Build the chaos agent binary for linux/amd64"
	@echo "                    (Uses direct build on native linux/amd64, otherwise uses Docker)"
	@echo "  build_arm64      - Build the chaos agent binary for linux/arm64"
	@echo "                    (Uses direct build on native linux/arm64, otherwise uses Docker)"
	@echo "  build_chart_amd64 - Package the Helm chart for the chaos agent (amd64)"
	@echo "  build_chart_arm64 - Package the Helm chart for the chaos agent (arm64)"
	@echo "  build_image_amd64 - Build the Docker image for the chaos agent (amd64)"
	@echo "  build_image_arm64 - Build the Docker image for the chaos agent (arm64)"
	@echo "  package          - Package agent binary, scripts and chaosblade tool"
	@echo "                    Usage: make package ARCH=amd64 (or ARCH=arm64)"
	@echo "                    Creates: chaos/chaosctl.sh, chaos/chaosrcv.sh, chaos/chaosblade/ (with tool and agent)"
	@echo "                    Note: Requires build, build_amd64 or build_arm64 to be run first"
	@echo "  format           - Format Go code using goimports and gofumpt"
	@echo "  verify           - Verify Go code formatting and import order"
	@echo "  license-check    - Check license headers in source files"
	@echo "  clean            - Clean up build artifacts"
	@echo "  help             - Show this help message"
