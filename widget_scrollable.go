package gui

// scrollableStore is the type-safe store for scrollable state.
// Uses the new FrameStore pattern instead of the old GetState/SetState.
var scrollableStore = NewFrameStore[ScrollableState]()

// EnsureScrollVisible scrolls a Scrollable to keep the given Y position visible.
// Call this when selection changes to auto-scroll to the selected item.
//
// Parameters:
//   - ctx: GUI context
//   - scrollID: The ID used when creating the Scrollable (e.g., "my_scroll")
//   - targetY: Y position relative to scrollable content (e.g., rowIndex * rowHeight)
//   - viewportHeight: Height of the scrollable viewport
//   - padding: Extra padding around target (e.g., rowHeight for one row margin)
//
// Usage:
//
//	// When selection changes via keyboard:
//	if selectionChanged {
//	    targetY := float32(selectedIndex) * rowHeight
//	    gui.EnsureScrollVisible(ctx, "items_scroll", targetY, scrollHeight, rowHeight)
//	}
func EnsureScrollVisible(ctx *Context, scrollID string, targetY, viewportHeight, padding float32) {
	// Look up the stored ID from the name->ID map (set when Scrollable is rendered)
	fullName := scrollID + "_scrollable"
	storedID, ok := scrollableNameToID[fullName]
	if !ok {
		// Scrollable hasn't been rendered yet, nothing to scroll
		return
	}
	state := scrollableStore.Get(storedID, ScrollableState{})

	maxScroll := maxf(0, state.ContentHeight-viewportHeight)

	// Check if target is outside visible area
	visibleTop := state.ScrollY + padding
	visibleBottom := state.ScrollY + viewportHeight - padding

	if targetY < visibleTop {
		// Target is above visible area - scroll up
		state.ScrollY = clampf(targetY-padding, 0, maxScroll)
	} else if targetY > visibleBottom {
		// Target is below visible area - scroll down
		state.ScrollY = clampf(targetY-viewportHeight+padding, 0, maxScroll)
	}
}

// Scrollable creates a scrollable area that can wrap any content.
// Returns a function that should be called with the content closure.
//
// Usage:
//
//	ctx.Scrollable("my_scroll", 300, ShowScrollbar(true))(func() {
//	    // Any widgets here become scrollable
//	    ctx.Text("Line 1")
//	    ctx.Text("Line 2")
//	    ctx.Button("Click me")
//	    ctx.SliderFloat("Volume", &vol, 0, 1)
//	    // ... unlimited content
//	})
func (ctx *Context) Scrollable(id string, height float32, opts ...Option) func(func()) {
	return func(contents func()) {
		o := applyOptions(opts)

		// Get/create scroll state using the new type-safe store
		scrollID := ctx.GetID(id + "_scrollable")
		// Initialize with UserScrollTime > cooldown so auto-scroll works immediately
		// The cooldown only applies AFTER the user has manually scrolled
		state := scrollableStore.Get(scrollID, ScrollableState{UserScrollTime: 1.0})

		// Register name -> ID mapping for GetScrollableState lookup
		scrollableNameToID[id+"_scrollable"] = scrollID

		// Save position BEFORE pushing scrollable (needed for contentOrigin calculation)
		x, y := ctx.cursor.X, ctx.cursor.Y
		w := ctx.currentLayoutWidth()
		if width := GetOpt(o, OptWidth); width > 0 {
			w = width
		}

		// Determine if scrollbar should be shown
		scrollbarVisibility := GetOpt(o, OptScrollbarVisibility)
		showScrollbar := scrollbarVisibility == ScrollbarAlways ||
			(scrollbarVisibility == ScrollbarAuto && state.ContentHeight > height)

		scrollbarWidth := float32(0)
		if showScrollbar {
			scrollbarWidth = ctx.style.ScrollbarSize
		}

		// Content area width (account for scrollbar on appropriate side)
		contentWidth := w - scrollbarWidth

		// Calculate scrollbar position based on side
		scrollbarX := x + w - scrollbarWidth
		contentX := x
		if GetOpt(o, OptScrollbarSide) == ScrollbarLeft {
			scrollbarX = x
			contentX = x + scrollbarWidth
		}

		// Begin focus scope for hierarchical focus tracking
		viewportRect := Rect{X: x, Y: y, W: w, H: height}
		ctx.BeginFocusScope(scrollID, id, FocusTypeContainer, viewportRect)

		// Push clip rect for visible area
		ctx.DrawList.PushClipRect(contentX, y, contentX+contentWidth, y+height)

		// Offset cursor by scroll
		ctx.cursor.X = contentX
		ctx.cursor.Y = y - state.ScrollY
		horizontalScroll := GetOpt(o, OptHorizontalScroll)
		if horizontalScroll {
			ctx.cursor.X -= state.ScrollX
		}

		// Push this scrollable onto the stack so children can call ctx.ScrollTo()
		// contentOriginY is cursor.Y AFTER scroll offset - this enables automatic
		// coordinate translation from screen position to content-relative position.
		// viewportY (y) and height enable visibility checking for click detection.
		contentOriginY := ctx.cursor.Y
		ctx.pushScrollable(scrollID, contentOriginY, y, height)

		// Track scroll focus from children - capture variables for processing after contents()
		var scrollFocus struct {
			y, padding float32
			ok         bool
		}

		// Use defer to ensure stack cleanup even if contents() panics
		// This prevents scrollableStack corruption in edge cases
		stackPopped := false
		defer func() {
			if !stackPopped {
				ctx.popScrollable() // Cleanup on panic
			}
		}()

		// Create layout for contents
		layout := &Layout{
			Type:   LayoutVertical,
			Width:  contentWidth,
			Height: height,
			Gap:    ctx.style.ItemSpacing,
		}
		ctx.pushLayoutWith(layout)

		// Execute contents
		contents()

		// Pop layout and get content size
		bounds := ctx.popLayout()
		state.ContentHeight = bounds.H
		state.ContentWidth = bounds.W

		// Pop clip rect
		ctx.DrawList.PopClipRect()

		// Pop scrollable from stack and get any focus set by children via ctx.ScrollTo()
		scrollFocus.y, scrollFocus.padding, scrollFocus.ok = ctx.popScrollable()
		stackPopped = true

		// End focus scope and get child focus info
		focusInfo := ctx.EndFocusScope()

		// Handle auto-scroll from focus hierarchy (new system)
		// Auto-scroll is enabled by default. Set keyboardNavigated=false to disable.
		// The user scroll cooldown (300ms) prevents fighting with manual scrolling.
		keyboardNav := ctx.FocusRegistry() == nil || ctx.FocusRegistry().WasKeyboardNavigated()

		// Debug: log auto-scroll decision
		if focusInfo.HasFocusedChild {
			cooldownExpired := state.UserScrollTime >= 0.3
			focusChanged := !state.FocusYSet || focusInfo.FocusedChildY != state.LastFocusY
			guiLogger.Debug("Scrollable auto-scroll check",
				"id", id,
				"focusY", focusInfo.FocusedChildY,
				"lastFocusY", state.LastFocusY,
				"focusYSet", state.FocusYSet,
				"focusChanged", focusChanged,
				"cooldownExpired", cooldownExpired,
				"scrollY", state.ScrollY,
				"height", height)
		}

		if focusInfo.HasFocusedChild && keyboardNav {
			// User scroll cooldown: suppress auto-scroll for 300ms after manual scrolling
			const userScrollCooldown = 0.3 // 300ms
			cooldownExpired := state.UserScrollTime >= userScrollCooldown

			// Only auto-scroll if:
			// 1. Cooldown has expired (user hasn't scrolled recently)
			// 2. Focus position has changed from last frame (prevents fighting user scroll)
			if cooldownExpired && (!state.FocusYSet || focusInfo.FocusedChildY != state.LastFocusY) {
				padding := focusInfo.FocusedChildHeight
				if padding <= 0 {
					padding = 40
				}

				maxScroll := maxf(0, state.ContentHeight-height)
				targetY := focusInfo.FocusedChildY

				visibleTop := state.ScrollY + padding
				visibleBottom := state.ScrollY + height - padding

				if targetY < visibleTop {
					state.ScrollY = clampf(targetY-padding, 0, maxScroll)
				} else if targetY > visibleBottom {
					state.ScrollY = clampf(targetY-height+padding, 0, maxScroll)
				}

				// Track focus position to avoid re-scrolling to same position
				state.LastFocusY = focusInfo.FocusedChildY
				state.FocusYSet = true
			}
		}

		// Handle auto-scroll from ctx.ScrollTo() - widgets call this to request focus
		// Auto-scroll is enabled by default. Set keyboardNavigated=false to disable.
		if scrollFocus.ok && keyboardNav {
			// User scroll cooldown: suppress auto-scroll for 300ms after manual scrolling
			const userScrollCooldown = 0.3 // 300ms
			cooldownExpired := state.UserScrollTime >= userScrollCooldown

			// Only auto-scroll if:
			// 1. Cooldown has expired (user hasn't scrolled recently)
			// 2. Focus position has changed from last frame
			if cooldownExpired && (!state.FocusYSet || scrollFocus.y != state.LastFocusY) {
				padding := scrollFocus.padding
				if padding <= 0 {
					padding = 40
				}

				maxScroll := maxf(0, state.ContentHeight-height)

				visibleTop := state.ScrollY + padding
				visibleBottom := state.ScrollY + height - padding

				if scrollFocus.y < visibleTop {
					state.ScrollY = clampf(scrollFocus.y-padding, 0, maxScroll)
				} else if scrollFocus.y > visibleBottom {
					state.ScrollY = clampf(scrollFocus.y-height+padding, 0, maxScroll)
				}
			}
			state.LastFocusY = scrollFocus.y
			state.FocusYSet = true
		}

		// Handle FocusY option - auto-scroll when focus position changes
		focus := GetOpt(o, OptFocus)
		if focus.Set {
			// Only scroll if focus changed from last frame
			if !state.FocusYSet || focus.Y != state.LastFocusY {
				padding := focus.Padding
				if padding <= 0 {
					padding = 40 // Default padding
				}

				maxScroll := maxf(0, state.ContentHeight-height)
				targetY := focus.Y

				// Check if target is outside visible area
				visibleTop := state.ScrollY + padding
				visibleBottom := state.ScrollY + height - padding

				if targetY < visibleTop {
					state.ScrollY = clampf(targetY-padding, 0, maxScroll)
				} else if targetY > visibleBottom {
					state.ScrollY = clampf(targetY-height+padding, 0, maxScroll)
				}
			}
			// Remember focus position for next frame
			state.LastFocusY = focus.Y
			state.FocusYSet = true
		}

		// Auto-scroll to focus registered by child widgets via ctx.SetScrollFocus()
		if focusY, focusPad, ok := ctx.ConsumeScrollFocus(); ok {
			// Only scroll if focus changed from last frame
			if !state.FocusYSet || focusY != state.LastFocusY {
				padding := focusPad
				if padding <= 0 {
					padding = 40
				}

				maxScroll := maxf(0, state.ContentHeight-height)

				visibleTop := state.ScrollY + padding
				visibleBottom := state.ScrollY + height - padding

				if focusY < visibleTop {
					state.ScrollY = clampf(focusY-padding, 0, maxScroll)
				} else if focusY > visibleBottom {
					state.ScrollY = clampf(focusY-height+padding, 0, maxScroll)
				}
			}
			state.LastFocusY = focusY
			state.FocusYSet = true
		}

		// Determine if scrollbar should now be shown (after measuring content)
		showScrollbar = scrollbarVisibility == ScrollbarAlways ||
			(scrollbarVisibility != ScrollbarNever && state.ContentHeight > height)

		// Handle scroll input when hovered (no focus required)
		if ctx.Input != nil && ctx.isHovered(scrollID, viewportRect) {
			// Mouse wheel vertical scrolling
			if ctx.Input.MouseWheelY != 0 {
				maxScroll := maxf(0, state.ContentHeight-height)
				newScroll := clampf(state.ScrollY-ctx.Input.MouseWheelY*30, 0, maxScroll)
				if GetOpt(o, OptClampToContent) {
					newScroll = clampf(newScroll, 0, maxScroll)
				}
				state.ScrollY = newScroll
				// Track user scroll to suppress auto-scroll during manual interaction
				state.UserScrolledThisFrame = true
				state.UserScrollTime = 0
			}

			// Mouse wheel horizontal scrolling (with Shift or if enabled)
			if horizontalScroll && ctx.Input.MouseWheelX != 0 {
				maxScroll := maxf(0, state.ContentWidth-contentWidth)
				newScroll := clampf(state.ScrollX-ctx.Input.MouseWheelX*30, 0, maxScroll)
				state.ScrollX = newScroll
				state.UserScrolledThisFrame = true
				state.UserScrollTime = 0
			}

			// Keyboard scrolling when hovered (PageUp, PageDown, Home, End)
			scrollAmount := height * 0.8 // Page up/down scrolls 80% of viewport
			keyboardScrolled := false

			if ctx.Input.KeyPressed(KeyPageDown) {
				maxScroll := maxf(0, state.ContentHeight-height)
				state.ScrollY = clampf(state.ScrollY+scrollAmount, 0, maxScroll)
				keyboardScrolled = true
			}
			if ctx.Input.KeyPressed(KeyPageUp) {
				maxScroll := maxf(0, state.ContentHeight-height)
				state.ScrollY = clampf(state.ScrollY-scrollAmount, 0, maxScroll)
				keyboardScrolled = true
			}
			if ctx.Input.KeyPressed(KeyHome) {
				state.ScrollY = 0
				keyboardScrolled = true
			}
			if ctx.Input.KeyPressed(KeyEnd) {
				state.ScrollY = maxf(0, state.ContentHeight-height)
				keyboardScrolled = true
			}

			if keyboardScrolled {
				state.UserScrolledThisFrame = true
				state.UserScrollTime = 0
			}
		}

		// Draw scrollbar if content exceeds height
		if showScrollbar && state.ContentHeight > height {
			// Calculate scrollbar thumb size and position
			scrollRatio := height / state.ContentHeight
			thumbHeight := maxf(20, height*scrollRatio)
			maxScroll := state.ContentHeight - height
			thumbPos := float32(0)
			if maxScroll > 0 {
				thumbPos = (state.ScrollY / maxScroll) * (height - thumbHeight)
			}

			thumbY := y + thumbPos

			// Scrollbar background
			ctx.DrawList.AddRect(scrollbarX, y, scrollbarWidth, height, ctx.style.ScrollbarBgColor)

			// Check if scrollbar thumb is hovered or being dragged
			thumbRect := Rect{X: scrollbarX, Y: thumbY, W: scrollbarWidth, H: thumbHeight}
			thumbHovered := ctx.isHovered(scrollID, thumbRect)

			// Handle scrollbar dragging
			if ctx.Input != nil {
				// Start drag on thumb click
				if thumbHovered && ctx.Input.MouseClicked(MouseButtonLeft) {
					state.Dragging = true
					state.DragStartY = ctx.Input.MouseY
					state.DragStartScr = state.ScrollY
				}

				// Handle ongoing drag
				if state.Dragging {
					if ctx.Input.MouseDown(MouseButtonLeft) {
						deltaY := ctx.Input.MouseY - state.DragStartY
						// Convert pixel delta to scroll delta
						scrollableTrack := height - thumbHeight
						if scrollableTrack > 0 {
							scrollDelta := deltaY * (maxScroll / scrollableTrack)
							state.ScrollY = clampf(state.DragStartScr+scrollDelta, 0, maxScroll)
						}
						// Track scrollbar drag as user interaction
						state.UserScrolledThisFrame = true
						state.UserScrollTime = 0
					} else {
						state.Dragging = false
					}
				}

				// Click on track (above or below thumb) to page scroll
				scrollbarRect := Rect{X: scrollbarX, Y: y, W: scrollbarWidth, H: height}
				if !thumbHovered && ctx.isHovered(scrollID, scrollbarRect) && ctx.Input.MouseClicked(MouseButtonLeft) {
					if ctx.Input.MouseY < thumbY {
						// Click above thumb - scroll up
						state.ScrollY = clampf(state.ScrollY-height, 0, maxScroll)
						state.UserScrolledThisFrame = true
						state.UserScrollTime = 0
					} else if ctx.Input.MouseY > thumbY+thumbHeight {
						// Click below thumb - scroll down
						state.ScrollY = clampf(state.ScrollY+height, 0, maxScroll)
						state.UserScrolledThisFrame = true
						state.UserScrollTime = 0
					}
				}
			}

			// Scrollbar thumb
			thumbColor := ctx.style.ScrollbarGrabColor
			if state.Dragging || thumbHovered {
				thumbColor = ctx.style.ScrollbarGrabHovered
			}
			ctx.DrawList.AddRect(scrollbarX, thumbY, scrollbarWidth, thumbHeight, thumbColor)
		}

		// Draw horizontal scrollbar if enabled and content exceeds width
		if horizontalScroll && state.ContentWidth > contentWidth {
			hScrollbarHeight := ctx.style.ScrollbarSize
			hScrollbarY := y + height - hScrollbarHeight

			// Calculate horizontal thumb
			hScrollRatio := contentWidth / state.ContentWidth
			hThumbWidth := maxf(20, contentWidth*hScrollRatio)
			hMaxScroll := state.ContentWidth - contentWidth
			hThumbPos := float32(0)
			if hMaxScroll > 0 {
				hThumbPos = (state.ScrollX / hMaxScroll) * (contentWidth - hThumbWidth)
			}

			// Horizontal scrollbar background
			ctx.DrawList.AddRect(contentX, hScrollbarY, contentWidth, hScrollbarHeight, ctx.style.ScrollbarBgColor)

			// Horizontal scrollbar thumb
			ctx.DrawList.AddRect(contentX+hThumbPos, hScrollbarY, hThumbWidth, hScrollbarHeight, ctx.style.ScrollbarGrabColor)
		}

		// Update user scroll cooldown timer
		// If user didn't scroll this frame, increment the timer
		// This allows auto-scroll to resume after a brief cooldown
		if !state.UserScrolledThisFrame {
			state.UserScrollTime += ctx.DeltaTime
		}
		state.UserScrolledThisFrame = false // Reset for next frame

		// State is automatically saved via pointer (no need to call SetState)

		// Restore cursor position after scrollable area
		ctx.cursor.X = x
		ctx.cursor.Y = y + height
	}
}

// scrollableNameToID maps scrollable names to their IDs for lookup.
// This enables GetScrollableState to find state by name instead of ID.
var scrollableNameToID = make(map[string]ID)

// GetScrollableState returns a pointer to the scrollable's state for advanced manipulation.
// Returns nil if the scrollable hasn't been rendered yet.
// Note: This returns state from the FrameStore which persists across frames until cleanup.
func GetScrollableState(ctx *Context, id string) *ScrollableState {
	fullName := id + "_scrollable"
	if storedID, ok := scrollableNameToID[fullName]; ok {
		return scrollableStore.GetIfExists(storedID)
	}
	return nil
}
