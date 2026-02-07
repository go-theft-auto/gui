package gui

// StateStore persists widget state between frames.
// Unlike ImGui's hidden state, this is explicit and inspectable.
type StateStore interface {
	Get(id ID) (any, bool)
	Set(id ID, value any)
	Delete(id ID)
}

// MapStateStore is a simple in-memory StateStore implementation.
type MapStateStore map[ID]any

// Get retrieves a value from the store.
func (m MapStateStore) Get(id ID) (any, bool) {
	v, ok := m[id]
	return v, ok
}

// Set stores a value in the store.
func (m MapStateStore) Set(id ID, value any) {
	m[id] = value
}

// Delete removes a value from the store.
func (m MapStateStore) Delete(id ID) {
	delete(m, id)
}

// GetState retrieves typed state from the context.
// Returns defaultVal if the state doesn't exist or has wrong type.
func GetState[T any](ctx *Context, id ID, defaultVal T) T {
	if v, ok := ctx.stateStore.Get(id); ok {
		if typed, ok := v.(T); ok {
			return typed
		}
	}
	return defaultVal
}

// SetState stores typed state in the context.
func SetState[T any](ctx *Context, id ID, value T) {
	ctx.stateStore.Set(id, value)
}

// DeleteState removes state from the context.
func DeleteState(ctx *Context, id ID) {
	ctx.stateStore.Delete(id)
}

// Common state types for widgets

// ScrollState tracks scroll position for scrollable areas.
type ScrollState struct {
	ScrollY       float32 // Current scroll position
	TargetScrollY float32 // Target for smooth scrolling
	ContentHeight float32 // Total content height
}

// UpdateSmooth smoothly interpolates scroll position toward target.
// Call this each frame with the frame's delta time.
// Returns true if still animating.
func (s *ScrollState) UpdateSmooth(deltaTime float32) bool {
	const smoothSpeed = 15.0 // Higher = faster convergence
	const threshold = 0.5    // Stop animating when this close

	diff := s.TargetScrollY - s.ScrollY
	if absf32(diff) < threshold {
		s.ScrollY = s.TargetScrollY
		return false
	}

	s.ScrollY += diff * deltaTime * smoothSpeed
	return true
}

// absf32 returns the absolute value of a float32.
func absf32(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}

// InputTextState tracks state for text input widgets.
// Supports cursor positioning, text selection, and undo/redo.
type InputTextState struct {
	// Editing indicates whether the widget is in active text editing mode.
	// When true, the widget captures keyboard input for text entry.
	// This is separate from registry focus - a widget can be registry-focused
	// (highlighted for navigation) without being in edit mode.
	Editing bool

	// Cursor position (in runes, not bytes)
	CursorPos int

	// Selection range (in runes). SelectionStart is the anchor point,
	// SelectionEnd follows the cursor. -1 means no selection.
	SelectionStart int
	SelectionEnd   int

	// Horizontal scroll offset for long text that exceeds input width
	ScrollOffset float32

	// Undo/redo stack
	UndoStack []string // Previous text states
	UndoIndex int      // Current position in undo stack

	// Cursor blink state (managed internally)
	CursorBlinkTime float32
}

// HasSelection returns true if there's an active text selection.
func (s *InputTextState) HasSelection() bool {
	return s.SelectionStart >= 0 && s.SelectionStart != s.SelectionEnd
}

// GetSelectedRange returns the selection range as (start, end) where start <= end.
// Returns (-1, -1) if no selection.
func (s *InputTextState) GetSelectedRange() (start, end int) {
	if !s.HasSelection() {
		return -1, -1
	}
	if s.SelectionStart < s.SelectionEnd {
		return s.SelectionStart, s.SelectionEnd
	}
	return s.SelectionEnd, s.SelectionStart
}

// ClearSelection removes the selection.
func (s *InputTextState) ClearSelection() {
	s.SelectionStart = -1
	s.SelectionEnd = -1
}

// SelectAll selects all text.
func (s *InputTextState) SelectAll(textLen int) {
	s.SelectionStart = 0
	s.SelectionEnd = textLen
	s.CursorPos = textLen
}

// PushUndo saves the current text to the undo stack.
// Call this before making changes to the text.
func (s *InputTextState) PushUndo(text string) {
	const maxUndoSize = 50

	// If we're not at the end of the stack, truncate forward history
	if s.UndoIndex < len(s.UndoStack) {
		s.UndoStack = s.UndoStack[:s.UndoIndex]
	}

	// Avoid duplicate entries
	if len(s.UndoStack) > 0 && s.UndoStack[len(s.UndoStack)-1] == text {
		return
	}

	s.UndoStack = append(s.UndoStack, text)
	s.UndoIndex = len(s.UndoStack)

	// Limit stack size
	if len(s.UndoStack) > maxUndoSize {
		s.UndoStack = s.UndoStack[1:]
		s.UndoIndex--
	}
}

// Undo returns the previous text state, or empty string if nothing to undo.
// Also updates the undo index.
func (s *InputTextState) Undo(currentText string) (string, bool) {
	// Save current state if at end of stack
	if s.UndoIndex == len(s.UndoStack) && len(s.UndoStack) > 0 {
		// Don't duplicate if same as last entry
		if s.UndoStack[len(s.UndoStack)-1] != currentText {
			s.UndoStack = append(s.UndoStack, currentText)
		}
	}

	if s.UndoIndex > 0 {
		s.UndoIndex--
		return s.UndoStack[s.UndoIndex], true
	}
	return "", false
}

// Redo returns the next text state, or empty string if nothing to redo.
func (s *InputTextState) Redo() (string, bool) {
	if s.UndoIndex < len(s.UndoStack)-1 {
		s.UndoIndex++
		return s.UndoStack[s.UndoIndex], true
	}
	return "", false
}

// CanUndo returns true if undo is available.
func (s *InputTextState) CanUndo() bool {
	return s.UndoIndex > 0
}

// CanRedo returns true if redo is available.
func (s *InputTextState) CanRedo() bool {
	return s.UndoIndex < len(s.UndoStack)-1
}

// TreeNodeState tracks expanded/collapsed state for tree nodes.
type TreeNodeState struct {
	Open bool
}

// CollapsingHeaderState tracks collapsed state for collapsing headers.
type CollapsingHeaderState struct {
	Open bool
}

// SliderState tracks state for slider widgets.
type SliderState struct {
	Dragging       bool    // True when the grab handle is being dragged
	DragStartX     float32 // Mouse X position when drag started
	DragStartValue float32 // Value when drag started
}

// ComboBoxState tracks state for combo box widgets.
type ComboBoxState struct {
	Open          bool    // True when dropdown is open
	ScrollY       float32 // Scroll position in dropdown
	HoveredIndex  int     // Currently hovered item index (-1 = none)
	KeyboardIndex int     // Currently keyboard-selected index (-1 = none)
	SearchText    string  // Text typed for filtering (when searchable)
}

// ScrollableState tracks state for scrollable areas.
type ScrollableState struct {
	ScrollY       float32 // Vertical scroll position
	ScrollX       float32 // Horizontal scroll position (when enabled)
	TargetScrollY float32 // Target vertical position (for smooth scrolling)
	TargetScrollX float32 // Target horizontal position (for smooth scrolling)
	ContentHeight float32 // Measured content height
	ContentWidth  float32 // Measured content width
	Dragging      bool    // True when scrollbar thumb is being dragged
	DragStartY    float32 // Mouse Y when scrollbar drag started
	DragStartScr  float32 // ScrollY when scrollbar drag started
	LastFocusY    float32 // Previous frame's focus Y (for change detection)
	FocusYSet     bool    // True if focus Y was set (to distinguish 0 from "not set")

	// User scroll tracking - suppresses auto-scroll during manual interaction
	UserScrolledThisFrame bool    // True if user scrolled via mouse/keyboard this frame
	UserScrollTime        float32 // Time since last user scroll (for cooldown)
}

// UpdateSmoothScroll smoothly interpolates scroll positions toward targets.
// Call this each frame. Returns true if still animating.
func (s *ScrollableState) UpdateSmoothScroll(deltaTime float32) bool {
	const smoothSpeed = 15.0
	const threshold = 0.5

	animating := false

	// Vertical
	diffY := s.TargetScrollY - s.ScrollY
	if absf32(diffY) < threshold {
		s.ScrollY = s.TargetScrollY
	} else {
		s.ScrollY += diffY * deltaTime * smoothSpeed
		animating = true
	}

	// Horizontal
	diffX := s.TargetScrollX - s.ScrollX
	if absf32(diffX) < threshold {
		s.ScrollX = s.TargetScrollX
	} else {
		s.ScrollX += diffX * deltaTime * smoothSpeed
		animating = true
	}

	return animating
}

// ListState tracks state for list components.
type ListState struct {
	ScrollY           float32         // Scroll position
	SearchText        string          // Current search/filter text
	FilterEditing     bool            // True when filter input is in edit mode
	CollapsedSections map[string]bool // Section collapsed states (true = collapsed)
	SelectedIndex     int             // Currently selected item index
}

// NumberInputState tracks state for number input widgets.
type NumberInputState struct {
	Editing        bool    // True when in text edit mode
	EditText       string  // Text being edited
	Dragging       bool    // True when value is being dragged
	DragStartX     float32 // Mouse X when drag started
	DragStartValue float32 // Value when drag started
}

// ResizableEdge represents which edge(s) of a panel are being resized.
type ResizableEdge uint8

const (
	ResizeEdgeNone   ResizableEdge = 0
	ResizeEdgeLeft   ResizableEdge = 1 << 0
	ResizeEdgeRight  ResizableEdge = 1 << 1
	ResizeEdgeTop    ResizableEdge = 1 << 2
	ResizeEdgeBottom ResizableEdge = 1 << 3
)

// ResizeState tracks the state of a panel resize operation.
type ResizeState struct {
	Active      bool          // Currently being resized
	Edge        ResizableEdge // Which edge(s) are being resized
	StartMouseX float32       // Mouse X when resize started
	StartMouseY float32       // Mouse Y when resize started
	StartX      float32       // Panel X when resize started
	StartY      float32       // Panel Y when resize started
	StartW      float32       // Panel width when resize started
	StartH      float32       // Panel height when resize started
}
