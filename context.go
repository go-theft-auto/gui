package gui

import (
	"log/slog"
	"os"
)

// guiLogger is the logger for GUI context debugging.
// Uses the shared guiLogLevel from focus_registry.go
var guiLogger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: guiLogLevel}))

// Context holds all state for UI rendering in a single frame.
// This is NOT context.Context - it's a dedicated GUI context type.
// Using a dedicated type avoids type assertions and map lookups,
// providing better performance and type safety.
type Context struct {
	// Drawing output
	DrawList           *DrawList
	ForegroundDrawList *DrawList // For popups, dropdowns, tooltips (drawn on top)

	// Styling
	style      Style
	styleStack []Style // For PushStyle/PopStyle

	// Layout
	cursor      Vec2
	layoutStack []*Layout

	// Input (read-only during frame)
	Input *InputState

	// Widget state (persisted between frames)
	stateStore StateStore

	// IDs
	idStack   []ID
	idCounter uint32 // Auto-increment for call-site IDs

	// Screen
	DisplaySize Vec2
	DPIScale    float32

	// Frame info
	FrameCount uint64
	DeltaTime  float32

	// Focus/Active/Hover tracking
	focusedID ID // Widget with keyboard focus
	activeID  ID // Widget being interacted with (e.g., pressed button)
	hoveredID ID // Widget under mouse cursor

	// Keyboard/mouse tracking for this frame
	hotID ID // Widget that will become active on next click

	// Two-pass layout support (for centering, etc.)
	measuringPass bool
	measuredSizes map[ID]Vec2

	// Font texture ID (set by renderer) - legacy field for built-in font
	FontTextureID uint32

	// FontProvider for advanced font support (optional, interface-based)
	fontProvider FontProvider

	// Input capture flags (output from GUI to application)
	// These tell the application whether GUI wants to consume input.
	WantCaptureMouse    bool // True if mouse is over any GUI element
	WantCaptureKeyboard bool // True if a text input has focus

	// Panel focus tracking (for Ctrl+Tab cycling)
	// These are set by the panel registry each frame.
	panelRegistry *PanelRegistry

	// Drag state tracking (for draggable panels)
	// Only one panel can be dragged at a time.
	activeDragPanel *DraggablePanel

	// Performance optimization: pre-allocated glyph buffer for text rendering.
	// Reused between addText() calls to avoid per-call allocations.
	glyphBuffer []GlyphQuad

	// Performance optimization: text measurement cache.
	// Avoids redundant MeasureText calls for the same text within a frame.
	// Key format: "text\x00scale" to differentiate scales.
	textMeasureCache map[string]Vec2

	// Scroll focus tracking - widgets can register their focus Y position
	// and parent Scrollable will auto-scroll to keep it visible.
	scrollFocusY   float32 // Current focus Y position (relative to scroll content)
	scrollFocusSet bool    // True if a widget set focus this frame
	scrollFocusPad float32 // Padding around focus target

	// Scrollable stack - tracks nested scrollables so widgets can call ctx.ScrollTo()
	// without needing to know their parent scrollable's ID
	scrollableStack []*scrollableContext

	// Section stack - tracks indent depths for BeginSection/EndSection API
	sectionStack []float32

	// Hierarchical focus tracking (new system, coexists with focusedID)
	// Enables parent widgets to know which child has focus and where.
	focusPath  *FocusPath  // Active path from root to focused leaf
	focusStack []FocusNode // Built during frame traversal

	// Child focus reporting - set by focused children for parent containers
	childFocusY      float32 // Y position of focused child (for auto-scroll)
	childFocusHeight float32 // Height of focused child
	childFocusSet    bool    // True if a child reported focus this frame

	// Focus registry - tracks all focusable widgets per frame
	// Implements the Focusable interface system for immediate-mode GUI
	focusRegistry *FocusRegistry

	// Active popup tracking - persists across frames for input handling
	// When a popup (dropdown, menu) is open, navigation should stay within it
	activePopupID ID

	// Debug visualization
	DebugFocusHighlight bool // When true, draw red overlays on all focused elements
}

// NewContext creates a new GUI context with default settings.
func NewContext() *Context {
	return &Context{
		styleStack:          make([]Style, 0, 8),
		layoutStack:         make([]*Layout, 0, 16),
		idStack:             make([]ID, 0, 32),
		measuredSizes:       make(map[ID]Vec2),
		glyphBuffer:         make([]GlyphQuad, 0, 256), // Pre-allocate for typical text
		textMeasureCache:    make(map[string]Vec2, 64), // Cache for text measurements
		focusPath:           NewFocusPath(),            // Hierarchical focus tracking
		focusStack:          make([]FocusNode, 0, 8),   // Focus scope stack
		focusRegistry:       NewFocusRegistry(),        // Focusable widget registry
		DPIScale:            1.0,
		DebugFocusHighlight: true, // Debug: highlight focused elements in red (F10 to toggle)
	}
}

// Style returns the current style.
func (ctx *Context) Style() Style {
	return ctx.style
}

// SetStyle sets the base style.
func (ctx *Context) SetStyle(style Style) {
	ctx.style = style
}

// PushStyle temporarily overrides the style.
func (ctx *Context) PushStyle(style Style) {
	ctx.styleStack = append(ctx.styleStack, ctx.style)
	ctx.style = style
}

// PopStyle restores the previous style.
func (ctx *Context) PopStyle() {
	n := len(ctx.styleStack)
	if n > 0 {
		ctx.style = ctx.styleStack[n-1]
		ctx.styleStack = ctx.styleStack[:n-1]
	}
}

// PushStyleColor temporarily overrides a single color.
func (ctx *Context) PushStyleColor(field StyleColorField, color uint32) {
	ctx.PushStyle(ctx.style)
	switch field {
	case StyleColorText:
		ctx.style.TextColor = color
	case StyleColorButton:
		ctx.style.ButtonColor = color
	case StyleColorButtonHovered:
		ctx.style.ButtonHoveredColor = color
	case StyleColorButtonActive:
		ctx.style.ButtonActiveColor = color
	case StyleColorPanel:
		ctx.style.PanelColor = color
	case StyleColorSelected:
		ctx.style.SelectedBgColor = color
	}
}

// StyleColorField identifies a color field in Style for PushStyleColor.
type StyleColorField int

const (
	StyleColorText StyleColorField = iota
	StyleColorButton
	StyleColorButtonHovered
	StyleColorButtonActive
	StyleColorPanel
	StyleColorSelected
)

// Reset prepares the context for a new frame.
func (ctx *Context) Reset(displaySize Vec2, deltaTime float32) {
	// Advance frame counter and clean up stale FrameStore entries
	NextFrame()

	ctx.cursor = Vec2{0, 0}
	ctx.layoutStack = ctx.layoutStack[:0]
	ctx.styleStack = ctx.styleStack[:0]
	ctx.idStack = ctx.idStack[:0]
	ctx.idCounter = 0
	ctx.DisplaySize = displaySize
	ctx.DeltaTime = deltaTime
	// Note: FrameCount is incremented in GUI.PrepareInputHandling() at the START
	// of the frame, not here. This ensures the same frame number is used for both
	// input handling and rendering phases.

	// Clear previous frame's hot/active state that wasn't renewed
	ctx.hotID = 0

	// Reset input capture flags - widgets will set these during the frame
	ctx.WantCaptureMouse = false
	ctx.WantCaptureKeyboard = false

	// Clear text measurement cache (valid only for current frame)
	clear(ctx.textMeasureCache)

	// Clear scroll focus (widgets will set it fresh each frame)
	ctx.scrollFocusSet = false

	// Clear focus stack for new frame (focusPath persists for continuity)
	ctx.focusStack = ctx.focusStack[:0]
	ctx.childFocusSet = false

	// Reset focus registry for new frame (uses frame count to prevent double-reset)
	if ctx.focusRegistry != nil {
		ctx.focusRegistry.ResetForFrame(ctx.FrameCount)
	}

	// Clear activePopupID - widgets with active popups must reclaim it each frame.
	// This happens AFTER HandleInput (which already used the previous value),
	// so if a popup is orphaned (its widget no longer draws), navigation becomes unblocked.
	if ctx.activePopupID != 0 {
		guiLogger.Debug("Reset: clearing activePopupID", "id", ctx.activePopupID)
	}
	ctx.activePopupID = 0
}

// Helper methods for widget interaction

// isHovered returns true if the widget area is under the mouse cursor.
func (ctx *Context) isHovered(id ID, rect Rect) bool {
	if ctx.Input == nil {
		return false
	}
	return rect.Contains(Vec2{ctx.Input.MouseX, ctx.Input.MouseY})
}

// IsHovered returns true if the widget area is under the mouse cursor (public API).
func (ctx *Context) IsHovered(id ID, rect Rect) bool {
	return ctx.isHovered(id, rect)
}

// isClicked returns true if the widget was clicked this frame.
func (ctx *Context) isClicked(id ID, rect Rect) bool {
	if ctx.Input == nil {
		return false
	}
	hovered := ctx.isHovered(id, rect)
	clicked := ctx.Input.MouseClicked(MouseButtonLeft)

	// Debug logging for click detection issues
	if clicked && guiVerbose() {
		if hovered {
			guiLogger.Debug("click detected",
				"id", id,
				"rect", rect,
				"mouse", Vec2{ctx.Input.MouseX, ctx.Input.MouseY})
		} else {
			guiLogger.Debug("click missed - not hovered",
				"id", id,
				"rect", rect,
				"mouse", Vec2{ctx.Input.MouseX, ctx.Input.MouseY})
		}
	}

	return hovered && clicked
}

// IsClicked returns true if the widget was clicked this frame (public API).
func (ctx *Context) IsClicked(id ID, rect Rect) bool {
	return ctx.isClicked(id, rect)
}

// isPressed returns true if the widget is being held down.
func (ctx *Context) isPressed(id ID, rect Rect) bool {
	if ctx.Input == nil {
		return false
	}
	return ctx.isHovered(id, rect) && ctx.Input.MouseDown(MouseButtonLeft)
}

// SetFocused sets the focused widget.
func (ctx *Context) SetFocused(id ID) {
	ctx.focusedID = id
}

// IsFocused returns true if the widget has keyboard focus.
func (ctx *Context) IsFocused(id ID) bool {
	return ctx.focusedID == id
}

// ClearFocus removes keyboard focus.
func (ctx *Context) ClearFocus() {
	ctx.focusedID = 0
}

// HasWidgetFocus returns true if any widget has keyboard focus (edit mode).
// This is different from registry focus (navigation focus).
func (ctx *Context) HasWidgetFocus() bool {
	return ctx.focusedID != 0
}

// SetActivePopup marks a popup (dropdown, menu) as open.
// While a popup is active, focus navigation should stay within it.
// Call with id=0 to close the popup.
func (ctx *Context) SetActivePopup(id ID) {
	ctx.activePopupID = id
	if id != 0 {
		ctx.WantCaptureKeyboard = true
	}
}

// HasActivePopup returns true if a popup is currently open.
func (ctx *Context) HasActivePopup() bool {
	return ctx.activePopupID != 0
}

// ActivePopupID returns the ID of the currently active popup, or 0 if none.
func (ctx *Context) ActivePopupID() ID {
	return ctx.activePopupID
}

// SetCursorPos sets the cursor position for the next widget.
func (ctx *Context) SetCursorPos(x, y float32) {
	ctx.cursor = Vec2{X: x, Y: y}
}

// GetCursorPos returns the current cursor position.
func (ctx *Context) GetCursorPos() Vec2 {
	return ctx.cursor
}

// lineHeight returns the height of a single line of text.
// Uses the font provider if available, otherwise falls back to CharHeight * FontScale.
func (ctx *Context) lineHeight() float32 {
	if f := ctx.activeFont(); f != nil {
		return f.LineHeight(ctx.style.FontScale)
	}
	return ctx.style.CharHeight * ctx.style.FontScale
}

// LineHeight returns the height of a single line of text (public API).
func (ctx *Context) LineHeight() float32 {
	return ctx.lineHeight()
}

// MeasureText returns the size of rendered text.
// Uses the font provider if available, otherwise falls back to monospace calculation.
// Results are cached per-frame to avoid redundant measurements.
func (ctx *Context) MeasureText(text string) Vec2 {
	// Check cache first (includes scale in key for differentiation)
	if ctx.textMeasureCache != nil {
		if cached, ok := ctx.textMeasureCache[text]; ok {
			return cached
		}
	}

	var result Vec2
	if f := ctx.activeFont(); f != nil {
		size := f.MeasureText(text, ctx.style.FontScale)
		result = Vec2{X: size.X, Y: size.Y}
	} else {
		// Fallback to monospace calculation
		charW := ctx.style.CharWidth * ctx.style.FontScale
		charH := ctx.style.CharHeight * ctx.style.FontScale
		result = Vec2{X: float32(len(text)) * charW, Y: charH}
	}

	// Cache the result
	if ctx.textMeasureCache != nil {
		ctx.textMeasureCache[text] = result
	}

	return result
}

// activeFont returns the current active font, or nil if no font provider is set.
// This is a helper to reduce repetitive null checks.
func (ctx *Context) activeFont() Font {
	if ctx.fontProvider != nil {
		return ctx.fontProvider.ActiveFont()
	}
	return nil
}

// SetFontProvider sets the font provider for advanced font support.
// The provider must implement the FontProvider interface.
// Pass nil to disable font provider and use built-in monospace font.
func (ctx *Context) SetFontProvider(fp FontProvider) {
	ctx.fontProvider = fp
}

// SetPanelRegistry associates a panel registry with this context.
// This enables panel focus tracking for Ctrl+Tab cycling.
func (ctx *Context) SetPanelRegistry(registry *PanelRegistry) {
	ctx.panelRegistry = registry
}

// PanelRegistry returns the associated panel registry, or nil if not set.
func (ctx *Context) PanelRegistry() *PanelRegistry {
	return ctx.panelRegistry
}

// IsPanelFocused returns true if the given panel is currently focused.
// Returns false if no panel registry is set.
func (ctx *Context) IsPanelFocused(panel Panel) bool {
	if ctx.panelRegistry == nil {
		return false
	}
	return ctx.panelRegistry.IsPanelFocused(panel)
}

// IsFocusVisible returns true if focus indicator rings should be drawn.
// Returns false if no panel registry is set or focus is not active.
func (ctx *Context) IsFocusVisible() bool {
	if ctx.panelRegistry == nil {
		return false
	}
	return ctx.panelRegistry.FocusManager().IsFocusVisible()
}

// SetActiveDragPanel sets the panel currently being dragged.
// Only one panel can be dragged at a time.
func (ctx *Context) SetActiveDragPanel(dp *DraggablePanel) {
	ctx.activeDragPanel = dp
}

// ActiveDragPanel returns the panel currently being dragged, or nil.
func (ctx *Context) ActiveDragPanel() *DraggablePanel {
	return ctx.activeDragPanel
}

// IsDraggingPanel returns true if any panel is currently being dragged.
func (ctx *Context) IsDraggingPanel() bool {
	return ctx.activeDragPanel != nil && ctx.activeDragPanel.IsDragging()
}

// SetScrollFocus registers a Y position that should be kept visible by parent Scrollable.
// Call this from widgets (like Table) when they have a selected/focused row.
// The parent Scrollable will automatically scroll to keep this Y visible.
// The Y is relative to the Scrollable's content area (not screen coordinates).
func (ctx *Context) SetScrollFocus(y float32, padding float32) {
	ctx.scrollFocusY = y
	ctx.scrollFocusPad = padding
	ctx.scrollFocusSet = true
}

// ConsumeScrollFocus returns the scroll focus if set, and clears it.
// Called by Scrollable to get the focus Y that widgets registered.
// Returns (y, padding, ok) where ok is false if no focus was set.
func (ctx *Context) ConsumeScrollFocus() (y, padding float32, ok bool) {
	if !ctx.scrollFocusSet {
		return 0, 0, false
	}
	y = ctx.scrollFocusY
	padding = ctx.scrollFocusPad
	ctx.scrollFocusSet = false
	return y, padding, true
}

// scrollableContext tracks a scrollable's state for the stack
type scrollableContext struct {
	id            ID
	contentOrigin float32 // Y position of content start (for coordinate translation)
	viewportY     float32 // Top Y of the scrollable viewport in screen space
	viewportH     float32 // Height of the scrollable viewport
	focusY        float32 // Content-relative Y of focused item
	padding       float32
	hasSet        bool
}

// pushScrollable adds a scrollable to the stack (called by Scrollable widget)
// contentOriginY is the Y position where content rendering starts (cursor.Y after scroll offset)
// viewportY is the top of the scrollable viewport in screen space
// viewportH is the height of the scrollable viewport
// This enables automatic coordinate translation in ScrollTo() and visibility checking for hit testing.
func (ctx *Context) pushScrollable(id ID, contentOriginY, viewportY, viewportH float32) {
	ctx.scrollableStack = append(ctx.scrollableStack, &scrollableContext{
		id:            id,
		contentOrigin: contentOriginY,
		viewportY:     viewportY,
		viewportH:     viewportH,
	})
}

// IsInsideScrollableViewport checks if a Y coordinate is inside the current scrollable's visible viewport.
// Items drawn outside the viewport (clipped) should not respond to mouse events.
// Returns true if not inside a scrollable (no clipping) or if inside the visible area.
func (ctx *Context) IsInsideScrollableViewport(y, h float32) bool {
	n := len(ctx.scrollableStack)
	if n == 0 {
		return true // Not inside a scrollable, always visible
	}
	sc := ctx.scrollableStack[n-1]
	// Check if item overlaps with visible viewport
	itemTop := y
	itemBottom := y + h
	viewportTop := sc.viewportY
	viewportBottom := sc.viewportY + sc.viewportH
	return itemBottom > viewportTop && itemTop < viewportBottom
}

// popScrollable removes and returns the current scrollable's focus info
func (ctx *Context) popScrollable() (focusY, padding float32, ok bool) {
	n := len(ctx.scrollableStack)
	if n == 0 {
		return 0, 0, false
	}
	sc := ctx.scrollableStack[n-1]
	ctx.scrollableStack = ctx.scrollableStack[:n-1]
	return sc.focusY, sc.padding, sc.hasSet
}

// ScrollTo registers the current cursor position as the focus for the parent Scrollable.
// Call this from any widget inside a Scrollable to make it scroll to keep the widget visible.
// The coordinate translation from screen position to content-relative is automatic.
//
// Usage (inside a Scrollable):
//
//	ctx.Scrollable("list", 300)(func() {
//	    for i, item := range items {
//	        if i == selectedIndex {
//	            ctx.ScrollTo(ctx.cursor.Y, rowHeight) // Just pass cursor.Y!
//	        }
//	        ctx.Text(item.Name)
//	    }
//	})
//
// The Scrollable will auto-scroll to keep this Y visible when it changes between frames.
func (ctx *Context) ScrollTo(screenY float32, padding float32) {
	n := len(ctx.scrollableStack)
	if n == 0 {
		return // No scrollable parent
	}
	sc := ctx.scrollableStack[n-1]
	// Convert screen position to content-relative position
	// screenY is where the widget renders on screen (already offset by scroll)
	// contentOrigin is where content starts on screen (also offset by scroll)
	// So contentRelativeY = screenY - contentOrigin
	sc.focusY = screenY - sc.contentOrigin
	sc.padding = padding
	sc.hasSet = true
}

// FontProvider returns the current font provider, or nil if not set.
func (ctx *Context) FontProvider() FontProvider {
	return ctx.fontProvider
}

// SetFont sets the active font by name.
// Returns an error if the font is not found.
// Does nothing if no font provider is set.
func (ctx *Context) SetFont(name string) error {
	if ctx.fontProvider == nil {
		return nil
	}
	return ctx.fontProvider.SetActiveFont(name)
}

// currentLayoutWidth returns the available width in the current layout.
func (ctx *Context) currentLayoutWidth() float32 {
	if len(ctx.layoutStack) > 0 {
		layout := ctx.layoutStack[len(ctx.layoutStack)-1]
		return layout.Width - layout.Padding*2 - layout.PaddingX*2
	}
	return ctx.DisplaySize.X
}

// CurrentLayoutWidth returns the available width in the current layout (public API).
func (ctx *Context) CurrentLayoutWidth() float32 {
	return ctx.currentLayoutWidth()
}

// currentLayoutHeight returns the available height in the current layout.
func (ctx *Context) currentLayoutHeight() float32 {
	if len(ctx.layoutStack) > 0 {
		layout := ctx.layoutStack[len(ctx.layoutStack)-1]
		return layout.Height - layout.Padding*2 - layout.PaddingY*2
	}
	return ctx.DisplaySize.Y
}

// currentLayout returns the current layout or nil.
func (ctx *Context) currentLayout() *Layout {
	if len(ctx.layoutStack) > 0 {
		return ctx.layoutStack[len(ctx.layoutStack)-1]
	}
	return nil
}

// addText is a helper to draw text with current style.
// Uses the font provider if available, otherwise falls back to built-in monospace font.
// Performance: reuses pre-allocated glyph buffer to avoid allocations in hot paths.
func (ctx *Context) addText(x, y float32, text string, color uint32) {
	ctx.AddText(x, y, text, color)
}

// addTextTo draws text to a specific DrawList (for foreground/overlay rendering).
func (ctx *Context) addTextTo(dl *DrawList, x, y float32, text string, color uint32) {
	ctx.AddTextTo(dl, x, y, text, color)
}

// AddTextTo draws text to a specific DrawList (public API).
// This is useful for drawing to foreground/overlay layers.
func (ctx *Context) AddTextTo(dl *DrawList, x, y float32, text string, color uint32) {
	if dl == nil {
		return
	}
	if f := ctx.activeFont(); f != nil {
		dl.SetTexture(f.TextureID())
		fontQuads := f.GetGlyphQuads(text, x, y, ctx.style.FontScale)

		if cap(ctx.glyphBuffer) < len(fontQuads) {
			ctx.glyphBuffer = make([]GlyphQuad, 0, len(fontQuads)*2)
		}
		ctx.glyphBuffer = ctx.glyphBuffer[:len(fontQuads)]

		for i, q := range fontQuads {
			ctx.glyphBuffer[i] = GlyphQuad{
				X0: q.X0, Y0: q.Y0,
				X1: q.X1, Y1: q.Y1,
				U0: q.U0, V0: q.V0,
				U1: q.U1, V1: q.V1,
			}
		}
		dl.AddGlyphQuads(ctx.glyphBuffer, color)
		dl.SetTexture(0)
		return
	}

	// Fallback to built-in monospace font (legacy renderer)
	dl.SetTexture(ctx.FontTextureID)
	dl.AddText(x, y, text, color, ctx.style.FontScale, ctx.style.CharWidth, ctx.style.CharHeight)
	dl.SetTexture(0)
}

// AddText draws text with current style (public API).
// Uses the font provider if available, otherwise falls back to built-in monospace font.
func (ctx *Context) AddText(x, y float32, text string, color uint32) {
	if f := ctx.activeFont(); f != nil {
		ctx.DrawList.SetTexture(f.TextureID())
		// Get glyph quads from font and convert to GUI format
		fontQuads := f.GetGlyphQuads(text, x, y, ctx.style.FontScale)

		// Reuse pre-allocated buffer instead of allocating each call
		if cap(ctx.glyphBuffer) < len(fontQuads) {
			// Grow buffer with some headroom to reduce future allocations
			ctx.glyphBuffer = make([]GlyphQuad, 0, len(fontQuads)*2)
		}
		ctx.glyphBuffer = ctx.glyphBuffer[:len(fontQuads)]

		for i, q := range fontQuads {
			ctx.glyphBuffer[i] = GlyphQuad{
				X0: q.X0, Y0: q.Y0,
				X1: q.X1, Y1: q.Y1,
				U0: q.U0, V0: q.V0,
				U1: q.U1, V1: q.V1,
			}
		}
		ctx.DrawList.AddGlyphQuads(ctx.glyphBuffer, color)
		ctx.DrawList.SetTexture(0)
		return
	}

	// Fallback to built-in monospace font (legacy renderer)
	ctx.DrawList.SetTexture(ctx.FontTextureID)
	ctx.DrawList.AddText(x, y, text, color, ctx.style.FontScale, ctx.style.CharWidth, ctx.style.CharHeight)
	ctx.DrawList.SetTexture(0)
}

// beginItem applies gap spacing before drawing an item.
// Call this before drawing any widget to ensure proper spacing.
func (ctx *Context) beginItem() {
	layout := ctx.currentLayout()
	if layout == nil {
		return
	}

	// Add gap BEFORE this item (if not first)
	if layout.ItemCount > 0 {
		if layout.Type == LayoutVertical {
			gap := layout.GapY
			if gap == 0 {
				gap = layout.Gap
			}
			if gap == 0 {
				gap = ctx.style.ItemSpacing
			}
			ctx.cursor.Y += gap
		} else {
			gap := layout.GapX
			if gap == 0 {
				gap = layout.Gap
			}
			if gap == 0 {
				gap = ctx.style.ItemSpacing
			}
			ctx.cursor.X += gap
		}
	}
}

// ItemPos returns the position for the next widget with gap applied.
// This is the recommended way for widgets to get their drawing position.
// It handles layout gaps automatically.
func (ctx *Context) ItemPos() Vec2 {
	ctx.beginItem()
	return ctx.cursor
}

// advanceCursor moves the cursor after drawing an item.
func (ctx *Context) advanceCursor(size Vec2) {
	ctx.AdvanceCursor(size)
}

// AdvanceCursor moves the cursor after drawing an item (public API).
func (ctx *Context) AdvanceCursor(size Vec2) {
	layout := ctx.currentLayout()
	if layout == nil {
		// No layout, just advance vertically
		ctx.cursor.Y += size.Y + ctx.style.ItemSpacing
		return
	}

	// Track content bounds
	if layout.Type == LayoutVertical {
		ctx.cursor.Y += size.Y
		layout.MaxWidth = maxf(layout.MaxWidth, size.X)
		layout.MaxHeight = ctx.cursor.Y - layout.StartY
	} else {
		ctx.cursor.X += size.X
		layout.MaxWidth = ctx.cursor.X - layout.StartX
		layout.MaxHeight = maxf(layout.MaxHeight, size.Y)
	}

	layout.ItemCount++
}

// =============================================================================
// Focus Hierarchy Methods
// =============================================================================

// BeginFocusScope starts a focusable container scope.
// Call this when entering a widget that can contain focusable children.
// Must be paired with EndFocusScope.
//
// Parameters:
//   - id: Widget ID for state lookup
//   - name: Debug-friendly identifier
//   - typ: Widget category (Container, Section, List, etc.)
//   - rect: Bounds for hit testing
//
// Usage:
//
//	ctx.BeginFocusScope(id, "my_section", FocusTypeSection, rect)
//	defer ctx.EndFocusScope()
//	// ... draw focusable children ...
func (ctx *Context) BeginFocusScope(id ID, name string, typ FocusType, rect Rect) {
	node := FocusNode{
		ID:       id,
		Name:     name,
		Type:     typ,
		ChildIdx: -1, // No child focused yet
		Rect:     rect,
		// Save parent's child focus state so we can restore it when this scope ends
		savedChildFocusSet:    ctx.childFocusSet,
		savedChildFocusY:      ctx.childFocusY,
		savedChildFocusHeight: ctx.childFocusHeight,
	}
	ctx.focusStack = append(ctx.focusStack, node)

	// Clear child focus tracking for this new scope
	ctx.childFocusSet = false
}

// EndFocusScope ends a focusable container scope and returns focus info.
// The returned FocusInfo tells the parent whether any child had focus
// and where that focus was (for auto-scroll purposes).
//
// Usage:
//
//	ctx.BeginFocusScope(id, "list", FocusTypeList, rect)
//	// ... draw items, some may call ReportChildFocus ...
//	info := ctx.EndFocusScope()
//	if info.HasFocusedChild {
//	    // Auto-scroll to info.FocusedChildY
//	}
func (ctx *Context) EndFocusScope() FocusInfo {
	info := FocusInfo{
		FocusedChildIdx: -1,
	}

	n := len(ctx.focusStack)
	if n == 0 {
		return info
	}

	// Pop the scope
	node := ctx.focusStack[n-1]
	ctx.focusStack = ctx.focusStack[:n-1]

	// Collect child focus info
	if ctx.childFocusSet {
		info.HasFocusedChild = true
		info.FocusedChildIdx = node.ChildIdx
		info.FocusedChildY = ctx.childFocusY
		info.FocusedChildHeight = ctx.childFocusHeight
	}

	// Restore parent's child focus state, but keep focus info if this scope had focus
	// This allows nested scopes (like Section inside Scrollable) to work correctly
	if info.HasFocusedChild {
		// This scope had a focused child - keep the focus info for parent
		// (childFocusSet, childFocusY, childFocusHeight are already set)
	} else if node.savedChildFocusSet {
		// This scope didn't have focus, but parent did - restore parent's focus
		ctx.childFocusSet = node.savedChildFocusSet
		ctx.childFocusY = node.savedChildFocusY
		ctx.childFocusHeight = node.savedChildFocusHeight
	} else {
		// Neither this scope nor parent had focus
		ctx.childFocusSet = false
	}

	return info
}

// IsFocusAncestor returns true if the given ID is anywhere in the current focus path.
// Use this to determine if a container has focus somewhere within its children.
//
// Example: A section can draw a highlight if any of its children are focused.
func (ctx *Context) IsFocusAncestor(id ID) bool {
	return ctx.focusPath.Contains(id)
}

// GetFocusDepth returns the depth of the given ID in the focus path.
// Returns -1 if the ID is not in the focus path.
//
// Depth 0 is the root (topmost focused container), increasing toward the leaf.
func (ctx *Context) GetFocusDepth(id ID) int {
	return ctx.focusPath.IndexOf(id)
}

// ReportChildFocus is called by focused children to inform their parent container.
// The parent's EndFocusScope will receive this information in FocusInfo.
//
// Parameters:
//   - y: Y position of the focused child (screen coordinates or content-relative)
//   - height: Height of the focused child
//
// Usage (from a list item that's selected):
//
//	if isSelected {
//	    ctx.ReportChildFocus(itemY, itemHeight)
//	}
func (ctx *Context) ReportChildFocus(y, height float32) {
	ctx.childFocusY = y
	ctx.childFocusHeight = height
	ctx.childFocusSet = true

	// Update current scope's ChildIdx if we're in a scope
	// This is optional - widgets can also set ChildIdx directly
}

// SetFocusChildIdx sets the focused child index for the current focus scope.
// Call this when you know which child index is focused (e.g., selected row in table).
func (ctx *Context) SetFocusChildIdx(idx int) {
	n := len(ctx.focusStack)
	if n == 0 {
		return
	}
	ctx.focusStack[n-1].ChildIdx = idx
}

// FocusPath returns the current focus path for inspection.
// Do not modify the returned path - it's the internal state.
func (ctx *Context) FocusPath() *FocusPath {
	return ctx.focusPath
}

// SetFocusPath sets a node in the focus path.
// This is typically called when focus changes (e.g., user clicks, keyboard nav).
func (ctx *Context) SetFocusPath(nodes ...FocusNode) {
	ctx.focusPath.Clear()
	for _, node := range nodes {
		ctx.focusPath.Push(node)
	}
}

// =============================================================================
// Debug Focus Visualization
// =============================================================================

// DebugFocusColor is the color used for debug focus highlighting (bright red, more visible).
var DebugFocusColor = RGBA(255, 0, 0, 180)

// DebugFocusBorderColor is the border color for debug focus highlighting (bright red, thick).
var DebugFocusBorderColor = RGBA(255, 50, 50, 255)

// DrawDebugFocusRect draws a debug focus highlight on the given rectangle.
// Only draws if DebugFocusHighlight is enabled.
// Use this in widgets to visualize focus state during debugging.
func (ctx *Context) DrawDebugFocusRect(x, y, w, h float32) {
	if !ctx.DebugFocusHighlight {
		return
	}
	// Draw bright red overlay (more visible)
	ctx.DrawList.AddRect(x, y, w, h, DebugFocusColor)
	// Draw thick bright red border
	ctx.DrawList.AddRectOutline(x, y, w, h, DebugFocusBorderColor, 3)
}

// DrawDebugFocusRectIf draws a debug focus highlight if the condition is true.
// Convenience method for widgets that check focus state.
func (ctx *Context) DrawDebugFocusRectIf(focused bool, x, y, w, h float32) {
	if focused {
		ctx.DrawDebugFocusRect(x, y, w, h)
	}
}

// DrawDebugFocusForID draws a debug focus highlight if the given ID is registry-focused.
// Use this in widgets that track focus by ID.
func (ctx *Context) DrawDebugFocusForID(id ID, x, y, w, h float32) {
	if ctx.IsRegistryFocused(id) {
		ctx.DrawDebugFocusRect(x, y, w, h)
	}
}

// =============================================================================
// Focusable Registry Methods
// =============================================================================

// RegisterFocusable registers a widget as focusable for this frame.
// Returns a FocusableHandle that can be used to check focus state and handle navigation.
// The debug focus highlight is drawn automatically if the widget is focused.
//
// Usage:
//
//	func (ctx *Context) MyWidget(label string) {
//	    id := ctx.GetID(label)
//	    rect := Rect{X: pos.X, Y: pos.Y, W: w, H: h}
//	    focusable := ctx.RegisterFocusable(id, label, rect, FocusTypeLeaf)
//	    // Debug highlight is drawn automatically
//	    if focusable.IsFocused() {
//	        // Apply focused styling
//	    }
//	}
func (ctx *Context) RegisterFocusable(id ID, name string, rect Rect, typ FocusType) *FocusableHandle {
	if ctx.focusRegistry == nil {
		return nil
	}

	// Register with the given rect (draw coordinates = screen coordinates)
	handle := ctx.focusRegistry.Register(id, name, rect, typ)

	// Click to focus: any focusable item can be clicked to set focus
	// This unifies mouse and keyboard focus - no need for manual click handlers
	// IMPORTANT: Only process clicks for items that are VISIBLE (inside scrollable viewport)
	isVisible := ctx.IsInsideScrollableViewport(rect.Y, rect.H)
	if handle != nil && handle.CanFocus() && ctx.Input != nil && ctx.Input.MouseClicked(MouseButtonLeft) {
		mouse := Vec2{ctx.Input.MouseX, ctx.Input.MouseY}
		inRect := rect.Contains(mouse)
		if inRect && isVisible {
			guiLogger.Debug("click-to-focus triggered",
				"id", id,
				"name", name,
				"rect", rect,
				"mouse", mouse)
			ctx.focusRegistry.SetFocus(id)
		} else if guiVerbose() && inRect && !isVisible {
			// Log when click would hit but item is clipped
			guiLogger.Debug("click-to-focus blocked (item outside viewport)",
				"id", id,
				"name", name,
				"rect", rect,
				"mouse", mouse)
		} else if guiVerbose() {
			// Log near-misses for debugging coordinate issues
			// Only log if mouse Y is within 50px of rect Y (potential scroll issue)
			if mouse.Y >= rect.Y-50 && mouse.Y <= rect.Y+rect.H+50 {
				guiLogger.Debug("click-to-focus near miss",
					"id", id,
					"name", name,
					"rect", rect,
					"mouse", mouse,
					"visible", isVisible,
					"deltaY", mouse.Y-rect.Y)
			}
		}
	}

	// Draw debug focus rect to FOREGROUND so it appears on top of widget content
	// Only draw for leaf widgets, not containers (containers are just for grouping)
	if handle != nil && handle.IsFocused() && ctx.DebugFocusHighlight && typ == FocusTypeLeaf {
		// Use ForegroundDrawList so the debug rect isn't covered by widget content
		if ctx.ForegroundDrawList != nil {
			ctx.ForegroundDrawList.AddRect(rect.X, rect.Y, rect.W, rect.H, DebugFocusColor)
			ctx.ForegroundDrawList.AddRectOutline(rect.X, rect.Y, rect.W, rect.H, DebugFocusBorderColor, 3)
		}
	}

	// Report focus position to parent Scrollable for auto-scroll
	// This enables auto-scroll when any focusable widget gains focus
	if handle != nil && handle.IsFocused() {
		guiLogger.Debug("RegisterFocusable: reporting focus",
			"name", name,
			"id", id,
			"y", rect.Y,
			"h", rect.H)
		ctx.ReportChildFocus(rect.Y, rect.H)
	}

	return handle
}

// RegisterFocusableDisabled registers a widget that cannot receive focus.
// It's still tracked for navigation purposes (e.g., skipped when navigating).
func (ctx *Context) RegisterFocusableDisabled(id ID, name string, rect Rect, typ FocusType) *FocusableHandle {
	if ctx.focusRegistry == nil {
		return nil
	}
	return ctx.focusRegistry.RegisterDisabled(id, name, rect, typ)
}

// BeginFocusGroup starts a focus scope for a container widget.
// All focusables registered until EndFocusGroup are considered children of this scope.
func (ctx *Context) BeginFocusGroup(id ID, name string, rect Rect) {
	if ctx.focusRegistry != nil {
		ctx.focusRegistry.BeginScope(id, name, FocusTypeContainer, rect)
	}
}

// EndFocusGroup ends the current focus scope.
// Returns info about which child had focus.
func (ctx *Context) EndFocusGroup() FocusScopeEntry {
	if ctx.focusRegistry == nil {
		return FocusScopeEntry{FocusedChild: -1}
	}
	return ctx.focusRegistry.EndScope()
}

// NavigateFocus moves focus in the given direction.
// Returns true if focus moved, false if at boundary or no focusable widgets.
//
// Usage (in panel HandleInput):
//
//	if input.KeyPressed(KeyUp) {
//	    ctx.NavigateFocus(NavUp)
//	}
func (ctx *Context) NavigateFocus(dir NavDirection) bool {
	if ctx.focusRegistry == nil {
		guiLogger.Debug("NavigateFocus: registry is nil")
		return false
	}
	// Skip navigation if a popup is active (navigation stays within popup)
	if ctx.activePopupID != 0 {
		guiLogger.Debug("NavigateFocus: blocked by activePopupID", "popupID", ctx.activePopupID)
		return false
	}
	result := ctx.focusRegistry.Navigate(dir)
	guiLogger.Debug("NavigateFocus", "dir", dir, "success", result)
	return result
}

// SetRegistryFocus sets focus to the widget with the given ID.
// This updates the focus registry, which is separate from the simple focusedID.
func (ctx *Context) SetRegistryFocus(id ID) {
	if ctx.focusRegistry != nil {
		ctx.focusRegistry.SetFocus(id)
	}
}

// ClearRegistryFocus removes focus from all widgets in the registry.
func (ctx *Context) ClearRegistryFocus() {
	if ctx.focusRegistry != nil {
		ctx.focusRegistry.ClearFocus()
	}
}

// FocusRegistry returns the focus registry for advanced usage.
// Most widgets should use the higher-level methods instead.
func (ctx *Context) FocusRegistry() *FocusRegistry {
	return ctx.focusRegistry
}

// MarkKeyboardNavigated signals that keyboard navigation occurred this frame.
// Call this from panels that use custom navigation (not NavigateFocus)
// to enable auto-scroll when navigating via keyboard.
// This is automatically called by NavigateFocus, but panels with manual
// navigation (e.g., custom row selection) should call this explicitly.
func (ctx *Context) MarkKeyboardNavigated() {
	if ctx.focusRegistry != nil {
		ctx.focusRegistry.MarkKeyboardNavigated()
	}
}

// FocusedItem returns the currently focused item from the registry.
// Returns nil if no widget has focus.
func (ctx *Context) FocusedItem() *FocusableItem {
	if ctx.focusRegistry == nil {
		return nil
	}
	return ctx.focusRegistry.CurrentFocusItem()
}

// IsRegistryFocused returns true if the given ID has focus in the registry.
func (ctx *Context) IsRegistryFocused(id ID) bool {
	if ctx.focusRegistry == nil {
		return false
	}
	return ctx.focusRegistry.CurrentFocusID() == id
}

// FocusFirstWidget sets focus to the first focusable widget.
func (ctx *Context) FocusFirstWidget() bool {
	if ctx.focusRegistry == nil {
		return false
	}
	return ctx.focusRegistry.FocusFirst()
}

// FocusLastWidget sets focus to the last focusable widget.
func (ctx *Context) FocusLastWidget() bool {
	if ctx.focusRegistry == nil {
		return false
	}
	return ctx.focusRegistry.FocusLast()
}

// FocusWidgetByIndex sets focus to the widget at the given registration index.
func (ctx *Context) FocusWidgetByIndex(idx int) bool {
	if ctx.focusRegistry == nil {
		return false
	}
	return ctx.focusRegistry.FocusByIndex(idx)
}
