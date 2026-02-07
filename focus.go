package gui

// FocusManager tracks which panel currently has keyboard focus and handles
// panel cycling with Ctrl+Tab. This enables ImGui-like window navigation
// without requiring a full docking system.
//
// Key features:
// - Tracks a single focused panel from the registry
// - Ctrl+Tab / Ctrl+Shift+Tab to cycle focus between open panels
// - Visual focus indicator ring on the focused panel
// - Arrow key navigation between adjacent panels (future)
type FocusManager struct {
	registry     *PanelRegistry
	focusedIndex int  // Index into registry's open panels (-1 = none)
	focusVisible bool // Whether to show the focus indicator ring
}

// NewFocusManager creates a new focus manager attached to a panel registry.
func NewFocusManager(registry *PanelRegistry) *FocusManager {
	return &FocusManager{
		registry:     registry,
		focusedIndex: -1,
		focusVisible: false,
	}
}

// FocusedPanel returns the currently focused panel, or nil if none.
func (fm *FocusManager) FocusedPanel() Panel {
	openPanels := fm.getOpenPanels()
	if fm.focusedIndex < 0 || fm.focusedIndex >= len(openPanels) {
		return nil
	}
	return openPanels[fm.focusedIndex].Panel
}

// FocusedPanelName returns the name of the currently focused panel, or empty string.
func (fm *FocusManager) FocusedPanelName() string {
	openPanels := fm.getOpenPanels()
	if fm.focusedIndex < 0 || fm.focusedIndex >= len(openPanels) {
		return ""
	}
	return openPanels[fm.focusedIndex].Name
}

// IsFocused returns true if the given panel is currently focused.
func (fm *FocusManager) IsFocused(panel Panel) bool {
	return panel != nil && panel == fm.FocusedPanel()
}

// IsFocusVisible returns true if the focus indicator should be drawn.
func (fm *FocusManager) IsFocusVisible() bool {
	return fm.focusVisible && fm.focusedIndex >= 0
}

// SetFocusVisible enables or disables the focus indicator ring.
func (fm *FocusManager) SetFocusVisible(visible bool) {
	fm.focusVisible = visible
}

// FocusPanel focuses a specific panel by reference.
func (fm *FocusManager) FocusPanel(panel Panel) {
	openPanels := fm.getOpenPanels()
	for i, entry := range openPanels {
		if entry.Panel == panel {
			fm.focusedIndex = i
			fm.focusVisible = true
			return
		}
	}
}

// FocusPanelByName focuses a specific panel by name.
func (fm *FocusManager) FocusPanelByName(name string) {
	openPanels := fm.getOpenPanels()
	for i, entry := range openPanels {
		if entry.Name == name {
			fm.focusedIndex = i
			fm.focusVisible = true
			return
		}
	}
}

// FocusNext cycles focus to the next open panel (Ctrl+Tab).
func (fm *FocusManager) FocusNext() {
	openPanels := fm.getOpenPanels()
	if len(openPanels) == 0 {
		fm.focusedIndex = -1
		return
	}

	fm.focusedIndex = (fm.focusedIndex + 1) % len(openPanels)
	fm.focusVisible = true
}

// FocusPrev cycles focus to the previous open panel (Ctrl+Shift+Tab).
func (fm *FocusManager) FocusPrev() {
	openPanels := fm.getOpenPanels()
	if len(openPanels) == 0 {
		fm.focusedIndex = -1
		return
	}

	fm.focusedIndex--
	if fm.focusedIndex < 0 {
		fm.focusedIndex = len(openPanels) - 1
	}
	fm.focusVisible = true
}

// ClearFocus removes focus from all panels.
func (fm *FocusManager) ClearFocus() {
	fm.focusedIndex = -1
	fm.focusVisible = false
}

// HandleInput processes focus-related keyboard input.
// Returns true if input was consumed.
func (fm *FocusManager) HandleInput(input *InputState) bool {
	if input == nil {
		return false
	}

	// Ctrl+Tab / Ctrl+Shift+Tab to cycle panels
	if input.KeyPressed(KeyTab) && input.ModCtrl {
		if input.ModShift {
			fm.FocusPrev()
		} else {
			fm.FocusNext()
		}
		return true
	}

	return false
}

// Update synchronizes the focused index with the registry's open panels.
// Call this each frame before handling input to ensure the focused panel
// is still valid (e.g., if a panel was closed externally).
func (fm *FocusManager) Update() {
	openPanels := fm.getOpenPanels()

	// If no panels are open, clear focus
	if len(openPanels) == 0 {
		fm.focusedIndex = -1
		fm.focusVisible = false
		return
	}

	// If the current focused index is out of bounds, reset it
	if fm.focusedIndex >= len(openPanels) {
		fm.focusedIndex = len(openPanels) - 1
	}

	// If no panel is focused but panels are open, focus the first one
	// (but don't show the indicator until the user explicitly cycles)
	if fm.focusedIndex < 0 {
		fm.focusedIndex = 0
		fm.focusVisible = false // Don't show ring until user explicitly cycles
	}
}

// getOpenPanels returns all currently open panels from the registry.
func (fm *FocusManager) getOpenPanels() []PanelEntry {
	if fm.registry == nil {
		return nil
	}

	entries := fm.registry.Entries()
	open := make([]PanelEntry, 0, len(entries))
	for _, e := range entries {
		if e.Panel.IsOpen() {
			open = append(open, e)
		}
	}
	return open
}

// DrawFocusRing draws a focus indicator ring around the given rectangle.
// Uses the style's FocusColor for theming support.
func DrawFocusRing(dl *DrawList, x, y, w, h float32, style Style) {
	offset := SpaceXS                 // Focus ring offset from panel edge
	thickness := style.BorderSize + 1 // Slightly thicker than normal border
	color := style.FocusColor
	if color == 0 {
		color = ColorCyan // Fallback to cyan if not set
	}

	// Draw outer glow/ring
	dl.AddRectOutline(
		x-offset,
		y-offset,
		w+offset*2,
		h+offset*2,
		color,
		thickness,
	)
}

// DrawFocusRingDebug draws a focus indicator ring with optional debug highlighting.
// When debugHighlight is true, uses red color instead of the style's FocusColor.
func DrawFocusRingDebug(dl *DrawList, x, y, w, h float32, style Style, debugHighlight bool) {
	offset := SpaceXS                 // Focus ring offset from panel edge
	thickness := style.BorderSize + 1 // Slightly thicker than normal border

	if debugHighlight {
		// Debug mode: draw red overlay and border
		dl.AddRect(x, y, w, h, DebugFocusColor)
		dl.AddRectOutline(
			x-offset,
			y-offset,
			w+offset*2,
			h+offset*2,
			DebugFocusBorderColor,
			thickness+1,
		)
		return
	}

	// Normal mode: use style's focus color
	color := style.FocusColor
	if color == 0 {
		color = ColorCyan // Fallback to cyan if not set
	}
	dl.AddRectOutline(
		x-offset,
		y-offset,
		w+offset*2,
		h+offset*2,
		color,
		thickness,
	)
}
