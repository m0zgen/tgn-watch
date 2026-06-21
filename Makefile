APP := tgn-watch

VERSION_FILE := version.txt
VERSION_COPY_FILE := VERSION

VERSION ?= $(shell cat $(VERSION_FILE) 2>/dev/null || echo 0.1.0)

# VERSION may be passed as v0.1.1, but binary version should be 0.1.1
BIN_VERSION := $(patsubst v%,%,$(VERSION))

COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo dev)
DATE   ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

PKG_VERSION := github.com/m0zgen/tgn-watch/internal/version

LDFLAGS := -s -w \
	-X $(PKG_VERSION).Version=$(BIN_VERSION) \
	-X $(PKG_VERSION).Commit=$(COMMIT) \
	-X $(PKG_VERSION).Date=$(DATE)

BIN_DIR := bin

.PHONY: build build-linux build-linux-amd64 build-linux-arm64 build-all
.PHONY: run test tidy clean snapshot status version
.PHONY: check-release update_version release release-current

build:
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(APP) ./cmd/$(APP)

build-linux: build-linux-amd64 build-linux-arm64

build-linux-amd64:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath \
		-ldflags "$(LDFLAGS)" \
		-o $(BIN_DIR)/linux/amd64/$(APP) \
		./cmd/$(APP)

build-linux-arm64:
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -trimpath \
		-ldflags "$(LDFLAGS)" \
		-o $(BIN_DIR)/linux/arm64/$(APP) \
		./cmd/$(APP)

build-all: build build-linux

run:
	go run ./cmd/$(APP) -config configs/config.example.yml

test:
	go test ./...

tidy:
	go mod tidy

clean:
	rm -rf bin dist

status:
	git status --short

version:
	@echo "$(BIN_VERSION)"

snapshot:
	go test ./...
	goreleaser check
	goreleaser release --snapshot --clean

update_version:
	../go-upapp-version-tool/update-version-tool*
	cat $(VERSION_FILE) > $(VERSION_COPY_FILE)

check-release:
	@if [ -n "$$(git status --short)" ]; then \
		echo "Git working tree is dirty. Commit changes first:"; \
		git status --short; \
		exit 1; \
	fi
	@echo "Git working tree is clean"

release:
	@if [ -n "$$(git status --short)" ]; then \
		echo "Git working tree is dirty. Commit changes first:"; \
		git status --short; \
		exit 1; \
	fi
	@echo "[release] updating version"
	@$(MAKE) update_version
	@NEW_VERSION="$$(cat $(VERSION_FILE) | tr -d '[:space:]')"; \
	TAG_VERSION="v$$NEW_VERSION"; \
	echo "[release] new version: $$NEW_VERSION"; \
	echo "[release] tag: $$TAG_VERSION"; \
	if ! echo "$$TAG_VERSION" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+$$'; then \
		echo "Invalid version in $(VERSION_FILE): $$NEW_VERSION"; \
		echo "Expected format: 0.1.1"; \
		exit 1; \
	fi; \
	git fetch --tags --quiet; \
	if git rev-parse "$$TAG_VERSION" >/dev/null 2>&1; then \
		echo "Tag already exists locally: $$TAG_VERSION"; \
		exit 1; \
	fi; \
	if git ls-remote --tags origin "refs/tags/$$TAG_VERSION" | grep -q "$$TAG_VERSION"; then \
		echo "Tag already exists on origin: $$TAG_VERSION"; \
		exit 1; \
	fi; \
	git add $(VERSION_FILE) $(VERSION_COPY_FILE); \
	if git diff --cached --quiet; then \
		echo "No version changes to commit."; \
		exit 1; \
	fi; \
	git commit -m "Bump version to $$NEW_VERSION"; \
	git push; \
	echo "[release] running release script"; \
	./tools/release.sh "$$TAG_VERSION"

release-current:
	@CURRENT_VERSION="$$(cat $(VERSION_FILE) | tr -d '[:space:]')"; \
	TAG_VERSION="v$$CURRENT_VERSION"; \
	echo "[release] current version: $$CURRENT_VERSION"; \
	echo "[release] tag: $$TAG_VERSION"; \
	if ! echo "$$TAG_VERSION" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+$$'; then \
		echo "Invalid version in $(VERSION_FILE): $$CURRENT_VERSION"; \
		echo "Expected format: 0.1.1"; \
		exit 1; \
	fi; \
	./tools/release.sh "$$TAG_VERSION"