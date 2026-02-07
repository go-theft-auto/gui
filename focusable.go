package gui

// NavDirection represents a navigation direction for keyboard focus movement.
type NavDirection uint8

const (
	NavUp NavDirection = iota
	NavDown
	NavLeft
	NavRight
)

// String returns a human-readable name for the navigation direction.
func (d NavDirection) String() string {
	switch d {
	case NavUp:
		return "Up"
	case NavDown:
		return "Down"
	case NavLeft:
		return "Left"
	case NavRight:
		return "Right"
	default:
		return "Unknown"
	}
}

// Opposite returns the opposite direction (Up<->Down, Left<->Right).
func (d NavDirection) Opposite() NavDirection {
	switch d {
	case NavUp:
		return NavDown
	case NavDown:
		return NavUp
	case NavLeft:
		return NavRight
	case NavRight:
		return NavLeft
	default:
		return d
	}
}

// IsVertical returns true for Up/Down directions.
func (d NavDirection) IsVertical() bool {
	return d == NavUp || d == NavDown
}

// IsHorizontal returns true for Left/Right directions.
func (d NavDirection) IsHorizontal() bool {
	return d == NavLeft || d == NavRight
}

// Focusable is implemented by widgets that can receive keyboard focus.
// This interface enables the focus hierarchy to navigate between widgets.
//
// Widgets that contain focusable children should implement the container
// methods (FocusedChildIndex, FocusChild, ChildCount) in addition to the
// basic focus methods.
type Focusable interface {
	// IsFocused returns true if this widget currently has focus.
	IsFocused() bool

	// CanFocus returns true if this widget can receive focus.
	// Some widgets may be disabled or hidden and should return false.
	CanFocus() bool

	// HandleNav processes a navigation input and returns true if handled.
	// If the widget handles the navigation internally (e.g., moving between
	// items in a list), it returns true. If the navigation should propagate
	// to the parent (e.g., trying to move up from the first item), return false.
	HandleNav(dir NavDirection) bool
}

// FocusableContainer extends Focusable for widgets that contain focusable children.
// Examples: Panels, Sections, Lists, Tables.
type FocusableContainer interface {
	Focusable

	// FocusedChildIndex returns the index of the currently focused child.
	// Returns -1 if no child is focused (the container itself may have focus).
	FocusedChildIndex() int

	// FocusChild sets focus to the child at the given index.
	// Pass -1 to focus the container itself.
	FocusChild(index int)

	// ChildCount returns the number of focusable children.
	ChildCount() int
}

// FocusableWithBounds extends Focusable with bounds information for auto-scroll.
type FocusableWithBounds interface {
	Focusable

	// FocusBounds returns the rectangle that should be visible when focused.
	// Parent scrollables use this to auto-scroll to keep focused items visible.
	FocusBounds() Rect
}
