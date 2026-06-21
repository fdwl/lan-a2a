# Contributing to LanA2A

Thank you for your interest in contributing to LanA2A!

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/<your-username>/lan-a2a.git`
3. Create a branch: `git checkout -b feature/my-feature`
4. Make your changes
5. Run tests: `make test`
6. Run linter: `make lint`
7. Commit your changes
8. Push to your fork and submit a Pull Request

## Development Setup

### Prerequisites

- Go 1.25+
- golangci-lint (for linting)

### Build

```bash
make build
```

### Run Tests

```bash
make test
```

### Lint

```bash
make lint
```

## Code Style

- Follow standard Go conventions (`gofmt`, `go vet`)
- Use meaningful variable and function names
- Keep functions focused and small
- Add tests for new functionality
- Update documentation for user-facing changes

## Commit Messages

- Use present tense ("Add feature" not "Added feature")
- Keep the first line under 72 characters
- Reference issues when applicable

## Pull Request Guidelines

- PRs should target the `main` branch
- Include a clear description of what changed and why
- Ensure all CI checks pass
- Add tests for new features
- Update documentation as needed

## Reporting Issues

- Use GitHub Issues for bug reports and feature requests
- Include reproduction steps for bugs
- Specify your Go version and OS

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
