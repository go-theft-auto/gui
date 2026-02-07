package gui

import "strings"

// listStore is the type-safe store for list state.
// Uses the new FrameStore pattern instead of the old GetState/SetState.
var listStore = NewFrameStore[ListState]()

// ListBuilder provides a fluent API for building list components.
type ListBuilder struct {
	ctx             *Context
	id              string
	height          float32
	opts            options
	state           *ListState
	scrollID        ID
	itemIndex       int
	selectedItem    int
	onSelectChanged func(int)
}

// SectionBuilder provides a fluent API for building list sections.
type SectionBuilder struct {
	list    *ListBuilder
	name    string
	isOpen  bool
	opts    options
	started bool
}

// List creates a new list component with search, collapsible sections, and nested widgets.
// Returns a ListBuilder for fluent configuration.
//
// Usage:
//
//	list := ctx.List("objects", 400, ShowScrollbar(true), WithFilter("Search..."))
//
//	list.Section("Vehicles", DefaultOpen()).
//	    Item("Infernus", selected == 1).
//	    ItemFunc("Custom", selected == 2, func() {
//	        ctx.SliderFloat("Speed", &speed, 0, 100)
//	    }).
//	    End()
//
//	list.Section("Settings").
//	    ItemFunc("Scale", false, func() {
//	        ctx.HStack()(func() {
//	            ctx.NumberInputFloat("", &x, WithPrefix("X:"))
//	        })
//	    }).
//	    End()
//
//	list.End()
func (ctx *Context) List(id string, height float32, opts ...Option) *ListBuilder {
	o := applyOptions(opts)

	scrollID := ctx.GetID(id + "_list")
	// Use the new type-safe store instead of GetState
	state := listStore.Get(scrollID, ListState{
		CollapsedSections: make(map[string]bool),
		SelectedIndex:     -1,
	})

	builder := &ListBuilder{
		ctx:          ctx,
		id:           id,
		height:       height,
		opts:         o,
		state:        state,
		scrollID:     scrollID,
		selectedItem: -1,
	}

	// Start the list container
	builder.begin()

	return builder
}

// begin starts the list rendering.
func (lb *ListBuilder) begin() {
	ctx := lb.ctx
	pos := ctx.ItemPos()
	w := ctx.currentLayoutWidth()
	if width := GetOpt(lb.opts, OptWidth); width > 0 {
		w = width
	}

	// Draw list background
	ctx.DrawList.AddRect(pos.X, pos.Y, w, lb.height, ctx.style.InputBgColor)
	ctx.DrawList.AddRectOutline(pos.X, pos.Y, w, lb.height, ctx.style.InputBorderColor, 1)

	// Calculate content area
	contentY := pos.Y
	filterHeight := float32(0)

	// Draw filter input if enabled
	if filterPlaceholder := GetOpt(lb.opts, OptFilterPlaceholder); filterPlaceholder != "" {
		filterHeight = ctx.lineHeight() + ctx.style.InputPadding*2 + ctx.style.ItemSpacing

		// Draw filter background
		filterRect := Rect{
			X: pos.X + 2,
			Y: pos.Y + 2,
			W: w - 4,
			H: ctx.lineHeight() + ctx.style.InputPadding*2,
		}
		ctx.DrawList.AddRect(filterRect.X, filterRect.Y, filterRect.W, filterRect.H, ctx.style.InputBgColor)
		ctx.DrawList.AddRectOutline(filterRect.X, filterRect.Y, filterRect.W, filterRect.H, ctx.style.InputBorderColor, 1)

		// Handle filter input
		filterID := ctx.GetID(lb.id + "_filter")

		// Register filter as focusable for keyboard navigation
		focusable := ctx.RegisterFocusable(filterID, "filter", filterRect, FocusTypeLeaf)
		isRegistryFocused := focusable != nil && focusable.IsFocused()

		// Enter to start editing when registry-focused but not in edit mode
		if isRegistryFocused && !lb.state.FilterEditing && ctx.Input != nil && ctx.Input.KeyPressed(KeyEnter) {
			lb.state.FilterEditing = true
		}

		if ctx.Input != nil {
			// Click to enter edit mode (RegisterFocusable handles setting registry focus)
			if ctx.isClicked(filterID, filterRect) {
				lb.state.FilterEditing = true
			}

			// Text input when in edit mode
			if lb.state.FilterEditing {
				ctx.WantCaptureKeyboard = true
				for _, ch := range ctx.Input.InputChars {
					if ch >= 32 && ch < 127 {
						lb.state.SearchText += string(ch)
					}
				}
				if ctx.Input.KeyRepeated(KeyBackspace) && len(lb.state.SearchText) > 0 {
					lb.state.SearchText = lb.state.SearchText[:len(lb.state.SearchText)-1]
				}
				if ctx.Input.KeyPressed(KeyEscape) || ctx.Input.KeyPressed(KeyEnter) {
					lb.state.FilterEditing = false
				}
			}

			// Exit edit mode if registry focus moved away
			if lb.state.FilterEditing && !isRegistryFocused {
				lb.state.FilterEditing = false
			}
		}

		// Draw filter text
		textX := filterRect.X + ctx.style.InputPadding
		textY := filterRect.Y + ctx.style.InputPadding
		if lb.state.SearchText != "" {
			ctx.addText(textX, textY, lb.state.SearchText, ctx.style.TextColor)
			// Draw cursor when in edit mode
			if lb.state.FilterEditing && (ctx.FrameCount/30)%2 == 0 {
				cursorX := textX + ctx.MeasureText(lb.state.SearchText).X
				ctx.DrawList.AddLine(cursorX, filterRect.Y+2, cursorX, filterRect.Y+filterRect.H-2, ctx.style.TextColor, 1)
			}
		} else {
			ctx.addText(textX, textY, filterPlaceholder, ctx.style.TextDisabledColor)
		}

		contentY += filterHeight
	}

	// Push clip rect for scrollable content
	scrollableHeight := lb.height - filterHeight
	ctx.DrawList.PushClipRect(pos.X, contentY, pos.X+w, contentY+scrollableHeight)

	// Set up cursor for content
	ctx.cursor.X = pos.X + 2
	ctx.cursor.Y = contentY + 2 - lb.state.ScrollY
}

// Section starts a new collapsible section.
func (lb *ListBuilder) Section(name string, opts ...Option) *SectionBuilder {
	o := applyOptions(opts)

	// Check if section should be collapsed
	isCollapsed, exists := lb.state.CollapsedSections[name]
	if !exists && GetOpt(o, OptDefaultOpen) {
		isCollapsed = false
	}
	isOpen := !isCollapsed

	return &SectionBuilder{
		list:   lb,
		name:   name,
		isOpen: isOpen,
		opts:   o,
	}
}

// Item adds a simple selectable item to the section.
// Returns the SectionBuilder for chaining.
func (sb *SectionBuilder) Item(label string, selected bool) *SectionBuilder {
	if !sb.started {
		sb.drawHeader()
		sb.started = true
	}

	if !sb.isOpen {
		return sb
	}

	// Apply filter
	if sb.list.state.SearchText != "" {
		searchLower := strings.ToLower(sb.list.state.SearchText)
		if !strings.Contains(strings.ToLower(label), searchLower) {
			return sb
		}
	}

	ctx := sb.list.ctx
	lb := sb.list

	lb.itemIndex++
	itemID := ctx.GetID(lb.id + "_item_" + label)

	// Calculate item rect
	x := ctx.cursor.X + ctx.style.ItemSpacing*2 // Indent
	y := ctx.cursor.Y
	w := ctx.currentLayoutWidth() - ctx.style.ItemSpacing*4
	h := ctx.lineHeight()

	rect := Rect{X: x, Y: y, W: w, H: h}

	// Register as focusable (auto-draws debug rect if registry-focused)
	ctx.RegisterFocusable(itemID, label, rect, FocusTypeLeaf)
	registryFocused := ctx.IsRegistryFocused(itemID)

	// Draw background
	var bgColor uint32
	hovered := ctx.isHovered(itemID, rect)
	if selected || registryFocused {
		bgColor = ctx.style.SelectedBgColor
	} else if hovered {
		bgColor = ctx.style.HoveredBgColor
	}
	if bgColor != 0 {
		ctx.DrawList.AddRect(x, y, w, h, bgColor)
	}
	// Debug focus highlighting for selected list items (not registry-focused)
	if selected && !registryFocused {
		ctx.DrawDebugFocusRectIf(selected, x, y, w, h)
	}

	// Draw text
	textColor := ctx.style.TextColor
	if selected || registryFocused {
		textColor = ctx.style.SelectedTextColor
	}
	ctx.addText(x+ctx.style.ItemSpacing, y, label, textColor)

	// Handle click
	if ctx.isClicked(itemID, rect) {
		lb.selectedItem = lb.itemIndex
	}

	// Advance cursor
	ctx.cursor.Y += h + ctx.style.ItemSpacing

	return sb
}

// ItemFunc adds an item with custom widget content.
// Returns the SectionBuilder for chaining.
func (sb *SectionBuilder) ItemFunc(label string, selected bool, content func()) *SectionBuilder {
	if !sb.started {
		sb.drawHeader()
		sb.started = true
	}

	if !sb.isOpen {
		return sb
	}

	// Apply filter
	if sb.list.state.SearchText != "" {
		searchLower := strings.ToLower(sb.list.state.SearchText)
		if !strings.Contains(strings.ToLower(label), searchLower) {
			return sb
		}
	}

	ctx := sb.list.ctx
	lb := sb.list

	lb.itemIndex++
	itemID := ctx.GetID(lb.id + "_item_" + label)

	// Draw item header with expansion capability
	x := ctx.cursor.X + ctx.style.ItemSpacing*2 // Indent
	y := ctx.cursor.Y
	w := ctx.currentLayoutWidth() - ctx.style.ItemSpacing*4
	h := ctx.lineHeight()

	rect := Rect{X: x, Y: y, W: w, H: h}

	// Register as focusable (auto-draws debug rect if registry-focused)
	ctx.RegisterFocusable(itemID, label, rect, FocusTypeLeaf)
	registryFocused := ctx.IsRegistryFocused(itemID)

	// Draw item label
	textColor := ctx.style.TextColor
	if selected || registryFocused {
		ctx.DrawList.AddRect(x, y, w, h, ctx.style.SelectedBgColor)
		// Debug focus highlighting for selected items (not registry-focused)
		if selected && !registryFocused {
			ctx.DrawDebugFocusRect(x, y, w, h)
		}
		textColor = ctx.style.SelectedTextColor
	}
	ctx.addText(x+ctx.style.ItemSpacing, y, label, textColor)

	ctx.cursor.Y += h + ctx.style.ItemSpacing

	// Draw nested content with additional indent
	ctx.cursor.X += ctx.style.ItemSpacing*2 + ctx.style.ItemSpacing*2
	content()
	ctx.cursor.X -= ctx.style.ItemSpacing*2 + ctx.style.ItemSpacing*2

	ctx.cursor.Y += ctx.style.ItemSpacing

	return sb
}

// End finishes the section.
func (sb *SectionBuilder) End() *ListBuilder {
	if !sb.started {
		sb.drawHeader()
	}
	return sb.list
}

// drawHeader draws the section header.
func (sb *SectionBuilder) drawHeader() {
	ctx := sb.list.ctx
	lb := sb.list

	// Calculate header rect
	x := ctx.cursor.X
	y := ctx.cursor.Y
	w := ctx.currentLayoutWidth() - 4
	h := ctx.lineHeight()

	headerID := ctx.GetID(lb.id + "_section_" + sb.name)
	rect := Rect{X: x, Y: y, W: w, H: h}

	// Draw header background
	bgColor := ctx.style.HeaderBgColor
	if ctx.isHovered(headerID, rect) {
		bgColor = ctx.style.ButtonHoveredColor
	}
	ctx.DrawList.AddRect(x, y, w, h, bgColor)

	// Draw collapse indicator
	indicator := "v"
	if !sb.isOpen {
		indicator = ">"
	}
	ctx.addText(x+ctx.style.ItemSpacing, y, indicator, ctx.style.TextColor)

	// Draw section name
	ctx.addText(x+ctx.style.ItemSpacing*3, y, sb.name, ctx.style.TextColor)

	// Handle click to toggle collapse
	if ctx.isClicked(headerID, rect) {
		sb.isOpen = !sb.isOpen
		lb.state.CollapsedSections[sb.name] = !sb.isOpen
	}

	// Advance cursor
	ctx.cursor.Y += h + ctx.style.ItemSpacing
}

// End finishes the list and handles cleanup.
func (lb *ListBuilder) End() int {
	ctx := lb.ctx

	// Pop clip rect
	ctx.DrawList.PopClipRect()

	// Calculate content bounds for scrolling
	pos := ctx.GetCursorPos()
	w := ctx.currentLayoutWidth()
	if width := GetOpt(lb.opts, OptWidth); width > 0 {
		w = width
	}

	// Calculate total content height (rough estimate based on cursor position)
	contentHeight := ctx.cursor.Y + lb.state.ScrollY - pos.Y

	// Handle scroll input
	listRect := Rect{X: pos.X, Y: pos.Y - lb.height, W: w, H: lb.height}
	if ctx.Input != nil && ctx.isHovered(lb.scrollID, listRect) {
		if ctx.Input.MouseWheelY != 0 {
			maxScroll := maxf(0, contentHeight-lb.height)
			lb.state.ScrollY = clampf(lb.state.ScrollY-ctx.Input.MouseWheelY*30, 0, maxScroll)
		}
	}

	// Draw scrollbar if needed
	filterHeight := float32(0)
	if GetOpt(lb.opts, OptFilterPlaceholder) != "" {
		filterHeight = ctx.lineHeight() + ctx.style.InputPadding*2 + ctx.style.ItemSpacing
	}
	scrollableHeight := lb.height - filterHeight

	if contentHeight > scrollableHeight {
		scrollbarWidth := ctx.style.ScrollbarSize
		scrollbarX := pos.X + w - scrollbarWidth - 2

		scrollRatio := scrollableHeight / contentHeight
		thumbHeight := maxf(20, scrollableHeight*scrollRatio)
		maxScroll := contentHeight - scrollableHeight
		thumbPos := float32(0)
		if maxScroll > 0 {
			thumbPos = (lb.state.ScrollY / maxScroll) * (scrollableHeight - thumbHeight)
		}

		// Scrollbar background
		ctx.DrawList.AddRect(scrollbarX, pos.Y-lb.height+filterHeight, scrollbarWidth, scrollableHeight, ctx.style.ScrollbarBgColor)

		// Scrollbar thumb
		thumbY := pos.Y - lb.height + filterHeight + thumbPos
		ctx.DrawList.AddRect(scrollbarX, thumbY, scrollbarWidth, thumbHeight, ctx.style.ScrollbarGrabColor)
	}

	// State is automatically saved via pointer (no need to call SetState)

	// Restore cursor position
	ctx.cursor.X = pos.X
	ctx.cursor.Y = pos.Y - lb.height + lb.height + ctx.style.ItemSpacing

	// Advance cursor for the list
	ctx.advanceCursor(Vec2{w, lb.height})

	return lb.selectedItem
}

// Selected returns the index of the item that was clicked this frame, or -1.
func (lb *ListBuilder) Selected() int {
	return lb.selectedItem
}

// OnSelect sets a callback for when an item is selected.
func (lb *ListBuilder) OnSelect(callback func(int)) *ListBuilder {
	lb.onSelectChanged = callback
	return lb
}
