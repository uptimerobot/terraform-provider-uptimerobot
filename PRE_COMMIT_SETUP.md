# Pre-commit Hooks Setup

This project has pre-commit hooks configured to automatically run linting and code generation before each commit. There are two ways to use pre-commit hooks:

## Option 1: Simple Git Hook (Already Active)

The simple git hook is already installed and active at `.git/hooks/pre-commit`. It will automatically run before each commit and includes:

- `go generate ./...` - Runs code generation
- `go mod tidy` - Tidies Go modules
- `gofmt` - Formats Go code
- `golangci-lint run` - Runs comprehensive linting
- `go test ./... -short` - Runs unit tests

### Manual Commands

You can also run the same checks manually using the Makefile:

```bash
# Run all pre-commit checks
make pre-commit

# Run individual checks
make fmt      # Format code
make generate # Run go generate
make lint     # Run linting
make test     # Run tests
make tidy     # Tidy dependencies
```

## Option 2: Pre-commit Framework (Advanced)

For more advanced features and better integration with development workflows, you can use the pre-commit framework:

### Installation

1. Install the pre-commit framework:
   ```bash
   pip install pre-commit
   ```

2. Install the hooks:
   ```bash
   pre-commit install
   ```

3. (Optional) Run on all files:
   ```bash
   pre-commit run --all-files
   ```

### Features

The pre-commit framework configuration includes:

- **Standard hooks**: trailing whitespace, end-of-file-fixer, etc.
- **Go-specific hooks**:
  - Code formatting with `gofmt`
  - Static analysis with `go vet`
  - Module tidying with `go mod tidy`
  - Code generation with `go generate`
  - Unit tests with `go test -short`
  - Comprehensive linting with `golangci-lint`

### Configuration

The configuration is stored in `.pre-commit-config.yaml` and uses your existing `.golangci.yml` configuration.

## Development Dependencies

Make sure you have the required tools installed:

```bash
# Install golangci-lint
make deps

# Or install manually
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

## Bypassing Hooks

In emergency situations, you can skip the pre-commit hooks:

```bash
git commit --no-verify -m "Emergency commit"
```

**Note**: This should be used sparingly and the code should be fixed in a follow-up commit.

## Troubleshooting

### Hook fails due to formatting issues
```bash
# Fix formatting
make fmt
git add -A
git commit -m "Your commit message"
```

### Hook fails due to linting issues
```bash
# Run linting to see issues
make lint

# Fix issues and try again
git add -A
git commit -m "Your commit message"
```

### Hook fails due to go generate changes
```bash
# Run go generate
make generate

# Add generated files
git add -A
git commit -m "Your commit message"
```

## CI/CD Integration

The same checks can be run in CI/CD pipelines:

```yaml
# Example GitHub Actions step
- name: Run pre-commit checks
  run: make pre-commit
```
