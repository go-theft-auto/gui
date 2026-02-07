# Contributing

Contributions are welcome. This document covers the basics.

## Prerequisites

This package uses CGO for OpenGL rendering. Install [Devbox](https://www.jetify.com/devbox) to get a reproducible environment with Go, OpenGL headers, and X11 libraries:

```bash
devbox shell
```

All commands below assume you are inside a devbox shell or prefixed with `devbox run --`.

## Development workflow

```bash
task fmt      # Format code (gci + gofumpt)
task lint     # Run golangci-lint
task test     # Run tests
task build    # Verify compilation
task deps     # Tidy and vendor dependencies
```

## Code style

- Imports are organized by [gci](https://github.com/daixiang0/gci): standard library, third-party, `github.com/go-theft-auto`.
- Code is formatted with [gofumpt](https://github.com/mvdan/gofumpt) (stricter gofmt).
- Run `task fmt` before committing.

## Architecture

The GUI follows an **immediate-mode** design:

- Widgets are drawn fresh every frame (no persistent widget objects).
- State is stored in a `StateStore` keyed by string IDs.
- The `backend/opengl/` package provides the concrete OpenGL 4.1 renderer and GLFW input adapter. The core `gui` package has no OpenGL dependency.

## Submitting changes

1. Fork the repository and create a branch.
2. Make your changes.
3. Run `task fmt && task lint && task test`.
4. Open a pull request with a clear description of the change.
