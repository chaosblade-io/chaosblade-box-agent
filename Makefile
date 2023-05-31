.PHONE: build clean
export AGENT_VERSION = 1.0.3
export BLADE_VERSION = 1.7.2

BLADE_SRC_ROOT=$(shell pwd)

BUILD_BINARY_MUSL_PATH=build/binary_musl
BUILD_BINARY_ARM64_PATH=build/binary_arm
BUILD_IMAGE_MUSL_PATH=build/image_musl
BUILD_IMAGE_ARM64_PATH=build/image_arm


BUILD_HELM_PATH=build/helm3/chaos-agent/
BUILD_BINARY_PATH=build

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

build_darwin: pre_build build_binary build_image build_chart

build_binary: cmd/chaos_agent.go
	$(GO) build $(GO_FLAGS) -o $(BUILD_BINARY_PATH)/agent $<

build_linux:
	docker build -f $(BUILD_BINARY_MUSL_PATH)/Dockerfile -t agent-build-musl:latest $(BUILD_BINARY_MUSL_PATH)
	docker run --rm \
    		-v $(shell echo -n ${GOPATH}):/go \
    		-w /chaos-agent \
    		-v $(BLADE_SRC_ROOT):/chaos-agent \
    		agent-build-musl:latest
build_arm64:
	docker build -f $(BUILD_BINARY_ARM64_PATH)/Dockerfile -t agent-build-arm:latest $(BUILD_BINARY_ARM64_PATH)
	docker run --rm \
    		-v $(shell echo -n ${GOPATH}):/go \
    		-w /chaos-agent \
    		-v $(BLADE_SRC_ROOT):/chaos-agent \
    		agent-build-arm:latest

build_chart:
	helm package $(BUILD_HELM_PATH)

build_image:
	rm -rf $(BUILD_IMAGE_MUSL_PATH)/agent
	cp $(BUILD_BINARY_PATH)/agent $(BUILD_IMAGE_MUSL_PATH)
	docker build --pull --build-arg BLADE_VERSION=${BLADE_VERSION} -f $(BUILD_IMAGE_MUSL_PATH)/Dockerfile \
		-t chaosbladeio/chaosblade-agent:$(AGENT_VERSION) $(BLADE_SRC_ROOT)/$(BUILD_IMAGE_MUSL_PATH)

build_image_arm:
	rm -rf $(BUILD_IMAGE_ARM64_PATH)/agent
	cp $(BUILD_BINARY_PATH)/agent $(BUILD_IMAGE_ARM64_PATH)
	docker build --pull --build-arg BLADE_VERSION=${BLADE_VERSION} -f $(BUILD_IMAGE_ARM64_PATH)/Dockerfile \
		-t chaosbladeio/chaosblade-agent-arm64:$(AGENT_VERSION) $(BLADE_SRC_ROOT)/$(BUILD_IMAGE_ARM64_PATH)
