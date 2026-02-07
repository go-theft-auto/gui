package gui

// MouseButton represents a mouse button.
type MouseButton int

const (
	MouseButtonLeft MouseButton = iota
	MouseButtonRight
	MouseButtonMiddle
	MouseButtonCount
)

// Key represents a keyboard key.
type Key int

const (
	KeyNone Key = iota
	KeyTab
	KeyLeft
	KeyRight
	KeyUp
	KeyDown
	KeyPageUp
	KeyPageDown
	KeyHome
	KeyEnd
	KeyInsert
	KeyDelete
	KeyBackspace
	KeySpace
	KeyEnter
	KeyEscape
	KeyA
	KeyC
	KeyS
	KeyT
	KeyV
	KeyX
	KeyY
	KeyZ
	KeyF1
	KeyF2
	KeyF3
	KeyF4
	KeyF5
	KeyF6
	KeyF7
	KeyF8
	KeyF9
	KeyF10
	KeyF11
	KeyF12
	KeyCount
)

// Key repeat timing constants
const (
	KeyRepeatDelay    float32 = 0.4  // Initial delay before repeat starts (seconds)
	KeyRepeatInterval float32 = 0.03 // Repeat interval once repeating (seconds)
)

// InputState holds input state for the current frame.
// This is typically populated by the application from GLFW or similar.
type InputState struct {
	// Mouse position
	MouseX, MouseY float32

	// Mouse buttons - current frame state
	mouseDown    [MouseButtonCount]bool
	mouseClicked [MouseButtonCount]bool // True on the frame button was pressed
	mouseUp      [MouseButtonCount]bool // True on the frame button was released

	// Mouse wheel
	MouseWheelX float32
	MouseWheelY float32

	// Keyboard - current frame state
	keyDown    [KeyCount]bool
	keyPressed [KeyCount]bool // True on the frame key was pressed
	keyUp      [KeyCount]bool // True on the frame key was released

	// Key repeat tracking
	keyHoldTime [KeyCount]float32 // How long each key has been held

	// Text input (Unicode characters typed this frame)
	InputChars []rune

	// Modifiers
	ModCtrl  bool
	ModShift bool
	ModAlt   bool
	ModSuper bool
}

// NewInputState creates a new InputState.
func NewInputState() *InputState {
	return &InputState{
		InputChars: make([]rune, 0, 16),
	}
}

// Reset clears per-frame input state.
// Call this at the start of each frame before collecting input.
func (s *InputState) Reset() {
	// Clear single-frame events
	for i := range s.mouseClicked {
		s.mouseClicked[i] = false
	}
	for i := range s.mouseUp {
		s.mouseUp[i] = false
	}
	for i := range s.keyPressed {
		s.keyPressed[i] = false
	}
	for i := range s.keyUp {
		s.keyUp[i] = false
	}
	s.InputChars = s.InputChars[:0]
	s.MouseWheelX = 0
	s.MouseWheelY = 0
}

// SetMousePos sets the mouse position.
func (s *InputState) SetMousePos(x, y float32) {
	s.MouseX = x
	s.MouseY = y
}

// SetMouseButton sets mouse button state.
func (s *InputState) SetMouseButton(button MouseButton, down bool) {
	if button < 0 || button >= MouseButtonCount {
		return
	}

	wasDown := s.mouseDown[button]
	s.mouseDown[button] = down

	if down && !wasDown {
		s.mouseClicked[button] = true
	}
	if !down && wasDown {
		s.mouseUp[button] = true
	}
}

// SetKey sets key state.
func (s *InputState) SetKey(key Key, down bool) {
	if key < 0 || key >= KeyCount {
		return
	}

	wasDown := s.keyDown[key]
	s.keyDown[key] = down

	if down && !wasDown {
		s.keyPressed[key] = true
		s.keyHoldTime[key] = 0 // Reset hold time on fresh press
	}
	if !down && wasDown {
		s.keyUp[key] = true
		s.keyHoldTime[key] = 0 // Reset hold time on release
	}
}

// UpdateKeyRepeat updates key hold times for repeat detection.
// Call this once per frame with the frame's delta time.
func (s *InputState) UpdateKeyRepeat(dt float32) {
	for key := Key(0); key < KeyCount; key++ {
		if s.keyDown[key] {
			s.keyHoldTime[key] += dt
		}
	}
}

// SetMouseWheel sets the mouse wheel delta.
func (s *InputState) SetMouseWheel(x, y float32) {
	s.MouseWheelX = x
	s.MouseWheelY = y
}

// AddInputChar adds a typed character.
func (s *InputState) AddInputChar(ch rune) {
	s.InputChars = append(s.InputChars, ch)
}

// MouseDown returns true if a mouse button is currently held.
func (s *InputState) MouseDown(button MouseButton) bool {
	if button < 0 || button >= MouseButtonCount {
		return false
	}
	return s.mouseDown[button]
}

// MouseClicked returns true if a mouse button was just clicked (pressed this frame).
func (s *InputState) MouseClicked(button MouseButton) bool {
	if button < 0 || button >= MouseButtonCount {
		return false
	}
	return s.mouseClicked[button]
}

// MouseReleased returns true if a mouse button was just released.
func (s *InputState) MouseReleased(button MouseButton) bool {
	if button < 0 || button >= MouseButtonCount {
		return false
	}
	return s.mouseUp[button]
}

// KeyDown returns true if a key is currently held.
func (s *InputState) KeyDown(key Key) bool {
	if key < 0 || key >= KeyCount {
		return false
	}
	return s.keyDown[key]
}

// KeyPressed returns true if a key was just pressed (pressed this frame).
func (s *InputState) KeyPressed(key Key) bool {
	if key < 0 || key >= KeyCount {
		return false
	}
	return s.keyPressed[key]
}

// KeyReleased returns true if a key was just released.
func (s *InputState) KeyReleased(key Key) bool {
	if key < 0 || key >= KeyCount {
		return false
	}
	return s.keyUp[key]
}

// KeyRepeated returns true if a key should trigger this frame.
// Returns true on initial press, then after KeyRepeatDelay, then every KeyRepeatInterval.
// Use this for actions that should repeat when holding a key (like backspace in text input).
func (s *InputState) KeyRepeated(key Key) bool {
	if key < 0 || key >= KeyCount {
		return false
	}

	// Trigger on initial press
	if s.keyPressed[key] {
		return true
	}

	// Check if held long enough to repeat
	if !s.keyDown[key] {
		return false
	}

	holdTime := s.keyHoldTime[key]
	if holdTime < KeyRepeatDelay {
		return false
	}

	// Calculate how many repeat intervals have passed since delay
	timeSinceDelay := holdTime - KeyRepeatDelay
	// Trigger if we just crossed an interval boundary this frame
	// This is approximate but works well for typical frame rates
	repeatCount := int(timeSinceDelay / KeyRepeatInterval)
	prevRepeatCount := int((timeSinceDelay - 0.016) / KeyRepeatInterval) // Assume ~60fps for prev frame
	return repeatCount > prevRepeatCount
}

// HasInputChars returns true if there are typed characters this frame.
func (s *InputState) HasInputChars() bool {
	return len(s.InputChars) > 0
}

// ConsumeInputChars clears all typed characters for this frame.
// Call this after processing a keyboard shortcut to prevent the shortcut key
// from also being typed into text fields (e.g., 'V' opens menu but shouldn't type 'v').
func (s *InputState) ConsumeInputChars() {
	s.InputChars = s.InputChars[:0]
}

// KeyName returns a human-readable name for a key.
func KeyName(k Key) string {
	names := map[Key]string{
		KeyNone:      "--",
		KeyTab:       "Tab",
		KeyLeft:      "Left",
		KeyRight:     "Right",
		KeyUp:        "Up",
		KeyDown:      "Down",
		KeyPageUp:    "PgUp",
		KeyPageDown:  "PgDn",
		KeyHome:      "Home",
		KeyEnd:       "End",
		KeyInsert:    "Ins",
		KeyDelete:    "Del",
		KeyBackspace: "Backspace",
		KeySpace:     "Space",
		KeyEnter:     "Enter",
		KeyEscape:    "Esc",
		KeyA:         "A",
		KeyC:         "C",
		KeyS:         "S",
		KeyT:         "T",
		KeyV:         "V",
		KeyX:         "X",
		KeyY:         "Y",
		KeyZ:         "Z",
		KeyF1:        "F1",
		KeyF2:        "F2",
		KeyF3:        "F3",
		KeyF4:        "F4",
		KeyF5:        "F5",
		KeyF6:        "F6",
		KeyF7:        "F7",
		KeyF8:        "F8",
		KeyF9:        "F9",
		KeyF10:       "F10",
		KeyF11:       "F11",
		KeyF12:       "F12",
	}
	if name, ok := names[k]; ok {
		return name
	}
	return "?"
}
