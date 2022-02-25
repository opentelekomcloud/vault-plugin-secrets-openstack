SHELL=/bin/bash

export PATH:=/usr/local/go/bin:~/go/bin/:$(PATH)

GOFMT_FILES?=$$(find . -name '*.go')
WORKDIR =$(patsubst %/,%,$(dir $(realpath $(lastword $(MAKEFILE_LIST)))))
PROJECT =$(notdir $(WORKDIR))
BUILT_DIR ="$(WORKDIR)/bin"
BINARY_PATH="$(BUILT_DIR)/$(PROJECT)"

build:
	@mkdir -p $(BUILT_DIR)
	@go build -o $(BINARY_PATH)

release:
	goreleaser release

snapshot:
	goreleaser release --snapshot --parallelism 2 --rm-dist

vet:
	go vet ./...

test: vet
	go test -covermode atomic -coverprofile coverage.out ./...

cov-report:
	go tool cover -html coverage.out -o coverage.html

fmt:
	@gofmt -w $(GOFMT_FILES)

lint:
	golangci-lint run ./...

functional: build
	@VAULT_PLUGIN_DIR=$(BUILT_DIR) ./scripts/acceptance.sh
