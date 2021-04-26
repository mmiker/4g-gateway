SHELL = /bin/bash
PROJECT_NAME    := "github.com/zsy-cn/4g-gateway"
PKG             := "$(PROJECT_NAME)"
PKG_LIST        := $(shell go list ${PKG}/... | grep -v /vendor/)
NOW             = $(shell date -u '+%Y%m%d%I%M%S')
APP             = 4g-gateway
RELEASE_VERSION = v1.0.0
GIT_COUNT 		= $(shell git rev-list --all --count)
GIT_HASH        = $(shell git rev-parse --short HEAD)
RELEASE_TAG     = $(RELEASE_VERSION).$(GIT_COUNT).$(GIT_HASH)

arm64:
	@GOOS=linux CGO_ENABLED=0 GOARCH=arm64 go build -v -o 4g-gateway main.go

arm32:
	@GOOS=linux CGO_ENABLED=0 GOARCH=arm go build -v -o 4g-gateway main.go

mac:
	@GOOS=darwin CGO_ENABLED=1 GOARCH=amd64 go build -v -race -o 4g-gateway main.go

run:
	@go run -ldflags "-X main.VERSION=$(RELEASE_TAG)" ./main.go config -c ./etc/config-local.ini