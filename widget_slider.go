package gui

import (
	"fmt"
	"strings"
)

// sliderStore is the type-safe store for slider state.
// Uses the new FrameStore pattern instead of the old GetState/SetState.
var sliderStore = NewFrameStore[SliderState]()

// SliderFloat draws a horizontal slider for float32 values.
// Returns true if the value was changed.
//
// Usage:
//
//	if ctx.SliderFloat("Volume", &volume, 0, 1) {
//	    updateVolume(volume)
//	}
func (ctx *Context) SliderFloat(label string, value *float32, minVal, maxVal float32, opts ...Option) bool {
	pos := ctx.ItemPos()
	o := applyOptions(opts)

	id := ctx.GetID(label)
	if optID := GetOpt(o, OptID); optID != "" {
		id = ctx.GetID(optID)
	}

	// Get slider state using the new type-safe store
	state := sliderStore.Get(id, SliderState{})

	// Calculate dimensions
	labelWidth := float32(0)
	if label != "" {
		labelWidth = ctx.MeasureText(label).X + ctx.style.ItemSpacing
	}

	sliderWidth := float32(150)
	if optWidth := GetOpt(o, OptWidth); optWidth > 0 {
		sliderWidth = optWidth
	}

	trackHeight := ctx.lineHeight() * 0.5
	h := ctx.lineHeight()
	grabWidth := float32(12)
	grabHeight := h

	// Draw label
	if label != "" {
		ctx.addText(pos.X, pos.Y+(h-ctx.lineHeight())/2, label, ctx.style.TextColor)
	}

	// Track position
	trackX := pos.X + labelWidth
	trackY := pos.Y + (h-trackHeight)/2

	// Interaction rect (covers the whole slider area)
	rect := Rect{X: trackX, Y: pos.Y, W: sliderWidth, H: h}

	// Register as focusable (enables click-to-focus and keyboard navigation)
	focusable := ctx.RegisterFocusable(id, label, rect, FocusTypeLeaf)
	isFocused := focusable != nil && focusable.IsFocused()

	hovered := ctx.isHovered(id, rect)
	changed := false

	// Handle mouse input
	if ctx.Input != nil {
		// Start dragging on mouse down
		if hovered && ctx.Input.MouseClicked(MouseButtonLeft) {
			state.Dragging = true
			state.DragStartX = ctx.Input.MouseX
			state.DragStartValue = *value
		}

		// Handle dragging
		if state.Dragging {
			if ctx.Input.MouseDown(MouseButtonLeft) {
				// Calculate new value from mouse position
				relX := ctx.Input.MouseX - trackX - grabWidth/2
				ratio := clampf(relX/(sliderWidth-grabWidth), 0, 1)
				newValue := minVal + ratio*(maxVal-minVal)

				// Apply step if configured
				if step := GetOpt(o, OptStep); step > 0 {
					newValue = minVal + float32(int((newValue-minVal)/step+0.5))*step
				}

				newValue = clampf(newValue, minVal, maxVal)
				if newValue != *value {
					*value = newValue
					changed = true
				}
			} else {
				// Stop dragging on mouse release
				state.Dragging = false
			}
		}

		// Mouse wheel support when hovered
		if hovered && ctx.Input.MouseWheelY != 0 {
			step := GetOpt(o, OptStep)
			if step == 0 {
				step = (maxVal - minVal) / 100 // Default 1% step
			}
			newValue := *value + ctx.Input.MouseWheelY*step
			newValue = clampf(newValue, minVal, maxVal)
			if newValue != *value {
				*value = newValue
				changed = true
			}
		}

		// Keyboard support when focused (Left/Right arrows to adjust)
		if isFocused {
			step := GetOpt(o, OptStep)
			if step == 0 {
				step = (maxVal - minVal) / 100 // Default 1% step
			}
			if ctx.Input.KeyRepeated(KeyLeft) {
				newValue := clampf(*value-step, minVal, maxVal)
				if newValue != *value {
					*value = newValue
					changed = true
				}
			}
			if ctx.Input.KeyRepeated(KeyRight) {
				newValue := clampf(*value+step, minVal, maxVal)
				if newValue != *value {
					*value = newValue
					changed = true
				}
			}
		}
	}

	// Calculate grab position
	ratio := float32(0)
	if maxVal > minVal {
		ratio = (*value - minVal) / (maxVal - minVal)
	}
	grabX := trackX + ratio*(sliderWidth-grabWidth)

	// Draw track background
	ctx.DrawList.AddRect(trackX, trackY, sliderWidth, trackHeight, ctx.style.SliderTrackColor)

	// Draw filled portion
	fillWidth := ratio * sliderWidth
	if fillWidth > 0 {
		ctx.DrawList.AddRect(trackX, trackY, fillWidth, trackHeight, ctx.style.SliderFillColor)
	}

	// Draw grab handle
	grabColor := ctx.style.SliderGrabColor
	if state.Dragging {
		grabColor = ctx.style.SliderGrabActive
	} else if hovered || isFocused {
		grabColor = ctx.style.SliderGrabHovered
	}
	ctx.DrawList.AddRect(grabX, pos.Y, grabWidth, grabHeight, grabColor)
	ctx.DrawList.AddRectOutline(grabX, pos.Y, grabWidth, grabHeight, ctx.style.InputBorderColor, 1)

	// Draw value text
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
	valueWidth := ctx.MeasureText(valueText).X
	ctx.addText(trackX+sliderWidth+ctx.style.ItemSpacing, pos.Y, valueText, ctx.style.TextColor)

	// State is automatically saved via pointer (no need to call SetState)

	// Advance cursor
	totalWidth := labelWidth + sliderWidth + ctx.style.ItemSpacing + valueWidth
	ctx.advanceCursor(Vec2{totalWidth, h})

	return changed
}

// SliderInt draws a horizontal slider for int values.
// Returns true if the value was changed.
//
// Usage:
//
//	if ctx.SliderInt("Count", &count, 0, 100) {
//	    updateCount(count)
//	}
func (ctx *Context) SliderInt(label string, value *int, minVal, maxVal int, opts ...Option) bool {
	// Convert to float for internal handling
	floatVal := float32(*value)
	opts = append(opts, WithStep(1)) // Force integer steps

	// Use format for integers if not specified
	found := false
	for _, opt := range opts {
		testOpts := options{}
		opt(&testOpts)
		if GetOpt(testOpts, OptFormat) != "" {
			found = true
			break
		}
	}
	if !found {
		opts = append(opts, WithFormat("%d"))
	}

	changed := ctx.SliderFloat(label, &floatVal, float32(minVal), float32(maxVal), opts...)
	if changed {
		*value = int(floatVal)
	}
	return changed
}

// GetSliderState returns a pointer to the slider's state for advanced manipulation.
// Returns nil if the slider hasn't been rendered yet this frame.
func GetSliderState(ctx *Context, label string) *SliderState {
	id := ctx.GetID(label)
	return sliderStore.GetIfExists(id)
}
