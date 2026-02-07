package gui

import "strings"

// ComboBox draws a dropdown selection widget.
// Returns true if the selection changed.
//
// Usage:
//
//	items := []string{"Low", "Medium", "High"}
//	if ctx.ComboBox("Quality", &selectedIndex, items) {
//	    applyQuality(selectedIndex)
//	}
func (ctx *Context) ComboBox(label string, selectedIndex *int, items []string, opts ...Option) bool {
	pos := ctx.ItemPos()
	o := applyOptions(opts)

	id := ctx.GetID(label)
	if optID := GetOpt(o, OptID); optID != "" {
		id = ctx.GetID(optID)
	}

	// Get state
	state := GetState(ctx, id, ComboBoxState{HoveredIndex: -1, KeyboardIndex: -1})

	// Calculate dimensions
	labelWidth := float32(0)
	if label != "" {
		labelWidth = ctx.MeasureText(label).X + ctx.style.ItemSpacing
	}

	// Calculate combo width based on longest item
	comboWidth := float32(150)
	if width := GetOpt(o, OptWidth); width > 0 {
		comboWidth = width
	} else {
		for _, item := range items {
			itemWidth := ctx.MeasureText(item).X + ctx.style.ButtonPadding*2 + 20 // +20 for arrow
			if itemWidth > comboWidth {
				comboWidth = itemWidth
			}
		}
	}

	h := ctx.lineHeight() + ctx.style.ButtonPadding*2
	arrowSize := float32(8)

	// Draw label
	if label != "" {
		ctx.addText(pos.X, pos.Y+(h-ctx.lineHeight())/2, label, ctx.style.TextColor)
	}

	// Header box position
	headerX := pos.X + labelWidth
	headerY := pos.Y

	// Interaction rect for header
	headerRect := Rect{X: headerX, Y: headerY, W: comboWidth, H: h}

	hovered := ctx.isHovered(id, headerRect)
	changed := false

	// Register as focusable (enables click-to-focus and keyboard navigation)
	focusable := ctx.RegisterFocusable(id, label, headerRect, FocusTypeLeaf)
	isFocused := focusable != nil && focusable.IsFocused()

	// Draw header background
	bgColor := ctx.style.ButtonColor
	if hovered || state.Open || isFocused {
		bgColor = ctx.style.ButtonHoveredColor
	}
	ctx.DrawList.AddRect(headerX, headerY, comboWidth, h, bgColor)
	ctx.DrawList.AddRectOutline(headerX, headerY, comboWidth, h, ctx.style.InputBorderColor, 1)

	// Draw selected item text
	selectedText := ""
	if *selectedIndex >= 0 && *selectedIndex < len(items) {
		selectedText = items[*selectedIndex]
	}
	textX := headerX + ctx.style.ButtonPadding
	textY := headerY + (h-ctx.lineHeight())/2
	ctx.addText(textX, textY, selectedText, ctx.style.TextColor)

	// Draw dropdown arrow
	arrowX := headerX + comboWidth - ctx.style.ButtonPadding - arrowSize
	arrowY := headerY + h/2
	if state.Open {
		// Up arrow when open
		ctx.DrawList.AddTriangle(
			arrowX+arrowSize/2, arrowY-arrowSize/4,
			arrowX, arrowY+arrowSize/4,
			arrowX+arrowSize, arrowY+arrowSize/4,
			ctx.style.ComboArrowColor,
		)
	} else {
		// Down arrow when closed
		ctx.DrawList.AddTriangle(
			arrowX+arrowSize/2, arrowY+arrowSize/4,
			arrowX, arrowY-arrowSize/4,
			arrowX+arrowSize, arrowY-arrowSize/4,
			ctx.style.ComboArrowColor,
		)
	}

	// Track if dropdown was just opened this frame (to prevent Enter from immediately closing)
	justOpened := false

	// Handle header click
	if ctx.isClicked(id, headerRect) {
		state.Open = !state.Open
		state.HoveredIndex = -1
		if state.Open {
			justOpened = true
			state.KeyboardIndex = *selectedIndex // Start keyboard nav at current selection
			ctx.SetActivePopup(id)               // Mark popup as active
			if GetOpt(o, OptSearchable) {
				state.SearchText = ""
			}
		} else {
			ctx.SetActivePopup(0) // Close popup
		}
	}

	// Keyboard support when focused
	if isFocused && ctx.Input != nil {
		// Enter/Space to toggle dropdown
		if ctx.Input.KeyPressed(KeyEnter) || ctx.Input.KeyPressed(KeySpace) {
			state.Open = !state.Open
			state.HoveredIndex = -1
			if state.Open {
				justOpened = true
				state.KeyboardIndex = *selectedIndex // Start keyboard nav at current selection
				ctx.SetActivePopup(id)               // Mark popup as active
				if GetOpt(o, OptSearchable) {
					state.SearchText = ""
				}
			} else {
				ctx.SetActivePopup(0) // Close popup
			}
		}
	}

	// Draw dropdown when open (uses ForegroundDrawList to render on top of everything)
	if state.Open {
		// Mark popup as active every frame it's open (for HandleInput in next frame)
		ctx.SetActivePopup(id)
		// Capture keyboard when dropdown is open to prevent focus navigation
		ctx.WantCaptureKeyboard = true

		// Use foreground draw list for popups so they appear above other widgets
		fgDrawList := ctx.ForegroundDrawList
		if fgDrawList == nil {
			fgDrawList = ctx.DrawList // Fallback if no foreground list
		}

		dropdownY := headerY + h

		// Filter items if searchable
		filteredItems := items
		filteredIndices := make([]int, len(items))
		for i := range items {
			filteredIndices[i] = i
		}

		searchable := GetOpt(o, OptSearchable)
		if searchable && state.SearchText != "" {
			filteredItems = nil
			filteredIndices = nil
			searchLower := strings.ToLower(state.SearchText)
			for i, item := range items {
				if strings.Contains(strings.ToLower(item), searchLower) {
					filteredItems = append(filteredItems, item)
					filteredIndices = append(filteredIndices, i)
				}
			}
		}

		// Calculate dropdown height
		itemHeight := ctx.lineHeight() + ctx.style.ItemSpacing
		searchHeight := float32(0)
		if searchable {
			searchHeight = ctx.lineHeight() + ctx.style.InputPadding*2 + ctx.style.ItemSpacing
		}

		maxItems := len(filteredItems)
		maxDropdownHeight := GetOpt(o, OptMaxDropdownHeight)
		if maxDropdownHeight == 0 {
			maxDropdownHeight = 200
		}

		contentHeight := float32(maxItems)*itemHeight + searchHeight
		dropdownHeight := minf(contentHeight, maxDropdownHeight)

		// Draw dropdown background (fully opaque for visibility)
		fgDrawList.AddRect(headerX, dropdownY, comboWidth, dropdownHeight, RGBA(20, 20, 25, 255))
		fgDrawList.AddRectOutline(headerX, dropdownY, comboWidth, dropdownHeight, ctx.style.InputBorderColor, 1)

		// Handle search input if searchable
		if searchable {
			searchRect := Rect{
				X: headerX + 2,
				Y: dropdownY + 2,
				W: comboWidth - 4,
				H: ctx.lineHeight() + ctx.style.InputPadding*2,
			}

			// Draw search box
			fgDrawList.AddRect(searchRect.X, searchRect.Y, searchRect.W, searchRect.H, ctx.style.InputBgColor)

			// Handle search input
			if ctx.Input != nil {
				for _, ch := range ctx.Input.InputChars {
					if ch >= 32 && ch < 127 {
						state.SearchText += string(ch)
					}
				}
				if ctx.Input.KeyRepeated(KeyBackspace) && len(state.SearchText) > 0 {
					state.SearchText = state.SearchText[:len(state.SearchText)-1]
				}
			}

			// Draw search text
			searchTextX := searchRect.X + ctx.style.InputPadding
			searchTextY := searchRect.Y + ctx.style.InputPadding
			if state.SearchText != "" {
				ctx.addTextTo(fgDrawList, searchTextX, searchTextY, state.SearchText, ctx.style.TextColor)
			} else {
				ctx.addTextTo(fgDrawList, searchTextX, searchTextY, "Search...", ctx.style.TextDisabledColor)
			}

			dropdownY += searchHeight
		}

		// Push clip rect for scrollable area
		scrollAreaHeight := dropdownHeight - searchHeight
		fgDrawList.PushClipRect(headerX, dropdownY, headerX+comboWidth, dropdownY+scrollAreaHeight)

		// Draw items with scroll offset
		itemY := dropdownY - state.ScrollY
		state.HoveredIndex = -1

		for i, item := range filteredItems {
			originalIndex := filteredIndices[i]

			// Skip items outside visible area
			if itemY+itemHeight < dropdownY {
				itemY += itemHeight
				continue
			}
			if itemY > dropdownY+scrollAreaHeight {
				break
			}

			itemRect := Rect{X: headerX + 2, Y: itemY, W: comboWidth - 4, H: itemHeight}

			// Check if hovered
			if ctx.isHovered(id, itemRect) && itemRect.Y >= dropdownY && itemRect.Y+itemRect.H <= dropdownY+scrollAreaHeight {
				state.HoveredIndex = i
			}

			// Draw item background and debug focus highlight
			isKeyboardSelected := state.KeyboardIndex == i
			if originalIndex == *selectedIndex {
				fgDrawList.AddRect(itemRect.X, itemRect.Y, itemRect.W, itemRect.H, ctx.style.SelectedBgColor)
			} else if isKeyboardSelected {
				// Keyboard navigation highlight (distinct from mouse hover)
				fgDrawList.AddRect(itemRect.X, itemRect.Y, itemRect.W, itemRect.H, ctx.style.SelectedBgColor)
			} else if state.HoveredIndex == i {
				fgDrawList.AddRect(itemRect.X, itemRect.Y, itemRect.W, itemRect.H, ctx.style.HoveredBgColor)
			}

			// Draw debug focus highlight for keyboard-selected item
			if isKeyboardSelected && ctx.DebugFocusHighlight {
				fgDrawList.AddRect(itemRect.X, itemRect.Y, itemRect.W, itemRect.H, DebugFocusColor)
				fgDrawList.AddRectOutline(itemRect.X, itemRect.Y, itemRect.W, itemRect.H, DebugFocusBorderColor, 3)
			}

			// Draw item text
			textColor := ctx.style.TextColor
			if originalIndex == *selectedIndex || isKeyboardSelected {
				textColor = ctx.style.SelectedTextColor
			}
			ctx.addTextTo(fgDrawList, itemRect.X+ctx.style.ItemSpacing, itemY, item, textColor)

			// Handle click on item
			if ctx.Input != nil && ctx.Input.MouseClicked(MouseButtonLeft) {
				if ctx.isHovered(id, itemRect) && itemRect.Y >= dropdownY && itemRect.Y+itemRect.H <= dropdownY+scrollAreaHeight {
					if originalIndex != *selectedIndex {
						*selectedIndex = originalIndex
						changed = true
					}
					state.Open = false
					ctx.SetActivePopup(0)
				}
			}

			itemY += itemHeight
		}

		fgDrawList.PopClipRect()

		// Handle scroll
		if ctx.Input != nil {
			dropdownRect := Rect{X: headerX, Y: headerY + h, W: comboWidth, H: dropdownHeight}
			if ctx.isHovered(id, dropdownRect) && ctx.Input.MouseWheelY != 0 {
				maxScroll := maxf(0, contentHeight-searchHeight-scrollAreaHeight)
				state.ScrollY = clampf(state.ScrollY-ctx.Input.MouseWheelY*20, 0, maxScroll)
			}
		}

		// Close on click outside
		if ctx.Input != nil && ctx.Input.MouseClicked(MouseButtonLeft) {
			dropdownRect := Rect{X: headerX, Y: headerY, W: comboWidth, H: h + dropdownHeight}
			if !ctx.isHovered(id, dropdownRect) {
				state.Open = false
				ctx.SetActivePopup(0)
			}
		}

		// Close on Escape
		if ctx.Input != nil && ctx.Input.KeyPressed(KeyEscape) {
			state.Open = false
			ctx.SetActivePopup(0)
		}

		// Keyboard navigation within dropdown (when focused or open)
		if ctx.Input != nil {
			// Use keyboard index if not set
			if state.KeyboardIndex < 0 {
				state.KeyboardIndex = *selectedIndex
			}

			// Up/Down to navigate items
			if ctx.Input.KeyRepeated(KeyUp) {
				if state.KeyboardIndex > 0 {
					state.KeyboardIndex--
				}
			}
			if ctx.Input.KeyRepeated(KeyDown) {
				if state.KeyboardIndex < len(filteredItems)-1 {
					state.KeyboardIndex++
				}
			}

			// Enter to select and close (skip if we just opened this frame)
			if !justOpened && ctx.Input.KeyPressed(KeyEnter) {
				if state.KeyboardIndex >= 0 && state.KeyboardIndex < len(filteredIndices) {
					originalIndex := filteredIndices[state.KeyboardIndex]
					if originalIndex != *selectedIndex {
						*selectedIndex = originalIndex
						changed = true
					}
					state.Open = false
					ctx.SetActivePopup(0)
				}
			}
		}
	} else {
		// Dropdown is closed - if this combobox owned the popup, clear it
		if ctx.ActivePopupID() == id {
			ctx.SetActivePopup(0)
		}
	}

	// Save state
	SetState(ctx, id, state)

	// Advance cursor
	totalWidth := labelWidth + comboWidth
	ctx.advanceCursor(Vec2{totalWidth, h})

	return changed
}
