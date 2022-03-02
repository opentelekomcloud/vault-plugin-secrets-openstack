SHELL=/bin/bash

export PATH:=/usr/local/go/bin:~/go/bin/:$(PATH)

GOFMT_FILES?=$$(find . -name '*.go')
WORKDIR =$(patsubst %/,%,$(dir $(realpath $(lastword $(MAKEFILE_LIST)))))
PROJECT =$(notdir $(WORKDIR))

export VAULT_PLUGIN_DIR =$(WORKDIR)/bin
BINARY_PATH =$(VAULT_PLUGIN_DIR)/$(PROJECT)

# unset OS_ variables
env_vars ::= $(shell env | grep -oE 'OS_[^=]+' )
unexport $(env_vars)

build:
	@mkdir -p $(VAULT_PLUGIN_DIR)
	@go build -o $(BINARY_PATH)

release:
	goreleaser release

snapshot:
	goreleaser release --snapshot --parallelism 2 --rm-dist

vet:
	$(info Running vet...)
	@go vet ./...

test: vet
	$(info Running unit tests...)
	@go test -covermode atomic -coverprofile coverage.out ./openstack/...

cov-report:
	go tool cover -html coverage.out -o coverage.html

fmt:
	@gofmt -w $(GOFMT_FILES)

lint:
	golangci-lint run ./...

functional: build
	$(info Running acceptance tests...)
	@./scripts/acceptance.sh
