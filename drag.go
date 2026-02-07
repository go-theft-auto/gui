package gui

// DragState tracks the state of a drag operation.
// Used by draggable panels to enable window movement.
type DragState struct {
	Active    bool    // Currently being dragged
	StartX    float32 // Mouse X when drag started
	StartY    float32 // Mouse Y when drag started
	OffsetX   float32 // Panel X offset from mouse when drag started
	OffsetY   float32 // Panel Y offset from mouse when drag started
	PanelName string  // Name of the panel being dragged (for identification)
}

// Reset clears the drag state.
func (d *DragState) Reset() {
	d.Active = false
	d.StartX = 0
	d.StartY = 0
	d.OffsetX = 0
	d.OffsetY = 0
	d.PanelName = ""
}

// SnapConfig configures panel snapping behavior.
type SnapConfig struct {
	Enabled     bool    // Enable snapping
	GridSize    float32 // Grid size for grid snapping (0 = disabled)
	EdgeMargin  float32 // Snap to screen edges within this margin
	PanelMargin float32 // Snap to other panels within this margin
}

// DefaultSnapConfig returns a sensible default snap configuration.
func DefaultSnapConfig() SnapConfig {
	return SnapConfig{
		Enabled:     true,
		GridSize:    0,  // No grid by default
		EdgeMargin:  10, // Snap to edges within 10px
		PanelMargin: 0,  // No panel-to-panel snapping by default
	}
}

// DraggablePanel wraps panel positioning and drag behavior.
// Panels can embed this to gain drag functionality.
type DraggablePanel struct {
	// Position is the panel's current position.
	Position Vec2

	// Size is the panel's current size (set after drawing).
	Size Vec2

	// Draggable enables/disables drag functionality.
	Draggable bool

	// TitleBarHeight is the height of the draggable title bar region.
	// If 0, uses the default (line height + padding).
	TitleBarHeight float32

	// SnapConfig controls snapping behavior.
	SnapConfig SnapConfig

	// Resizable enables/disables resize functionality.
	Resizable bool

	// MinSize is the minimum allowed size when resizing.
	MinSize Vec2

	// MaxSize is the maximum allowed size when resizing.
	// Zero means no maximum.
	MaxSize Vec2

	// ResizeHandleSize is the width of the resize handle area in pixels.
	// Default is 6 pixels.
	ResizeHandleSize float32

	// Internal drag state
	dragState DragState

	// Internal resize state
	resizeState ResizeState

	// Snap manager for live snap visualization
	snapManager *SnapManager

	// Active snap guides (populated during drag)
	activeSnapGuides []SnapGuide

	// Panel name for snap exclusion
	panelName string
}

// NewDraggablePanel creates a new draggable panel with default settings.
func NewDraggablePanel(x, y float32) *DraggablePanel {
	return &DraggablePanel{
		Position:         Vec2{X: x, Y: y},
		Draggable:        true,
		SnapConfig:       DefaultSnapConfig(),
		Resizable:        false,
		MinSize:          Vec2{X: 100, Y: 50},
		MaxSize:          Vec2{X: 0, Y: 0}, // No maximum
		ResizeHandleSize: 6,
	}
}

// NewResizablePanel creates a new draggable and resizable panel.
func NewResizablePanel(x, y, w, h float32) *DraggablePanel {
	return &DraggablePanel{
		Position:         Vec2{X: x, Y: y},
		Size:             Vec2{X: w, Y: h},
		Draggable:        true,
		Resizable:        true,
		SnapConfig:       DefaultSnapConfig(),
		MinSize:          Vec2{X: 100, Y: 50},
		MaxSize:          Vec2{X: 0, Y: 0},
		ResizeHandleSize: 6,
	}
}

// TitleBarRect returns the rectangle for the draggable title bar area.
// The title bar is at the top of the panel.
func (dp *DraggablePanel) TitleBarRect(ctx *Context) Rect {
	h := dp.TitleBarHeight
	if h == 0 {
		h = ctx.LineHeight() + ctx.Style().PanelPadding*2
	}
	return Rect{
		X: dp.Position.X,
		Y: dp.Position.Y,
		W: dp.Size.X,
		H: h,
	}
}

// HandleDrag processes drag input for the panel.
// Call this before drawing the panel each frame.
// Returns true if the panel is currently being dragged.
func (dp *DraggablePanel) HandleDrag(ctx *Context) bool {
	if !dp.Draggable || ctx.Input == nil {
		return false
	}

	input := ctx.Input
	mousePos := Vec2{X: input.MouseX, Y: input.MouseY}

	// Check for drag start (mouse click in title bar)
	if input.MouseClicked(MouseButtonLeft) {
		titleBar := dp.TitleBarRect(ctx)
		if titleBar.Contains(mousePos) {
			dp.dragState.Active = true
			dp.dragState.StartX = mousePos.X
			dp.dragState.StartY = mousePos.Y
			dp.dragState.OffsetX = dp.Position.X - mousePos.X
			dp.dragState.OffsetY = dp.Position.Y - mousePos.Y
		}
	}

	// Handle ongoing drag
	if dp.dragState.Active {
		if input.MouseDown(MouseButtonLeft) {
			// Update position based on mouse movement
			newX := mousePos.X + dp.dragState.OffsetX
			newY := mousePos.Y + dp.dragState.OffsetY

			// Clamp to screen bounds
			newX = clampf(newX, 0, ctx.DisplaySize.X-dp.Size.X)
			newY = clampf(newY, 0, ctx.DisplaySize.Y-dp.Size.Y)

			// Calculate snap preview if snap manager is available
			if dp.snapManager != nil && dp.SnapConfig.Enabled {
				bounds := PanelBounds{
					X: newX, Y: newY,
					W: dp.Size.X, H: dp.Size.Y,
					Name: dp.panelName,
				}
				snappedPos, guides := dp.snapManager.CalculateSnap(bounds, dp.panelName)
				dp.activeSnapGuides = guides
				// Show the snapped position (preview)
				newX = snappedPos.X
				newY = snappedPos.Y
			} else {
				dp.activeSnapGuides = nil
			}

			dp.Position = Vec2{X: newX, Y: newY}
		} else {
			// Mouse released, end drag and apply snapping
			dp.endDrag(ctx)
		}
	}

	return dp.dragState.Active
}

// endDrag ends the drag operation and applies snapping.
func (dp *DraggablePanel) endDrag(ctx *Context) {
	if !dp.dragState.Active {
		return
	}

	dp.dragState.Active = false
	dp.activeSnapGuides = nil // Clear snap guides

	if !dp.SnapConfig.Enabled {
		return
	}

	// If using snap manager, position is already snapped from preview
	if dp.snapManager != nil {
		return
	}

	// Apply legacy edge snapping (fallback if no snap manager)
	if dp.SnapConfig.EdgeMargin > 0 {
		dp.snapToEdges(ctx)
	}

	// Apply grid snapping
	if dp.SnapConfig.GridSize > 0 {
		dp.snapToGrid()
	}
}

// snapToEdges snaps the panel to screen edges if within margin.
func (dp *DraggablePanel) snapToEdges(ctx *Context) {
	margin := dp.SnapConfig.EdgeMargin

	// Left edge
	if dp.Position.X < margin {
		dp.Position.X = 0
	}

	// Top edge
	if dp.Position.Y < margin {
		dp.Position.Y = 0
	}

	// Right edge
	rightEdge := ctx.DisplaySize.X - dp.Size.X
	if dp.Position.X > rightEdge-margin {
		dp.Position.X = rightEdge
	}

	// Bottom edge
	bottomEdge := ctx.DisplaySize.Y - dp.Size.Y
	if dp.Position.Y > bottomEdge-margin {
		dp.Position.Y = bottomEdge
	}
}

// snapToGrid snaps the panel position to a grid.
func (dp *DraggablePanel) snapToGrid() {
	grid := dp.SnapConfig.GridSize
	if grid <= 0 {
		return
	}

	dp.Position.X = float32(int(dp.Position.X/grid+0.5)) * grid
	dp.Position.Y = float32(int(dp.Position.Y/grid+0.5)) * grid
}

// IsDragging returns true if the panel is currently being dragged.
func (dp *DraggablePanel) IsDragging() bool {
	return dp.dragState.Active
}

// SetPosition sets the panel position.
func (dp *DraggablePanel) SetPosition(x, y float32) {
	dp.Position = Vec2{X: x, Y: y}
}

// SetSize sets the panel size (typically called after measuring content).
func (dp *DraggablePanel) SetSize(w, h float32) {
	dp.Size = Vec2{X: w, Y: h}
}

// GetPosition returns the current panel position.
func (dp *DraggablePanel) GetPosition() Vec2 {
	return dp.Position
}

// GetSize returns the current panel size.
func (dp *DraggablePanel) GetSize() Vec2 {
	return dp.Size
}

// SetSnapManager sets the snap manager for live snap visualization.
// Pass nil to disable snap visualization.
func (dp *DraggablePanel) SetSnapManager(sm *SnapManager, panelName string) {
	dp.snapManager = sm
	dp.panelName = panelName
}

// DrawSnapGuides draws the snap guide lines during a drag operation.
// Call this after drawing all panels to show snap feedback on top.
func (dp *DraggablePanel) DrawSnapGuides(ctx *Context) {
	if len(dp.activeSnapGuides) == 0 || !dp.dragState.Active {
		return
	}

	// Draw cyan guide lines with glow effect
	guideColor := RGBA(0, 180, 255, 200)
	glowColor := RGBA(0, 180, 255, 60)

	for _, guide := range dp.activeSnapGuides {
		// Draw glow (thicker, semi-transparent)
		ctx.DrawList.AddLine(guide.X1, guide.Y1, guide.X2, guide.Y2, glowColor, 5)
		// Draw main line
		ctx.DrawList.AddLine(guide.X1, guide.Y1, guide.X2, guide.Y2, guideColor, 1.5)
	}
}

// HasActiveSnapGuides returns true if there are snap guides to display.
func (dp *DraggablePanel) HasActiveSnapGuides() bool {
	return len(dp.activeSnapGuides) > 0 && dp.dragState.Active
}

// Constrain ensures the panel stays within screen bounds.
func (dp *DraggablePanel) Constrain(displaySize Vec2) {
	// Ensure panel is at least partially visible
	minVisible := float32(50) // At least 50px visible

	if dp.Position.X < -dp.Size.X+minVisible {
		dp.Position.X = -dp.Size.X + minVisible
	}
	if dp.Position.X > displaySize.X-minVisible {
		dp.Position.X = displaySize.X - minVisible
	}
	if dp.Position.Y < 0 {
		dp.Position.Y = 0
	}
	if dp.Position.Y > displaySize.Y-minVisible {
		dp.Position.Y = displaySize.Y - minVisible
	}
}

// GetResizeEdge returns which edge(s) the mouse is near for resize.
// Returns ResizeEdgeNone if mouse is not near any edge or if not resizable.
func (dp *DraggablePanel) GetResizeEdge(ctx *Context) ResizableEdge {
	if !dp.Resizable || ctx.Input == nil {
		return ResizeEdgeNone
	}

	handleSize := dp.ResizeHandleSize
	if handleSize <= 0 {
		handleSize = 6
	}

	mx, my := ctx.Input.MouseX, ctx.Input.MouseY
	px, py := dp.Position.X, dp.Position.Y
	pw, ph := dp.Size.X, dp.Size.Y

	var edge ResizableEdge

	// Check if mouse is within panel bounds (with handle extension)
	if mx < px-handleSize || mx > px+pw+handleSize ||
		my < py-handleSize || my > py+ph+handleSize {
		return ResizeEdgeNone
	}

	// Check edges
	if mx >= px-handleSize && mx <= px+handleSize {
		edge |= ResizeEdgeLeft
	}
	if mx >= px+pw-handleSize && mx <= px+pw+handleSize {
		edge |= ResizeEdgeRight
	}
	if my >= py-handleSize && my <= py+handleSize {
		edge |= ResizeEdgeTop
	}
	if my >= py+ph-handleSize && my <= py+ph+handleSize {
		edge |= ResizeEdgeBottom
	}

	return edge
}

// HandleResize processes resize input for the panel.
// Call this before drawing the panel each frame.
// Returns true if the panel is currently being resized.
func (dp *DraggablePanel) HandleResize(ctx *Context) bool {
	if !dp.Resizable || ctx.Input == nil {
		return false
	}

	input := ctx.Input
	mousePos := Vec2{X: input.MouseX, Y: input.MouseY}

	// Check for resize start (mouse click on edge)
	if input.MouseClicked(MouseButtonLeft) && !dp.resizeState.Active {
		edge := dp.GetResizeEdge(ctx)
		if edge != ResizeEdgeNone {
			dp.resizeState.Active = true
			dp.resizeState.Edge = edge
			dp.resizeState.StartMouseX = mousePos.X
			dp.resizeState.StartMouseY = mousePos.Y
			dp.resizeState.StartX = dp.Position.X
			dp.resizeState.StartY = dp.Position.Y
			dp.resizeState.StartW = dp.Size.X
			dp.resizeState.StartH = dp.Size.Y
		}
	}

	// Handle ongoing resize
	if dp.resizeState.Active {
		if input.MouseDown(MouseButtonLeft) {
			deltaX := mousePos.X - dp.resizeState.StartMouseX
			deltaY := mousePos.Y - dp.resizeState.StartMouseY

			newX := dp.resizeState.StartX
			newY := dp.resizeState.StartY
			newW := dp.resizeState.StartW
			newH := dp.resizeState.StartH

			// Apply resize based on which edge is being dragged
			if dp.resizeState.Edge&ResizeEdgeLeft != 0 {
				newX = dp.resizeState.StartX + deltaX
				newW = dp.resizeState.StartW - deltaX
			}
			if dp.resizeState.Edge&ResizeEdgeRight != 0 {
				newW = dp.resizeState.StartW + deltaX
			}
			if dp.resizeState.Edge&ResizeEdgeTop != 0 {
				newY = dp.resizeState.StartY + deltaY
				newH = dp.resizeState.StartH - deltaY
			}
			if dp.resizeState.Edge&ResizeEdgeBottom != 0 {
				newH = dp.resizeState.StartH + deltaY
			}

			// Apply min/max constraints
			if dp.MinSize.X > 0 && newW < dp.MinSize.X {
				if dp.resizeState.Edge&ResizeEdgeLeft != 0 {
					newX = dp.resizeState.StartX + dp.resizeState.StartW - dp.MinSize.X
				}
				newW = dp.MinSize.X
			}
			if dp.MinSize.Y > 0 && newH < dp.MinSize.Y {
				if dp.resizeState.Edge&ResizeEdgeTop != 0 {
					newY = dp.resizeState.StartY + dp.resizeState.StartH - dp.MinSize.Y
				}
				newH = dp.MinSize.Y
			}
			if dp.MaxSize.X > 0 && newW > dp.MaxSize.X {
				if dp.resizeState.Edge&ResizeEdgeLeft != 0 {
					newX = dp.resizeState.StartX + dp.resizeState.StartW - dp.MaxSize.X
				}
				newW = dp.MaxSize.X
			}
			if dp.MaxSize.Y > 0 && newH > dp.MaxSize.Y {
				if dp.resizeState.Edge&ResizeEdgeTop != 0 {
					newY = dp.resizeState.StartY + dp.resizeState.StartH - dp.MaxSize.Y
				}
				newH = dp.MaxSize.Y
			}

			dp.Position = Vec2{X: newX, Y: newY}
			dp.Size = Vec2{X: newW, Y: newH}
		} else {
			// Mouse released, end resize
			dp.resizeState.Active = false
		}
	}

	return dp.resizeState.Active
}

// IsResizing returns true if the panel is currently being resized.
func (dp *DraggablePanel) IsResizing() bool {
	return dp.resizeState.Active
}

// DrawResizeHandles draws visual indicators for resize handles.
// Call this after drawing the panel content.
// Uses dp.Position and dp.Size for bounds - ensure Size is updated before calling.
func (dp *DraggablePanel) DrawResizeHandles(ctx *Context) {
	dp.DrawResizeHandlesAt(ctx, dp.Position.X, dp.Position.Y, dp.Size.X, dp.Size.Y)
}

// DrawResizeHandlesAt draws resize handles at specific bounds.
// Use this when the panel's actual rendered bounds differ from dp.Size.
func (dp *DraggablePanel) DrawResizeHandlesAt(ctx *Context, x, y, w, h float32) {
	if !dp.Resizable {
		return
	}

	handleSize := dp.ResizeHandleSize
	if handleSize <= 0 {
		handleSize = 8 // Slightly larger for easier targeting
	}

	edge := dp.GetResizeEdge(ctx)
	isResizing := dp.resizeState.Active

	// Colors for different states - more visible and interesting
	gripDotColor := RGBA(200, 200, 200, 255) // Bright dots (always visible)
	gripBgColor := RGBA(60, 60, 60, 200)     // Dark background for contrast
	hoverColor := RGBA(0, 200, 255, 255)     // Cyan on hover
	activeColor := RGBA(100, 255, 255, 255)  // Bright cyan when resizing

	// Determine grip color state
	isHoveringCorner := edge&ResizeEdgeRight != 0 && edge&ResizeEdgeBottom != 0
	isResizingCorner := isResizing && dp.resizeState.Edge&ResizeEdgeRight != 0 && dp.resizeState.Edge&ResizeEdgeBottom != 0

	gripColor := gripDotColor
	if isResizingCorner {
		gripColor = activeColor
	} else if isHoveringCorner {
		gripColor = hoverColor
	}

	// Draw resize grip icon at bottom-right corner using a dot grid pattern
	dotSize := float32(2)
	dotSpacing := float32(4)
	gripPadding := float32(5)

	// Calculate grip area position
	gripAreaSize := dotSpacing*2 + dotSize
	gripX := x + w - gripAreaSize - gripPadding
	gripY := y + h - gripAreaSize - gripPadding

	// Draw background rectangle for contrast
	ctx.DrawList.AddRect(gripX-2, gripY-2, gripAreaSize+4, gripAreaSize+4, gripBgColor)

	// Draw 6 dots in lower-right triangle pattern (more interesting than lines)
	for row := range 3 {
		for col := range 3 {
			// Only draw lower-right triangle of dots
			if row+col >= 2 {
				dotX := gripX + float32(col)*dotSpacing
				dotY := gripY + float32(row)*dotSpacing
				ctx.DrawList.AddRect(dotX, dotY, dotSize, dotSize, gripColor)
			}
		}
	}

	// Always draw corner triangle indicator (makes resize affordance clear)
	cornerSize := float32(12)
	cornerColor := RGBA(80, 80, 80, 180)
	if isResizingCorner {
		cornerColor = activeColor
	} else if isHoveringCorner {
		cornerColor = hoverColor
	}
	ctx.DrawList.AddTriangle(
		x+w, y+h-cornerSize,
		x+w, y+h,
		x+w-cornerSize, y+h,
		cornerColor,
	)

	// Draw edge highlights when hovering or resizing
	edgeHighlightColor := hoverColor
	if isResizing {
		edgeHighlightColor = activeColor
	}

	thickness := float32(2)

	// Draw highlighted edges using explicit bounds
	if edge&ResizeEdgeLeft != 0 || (isResizing && dp.resizeState.Edge&ResizeEdgeLeft != 0) {
		ctx.DrawList.AddLine(x, y, x, y+h, edgeHighlightColor, thickness)
	}
	if edge&ResizeEdgeRight != 0 || (isResizing && dp.resizeState.Edge&ResizeEdgeRight != 0) {
		ctx.DrawList.AddLine(x+w, y, x+w, y+h, edgeHighlightColor, thickness)
	}
	if edge&ResizeEdgeTop != 0 || (isResizing && dp.resizeState.Edge&ResizeEdgeTop != 0) {
		ctx.DrawList.AddLine(x, y, x+w, y, edgeHighlightColor, thickness)
	}
	if edge&ResizeEdgeBottom != 0 || (isResizing && dp.resizeState.Edge&ResizeEdgeBottom != 0) {
		ctx.DrawList.AddLine(x, y+h, x+w, y+h, edgeHighlightColor, thickness)
	}
}
