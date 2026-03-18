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
RELEASE_DIR := clock8002-$(GIT_TAG)-linux-arm64

.PHONY: all dep build clean test release

all: build

test: ## Run unittests
	@go test -short ./clock

dep: ## Get the dependencies
	@go get -v -d ./...

clean:
	@rm -f sdl-clock alsa-ltc
	@rm -fr clock8002-*-linux-arm64 clock8002-*-linux-arm64.tar.gz

build: dep sdl-clock ## Build the binary file

sdl-clock:
	@echo Building sdl-clock tag $(GIT_TAG) commit $(GIT_COMMIT)
	@go build -ldflags $(GO_LD_FLAGS) gitlab.com/clock-8001/clock-8001/v4/cmd/sdl-clock

alsa-ltc: alsa-ltc.c ## Build the LTC decoder
	@echo Building alsa-ltc
	gcc -O2 -o alsa-ltc alsa-ltc.c -lasound -lltc

release: sdl-clock alsa-ltc ## Package a release tarball
	@echo Packaging release $(GIT_TAG)
	@mkdir -p $(RELEASE_DIR)
	@cp sdl-clock $(RELEASE_DIR)/
	@cp alsa-ltc $(RELEASE_DIR)/
	@cp clock.ini $(RELEASE_DIR)/
	@cp ttf_fonts/*.ttf $(RELEASE_DIR)/
	@cp -r fonts $(RELEASE_DIR)/
	@cp -r voices $(RELEASE_DIR)/
	@cp -r oled $(RELEASE_DIR)/
	@cp -r splash $(RELEASE_DIR)/
	@cp clock8002.service $(RELEASE_DIR)/
	@cp alsa-ltc.service $(RELEASE_DIR)/
	@cp piclock-network.sh $(RELEASE_DIR)/
	@cp piclock-network.service $(RELEASE_DIR)/
	@cp install.sh $(RELEASE_DIR)/
	@cp network.ini $(RELEASE_DIR)/
	@chmod +x $(RELEASE_DIR)/install.sh
	@tar czf $(RELEASE_DIR).tar.gz $(RELEASE_DIR)
	@rm -rf $(RELEASE_DIR)
	@echo Release tarball: $(RELEASE_DIR).tar.gz

debian_amd64: clock_port8080.ini sdl-clock
	@export GIT_TAG=$(GIT_TAG); export NFPM_ARCH=amd64; nfpm pkg --packager deb

help: ## Display this help screen
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
