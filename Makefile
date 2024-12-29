# Makefile
GOARCH?=$(shell arch)
GOOS?=$(shell uname | awk 'print tolower($$0)}')
MODBASE=$(shell go list -m)
BINDIR=$(CURDIR)/bin
TESTDIRS=$(shell echo ./...)

# get the short commit hash
GIT_COMMIT:=$(shell git rev-parse --short HEAD)
# get the version relative to the last tag
GIT_BUILD:=$(shell git describe --dirty --always)
# get the current git branch
GIT_BRANCH:=$(shell git rev-parse --abbrev-ref HEAD)

.DEFAULT_GOAL := help

.PHONY: help
help: Makefile
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: all
all: build ## build all binaries

.PHONY: build
build: minimal groundwork ## build all binaries

.PHONY: minimal
minimal: ## build the minimal binary
	@echo "Building minimal"
	CGO_ENABLED=0 go build -o bin/minimal ./cmd/minimal/

.PHONY: groundwork
groundwork: ## build the groundwork binary
	@echo "Building groundwork"
	CGO_ENABLED=0 go build -ldflags "\
	-s -w \
	-X $(MODBASE)/internal/buildversion.GitCommit=$(GIT_COMMIT) \
	-X $(MODBASE)/internal/buildversion.Build=$(GIT_BUILD) \
	-X $(MODBASE)/internal/buildversion.GitBranch=$(GIT_BRANCH) \
	" \
	-o bin/groundwork ./cmd/groundwork/

.PHONY: test
test: ## run the unit tests
	go test -timeout=5s -v $(TESTDIRS)
