package gui_test

import (
	"testing"

	"github.com/go-theft-auto/gui"
)

// Helper to create a test GUI context
func setupScrollableTest() (*gui.GUI, *gui.InputState) {
	renderer := &mockRenderer{}
	ui := gui.New(renderer, gui.WithStyle(gui.GTAStyle()))
	input := gui.NewInputState()
	return ui, input
}

// getScrollableState is a helper to get scrollable state in tests.
// Uses the new FrameStore-based state system.
func getScrollableState(ctx *gui.Context, id string) *gui.ScrollableState {
	return gui.GetScrollableState(ctx, id)
}

// getScrollableID returns the ID that Scrollable uses internally.
// IMPORTANT: This must be called as the FIRST GetID call in a frame to match
// what Scrollable would use. Do NOT call this in the same frame as Scrollable.
func getScrollableID(ctx *gui.Context, id string) gui.ID {
	return ctx.GetID(id + "_scrollable")
}

func TestScrollableBasic(t *testing.T) {
	ui, input := setupScrollableTest()
	displaySize := gui.Vec2{X: 800, Y: 600}

	ctx := ui.Begin(input, displaySize, 0.016)

	// Create a scrollable with content taller than viewport
	ctx.Scrollable("test_scroll", 100)(func() {
		for i := 0; i < 20; i++ {
			ctx.Text("Line")
		}
	})

	err := ui.End()
	if err != nil {
		t.Fatalf("End() returned error: %v", err)
	}
}

func TestScrollableMouseWheelScrolls(t *testing.T) {
	ui, input := setupScrollableTest()
	displaySize := gui.Vec2{X: 800, Y: 600}

	// First frame - create scrollable with tall content
	ctx := ui.Begin(input, displaySize, 0.016)
	ctx.Scrollable("test_scroll", 100)(func() {
		for i := 0; i < 50; i++ {
			ctx.Text("Line")
		}
	})
	_ = ui.End()

	// Get the state after the first frame (state is now stored in FrameStore)
	state := getScrollableState(ctx, "test_scroll")
	if state == nil {
		t.Fatal("scrollable state should exist after rendering")
	}
	if state.ScrollY != 0 {
		t.Errorf("initial scroll should be 0, got %v", state.ScrollY)
	}
	initialScrollY := state.ScrollY

	// Scroll frame - mouse wheel scroll
	input.Reset()
	input.SetMousePos(50, 50) // Inside scrollable viewport (starts at 0,0)
	input.MouseWheelY = -3    // Scroll down (negative = scroll content up = scrollY increases)

	ctx = ui.Begin(input, displaySize, 0.016)
	ctx.Scrollable("test_scroll", 100)(func() {
		for i := 0; i < 50; i++ {
			ctx.Text("Line")
		}
	})
	_ = ui.End()

	// Get state after scrolling
	state = getScrollableState(ctx, "test_scroll")
	if state == nil {
		t.Fatal("scrollable state should exist")
	}
	if state.ScrollY <= initialScrollY {
		t.Errorf("mouse wheel should increase scroll position, got %v -> %v", initialScrollY, state.ScrollY)
	}
}

func TestScrollableUserScrollResetsTimer(t *testing.T) {
	ui, input := setupScrollableTest()
	displaySize := gui.Vec2{X: 800, Y: 600}

	// First frame - scroll with mouse wheel
	input.SetMousePos(50, 50)
	input.MouseWheelY = -3

	ctx := ui.Begin(input, displaySize, 0.016)
	ctx.Scrollable("test_scroll", 100)(func() {
		for i := 0; i < 50; i++ {
			ctx.Text("Line")
		}
	})
	_ = ui.End()

	// Get state after scrolling
	state := getScrollableState(ctx, "test_scroll")
	if state == nil {
		t.Fatal("scrollable state should exist")
	}
	// After user scroll, UserScrollTime should be 0 (timer reset)
	if state.UserScrollTime != 0 {
		t.Errorf("UserScrollTime should be 0 after user scroll, got %v", state.UserScrollTime)
	}
}

func TestScrollableCooldownTimerIncreases(t *testing.T) {
	ui, input := setupScrollableTest()
	displaySize := gui.Vec2{X: 800, Y: 600}

	// First frame - scroll with mouse wheel
	input.SetMousePos(50, 50)
	input.MouseWheelY = -3

	ctx := ui.Begin(input, displaySize, 0.016)
	ctx.Scrollable("test_scroll", 100)(func() {
		for i := 0; i < 50; i++ {
			ctx.Text("Line")
		}
	})
	_ = ui.End()

	// Second frame - no scroll, time passes
	input.Reset()
	input.SetMousePos(50, 50) // Stay hovered but don't scroll

	ctx = ui.Begin(input, displaySize, 0.1) // 100ms delta time
	ctx.Scrollable("test_scroll", 100)(func() {
		for i := 0; i < 50; i++ {
			ctx.Text("Line")
		}
	})
	_ = ui.End()

	// Get state after second frame
	state := getScrollableState(ctx, "test_scroll")
	if state == nil {
		t.Fatal("scrollable state should exist")
	}
	if state.UserScrollTime < 0.09 { // Allow small float tolerance
		t.Errorf("UserScrollTime should have increased to ~0.1, got %v", state.UserScrollTime)
	}
}

func TestScrollableFocusOnClick(t *testing.T) {
	// Scrollable is no longer focusable - it uses hover-based keyboard scrolling
	// This test verifies that clicking on scrollable doesn't break anything
	ui, input := setupScrollableTest()
	displaySize := gui.Vec2{X: 800, Y: 600}

	// First frame - render scrollable
	ctx := ui.Begin(input, displaySize, 0.016)
	ctx.Scrollable("test_scroll", 100)(func() {
		for i := 0; i < 50; i++ {
			ctx.Text("Line")
		}
	})
	_ = ui.End()

	// Second frame - click on scrollable area
	input.Reset()
	input.SetMousePos(50, 50) // Inside scrollable viewport
	input.SetMouseButton(gui.MouseButtonLeft, true)

	ctx = ui.Begin(input, displaySize, 0.016)
	ctx.Scrollable("test_scroll", 100)(func() {
		for i := 0; i < 50; i++ {
			ctx.Text("Line")
		}
	})
	_ = ui.End()

	// Scrollable should NOT be registry-focused (uses hover for keyboard scrolling)
	scrollID := getScrollableID(ctx, "test_scroll")
	if ctx.IsRegistryFocused(scrollID) {
		t.Error("scrollable should NOT be focusable (uses hover-based keyboard scrolling)")
	}
}

func TestScrollableKeyboardScrollingWhenHovered(t *testing.T) {
	ui, input := setupScrollableTest()
	displaySize := gui.Vec2{X: 800, Y: 600}

	// First frame - render scrollable with mouse hovering over it
	input.SetMousePos(50, 50) // Inside scrollable viewport

	ctx := ui.Begin(input, displaySize, 0.016)
	ctx.Scrollable("test_scroll", 100)(func() {
		for i := 0; i < 100; i++ { // Tall content
			ctx.Text("Line")
		}
	})
	_ = ui.End()

	// Get initial state
	initialState := getScrollableState(ctx, "test_scroll")
	if initialState == nil {
		t.Fatal("initial state should exist")
	}
	initialScrollY := initialState.ScrollY

	// PageDown frame - keyboard scrolling works when hovered (no focus needed)
	input.Reset()
	input.SetMousePos(50, 50) // Keep hovering
	input.SetKey(gui.KeyPageDown, true)

	ctx = ui.Begin(input, displaySize, 0.016)
	ctx.Scrollable("test_scroll", 100)(func() {
		for i := 0; i < 100; i++ {
			ctx.Text("Line")
		}
	})
	_ = ui.End()

	// Get state after PageDown
	newState := getScrollableState(ctx, "test_scroll")
	if newState == nil {
		t.Fatal("new state should exist")
	}
	if newState.ScrollY <= initialScrollY {
		t.Errorf("PageDown should increase scroll when hovered, got %v -> %v", initialScrollY, newState.ScrollY)
	}
}

func TestScrollableHomeEndKeys(t *testing.T) {
	ui, input := setupScrollableTest()
	displaySize := gui.Vec2{X: 800, Y: 600}

	// First frame - render scrollable with mouse hovering
	input.SetMousePos(50, 50) // Inside scrollable viewport

	ctx := ui.Begin(input, displaySize, 0.016)
	ctx.Scrollable("test_scroll", 100)(func() {
		for i := 0; i < 100; i++ { // Tall content
			ctx.Text("Line")
		}
	})
	_ = ui.End()

	// End key frame - keyboard scrolling works when hovered
	input.Reset()
	input.SetMousePos(50, 50) // Keep hovering
	input.SetKey(gui.KeyEnd, true)

	ctx = ui.Begin(input, displaySize, 0.016)
	ctx.Scrollable("test_scroll", 100)(func() {
		for i := 0; i < 100; i++ {
			ctx.Text("Line")
		}
	})
	_ = ui.End()

	// Get state after End
	stateAfterEnd := getScrollableState(ctx, "test_scroll")
	if stateAfterEnd == nil {
		t.Fatal("state should exist after End")
	}
	if stateAfterEnd.ScrollY == 0 {
		t.Error("End key should scroll to bottom when hovered")
	}

	maxScroll := stateAfterEnd.ScrollY

	// Home key frame
	input.Reset()
	input.SetMousePos(50, 50) // Keep hovering
	input.SetKey(gui.KeyHome, true)

	ctx = ui.Begin(input, displaySize, 0.016)
	ctx.Scrollable("test_scroll", 100)(func() {
		for i := 0; i < 100; i++ {
			ctx.Text("Line")
		}
	})
	_ = ui.End()

	// Get state after Home
	stateAfterHome := getScrollableState(ctx, "test_scroll")
	if stateAfterHome == nil {
		t.Fatal("state should exist after Home")
	}
	if stateAfterHome.ScrollY != 0 {
		t.Errorf("Home key should scroll to top (0), got %v", stateAfterHome.ScrollY)
	}

	if maxScroll == 0 {
		t.Log("Warning: End key didn't scroll, content may not exceed viewport")
	}
}

func TestScrollableStateFields(t *testing.T) {
	// Test that ScrollableState has required fields with correct defaults
	state := gui.ScrollableState{}

	if state.UserScrolledThisFrame != false {
		t.Error("UserScrolledThisFrame should default to false")
	}
	// Note: UserScrollTime defaults to 0 in struct, but Scrollable widget
	// initializes it to 1.0 so auto-scroll works immediately for new scrollables
	if state.UserScrollTime != 0 {
		t.Error("UserScrollTime struct default should be 0")
	}
	if state.LastFocusY != 0 {
		t.Error("LastFocusY should default to 0")
	}
	if state.FocusYSet != false {
		t.Error("FocusYSet should default to false")
	}
}

func TestScrollableScrollToTriggersAutoScroll(t *testing.T) {
	ui, input := setupScrollableTest()
	displaySize := gui.Vec2{X: 800, Y: 600}

	// Run several frames to expire cooldown timer and register focusable items
	var ctx *gui.Context
	for i := 0; i < 25; i++ {
		input.Reset()
		ctx = ui.Begin(input, displaySize, 0.02) // 20ms per frame = 500ms total
		ctx.Scrollable("test_scroll", 100)(func() {
			for j := 0; j < 50; j++ {
				// Use buttons so we have focusable items for keyboard navigation
				ctx.Button("Button")
			}
		})
		_ = ui.End()
	}

	// Frame with ScrollTo AND keyboard navigation (required for auto-scroll)
	// The keyboard-only auto-scroll feature requires focusable items and keyboard navigation
	input.Reset()
	input.SetKey(gui.KeyDown, true) // Simulate keyboard navigation
	ctx = ui.Begin(input, displaySize, 0.016)
	ctx.NavigateFocus(gui.NavDown) // Trigger keyboard navigation flag (needs focusable items)
	ctx.Scrollable("test_scroll", 100)(func() {
		for i := 0; i < 50; i++ {
			if i == 40 { // Request focus near bottom
				ctx.ScrollTo(ctx.GetCursorPos().Y, 20)
			}
			ctx.Button("Button")
		}
	})
	_ = ui.End()

	// Get state after ScrollTo
	state := getScrollableState(ctx, "test_scroll")
	if state == nil {
		t.Fatal("state should exist")
	}
	if !state.FocusYSet {
		t.Error("ScrollTo should have set FocusYSet (requires keyboard navigation with focusable items)")
	}
}

func TestEnsureScrollVisible(t *testing.T) {
	ui, input := setupScrollableTest()
	displaySize := gui.Vec2{X: 800, Y: 600}

	// Use unique ID to avoid state pollution from other tests
	scrollID := "ensure_visible_scroll"

	// Create a scrollable with content
	ctx := ui.Begin(input, displaySize, 0.016)
	ctx.Scrollable(scrollID, 100)(func() {
		for i := 0; i < 50; i++ {
			ctx.Text("Line")
		}
	})
	_ = ui.End()

	// Get initial state - should be 0 for a fresh scrollable
	initialState := getScrollableState(ctx, scrollID)
	if initialState == nil {
		t.Fatal("initial state should exist")
	}

	// Frame 2: Call EnsureScrollVisible while also rendering the Scrollable
	// (EnsureScrollVisible needs the Scrollable to be rendered in the same frame
	// because state cleanup happens per-frame)
	ctx = ui.Begin(input, displaySize, 0.016)
	ctx.Scrollable(scrollID, 100)(func() {
		for i := 0; i < 50; i++ {
			ctx.Text("Line")
		}
	})
	// Request scroll to Y=500 which is well below the viewport (100px)
	gui.EnsureScrollVisible(ctx, scrollID, 500, 100, 20)
	_ = ui.End()

	// Get state after EnsureScrollVisible
	state := getScrollableState(ctx, scrollID)
	if state == nil {
		t.Fatal("state should exist")
	}

	// Should have scrolled to make target visible
	// Target is at Y=500, viewport is 100px, padding is 20px
	// Expected: scroll to approximately 500 - 100 + 20 = 420
	if state.ScrollY < 400 {
		t.Errorf("EnsureScrollVisible should have scrolled to ~420, got %v", state.ScrollY)
	}
}

func TestScrollableContentMeasurement(t *testing.T) {
	ui, input := setupScrollableTest()
	displaySize := gui.Vec2{X: 800, Y: 600}

	ctx := ui.Begin(input, displaySize, 0.016)
	ctx.Scrollable("test_scroll", 100)(func() {
		for i := 0; i < 50; i++ { // 50 lines should exceed 100px height
			ctx.Text("Line")
		}
	})
	_ = ui.End()

	// Get state after rendering
	state := getScrollableState(ctx, "test_scroll")
	if state == nil {
		t.Fatal("state should exist")
	}

	// Content height should be measured and greater than viewport height
	if state.ContentHeight <= 100 {
		t.Errorf("ContentHeight should exceed viewport (100), got %v", state.ContentHeight)
	}
}

// Benchmark for scrollable rendering performance
func BenchmarkScrollable(b *testing.B) {
	ui, input := setupScrollableTest()
	displaySize := gui.Vec2{X: 800, Y: 600}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := ui.Begin(input, displaySize, 0.016)
		ctx.Scrollable("bench_scroll", 200)(func() {
			for j := 0; j < 100; j++ {
				ctx.Text("Benchmark line")
			}
		})
		_ = ui.End()
	}
}

// Benchmark for scrollable with ScrollTo calls
func BenchmarkScrollableWithScrollTo(b *testing.B) {
	ui, input := setupScrollableTest()
	displaySize := gui.Vec2{X: 800, Y: 600}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := ui.Begin(input, displaySize, 0.016)
		ctx.Scrollable("bench_scroll", 200)(func() {
			for j := 0; j < 100; j++ {
				if j == 50 {
					ctx.ScrollTo(ctx.GetCursorPos().Y, 20)
				}
				ctx.Text("Benchmark line")
			}
		})
		_ = ui.End()
	}
}
