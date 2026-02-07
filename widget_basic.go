package gui

import "strings"

// Text draws text at the current cursor position.
func (ctx *Context) Text(text string) {
	pos := ctx.ItemPos()
	ctx.addText(pos.X, pos.Y, text, ctx.style.TextColor)
	ctx.advanceCursor(ctx.MeasureText(text))
}

// TextColored draws text with a specific color.
func (ctx *Context) TextColored(text string, color uint32) {
	pos := ctx.ItemPos()
	ctx.addText(pos.X, pos.Y, text, color)
	ctx.advanceCursor(ctx.MeasureText(text))
}

// TextDisabled draws text with the disabled color.
func (ctx *Context) TextDisabled(text string) {
	pos := ctx.ItemPos()
	ctx.addText(pos.X, pos.Y, text, ctx.style.TextDisabledColor)
	ctx.advanceCursor(ctx.MeasureText(text))
}

// SelectableRow wraps content with selection highlighting.
// Use this to create custom selectable rows with consistent styling.
// The content function renders the row's contents.
// Pass rowWidth to specify the highlight width (0 = use default 200px).
//
// Example:
//
//	ctx.SelectableRow(isSelected, 180)(func() {
//	    ctx.Text("Label:")
//	    ctx.InputText("", &value, gui.WithID("input"))
//	})
func (ctx *Context) SelectableRow(selected bool, rowWidth float32) func(func()) {
	return func(content func()) {
		pos := ctx.ItemPos()
		h := ctx.lineHeight()

		if rowWidth <= 0 {
			rowWidth = 200 // Default width
		}

		// Draw selection highlight first (background)
		if selected {
			ctx.DrawList.AddRect(pos.X, pos.Y, rowWidth, h, ctx.style.SelectedBgColor)
			ctx.DrawList.AddRect(pos.X, pos.Y, 4, h, ColorCyan) // Left edge bar
		}

		// Render content on top
		content()
	}
}

// TextWrapped draws text with automatic word wrapping.
// maxWidth specifies the maximum line width (0 = use current layout width).
// This fixes ImGui's missing text wrapping feature.
func (ctx *Context) TextWrapped(text string, maxWidth float32) {
	if maxWidth <= 0 {
		maxWidth = ctx.currentLayoutWidth()
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return
	}

	pos := ctx.ItemPos()
	lineH := ctx.lineHeight()

	line := ""
	y := pos.Y
	lineCount := 0

	for _, word := range words {
		testLine := line
		if testLine != "" {
			testLine += " "
		}
		testLine += word

		// Use proper text measurement (works with proportional fonts)
		width := ctx.MeasureText(testLine).X
		if width > maxWidth && line != "" {
			// Draw current line and start new one
			ctx.addText(pos.X, y, line, ctx.style.TextColor)
			y += lineH
			lineCount++
			line = word
		} else {
			line = testLine
		}
	}

	// Draw remaining text
	if line != "" {
		ctx.addText(pos.X, y, line, ctx.style.TextColor)
		lineCount++
	}

	ctx.advanceCursor(Vec2{maxWidth, float32(lineCount) * lineH})
}

// LabelText draws a label and value side by side.
func (ctx *Context) LabelText(label, value string) {
	ctx.HStack()(func() {
		ctx.Text(label)
		ctx.Text(value)
	})
}

// Button draws a button and returns true if clicked.
func (ctx *Context) Button(label string, opts ...Option) bool {
	pos := ctx.ItemPos()
	o := applyOptions(opts)

	// Generate ID
	id := ctx.GetID(label)
	if optID := GetOpt(o, OptID); optID != "" {
		id = ctx.GetID(optID)
	}

	// Calculate size
	textSize := ctx.MeasureText(label)
	size := Vec2{
		X: textSize.X + ctx.style.ButtonPadding*2,
		Y: textSize.Y + ctx.style.ButtonPadding*2,
	}

	// Apply custom dimensions
	if optWidth := GetOpt(o, OptWidth); optWidth > 0 {
		size.X = optWidth
	}
	if optHeight := GetOpt(o, OptHeight); optHeight > 0 {
		size.Y = optHeight
	}

	// Interaction rect
	rect := Rect{X: pos.X, Y: pos.Y, W: size.X, H: size.Y}

	// Register as focusable (auto-draws debug rect if focused)
	disabled := GetOpt(o, OptDisabled)
	if disabled {
		ctx.RegisterFocusableDisabled(id, label, rect, FocusTypeLeaf)
	} else {
		ctx.RegisterFocusable(id, label, rect, FocusTypeLeaf)
	}

	// State-based coloring
	bgColor := ctx.style.ButtonColor
	hovered := ctx.isHovered(id, rect) && !disabled
	pressed := ctx.isPressed(id, rect) && !disabled
	focused := ctx.IsRegistryFocused(id)

	if focused {
		bgColor = ctx.style.ButtonActiveColor
	} else if hovered {
		bgColor = ctx.style.ButtonHoveredColor
	}
	if pressed {
		bgColor = ctx.style.ButtonActiveColor
	}
	if disabled {
		bgColor = ctx.style.ButtonDisabledColor
	}

	// Draw background
	ctx.DrawList.AddRect(pos.X, pos.Y, size.X, size.Y, bgColor)

	// Draw text (centered in button)
	textX := pos.X + (size.X-textSize.X)/2
	textY := pos.Y + (size.Y-textSize.Y)/2
	textColor := ctx.style.TextColor
	if disabled {
		textColor = ctx.style.TextDisabledColor
	}
	ctx.addText(textX, textY, label, textColor)

	// Check for click
	clicked := !disabled && ctx.isClicked(id, rect)
	ctx.advanceCursor(size)

	return clicked
}

// SmallButton draws a smaller button without extra padding.
func (ctx *Context) SmallButton(label string, opts ...Option) bool {
	// Temporarily reduce padding
	savedPadding := ctx.style.ButtonPadding
	ctx.style.ButtonPadding = 2
	result := ctx.Button(label, opts...)
	ctx.style.ButtonPadding = savedPadding
	return result
}

// Selectable draws a selectable list item.
// Returns true if clicked.
func (ctx *Context) Selectable(label string, selected bool, opts ...Option) bool {
	pos := ctx.ItemPos()
	o := applyOptions(opts)

	// Generate ID
	id := ctx.GetID(label)
	if optID := GetOpt(o, OptID); optID != "" {
		id = ctx.GetID(optID)
	}

	// Determine prefix first (needed for width calculation)
	prefix := "  "
	if selected {
		prefix = "> "
	}

	// Calculate size based on text width (auto-size to content)
	textSize := ctx.MeasureText(prefix + label)
	w := textSize.X + ctx.style.ItemSpacing*2 // Add some horizontal padding
	h := ctx.lineHeight()

	// Interaction rect
	rect := Rect{X: pos.X, Y: pos.Y, W: w, H: h}

	// Register as focusable (auto-draws debug rect if focused)
	disabled := GetOpt(o, OptDisabled)
	if disabled {
		ctx.RegisterFocusableDisabled(id, label, rect, FocusTypeLeaf)
	} else {
		ctx.RegisterFocusable(id, label, rect, FocusTypeLeaf)
	}

	// Determine appearance
	var bgColor uint32
	textColor := ctx.style.TextColor

	hovered := ctx.isHovered(id, rect) && !disabled
	focused := ctx.IsRegistryFocused(id)

	if selected || focused {
		bgColor = ctx.style.SelectedBgColor
		textColor = ctx.style.SelectedTextColor
	} else if hovered {
		bgColor = ctx.style.HoveredBgColor
	}
	if disabled {
		textColor = ctx.style.TextDisabledColor
	}

	// Draw background
	if bgColor != 0 {
		ctx.DrawList.AddRect(pos.X, pos.Y, w, h, bgColor)
	}

	// Draw selection cursor bar (left edge indicator) for selected items
	if selected || focused {
		cursorWidth := float32(4)
		ctx.DrawList.AddRect(pos.X, pos.Y, cursorWidth, h, ColorCyan)
		// Debug focus highlighting - only draw manually for selected items not using registry focus
		if selected && !focused {
			ctx.DrawDebugFocusRect(pos.X, pos.Y, w, h)
		}
	}

	// Draw text
	ctx.addText(pos.X, pos.Y, prefix+label, textColor)

	// Check for click
	clicked := !disabled && ctx.isClicked(id, rect)
	ctx.advanceCursor(Vec2{w, h})

	return clicked
}

// Checkbox draws a checkbox with label.
// Returns true if the value changed.
func (ctx *Context) Checkbox(label string, value *bool, opts ...Option) bool {
	pos := ctx.ItemPos()
	o := applyOptions(opts)

	id := ctx.GetID(label)
	if optID := GetOpt(o, OptID); optID != "" {
		id = ctx.GetID(optID)
	}

	// Size of checkbox box
	boxSize := ctx.lineHeight()
	totalWidth := boxSize + ctx.style.ItemSpacing + ctx.MeasureText(label).X

	// Interaction rect
	rect := Rect{X: pos.X, Y: pos.Y, W: totalWidth, H: boxSize}

	// Register as focusable (auto-draws debug rect if focused)
	disabled := GetOpt(o, OptDisabled)
	if disabled {
		ctx.RegisterFocusableDisabled(id, label, rect, FocusTypeLeaf)
	} else {
		ctx.RegisterFocusable(id, label, rect, FocusTypeLeaf)
	}

	hovered := ctx.isHovered(id, rect) && !disabled
	focused := ctx.IsRegistryFocused(id)

	// Draw checkbox box
	boxColor := ctx.style.InputBgColor
	if focused {
		boxColor = ctx.style.InputFocusedBgColor
	} else if hovered {
		boxColor = ctx.style.InputFocusedBgColor
	}
	ctx.DrawList.AddRect(pos.X, pos.Y, boxSize, boxSize, boxColor)
	ctx.DrawList.AddRectOutline(pos.X, pos.Y, boxSize, boxSize,
		ctx.style.InputBorderColor, 1)

	// Draw checkmark if checked
	if *value {
		// Simple X checkmark
		padding := boxSize * 0.2
		x1, y1 := pos.X+padding, pos.Y+padding
		x2, y2 := pos.X+boxSize-padding, pos.Y+boxSize-padding
		ctx.DrawList.AddLine(x1, y1, x2, y2, ctx.style.TextColor, 2)
		ctx.DrawList.AddLine(x1, y2, x2, y1, ctx.style.TextColor, 2)
	}

	// Draw label
	textX := pos.X + boxSize + ctx.style.ItemSpacing
	textColor := ctx.style.TextColor
	if disabled {
		textColor = ctx.style.TextDisabledColor
	}
	ctx.addText(textX, pos.Y, label, textColor)

	// Handle click
	changed := false
	if !disabled && ctx.isClicked(id, rect) {
		*value = !*value
		changed = true
	}

	ctx.advanceCursor(Vec2{totalWidth, boxSize})
	return changed
}

// RadioButton draws a radio button.
// Returns true if this option was selected.
func (ctx *Context) RadioButton(label string, active bool, opts ...Option) bool {
	pos := ctx.ItemPos()
	o := applyOptions(opts)

	id := ctx.GetID(label)
	if optID := GetOpt(o, OptID); optID != "" {
		id = ctx.GetID(optID)
	}

	// Size of radio circle
	circleSize := ctx.lineHeight()
	totalWidth := circleSize + ctx.style.ItemSpacing + ctx.MeasureText(label).X

	// Interaction rect
	rect := Rect{X: pos.X, Y: pos.Y, W: totalWidth, H: circleSize}

	// Register as focusable (auto-draws debug rect if focused)
	disabled := GetOpt(o, OptDisabled)
	if disabled {
		ctx.RegisterFocusableDisabled(id, label, rect, FocusTypeLeaf)
	} else {
		ctx.RegisterFocusable(id, label, rect, FocusTypeLeaf)
	}

	hovered := ctx.isHovered(id, rect) && !disabled
	focused := ctx.IsRegistryFocused(id)

	// Draw outer circle (as square for simplicity - could use actual circle)
	boxColor := ctx.style.InputBgColor
	if focused {
		boxColor = ctx.style.InputFocusedBgColor
	} else if hovered {
		boxColor = ctx.style.InputFocusedBgColor
	}
	ctx.DrawList.AddRect(pos.X, pos.Y, circleSize, circleSize, boxColor)
	ctx.DrawList.AddRectOutline(pos.X, pos.Y, circleSize, circleSize,
		ctx.style.InputBorderColor, 1)

	// Draw inner filled circle if active
	if active {
		padding := circleSize * 0.25
		ctx.DrawList.AddRect(
			pos.X+padding, pos.Y+padding,
			circleSize-padding*2, circleSize-padding*2,
			ctx.style.SelectedBgColor)
	}

	// Draw label
	textX := pos.X + circleSize + ctx.style.ItemSpacing
	textColor := ctx.style.TextColor
	if disabled {
		textColor = ctx.style.TextDisabledColor
	}
	ctx.addText(textX, pos.Y, label, textColor)

	// Handle click
	clicked := !disabled && ctx.isClicked(id, rect)

	ctx.advanceCursor(Vec2{totalWidth, circleSize})
	return clicked
}

// ProgressBar draws a progress bar.
// fraction should be between 0.0 and 1.0.
func (ctx *Context) ProgressBar(fraction float32, opts ...Option) {
	pos := ctx.ItemPos()
	o := applyOptions(opts)

	w := ctx.currentLayoutWidth()
	if optWidth := GetOpt(o, OptWidth); optWidth > 0 {
		w = optWidth
	}
	h := ctx.lineHeight()
	if optHeight := GetOpt(o, OptHeight); optHeight > 0 {
		h = optHeight
	}

	fraction = clampf(fraction, 0, 1)

	// Background
	ctx.DrawList.AddRect(pos.X, pos.Y, w, h, ctx.style.InputBgColor)

	// Fill
	fillW := w * fraction
	if fillW > 0 {
		ctx.DrawList.AddRect(pos.X, pos.Y, fillW, h, ctx.style.SelectedBgColor)
	}

	// Border
	ctx.DrawList.AddRectOutline(pos.X, pos.Y, w, h, ctx.style.InputBorderColor, 1)

	ctx.advanceCursor(Vec2{w, h})
}

// InputText draws a text input field with full editing support.
// Features: cursor positioning, text selection, clipboard (Ctrl+C/V/X),
// undo/redo (Ctrl+Z/Y), and keyboard navigation (arrows, Home/End).
// Returns true if the value changed.
func (ctx *Context) InputText(label string, value *string, opts ...Option) bool {
	pos := ctx.ItemPos()
	o := applyOptions(opts)

	id := ctx.GetID(label)
	if optID := GetOpt(o, OptID); optID != "" {
		id = ctx.GetID(optID)
	}

	// Get or create state
	state := GetState(ctx, id, InputTextState{
		CursorPos:      len([]rune(*value)),
		SelectionStart: -1,
		SelectionEnd:   -1,
	})

	// Handle programmatic focus request
	// Track if we just started editing this frame (to skip processing the Enter key that triggered it)
	justStartedEditing := false
	if GetOpt(o, OptForceFocus) && !state.Editing {
		state.Editing = true
		justStartedEditing = true
	}

	// Track position for label + input box
	drawX := pos.X
	startX := pos.X

	// Start with label if provided
	if label != "" {
		ctx.addText(drawX, pos.Y, label, ctx.style.TextColor)
		drawX += ctx.MeasureText(label).X + ctx.style.ItemSpacing
	}

	// Input box dimensions
	w := float32(200)
	if optWidth := GetOpt(o, OptWidth); optWidth > 0 {
		w = optWidth
	}
	h := ctx.lineHeight() + ctx.style.InputPadding*2

	// Interaction rect
	rect := Rect{X: drawX, Y: pos.Y, W: w, H: h}

	// Register as focusable (enables click-to-focus and keyboard navigation)
	focusable := ctx.RegisterFocusable(id, label, rect, FocusTypeLeaf)
	isRegistryFocused := focusable != nil && focusable.IsFocused()

	// Enter to start editing when registry-focused but not in edit mode
	if isRegistryFocused && !state.Editing && ctx.Input != nil && ctx.Input.KeyPressed(KeyEnter) {
		state.Editing = true
		justStartedEditing = true
		state.CursorBlinkTime = 0
		state.CursorPos = len([]rune(*value))
		state.SelectAll(len([]rune(*value))) // Select all text on enter
	}

	// Draw background
	bgColor := ctx.style.InputBgColor
	if state.Editing {
		bgColor = ctx.style.InputFocusedBgColor
	}
	ctx.DrawList.AddRect(drawX, pos.Y, w, h, bgColor)
	ctx.DrawList.AddRectOutline(drawX, pos.Y, w, h, ctx.style.InputBorderColor, 1)

	// Convert to runes for proper Unicode handling
	runes := []rune(*value)
	textLen := len(runes)

	// Clamp cursor position
	if state.CursorPos > textLen {
		state.CursorPos = textLen
	}
	if state.CursorPos < 0 {
		state.CursorPos = 0
	}

	// Calculate text metrics
	textX := drawX + ctx.style.InputPadding
	textY := pos.Y + ctx.style.InputPadding
	maxWidth := w - ctx.style.InputPadding*2

	// Calculate scroll offset to keep cursor visible
	cursorTextWidth := ctx.MeasureText(string(runes[:state.CursorPos])).X
	if cursorTextWidth-state.ScrollOffset > maxWidth {
		state.ScrollOffset = cursorTextWidth - maxWidth + 10
	}
	if cursorTextWidth < state.ScrollOffset {
		state.ScrollOffset = cursorTextWidth
	}
	if state.ScrollOffset < 0 {
		state.ScrollOffset = 0
	}

	// Push clip rect for text area
	ctx.DrawList.PushClipRect(textX, pos.Y, textX+maxWidth, pos.Y+h)

	// Draw selection highlight if active
	if state.Editing && state.HasSelection() {
		selStart, selEnd := state.GetSelectedRange()
		selStartX := ctx.MeasureText(string(runes[:selStart])).X - state.ScrollOffset
		selEndX := ctx.MeasureText(string(runes[:selEnd])).X - state.ScrollOffset
		ctx.DrawList.AddRect(textX+selStartX, pos.Y+2, selEndX-selStartX, h-4, ctx.style.SelectedBgColor)
	}

	// Draw text
	ctx.addText(textX-state.ScrollOffset, textY, *value, ctx.style.TextColor)

	// Pop clip rect
	ctx.DrawList.PopClipRect()

	// Draw cursor when in edit mode
	if state.Editing {
		state.CursorBlinkTime += ctx.DeltaTime
		if int(state.CursorBlinkTime*2)%2 == 0 { // Blink every 0.5s
			cursorX := textX + cursorTextWidth - state.ScrollOffset
			ctx.DrawList.AddLine(cursorX, pos.Y+2, cursorX, pos.Y+h-2, ctx.style.TextColor, 1)
		}
	}

	// Handle click to enter edit mode
	// RegisterFocusable handles setting registry focus on click, but we also enter edit mode
	if ctx.isClicked(id, rect) {
		state.Editing = true
		state.CursorBlinkTime = 0

		// Calculate cursor position from click
		clickX := ctx.Input.MouseX - textX + state.ScrollOffset
		newCursorPos := 0
		for i := 0; i <= textLen; i++ {
			charX := ctx.MeasureText(string(runes[:i])).X
			if charX > clickX {
				break
			}
			newCursorPos = i
		}
		state.CursorPos = newCursorPos
		state.ClearSelection()
	}

	// Exit edit mode if registry focus moved to a different widget
	if state.Editing && !isRegistryFocused {
		state.Editing = false
	}

	// Handle input when in edit mode
	changed := false
	if state.Editing && ctx.Input != nil {
		// Mark that keyboard is captured
		ctx.WantCaptureKeyboard = true

		// Skip keyboard processing on the frame we just started editing via ForceFocus
		// This prevents the Enter key that triggered editing from also closing the input
		if !justStartedEditing {
			changed = ctx.processInputTextKeyboard(value, &state, &runes)
		}
	}

	// Save state
	SetState(ctx, id, state)

	// Advance cursor
	ctx.cursor.X = startX
	ctx.advanceCursor(Vec2{w + (drawX - startX), h})

	return changed
}

// processInputTextKeyboard handles keyboard input for InputText.
// Returns true if the value changed.
func (ctx *Context) processInputTextKeyboard(value *string, state *InputTextState, runes *[]rune) bool {
	changed := false
	textLen := len(*runes)
	input := ctx.Input

	// Helper to delete selected text
	deleteSelection := func() bool {
		if !state.HasSelection() {
			return false
		}
		start, end := state.GetSelectedRange()
		state.PushUndo(*value)
		*runes = append((*runes)[:start], (*runes)[end:]...)
		*value = string(*runes)
		state.CursorPos = start
		state.ClearSelection()
		return true
	}

	// Ctrl+A: Select All
	if input.ModCtrl && input.KeyPressed(KeyA) {
		state.SelectAll(textLen)
		return false
	}

	// Ctrl+C: Copy
	if input.ModCtrl && input.KeyPressed(KeyC) {
		if state.HasSelection() {
			start, end := state.GetSelectedRange()
			ClipboardSetText(string((*runes)[start:end]))
		}
		return false
	}

	// Ctrl+X: Cut
	if input.ModCtrl && input.KeyPressed(KeyX) {
		if state.HasSelection() {
			start, end := state.GetSelectedRange()
			ClipboardSetText(string((*runes)[start:end]))
			deleteSelection()
			changed = true
		}
		return changed
	}

	// Ctrl+V: Paste
	if input.ModCtrl && input.KeyPressed(KeyV) {
		clipboard := ClipboardGetText()
		if clipboard != "" {
			deleteSelection() // Delete selection if any
			state.PushUndo(*value)
			clipRunes := []rune(clipboard)
			*runes = append((*runes)[:state.CursorPos], append(clipRunes, (*runes)[state.CursorPos:]...)...)
			*value = string(*runes)
			state.CursorPos += len(clipRunes)
			changed = true
		}
		return changed
	}

	// Ctrl+Z: Undo
	if input.ModCtrl && input.KeyPressed(KeyZ) {
		if !input.ModShift {
			if undone, ok := state.Undo(*value); ok {
				*value = undone
				*runes = []rune(undone)
				state.CursorPos = len(*runes)
				state.ClearSelection()
				changed = true
			}
		} else {
			// Ctrl+Shift+Z: Redo
			if redone, ok := state.Redo(); ok {
				*value = redone
				*runes = []rune(redone)
				state.CursorPos = len(*runes)
				state.ClearSelection()
				changed = true
			}
		}
		return changed
	}

	// Ctrl+Y: Redo (alternative)
	if input.ModCtrl && input.KeyPressed(KeyY) {
		if redone, ok := state.Redo(); ok {
			*value = redone
			*runes = []rune(redone)
			state.CursorPos = len(*runes)
			state.ClearSelection()
			changed = true
		}
		return changed
	}

	// Arrow keys: cursor movement
	if input.KeyRepeated(KeyLeft) {
		if state.CursorPos > 0 {
			if input.ModCtrl {
				// Word jump left
				state.CursorPos = findWordBoundaryLeft(*runes, state.CursorPos)
			} else {
				state.CursorPos--
			}
		}
		if !input.ModShift {
			state.ClearSelection()
		} else {
			// Extend selection
			if state.SelectionStart < 0 {
				state.SelectionStart = state.CursorPos + 1
			}
			state.SelectionEnd = state.CursorPos
		}
		state.CursorBlinkTime = 0
	}

	if input.KeyRepeated(KeyRight) {
		if state.CursorPos < textLen {
			if input.ModCtrl {
				// Word jump right
				state.CursorPos = findWordBoundaryRight(*runes, state.CursorPos)
			} else {
				state.CursorPos++
			}
		}
		if !input.ModShift {
			state.ClearSelection()
		} else {
			// Extend selection
			if state.SelectionStart < 0 {
				state.SelectionStart = state.CursorPos - 1
			}
			state.SelectionEnd = state.CursorPos
		}
		state.CursorBlinkTime = 0
	}

	// Home: jump to start
	if input.KeyPressed(KeyHome) {
		state.CursorPos = 0
		if !input.ModShift {
			state.ClearSelection()
		} else {
			if state.SelectionStart < 0 {
				state.SelectionStart = textLen
			}
			state.SelectionEnd = 0
		}
		state.CursorBlinkTime = 0
	}

	// End: jump to end
	if input.KeyPressed(KeyEnd) {
		state.CursorPos = textLen
		if !input.ModShift {
			state.ClearSelection()
		} else {
			if state.SelectionStart < 0 {
				state.SelectionStart = 0
			}
			state.SelectionEnd = textLen
		}
		state.CursorBlinkTime = 0
	}

	// Backspace
	if input.KeyRepeated(KeyBackspace) {
		if state.HasSelection() {
			deleteSelection()
			changed = true
		} else if state.CursorPos > 0 {
			state.PushUndo(*value)
			*runes = append((*runes)[:state.CursorPos-1], (*runes)[state.CursorPos:]...)
			*value = string(*runes)
			state.CursorPos--
			changed = true
		}
		state.CursorBlinkTime = 0
	}

	// Delete
	if input.KeyRepeated(KeyDelete) {
		if state.HasSelection() {
			deleteSelection()
			changed = true
		} else if state.CursorPos < textLen {
			state.PushUndo(*value)
			*runes = append((*runes)[:state.CursorPos], (*runes)[state.CursorPos+1:]...)
			*value = string(*runes)
			changed = true
		}
		state.CursorBlinkTime = 0
	}

	// Escape: exit edit mode
	if input.KeyPressed(KeyEscape) {
		state.Editing = false
		return changed
	}

	// Enter: exit edit mode
	if input.KeyPressed(KeyEnter) {
		state.Editing = false
		return changed
	}

	// Text input (printable characters)
	for _, ch := range input.InputChars {
		if ch >= 32 { // Printable character
			deleteSelection() // Delete selection if any
			state.PushUndo(*value)
			*runes = append((*runes)[:state.CursorPos], append([]rune{ch}, (*runes)[state.CursorPos:]...)...)
			*value = string(*runes)
			state.CursorPos++
			changed = true
		}
	}

	return changed
}

// findWordBoundaryLeft finds the start of the word to the left of pos.
func findWordBoundaryLeft(runes []rune, pos int) int {
	if pos <= 0 {
		return 0
	}
	pos--
	// Skip whitespace
	for pos > 0 && isWhitespace(runes[pos]) {
		pos--
	}
	// Find start of word
	for pos > 0 && !isWhitespace(runes[pos-1]) {
		pos--
	}
	return pos
}

// findWordBoundaryRight finds the end of the word to the right of pos.
func findWordBoundaryRight(runes []rune, pos int) int {
	n := len(runes)
	if pos >= n {
		return n
	}
	// Skip current word
	for pos < n && !isWhitespace(runes[pos]) {
		pos++
	}
	// Skip whitespace
	for pos < n && isWhitespace(runes[pos]) {
		pos++
	}
	return pos
}

// isWhitespace returns true if the rune is a whitespace character.
func isWhitespace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\r'
}

// Tooltip shows a tooltip at the mouse position.
// Should be called right after the widget you want to add a tooltip to.
func (ctx *Context) Tooltip(text string) {
	if ctx.Input == nil {
		return
	}

	// Only show if previous widget was hovered
	// (This is a simplified implementation - a real one would track the last widget)
	mx, my := ctx.Input.MouseX, ctx.Input.MouseY

	// Draw tooltip background
	padding := float32(4)
	textSize := ctx.MeasureText(text)
	w := textSize.X + padding*2
	h := textSize.Y + padding*2

	// Position tooltip near mouse, but keep on screen
	x := mx + 10
	y := my + 10
	if x+w > ctx.DisplaySize.X {
		x = ctx.DisplaySize.X - w
	}
	if y+h > ctx.DisplaySize.Y {
		y = ctx.DisplaySize.Y - h
	}

	ctx.DrawList.AddRect(x, y, w, h, ctx.style.PanelColor)
	ctx.DrawList.AddRectOutline(x, y, w, h, ctx.style.PanelBorderColor, 1)
	ctx.addText(x+padding, y+padding, text, ctx.style.TextColor)
}

// CollapsingHeader draws a collapsible header.
// Returns true if the section is expanded.
func (ctx *Context) CollapsingHeader(label string, opts ...Option) bool {
	pos := ctx.ItemPos()
	o := applyOptions(opts)

	id := ctx.GetID(label)
	if optID := GetOpt(o, OptID); optID != "" {
		id = ctx.GetID(optID)
	}

	// Get stored state
	state := GetState(ctx, id, CollapsingHeaderState{Open: true})

	// Calculate size
	w := ctx.currentLayoutWidth()
	h := ctx.lineHeight()

	// Interaction rect
	rect := Rect{X: pos.X, Y: pos.Y, W: w, H: h}

	// Register as focusable (auto-draws debug rect if focused)
	ctx.RegisterFocusable(id, label, rect, FocusTypeSection)

	hovered := ctx.isHovered(id, rect)

	// Check focus state from both option and registry
	focused := GetOpt(o, OptFocused) || ctx.IsRegistryFocused(id)

	// Draw background
	bgColor := ctx.style.ButtonColor
	if focused {
		bgColor = ctx.style.ButtonActiveColor // Highlight when focused
	} else if hovered {
		bgColor = ctx.style.ButtonHoveredColor
	}
	ctx.DrawList.AddRect(pos.X, pos.Y, w, h, bgColor)

	// Debug focus highlighting is handled by RegisterFocusable

	// Draw arrow indicator - use cyan when focused
	arrow := "►"
	if state.Open {
		arrow = "▼"
	}
	arrowColor := ctx.style.TextColor
	if focused {
		arrowColor = ColorCyan
	}
	ctx.addText(pos.X+2, pos.Y, arrow, arrowColor)

	// Draw label
	ctx.addText(pos.X+ctx.MeasureText(arrow).X+4, pos.Y, label, ctx.style.TextColor)

	// Handle click
	if ctx.isClicked(id, rect) {
		state.Open = !state.Open
		SetState(ctx, id, state)
	}

	ctx.advanceCursor(Vec2{w, h})

	return state.Open
}

// TreeNode draws a tree node that can be expanded/collapsed.
// Returns true if the node is expanded (call TreePop when done).
func (ctx *Context) TreeNode(label string, opts ...Option) bool {
	open := ctx.CollapsingHeader(label, opts...)
	if open {
		ctx.Indent(ctx.style.ItemSpacing * 2)
	}
	return open
}

// TreePop ends a tree node started with TreeNode.
func (ctx *Context) TreePop() {
	ctx.Unindent(ctx.style.ItemSpacing * 2)
}

// Bullet draws a bullet point.
func (ctx *Context) Bullet() {
	pos := ctx.ItemPos()
	size := ctx.lineHeight() * 0.3
	x := pos.X + size
	y := pos.Y + ctx.lineHeight()/2

	// Draw circle as small square (could enhance to actual circle)
	ctx.DrawList.AddRect(x-size/2, y-size/2, size, size, ctx.style.TextColor)

	// Bullet is an inline element - advance horizontally
	ctx.cursor.X = pos.X + size*2 + ctx.style.ItemSpacing
}

// BulletText draws a bullet point with text.
func (ctx *Context) BulletText(text string) {
	ctx.HStack()(func() {
		ctx.Bullet()
		ctx.Text(text)
	})
}
