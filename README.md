# gui

Immediate-mode GUI library for Go with an OpenGL 4.1 backend. Built for [go-theft-auto](https://github.com/go-theft-auto), usable as a standalone library.

## Features

- Immediate-mode API — no persistent widget objects, draw everything every frame
- Built-in widget set: buttons, sliders, tables, lists, comboboxes, panels, graphs, toasts, and more
- Gamepad/keyboard focus navigation with double-buffered focus registry
- OpenGL 4.1 renderer with built-in monospace font
- GLFW input adapter
- Drag-and-drop, clipboard, and scroll support
- GTA-inspired default style

## Install

```bash
go get github.com/go-theft-auto/gui
```

Requires CGO and OpenGL/X11 development headers (see Development below).

## Usage

```go
package main

import (
    "github.com/go-gl/gl/v4.1-core/gl"
    "github.com/go-gl/glfw/v3.3/glfw"

    "github.com/go-theft-auto/gui"
    "github.com/go-theft-auto/gui/backend/opengl"
)

func main() {
    // ... initialize GLFW window and OpenGL context ...

    renderer, _ := opengl.NewRenderer(800, 600)
    defer renderer.Delete()

    inputAdapter := opengl.NewGLFWInputAdapter(window)

    ui := gui.New(renderer, gui.WithStyle(gui.GTAStyle()))

    for !window.ShouldClose() {
        glfw.PollEvents()
        inputAdapter.Update()

        gl.ClearColor(0.1, 0.1, 0.1, 1)
        gl.Clear(gl.COLOR_BUFFER_BIT)

        ctx := ui.Begin(inputAdapter.Input(), gui.Vec2{X: 800, Y: 600}, 1.0/60.0)

        ctx.Panel("My Panel", gui.Width(300))(func() {
            ctx.Text("Hello!")
            if ctx.Button("Click me") {
                // handle click
            }
        })

        ui.End()
        window.SwapBuffers()
    }
}
```

See [example/main.go](example/main.go) for a complete runnable example.

## Development

This package uses CGO for OpenGL. Install [Devbox](https://www.jetify.com/devbox) to get a reproducible environment with Go, OpenGL headers, X11 libraries, formatters, and linters:

```bash
devbox shell       # enter the dev environment
task fmt           # format code (gci + gofumpt)
task lint          # run golangci-lint
task test          # run tests
task build         # verify compilation
task deps          # tidy and vendor dependencies
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for more details.

## Architecture

```
gui/                    Core library (no OpenGL dependency)
  backend/opengl/       OpenGL 4.1 renderer + GLFW input adapter
  example/              Runnable example
```

The core `gui` package defines interfaces (`Renderer`, `FontProvider`) and all widgets. The `backend/opengl` package provides concrete implementations. This split means you could implement a different rendering backend (Vulkan, software, etc.) without touching the core.

## License

[MIT](LICENSE)
