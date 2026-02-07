package gui

// ActionHandler is called when an action's hotkey is triggered.
type ActionHandler func()

// ActionCondition returns true if the action can be executed.
type ActionCondition func() bool

// ActionEntry holds a registered action with its hotkey and handler.
type ActionEntry struct {
	Name        string          // Action name for debugging
	CheckHotkey HotkeyCheck     // Returns true if hotkey is pressed
	Handler     ActionHandler   // Called when hotkey triggered
	Condition   ActionCondition // Optional: must return true to execute (nil = always)
	BlockedBy   []string        // Panel names that block this action
}

// ActionRegistry manages hotkey-triggered actions.
// Use this for global shortcuts that aren't panel toggles.
type ActionRegistry struct {
	actions []ActionEntry
	panels  *PanelRegistry // Reference to check blocked panels
}

// NewActionRegistry creates a new action registry.
// Pass the panel registry to enable BlockedBy checking.
func NewActionRegistry(panels *PanelRegistry) *ActionRegistry {
	return &ActionRegistry{
		actions: make([]ActionEntry, 0, 16),
		panels:  panels,
	}
}

// Register adds an action with a hotkey check and handler.
func (r *ActionRegistry) Register(name string, checkHotkey HotkeyCheck, handler ActionHandler) {
	r.actions = append(r.actions, ActionEntry{
		Name:        name,
		CheckHotkey: checkHotkey,
		Handler:     handler,
	})
}

// RegisterWithCondition adds an action with a condition that must be true to execute.
func (r *ActionRegistry) RegisterWithCondition(name string, checkHotkey HotkeyCheck, handler ActionHandler, condition ActionCondition) {
	r.actions = append(r.actions, ActionEntry{
		Name:        name,
		CheckHotkey: checkHotkey,
		Handler:     handler,
		Condition:   condition,
	})
}

// RegisterBlocked adds an action that's blocked when certain panels are open.
func (r *ActionRegistry) RegisterBlocked(name string, checkHotkey HotkeyCheck, handler ActionHandler, blockedBy ...string) {
	r.actions = append(r.actions, ActionEntry{
		Name:        name,
		CheckHotkey: checkHotkey,
		Handler:     handler,
		BlockedBy:   blockedBy,
	})
}

// RegisterFull adds an action with all options.
func (r *ActionRegistry) RegisterFull(name string, checkHotkey HotkeyCheck, handler ActionHandler, condition ActionCondition, blockedBy ...string) {
	r.actions = append(r.actions, ActionEntry{
		Name:        name,
		CheckHotkey: checkHotkey,
		Handler:     handler,
		Condition:   condition,
		BlockedBy:   blockedBy,
	})
}

// HandleActions checks all registered actions and executes matching handlers.
// Returns true if any action was triggered.
func (r *ActionRegistry) HandleActions() bool {
	for i := range r.actions {
		a := &r.actions[i]

		// Check if hotkey is pressed
		if a.CheckHotkey == nil || !a.CheckHotkey() {
			continue
		}

		// Check if blocked by open panel
		if r.isBlockedBy(a.BlockedBy) {
			continue
		}

		// Check condition
		if a.Condition != nil && !a.Condition() {
			continue
		}

		// Execute handler
		a.Handler()
		return true
	}
	return false
}

// isBlockedBy returns true if any of the named panels are open.
func (r *ActionRegistry) isBlockedBy(blockers []string) bool {
	if r.panels == nil || len(blockers) == 0 {
		return false
	}
	for _, blocker := range blockers {
		if p := r.panels.GetPanel(blocker); p != nil && p.IsOpen() {
			return true
		}
	}
	return false
}

// Unregister removes an action by name.
func (r *ActionRegistry) Unregister(name string) {
	for i, a := range r.actions {
		if a.Name == name {
			r.actions = append(r.actions[:i], r.actions[i+1:]...)
			return
		}
	}
}

// Clear removes all registered actions.
func (r *ActionRegistry) Clear() {
	r.actions = r.actions[:0]
}
