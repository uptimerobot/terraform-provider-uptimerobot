# Minimal dev build
BIN        := terraform-provider-uptimerobot
OUT_DIR    := ./bin
DEV_VER    := 0.0.0-dev
LDFLAGS    := -X 'main.version=$(DEV_VER)'

.PHONY: build-dev clean
build-dev:
	@mkdir -p $(OUT_DIR)
	go build -trimpath -ldflags "$(LDFLAGS)" \
		-o $(OUT_DIR)/$(BIN)_v$(DEV_VER) .


default: testacc

# Run acceptance tests
.PHONY: testacc
testacc:
	set -a; [ -f .env ] && source .env; set +a && TF_ACC=1 go test ./internal/provider -tags=acceptance -run TestAcc -v $(TESTARGS) -parallel=1 -timeout 45m

# Run unit tests
.PHONY: test
test:
	go test ./... -short

# Run go generate
.PHONY: generate
generate:
	go generate ./...

# Format code
.PHONY: fmt
fmt:
	gofmt -w .

# Run linting
.PHONY: lint
lint:
	golangci-lint run

# Run all checks (format, lint, generate, test)
.PHONY: check
check: fmt generate lint test

# Tidy dependencies
.PHONY: tidy
tidy:
	go mod tidy

# Install development dependencies
.PHONY: deps
deps:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run all pre-commit checks
.PHONY: pre-commit
pre-commit: tidy fmt generate lint test
	@echo "All pre-commit checks passed!"

# Clean build artifacts
.PHONY: clean
clean:
	go clean -testcache
	rm -rf $(OUT_DIR)
