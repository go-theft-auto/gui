package gui

import "testing"

func TestDraggablePanel_BasicDrag(t *testing.T) {
	ctx := NewContext()
	ctx.SetStyle(DefaultStyle())
	ctx.Input = NewInputState()
	ctx.DisplaySize = Vec2{X: 800, Y: 600}

	dp := NewDraggablePanel(100, 100)
	dp.Size = Vec2{X: 200, Y: 150}

	// Verify initial position
	if dp.Position.X != 100 || dp.Position.Y != 100 {
		t.Errorf("Expected initial position (100, 100), got (%f, %f)", dp.Position.X, dp.Position.Y)
	}

	// Simulate click in title bar area
	ctx.Input.SetMousePos(150, 110) // Within title bar
	ctx.Input.SetMouseButton(MouseButtonLeft, true)

	dp.HandleDrag(ctx)

	if !dp.IsDragging() {
		t.Error("Expected panel to be dragging after click in title bar")
	}

	// Simulate drag movement
	ctx.Input.Reset()
	ctx.Input.SetMousePos(250, 210) // Move mouse 100px right, 100px down
	ctx.Input.SetMouseButton(MouseButtonLeft, true)

	dp.HandleDrag(ctx)

	// Panel should have moved
	if dp.Position.X != 200 || dp.Position.Y != 200 {
		t.Errorf("Expected position (200, 200) after drag, got (%f, %f)", dp.Position.X, dp.Position.Y)
	}

	// Release mouse to end drag
	ctx.Input.Reset()
	ctx.Input.SetMousePos(250, 210)
	ctx.Input.SetMouseButton(MouseButtonLeft, false)

	dp.HandleDrag(ctx)

	if dp.IsDragging() {
		t.Error("Expected panel to stop dragging after mouse release")
	}
}

func TestDraggablePanel_ClickOutsideTitleBar(t *testing.T) {
	ctx := NewContext()
	ctx.SetStyle(DefaultStyle())
	ctx.Input = NewInputState()
	ctx.DisplaySize = Vec2{X: 800, Y: 600}

	dp := NewDraggablePanel(100, 100)
	dp.Size = Vec2{X: 200, Y: 150}

	// Click outside title bar (in content area)
	titleBarHeight := ctx.LineHeight() + ctx.Style().PanelPadding*2
	ctx.Input.SetMousePos(150, 100+titleBarHeight+10) // Below title bar
	ctx.Input.SetMouseButton(MouseButtonLeft, true)

	dp.HandleDrag(ctx)

	if dp.IsDragging() {
		t.Error("Expected panel NOT to be dragging when clicking outside title bar")
	}
}

func TestDraggablePanel_ScreenBoundsClamp(t *testing.T) {
	ctx := NewContext()
	ctx.SetStyle(DefaultStyle())
	ctx.Input = NewInputState()
	ctx.DisplaySize = Vec2{X: 800, Y: 600}

	dp := NewDraggablePanel(100, 100)
	dp.Size = Vec2{X: 200, Y: 150}

	// Start drag
	ctx.Input.SetMousePos(150, 110)
	ctx.Input.SetMouseButton(MouseButtonLeft, true)
	dp.HandleDrag(ctx)

	// Try to drag off screen (to the left/top)
	ctx.Input.Reset()
	ctx.Input.SetMousePos(-100, -100)
	ctx.Input.SetMouseButton(MouseButtonLeft, true)
	dp.HandleDrag(ctx)

	// Position should be clamped to (0, 0)
	if dp.Position.X < 0 || dp.Position.Y < 0 {
		t.Errorf("Expected position clamped to >= 0, got (%f, %f)", dp.Position.X, dp.Position.Y)
	}

	// Try to drag off screen (to the right/bottom)
	ctx.Input.Reset()
	ctx.Input.SetMousePos(1000, 800)
	ctx.Input.SetMouseButton(MouseButtonLeft, true)
	dp.HandleDrag(ctx)

	// Position should be clamped to keep panel on screen
	maxX := ctx.DisplaySize.X - dp.Size.X
	maxY := ctx.DisplaySize.Y - dp.Size.Y
	if dp.Position.X > maxX || dp.Position.Y > maxY {
		t.Errorf("Expected position clamped to max (%f, %f), got (%f, %f)", maxX, maxY, dp.Position.X, dp.Position.Y)
	}
}

func TestDraggablePanel_EdgeSnapping(t *testing.T) {
	ctx := NewContext()
	ctx.SetStyle(DefaultStyle())
	ctx.Input = NewInputState()
	ctx.DisplaySize = Vec2{X: 800, Y: 600}

	dp := NewDraggablePanel(100, 100)
	dp.Size = Vec2{X: 200, Y: 150}
	dp.SnapConfig.Enabled = true
	dp.SnapConfig.EdgeMargin = 10

	// Start drag
	ctx.Input.SetMousePos(150, 110)
	ctx.Input.SetMouseButton(MouseButtonLeft, true)
	dp.HandleDrag(ctx)

	// Drag to position close to left edge (within margin)
	ctx.Input.Reset()
	ctx.Input.SetMousePos(55, 110) // Would put panel at X=5, within margin of 10
	ctx.Input.SetMouseButton(MouseButtonLeft, true)
	dp.HandleDrag(ctx)

	// Release to trigger snapping
	ctx.Input.Reset()
	ctx.Input.SetMousePos(55, 110)
	ctx.Input.SetMouseButton(MouseButtonLeft, false)
	dp.HandleDrag(ctx)

	// Should snap to X=0
	if dp.Position.X != 0 {
		t.Errorf("Expected X=0 after edge snapping, got %f", dp.Position.X)
	}
}

func TestDraggablePanel_GridSnapping(t *testing.T) {
	ctx := NewContext()
	ctx.SetStyle(DefaultStyle())
	ctx.Input = NewInputState()
	ctx.DisplaySize = Vec2{X: 800, Y: 600}

	dp := NewDraggablePanel(100, 100)
	dp.Size = Vec2{X: 200, Y: 150}
	dp.SnapConfig.Enabled = true
	dp.SnapConfig.EdgeMargin = 0 // Disable edge snapping
	dp.SnapConfig.GridSize = 20

	// Start drag
	ctx.Input.SetMousePos(150, 110)
	ctx.Input.SetMouseButton(MouseButtonLeft, true)
	dp.HandleDrag(ctx)

	// Drag to non-grid position
	ctx.Input.Reset()
	ctx.Input.SetMousePos(137, 117) // Would put panel at (87, 107)
	ctx.Input.SetMouseButton(MouseButtonLeft, true)
	dp.HandleDrag(ctx)

	// Release to trigger snapping
	ctx.Input.Reset()
	ctx.Input.SetMousePos(137, 117)
	ctx.Input.SetMouseButton(MouseButtonLeft, false)
	dp.HandleDrag(ctx)

	// Should snap to nearest grid (80, 100) or (100, 120) depending on rounding
	if int(dp.Position.X)%20 != 0 || int(dp.Position.Y)%20 != 0 {
		t.Errorf("Expected position on 20px grid, got (%f, %f)", dp.Position.X, dp.Position.Y)
	}
}

func TestDraggablePanel_DraggableDisabled(t *testing.T) {
	ctx := NewContext()
	ctx.SetStyle(DefaultStyle())
	ctx.Input = NewInputState()
	ctx.DisplaySize = Vec2{X: 800, Y: 600}

	dp := NewDraggablePanel(100, 100)
	dp.Size = Vec2{X: 200, Y: 150}
	dp.Draggable = false // Disable dragging

	originalPos := dp.Position

	// Try to drag
	ctx.Input.SetMousePos(150, 110)
	ctx.Input.SetMouseButton(MouseButtonLeft, true)
	dp.HandleDrag(ctx)

	if dp.IsDragging() {
		t.Error("Expected panel NOT to be dragging when Draggable=false")
	}

	// Position should not change
	if dp.Position != originalPos {
		t.Errorf("Expected position unchanged, got %v", dp.Position)
	}
}

func TestDraggablePanel_Constrain(t *testing.T) {
	displaySize := Vec2{X: 800, Y: 600}

	dp := NewDraggablePanel(-500, -100) // Way off screen to top-left
	dp.Size = Vec2{X: 200, Y: 150}

	dp.Constrain(displaySize)

	// Should be brought back on screen (at least minVisible px visible)
	minVisible := float32(50)
	if dp.Position.X < -dp.Size.X+minVisible {
		t.Errorf("Expected X >= %f, got %f", -dp.Size.X+minVisible, dp.Position.X)
	}
	if dp.Position.Y < 0 {
		t.Errorf("Expected Y >= 0, got %f", dp.Position.Y)
	}
}

func TestDragState_Reset(t *testing.T) {
	ds := DragState{
		Active:    true,
		StartX:    100,
		StartY:    200,
		OffsetX:   10,
		OffsetY:   20,
		PanelName: "test",
	}

	ds.Reset()

	if ds.Active || ds.StartX != 0 || ds.StartY != 0 || ds.OffsetX != 0 || ds.OffsetY != 0 || ds.PanelName != "" {
		t.Error("DragState.Reset() did not clear all fields")
	}
}
