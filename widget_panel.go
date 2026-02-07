package gui

// Panel is the interface for any openable UI panel or menu.
// Panels can register with a PanelRegistry to get automatic hotkey handling,
// input routing, and mutual exclusion.
type Panel interface {
	// Open opens the panel.
	Open()

	// Close closes the panel.
	Close()

	// Toggle toggles the panel open/closed state.
	// Returns true if the panel is now open.
	Toggle() bool

	// IsOpen returns true if the panel is currently open.
	IsOpen() bool

	// CanOpen returns true if the panel can be opened.
	// Use this for preconditions (e.g., needs a model loaded).
	// Default should return true.
	CanOpen() bool

	// Draw renders the panel using the provided GUI context.
	// This is called every frame regardless of open state - panels
	// should return early if not open.
	Draw(ctx *Context)

	// HandleInput processes input for the panel.
	// Returns true if input was consumed.
	// Called only when the panel is open.
	HandleInput(input *InputState) bool
}

// HotkeyCheck is a function that returns true if the panel's hotkey is pressed.
// This allows integration with settings-based rebindable keys.
type HotkeyCheck func() bool

// PanelEntry holds a registered panel with its configuration.
type PanelEntry struct {
	Name        string      // Display name (e.g., "Model Menu")
	Panel       Panel       // The panel itself
	Hotkey      Key         // Key to toggle the panel (simple mode)
	HotkeyName  string      // Display name for hotkey (used when CheckHotkey is set)
	CheckHotkey HotkeyCheck // Custom hotkey check (overrides Hotkey if set)
	CloseKey    Key         // Key to close the panel (default: KeyEscape)
	CheckClose  HotkeyCheck // Custom close key check (overrides CloseKey if set)
	Priority    int         // Higher priority panels handle input first
	NeedsCursor bool        // If true, opening this panel releases cursor
	BlockedBy   []string    // Panel names that block this panel's hotkey
	Global      bool        // If true, hotkey works in all modes (not just model view)
}

// IsCloseKeyPressed returns true if the panel's close key is pressed.
// Uses CheckClose if set, otherwise checks CloseKey.
// If neither is set, use defaultCheck (typically the registry's default close check).
func (e *PanelEntry) IsCloseKeyPressed(input *InputState, defaultCheck HotkeyCheck) bool {
	// Panel-specific close check takes priority
	if e.CheckClose != nil {
		return e.CheckClose()
	}
	// Panel-specific close key
	if e.CloseKey != KeyNone {
		return input.KeyPressed(e.CloseKey)
	}
	// Registry-level default close check (from settings)
	if defaultCheck != nil {
		return defaultCheck()
	}
	// Fallback to Escape if nothing else is configured
	return input.KeyPressed(KeyEscape)
}

// CursorChangeCallback is called when cursor capture state should change.
// captured=true means cursor should be captured (hidden, for FPS camera).
// captured=false means cursor should be released (visible, for UI interaction).
type CursorChangeCallback func(captured bool)

// PanelRegistry manages a collection of panels with automatic hotkey handling.
// It handles:
// - Opening/closing panels via hotkeys
// - Mutual exclusion (optional - close others when one opens)
// - Routing input to the currently active panel
// - Drawing all panels
// - Cursor capture state management
// - Focus management with Ctrl+Tab cycling
type PanelRegistry struct {
	entries           []PanelEntry
	exclusive         bool                 // If true, opening one panel closes others
	inputChars        bool                 // If true, consume input chars when a panel is open
	defaultCloseCheck HotkeyCheck          // Default close check for all panels (e.g., settings.Cancel)
	onCursorChange    CursorChangeCallback // Called when cursor state should change
	cursorReleased    bool                 // Current cursor state
	focusManager      *FocusManager        // Focus management for Ctrl+Tab cycling
}

// NewPanelRegistry creates a new panel registry.
func NewPanelRegistry() *PanelRegistry {
	r := &PanelRegistry{
		entries:    make([]PanelEntry, 0, 8),
		exclusive:  true,
		inputChars: true,
	}
	r.focusManager = NewFocusManager(r)
	return r
}

// SetExclusive sets whether opening one panel closes others.
func (r *PanelRegistry) SetExclusive(exclusive bool) {
	r.exclusive = exclusive
}

// SetCursorChangeCallback sets the callback for cursor state changes.
// This is called when panels open/close that require cursor interaction.
func (r *PanelRegistry) SetCursorChangeCallback(fn CursorChangeCallback) {
	r.onCursorChange = fn
}

// SetDefaultCloseCheck sets the default close key check for all panels.
// This is typically set to check the settings' Cancel binding.
// Panels can override this with their own CloseKey or CheckClose.
func (r *PanelRegistry) SetDefaultCloseCheck(check HotkeyCheck) {
	r.defaultCloseCheck = check
}

// Register adds a panel to the registry with its hotkey.
// Priority determines input handling order (higher = first).
// Use RegisterWithCursor for panels that need cursor interaction.
func (r *PanelRegistry) Register(name string, panel Panel, hotkey Key, priority int) {
	r.RegisterWithCursor(name, panel, hotkey, priority, false)
}

// RegisterWithCursor adds a panel with cursor capture control.
// If needsCursor is true, opening this panel will release cursor capture.
func (r *PanelRegistry) RegisterWithCursor(name string, panel Panel, hotkey Key, priority int, needsCursor bool) {
	r.entries = append(r.entries, PanelEntry{
		Name:        name,
		Panel:       panel,
		Hotkey:      hotkey,
		Priority:    priority,
		NeedsCursor: needsCursor,
	})
	// Sort by priority (descending)
	r.sortByPriority()
}

// RegisterWithBinding adds a panel with a custom hotkey check function.
// This allows integration with settings-based rebindable keys.
// If global is true, the hotkey works in all modes (not just model view mode).
// blockedBy lists panel names that prevent this panel's hotkey from working (e.g., typing in search).
func (r *PanelRegistry) RegisterWithBinding(name string, panel Panel, checkHotkey HotkeyCheck, priority int, needsCursor bool, global bool, blockedBy ...string) {
	r.entries = append(r.entries, PanelEntry{
		Name:        name,
		Panel:       panel,
		CheckHotkey: checkHotkey,
		Priority:    priority,
		NeedsCursor: needsCursor,
		Global:      global,
		BlockedBy:   blockedBy,
	})
	r.sortByPriority()
}

// SetCloseKey sets the close key for a panel by name.
// Pass KeyNone to use the default (Escape).
func (r *PanelRegistry) SetCloseKey(name string, closeKey Key) {
	for i := range r.entries {
		if r.entries[i].Name == name {
			r.entries[i].CloseKey = closeKey
			r.entries[i].CheckClose = nil // Clear custom check
			return
		}
	}
}

// SetCloseBinding sets a custom close key check function for a panel.
// This allows integration with settings-based rebindable close keys.
func (r *PanelRegistry) SetCloseBinding(name string, checkClose HotkeyCheck) {
	for i := range r.entries {
		if r.entries[i].Name == name {
			r.entries[i].CheckClose = checkClose
			return
		}
	}
}

// SetHotkeyName sets the display name for a panel's hotkey.
// Use this for panels with custom hotkey checks to show the actual key in the F1 help.
func (r *PanelRegistry) SetHotkeyName(name string, hotkeyName string) {
	for i := range r.entries {
		if r.entries[i].Name == name {
			r.entries[i].HotkeyName = hotkeyName
			return
		}
	}
}

// Unregister removes a panel from the registry.
func (r *PanelRegistry) Unregister(name string) {
	for i, e := range r.entries {
		if e.Name == name {
			r.entries = append(r.entries[:i], r.entries[i+1:]...)
			return
		}
	}
}

// GetPanel returns a panel by name, or nil if not found.
func (r *PanelRegistry) GetPanel(name string) Panel {
	for _, e := range r.entries {
		if e.Name == name {
			return e.Panel
		}
	}
	return nil
}

// IsAnyOpen returns true if any panel is currently open.
func (r *PanelRegistry) IsAnyOpen() bool {
	for _, e := range r.entries {
		if e.Panel.IsOpen() {
			return true
		}
	}
	return false
}

// CloseAll closes all panels and restores cursor capture.
func (r *PanelRegistry) CloseAll() {
	for _, e := range r.entries {
		e.Panel.Close()
	}
	r.updateCursorState()
}

// OpenPanel opens a specific panel by name.
// If exclusive mode is enabled, closes other panels first.
func (r *PanelRegistry) OpenPanel(name string) {
	if r.exclusive {
		for _, e := range r.entries {
			e.Panel.Close()
		}
	}
	if p := r.GetPanel(name); p != nil {
		p.Open()
	}
	r.updateCursorState()
}

// TogglePanel toggles a panel by name.
// Returns true if the panel is now open.
// If input is provided, consumes input chars to prevent hotkey from typing.
func (r *PanelRegistry) TogglePanel(name string) bool {
	return r.TogglePanelWithInput(name, nil)
}

// TogglePanelWithInput toggles a panel and optionally consumes input chars.
func (r *PanelRegistry) TogglePanelWithInput(name string, input *InputState) bool {
	for i := range r.entries {
		if r.entries[i].Name == name {
			panel := r.entries[i].Panel
			if panel.IsOpen() {
				panel.Close()
				r.updateCursorState()
				return false
			}
			// Check if panel can be opened
			if !panel.CanOpen() {
				return false
			}
			if r.exclusive {
				for j := range r.entries {
					r.entries[j].Panel.Close()
				}
			}
			panel.Open()
			r.updateCursorState()
			// Consume input chars to prevent hotkey from being typed
			if input != nil && r.inputChars {
				input.ConsumeInputChars()
			}
			return true
		}
	}
	return false
}

// updateCursorState checks if any cursor-needing panel is open and updates cursor.
func (r *PanelRegistry) updateCursorState() {
	needsCursor := false
	for _, e := range r.entries {
		if e.NeedsCursor && e.Panel.IsOpen() {
			needsCursor = true
			break
		}
	}

	if needsCursor != r.cursorReleased {
		r.cursorReleased = needsCursor
		if r.onCursorChange != nil {
			// captured = !needsCursor (if panel needs cursor, release capture)
			r.onCursorChange(!needsCursor)
		}
	}
}

// HandleHotkeys checks for hotkey presses and opens/closes panels.
// Call this each frame to handle panel hotkeys automatically.
// Returns true if a hotkey was handled.
// HandleHotkeys checks panel hotkeys and toggles them.
// modelViewMode indicates if we're in model view mode (vs world mode).
// Global panels work in all modes, non-global panels only work in model view mode.
func (r *PanelRegistry) HandleHotkeys(input *InputState, modelViewMode bool) bool {
	if input == nil {
		return false
	}

	// Check each panel's hotkey
	for i := range r.entries {
		e := &r.entries[i]

		// Skip non-global panels when not in model view mode
		if !modelViewMode && !e.Global {
			continue
		}

		// Check if hotkey is pressed (custom check or simple key)
		hotkeyPressed := false
		if e.CheckHotkey != nil {
			hotkeyPressed = e.CheckHotkey()
		} else if e.Hotkey != KeyNone {
			hotkeyPressed = input.KeyPressed(e.Hotkey)
		}

		if !hotkeyPressed {
			continue
		}

		// Check if blocked by another open panel
		if r.isBlockedBy(e.BlockedBy) {
			continue
		}

		// Toggle the panel
		r.TogglePanelWithInput(e.Name, input)
		return true
	}

	return false
}

// isBlockedBy returns true if any of the named panels are open.
func (r *PanelRegistry) isBlockedBy(blockers []string) bool {
	for _, blocker := range blockers {
		if p := r.GetPanel(blocker); p != nil && p.IsOpen() {
			return true
		}
	}
	return false
}

// HandleInput routes input to open panels.
// Returns true if input was consumed by any panel.
func (r *PanelRegistry) HandleInput(input *InputState) bool {
	if input == nil {
		return false
	}

	// Update focus manager to track open panels
	r.focusManager.Update()

	// Handle focus cycling (Ctrl+Tab / Ctrl+Shift+Tab)
	if r.focusManager.HandleInput(input) {
		return true
	}

	// Handle close keys for open panels (centralized close key handling)
	// This prevents the race condition where toggle and close happen in same frame
	for i := range r.entries {
		e := &r.entries[i]
		if e.Panel.IsOpen() && e.IsCloseKeyPressed(input, r.defaultCloseCheck) {
			e.Panel.Close()
			r.updateCursorState()
			return true
		}
	}

	// Route input to the focused panel first (if focus is visible)
	if r.focusManager.IsFocusVisible() {
		if focused := r.focusManager.FocusedPanel(); focused != nil && focused.IsOpen() {
			if focused.HandleInput(input) {
				r.updateCursorState()
				return true
			}
		}
	}

	// Fallback: route input to open panels by priority order
	for _, e := range r.entries {
		if e.Panel.IsOpen() {
			if e.Panel.HandleInput(input) {
				// After handling, check if cursor state needs updating
				r.updateCursorState()
				return true
			}
		}
	}

	return false
}

// Draw renders all open panels.
func (r *PanelRegistry) Draw(ctx *Context) {
	// Draw in reverse priority order (lowest priority drawn first = behind)
	for i := len(r.entries) - 1; i >= 0; i-- {
		r.entries[i].Panel.Draw(ctx)
	}
}

// sortByPriority sorts entries by priority (highest first).
func (r *PanelRegistry) sortByPriority() {
	// Simple insertion sort (small list)
	for i := 1; i < len(r.entries); i++ {
		key := r.entries[i]
		j := i - 1
		for j >= 0 && r.entries[j].Priority < key.Priority {
			r.entries[j+1] = r.entries[j]
			j--
		}
		r.entries[j+1] = key
	}
}

// Entries returns all registered panel entries (for inspection/debugging).
func (r *PanelRegistry) Entries() []PanelEntry {
	return r.entries
}

// FocusManager returns the focus manager for this registry.
// Use this for advanced focus control (cycling, checking focus state, etc.).
func (r *PanelRegistry) FocusManager() *FocusManager {
	return r.focusManager
}

// FocusedPanel returns the currently focused panel, or nil if none.
func (r *PanelRegistry) FocusedPanel() Panel {
	return r.focusManager.FocusedPanel()
}

// IsPanelFocused returns true if the given panel is currently focused.
func (r *PanelRegistry) IsPanelFocused(panel Panel) bool {
	return r.focusManager.IsFocused(panel)
}
