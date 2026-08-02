BINARY  := sporttrax
VERSION ?= dev
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
LDFLAGS := -s -w \
	-X github.com/sporttrax-inc/sporttrax-cli/internal/version.Version=$(VERSION) \
	-X github.com/sporttrax-inc/sporttrax-cli/internal/version.Commit=$(COMMIT)

.PHONY: build test lint vuln cross docs notices setup clean

# One-time dev setup: activate the committed git hooks and check tooling.
setup:
	git config core.hooksPath .githooks
	@command -v golangci-lint > /dev/null || echo "install golangci-lint: brew install golangci-lint"
	@command -v govulncheck > /dev/null || echo "install govulncheck: brew install govulncheck"
	@echo "hooks active (core.hooksPath=.githooks)"

# Regenerate the markdown command reference from the cobra definitions.
docs:
	rm -f docs/*.md
	go run ./cmd/gen-docs --out docs

# Regenerate third-party license notices from the linked module graph.
# Run after any dependency change; the file ships in the release archives.
notices:
	go run ./cmd/gen-notices --out THIRD_PARTY_NOTICES.md

build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/sporttrax

test:
	go test ./...

lint:
	golangci-lint run ./...

# Scan dependencies against the Go vulnerability database. Also runs in CI
# on every push, since the database moves independently of the code.
vuln:
	govulncheck ./...

# Build for every supported platform into dist/
cross:
	GOOS=darwin  GOARCH=arm64 go build -trimpath -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-darwin-arm64 ./cmd/sporttrax
	GOOS=darwin  GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-darwin-amd64 ./cmd/sporttrax
	GOOS=linux   GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-linux-amd64 ./cmd/sporttrax
	GOOS=linux   GOARCH=arm64 go build -trimpath -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-linux-arm64 ./cmd/sporttrax
	GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-windows-amd64.exe ./cmd/sporttrax
	GOOS=windows GOARCH=arm64 go build -trimpath -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-windows-arm64.exe ./cmd/sporttrax

clean:
	rm -rf bin dist
