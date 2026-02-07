package gui

import "fmt"

// SequencerTrack represents a single track (e.g., bone animation) in the sequencer.
type SequencerTrack struct {
	Name      string
	Keyframes []float32 // Times in seconds where keyframes exist
	Color     uint32    // Track color (0 = default)
}

// SequencerConfig holds the configuration for a sequencer widget.
type SequencerConfig struct {
	Duration    float32          // Total duration in seconds
	CurrentTime float32          // Current playhead position
	Tracks      []SequencerTrack // Animation tracks
	Playing     bool             // True if playing

	// Callbacks (optional)
	OnSeek  func(time float32) // Called when user seeks
	OnPlay  func()             // Called when play is pressed
	OnPause func()             // Called when pause is pressed
}

// SequencerState holds the interactive state of a sequencer widget.
type SequencerState struct {
	ZoomLevel       float32         // Zoom factor (1.0 = fit duration to width)
	PanOffsetX      float32         // Horizontal pan offset in pixels
	CollapsedTracks map[string]bool // Track collapse state (true = collapsed)
	SelectedTrack   string          // Name of selected track
	SelectedKeyIdx  int             // Index of selected keyframe (-1 = none)
	Scrubbing       bool            // True when dragging playhead
	HoveredTrack    string          // Name of hovered track
	HoveredKeyIdx   int             // Index of hovered keyframe (-1 = none)
}

// Sequencer draws an animation timeline with tracks and keyframes.
// height specifies the total sequencer height in pixels.
// Returns true if the current time changed due to user interaction.
//
// Layout:
//
//	+------------------------------------------+
//	| [>] [||]    | 0:00      0:30      1:00  |  <- Controls + time ruler
//	+------------------------------------------+
//	| v Root      |    o----o-------o----     |  <- Track with keyframes
//	|   Pelvis    |      o-------o--------    |
//	| > L_Leg     |    ---------------        |  <- Collapsed track
//	+------------------------------------------+
//	                   ^
//	                   | Playhead (red line)
func (ctx *Context) Sequencer(id string, config SequencerConfig, height float32, opts ...Option) bool {
	if config.Duration <= 0 {
		return false
	}

	pos := ctx.ItemPos()
	o := applyOptions(opts)

	seqID := ctx.GetID(id)

	// Get or create state
	state := GetState(ctx, seqID, SequencerState{
		ZoomLevel:       1.0,
		CollapsedTracks: make(map[string]bool),
		SelectedKeyIdx:  -1,
		HoveredKeyIdx:   -1,
	})

	// Calculate dimensions
	w := ctx.currentLayoutWidth()
	if width := GetOpt(o, OptWidth); width > 0 {
		w = width
	}

	// Layout constants
	const (
		trackLabelWidth = float32(120) // Width for track names
		rulerHeight     = float32(24)  // Height of time ruler
		trackHeight     = float32(24)  // Height of each track row
		controlsHeight  = float32(28)  // Height of play/pause controls
		keyframeRadius  = float32(4)   // Radius of keyframe markers
	)

	timelineX := pos.X + trackLabelWidth
	timelineW := w - trackLabelWidth
	if timelineW < 50 {
		timelineW = 50
	}

	// Draw background
	ctx.DrawList.AddRect(pos.X, pos.Y, w, height, ctx.style.InputBgColor)

	currentY := pos.Y

	// === Controls Bar ===
	if GetOpt(o, OptSequencerControls) {
		ctx.drawSequencerControls(pos.X, currentY, trackLabelWidth, controlsHeight, &config, &state)
		currentY += controlsHeight
	}

	// === Time Ruler ===
	ctx.drawSequencerRuler(timelineX, currentY, timelineW, rulerHeight, config.Duration, state.ZoomLevel, state.PanOffsetX)
	currentY += rulerHeight

	// === Track Labels Column Background ===
	tracksAreaY := currentY
	tracksAreaH := height - (currentY - pos.Y)
	ctx.DrawList.AddRect(pos.X, tracksAreaY, trackLabelWidth, tracksAreaH, RGBA(30, 30, 35, 255))

	// === Tracks ===
	trackY := currentY
	for i, track := range config.Tracks {
		if trackY > pos.Y+height-trackHeight {
			break // No more room
		}

		isCollapsed := state.CollapsedTracks[track.Name]
		isSelected := state.SelectedTrack == track.Name
		isHovered := state.HoveredTrack == track.Name

		// Track label background
		labelBg := RGBA(35, 35, 40, 255)
		if isSelected {
			labelBg = ctx.style.SelectedBgColor
		} else if isHovered {
			labelBg = ctx.style.HoveredBgColor
		}
		ctx.DrawList.AddRect(pos.X, trackY, trackLabelWidth, trackHeight, labelBg)

		// Collapse indicator and label
		indicator := "v"
		if isCollapsed {
			indicator = ">"
		}
		ctx.addText(pos.X+4, trackY+(trackHeight-ctx.lineHeight())/2, indicator, ctx.style.TextDisabledColor)
		ctx.addText(pos.X+16, trackY+(trackHeight-ctx.lineHeight())/2, track.Name, ctx.style.TextColor)

		// Track timeline background (alternating)
		timelineBg := RGBA(25, 25, 30, 255)
		if i%2 == 1 {
			timelineBg = RGBA(30, 30, 35, 255)
		}
		ctx.DrawList.AddRect(timelineX, trackY, timelineW, trackHeight, timelineBg)

		// Draw keyframes (if not collapsed)
		if !isCollapsed {
			trackColor := track.Color
			if trackColor == 0 {
				trackColor = ctx.style.SelectedBgColor
			}

			// Draw single duration bar spanning from first to last keyframe
			barHeight := trackHeight * 0.6
			barY := trackY + (trackHeight-barHeight)/2

			if len(track.Keyframes) >= 1 {
				// Get first and last keyframe times
				firstTime := track.Keyframes[0]
				lastTime := track.Keyframes[len(track.Keyframes)-1]

				x1 := ctx.sequencerTimeToX(firstTime, timelineX, timelineW, config.Duration, state.ZoomLevel, state.PanOffsetX)
				x2 := ctx.sequencerTimeToX(lastTime, timelineX, timelineW, config.Duration, state.ZoomLevel, state.PanOffsetX)

				// Clamp to visible area
				x1 = maxf(x1, timelineX)
				x2 = minf(x2, timelineX+timelineW)

				if x1 < x2 {
					// Duration bar color (semi-transparent)
					barColor := trackColor
					r, g, b, _ := UnpackRGBA(barColor)
					barColor = RGBA(r, g, b, 140)

					// Highlight if track is selected or hovered
					if state.SelectedTrack == track.Name {
						barColor = RGBA(r, g, b, 200)
					} else if state.HoveredTrack == track.Name {
						barColor = RGBA(uint8(mini(int(r)+40, 255)), uint8(mini(int(g)+40, 255)), uint8(mini(int(b)+40, 255)), 180)
					}

					ctx.DrawList.AddRect(x1, barY, x2-x1, barHeight, barColor)
				}
			}

			// Draw keyframe markers at each keyframe point (small diamonds on top of bar)
			for _, kfTime := range track.Keyframes {
				kfX := ctx.sequencerTimeToX(kfTime, timelineX, timelineW, config.Duration, state.ZoomLevel, state.PanOffsetX)
				if kfX < timelineX-keyframeRadius || kfX > timelineX+timelineW+keyframeRadius {
					continue // Off screen
				}

				kfY := trackY + trackHeight/2

				// Keyframe marker (bright point at exact keyframe time)
				markerColor := trackColor
				ctx.DrawList.AddRect(kfX-keyframeRadius, kfY-keyframeRadius, keyframeRadius*2, keyframeRadius*2, markerColor)
				// Add outline for visibility
				ctx.DrawList.AddRectOutline(kfX-keyframeRadius, kfY-keyframeRadius, keyframeRadius*2, keyframeRadius*2, RGBA(255, 255, 255, 150), 1)
			}
		}

		// Handle track hover/click
		trackRect := Rect{X: pos.X, Y: trackY, W: trackLabelWidth, H: trackHeight}
		if ctx.Input != nil && trackRect.Contains(Vec2{ctx.Input.MouseX, ctx.Input.MouseY}) {
			state.HoveredTrack = track.Name
			if ctx.Input.MouseClicked(MouseButtonLeft) {
				state.SelectedTrack = track.Name
				// Toggle collapse on double-click or click on indicator
				if ctx.Input.MouseX < pos.X+16 {
					state.CollapsedTracks[track.Name] = !isCollapsed
				}
			}
		}

		trackY += trackHeight
	}

	// === Playhead ===
	playheadX := ctx.sequencerTimeToX(config.CurrentTime, timelineX, timelineW, config.Duration, state.ZoomLevel, state.PanOffsetX)
	// Round to nearest pixel to prevent subpixel flickering during animation
	playheadX = float32(int(playheadX + 0.5))
	if playheadX >= timelineX && playheadX <= timelineX+timelineW {
		playheadColor := RGBA(255, 50, 50, 255) // Bright red
		playheadLineWidth := float32(3)
		playheadHandleSize := float32(10)

		// Calculate Y coordinates for the playhead
		playheadTopY := tracksAreaY - rulerHeight // Top of ruler
		playheadBottomY := pos.Y + height         // Bottom of sequencer

		// Draw playhead line as a filled rect (more reliable than AddLine)
		ctx.DrawList.AddRect(
			playheadX-playheadLineWidth/2,
			playheadTopY,
			playheadLineWidth,
			playheadBottomY-playheadTopY,
			playheadColor,
		)

		// Playhead handle (larger filled triangle at top)
		ctx.DrawList.AddTriangle(
			playheadX-playheadHandleSize, playheadTopY,
			playheadX+playheadHandleSize, playheadTopY,
			playheadX, playheadTopY+playheadHandleSize*1.5,
			playheadColor,
		)

		// Add white outline to handle for better visibility
		ctx.DrawList.AddLine(playheadX-playheadHandleSize, playheadTopY, playheadX+playheadHandleSize, playheadTopY, ColorWhite, 1)
		ctx.DrawList.AddLine(playheadX+playheadHandleSize, playheadTopY, playheadX, playheadTopY+playheadHandleSize*1.5, ColorWhite, 1)
		ctx.DrawList.AddLine(playheadX, playheadTopY+playheadHandleSize*1.5, playheadX-playheadHandleSize, playheadTopY, ColorWhite, 1)
	}

	// === Handle Input ===
	changed := false
	timelineRect := Rect{X: timelineX, Y: pos.Y, W: timelineW, H: height}

	if ctx.Input != nil {
		// Scrubbing (click/drag on timeline)
		if timelineRect.Contains(Vec2{ctx.Input.MouseX, ctx.Input.MouseY}) {
			if ctx.Input.MouseClicked(MouseButtonLeft) {
				state.Scrubbing = true
			}

			// Mouse wheel for zoom
			if ctx.Input.MouseWheelY != 0 {
				oldZoom := state.ZoomLevel
				state.ZoomLevel *= 1 + ctx.Input.MouseWheelY*0.1
				state.ZoomLevel = clampf(state.ZoomLevel, 0.1, 10.0)

				// Adjust pan to keep mouse position stable
				mouseRelX := ctx.Input.MouseX - timelineX
				state.PanOffsetX = mouseRelX - (mouseRelX-state.PanOffsetX)*(state.ZoomLevel/oldZoom)
			}

			// Hover detection for keyframes
			state.HoveredKeyIdx = -1
			for _, track := range config.Tracks {
				if state.CollapsedTracks[track.Name] {
					continue
				}
				for ki, kfTime := range track.Keyframes {
					kfX := ctx.sequencerTimeToX(kfTime, timelineX, timelineW, config.Duration, state.ZoomLevel, state.PanOffsetX)
					if absf32(ctx.Input.MouseX-kfX) < keyframeRadius*2 {
						state.HoveredTrack = track.Name
						state.HoveredKeyIdx = ki
						break
					}
				}
			}
		}

		if state.Scrubbing {
			if ctx.Input.MouseDown(MouseButtonLeft) {
				newTime := ctx.sequencerXToTime(ctx.Input.MouseX, timelineX, timelineW, config.Duration, state.ZoomLevel, state.PanOffsetX)
				newTime = clampf(newTime, 0, config.Duration)
				if newTime != config.CurrentTime {
					config.CurrentTime = newTime
					if config.OnSeek != nil {
						config.OnSeek(newTime)
					}
					changed = true
				}
			} else {
				state.Scrubbing = false
			}
		}

		// Space to toggle play/pause
		if ctx.Input.KeyPressed(KeySpace) && timelineRect.Contains(Vec2{ctx.Input.MouseX, ctx.Input.MouseY}) {
			if config.Playing {
				if config.OnPause != nil {
					config.OnPause()
				}
			} else {
				if config.OnPlay != nil {
					config.OnPlay()
				}
			}
		}
	}

	// Draw border
	ctx.DrawList.AddRectOutline(pos.X, pos.Y, w, height, ctx.style.BorderColor, 1)

	// Draw separator between label area and timeline
	ctx.DrawList.AddLine(timelineX, pos.Y, timelineX, pos.Y+height, ctx.style.BorderColor, 1)

	// Save state
	SetState(ctx, seqID, state)

	ctx.advanceCursor(Vec2{w, height})

	return changed
}

// drawSequencerControls draws the play/pause controls.
func (ctx *Context) drawSequencerControls(x, y, labelW, h float32, config *SequencerConfig, _ *SequencerState) {
	// Background
	ctx.DrawList.AddRect(x, y, labelW, h, RGBA(40, 40, 45, 255))

	// Play/Pause button
	btnSize := h - 4
	btnX := x + 4
	btnY := y + 2

	btnRect := Rect{X: btnX, Y: btnY, W: btnSize, H: btnSize}
	hovered := ctx.Input != nil && btnRect.Contains(Vec2{ctx.Input.MouseX, ctx.Input.MouseY})

	btnColor := ctx.style.ButtonColor
	if hovered {
		btnColor = ctx.style.ButtonHoveredColor
	}
	ctx.DrawList.AddRect(btnX, btnY, btnSize, btnSize, btnColor)

	// Draw play or pause icon
	iconColor := ctx.style.TextColor
	if config.Playing {
		// Pause icon (two vertical bars)
		barW := btnSize * 0.15
		gap := btnSize * 0.2
		ctx.DrawList.AddRect(btnX+gap, btnY+gap, barW, btnSize-gap*2, iconColor)
		ctx.DrawList.AddRect(btnX+btnSize-gap-barW, btnY+gap, barW, btnSize-gap*2, iconColor)
	} else {
		// Play icon (triangle)
		ctx.DrawList.AddTriangle(
			btnX+btnSize*0.3, btnY+btnSize*0.2,
			btnX+btnSize*0.3, btnY+btnSize*0.8,
			btnX+btnSize*0.8, btnY+btnSize*0.5,
			iconColor,
		)
	}

	// Handle click
	if ctx.Input != nil && hovered && ctx.Input.MouseClicked(MouseButtonLeft) {
		if config.Playing {
			if config.OnPause != nil {
				config.OnPause()
			}
		} else {
			if config.OnPlay != nil {
				config.OnPlay()
			}
		}
	}

	// Time display
	timeText := formatTime(config.CurrentTime) + " / " + formatTime(config.Duration)
	ctx.addText(btnX+btnSize+8, y+(h-ctx.lineHeight())/2, timeText, ctx.style.TextColor)
}

// drawSequencerRuler draws the time ruler.
func (ctx *Context) drawSequencerRuler(x, y, w, h, duration, zoom, pan float32) {
	// Background
	ctx.DrawList.AddRect(x, y, w, h, RGBA(35, 35, 40, 255))

	// Calculate tick spacing
	visibleDuration := duration / zoom
	tickSpacing := calculateTickSpacing(visibleDuration, w)

	// Draw ticks and labels
	startTime := maxf(0, -pan/(w/visibleDuration))
	endTime := minf(duration, startTime+visibleDuration)

	// Round start time down to nearest tick
	startTick := float32(int(startTime/tickSpacing)) * tickSpacing

	for t := startTick; t <= endTime; t += tickSpacing {
		tickX := x + (t-startTime)*(w/visibleDuration) + pan
		if tickX < x || tickX > x+w {
			continue
		}

		// Draw tick mark
		tickH := h * 0.3
		if int(t*10)%int(tickSpacing*50) == 0 {
			tickH = h * 0.6 // Major tick
		}
		ctx.DrawList.AddLine(tickX, y+h-tickH, tickX, y+h, ctx.style.TextDisabledColor, 1)

		// Draw time label for major ticks
		if int(t*10)%int(tickSpacing*50) == 0 || tickSpacing >= 1 {
			label := formatTime(t)
			ctx.addText(tickX+2, y+2, label, ctx.style.TextDisabledColor)
		}
	}

	// Bottom border
	ctx.DrawList.AddLine(x, y+h, x+w, y+h, ctx.style.BorderColor, 1)
}

// sequencerTimeToX converts a time value to an X coordinate.
func (ctx *Context) sequencerTimeToX(time, timelineX, timelineW, duration, zoom, pan float32) float32 {
	visibleDuration := duration / zoom
	return timelineX + (time/visibleDuration)*timelineW + pan
}

// sequencerXToTime converts an X coordinate to a time value.
func (ctx *Context) sequencerXToTime(x, timelineX, timelineW, duration, zoom, pan float32) float32 {
	visibleDuration := duration / zoom
	return ((x - timelineX - pan) / timelineW) * visibleDuration
}

// calculateTickSpacing determines appropriate tick spacing based on visible duration.
func calculateTickSpacing(visibleDuration, _ float32) float32 {
	// Target about 10 ticks visible
	targetTicks := 10
	rawSpacing := visibleDuration / float32(targetTicks)

	// Round to nice values
	if rawSpacing < 0.1 {
		return 0.05
	} else if rawSpacing < 0.5 {
		return 0.1
	} else if rawSpacing < 1 {
		return 0.5
	} else if rawSpacing < 5 {
		return 1
	} else if rawSpacing < 10 {
		return 5
	} else if rawSpacing < 30 {
		return 10
	} else if rawSpacing < 60 {
		return 30
	}
	return 60
}

// formatTime formats a time in seconds as MM:SS or SS.ms
func formatTime(seconds float32) string {
	if seconds < 60 {
		return fmt.Sprintf("%.1fs", seconds)
	}
	mins := int(seconds) / 60
	secs := int(seconds) % 60
	return fmt.Sprintf("%d:%02d", mins, secs)
}
