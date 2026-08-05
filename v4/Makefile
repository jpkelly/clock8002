PROJECT_NAME := "clock-8002"
PKG := "gitlab.com/clock-8001/clock-8001/v4"
PKG_LIST := $(shell go list ${PKG}/... | grep -v /vendor/)
GO_FILES := $(shell find . -name '*.go' | grep -v /vendor/ | grep -v _test.go)
BINARIES := sdl-clock
GOLINT := "$(GOPATH)/bin/golint"
GIT_TAG ?= $(shell git describe --tags --abbrev=0 HEAD 2>/dev/null || echo "v0.0.1")
GIT_COMMIT ?= $(shell git rev-list -1 HEAD)
VERSION_PKG := gitlab.com/clock-8001/clock-8001/v4/clock
GO_LD_FLAGS := "-X $(VERSION_PKG).gitCommit=$(GIT_COMMIT) -X $(VERSION_PKG).gitTag=$(GIT_TAG)"
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
ALSA_LTC_CFLAGS := -O2 -DALSA_LTC_GIT_TAG=\"$(GIT_TAG)\" -DALSA_LTC_GIT_COMMIT=\"$(GIT_COMMIT)\" -DALSA_LTC_BUILD_DATE=\"$(BUILD_DATE)\"
NETWORK_CONFIG ?= default
RELEASE_DIR := piClock-$(GIT_TAG)-linux-arm64
WIN_VERSION := $(GIT_TAG:v%=%)

.PHONY: all dep build clean test release release-all linux sdl-clock sdl3-clock sdl3-clock_linux_amd64 sdl3-clock_linux_arm64

all: build

test: ## Run unittests
	@go test -short ./clock

dep: ## Get the dependencies
	@go get -v -d ./...

clean:
	@rm -f sdl-clock sdl3-clock alsa-ltc
	@rm -fr piClock-*-linux-arm64 piClock-*-linux-arm64.tar.gz

build: sdl-clock ## Build the binary file

sdl-clock: sdl3-clock
	@cp sdl3-clock sdl-clock

sdl3-clock:
	@echo Building sdl3-clock tag $(GIT_TAG) commit $(GIT_COMMIT)
	@go build -ldflags $(GO_LD_FLAGS) gitlab.com/clock-8001/clock-8001/v4/cmd/sdl3-clock

alsa-ltc: alsa-ltc.c ## Build the LTC decoder
	@echo Building alsa-ltc tag $(GIT_TAG) commit $(GIT_COMMIT) built $(BUILD_DATE)
	gcc $(ALSA_LTC_CFLAGS) -o alsa-ltc alsa-ltc.c -lasound -lltc

release: sdl-clock alsa-ltc ## Package an installer tarball for NETWORK_CONFIG variant
	@echo Packaging release $(GIT_TAG) variant=$(NETWORK_CONFIG)
	@mkdir -p $(RELEASE_DIR)
	@cp sdl-clock $(RELEASE_DIR)/
	@cp alsa-ltc $(RELEASE_DIR)/
	@cp boot-README.txt $(RELEASE_DIR)/
	@if [ -d lib ]; then mkdir -p $(RELEASE_DIR)/lib && cp -P lib/*.so* $(RELEASE_DIR)/lib/; fi
	@cp clock.ini.$(NETWORK_CONFIG) $(RELEASE_DIR)/clock.ini
	@cp ttf_fonts/*.ttf $(RELEASE_DIR)/
	@cp -r fonts $(RELEASE_DIR)/
	@cp -r voices $(RELEASE_DIR)/
	@cp -r oled $(RELEASE_DIR)/
	@cp -r splash $(RELEASE_DIR)/
	@cp clock8002.service $(RELEASE_DIR)/
	@cp alsa-ltc.service $(RELEASE_DIR)/
	@cp piclock-network.sh $(RELEASE_DIR)/
	@cp piclock-network.service $(RELEASE_DIR)/
	@cp piclock-authorized-keys.sh $(RELEASE_DIR)/
	@cp piclock-authorized-keys.service $(RELEASE_DIR)/
	@cp install.sh $(RELEASE_DIR)/
	@cp network.ini.$(NETWORK_CONFIG) $(RELEASE_DIR)/network.ini
	@chmod +x $(RELEASE_DIR)/install.sh
	@(cd $(RELEASE_DIR) && find . -type f ! -name SHA256SUMS -print0 | sort -z | xargs -0 sha256sum > SHA256SUMS)
	@tar czf $(RELEASE_DIR).tar.gz $(RELEASE_DIR)
	@rm -rf $(RELEASE_DIR)
	@echo Release tarball: $(RELEASE_DIR).tar.gz

release-all: sdl-clock alsa-ltc ## Package release tarballs for all variants
	$(MAKE) release NETWORK_CONFIG=default

linux: sdl3-clock_linux_amd64 sdl3-clock_linux_arm64

sdl3-clock_linux_amd64:
	@echo Building tag $(GIT_TAG) commit $(GIT_COMMIT) for linux amd64
	$(shell export GOOS=linux; export GOARCH=amd64; go build -ldflags "-X $(VERSION_PKG).gitCommit=$(GIT_COMMIT) -X $(VERSION_PKG).gitTag=$(GIT_TAG)" -o sdl3-clock_linux_amd64 gitlab.com/clock-8001/clock-8001/v4/cmd/sdl3-clock)

sdl3-clock_linux_arm64:
	@echo Building tag $(GIT_TAG) commit $(GIT_COMMIT) for linux arm64
	$(shell export GOOS=linux; export GOARCH=arm64; go build -ldflags "-X $(VERSION_PKG).gitCommit=$(GIT_COMMIT) -X $(VERSION_PKG).gitTag=$(GIT_TAG)" -o sdl3-clock_linux_arm64 gitlab.com/clock-8001/clock-8001/v4/cmd/sdl3-clock)

help: ## Display this help screen
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
