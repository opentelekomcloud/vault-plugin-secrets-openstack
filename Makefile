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

module_path ::= github.com/opentelekomcloud/vault-plugin-secrets-openstack
ldflags ::= -s -w \
  -X $(module_path)/vars.ProjectName=vault-plugin-secrets-openstack \
  -X $(module_path)/vars.ProjectDocs=https://$(module_path) \
  -X $(module_path)/vars.BuildVersion=$(shell git rev-parse --abbrev-ref HEAD) \
  -X $(module_path)/vars.BuildRevision=$(shell git rev-parse --short HEAD) \
  -X $(module_path)/vars.BuildDate=$(shell date --iso-8601)

build:
	@mkdir -p $(VAULT_PLUGIN_DIR)
	@go build -o $(BINARY_PATH) -ldflags "$(ldflags)"

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
	@bash ./scripts/acceptance.sh
