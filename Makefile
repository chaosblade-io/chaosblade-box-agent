.PHONY: build clean

VERSION=0.0.1
BUILD_TARGET=target
BUILD_TARGET_DIR_NAME=chaos-agent-$(VERSION)
BUILD_TARGET_PKG_DIR=$(BUILD_TARGET)/chaos-agent-$(VERSION)
BUILD_TARGET_PKG_NAME=$(BUILD_TARGET)/chaos-agent-$(VERSION).tar.gz

GO_ENV=CGO_ENABLED=0

clean:
	rm -rf $(BUILD_TARGET)

build:
	rm -rf $(BUILD_TARGET_PKG_DIR)
	$(GO) build -o $(BUILD_TARGET_PKG_DIR)/agent ./

build_linux_amd64:
	rm -rf $(BUILD_TARGET_PKG_DIR)
	env GOOS=linux GOARCH=amd64 go build -o $(BUILD_TARGET_PKG_DIR)/chaosagent ./

package:
	tar zcvf $(BUILD_TARGET_PKG_NAME) -C $(BUILD_TARGET) $(BUILD_TARGET_DIR_NAME)

build_image:
	docker build --rm -t chaos-agent .

