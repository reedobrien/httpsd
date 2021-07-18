.SILENT: ; # no need for @

PROJECT			=xrid
PROJECT_DIR		=$(shell pwd)
GOFILES         :=$(shell find . -name '*.go' -not -path './vendor/*')
GOPACKAGES      :=$(shell go list ./... | grep -v /vendor/| grep -v /checkers)
OS              := $(shell go env GOOS)
ARCH            := $(shell go env GOARCH)

GITHASH         :=$(shell git rev-parse --short HEAD)
GITBRANCH       :=$(shell git rev-parse --abbrev-ref HEAD)
GITTAGORBRANCH 	:=$(shell sh -c 'git describe --always --dirty 2>/dev/null')
BUILDDATE      	:=$(shell date -u +%Y%m%d%H%M)
GO_LDFLAGS		?= -s -w
GO_BUILD_FLAGS  :=-ldflags "${GOLDFLAGS} -X main.BuildVersion=${GITTAGORBRANCH} -X main.GitHash=${GITHASH} -X main.GitBranch=${GITBRANCH} -X main.BuildDate=${BUILDDATE}"


## What if there's no CIRCLE_BUILD_NUM
ifeq ($$CIRCLE_BUILD_NUM, "")
		BUILD_NUM:=""
else
		CB:=$$CIRCLE_BUILD_NUM
		BUILD_NUM:=$(CB)/
endif

WORKDIR         :=$(PROJECT_DIR)/_workdir

default: build-linux

build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $(WORKDIR)/$(PROJECT)_linux_amd64 $(GO_BUILD_FLAGS)

build:
	CGO_ENABLED=0 go build -o $(WORKDIR)/$(PROJECT)_$(OS)_$(ARCH) $(GO_BUILD_FLAGS)

clean:
	rm -f $(WORKDIR)/*
	rm -rf .cover
	go clean -r

coverage:
	./_misc/coverage.sh

coverage-html:
	./_misc/coverage.sh --html

dependencies:
	go get honnef.co/go/tools/cmd/megacheck
	go get github.com/alecthomas/gometalinter
	go get github.com/golang/dep/cmd/dep
	dep ensure
	gometalinter --install

develop: dependencies
	(cd .git/hooks && ln -sf ../../_misc/pre-push.bash pre-push )

lint:
	# TODO(ro) 2018-12-27 https://github.com/golangci/golangci-lint is a new player. Try it?
# https://github.com/alecthomas/gometalinter#supported-linters and the enabled
# ones here.
# At the time of this writing megacheck runs gosimple, staticcheck, and
# unused. All production honnef tools.
	# gometalinter --enable=goimports --enable=unparam --enable=unused --disable=golint --disable=govet .
	echo "metalinter..."
	gometalinter --enable=goimports --enable=unparam --enable=unused --disable=golint --disable=govet $(GOPACKAGES)
	echo "megacheck..."
	megacheck $(GOPACKAGES)
	echo "golint..."
	golint -set_exit_status $(GOPACKAGES)
	echo "go vet..."
	go vet --all $(GOPACKAGES)

test:
	CGO_ENABLED=0 go test $(GOPACKAGES)

test-race:
	CGO_ENABLED=1 go test -race $(GOPACKAGES)
