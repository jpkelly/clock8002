# Base image: https://hub.docker.com/_/golang/
FROM golang:1.24
MAINTAINER Vesa-Pekka Palmu <vpalmu@depili.fi>

# Install golint
ENV GOPATH=/go
ENV PATH=${GOPATH}/bin:$PATH
RUN go install golang.org/x/lint/golint@latest
RUN go install github.com/goreleaser/nfpm/v2/cmd/nfpm@v2.28.0

# Install clang from LLVM repository and sdl2 headers
RUN apt-get update && apt-get install -y --no-install-recommends \
    clang \
    libsdl2-dev \
    libsdl2-gfx-dev \
    libsdl2-ttf-dev \
    libsdl2-image-dev \
    libsdl2-mixer-dev \
    mingw-w64 \
    nodejs \
    npm \
    wixl \
    msitools \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

RUN npm install -g depili/msi-packager#isolate-mainExecutableFile-error-to-runAfter
