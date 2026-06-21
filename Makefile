APP := tgn-watch
VERSION_FILE := version.txt
VERSION ?= $(shell cat $(VERSION_FILE) 2>/dev/null || echo 0.1.2)
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo dev)
DATE   ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
PKG_VERSION := github.com/m0zgen/tgn-watch/internal/version
LDFLAGS := -s -w \
	-X $(PKG_VERSION).Version=$(VERSION) \
	-X $(PKG_VERSION).Commit=$(COMMIT) \
	-X $(PKG_VERSION).Date=$(DATE)

.PHONY: tidy build run test clean

tidy:
	go mod tidy

build:
	mkdir -p bin
	go build -trimpath -ldflags "$(LDFLAGS)" -o bin/$(APP) ./cmd/$(APP)

run:
	go run ./cmd/$(APP) -config configs/config.example.yml

test:
	go test ./...

clean:
	rm -rf bin
