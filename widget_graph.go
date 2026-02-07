package gui

import "fmt"

// GraphData represents a single data series in a graph.
type GraphData struct {
	Label  string
	Values []float32
	Color  uint32
}

// GraphState holds the interactive state of a graph widget.
type GraphState struct {
	HoveredIndex int     // Index of hovered data point (-1 = none)
	ZoomLevel    float32 // Zoom factor (1.0 = no zoom)
	PanOffset    float32 // Horizontal pan offset in pixels
}

// Graph draws a line graph for time-series data.
// height specifies the graph height in pixels.
//
// Usage:
//
//	data := []gui.GraphData{
//	    {Label: "FPS", Values: fpsHistory, Color: gui.ColorGreen},
//	    {Label: "Frame Time", Values: frameTimeHistory, Color: gui.ColorYellow},
//	}
//	ctx.Graph("perf_graph", data, 100, gui.WithGraphGridLines(4))
func (ctx *Context) Graph(id string, data []GraphData, height float32, opts ...Option) {
	if len(data) == 0 {
		return
	}

	pos := ctx.ItemPos()
	o := applyOptions(opts)

	graphID := ctx.GetID(id)

	// Get or create state
	state := GetState(ctx, graphID, GraphState{
		HoveredIndex: -1,
		ZoomLevel:    1.0,
	})

	// Calculate dimensions
	w := ctx.currentLayoutWidth()
	if width := GetOpt(o, OptWidth); width > 0 {
		w = width
	}

	// Find data range
	yMin, yMax := GetOpt(o, OptGraphYMin), GetOpt(o, OptGraphYMax)
	maxLen := 0
	if yMin == yMax {
		// Auto-calculate range
		yMin = float32(1e9)
		yMax = float32(-1e9)
		for _, series := range data {
			for _, v := range series.Values {
				yMin = minf(yMin, v)
				yMax = maxf(yMax, v)
			}
			maxLen = maxi(maxLen, len(series.Values))
		}
		// Add some padding
		padding := (yMax - yMin) * 0.1
		if padding == 0 {
			padding = 1
		}
		yMin -= padding
		yMax += padding
	} else {
		for _, series := range data {
			maxLen = maxi(maxLen, len(series.Values))
		}
	}

	if maxLen < 2 {
		// Need at least 2 points to draw a line
		ctx.advanceCursor(Vec2{w, height})
		return
	}

	// Draw background
	ctx.DrawList.AddRect(pos.X, pos.Y, w, height, ctx.style.InputBgColor)

	// Draw grid lines
	if gridLines := GetOpt(o, OptGraphGridLines); gridLines > 0 {
		gridColor := RGBA(80, 80, 80, 100)
		for i := 0; i <= gridLines; i++ {
			y := pos.Y + height*float32(i)/float32(gridLines)
			ctx.DrawList.AddLine(pos.X, y, pos.X+w, y, gridColor, 1)
		}
	}

	// Draw data series
	yRange := yMax - yMin
	if yRange == 0 {
		yRange = 1
	}

	for _, series := range data {
		if len(series.Values) < 2 {
			continue
		}

		// Draw line connecting points
		for i := 1; i < len(series.Values); i++ {
			x1 := pos.X + float32(i-1)*w/float32(maxLen-1)
			x2 := pos.X + float32(i)*w/float32(maxLen-1)
			y1 := pos.Y + height - (series.Values[i-1]-yMin)/yRange*height
			y2 := pos.Y + height - (series.Values[i]-yMin)/yRange*height

			ctx.DrawList.AddLine(x1, y1, x2, y2, series.Color, 1.5)
		}
	}

	// Handle hover interaction
	graphRect := Rect{X: pos.X, Y: pos.Y, W: w, H: height}
	state.HoveredIndex = -1

	if ctx.Input != nil && graphRect.Contains(Vec2{ctx.Input.MouseX, ctx.Input.MouseY}) {
		// Calculate which data index is hovered
		relX := ctx.Input.MouseX - pos.X
		idx := int(relX/w*float32(maxLen-1) + 0.5)
		if idx >= 0 && idx < maxLen {
			state.HoveredIndex = idx

			// Draw vertical line at hover position
			hoverX := pos.X + float32(idx)*w/float32(maxLen-1)
			ctx.DrawList.AddLine(hoverX, pos.Y, hoverX, pos.Y+height, RGBA(255, 255, 255, 100), 1)

			// Draw tooltip
			tooltipY := ctx.Input.MouseY - 20
			tooltipLines := make([]string, 0, len(data))
			for _, series := range data {
				if idx < len(series.Values) {
					tooltipLines = append(tooltipLines, fmt.Sprintf("%s: %.2f", series.Label, series.Values[idx]))
				}
			}
			if len(tooltipLines) > 0 {
				ctx.drawGraphTooltip(ctx.Input.MouseX+10, tooltipY, tooltipLines)
			}
		}
	}

	// Draw legend if enabled
	if GetOpt(o, OptGraphLegend) && len(data) > 1 {
		legendX := pos.X + 4
		legendY := pos.Y + 4
		for _, series := range data {
			// Draw color indicator
			ctx.DrawList.AddRect(legendX, legendY+2, 8, 8, series.Color)
			// Draw label
			ctx.addText(legendX+12, legendY, series.Label, ctx.style.TextColor)
			legendY += ctx.lineHeight()
		}
	}

	// Draw Y-axis labels (min/max)
	labelColor := ctx.style.TextDisabledColor
	ctx.addText(pos.X+2, pos.Y+2, fmt.Sprintf("%.1f", yMax), labelColor)
	ctx.addText(pos.X+2, pos.Y+height-ctx.lineHeight()-2, fmt.Sprintf("%.1f", yMin), labelColor)

	// Draw border
	ctx.DrawList.AddRectOutline(pos.X, pos.Y, w, height, ctx.style.BorderColor, 1)

	// Save state
	SetState(ctx, graphID, state)

	ctx.advanceCursor(Vec2{w, height})
}

// drawGraphTooltip draws a tooltip with multiple lines.
func (ctx *Context) drawGraphTooltip(x, y float32, lines []string) {
	if len(lines) == 0 {
		return
	}

	// Calculate tooltip size
	maxWidth := float32(0)
	for _, line := range lines {
		w := ctx.MeasureText(line).X
		maxWidth = maxf(maxWidth, w)
	}

	padding := float32(4)
	tooltipW := maxWidth + padding*2
	tooltipH := float32(len(lines))*ctx.lineHeight() + padding*2

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
	textY := y + padding
	for _, line := range lines {
		ctx.addText(x+padding, textY, line, ctx.style.TextColor)
		textY += ctx.lineHeight()
	}
}

// maxi returns the maximum of two ints.
func maxi(a, b int) int {
	if a > b {
		return a
	}
	return b
}
