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

.PHONY: all dep build clean test coverage coverhtml lint

all: build

lint: ## Lint the files
	@golint -set_exit_status ./...

test: ## Run unittests
	@go test -short ./clock ./picturall

race: dep ## Run data race detector
	@go test -race -short ./...

msan: dep ## Run memory sanitizer
	@go test -msan -short ./...

coverage: ## Generate global code coverage report
	@go test $(PKG_LIST) -v -coverprofile .coverage.txt
	@go tool cover -func .coverage.txt

coverhtml: ## Generate global code coverage report in HTML
	@go tool cover -html=.coverage.txt

dep: ## Get the dependencies
	@go get -v -d ./...

clean:
	@rm -f $(BINARIES)
	@rm -f sdl-clock sdl-clock.exe multi-clock matrix-clock clock_port80.ini clock_port8080.ini clock-8001.msi sdl-clock_amd64 sdl-clock_arm64
	@rm -fr windows

build: dep sdl-clock ## Build the binary file

matrix-clock:
	@echo Building matrix-clock tag $(GIT_TAG) commit $(GIT_COMMIT)
	@go build -ldflags $(GO_LD_FLAGS) gitlab.com/clock-8001/clock-8001/v4/cmd/matrix-clock

multi-clock:
	@echo Building multi-clock tag $(GIT_TAG) commit $(GIT_COMMIT)
	@go build -ldflags $(GO_LD_FLAGS) gitlab.com/clock-8001/clock-8001/v4/cmd/multi-clock

sdl-clock:
	@echo Building sdl-clock tag $(GIT_TAG) commit $(GIT_COMMIT)
	@go build -ldflags $(GO_LD_FLAGS) gitlab.com/clock-8001/clock-8001/v4/cmd/sdl-clock

config: clock_port80.ini clock_port8080.ini

clock_port80.ini: sdl-clock
	./sdl-clock --http-port :80 --countdown-target "$(shell date +%Y)-12-24 00:00:00" --voice-dir /boot/voices/us --defaults --dump-config > clock_port80.ini

clock_port8080.ini: sdl-clock
	./sdl-clock --http-port :8080 --countdown-target "$(shell date +%Y)-12-24 00:00:00" --defaults --dump-config > clock_port8080.ini


windows: dep clock_port8080.ini ## Build the binary file
	@echo Building sdl-clock.exe tag $(GIT_TAG) commit $(GIT_COMMIT) for windows
	$(shell export CGO_ENABLED=1; export CC=x86_64-w64-mingw32-gcc; export GOOS=windows; export GOARCH=amd64; go build -tags static -ldflags "-s -w -X $(VERSION_PKG).gitCommit=$(GIT_COMMIT) -X $(VERSION_PKG).gitTag=$(GIT_TAG) -H windowsgui" gitlab.com/clock-8001/clock-8001/v4/cmd/sdl-clock)
	@echo Building the msi
	@mkdir -p windows
	@cp ttf_fonts/* windows/
	@cp -r fonts windows/
	@cp clock_port8080.ini windows/clock.ini
	@cp sdl-clock.exe windows/
	@cp -r voices windows/
	@msi-packager windows/ clock-8001.msi -l -n clock-8001 -v $(WIN_VERSION) -a x86 -u 28bebbbc-0430-4ab0-a322-1660d5824663 -e "sdl-clock.exe" -m Depili -i cmd/sdl-clock/winres/icon.ico

macos:
	@echo Building tag $(GIT_TAG) commit $(GIT_COMMIT) for macos
	$(shell export GOOS=darwin; export GOARCH=amd64; go build -tags static -ldflags "-s -w -X $(VERSION_PKG).gitCommit=$(GIT_COMMIT) -X $(VERSION_PKG).gitTag=$(GIT_TAG)" -o sdl-clock_amd64 gitlab.com/clock-8001/clock-8001/v4/cmd/sdl-clock)

osxcross:
	@echo Building tag $(GIT_TAG) commit $(GIT_COMMIT) for macos via osxcross
	$(shell export CGO_ENABLED=1; export export CC=x86_64-apple-darwin18-clang; export GOOS=darwin; export GOARCH=amd64; go build -tags static -ldflags "-s -w -X $(VERSION_PKG).gitCommit=$(GIT_COMMIT) -X $(VERSION_PKG).gitTag=$(GIT_TAG)" -o sdl-clock_amd64_darwin gitlab.com/clock-8001/clock-8001/v4/cmd/sdl-clock)

help: ## Display this help screen
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
