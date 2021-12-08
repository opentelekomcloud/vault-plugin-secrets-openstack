SHELL=/bin/bash

export PATH:=/usr/local/go/bin:~/go/bin/:$(PATH)

GOFMT_FILES?=$$(find . -name '*.go')

release:
	goreleaser release

snapshot:
	goreleaser release --snapshot --parallelism 2 --rm-dist

vet:
	go vet ./...

test: vet
	go test -v ./...

fmt:
	@gofmt -w $(GOFMT_FILES)

lint:
	golangci-lint run ./...
