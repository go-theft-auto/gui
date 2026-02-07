package gui

import (
	"fmt"
	"strconv"
	"strings"
)

// NumberInputFloat draws a numeric input field for float32 values.
// Click to enter text edit mode, or drag left/right to adjust the value.
// Returns true if the value was changed.
//
// Usage:
//
//	ctx.HStack()(func() {
//	    ctx.NumberInputFloat("", &scaleX, WithPrefix("X:"), WithWidth(60))
//	    ctx.NumberInputFloat("", &scaleY, WithPrefix("Y:"), WithWidth(60))
//	    ctx.NumberInputFloat("", &scaleZ, WithPrefix("Z:"), WithWidth(60))
//	})
func (ctx *Context) NumberInputFloat(label string, value *float32, opts ...Option) bool {
	pos := ctx.ItemPos()
	o := applyOptions(opts)

	prefix := GetOpt(o, OptPrefix)
	suffix := GetOpt(o, OptSuffix)

	id := ctx.GetID(label + prefix + suffix)
	if optID := GetOpt(o, OptID); optID != "" {
		id = ctx.GetID(optID)
	}

	// Get input state
	state := GetState(ctx, id, NumberInputState{})

	// Calculate dimensions
	w := float32(80)
	if width := GetOpt(o, OptWidth); width > 0 {
		w = width
	}
	h := ctx.lineHeight() + ctx.style.InputPadding*2

	// Label width if present
	labelWidth := float32(0)
	if label != "" {
		labelWidth = ctx.MeasureText(label).X + ctx.style.ItemSpacing
	}

	// Draw label
	if label != "" {
		ctx.addText(pos.X, pos.Y+ctx.style.InputPadding, label, ctx.style.TextColor)
	}

	// Input box position
	boxX := pos.X + labelWidth
	boxY := pos.Y

	// Interaction rect
	rect := Rect{X: boxX, Y: boxY, W: w, H: h}

	// Register as focusable (enables click-to-focus and keyboard navigation)
	focusable := ctx.RegisterFocusable(id, label+prefix+suffix, rect, FocusTypeLeaf)
	isFocused := focusable != nil && focusable.IsFocused()

	hovered := ctx.isHovered(id, rect)
	changed := false

	// Track if we just started editing this frame (to prevent Enter from immediately closing)
	justStartedEditing := false

	// Drag speed (pixels per unit change)
	dragSpeed := GetOpt(o, OptDragSpeed)
	if dragSpeed == 0 {
		dragSpeed = 1.0 // Default: 1 pixel = 1 unit
	}

	// Handle input
	if ctx.Input != nil {
		// Click to start editing or dragging
		if hovered && ctx.Input.MouseClicked(MouseButtonLeft) {
			if state.Editing {
				// Already editing, don't restart
			} else {
				// Start dragging mode initially
				state.Dragging = true
				state.DragStartX = ctx.Input.MouseX
				state.DragStartValue = *value
			}
		}

		// Double-click to enter edit mode
		// For simplicity, we'll use a small movement threshold to distinguish click from drag
		if state.Dragging && ctx.Input.MouseReleased(MouseButtonLeft) {
			dragDist := absf(ctx.Input.MouseX - state.DragStartX)
			if dragDist < 3 {
				// Small movement = click, enter edit mode
				state.Editing = true
				justStartedEditing = true
				format := GetOpt(o, OptFormat)
				if format == "" {
					format = "%.2f"
				}
				// Handle integer format specifiers (%d) with float32 value
				if strings.Contains(format, "%d") {
					state.EditText = fmt.Sprintf(format, int(*value))
				} else {
					state.EditText = fmt.Sprintf(format, *value)
				}
			}
			state.Dragging = false
		}

		// Handle dragging
		if state.Dragging && ctx.Input.MouseDown(MouseButtonLeft) {
			deltaX := ctx.Input.MouseX - state.DragStartX
			deltaValue := deltaX / dragSpeed
			newValue := state.DragStartValue + deltaValue

			// Apply step if configured
			if step := GetOpt(o, OptStep); step > 0 {
				newValue = float32(int(newValue/step+0.5)) * step
			}

			// Clamp to range if configured
			rangeVal := GetOpt(o, OptRange)
			if rangeVal.HasRange {
				newValue = clampf(newValue, rangeVal.Min, rangeVal.Max)
			}

			if newValue != *value {
				*value = newValue
				changed = true
			}
		}

		// Handle text editing
		if state.Editing {
			// Mark that keyboard is captured (prevents hotkeys from triggering)
			ctx.WantCaptureKeyboard = true

			// Text input
			for _, ch := range ctx.Input.InputChars {
				// Allow digits, decimal point, minus sign
				if (ch >= '0' && ch <= '9') || ch == '.' || ch == '-' {
					state.EditText += string(ch)
				}
			}

			// Backspace
			if ctx.Input.KeyRepeated(KeyBackspace) && len(state.EditText) > 0 {
				state.EditText = state.EditText[:len(state.EditText)-1]
			}

			// Enter to confirm (skip if we just started editing this frame)
			if !justStartedEditing && ctx.Input.KeyPressed(KeyEnter) {
				if v, err := strconv.ParseFloat(strings.TrimSpace(state.EditText), 32); err == nil {
					newValue := float32(v)
					rangeVal := GetOpt(o, OptRange)
					if rangeVal.HasRange {
						newValue = clampf(newValue, rangeVal.Min, rangeVal.Max)
					}
					if newValue != *value {
						*value = newValue
						changed = true
					}
				}
				state.Editing = false
			}

			// Escape to cancel
			if ctx.Input.KeyPressed(KeyEscape) {
				state.Editing = false
			}
		}

		// Exit edit mode if registry focus moved to a different widget
		if state.Editing && !isFocused {
			// Confirm current value
			if v, err := strconv.ParseFloat(strings.TrimSpace(state.EditText), 32); err == nil {
				newValue := float32(v)
				rangeVal := GetOpt(o, OptRange)
				if rangeVal.HasRange {
					newValue = clampf(newValue, rangeVal.Min, rangeVal.Max)
				}
				if newValue != *value {
					*value = newValue
					changed = true
				}
			}
			state.Editing = false
		}

		// Click outside to confirm edit
		if state.Editing && !hovered && ctx.Input.MouseClicked(MouseButtonLeft) {
			if v, err := strconv.ParseFloat(strings.TrimSpace(state.EditText), 32); err == nil {
				newValue := float32(v)
				rangeVal := GetOpt(o, OptRange)
				if rangeVal.HasRange {
					newValue = clampf(newValue, rangeVal.Min, rangeVal.Max)
				}
				if newValue != *value {
					*value = newValue
					changed = true
				}
			}
			state.Editing = false
		}

		// Keyboard support when focused but not editing (Left/Right arrows to adjust)
		if isFocused && !state.Editing {
			step := GetOpt(o, OptStep)
			if step == 0 {
				step = 1.0 // Default step for number input
			}

			// Enter to start editing
			if ctx.Input.KeyPressed(KeyEnter) {
				state.Editing = true
				justStartedEditing = true
				format := GetOpt(o, OptFormat)
				if format == "" {
					format = "%.2f"
				}
				if strings.Contains(format, "%d") {
					state.EditText = fmt.Sprintf(format, int(*value))
				} else {
					state.EditText = fmt.Sprintf(format, *value)
				}
			}

			// Left/Right arrows to adjust value
			if ctx.Input.KeyRepeated(KeyLeft) {
				newValue := *value - step
				rangeVal := GetOpt(o, OptRange)
				if rangeVal.HasRange {
					newValue = clampf(newValue, rangeVal.Min, rangeVal.Max)
				}
				if newValue != *value {
					*value = newValue
					changed = true
				}
			}
			if ctx.Input.KeyRepeated(KeyRight) {
				newValue := *value + step
				rangeVal := GetOpt(o, OptRange)
				if rangeVal.HasRange {
					newValue = clampf(newValue, rangeVal.Min, rangeVal.Max)
				}
				if newValue != *value {
					*value = newValue
					changed = true
				}
			}
		}
	}

	// Draw background
	bgColor := ctx.style.InputBgColor
	if state.Editing {
		bgColor = ctx.style.InputFocusedBgColor
	} else if hovered || isFocused {
		bgColor = ctx.style.InputFocusedBgColor
	}
	ctx.DrawList.AddRect(boxX, boxY, w, h, bgColor)
	ctx.DrawList.AddRectOutline(boxX, boxY, w, h, ctx.style.InputBorderColor, 1)

	// Draw content
	textX := boxX + ctx.style.InputPadding
	textY := boxY + ctx.style.InputPadding

	if state.Editing {
		// Draw edit text with cursor
		displayText := prefix + state.EditText + suffix
		ctx.addText(textX, textY, displayText, ctx.style.TextColor)

		// Draw cursor
		if (ctx.FrameCount/30)%2 == 0 {
			cursorX := textX + ctx.MeasureText(prefix+state.EditText).X
			ctx.DrawList.AddLine(cursorX, boxY+2, cursorX, boxY+h-2, ctx.style.TextColor, 1)
		}
	} else {
		// Draw formatted value
		format := GetOpt(o, OptFormat)
		if format == "" {
			format = "%.2f"
		}
		// Handle integer format specifiers (%d) with float32 value
		var valueText string
		if strings.Contains(format, "%d") {
			valueText = fmt.Sprintf(format, int(*value))
		} else {
			valueText = fmt.Sprintf(format, *value)
		}
		displayText := prefix + valueText + suffix
		ctx.addText(textX, textY, displayText, ctx.style.TextColor)
	}

	// Draw drag indicator when hovering (not editing)
	if hovered && !state.Editing {
		// Draw left/right arrows to indicate draggable
		arrowY := boxY + h/2
		arrowSize := float32(4)
		// Left arrow
		ctx.DrawList.AddTriangle(
			boxX+3, arrowY,
			boxX+3+arrowSize, arrowY-arrowSize/2,
			boxX+3+arrowSize, arrowY+arrowSize/2,
			ctx.style.TextDisabledColor,
		)
		// Right arrow
		ctx.DrawList.AddTriangle(
			boxX+w-3, arrowY,
			boxX+w-3-arrowSize, arrowY-arrowSize/2,
			boxX+w-3-arrowSize, arrowY+arrowSize/2,
			ctx.style.TextDisabledColor,
		)
	}

	// Save state
	SetState(ctx, id, state)

	// Advance cursor
	totalWidth := labelWidth + w
	ctx.advanceCursor(Vec2{totalWidth, h})

	return changed
}

// NumberInputInt draws a numeric input field for int values.
// Returns true if the value was changed.
func (ctx *Context) NumberInputInt(label string, value *int, opts ...Option) bool {
	floatVal := float32(*value)

	// Set default format and step for integers
	hasFormat := false
	hasStep := false
	for _, opt := range opts {
		testOpts := options{}
		opt(&testOpts)
		if HasOpt(testOpts, OptFormat) {
			hasFormat = true
		}
		if HasOpt(testOpts, OptStep) {
			hasStep = true
		}
	}
	if !hasFormat {
		opts = append(opts, WithFormat("%d"))
	}
	if !hasStep {
		opts = append(opts, WithStep(1))
	}

	changed := ctx.NumberInputFloat(label, &floatVal, opts...)
	if changed {
		*value = int(floatVal)
	}
	return changed
}

// absf returns the absolute value of a float32.
func absf(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}
