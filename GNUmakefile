default: testacc

# Run acceptance tests
.PHONY: testacc
testacc:
	set -a; source .env; set +a && TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m

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
	rm -rf ./bin/
