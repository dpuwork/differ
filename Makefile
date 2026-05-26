.PHONY: build install test clean release-dry-run dev

VERSION ?= dev
LDFLAGS := -s -w -X github.com/dpuwork/differ/cmd.version=$(VERSION)

dev:
	mise x go -- go run .

build:
	go build -ldflags "$(LDFLAGS)" -o bin/differ .

install:
	go install -ldflags "$(LDFLAGS)" .

test:
	go test ./...

clean:
	rm -rf bin/ dist/

release-dry-run:
	goreleaser release --snapshot --clean
