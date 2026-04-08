PROJECT_NAME := "clock-8001"
PKG := "gitlab.com/clock-8001/$(PROJECT_NAME)/v4"
PKG_LIST := $(shell go list ${PKG}/... | grep -v /vendor/)
GO_FILES := $(shell find . -name '*.go' | grep -v /vendor/ | grep -v _test.go)
BINARIES := clock-bridge matrix-clock sdl-clock multi-clock sdl-clock.exe
GOLINT := "$(GOPATH)/bin/golint"
GIT_TAG ?= $(shell git describe --tags --abbrev=0 HEAD)
GIT_COMMIT ?= $(shell git rev-list -1 HEAD)
VERSION_PKG := gitlab.com/clock-8001/clock-8001/v4/clock
GO_LD_FLAGS := "-X $(VERSION_PKG).gitCommit=$(GIT_COMMIT) -X $(VERSION_PKG).gitTag=$(GIT_TAG)"
WIN_VERSION := $(GIT_TAG:v%=%)

.PHONY: all dep build clean test coverage coverhtml lint linux macos sdl3-clock_linux_amd64 sdl3-clock_linux_arm64 sdl3-clock sdl-clock

all: build

lint: ## Lint the files
	@golint -set_exit_status ./...

test: ## Run unittests
	@go test -short ./clock ./picturall ./vmix

race: ## Run data race detector
	@go test -race -short ./...

msan: ## Run memory sanitizer
	@go test -msan -short ./...

coverage: ## Generate global code coverage report
	@go test $(PKG_LIST) -v -coverprofile .coverage.txt
	@go tool cover -func .coverage.txt

coverhtml: ## Generate global code coverage report in HTML
	@go tool cover -html=.coverage.txt

clean:
	@rm -f $(BINARIES)
	@rm -f sdl-clock sdl-clock.exe multi-clock matrix-clock clock_port80.ini clock_port8080.ini clock-8001.msi sdl-clock_amd64 sdl-clock_arm64
	@rm -fr windows

build: sdl-clock sdl3-clock ## Build the binary file

matrix-clock:
	@echo Building matrix-clock tag $(GIT_TAG) commit $(GIT_COMMIT)
	@go build -ldflags $(GO_LD_FLAGS) gitlab.com/clock-8001/clock-8001/v4/cmd/matrix-clock

multi-clock:
	@echo Building multi-clock tag $(GIT_TAG) commit $(GIT_COMMIT)
	@go build -ldflags $(GO_LD_FLAGS) gitlab.com/clock-8001/clock-8001/v4/cmd/multi-clock

sdl-clock:
	@echo Building sdl-clock tag $(GIT_TAG) commit $(GIT_COMMIT)
	@go build -ldflags $(GO_LD_FLAGS) gitlab.com/clock-8001/clock-8001/v4/cmd/sdl-clock

sdl3-clock:
	@echo Building sdl-clock tag $(GIT_TAG) commit $(GIT_COMMIT)
	@go build -ldflags $(GO_LD_FLAGS) gitlab.com/clock-8001/clock-8001/v4/cmd/sdl3-clock

config: clock_port80.ini clock_port8080.ini

clock_port80.ini: sdl3-clock
	./sdl3-clock --http-port :80 --countdown-target "$(shell date +%Y)-12-24 00:00:00" --voice-dir /boot/voices/us --defaults --dump-config > clock_port80.ini

clock_port8080.ini: sdl3-clock
	./sdl3-clock --http-port :8080 --countdown-target "$(shell date +%Y)-12-24 00:00:00" --defaults --dump-config > clock_port8080.ini


windows: clock_port8080.ini ## Build the binary file
	@echo Building sdl3-clock.exe tag $(GIT_TAG) commit $(GIT_COMMIT) for windows
	$(shell export GOOS=windows; export GOARCH=amd64; go build -ldflags "-X $(VERSION_PKG).gitCommit=$(GIT_COMMIT) -X $(VERSION_PKG).gitTag=$(GIT_TAG)" gitlab.com/clock-8001/clock-8001/v4/cmd/sdl3-clock)
	@echo Building the msi
	@mkdir -p windows
	@cp ttf_fonts/* windows/
	@cp -r fonts windows/
	@cp clock_port8080.ini windows/clock.ini
	@cp sdl3-clock.exe windows/
	@cp -r voices windows/
	@mkdir -p windows/bg/timer0
	@mkdir -p windows/bg/timer1
	@mkdir -p windows/bg/timer2
	@mkdir -p windows/bg/timer3
	@mkdir -p windows/bg/timer4
	@mkdir -p windows/bg/timer5
	@mkdir -p windows/bg/timer6
	@mkdir -p windows/bg/timer7
	@mkdir -p windows/bg/timer8
	@mkdir -p windows/bg/timer9
	@msi-packager windows/ clock-8001.msi -l -n clock-8001 -v $(WIN_VERSION) -a x86 -u 28bebbbc-0430-4ab0-a322-1660d5824663 -e "sdl3-clock.exe" -m Depili -i cmd/sdl3-clock/winres/icon.ico

linux: sdl3-clock_linux_amd64 sdl3-clock_linux_arm64

sdl3-clock_linux_amd64:
	@echo Building tag $(GIT_TAG) commit $(GIT_COMMIT) for linux amd64
	$(shell export GOOS=linux; export GOARCH=amd64; go build -ldflags "-X $(VERSION_PKG).gitCommit=$(GIT_COMMIT) -X $(VERSION_PKG).gitTag=$(GIT_TAG)" -o sdl3-clock_linux_amd64 gitlab.com/clock-8001/clock-8001/v4/cmd/sdl3-clock)

sdl3-clock_linux_arm64:
	@echo Building tag $(GIT_TAG) commit $(GIT_COMMIT) for linux arm64
	$(shell export GOOS=linux; export GOARCH=arm64; go build -ldflags "-X $(VERSION_PKG).gitCommit=$(GIT_COMMIT) -X $(VERSION_PKG).gitTag=$(GIT_TAG)" -o sdl3-clock_linux_arm64 gitlab.com/clock-8001/clock-8001/v4/cmd/sdl3-clock)

macos:
	@echo Building tag $(GIT_TAG) commit $(GIT_COMMIT) for macos
	$(shell export GOOS=darwin; export GOARCH=amd64; go build -ldflags "-X $(VERSION_PKG).gitCommit=$(GIT_COMMIT) -X $(VERSION_PKG).gitTag=$(GIT_TAG)" -o sdl3-clock_darwin_amd64 gitlab.com/clock-8001/clock-8001/v4/cmd/sdl3-clock)
	$(shell export GOOS=darwin; export GOARCH=arm64; go build -ldflags "-X $(VERSION_PKG).gitCommit=$(GIT_COMMIT) -X $(VERSION_PKG).gitTag=$(GIT_TAG)" -o sdl3-clock_darwin_arm64 gitlab.com/clock-8001/clock-8001/v4/cmd/sdl3-clock)

osxcross:
	@echo Building tag $(GIT_TAG) commit $(GIT_COMMIT) for macos via osxcross
	$(shell export CGO_ENABLED=1; export export CC=x86_64-apple-darwin18-clang; export GOOS=darwin; export GOARCH=amd64; go build -tags static -ldflags "-s -w -X $(VERSION_PKG).gitCommit=$(GIT_COMMIT) -X $(VERSION_PKG).gitTag=$(GIT_TAG)" -o sdl3-clock_amd64_darwin gitlab.com/clock-8001/clock-8001/v4/cmd/sdl3-clock)
	$(shell export CGO_ENABLED=1; export export CC=x86_64-apple-darwin18-clang; export GOOS=darwin; export GOARCH=amd64; go build -tags static -ldflags "-s -w -X $(VERSION_PKG).gitCommit=$(GIT_COMMIT) -X $(VERSION_PKG).gitTag=$(GIT_TAG)" -o sdl-clock_amd64_darwin gitlab.com/clock-8001/clock-8001/v4/cmd/sdl-clock)

debian_arm64: clock_port8080.ini sdl3-clock_linux_arm64
	cp sdl3-clock_linux_arm64 sdl3-clock_linux.tmp
	@export GIT_TAG=$(GIT_TAG); export NFPM_ARCH=arm64; nfpm pkg --packager deb

debian_amd64: clock_port8080.ini sdl3-clock_linux_amd64
	cp sdl3-clock_linux_amd64 sdl3-clock_linux.tmp
	@export GIT_TAG=$(GIT_TAG); export NFPM_ARCH=amd64; nfpm pkg --packager deb

help: ## Display this help screen
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
