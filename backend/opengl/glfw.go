package opengl

import (
	"github.com/go-gl/glfw/v3.3/glfw"

	"github.com/go-theft-auto/gui"
)

// GLFWInputAdapter adapts GLFW input to gui.InputState.
type GLFWInputAdapter struct {
	window *glfw.Window
	input  *gui.InputState
}

// NewGLFWInputAdapter creates a new GLFW input adapter.
func NewGLFWInputAdapter(window *glfw.Window) *GLFWInputAdapter {
	adapter := &GLFWInputAdapter{
		window: window,
		input:  gui.NewInputState(),
	}

	// Setup callbacks
	window.SetKeyCallback(adapter.keyCallback)
	window.SetCharCallback(adapter.charCallback)
	window.SetMouseButtonCallback(adapter.mouseButtonCallback)
	window.SetScrollCallback(adapter.scrollCallback)
	window.SetCursorPosCallback(adapter.cursorPosCallback)

	return adapter
}

// Update updates the input state for a new frame.
// Call this at the start of each frame.
func (a *GLFWInputAdapter) Update() *gui.InputState {
	a.input.Reset()

	// Update mouse position
	x, y := a.window.GetCursorPos()
	a.input.SetMousePos(float32(x), float32(y))

	// Update modifiers
	a.input.ModCtrl = a.window.GetKey(glfw.KeyLeftControl) == glfw.Press ||
		a.window.GetKey(glfw.KeyRightControl) == glfw.Press
	a.input.ModShift = a.window.GetKey(glfw.KeyLeftShift) == glfw.Press ||
		a.window.GetKey(glfw.KeyRightShift) == glfw.Press
	a.input.ModAlt = a.window.GetKey(glfw.KeyLeftAlt) == glfw.Press ||
		a.window.GetKey(glfw.KeyRightAlt) == glfw.Press
	a.input.ModSuper = a.window.GetKey(glfw.KeyLeftSuper) == glfw.Press ||
		a.window.GetKey(glfw.KeyRightSuper) == glfw.Press

	return a.input
}

// Input returns the current input state.
func (a *GLFWInputAdapter) Input() *gui.InputState {
	return a.input
}

func (a *GLFWInputAdapter) keyCallback(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
	guiKey := glfwKeyToGUIKey(key)
	if guiKey == gui.KeyNone {
		return
	}

	switch action {
	case glfw.Press, glfw.Repeat:
		a.input.SetKey(guiKey, true)
	case glfw.Release:
		a.input.SetKey(guiKey, false)
	}
}

func (a *GLFWInputAdapter) charCallback(w *glfw.Window, char rune) {
	a.input.AddInputChar(char)
}

func (a *GLFWInputAdapter) mouseButtonCallback(w *glfw.Window, button glfw.MouseButton, action glfw.Action, mods glfw.ModifierKey) {
	guiButton := glfwMouseButtonToGUI(button)
	if guiButton < 0 {
		return
	}

	switch action {
	case glfw.Press:
		a.input.SetMouseButton(guiButton, true)
	case glfw.Release:
		a.input.SetMouseButton(guiButton, false)
	}
}

func (a *GLFWInputAdapter) scrollCallback(w *glfw.Window, xoff, yoff float64) {
	a.input.SetMouseWheel(float32(xoff), float32(yoff))
}

func (a *GLFWInputAdapter) cursorPosCallback(w *glfw.Window, xpos, ypos float64) {
	a.input.SetMousePos(float32(xpos), float32(ypos))
}

// glfwKeyToGUIKey maps GLFW keys to GUI keys.
func glfwKeyToGUIKey(key glfw.Key) gui.Key {
	switch key {
	case glfw.KeyTab:
		return gui.KeyTab
	case glfw.KeyLeft:
		return gui.KeyLeft
	case glfw.KeyRight:
		return gui.KeyRight
	case glfw.KeyUp:
		return gui.KeyUp
	case glfw.KeyDown:
		return gui.KeyDown
	case glfw.KeyPageUp:
		return gui.KeyPageUp
	case glfw.KeyPageDown:
		return gui.KeyPageDown
	case glfw.KeyHome:
		return gui.KeyHome
	case glfw.KeyEnd:
		return gui.KeyEnd
	case glfw.KeyInsert:
		return gui.KeyInsert
	case glfw.KeyDelete:
		return gui.KeyDelete
	case glfw.KeyBackspace:
		return gui.KeyBackspace
	case glfw.KeySpace:
		return gui.KeySpace
	case glfw.KeyEnter:
		return gui.KeyEnter
	case glfw.KeyEscape:
		return gui.KeyEscape
	case glfw.KeyA:
		return gui.KeyA
	case glfw.KeyC:
		return gui.KeyC
	case glfw.KeyS:
		return gui.KeyS
	case glfw.KeyV:
		return gui.KeyV
	case glfw.KeyX:
		return gui.KeyX
	case glfw.KeyY:
		return gui.KeyY
	case glfw.KeyZ:
		return gui.KeyZ
	case glfw.KeyF1:
		return gui.KeyF1
	case glfw.KeyF2:
		return gui.KeyF2
	case glfw.KeyF3:
		return gui.KeyF3
	case glfw.KeyF4:
		return gui.KeyF4
	case glfw.KeyF5:
		return gui.KeyF5
	case glfw.KeyF6:
		return gui.KeyF6
	case glfw.KeyF7:
		return gui.KeyF7
	case glfw.KeyF8:
		return gui.KeyF8
	case glfw.KeyF9:
		return gui.KeyF9
	case glfw.KeyF10:
		return gui.KeyF10
	case glfw.KeyF11:
		return gui.KeyF11
	case glfw.KeyF12:
		return gui.KeyF12
	default:
		return gui.KeyNone
	}
}

// glfwMouseButtonToGUI maps GLFW mouse buttons to GUI mouse buttons.
func glfwMouseButtonToGUI(button glfw.MouseButton) gui.MouseButton {
	switch button {
	case glfw.MouseButtonLeft:
		return gui.MouseButtonLeft
	case glfw.MouseButtonRight:
		return gui.MouseButtonRight
	case glfw.MouseButtonMiddle:
		return gui.MouseButtonMiddle
	default:
		return -1
	}
}
