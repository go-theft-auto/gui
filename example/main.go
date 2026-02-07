// Example demonstrates a minimal GUI window with a panel and a few widgets.
//
// Prerequisites:
//
//	Install devbox: https://www.jetify.com/devbox
//	devbox shell              # enter the dev environment (provides Go + OpenGL/X11 headers)
//	go run ./example/         # run this example
//
// The example creates a GLFW window, initializes the OpenGL GUI renderer,
// and renders a simple panel with a button, text, and slider.
package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"

	"github.com/go-theft-auto/gui"
	"github.com/go-theft-auto/gui/backend/opengl"
)

const (
	windowWidth  = 800
	windowHeight = 600
	windowTitle  = "gui example"
)

func init() {
	// GLFW must run on the main thread.
	runtime.LockOSThread()
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	// Initialize GLFW.
	if err := glfw.Init(); err != nil {
		return fmt.Errorf("glfw init: %w", err)
	}
	defer glfw.Terminate()

	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)

	window, err := glfw.CreateWindow(windowWidth, windowHeight, windowTitle, nil, nil)
	if err != nil {
		return fmt.Errorf("create window: %w", err)
	}
	window.MakeContextCurrent()
	glfw.SwapInterval(1) // vsync

	// Initialize OpenGL.
	if err := gl.Init(); err != nil {
		return fmt.Errorf("gl init: %w", err)
	}

	// Create the GUI renderer (takes initial viewport size) and input adapter.
	renderer, err := opengl.NewRenderer(windowWidth, windowHeight)
	if err != nil {
		return fmt.Errorf("gui renderer: %w", err)
	}
	defer renderer.Delete()

	inputAdapter := opengl.NewGLFWInputAdapter(window)

	// Create the GUI instance with the GTA style.
	ui := gui.New(renderer, gui.WithStyle(gui.GTAStyle()))

	// Application state.
	clickCount := 0
	sliderVal := float32(0.5)

	// Main loop.
	for !window.ShouldClose() {
		glfw.PollEvents()
		inputAdapter.Update()

		w, h := window.GetFramebufferSize()
		gl.Viewport(0, 0, int32(w), int32(h))
		gl.ClearColor(0.12, 0.12, 0.14, 1.0)
		gl.Clear(gl.COLOR_BUFFER_BIT)

		// Start a GUI frame.
		displaySize := gui.Vec2{X: float32(w), Y: float32(h)}
		ctx := ui.Begin(inputAdapter.Input(), displaySize, 1.0/60.0)

		// Draw a panel using the closure-based API.
		ctx.Panel("Example Panel", gui.Width(300))(func() {
			ctx.Text("Hello from gui!")
			ctx.Spacing(8)

			if ctx.Button(fmt.Sprintf("Click me (%d)", clickCount)) {
				clickCount++
			}

			ctx.Spacing(8)
			ctx.Text(fmt.Sprintf("Slider: %.2f", sliderVal))
			ctx.SliderFloat("example-slider", &sliderVal, 0, 1)
		})

		// End the GUI frame and render.
		if err := ui.End(); err != nil {
			return fmt.Errorf("gui render: %w", err)
		}

		window.SwapBuffers()
	}

	return nil
}
