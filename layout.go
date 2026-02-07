package gui

// LayoutType defines the direction of a layout.
type LayoutType uint8

const (
	LayoutVertical   LayoutType = iota // Items stack vertically (default)
	LayoutHorizontal                   // Items stack horizontally
)

// Layout tracks the current layout state.
type Layout struct {
	Type LayoutType

	// Position tracking
	StartX, StartY   float32
	CursorX, CursorY float32

	// Sizing
	Width, Height       float32 // Available size
	MaxWidth, MaxHeight float32 // Accumulated content size

	// Spacing (Tailwind-style)
	Gap      float32 // Space between children (gap-*)
	GapX     float32 // Horizontal gap override
	GapY     float32 // Vertical gap override
	Padding  float32 // Inner padding (p-*)
	PaddingX float32 // Horizontal padding override
	PaddingY float32 // Vertical padding override

	// Alignment
	Align   Alignment     // Cross-axis (items-*)
	Justify Justification // Main-axis (justify-*)

	// State
	ItemCount int // For gap calculation

	// Panel-specific options
	Hotkey           string  // Keyboard shortcut to display (e.g., "T" -> "Title [T]")
	HeightConstraint float32 // Maximum height constraint (0 = no limit, > 0 = limit)
}

// Alignment values (like Tailwind items-*)
type Alignment uint8

const (
	AlignStart   Alignment = iota // items-start
	AlignCenter                   // items-center
	AlignEnd                      // items-end
	AlignStretch                  // items-stretch (default)
)

// Justification values (like Tailwind justify-*)
type Justification uint8

const (
	JustifyStart   Justification = iota // justify-start (default)
	JustifyCenter                       // justify-center
	JustifyEnd                          // justify-end
	JustifyBetween                      // justify-between
)

// LayoutOption configures a layout container.
type LayoutOption func(*Layout)

// Gap sets spacing between children (like Tailwind gap-*).
func Gap(pixels float32) LayoutOption {
	return func(l *Layout) { l.Gap = pixels }
}

// GapX sets horizontal spacing (like Tailwind gap-x-*).
func GapX(pixels float32) LayoutOption {
	return func(l *Layout) { l.GapX = pixels }
}

// GapY sets vertical spacing (like Tailwind gap-y-*).
func GapY(pixels float32) LayoutOption {
	return func(l *Layout) { l.GapY = pixels }
}

// Padding sets inner padding (like Tailwind p-*).
func Padding(pixels float32) LayoutOption {
	return func(l *Layout) { l.Padding = pixels }
}

// PaddingXY sets horizontal and vertical padding separately.
func PaddingXY(x, y float32) LayoutOption {
	return func(l *Layout) {
		l.PaddingX = x
		l.PaddingY = y
	}
}

// Align sets cross-axis alignment (like Tailwind items-*).
func Align(a Alignment) LayoutOption {
	return func(l *Layout) { l.Align = a }
}

// Justify sets main-axis alignment (like Tailwind justify-*).
func Justify(j Justification) LayoutOption {
	return func(l *Layout) { l.Justify = j }
}

// Width sets a fixed width for the layout.
func Width(w float32) LayoutOption {
	return func(l *Layout) { l.Width = w }
}

// Height sets a fixed height for the layout.
func Height(h float32) LayoutOption {
	return func(l *Layout) { l.Height = h }
}

// WithHotkey sets the keyboard shortcut to display in panel headers.
// The hotkey is shown as "[Key]" after the title.
func WithHotkey(key string) LayoutOption {
	return func(l *Layout) { l.Hotkey = key }
}

// MaxHeight sets a maximum height constraint for the panel.
// If content exceeds this, it will be clipped (use with Scrollable).
// Pass 0 to disable the constraint.
func MaxHeight(h float32) LayoutOption {
	return func(l *Layout) { l.HeightConstraint = h }
}

// pushLayout creates and pushes a new layout onto the stack.
func (ctx *Context) pushLayout(layoutType LayoutType) *Layout {
	layout := &Layout{
		Type:   layoutType,
		StartX: ctx.cursor.X,
		StartY: ctx.cursor.Y,
		Width:  ctx.currentLayoutWidth(),
		Height: ctx.currentLayoutHeight(),
	}
	ctx.layoutStack = append(ctx.layoutStack, layout)
	return layout
}

// pushLayoutWith creates a layout with options and pushes it.
func (ctx *Context) pushLayoutWith(layout *Layout) {
	layout.StartX = ctx.cursor.X
	layout.StartY = ctx.cursor.Y
	if layout.Width == 0 {
		layout.Width = ctx.currentLayoutWidth()
	}
	if layout.Height == 0 {
		layout.Height = ctx.currentLayoutHeight()
	}
	ctx.layoutStack = append(ctx.layoutStack, layout)
}

// popLayout removes and returns the current layout's bounds.
func (ctx *Context) popLayout() Rect {
	n := len(ctx.layoutStack)
	if n == 0 {
		return Rect{}
	}

	layout := ctx.layoutStack[n-1]
	ctx.layoutStack = ctx.layoutStack[:n-1]

	bounds := Rect{
		X: layout.StartX,
		Y: layout.StartY,
		W: layout.MaxWidth,
		H: layout.MaxHeight,
	}

	// Update parent layout to include this child's content size
	if len(ctx.layoutStack) > 0 {
		parent := ctx.layoutStack[len(ctx.layoutStack)-1]

		// Treat the popped layout as a single item in the parent
		childSize := Vec2{X: layout.MaxWidth, Y: layout.MaxHeight}

		// Add gap before this item if not first
		if parent.ItemCount > 0 {
			if parent.Type == LayoutVertical {
				gap := parent.GapY
				if gap == 0 {
					gap = parent.Gap
				}
				if gap == 0 {
					gap = ctx.style.ItemSpacing
				}
				ctx.cursor.Y += gap
			} else {
				gap := parent.GapX
				if gap == 0 {
					gap = parent.Gap
				}
				if gap == 0 {
					gap = ctx.style.ItemSpacing
				}
				ctx.cursor.X += gap
			}
		}

		// Update parent's content tracking
		if parent.Type == LayoutVertical {
			ctx.cursor.X = parent.StartX + parent.Padding + parent.PaddingX
			ctx.cursor.Y = layout.StartY + layout.MaxHeight
			parent.MaxWidth = maxf(parent.MaxWidth, childSize.X)
			parent.MaxHeight = ctx.cursor.Y - parent.StartY
		} else {
			ctx.cursor.X = layout.StartX + layout.MaxWidth
			ctx.cursor.Y = parent.StartY + parent.Padding + parent.PaddingY
			parent.MaxWidth = ctx.cursor.X - parent.StartX
			parent.MaxHeight = maxf(parent.MaxHeight, childSize.Y)
		}

		parent.ItemCount++
	}

	return bounds
}

// Panel draws a panel with a title and content.
// Returns a function that should be called with the content closure.
//
// Usage:
//
//	ctx.Panel("Menu", Gap(8), Padding(12))(func() {
//	    ctx.Text("Hello")
//	    ctx.Button("Click")
//	})
//
//	// With hotkey display:
//	ctx.Panel("Model Menu", WithHotkey("T"))(func() {
//	    // Renders header as "Model Menu [T]"
//	})
func (ctx *Context) Panel(title string, opts ...LayoutOption) func(func()) {
	return func(contents func()) {
		// Create layout with defaults
		layout := &Layout{
			Type:    LayoutVertical,
			Padding: ctx.style.PanelPadding,
			Gap:     ctx.style.ItemSpacing,
		}

		// Apply options
		for _, opt := range opts {
			opt(layout)
		}

		// Calculate effective padding
		padX := layout.PaddingX
		if padX == 0 {
			padX = layout.Padding
		}
		padY := layout.PaddingY
		if padY == 0 {
			padY = layout.Padding
		}

		// Save user-specified size BEFORE pushLayoutWith auto-fills them
		// (0 means auto-size to content, don't enforce minimum)
		userWidth := layout.Width
		userHeight := layout.Height

		// Save start position
		startX := ctx.cursor.X
		startY := ctx.cursor.Y

		// Calculate header height if we have a title
		headerH := float32(0)
		if title != "" {
			headerH = ctx.lineHeight() + padY*2
		}

		// Apply padding to cursor (after header)
		ctx.cursor.X += padX
		ctx.cursor.Y += padY + headerH

		// Push layout (this may auto-fill Width/Height to display size)
		ctx.pushLayoutWith(layout)

		// Execute contents (title is drawn separately in the header)
		contents()

		// Pop layout and get bounds
		bounds := ctx.popLayout()

		// Calculate panel size including padding and header
		panelW := bounds.W + padX*2
		panelH := bounds.H + padY*2 + headerH

		// Ensure minimum size only if user explicitly specified dimensions
		// (userWidth/userHeight are 0 if not specified, meaning auto-size)
		if userWidth > 0 && panelW < userWidth {
			panelW = userWidth
		}
		if userHeight > 0 && panelH < userHeight {
			panelH = userHeight
		}

		// Apply maximum height constraint (if specified)
		// This prevents panels from growing beyond a certain size
		if layout.HeightConstraint > 0 && panelH > layout.HeightConstraint {
			panelH = layout.HeightConstraint
		}

		// Insert background (drawn first, behind content)
		ctx.DrawList.InsertRect(startX, startY, panelW, panelH, ctx.style.PanelColor)

		// Draw header background and title if provided
		if title != "" {
			// Header background
			headerBg := ctx.style.PanelHeaderBgColor
			if headerBg == 0 {
				headerBg = ctx.style.ButtonColor
			}
			ctx.DrawList.AddRect(startX, startY, panelW, headerH, headerBg)

			// Header text color
			headerTextColor := ctx.style.PanelHeaderTextColor
			if headerTextColor == 0 {
				headerTextColor = ctx.style.TextColor
			}

			// Format title with hotkey if provided
			displayTitle := title
			if layout.Hotkey != "" {
				displayTitle = title + " [" + layout.Hotkey + "]"
			}

			// Center title vertically in header
			textY := startY + (headerH-ctx.lineHeight())/2
			ctx.addText(startX+padX, textY, displayTitle, headerTextColor)
		}

		// Draw border if style has one
		if ctx.style.BorderSize > 0 {
			ctx.DrawList.AddRectOutline(startX, startY, panelW, panelH,
				ctx.style.PanelBorderColor, ctx.style.BorderSize)
		}

		// Check if mouse is inside panel and set capture flag
		if ctx.Input != nil {
			panelRect := Rect{X: startX, Y: startY, W: panelW, H: panelH}
			if panelRect.Contains(Vec2{ctx.Input.MouseX, ctx.Input.MouseY}) {
				ctx.WantCaptureMouse = true
			}
		}

		// Update cursor for next element
		ctx.cursor.X = startX
		ctx.cursor.Y = startY + panelH
	}
}

// CenteredPanel draws a panel centered on screen.
// Uses cached size from previous frame for accurate centering.
//
// This fixes ImGui's "can't center without knowing size" flaw.
func (ctx *Context) CenteredPanel(id string, opts ...LayoutOption) func(func()) {
	return func(contents func()) {
		panelID := ctx.GetID(id)

		// Get cached size from previous frame (or default)
		cachedSize := GetState(ctx, panelID, Vec2{200, 100})

		// Calculate centered position
		x := (ctx.DisplaySize.X - cachedSize.X) / 2
		y := (ctx.DisplaySize.Y - cachedSize.Y) / 2

		// Position cursor
		ctx.cursor.X = x
		ctx.cursor.Y = y

		startY := ctx.cursor.Y

		// Draw panel
		ctx.Panel("", opts...)(contents)

		// Store measured size for next frame
		measuredSize := Vec2{
			X: ctx.currentLayoutWidth(),
			Y: ctx.cursor.Y - startY,
		}
		SetState(ctx, panelID, measuredSize)
	}
}

// VStack creates a vertical layout container.
//
// Usage:
//
//	ctx.VStack(Gap(8))(func() {
//	    ctx.Text("Line 1")
//	    ctx.Text("Line 2")
//	})
func (ctx *Context) VStack(opts ...LayoutOption) func(func()) {
	return func(contents func()) {
		layout := &Layout{Type: LayoutVertical, Gap: ctx.style.ItemSpacing}
		for _, opt := range opts {
			opt(layout)
		}
		ctx.pushLayoutWith(layout)
		contents()
		ctx.popLayout()
	}
}

// HStack creates a horizontal layout container.
//
// Usage:
//
//	ctx.HStack(Gap(8))(func() {
//	    ctx.Text("Label:")
//	    ctx.InputText("", &value)
//	})
func (ctx *Context) HStack(opts ...LayoutOption) func(func()) {
	return func(contents func()) {
		layout := &Layout{Type: LayoutHorizontal, Gap: ctx.style.ItemSpacing}
		for _, opt := range opts {
			opt(layout)
		}
		ctx.pushLayoutWith(layout)
		contents()
		ctx.popLayout()
	}
}

// Row creates a horizontal layout for its contents (alias for HStack).
func (ctx *Context) Row(contents func()) {
	ctx.HStack()(contents)
}

// Spacing adds vertical space.
func (ctx *Context) Spacing(pixels float32) {
	ctx.cursor.Y += pixels
}

// Separator draws a horizontal line.
func (ctx *Context) Separator() {
	w := ctx.currentLayoutWidth()
	y := ctx.cursor.Y + 2
	ctx.DrawList.AddLine(ctx.cursor.X, y, ctx.cursor.X+w, y, ctx.style.SeparatorColor, 1)
	ctx.cursor.Y += 4
}

// ListBox draws a scrollable list area with smooth scrolling.
// height specifies the visible height; contents can be larger.
//
// Usage:
//
//	ctx.ListBox("items", 200, Gap(4))(func() {
//	    for i, item := range items {
//	        ctx.Selectable(item.Name, i == selected, WithID(item.ID))
//	    }
//	})
func (ctx *Context) ListBox(id string, height float32, opts ...LayoutOption) func(func()) {
	return func(contents func()) {
		// Get/create scroll state
		scrollID := ctx.GetID(id + "_scroll")
		scrollState := GetState(ctx, scrollID, ScrollState{})

		// Update smooth scrolling
		scrollState.UpdateSmooth(ctx.DeltaTime)

		// Save position
		x, y := ctx.cursor.X, ctx.cursor.Y
		w := ctx.currentLayoutWidth()

		// Push clip rect for visible area
		ctx.DrawList.PushClipRect(x, y, x+w, y+height)

		// Offset cursor by scroll (use current position, not target)
		ctx.cursor.Y -= scrollState.ScrollY

		// Create layout for contents
		layout := &Layout{
			Type:   LayoutVertical,
			Width:  w - ctx.style.ScrollbarSize,
			Height: height,
			Gap:    ctx.style.ItemSpacing,
		}
		for _, opt := range opts {
			opt(layout)
		}
		ctx.pushLayoutWith(layout)

		// Execute contents
		contents()

		// Pop layout and get content size
		bounds := ctx.popLayout()
		contentHeight := bounds.H

		// Pop clip rect
		ctx.DrawList.PopClipRect()

		// Handle scroll input (update target for smooth scrolling)
		if ctx.Input != nil && ctx.Input.MouseWheelY != 0 {
			mouseRect := Rect{X: x, Y: y, W: w, H: height}
			if mouseRect.Contains(Vec2{ctx.Input.MouseX, ctx.Input.MouseY}) {
				maxScroll := maxf(0, contentHeight-height)
				newTarget := clampf(scrollState.TargetScrollY-ctx.Input.MouseWheelY*30, 0, maxScroll)
				scrollState.TargetScrollY = newTarget
				scrollState.ContentHeight = contentHeight
			}
		}

		// Save state
		SetState(ctx, scrollID, scrollState)

		// Draw scrollbar if content exceeds height
		if contentHeight > height {
			scrollbarX := x + w - ctx.style.ScrollbarSize
			scrollRatio := height / contentHeight
			scrollbarHeight := maxf(20, height*scrollRatio)
			maxScroll := contentHeight - height
			scrollPos := (scrollState.ScrollY / maxScroll) * (height - scrollbarHeight)

			// Scrollbar background
			ctx.DrawList.AddRect(scrollbarX, y, ctx.style.ScrollbarSize, height,
				ctx.style.ScrollbarBgColor)

			// Scrollbar thumb
			ctx.DrawList.AddRect(scrollbarX, y+scrollPos, ctx.style.ScrollbarSize, scrollbarHeight,
				ctx.style.ScrollbarGrabColor)
		}

		// Restore cursor position after list
		ctx.cursor.X = x
		ctx.cursor.Y = y + height
	}
}

// SameLine places the next widget on the same line as the previous.
func (ctx *Context) SameLine() {
	if layout := ctx.currentLayout(); layout != nil {
		// Move cursor back to previous line
		ctx.cursor.Y -= ctx.lineHeight() + layout.Gap
		// Add horizontal spacing
		ctx.cursor.X += ctx.style.ItemSpacing
	}
}

// Indent increases the cursor X position.
func (ctx *Context) Indent(pixels float32) {
	ctx.cursor.X += pixels
}

// Unindent decreases the cursor X position.
func (ctx *Context) Unindent(pixels float32) {
	ctx.cursor.X -= pixels
}
