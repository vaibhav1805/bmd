# Contributing to BMD

Thank you for your interest in contributing to BMD! This document provides guidelines and instructions for contributing.

## Code of Conduct

Please be respectful and constructive. We welcome all contributions that improve BMD.

## Getting Started

### Prerequisites
- Go 1.18 or later
- Git
- Familiarity with terminal applications

### Setup Development Environment

```bash
# Clone repository
git clone https://github.com/flurryhead/bmd
cd bmd

# Install dependencies
go mod download

# Build
go build -o bmd ./cmd/bmd

# Verify build
./bmd README.md
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package
go test ./internal/knowledge/...

# Run with verbose output
go test -v ./...
```

### Code Quality

```bash
# Type check
go vet ./...

# Format code
go fmt ./...

# Check for common issues
go vet ./... 2>&1 | grep -v "composite literal uses unkeyed fields"
```

## Contribution Types

### Bug Reports

1. Check existing issues to avoid duplicates
2. Create a new issue with:
   - Terminal emulator and version
   - Markdown file example (if applicable)
   - Steps to reproduce
   - Expected vs. actual behavior
   - Screenshots (if visual issue)

### Feature Requests

1. Describe the feature and why it's useful
2. Provide examples of how you'd use it
3. Suggest implementation approach (if you have ideas)

### Code Contributions

1. **Fork** the repository
2. **Create** a feature branch: `git checkout -b feature/my-feature`
3. **Write** code following the guidelines below
4. **Test** your changes: `go test ./...`
5. **Commit** with descriptive messages
6. **Push** to your fork
7. **Create** a pull request

## Coding Guidelines

### Style

- Follow Go conventions (gofmt, golint)
- Use clear, descriptive variable names
- Write comments for exported functions
- Keep functions focused and modular
- Maximum line length: 100 characters (soft limit, break for readability)

### Testing

- Write unit tests for new functionality
- Aim for >80% code coverage
- Use table-driven tests for multiple cases
- Test edge cases and error conditions

### Commit Messages

```
feat: Add feature description

Body (optional): Detailed explanation of changes, why they're needed, etc.

Closes #123
```

Types: `feat`, `fix`, `docs`, `test`, `refactor`, `perf`, `chore`

## Project Structure

Understanding the project layout helps with contributions:

```
internal/
├── knowledge/       Knowledge system (BM25, graph, services)
├── renderer/        ANSI rendering, images
├── theme/           Color themes
├── tui/             Terminal UI framework
├── nav/             Navigation
├── search/          Search algorithms
├── parser/          Markdown parsing
└── terminal/        Terminal utilities

cmd/bmd/
└── main.go          Entry point, CLI routing
```

## Adding a Feature

### Example: Add a new color theme

1. **Design:** Sketch the theme colors
2. **Code:** Edit `internal/theme/colors.go`
   ```go
   case "neon":
     return NewPalette(
       fg: 0xFF00FF,    // Neon magenta
       bg: 0x000000,    // Black
       ...
     )
   ```
3. **Test:** Update `internal/theme/colors_test.go`
4. **Document:** Update `QUICKSTART.md` with new theme
5. **Commit:** `feat: Add neon color theme`

### Example: Fix a bug

1. **Reproduce:** Create a test that fails
   ```go
   func TestLinkNavigationBug(t *testing.T) {
     // Test that fails with current code
     result := nav.Follow("../file.md")
     if result == "" {
       t.Fatal("link navigation failed")
     }
   }
   ```
2. **Fix:** Implement the fix
3. **Verify:** Test passes
4. **Commit:** `fix: Handle relative paths correctly in link navigation`

## Documentation

When contributing, ensure documentation is updated:

- **Code comments:** For complex logic or exported functions
- **README.md:** For new user-facing features
- **ARCHITECTURE.md:** For architectural changes
- **COMMANDS.md:** For new CLI commands
- **Commit messages:** For what changed and why

## Performance Considerations

BMD prioritizes responsiveness. When contributing:

- Profile before optimizing
- Avoid blocking operations
- Cache when appropriate
- Consider memory usage
- Test with large files (1MB+)

Benchmarks:
```bash
go test -bench=. -benchmem ./internal/knowledge/...
```

## Review Process

1. Automated checks run (tests, formatting)
2. Project maintainer reviews code
3. Feedback provided (if needed)
4. Changes addressed
5. Approval and merge

Reviews typically focus on:
- Correctness and robustness
- Code style consistency
- Test coverage
- Documentation
- Performance impact

## Release Process

Releases follow semantic versioning:
- **Major (X.0.0):** Breaking changes
- **Minor (0.X.0):** New features
- **Patch (0.0.X):** Bug fixes

### Creating a Release

```bash
# Tag the release
git tag -a v0.6.0 -m "Add knowledge graphs and agent intelligence"

# Push tag
git push origin v0.6.0

# Create GitHub release with changelog
# Release notes should include:
# - New features
# - Bug fixes
# - Breaking changes
# - Contributors
```

## Common Issues

### Build fails with "module not found"

```bash
go mod tidy
go mod download
```

### Tests fail with "database locked"

Ensure no other processes are accessing the test database:
```bash
# On macOS/Linux
pkill -f "bmd.*test"
```

### Performance issues

Profile with pprof:
```bash
go test -cpuprofile=cpu.prof ./...
go tool pprof cpu.prof
```

## Getting Help

- **Questions:** Create a discussion or issue
- **Ideas:** Suggest in GitHub discussions
- **Bugs:** Report with detailed reproduction steps
- **Contact:** Open an issue to get in touch

## Thank You!

Contributors who've helped make BMD better:
- Feature implementations
- Bug fixes
- Documentation improvements
- Testing and validation
- User feedback

Every contribution, no matter how small, helps improve BMD for everyone.

---

See [README.md](README.md) for project overview and [ARCHITECTURE.md](ARCHITECTURE.md) for technical details.
