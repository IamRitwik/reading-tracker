BINARY        := reading-tracker
CMD           := ./cmd/reading-tracker

VERSION       := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
VERSION_CLEAN := $(VERSION:v%=%)
COMMIT        := $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE          := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS       := -s -w \
                 -X main.version=$(VERSION) \
                 -X main.commit=$(COMMIT) \
                 -X main.date=$(DATE)

GOFLAGS       := -trimpath
PLATFORMS     := darwin/arm64 darwin/amd64 linux/amd64 linux/arm64

.PHONY: build install run test vet fmt clean release cross package checksums

build:
	go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BINARY) $(CMD)

install:
	go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(shell go env GOPATH)/bin/$(BINARY) $(CMD)

run:
	go run $(CMD)

test:
	go test ./... -race

vet:
	go vet ./...

fmt:
	gofmt -w .

clean:
	rm -f $(BINARY)
	rm -rf dist/

# --- Release -------------------------------------------------------------

release: clean cross package checksums
	@echo ""
	@echo "✓ Release artifacts for $(VERSION) are in dist/:"
	@ls -lh dist/

cross:
	@mkdir -p dist
	@for target in $(PLATFORMS); do \
		os=$${target%/*}; arch=$${target#*/}; \
		out="dist/$(BINARY)-$${os}-$${arch}"; \
		echo "  → $$out"; \
		GOOS=$$os GOARCH=$$arch CGO_ENABLED=0 \
		go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o "$$out" $(CMD); \
	done

# Package each binary into a tarball named exactly as install.sh expects:
#   reading-tracker_<version>_<os>_<arch>.tar.gz
package:
	@echo "Packaging archives..."
	@cd dist && for target in $(PLATFORMS); do \
		os=$${target%/*}; arch=$${target#*/}; \
		src="$(BINARY)-$${os}-$${arch}"; \
		dst="$(BINARY)_$(VERSION_CLEAN)_$${os}_$${arch}"; \
		cp "$$src" "$(BINARY)"; \
		tar -czf "$$dst.tar.gz" "$(BINARY)"; \
		rm -f "$(BINARY)" "$$src"; \
	done

checksums:
	@echo "Generating checksums..."
	@cd dist && shasum -a 256 *.tar.gz > "$(BINARY)_$(VERSION_CLEAN)_SHA256SUMS"
	@cat dist/$(BINARY)_$(VERSION_CLEAN)_SHA256SUMS