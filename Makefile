GOCMD := go
GOTOOLS_DIR := tools/bin
GOLANGCILINT_CMD := $(GOTOOLS_DIR)/golangci-lint-v2
GOLANGCILINT_VERSION := v2.5.0
GOLANGCILINT_CFG := .golangci.yml

.PHONY: test
test:
	$(GOCMD) test -race -v ./...

.PHONY: test/no-cache
test/no-cache:
	$(GOCMD) test -race -v -count=1 ./...

.PHONY: test/cover
test/cover:
	$(GOCMD) test -coverprofile cover.out ./...
	$(GOCMD) tool cover -html cover.out -o cover.html

.PHONY: audit
audit: audit/tidy audit/verify-deps audit/lint

.PHONY: audit/tidy
audit/tidy:
	$(GOCMD) mod tidy -diff

.PHONY: audit/verify-deps
audit/verify-deps:
	$(GOCMD) mod verify

.PHONY: audit/lint
audit/lint:
	$(GOLANGCILINT_CMD) run --config $(CURDIR)/$(GOLANGCILINT_CFG)

.PHONY: install-golangci-lint
install-golangci-lint:
	curl -sSfL "https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh" | sh -s -- -b $(GOTOOLS_DIR) $(GOLANGCILINT_VERSION)
	ln $(GOTOOLS_DIR)/golangci-lint{,-v2}

.PHONY: run
run:
	$(GOCMD) run ./cmd/gophermart

.PHONY: build
build:
	CGO_ENABLED=0 $(GOCMD) build -o bin/gophermart ./cmd/gophermart && \
	ln -sf ../../bin/gophermart cmd/gophermart/gophermart
