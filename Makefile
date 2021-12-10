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
	go test -covermode atomic -coverprofile coverage.out ./...

cov-report:
	go tool cover -html coverage.out -o coverage.html

fmt:
	@gofmt -w $(GOFMT_FILES)

lint:
	golangci-lint run ./...
