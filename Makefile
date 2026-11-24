# WordlistKingdom build and development tasks.
#
# The Go toolchain is expected on PATH. Override it with:
#   make GO=/path/to/go build
# or use the standard environment variable:
#   make GO=go1.21 build

SHELL   := /bin/bash
VERSION ?= 2.0.0
BINARY  ?= wordlistkingdom
GO      ?= go
GOFMT   ?= gofmt
LDFLAGS := -s -w -X main.version=$(VERSION)
GOFILES := $(shell find . -name '*.go' -not -path './.tmp/*' -not -path './dist/*')

.PHONY: all build test race vet fmt check bench fuzz cross checksums release install clean help

all: build

build:
	$(GO) build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) .

test:
	$(GO) test ./...

race:
	$(GO) test -race ./...

vet:
	$(GO) vet ./...

fmt:
	@files="$$($(GOFMT) -l $(GOFILES))"; \
	if [ -n "$$files" ]; then echo "$$files"; $(GOFMT) -w $$files; fi

check:
	@files="$$($(GOFMT) -l $(GOFILES))"; \
	if [ -n "$$files" ]; then echo "unformatted files:"; echo "$$files"; exit 1; fi
	$(GO) vet ./...
	$(GO) test ./...

bench:
	$(GO) test ./pkg/generate -run '^$$' -bench . -benchmem

fuzz:
	$(GO) test ./pkg/config -run '^$$' -fuzz '^FuzzParseDates$$' -fuzztime 5s
	$(GO) test ./pkg/config -run '^$$' -fuzz '^FuzzParsePattern$$' -fuzztime 5s
	$(GO) test ./pkg/config -run '^$$' -fuzz '^FuzzParseSpecials$$' -fuzztime 5s
	$(GO) test ./pkg/source -run '^$$' -fuzz '^FuzzReadKeywords$$' -fuzztime 5s
	$(GO) test ./pkg/source -run '^$$' -fuzz '^FuzzReadUserdata$$' -fuzztime 5s
	$(GO) test ./pkg/textutil -run '^$$' -fuzz '^FuzzForms$$' -fuzztime 5s
	$(GO) test ./pkg/policy -run '^$$' -fuzz '^FuzzCheck$$' -fuzztime 5s

cross:
	mkdir -p dist
	GOOS=linux   GOARCH=amd64 $(GO) build -trimpath -ldflags "$(LDFLAGS)" -o dist/wordlistkingdom-linux-amd64 .
	GOOS=linux   GOARCH=arm64 $(GO) build -trimpath -ldflags "$(LDFLAGS)" -o dist/wordlistkingdom-linux-arm64 .
	GOOS=darwin  GOARCH=amd64 $(GO) build -trimpath -ldflags "$(LDFLAGS)" -o dist/wordlistkingdom-darwin-amd64 .
	GOOS=darwin  GOARCH=arm64 $(GO) build -trimpath -ldflags "$(LDFLAGS)" -o dist/wordlistkingdom-darwin-arm64 .
	GOOS=windows GOARCH=amd64 $(GO) build -trimpath -ldflags "$(LDFLAGS)" -o dist/wordlistkingdom-windows-amd64.exe .
	@echo "binaries written to dist/"

checksums:
	@cd dist && if command -v sha256sum >/dev/null 2>&1; then \
		sha256sum wordlistkingdom-* > SHA256SUMS; \
	else \
		shasum -a 256 wordlistkingdom-* > SHA256SUMS; \
	fi
	@cat dist/SHA256SUMS

release: check cross checksums

install:
	$(GO) install -trimpath -ldflags "$(LDFLAGS)" .

clean:
	rm -f $(BINARY)
	rm -rf dist .tmp

help:
	@printf '%s\n' \
	  'WordlistKingdom targets:' \
	  '  make build    build the current platform binary' \
	  '  make test     run unit and integration tests' \
	  '  make race     run the race detector' \
	  '  make vet      run go vet' \
	  '  make fmt      format all Go files' \
	  '  make check    fmt check + vet + tests (CI equivalent)' \
	  '  make bench    run the generator benchmark' \
	  '  make fuzz     run short fuzz sessions for parsers' \
	  '  make cross    build Linux, macOS and Windows binaries into dist/' \
	  '  make checksums  write dist/SHA256SUMS' \
	  '  make release  check + cross + checksums' \
	  '  make install  install the binary with the version stamped' \
	  '  make clean    remove build artefacts'
