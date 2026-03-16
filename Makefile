.PHONY: build test vet fmt lint install clean all

BINARY  := tmux-orchid
BINDIR  := bin
PREFIX  ?= /usr/local

all: build

## build: compile the binary
build:
	go build -o $(BINDIR)/$(BINARY) .

## test: run all tests with the race detector
test:
	go test ./... -race

## vet: run go vet
vet:
	go vet ./...

## fmt: format code with gofumpt (falls back to gofmt)
fmt:
	@if command -v gofumpt >/dev/null 2>&1; then \
		gofumpt -w .; \
	else \
		gofmt -w .; \
	fi

## lint: run vet + test
lint: vet test

## install: build and install to PREFIX/bin
install: build
	install -d $(PREFIX)/bin
	install -m 755 $(BINDIR)/$(BINARY) $(PREFIX)/bin/$(BINARY)

## uninstall: remove installed binary
uninstall:
	rm -f $(PREFIX)/bin/$(BINARY)

## clean: remove build artifacts
clean:
	rm -rf $(BINDIR)

## tidy: clean up go.mod / go.sum
tidy:
	go mod tidy

## check: full CI pipeline (fmt + vet + test + build)
check: fmt vet test build
	@echo "all checks passed"
