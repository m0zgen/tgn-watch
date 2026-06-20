APP := tgn-watch
BIN_DIR := bin
VERSION_FILE := version.txt
VERSION ?= $(shell cat $(VERSION_FILE) 2>/dev/null || echo 0.1.0)
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo dev)
DATE   ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
PKG_VERSION := github.com/m0zgen/tgn-watch/internal/version
LDFLAGS := -s -w \
    -X $(PKG_VERSION).Version=$(VERSION) \
    -X $(PKG_VERSION).Commit=$(COMMIT) \
    -X $(PKG_VERSION).Date=$(DATE)

.PHONY: build build-linux build-linux-amd64 build-linux-arm64 build-all
.PHONY: tidy run test clean

build:
	mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(APP) ./cmd/$(APP)

build-linux: build-linux-amd64 build-linux-arm64

build-linux-amd64:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build \
		-ldflags "$(LDFLAGS)" \
		-o $(BIN_DIR)/linux/amd64/$(APP) \
		./cmd/$(APP)

build-linux-arm64:
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build \
		-ldflags "$(LDFLAGS)" \
		-o $(BIN_DIR)/linux/arm64/$(APP) \
		./cmd/$(APP)

build-all: build build-linux

tidy:
	go mod tidy

run:
	go run ./cmd/$(APP) -config configs/config.example.yml

test:
	go test ./...

clean:
	rm -rf $(BIN_DIR)