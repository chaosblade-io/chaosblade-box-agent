.PHONE: build clean
export AGENT_VERSION = 1.0.0
export BLADE_VERSION = 1.5.0

BLADE_SRC_ROOT=$(shell pwd)

BUILD_BINARY_PATH=build/build_binary
BUILD_HELM_PATH=build/helm3/chaos_agent
BUILD_IMAGE_PATH=build/build_image
BUILD_TARGET_PKG_DIR=$(BUILD_IMAGE_PATH)

VERSION_PKG=github.com/chaosblade-io/chaos-agent/version

GO_ENV=CGO_ENABLED=1
GO_MODULE=GO111MODULE=on
GO=env $(GO_ENV) $(GO_MODULE) go
#GO_X_FLAGS=-X ${VERSION_PKG}.AgentVersion=$(AGENT_VERSION) -X '${VERSION_PKG}.Env=`uname -mv`' -X '${VERSION_PKG}.BuildTime=`date`'
GO_FLAGS=-ldflags="-s -w"

ifeq ($(GOOS), linux)
	GO_FLAGS=-ldflags="-linkmode external -extldflags -static -s -w"
endif

build: build_binary

build_darwin: pre_build build_binary build_image

build_binary: cmd/chaos_agent.go
	$(GO) build $(GO_FLAGS) -o $(BUILD_TARGET_PKG_DIR)/agent $<

build_linux:
	docker build -f $(BUILD_BINARY_PATH)/Dockerfile -t agent-build-musl:latest $(BUILD_BINARY_PATH)
	docker run --rm \
    		-v $(shell echo -n ${GOPATH}):/go \
    		-w /chaos-agent \
    		-v $(BLADE_SRC_ROOT):/chaos-agent \
    		agent-build-musl:latest

build_image:
	helm package $(BUILD_HELM_PATH)
	cd $(BUILD_IMAGE_PATH)
	docker build --pull --build-arg BLADE_VERSION=${BLADE_VERSION} -f $(BUILD_IMAGE_PATH)/Dockerfile \
		-t chaos-agent:$(AGENT_VERSION) $(BLADE_SRC_ROOT)/$(BUILD_IMAGE_PATH)
