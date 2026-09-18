GO ?= go
APP_NAME ?= microsctl
DIST_DIR ?= dist
BUILD_FLAGS ?= -trimpath

DARWIN_ARM64_BINARY := $(DIST_DIR)/$(APP_NAME)-darwin-arm64
LINUX_AMD64_BINARY := $(DIST_DIR)/$(APP_NAME)-linux-amd64
WINDOWS_AMD64_BINARY := $(DIST_DIR)/$(APP_NAME)-windows-amd64.exe

.PHONY: all build macos-arm64 linux-amd64 windows-amd64 clean help

all: build

build: macos-arm64 linux-amd64 windows-amd64

macos-arm64: | $(DIST_DIR)
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GO) build $(BUILD_FLAGS) -o $(DARWIN_ARM64_BINARY) .

linux-amd64: | $(DIST_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build $(BUILD_FLAGS) -o $(LINUX_AMD64_BINARY) .

windows-amd64: | $(DIST_DIR)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GO) build $(BUILD_FLAGS) -o $(WINDOWS_AMD64_BINARY) .

$(DIST_DIR):
	mkdir -p $(DIST_DIR)

clean:
	rm -f $(DARWIN_ARM64_BINARY) $(LINUX_AMD64_BINARY) $(WINDOWS_AMD64_BINARY)

help:
	@echo "Available targets:"
	@echo "  make build           Build all supported platform binaries (default)"
	@echo "  make macos-arm64     Build macOS ARM64 binary"
	@echo "  make linux-amd64     Build Linux x64 binary"
	@echo "  make windows-amd64   Build Windows x64 binary"
	@echo "  make clean           Remove built binaries from $(DIST_DIR)"
	@echo "  make help            Show this help"
