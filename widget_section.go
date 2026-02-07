package gui

// sectionStore is the type-safe store for section state.
// Uses the new FrameStore pattern instead of the old GetState/SetState.
var sectionStore = NewFrameStore[SectionState]()

// Section creates a collapsible section that can contain any widgets, including nested sections.
// Returns a function that should be called with the content closure.
//
// The Section widget provides:
// - Click to expand/collapse
// - Arrow indicator (► collapsed, ▼ expanded)
// - Auto-indentation of content
// - Keyboard focus support (cyan highlight when focused)
// - Auto-scroll to focused section via ctx.ScrollTo()
//
// Usage:
//
//	ctx.Section("Settings")(func() {
//	    ctx.SliderFloat("Volume", &vol, 0, 1)
//	    ctx.Section("Advanced")(func() {  // Nesting supported
//	        ctx.Text("Nested content")
//	    })
//	})
//
// With options:
//
//	ctx.Section("Graphics", DefaultOpen(), Focused())(func() {
//	    ctx.Text("Content")
//	})
func (ctx *Context) Section(label string, opts ...Option) func(func()) {
	return func(contents func()) {
		if !ctx.BeginSection(label, opts...) {
			return
		}
		contents()
		ctx.EndSection()
	}
}

// SectionState holds the state needed for section expansion.
// This extends CollapsingHeaderState with section-specific tracking.
type SectionState struct {
	Open   bool // Whether the section is expanded
	indent float32
}

// BeginSection starts a collapsible section.
// Returns true if the section is expanded and content should be drawn.
// Must call EndSection() after content if this returns true.
//
// This is the manual-control API for cases where the closure pattern doesn't fit:
//
//	if ctx.BeginSection("Settings") {
//	    ctx.Text("Content")
//	    ctx.EndSection()
//	}
func (ctx *Context) BeginSection(label string, opts ...Option) bool {
	o := applyOptions(opts)

	// Generate ID
	id := ctx.GetID(label)
	if optID := GetOpt(o, OptID); optID != "" {
		id = ctx.GetID(optID)
	}

	// Get stored state - default to closed unless DefaultOpen() is specified
	defaultOpen := GetOpt(o, OptDefaultOpen)
	state := sectionStore.Get(id, SectionState{Open: defaultOpen})

	// Check for controlled mode (external state via pointer)
	openOpt := GetOpt(o, OptOpen)
	controlledMode := openOpt.Ptr != nil
	if controlledMode {
		// Controlled mode: read state from external pointer
		state.Open = *openOpt.Ptr
	}

	// Get position and dimensions
	pos := ctx.ItemPos()
	w := ctx.currentLayoutWidth()
	h := ctx.lineHeight()

	// Check if explicitly focused via Focused() option
	focused := GetOpt(o, OptFocused)

	// Note: auto-scroll for registry focus is handled by Scrollable via EndFocusScope()
	// Only manually trigger ScrollTo for explicitly focused sections
	if focused {
		ctx.ScrollTo(pos.Y, h)
	}

	// Interaction rect
	rect := Rect{X: pos.X, Y: pos.Y, W: w, H: h}
	hovered := ctx.isHovered(id, rect)

	// Register as focusable in the registry (auto-draws debug rect if focused)
	ctx.RegisterFocusable(id, label, rect, FocusTypeSection)

	// Begin focus scope for hierarchical focus tracking
	ctx.BeginFocusScope(id, label, FocusTypeSection, rect)

	// Check both explicit Focused() option AND registry focus (from arrow key navigation)
	registryFocused := ctx.IsRegistryFocused(id)
	isFocused := focused || registryFocused

	// Determine background color
	bgColor := ctx.style.ButtonColor
	if isFocused {
		bgColor = ctx.style.ButtonActiveColor
	} else if hovered {
		bgColor = ctx.style.ButtonHoveredColor
	}
	ctx.DrawList.AddRect(pos.X, pos.Y, w, h, bgColor)

	// Draw debug focus rect for sections when registry-focused (FocusTypeSection skips auto-draw)
	if registryFocused && ctx.DebugFocusHighlight {
		ctx.DrawDebugFocusRect(pos.X, pos.Y, w, h)
	}
	// Also draw for explicit Focused() option when not already registry-focused
	if focused && !registryFocused {
		ctx.DrawDebugFocusRectIf(focused, pos.X, pos.Y, w, h)
	}

	// Draw arrow indicator
	arrow := "►"
	if state.Open {
		arrow = "▼"
	}
	arrowColor := ctx.style.TextColor
	if isFocused {
		arrowColor = ColorCyan
	}
	ctx.addText(pos.X+2, pos.Y, arrow, arrowColor)

	// Draw label
	arrowWidth := ctx.MeasureText(arrow).X
	ctx.addText(pos.X+arrowWidth+4, pos.Y, label, ctx.style.TextColor)

	// Handle click to toggle
	if ctx.isClicked(id, rect) {
		state.Open = !state.Open

		// In controlled mode, write back to external pointer
		if controlledMode {
			*openOpt.Ptr = state.Open
		}
	}

	ctx.advanceCursor(Vec2{X: w, Y: h})

	if !state.Open {
		ctx.EndFocusScope() // Close focus scope even when collapsed
		return false
	}

	// Calculate indent
	indent := ctx.style.ItemSpacing * 2
	if customIndent := GetOpt(o, OptIndentSize); customIndent > 0 {
		indent = customIndent
	}
	if GetOpt(o, OptNoIndent) {
		indent = 0
	}

	// Store indent for EndSection
	state.indent = indent

	// Push to section stack for EndSection to pop
	ctx.sectionStack = append(ctx.sectionStack, indent)

	if indent > 0 {
		ctx.Indent(indent)
	}

	return true
}

// EndSection ends a section started with BeginSection.
// Must be called after BeginSection returns true.
func (ctx *Context) EndSection() {
	// End focus scope and get child focus info
	info := ctx.EndFocusScope()

	// If a child was focused, we could use info.FocusedChildY for auto-scroll
	// This is handled automatically by the Scrollable widget
	_ = info

	// Pop indent from section stack
	if n := len(ctx.sectionStack); n > 0 {
		indent := ctx.sectionStack[n-1]
		ctx.sectionStack = ctx.sectionStack[:n-1]
		if indent > 0 {
			ctx.Unindent(indent)
		}
	}
}

// ToggleSectionState toggles the open/closed state of a section.
// Use this for external control (e.g., keyboard shortcuts).
//
// Usage:
//
//	sectionID := ctx.GetID("my_section")
//	gui.ToggleSectionState(ctx, sectionID)
func ToggleSectionState(ctx *Context, id ID) {
	state := sectionStore.Get(id, SectionState{Open: false})
	state.Open = !state.Open
}

// SetSectionOpen sets the open/closed state of a section.
// Use this for external control (e.g., keyboard shortcuts).
func SetSectionOpen(ctx *Context, id ID, open bool) {
	state := sectionStore.Get(id, SectionState{Open: false})
	state.Open = open
}

// IsSectionOpen returns whether a section is currently open.
func IsSectionOpen(ctx *Context, id ID) bool {
	state := sectionStore.GetIfExists(id)
	if state == nil {
		return false
	}
	return state.Open
}

// GetSectionState returns a pointer to the section's state for advanced manipulation.
// Returns nil if the section hasn't been rendered yet this frame.
func GetSectionState(id ID) *SectionState {
	return sectionStore.GetIfExists(id)
}
