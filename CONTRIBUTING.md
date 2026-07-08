# Contributing to TaskFlow

Thank you for your interest in contributing to TaskFlow! This document provides guidelines and instructions for contributing.

## Code of Conduct

Be respectful, inclusive, and constructive in all interactions.

## Getting Started

1. **Fork the repository** on GitHub
2. **Clone your fork**:
   ```bash
   git clone https://github.com/yourusername/taskflow.git
   cd taskflow
   ```
3. **Create a feature branch**:
   ```bash
   git checkout -b feature/your-feature-name
   ```

## Development Setup

```bash
# Install Go 1.21+
# Start PostgreSQL
docker-compose up -d

# Install dependencies
go mod download

# Run tests
go test ./...

# Build
go build -o bin/server ./cmd/server
```

## Making Changes

### Code Style

- Follow [Go Code Review Comments](https://golang.org/doc/effective_go)
- Use `gofmt` for formatting
- Keep functions focused and testable
- Add comments for exported functions and packages

### Testing

- Write tests for new features
- Ensure all tests pass: `go test ./...`
- Aim for >80% code coverage on new code
- Property-based tests in `test/property/` are optional but encouraged

### Commit Messages

Use clear, descriptive commit messages:
```
Short summary (50 chars max)

More detailed explanation if needed, wrapped at 72 chars.
Explain *what* and *why*, not just what changed.

Fixes #123
```

## Submitting Changes

1. **Push to your fork**:
   ```bash
   git push origin feature/your-feature-name
   ```

2. **Open a Pull Request** on GitHub with:
   - Clear title describing the change
   - Description of what the PR does
   - Reference to any related issues (e.g., `Fixes #123`)
   - Evidence that tests pass

3. **Address review feedback** if requested

## Areas for Contribution

### Easy (Good for First-Time Contributors)
- Documentation improvements
- Bug fixes with clear reproduction steps
- Test coverage improvements
- Code comments and examples

### Medium
- New configuration options
- Additional built-in task handlers
- Enhanced error messages
- Performance optimizations

### Complex
- Distributed scheduler implementation
- Alternative persistence layers
- gRPC API implementation
- Prometheus metrics

## Reporting Issues

When reporting bugs, please include:
- Go version (`go version`)
- PostgreSQL version
- Steps to reproduce
- Expected vs. actual behavior
- Error messages and logs

## Pull Request Process

1. **Ensure tests pass**: `go test -race ./...`
2. **Update documentation** if needed
3. **Keep commits focused** — one feature per PR
4. **Request review** from maintainers
5. **Squash commits** before merging (if requested)

## Questions?

Open an issue with the `question` label or reach out to maintainers.

Thank you for contributing! 🎉
