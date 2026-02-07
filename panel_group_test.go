package gui

import "testing"

func TestPanelGroup_AddRemove(t *testing.T) {
	pg := NewPanelGroup("test")

	panel1 := newMockPanel("Panel1")
	panel2 := newMockPanel("Panel2")

	pg.AddPanel("Tab1", panel1)
	pg.AddPanel("Tab2", panel2)

	if pg.PanelCount() != 2 {
		t.Errorf("Expected 2 panels, got %d", pg.PanelCount())
	}

	if pg.GetPanel("Tab1") != panel1 {
		t.Error("GetPanel returned wrong panel for Tab1")
	}

	// Remove panel
	if !pg.RemovePanel("Tab1") {
		t.Error("RemovePanel returned false for existing panel")
	}

	if pg.PanelCount() != 1 {
		t.Errorf("Expected 1 panel after remove, got %d", pg.PanelCount())
	}

	if pg.GetPanel("Tab1") != nil {
		t.Error("GetPanel should return nil for removed panel")
	}
}

func TestPanelGroup_TabSwitching(t *testing.T) {
	pg := NewPanelGroup("test")

	panel1 := newMockPanel("Panel1")
	panel2 := newMockPanel("Panel2")
	panel3 := newMockPanel("Panel3")

	pg.AddPanel("Tab1", panel1)
	pg.AddPanel("Tab2", panel2)
	pg.AddPanel("Tab3", panel3)

	// Initial state
	if pg.ActiveTab != 0 {
		t.Errorf("Expected initial ActiveTab=0, got %d", pg.ActiveTab)
	}

	if pg.ActivePanel() != panel1 {
		t.Error("Initial ActivePanel should be panel1")
	}

	// Set active tab by index
	pg.SetActiveTab(2)
	if pg.ActiveTab != 2 {
		t.Errorf("Expected ActiveTab=2, got %d", pg.ActiveTab)
	}

	if pg.ActivePanel() != panel3 {
		t.Error("ActivePanel should be panel3")
	}

	// Set active tab by name
	pg.SetActiveTabByName("Tab2")
	if pg.ActiveTab != 1 {
		t.Errorf("Expected ActiveTab=1 for Tab2, got %d", pg.ActiveTab)
	}

	// Invalid name should not change tab
	pg.SetActiveTabByName("NonExistent")
	if pg.ActiveTab != 1 {
		t.Errorf("Expected ActiveTab unchanged, got %d", pg.ActiveTab)
	}
}

func TestPanelGroup_CycleNext(t *testing.T) {
	pg := NewPanelGroup("test")

	pg.AddPanel("Tab1", newMockPanel("Panel1"))
	pg.AddPanel("Tab2", newMockPanel("Panel2"))
	pg.AddPanel("Tab3", newMockPanel("Panel3"))

	// Start at 0
	pg.ActiveTab = 0

	pg.NextTab()
	if pg.ActiveTab != 1 {
		t.Errorf("Expected ActiveTab=1 after NextTab, got %d", pg.ActiveTab)
	}

	pg.NextTab()
	if pg.ActiveTab != 2 {
		t.Errorf("Expected ActiveTab=2 after NextTab, got %d", pg.ActiveTab)
	}

	// Should wrap
	pg.NextTab()
	if pg.ActiveTab != 0 {
		t.Errorf("Expected ActiveTab=0 after wrap, got %d", pg.ActiveTab)
	}
}

func TestPanelGroup_CyclePrev(t *testing.T) {
	pg := NewPanelGroup("test")

	pg.AddPanel("Tab1", newMockPanel("Panel1"))
	pg.AddPanel("Tab2", newMockPanel("Panel2"))
	pg.AddPanel("Tab3", newMockPanel("Panel3"))

	// Start at 0
	pg.ActiveTab = 0

	// Should wrap to last
	pg.PrevTab()
	if pg.ActiveTab != 2 {
		t.Errorf("Expected ActiveTab=2 after wrap, got %d", pg.ActiveTab)
	}

	pg.PrevTab()
	if pg.ActiveTab != 1 {
		t.Errorf("Expected ActiveTab=1 after PrevTab, got %d", pg.ActiveTab)
	}
}

func TestPanelGroup_OpenClose(t *testing.T) {
	pg := NewPanelGroup("test")
	pg.AddPanel("Tab1", newMockPanel("Panel1"))

	if !pg.IsOpen() {
		t.Error("Expected group to be open initially")
	}

	closeCalled := false
	pg.SetOnClose(func() { closeCalled = true })

	pg.Close()

	if pg.IsOpen() {
		t.Error("Expected group to be closed")
	}
	if !closeCalled {
		t.Error("Expected onClose callback to be called")
	}

	closeCalled = false
	pg.Open()

	if !pg.IsOpen() {
		t.Error("Expected group to be open after Open()")
	}
	if closeCalled {
		t.Error("onClose should not be called on Open()")
	}
}

func TestPanelGroup_Toggle(t *testing.T) {
	pg := NewPanelGroup("test")
	pg.AddPanel("Tab1", newMockPanel("Panel1"))

	if !pg.IsOpen() {
		t.Fatal("Expected group to be open initially")
	}

	result := pg.Toggle()
	if result || pg.IsOpen() {
		t.Error("Expected Toggle to close group")
	}

	result = pg.Toggle()
	if !result || !pg.IsOpen() {
		t.Error("Expected Toggle to open group")
	}
}

func TestPanelGroup_HandleInput_TabCycle(t *testing.T) {
	pg := NewPanelGroup("test")
	pg.AddPanel("Tab1", newMockPanel("Panel1"))
	pg.AddPanel("Tab2", newMockPanel("Panel2"))

	input := NewInputState()
	input.ModCtrl = true
	input.SetKey(KeyPageDown, true)

	consumed := pg.HandleInput(input)
	if !consumed {
		t.Error("Expected Ctrl+PageDown to be consumed")
	}
	if pg.ActiveTab != 1 {
		t.Errorf("Expected ActiveTab=1 after Ctrl+PageDown, got %d", pg.ActiveTab)
	}

	// Test PgUp
	input.Reset()
	input.ModCtrl = true
	input.SetKey(KeyPageUp, true)

	consumed = pg.HandleInput(input)
	if !consumed {
		t.Error("Expected Ctrl+PageUp to be consumed")
	}
	if pg.ActiveTab != 0 {
		t.Errorf("Expected ActiveTab=0 after Ctrl+PageUp, got %d", pg.ActiveTab)
	}
}

func TestPanelGroup_HandleInput_Escape(t *testing.T) {
	pg := NewPanelGroup("test")
	pg.AddPanel("Tab1", newMockPanel("Panel1"))

	input := NewInputState()
	input.SetKey(KeyEscape, true)

	pg.HandleInput(input)

	if pg.IsOpen() {
		t.Error("Expected Escape to close the group")
	}
}

func TestPanelGroup_RemoveAdjustsActiveTab(t *testing.T) {
	pg := NewPanelGroup("test")
	pg.AddPanel("Tab1", newMockPanel("Panel1"))
	pg.AddPanel("Tab2", newMockPanel("Panel2"))
	pg.AddPanel("Tab3", newMockPanel("Panel3"))

	// Set active to last tab
	pg.SetActiveTab(2)

	// Remove last tab
	pg.RemovePanel("Tab3")

	// Active tab should adjust
	if pg.ActiveTab >= pg.PanelCount() {
		t.Errorf("ActiveTab %d is out of bounds after remove (count=%d)", pg.ActiveTab, pg.PanelCount())
	}
}

func TestPanelGroup_CanOpen(t *testing.T) {
	pg := NewPanelGroup("test")

	if !pg.CanOpen() {
		t.Error("PanelGroup.CanOpen() should always return true")
	}
}

func TestPanelGroup_EmptyGroup(t *testing.T) {
	pg := NewPanelGroup("test")

	// Empty group should handle gracefully
	if pg.ActivePanel() != nil {
		t.Error("ActivePanel should return nil for empty group")
	}

	// Tab cycling should not panic
	pg.NextTab()
	pg.PrevTab()
}
