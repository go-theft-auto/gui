package gui

import "fmt"

// HistogramBar represents a single bar in a histogram.
type HistogramBar struct {
	Label string
	Value float32
	Color uint32 // 0 = use default color
}

// HistogramState holds the interactive state of a histogram widget.
type HistogramState struct {
	HoveredBar int // Index of hovered bar (-1 = none)
}

// Histogram draws a bar chart for comparing values.
// height specifies the histogram height in pixels.
//
// Usage:
//
//	bars := []gui.HistogramBar{
//	    {Label: "Core 0", Value: 75, Color: gui.ColorGreen},
//	    {Label: "Core 1", Value: 45, Color: gui.ColorYellow},
//	    {Label: "Core 2", Value: 90, Color: gui.ColorRed},
//	}
//	ctx.Histogram("cpu_usage", bars, 100, gui.WithHistogramShowValues())
func (ctx *Context) Histogram(id string, bars []HistogramBar, height float32, opts ...Option) {
	if len(bars) == 0 {
		return
	}

	pos := ctx.ItemPos()
	o := applyOptions(opts)

	histID := ctx.GetID(id)

	// Get or create state
	state := GetState(ctx, histID, HistogramState{HoveredBar: -1})

	// Calculate dimensions
	w := ctx.currentLayoutWidth()
	if width := GetOpt(o, OptWidth); width > 0 {
		w = width
	}

	// Find value range
	yMin, yMax := GetOpt(o, OptHistogramYMin), GetOpt(o, OptHistogramYMax)
	if yMin == yMax {
		// Auto-calculate range
		yMin = 0
		yMax = float32(0)
		for _, bar := range bars {
			yMax = maxf(yMax, bar.Value)
		}
		if yMax == 0 {
			yMax = 1
		}
		// Add padding for value labels
		yMax *= 1.1
	}

	yRange := yMax - yMin
	if yRange == 0 {
		yRange = 1
	}

	// Draw background
	ctx.DrawList.AddRect(pos.X, pos.Y, w, height, ctx.style.InputBgColor)

	// Calculate bar dimensions
	barGap := float32(4)
	barWidth := (w - barGap*float32(len(bars)+1)) / float32(len(bars))
	if barWidth < 8 {
		barWidth = 8
		barGap = 2
	}

	// Default bar color
	defaultColor := ctx.style.SelectedBgColor

	// Track hovered bar
	state.HoveredBar = -1

	if GetOpt(o, OptHistogramHorizontal) {
		// Horizontal bars (value determines width, bars stack vertically)
		barHeight := (height - barGap*float32(len(bars)+1)) / float32(len(bars))
		if barHeight < 8 {
			barHeight = 8
		}

		for i, bar := range bars {
			barY := pos.Y + barGap + float32(i)*(barHeight+barGap)
			barW := (bar.Value - yMin) / yRange * (w - 60) // Leave room for labels
			if barW < 0 {
				barW = 0
			}

			barColor := bar.Color
			if barColor == 0 {
				barColor = defaultColor
			}

			// Check hover
			barRect := Rect{X: pos.X, Y: barY, W: w, H: barHeight}
			if ctx.Input != nil && barRect.Contains(Vec2{ctx.Input.MouseX, ctx.Input.MouseY}) {
				state.HoveredBar = i
				// Brighten on hover
				r, g, b, a := UnpackRGBA(barColor)
				barColor = RGBA(uint8(mini(int(r)+30, 255)), uint8(mini(int(g)+30, 255)), uint8(mini(int(b)+30, 255)), a)
			}

			// Draw bar
			ctx.DrawList.AddRect(pos.X+50, barY, barW, barHeight, barColor)

			// Draw label
			ctx.addText(pos.X+2, barY+(barHeight-ctx.lineHeight())/2, bar.Label, ctx.style.TextColor)

			// Draw value if enabled
			if GetOpt(o, OptHistogramShowValues) {
				valueText := fmt.Sprintf("%.1f", bar.Value)
				ctx.addText(pos.X+52+barW, barY+(barHeight-ctx.lineHeight())/2, valueText, ctx.style.TextColor)
			}
		}
	} else {
		// Vertical bars (default)
		for i, bar := range bars {
			barX := pos.X + barGap + float32(i)*(barWidth+barGap)
			barH := (bar.Value - yMin) / yRange * (height - ctx.lineHeight() - 4) // Leave room for labels
			if barH < 0 {
				barH = 0
			}
			barY := pos.Y + height - ctx.lineHeight() - 2 - barH

			barColor := bar.Color
			if barColor == 0 {
				barColor = defaultColor
			}

			// Check hover
			barRect := Rect{X: barX, Y: barY, W: barWidth, H: barH}
			if ctx.Input != nil && barRect.Contains(Vec2{ctx.Input.MouseX, ctx.Input.MouseY}) {
				state.HoveredBar = i
				// Brighten on hover
				r, g, b, a := UnpackRGBA(barColor)
				barColor = RGBA(uint8(mini(int(r)+30, 255)), uint8(mini(int(g)+30, 255)), uint8(mini(int(b)+30, 255)), a)
			}

			// Draw bar
			ctx.DrawList.AddRect(barX, barY, barWidth, barH, barColor)

			// Draw value above bar if enabled
			if GetOpt(o, OptHistogramShowValues) {
				valueText := fmt.Sprintf("%.0f", bar.Value)
				valueW := ctx.MeasureText(valueText).X
				valueX := barX + (barWidth-valueW)/2
				ctx.addText(valueX, barY-ctx.lineHeight()-2, valueText, ctx.style.TextColor)
			}

			// Draw label below bar
			labelW := ctx.MeasureText(bar.Label).X
			labelX := barX + (barWidth-labelW)/2
			if labelX < pos.X {
				labelX = pos.X
			}
			ctx.addText(labelX, pos.Y+height-ctx.lineHeight(), bar.Label, ctx.style.TextDisabledColor)
		}
	}

	// Draw tooltip for hovered bar
	if state.HoveredBar >= 0 && ctx.Input != nil {
		bar := bars[state.HoveredBar]
		tooltipText := fmt.Sprintf("%s: %.2f", bar.Label, bar.Value)
		ctx.drawHistogramTooltip(ctx.Input.MouseX+10, ctx.Input.MouseY-20, tooltipText)
	}

	// Draw border
	ctx.DrawList.AddRectOutline(pos.X, pos.Y, w, height, ctx.style.BorderColor, 1)

	// Save state
	SetState(ctx, histID, state)

	ctx.advanceCursor(Vec2{w, height})
}

// drawHistogramTooltip draws a tooltip for the histogram.
func (ctx *Context) drawHistogramTooltip(x, y float32, text string) {
	padding := float32(4)
	textSize := ctx.MeasureText(text)
	tooltipW := textSize.X + padding*2
	tooltipH := textSize.Y + padding*2

	// Keep tooltip on screen
	if x+tooltipW > ctx.DisplaySize.X {
		x = ctx.DisplaySize.X - tooltipW
	}
	if y < 0 {
		y = 0
	}

	// Draw background
	ctx.DrawList.AddRect(x, y, tooltipW, tooltipH, ctx.style.PanelColor)
	ctx.DrawList.AddRectOutline(x, y, tooltipW, tooltipH, ctx.style.PanelBorderColor, 1)

	// Draw text
	ctx.addText(x+padding, y+padding, text, ctx.style.TextColor)
}

// mini returns the minimum of two ints.
func mini(a, b int) int {
	if a < b {
		return a
	}
	return b
}
