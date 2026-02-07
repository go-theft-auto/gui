package gui

import "testing"

// mockPanel is a simple Panel implementation for testing.
type mockPanel struct {
	open    bool
	canOpen bool
	name    string
}

func (p *mockPanel) Open()                        { p.open = true }
func (p *mockPanel) Close()                       { p.open = false }
func (p *mockPanel) Toggle() bool                 { p.open = !p.open; return p.open }
func (p *mockPanel) IsOpen() bool                 { return p.open }
func (p *mockPanel) CanOpen() bool                { return p.canOpen }
func (p *mockPanel) Draw(ctx *Context)            {}
func (p *mockPanel) HandleInput(*InputState) bool { return false }

func newMockPanel(name string) *mockPanel {
	return &mockPanel{name: name, canOpen: true}
}

func TestFocusManager_CycleNext(t *testing.T) {
	registry := NewPanelRegistry()
	registry.SetExclusive(false)

	panel1 := newMockPanel("Panel1")
	panel2 := newMockPanel("Panel2")
	panel3 := newMockPanel("Panel3")

	registry.Register("Panel1", panel1, KeyNone, 0)
	registry.Register("Panel2", panel2, KeyNone, 0)
	registry.Register("Panel3", panel3, KeyNone, 0)

	fm := registry.FocusManager()

	// No panels open, focus should be invalid
	fm.Update()
	if fm.FocusedPanel() != nil {
		t.Error("Expected no focused panel when all are closed")
	}

	// Open some panels
	panel1.Open()
	panel2.Open()
	fm.Update()

	// First update should set focus to first open panel (index 0)
	if fm.focusedIndex != 0 {
		t.Errorf("Expected focusedIndex=0 after update with open panels, got %d", fm.focusedIndex)
	}

	// Cycle next
	fm.FocusNext()
	if fm.focusedIndex != 1 {
		t.Errorf("Expected focusedIndex=1 after FocusNext, got %d", fm.focusedIndex)
	}

	// Cycle next should wrap around
	fm.FocusNext()
	if fm.focusedIndex != 0 {
		t.Errorf("Expected focusedIndex=0 after wrap, got %d", fm.focusedIndex)
	}
}

func TestFocusManager_CyclePrev(t *testing.T) {
	registry := NewPanelRegistry()
	registry.SetExclusive(false)

	panel1 := newMockPanel("Panel1")
	panel2 := newMockPanel("Panel2")

	registry.Register("Panel1", panel1, KeyNone, 0)
	registry.Register("Panel2", panel2, KeyNone, 0)

	fm := registry.FocusManager()

	panel1.Open()
	panel2.Open()
	fm.Update()

	// Start at index 0, go prev should wrap to last
	fm.FocusPrev()
	if fm.focusedIndex != 1 {
		t.Errorf("Expected focusedIndex=1 after FocusPrev wrap, got %d", fm.focusedIndex)
	}

	// Go prev again
	fm.FocusPrev()
	if fm.focusedIndex != 0 {
		t.Errorf("Expected focusedIndex=0 after FocusPrev, got %d", fm.focusedIndex)
	}
}

func TestFocusManager_FocusByName(t *testing.T) {
	registry := NewPanelRegistry()
	registry.SetExclusive(false)

	panel1 := newMockPanel("Panel1")
	panel2 := newMockPanel("Panel2")

	registry.Register("Panel1", panel1, KeyNone, 0)
	registry.Register("Panel2", panel2, KeyNone, 0)

	fm := registry.FocusManager()

	panel1.Open()
	panel2.Open()
	fm.Update()

	fm.FocusPanelByName("Panel2")
	if fm.FocusedPanelName() != "Panel2" {
		t.Errorf("Expected focused panel 'Panel2', got '%s'", fm.FocusedPanelName())
	}

	fm.FocusPanelByName("Panel1")
	if fm.FocusedPanelName() != "Panel1" {
		t.Errorf("Expected focused panel 'Panel1', got '%s'", fm.FocusedPanelName())
	}
}

func TestFocusManager_HandleInput_CtrlTab(t *testing.T) {
	registry := NewPanelRegistry()
	registry.SetExclusive(false)

	panel1 := newMockPanel("Panel1")
	panel2 := newMockPanel("Panel2")

	registry.Register("Panel1", panel1, KeyNone, 0)
	registry.Register("Panel2", panel2, KeyNone, 0)

	fm := registry.FocusManager()

	panel1.Open()
	panel2.Open()
	fm.Update()

	// Simulate Ctrl+Tab
	input := NewInputState()
	input.SetKey(KeyTab, true)
	input.ModCtrl = true

	consumed := fm.HandleInput(input)
	if !consumed {
		t.Error("Expected Ctrl+Tab to be consumed")
	}

	if fm.focusedIndex != 1 {
		t.Errorf("Expected focusedIndex=1 after Ctrl+Tab, got %d", fm.focusedIndex)
	}
}

func TestFocusManager_HandleInput_CtrlShiftTab(t *testing.T) {
	registry := NewPanelRegistry()
	registry.SetExclusive(false)

	panel1 := newMockPanel("Panel1")
	panel2 := newMockPanel("Panel2")

	registry.Register("Panel1", panel1, KeyNone, 0)
	registry.Register("Panel2", panel2, KeyNone, 0)

	fm := registry.FocusManager()

	panel1.Open()
	panel2.Open()
	fm.Update()

	// Simulate Ctrl+Shift+Tab
	input := NewInputState()
	input.SetKey(KeyTab, true)
	input.ModCtrl = true
	input.ModShift = true

	consumed := fm.HandleInput(input)
	if !consumed {
		t.Error("Expected Ctrl+Shift+Tab to be consumed")
	}

	if fm.focusedIndex != 1 { // Should wrap from 0 to last (1)
		t.Errorf("Expected focusedIndex=1 after Ctrl+Shift+Tab wrap, got %d", fm.focusedIndex)
	}
}

func TestFocusManager_FocusVisible(t *testing.T) {
	registry := NewPanelRegistry()
	fm := registry.FocusManager()

	panel1 := newMockPanel("Panel1")
	registry.Register("Panel1", panel1, KeyNone, 0)

	// Focus not visible initially
	if fm.IsFocusVisible() {
		t.Error("Expected focus not visible initially")
	}

	panel1.Open()
	fm.Update()

	// Still not visible just from Update (auto-focus doesn't show ring)
	if fm.IsFocusVisible() {
		t.Error("Expected focus not visible after auto-focus")
	}

	// After explicit cycling, should be visible
	fm.FocusNext()
	if !fm.IsFocusVisible() {
		t.Error("Expected focus visible after explicit cycle")
	}

	// Clear focus
	fm.ClearFocus()
	if fm.IsFocusVisible() {
		t.Error("Expected focus not visible after clear")
	}
}

func TestFocusManager_PanelClose(t *testing.T) {
	registry := NewPanelRegistry()
	registry.SetExclusive(false)

	panel1 := newMockPanel("Panel1")
	panel2 := newMockPanel("Panel2")

	registry.Register("Panel1", panel1, KeyNone, 0)
	registry.Register("Panel2", panel2, KeyNone, 0)

	fm := registry.FocusManager()

	panel1.Open()
	panel2.Open()
	fm.Update()

	// Focus panel2
	fm.FocusPanelByName("Panel2")
	if fm.focusedIndex != 1 {
		t.Fatalf("Setup: expected focusedIndex=1, got %d", fm.focusedIndex)
	}

	// Close panel2, update should adjust index
	panel2.Close()
	fm.Update()

	// Should now be focused on only remaining panel (panel1)
	if fm.focusedIndex != 0 {
		t.Errorf("Expected focusedIndex=0 after closing panel2, got %d", fm.focusedIndex)
	}
}

// TestFocusRegistry_NavigateWidgets tests widget focus navigation,
// simulating the flow used by Widget Showcase panel.
func TestFocusRegistry_NavigateWidgets(t *testing.T) {
	registry := NewFocusRegistry()

	// Simulate Frame 1: Register some widgets (simulating Draw)
	registry.Register(1, "Section1", Rect{X: 0, Y: 0, W: 100, H: 20}, FocusTypeSection)
	registry.Register(2, "ComboBox", Rect{X: 0, Y: 25, W: 100, H: 20}, FocusTypeLeaf)
	registry.Register(3, "Section2", Rect{X: 0, Y: 50, W: 100, H: 20}, FocusTypeSection)
	registry.Register(4, "Button1", Rect{X: 0, Y: 75, W: 100, H: 20}, FocusTypeLeaf)

	// Verify items are registered
	if len(registry.items) != 4 {
		t.Fatalf("Expected 4 items registered, got %d", len(registry.items))
	}

	// Simulate Frame 2: Reset swaps buffers (like PrepareInputHandling)
	registry.ResetForFrame(2)

	// Now prevItems should have the widgets from Frame 1
	if len(registry.prevItems) != 4 {
		t.Fatalf("Expected 4 prevItems after reset, got %d", len(registry.prevItems))
	}
	if len(registry.items) != 0 {
		t.Fatalf("Expected 0 items after reset, got %d", len(registry.items))
	}

	// No focus yet
	if registry.CurrentFocusID() != 0 {
		t.Errorf("Expected no focus initially, got ID %d", registry.CurrentFocusID())
	}

	// Navigate down should focus first widget
	if !registry.Navigate(NavDown) {
		t.Error("Navigate(NavDown) should succeed and focus first widget")
	}

	// Should now be focused on first widget (Section1)
	if registry.CurrentFocusID() != 1 {
		t.Errorf("Expected focus on ID 1 (Section1), got %d", registry.CurrentFocusID())
	}

	// Navigate down again
	if !registry.Navigate(NavDown) {
		t.Error("Navigate(NavDown) should succeed")
	}
	if registry.CurrentFocusID() != 2 {
		t.Errorf("Expected focus on ID 2 (ComboBox), got %d", registry.CurrentFocusID())
	}

	// Navigate up
	if !registry.Navigate(NavUp) {
		t.Error("Navigate(NavUp) should succeed")
	}
	if registry.CurrentFocusID() != 1 {
		t.Errorf("Expected focus on ID 1 (Section1), got %d", registry.CurrentFocusID())
	}

	// Navigate up at boundary should fail
	if registry.Navigate(NavUp) {
		t.Error("Navigate(NavUp) at boundary should return false")
	}
}

// TestFocusRegistry_NavigateAcrossFrames simulates the exact flow
// that happens when Widget Showcase panel handles arrow keys.
func TestFocusRegistry_NavigateAcrossFrames(t *testing.T) {
	registry := NewFocusRegistry()

	// Frame 0: No widgets registered yet (panel not open)
	registry.ResetForFrame(1)

	// Frame 1: Panel opens, widgets are registered
	// (ResetForFrame happens at start, then Draw registers widgets)
	registry.ResetForFrame(2)
	// prevItems is empty (nothing from Frame 0)
	if len(registry.prevItems) != 0 {
		t.Fatalf("Expected 0 prevItems initially, got %d", len(registry.prevItems))
	}

	// Simulate Draw: register widgets
	registry.Register(1, "Section1", Rect{X: 0, Y: 0, W: 100, H: 20}, FocusTypeSection)
	registry.Register(2, "Button1", Rect{X: 0, Y: 25, W: 100, H: 20}, FocusTypeLeaf)

	// Frame 2: User presses Down arrow
	registry.ResetForFrame(3)
	// NOW prevItems has widgets from Frame 1
	if len(registry.prevItems) != 2 {
		t.Fatalf("Expected 2 prevItems after frame reset, got %d", len(registry.prevItems))
	}

	// Navigate should work now
	if !registry.Navigate(NavDown) {
		t.Error("Navigate(NavDown) should succeed on Frame 2")
	}
	if registry.CurrentFocusID() != 1 {
		t.Errorf("Expected focus on ID 1, got %d", registry.CurrentFocusID())
	}

	// Register same widgets in this frame (simulating Draw)
	registry.Register(1, "Section1", Rect{X: 0, Y: 0, W: 100, H: 20}, FocusTypeSection)
	registry.Register(2, "Button1", Rect{X: 0, Y: 25, W: 100, H: 20}, FocusTypeLeaf)

	// Frame 3: User presses Down again
	registry.ResetForFrame(4)

	// Focus should be preserved
	if registry.CurrentFocusID() != 1 {
		t.Errorf("Expected focus preserved on ID 1, got %d", registry.CurrentFocusID())
	}

	// Navigate down
	if !registry.Navigate(NavDown) {
		t.Error("Navigate(NavDown) should succeed")
	}
	if registry.CurrentFocusID() != 2 {
		t.Errorf("Expected focus on ID 2 (Button1), got %d", registry.CurrentFocusID())
	}
}
