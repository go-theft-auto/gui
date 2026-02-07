package gui

import (
	"log/slog"
	"os"
)

// guiLogLevel controls the log level for GUI debug logging.
// Default is LevelInfo, which suppresses Debug messages.
// SetVerbose(true) sets it to LevelDebug.
var guiLogLevel = new(slog.LevelVar)

// SetVerbose enables or disables verbose/debug logging for GUI components.
// Call this from main() after parsing flags.
func SetVerbose(v bool) {
	if v {
		guiLogLevel.Set(slog.LevelDebug)
	} else {
		guiLogLevel.Set(slog.LevelInfo)
	}
}

// guiVerbose returns true if GUI debug logging is enabled.
func guiVerbose() bool {
	return guiLogLevel.Level() <= slog.LevelDebug
}

// focusLogger is the logger for focus registry debugging.
var focusLogger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: guiLogLevel}))

// FocusRegistry manages focusable widgets within a single frame.
// This bridges the immediate-mode GUI paradigm with the Focusable interface.
//
// In immediate-mode GUI, widgets don't persist between frames - they're drawn fresh each frame.
// The FocusRegistry solves this by:
//  1. Widgets register themselves as focusable during drawing
//  2. The registry tracks which widget has focus via ID matching
//  3. Navigation methods move focus between registered widgets
//  4. Debug highlighting is drawn automatically for focused widgets
//
// IMPORTANT: Due to frame ordering (HandleInput before Draw), the registry uses
// double-buffering. Navigation uses the previous frame's registrations while
// new registrations build up in the current frame's buffer.
//
// Usage:
//
//	// In Context initialization
//	ctx.focusRegistry = NewFocusRegistry()
//
//	// In widget drawing
//	focusable := ctx.RegisterFocusable(id, "button", rect, FocusTypeLeaf)
//	if focusable.IsFocused() {
//	    ctx.DrawDebugFocusRect(rect.X, rect.Y, rect.W, rect.H)
//	}
//
//	// In panel HandleInput
//	if input.KeyPressed(KeyUp) {
//	    ctx.NavigateFocus(NavUp)
//	}
type FocusRegistry struct {
	// Double-buffered items: previous frame's items used for navigation,
	// current frame's items being built during Draw
	prevItems []FocusableItem // Previous frame - used for navigation in HandleInput
	items     []FocusableItem // Current frame - being built during Draw

	// currentFocusID is the ID of the currently focused widget
	currentFocusID ID

	// currentFocusIdx is the index of the focused widget in prevItems (-1 if none)
	currentFocusIdx int

	// scopeStack tracks nested focus scopes (containers)
	scopeStack []FocusScopeEntry

	// pendingFocusID is set when focus should change next frame
	pendingFocusID ID

	// navHandler is called when navigation occurs, allowing custom behavior
	navHandler func(dir NavDirection) bool

	// lastResetFrame tracks which frame we last reset on to prevent double-reset
	lastResetFrame uint64

	// keyboardNavigated controls whether Scrollable should auto-scroll to focused items.
	// Defaults to true (auto-scroll enabled). Set to false to disable auto-scroll
	// for specific interactions (e.g., mouse clicks that shouldn't trigger scroll).
	keyboardNavigated bool
}

// FocusScopeEntry represents a nested focus scope (container).
type FocusScopeEntry struct {
	ID           ID
	Name         string
	Type         FocusType
	Rect         Rect
	StartIdx     int // Index of first child in items
	FocusedChild int // Which child has focus (-1 = none)
}

// FocusableItem represents a widget that can receive focus.
// This is the immediate-mode equivalent of implementing the Focusable interface.
type FocusableItem struct {
	ID       ID        // Unique widget identifier
	Name     string    // Debug-friendly name
	Rect     Rect      // Bounds for hit testing and navigation
	Type     FocusType // Widget category
	ScopeIdx int       // Index of parent scope (-1 if root level)
	CanFocus bool      // Whether this widget can receive focus
	NavUp    ID        // Custom navigation target for up direction (0 = auto)
	NavDown  ID        // Custom navigation target for down direction (0 = auto)
	NavLeft  ID        // Custom navigation target for left direction (0 = auto)
	NavRight ID        // Custom navigation target for right direction (0 = auto)
}

// FocusableHandle is returned by RegisterFocusable and implements the Focusable interface.
// It provides methods to check focus state and handle navigation.
type FocusableHandle struct {
	registry *FocusRegistry
	item     *FocusableItem
	index    int
}

// NewFocusRegistry creates a new focus registry.
func NewFocusRegistry() *FocusRegistry {
	return &FocusRegistry{
		prevItems:       make([]FocusableItem, 0, 64),
		items:           make([]FocusableItem, 0, 64),
		currentFocusIdx: -1,
		scopeStack:      make([]FocusScopeEntry, 0, 8),
	}
}

// Reset prepares the registry for a new frame.
// Called at the start of each frame by GUI.PrepareInputHandling().
// Uses double-buffering: previous frame's items are kept for navigation
// while current frame builds new registrations.
//
// The frameNumber parameter prevents double-reset when called multiple times
// in the same frame (e.g., from PrepareInputHandling and Context.Reset).
func (r *FocusRegistry) ResetForFrame(frameNumber uint64) {
	// Prevent double-reset in the same frame
	if r.lastResetFrame == frameNumber && frameNumber > 0 {
		return
	}
	r.lastResetFrame = frameNumber

	// Enable auto-scroll by default at frame start
	// Set to false explicitly to disable auto-scroll for specific interactions
	r.keyboardNavigated = true

	// Swap buffers: current items become previous, then clear current
	r.prevItems, r.items = r.items, r.prevItems
	r.items = r.items[:0]
	r.currentFocusIdx = -1
	r.scopeStack = r.scopeStack[:0]

	// Update currentFocusIdx for prevItems
	for i, item := range r.prevItems {
		if item.ID == r.currentFocusID {
			r.currentFocusIdx = i
			break
		}
	}

	// Apply pending focus change
	if r.pendingFocusID != 0 {
		r.currentFocusID = r.pendingFocusID
		r.pendingFocusID = 0
		// Update index for the new focus
		r.currentFocusIdx = -1
		for i, item := range r.prevItems {
			if item.ID == r.currentFocusID {
				r.currentFocusIdx = i
				break
			}
		}
	}
}

// Reset is a convenience method that calls ResetForFrame(0).
// Deprecated: Use ResetForFrame with a frame number for proper double-reset protection.
func (r *FocusRegistry) Reset() {
	r.ResetForFrame(0)
}

// debugRegistration enables verbose logging of widget registration.
// Set to true to debug focus registration issues.
var debugRegistration = true

// Register adds a focusable widget to the registry.
// Returns a FocusableHandle that can be used to check focus state.
func (r *FocusRegistry) Register(id ID, name string, rect Rect, typ FocusType) *FocusableHandle {
	// Determine parent scope
	scopeIdx := -1
	if len(r.scopeStack) > 0 {
		scopeIdx = len(r.scopeStack) - 1
	}

	item := FocusableItem{
		ID:       id,
		Name:     name,
		Rect:     rect,
		Type:     typ,
		ScopeIdx: scopeIdx,
		CanFocus: true,
	}

	idx := len(r.items)
	r.items = append(r.items, item)

	// Update current focus index if this is the focused widget
	if id == r.currentFocusID {
		r.currentFocusIdx = idx
	}

	return &FocusableHandle{
		registry: r,
		item:     &r.items[idx],
		index:    idx,
	}
}

// RegisterDisabled adds a widget that cannot receive focus but is tracked for navigation.
func (r *FocusRegistry) RegisterDisabled(id ID, name string, rect Rect, typ FocusType) *FocusableHandle {
	handle := r.Register(id, name, rect, typ)
	handle.item.CanFocus = false
	return handle
}

// BeginScope starts a new focus scope (container).
// Child widgets registered after this are considered part of the scope.
func (r *FocusRegistry) BeginScope(id ID, name string, typ FocusType, rect Rect) {
	entry := FocusScopeEntry{
		ID:           id,
		Name:         name,
		Type:         typ,
		Rect:         rect,
		StartIdx:     len(r.items),
		FocusedChild: -1,
	}
	r.scopeStack = append(r.scopeStack, entry)
}

// EndScope ends the current focus scope.
// Returns info about which child had focus.
func (r *FocusRegistry) EndScope() FocusScopeEntry {
	n := len(r.scopeStack)
	if n == 0 {
		return FocusScopeEntry{FocusedChild: -1}
	}

	entry := r.scopeStack[n-1]
	r.scopeStack = r.scopeStack[:n-1]

	// Find which child had focus (check current items being built)
	for i := entry.StartIdx; i < len(r.items); i++ {
		if r.items[i].ID == r.currentFocusID {
			entry.FocusedChild = i - entry.StartIdx
			break
		}
	}

	return entry
}

// SetFocus sets focus to the widget with the given ID.
// Searches prevItems for navigation (double-buffered).
func (r *FocusRegistry) SetFocus(id ID) {
	r.currentFocusID = id
	// Update index if widget is in prevItems
	for i, item := range r.prevItems {
		if item.ID == id {
			r.currentFocusIdx = i
			return
		}
	}
	// Widget not in prevItems - will be matched on next frame
	r.currentFocusIdx = -1
}

// SetFocusDeferred sets focus to take effect next frame.
// Use this when setting focus from outside the render loop.
func (r *FocusRegistry) SetFocusDeferred(id ID) {
	r.pendingFocusID = id
}

// ClearFocus removes focus from all widgets.
func (r *FocusRegistry) ClearFocus() {
	r.currentFocusID = 0
	r.currentFocusIdx = -1
}

// CurrentFocusID returns the ID of the currently focused widget.
func (r *FocusRegistry) CurrentFocusID() ID {
	return r.currentFocusID
}

// CurrentFocusIdx returns the index of the currently focused widget in prevItems.
// Returns -1 if no widget is focused.
func (r *FocusRegistry) CurrentFocusIdx() int {
	return r.currentFocusIdx
}

// CurrentFocusItem returns the currently focused item, or nil if none.
// Uses prevItems for consistency with navigation (double-buffered).
func (r *FocusRegistry) CurrentFocusItem() *FocusableItem {
	if r.currentFocusIdx >= 0 && r.currentFocusIdx < len(r.prevItems) {
		return &r.prevItems[r.currentFocusIdx]
	}
	return nil
}

// WasKeyboardNavigated returns true if auto-scroll should be enabled.
// Defaults to true at frame start. Set to false to disable auto-scroll
// for specific interactions that shouldn't trigger scrolling.
func (r *FocusRegistry) WasKeyboardNavigated() bool {
	return r.keyboardNavigated
}

// MarkKeyboardNavigated manually sets the keyboard navigation flag.
// Call this from panels that use custom navigation (not NavigateFocus)
// to enable auto-scroll when navigating via keyboard.
func (r *FocusRegistry) MarkKeyboardNavigated() {
	r.keyboardNavigated = true
}

// Navigate moves focus in the given direction.
// Returns true if focus moved, false if at boundary or no focusable widgets.
// Uses the previous frame's items for navigation (double-buffered).
// Sets keyboardNavigated flag on success, enabling auto-scroll in Scrollable.
func (r *FocusRegistry) Navigate(dir NavDirection) bool {
	if len(r.prevItems) == 0 {
		focusLogger.Debug("Navigate: no prevItems available (widgets not registered yet)")
		return false
	}

	// Find current focus index
	currentIdx := r.currentFocusIdx
	if currentIdx < 0 {
		// No current focus - focus first focusable item
		focusLogger.Debug("Navigate: no current focus, looking for first focusable",
			"prevItemsCount", len(r.prevItems))
		for i, item := range r.prevItems {
			if item.CanFocus {
				focusLogger.Debug("Navigate: auto-focusing first widget",
					"idx", i, "name", item.Name, "id", item.ID)
				r.setFocusByIndex(i)
				r.keyboardNavigated = true
				return true
			}
		}
		focusLogger.Debug("Navigate: no focusable items found in prevItems")
		return false
	}

	// Check for custom navigation target
	if currentIdx >= len(r.prevItems) {
		return false
	}
	current := &r.prevItems[currentIdx]
	var targetID ID
	switch dir {
	case NavUp:
		targetID = current.NavUp
	case NavDown:
		targetID = current.NavDown
	case NavLeft:
		targetID = current.NavLeft
	case NavRight:
		targetID = current.NavRight
	}

	if targetID != 0 {
		// Use custom target
		for i, item := range r.prevItems {
			if item.ID == targetID && item.CanFocus {
				r.setFocusByIndex(i)
				r.keyboardNavigated = true
				return true
			}
		}
	}

	// Auto-navigation based on direction
	var success bool
	switch dir {
	case NavUp, NavDown:
		success = r.navigateVertical(dir)
	case NavLeft, NavRight:
		success = r.navigateHorizontal(dir)
	}

	if success {
		r.keyboardNavigated = true
	}
	return success
}

// navigateVertical handles up/down navigation.
// Uses prevItems for navigation (double-buffered).
func (r *FocusRegistry) navigateVertical(dir NavDirection) bool {
	currentIdx := r.currentFocusIdx
	if currentIdx < 0 || currentIdx >= len(r.prevItems) {
		focusLogger.Debug("navigateVertical: invalid index", "currentIdx", currentIdx, "prevItemsLen", len(r.prevItems))
		return false
	}

	// Simple linear navigation for now
	delta := 1
	if dir == NavUp {
		delta = -1
	}

	focusLogger.Debug("navigateVertical: from", "idx", currentIdx, "name", r.prevItems[currentIdx].Name, "delta", delta)

	// Find next focusable item
	for i := currentIdx + delta; i >= 0 && i < len(r.prevItems); i += delta {
		item := &r.prevItems[i]
		focusLogger.Debug("navigateVertical: checking", "idx", i, "name", item.Name, "canFocus", item.CanFocus)
		if item.CanFocus {
			r.setFocusByIndex(i)
			focusLogger.Debug("navigateVertical: moved to", "idx", i, "name", item.Name)
			return true
		}
	}

	focusLogger.Debug("navigateVertical: no focusable item found")
	return false
}

// navigateHorizontal handles left/right navigation.
// Uses prevItems for navigation (double-buffered).
func (r *FocusRegistry) navigateHorizontal(dir NavDirection) bool {
	currentIdx := r.currentFocusIdx
	if currentIdx < 0 || currentIdx >= len(r.prevItems) {
		return false
	}

	current := r.prevItems[currentIdx]

	// Find nearest focusable item in the horizontal direction
	bestIdx := -1
	bestDist := float32(1e9)

	for i, item := range r.prevItems {
		if i == currentIdx || !item.CanFocus {
			continue
		}

		// Check horizontal direction
		if dir == NavLeft {
			if item.Rect.X >= current.Rect.X {
				continue // Not to the left
			}
		} else {
			if item.Rect.X <= current.Rect.X {
				continue // Not to the right
			}
		}

		// Calculate distance (Manhattan for simplicity)
		dx := absf(item.Rect.X - current.Rect.X)
		dy := absf(item.Rect.Y - current.Rect.Y)
		dist := dx + dy*2 // Penalize vertical distance

		if dist < bestDist {
			bestDist = dist
			bestIdx = i
		}
	}

	if bestIdx >= 0 {
		r.setFocusByIndex(bestIdx)
		return true
	}

	return false
}

// setFocusByIndex sets focus to the item at the given index in prevItems.
func (r *FocusRegistry) setFocusByIndex(idx int) {
	if idx >= 0 && idx < len(r.prevItems) {
		r.currentFocusIdx = idx
		r.currentFocusID = r.prevItems[idx].ID
	}
}

// FocusFirst sets focus to the first focusable widget.
// Uses prevItems for navigation (double-buffered).
func (r *FocusRegistry) FocusFirst() bool {
	for i, item := range r.prevItems {
		if item.CanFocus {
			r.setFocusByIndex(i)
			return true
		}
	}
	return false
}

// FocusLast sets focus to the last focusable widget.
// Uses prevItems for navigation (double-buffered).
func (r *FocusRegistry) FocusLast() bool {
	for i := len(r.prevItems) - 1; i >= 0; i-- {
		if r.prevItems[i].CanFocus {
			r.setFocusByIndex(i)
			return true
		}
	}
	return false
}

// FocusByIndex sets focus to the item at the given registration index.
// Uses prevItems for navigation (double-buffered).
func (r *FocusRegistry) FocusByIndex(idx int) bool {
	if idx >= 0 && idx < len(r.prevItems) && r.prevItems[idx].CanFocus {
		r.setFocusByIndex(idx)
		return true
	}
	return false
}

// ItemCount returns the number of registered focusable items from the previous frame.
// This is the count used for navigation (double-buffered).
func (r *FocusRegistry) ItemCount() int {
	return len(r.prevItems)
}

// CurrentItemCount returns the number of items registered in the current frame so far.
// This is for debugging - shows items being built during Draw.
func (r *FocusRegistry) CurrentItemCount() int {
	return len(r.items)
}

// Items returns all registered items from the previous frame (for debugging/inspection).
// This is what's used for navigation (double-buffered).
func (r *FocusRegistry) Items() []FocusableItem {
	return r.prevItems
}

// SetNavHandler sets a custom navigation handler.
// Return true to indicate navigation was handled, false for default behavior.
func (r *FocusRegistry) SetNavHandler(handler func(dir NavDirection) bool) {
	r.navHandler = handler
}

// =============================================================================
// FocusableHandle methods - implements Focusable interface conceptually
// =============================================================================

// IsFocused returns true if this widget currently has focus.
func (h *FocusableHandle) IsFocused() bool {
	return h.registry.currentFocusID == h.item.ID
}

// CanFocus returns true if this widget can receive focus.
func (h *FocusableHandle) CanFocus() bool {
	return h.item.CanFocus
}

// HandleNav processes a navigation input.
// For leaf widgets, this always returns false (propagate to parent).
// Container widgets should override by setting custom nav targets.
func (h *FocusableHandle) HandleNav(dir NavDirection) bool {
	// Leaf widgets don't handle nav internally
	return false
}

// FocusBounds returns the rectangle for auto-scroll purposes.
func (h *FocusableHandle) FocusBounds() Rect {
	return h.item.Rect
}

// SetNavTarget sets a custom navigation target for a direction.
func (h *FocusableHandle) SetNavTarget(dir NavDirection, targetID ID) {
	switch dir {
	case NavUp:
		h.item.NavUp = targetID
	case NavDown:
		h.item.NavDown = targetID
	case NavLeft:
		h.item.NavLeft = targetID
	case NavRight:
		h.item.NavRight = targetID
	}
}

// Focus requests focus for this widget.
func (h *FocusableHandle) Focus() {
	h.registry.SetFocus(h.item.ID)
}

// Index returns the registration index of this item.
func (h *FocusableHandle) Index() int {
	return h.index
}

// Note: absf is defined in helpers.go
