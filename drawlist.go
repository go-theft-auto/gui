package gui

import "sync"

// drawListPool provides efficient reuse of DrawList buffers.
// This avoids allocations on every frame, which is critical for
// immediate-mode UI where we rebuild the entire draw list each frame.
var drawListPool = sync.Pool{
	New: func() any {
		return &DrawList{
			VtxBuffer: make([]Vertex, 0, 1024),
			IdxBuffer: make([]uint16, 0, 2048),
			CmdBuffer: make([]DrawCmd, 0, 16),
			clipStack: make([][4]float32, 0, 8),
		}
	},
}

// AcquireDrawList gets a DrawList from the pool.
// Call ReleaseDrawList when done to return it.
func AcquireDrawList() *DrawList {
	dl := drawListPool.Get().(*DrawList)
	dl.Clear()
	return dl
}

// ReleaseDrawList returns a DrawList to the pool for reuse.
func ReleaseDrawList(dl *DrawList) {
	if dl != nil {
		drawListPool.Put(dl)
	}
}

// DrawList accumulates draw commands for a frame.
// It batches primitives by texture to minimize GPU state changes.
type DrawList struct {
	CmdBuffer []DrawCmd // Draw commands
	VtxBuffer []Vertex  // Vertex data
	IdxBuffer []uint16  // Index data

	clipStack    [][4]float32 // Clip rectangle stack
	currentClip  [4]float32   // Current clip rectangle
	textureID    uint32       // Current texture for batching
	cmdOffset    uint32       // Vertex offset for current command
	idxCmdOffset uint32       // Index offset for current command
}

// Clear resets the DrawList for a new frame.
// Retains allocated capacity to avoid reallocations.
func (dl *DrawList) Clear() {
	dl.CmdBuffer = dl.CmdBuffer[:0]
	dl.VtxBuffer = dl.VtxBuffer[:0]
	dl.IdxBuffer = dl.IdxBuffer[:0]
	dl.clipStack = dl.clipStack[:0]
	dl.currentClip = [4]float32{-1e9, -1e9, 1e9, 1e9} // Very large default clip
	dl.textureID = 0
	dl.cmdOffset = 0
	dl.idxCmdOffset = 0
}

// PushClipRect pushes a new clip rectangle onto the stack.
// All subsequent primitives will be clipped to this rectangle.
func (dl *DrawList) PushClipRect(x1, y1, x2, y2 float32) {
	dl.clipStack = append(dl.clipStack, dl.currentClip)
	dl.currentClip = [4]float32{x1, y1, x2, y2}
	dl.splitDraw() // Force new command with new clip rect
}

// PopClipRect pops the clip rectangle stack.
func (dl *DrawList) PopClipRect() {
	n := len(dl.clipStack)
	if n > 0 {
		dl.currentClip = dl.clipStack[n-1]
		dl.clipStack = dl.clipStack[:n-1]
		dl.splitDraw() // Force new command with restored clip rect
	}
}

// SetTexture sets the current texture for subsequent primitives.
func (dl *DrawList) SetTexture(textureID uint32) {
	if dl.textureID != textureID {
		// Finalize any pending primitives with the old texture first
		if len(dl.CmdBuffer) > 0 {
			lastCmd := &dl.CmdBuffer[len(dl.CmdBuffer)-1]
			lastCmd.ElemCount = uint32(len(dl.IdxBuffer)) - dl.idxCmdOffset
		}
		// Update texture ID for the new command
		dl.textureID = textureID
		// Create new command with the new texture ID
		dl.CmdBuffer = append(dl.CmdBuffer, DrawCmd{
			ClipRect:     dl.currentClip,
			TextureID:    dl.textureID,
			VertexOffset: uint32(len(dl.VtxBuffer)),
			IndexOffset:  uint32(len(dl.IdxBuffer)),
		})
		dl.cmdOffset = uint32(len(dl.VtxBuffer))
		dl.idxCmdOffset = uint32(len(dl.IdxBuffer))
	}
}

// splitDraw finalizes the current command and starts a new one.
func (dl *DrawList) splitDraw() {
	// Finalize current command if it has any indices
	if len(dl.CmdBuffer) > 0 {
		lastCmd := &dl.CmdBuffer[len(dl.CmdBuffer)-1]
		lastCmd.ElemCount = uint32(len(dl.IdxBuffer)) - dl.idxCmdOffset
	}

	// Start new command
	dl.CmdBuffer = append(dl.CmdBuffer, DrawCmd{
		ClipRect:     dl.currentClip,
		TextureID:    dl.textureID,
		VertexOffset: uint32(len(dl.VtxBuffer)),
		IndexOffset:  uint32(len(dl.IdxBuffer)),
	})
	dl.cmdOffset = uint32(len(dl.VtxBuffer))
	dl.idxCmdOffset = uint32(len(dl.IdxBuffer))
}

// ensureCommand ensures there's an active draw command.
func (dl *DrawList) ensureCommand() {
	if len(dl.CmdBuffer) == 0 {
		dl.splitDraw()
	}
}

// addVertices adds vertices and returns the starting index.
func (dl *DrawList) addVertices(verts ...Vertex) uint16 {
	dl.ensureCommand()
	startIdx := uint16(len(dl.VtxBuffer) - int(dl.cmdOffset))
	dl.VtxBuffer = append(dl.VtxBuffer, verts...)
	return startIdx
}

// addIndices adds indices (relative to current command's vertex offset).
func (dl *DrawList) addIndices(indices ...uint16) {
	dl.IdxBuffer = append(dl.IdxBuffer, indices...)
}

// AddRect draws a filled rectangle.
func (dl *DrawList) AddRect(x, y, w, h float32, color uint32) {
	if color&0xFF000000 == 0 { // Skip fully transparent
		return
	}

	idx := dl.addVertices(
		Vertex{Pos: [2]float32{x, y}, Color: color},
		Vertex{Pos: [2]float32{x + w, y}, Color: color},
		Vertex{Pos: [2]float32{x + w, y + h}, Color: color},
		Vertex{Pos: [2]float32{x, y + h}, Color: color},
	)

	dl.addIndices(idx, idx+1, idx+2, idx, idx+2, idx+3)
}

// AddRectOutline draws a rectangle outline.
func (dl *DrawList) AddRectOutline(x, y, w, h float32, color uint32, thickness float32) {
	if color&0xFF000000 == 0 {
		return
	}

	// Top edge
	dl.AddRect(x, y, w, thickness, color)
	// Bottom edge
	dl.AddRect(x, y+h-thickness, w, thickness, color)
	// Left edge
	dl.AddRect(x, y+thickness, thickness, h-2*thickness, color)
	// Right edge
	dl.AddRect(x+w-thickness, y+thickness, thickness, h-2*thickness, color)
}

// AddLine draws a line between two points.
// Uses a quad to create thickness.
func (dl *DrawList) AddLine(x1, y1, x2, y2 float32, color uint32, thickness float32) {
	if color&0xFF000000 == 0 {
		return
	}

	// Calculate perpendicular direction for thickness
	dx := x2 - x1
	dy := y2 - y1
	len := float32(1.0)
	if dx != 0 || dy != 0 {
		len = 1.0 / sqrtf(dx*dx+dy*dy)
	}

	// Normal perpendicular to line
	nx := -dy * len * thickness * 0.5
	ny := dx * len * thickness * 0.5

	idx := dl.addVertices(
		Vertex{Pos: [2]float32{x1 + nx, y1 + ny}, Color: color},
		Vertex{Pos: [2]float32{x2 + nx, y2 + ny}, Color: color},
		Vertex{Pos: [2]float32{x2 - nx, y2 - ny}, Color: color},
		Vertex{Pos: [2]float32{x1 - nx, y1 - ny}, Color: color},
	)

	dl.addIndices(idx, idx+1, idx+2, idx, idx+2, idx+3)
}

// AddTriangle draws a filled triangle.
func (dl *DrawList) AddTriangle(x1, y1, x2, y2, x3, y3 float32, color uint32) {
	if color&0xFF000000 == 0 {
		return
	}

	idx := dl.addVertices(
		Vertex{Pos: [2]float32{x1, y1}, Color: color},
		Vertex{Pos: [2]float32{x2, y2}, Color: color},
		Vertex{Pos: [2]float32{x3, y3}, Color: color},
	)

	dl.addIndices(idx, idx+1, idx+2)
}

// AddText draws text at the specified position.
// fontScale is typically 1.0 for normal size.
// charWidth and charHeight define the size of each character cell.
func (dl *DrawList) AddText(x, y float32, text string, color uint32, fontScale float32, charWidth, charHeight float32) {
	if color&0xFF000000 == 0 || len(text) == 0 {
		return
	}

	cw := charWidth * fontScale
	cellH := charHeight * fontScale

	for i, r := range text {
		// Map character to texture coordinates
		// Assumes a 16x6 grid of 8x8 characters for ASCII 32-127
		char := unicodeFallback(r)
		if char < 32 || char > 127 {
			char = '?'
		}

		idx := int(char - 32)
		col := float32(idx % 16)
		row := float32(idx / 16)

		// Texture coordinates (16x6 grid in 128x48 texture)
		u0 := col * 8 / 128
		v0 := row * 8 / 48
		u1 := (col + 1) * 8 / 128
		v1 := (row + 1) * 8 / 48

		px := x + float32(i)*cw

		vtxIdx := dl.addVertices(
			Vertex{Pos: [2]float32{px, y}, TexCoord: [2]float32{u0, v0}, Color: color},
			Vertex{Pos: [2]float32{px + cw, y}, TexCoord: [2]float32{u1, v0}, Color: color},
			Vertex{Pos: [2]float32{px + cw, y + cellH}, TexCoord: [2]float32{u1, v1}, Color: color},
			Vertex{Pos: [2]float32{px, y + cellH}, TexCoord: [2]float32{u0, v1}, Color: color},
		)

		dl.addIndices(vtxIdx, vtxIdx+1, vtxIdx+2, vtxIdx, vtxIdx+2, vtxIdx+3)
	}
}

// unicodeFallback maps common Unicode symbols to ASCII equivalents
// for the built-in bitmap font (ASCII 32-127 only).
func unicodeFallback(r rune) rune {
	if r >= 32 && r <= 127 {
		return r
	}
	switch r {
	case '►', '▶', '▸', '→', '⯈':
		return '>'
	case '◄', '◀', '◂', '←', '⯇':
		return '<'
	case '▼', '▾', '↓':
		return 'v'
	case '▲', '▴', '↑':
		return '^'
	case '●', '•', '◆':
		return '*'
	case '✓', '✔':
		return '+'
	case '✗', '✘':
		return 'x'
	case '—', '–':
		return '-'
	default:
		return r
	}
}

// GlyphQuad represents a single character's rendering quad.
// Used for passing glyph data to AddGlyphQuads.
type GlyphQuad struct {
	X0, Y0 float32 // Screen coordinates (top-left)
	X1, Y1 float32 // Screen coordinates (bottom-right)
	U0, V0 float32 // Texture coordinates (top-left)
	U1, V1 float32 // Texture coordinates (bottom-right)
}

// AddGlyphQuads draws a slice of glyph quads with the specified color.
// This is used for rendering text from proportional fonts.
func (dl *DrawList) AddGlyphQuads(quads []GlyphQuad, color uint32) {
	if color&0xFF000000 == 0 || len(quads) == 0 {
		return
	}

	for _, q := range quads {
		vtxIdx := dl.addVertices(
			Vertex{Pos: [2]float32{q.X0, q.Y0}, TexCoord: [2]float32{q.U0, q.V0}, Color: color},
			Vertex{Pos: [2]float32{q.X1, q.Y0}, TexCoord: [2]float32{q.U1, q.V0}, Color: color},
			Vertex{Pos: [2]float32{q.X1, q.Y1}, TexCoord: [2]float32{q.U1, q.V1}, Color: color},
			Vertex{Pos: [2]float32{q.X0, q.Y1}, TexCoord: [2]float32{q.U0, q.V1}, Color: color},
		)
		dl.addIndices(vtxIdx, vtxIdx+1, vtxIdx+2, vtxIdx, vtxIdx+2, vtxIdx+3)
	}
}

// InsertRect inserts a rectangle at the beginning of the draw list.
// Useful for drawing backgrounds after content (to get correct size).
func (dl *DrawList) InsertRect(x, y, w, h float32, color uint32) {
	if color&0xFF000000 == 0 {
		return
	}

	// Create the vertices
	verts := []Vertex{
		{Pos: [2]float32{x, y}, Color: color},
		{Pos: [2]float32{x + w, y}, Color: color},
		{Pos: [2]float32{x + w, y + h}, Color: color},
		{Pos: [2]float32{x, y + h}, Color: color},
	}

	// Insert at beginning
	dl.VtxBuffer = append(verts, dl.VtxBuffer...)

	// Indices for the new rect (these are absolute since VertexOffset=0)
	newIndices := []uint16{0, 1, 2, 0, 2, 3}

	// Insert new indices at beginning
	dl.IdxBuffer = append(newIndices, dl.IdxBuffer...)

	// Update command offsets - indices are relative to VertexOffset,
	// so we only need to shift VertexOffset and IndexOffset.
	// DO NOT modify the index values themselves - they're relative indices
	// that work with DrawElementsBaseVertex.
	for i := range dl.CmdBuffer {
		dl.CmdBuffer[i].VertexOffset += 4
		dl.CmdBuffer[i].IndexOffset += 6
	}

	// Also update the tracking offsets so that subsequent SetTexture calls
	// correctly calculate ElemCount for any pending command.
	dl.cmdOffset += 4
	dl.idxCmdOffset += 6

	// Insert a new command at the beginning for the background
	bgCmd := DrawCmd{
		ElemCount:    6,
		ClipRect:     dl.currentClip,
		TextureID:    0,
		VertexOffset: 0,
		IndexOffset:  0,
	}
	dl.CmdBuffer = append([]DrawCmd{bgCmd}, dl.CmdBuffer...)
}

// Finalize prepares the DrawList for rendering.
// Must be called after all primitives are added.
func (dl *DrawList) Finalize() {
	// Finalize the last command
	if len(dl.CmdBuffer) > 0 {
		lastCmd := &dl.CmdBuffer[len(dl.CmdBuffer)-1]
		lastCmd.ElemCount = uint32(len(dl.IdxBuffer)) - dl.idxCmdOffset
	}

	// Remove empty commands
	filtered := dl.CmdBuffer[:0]
	for _, cmd := range dl.CmdBuffer {
		if cmd.ElemCount > 0 {
			filtered = append(filtered, cmd)
		}
	}
	dl.CmdBuffer = filtered
}

// sqrtf is a simple square root approximation.
// For UI purposes, precision isn't critical.
func sqrtf(x float32) float32 {
	if x <= 0 {
		return 0
	}
	// Newton-Raphson iteration (2 iterations is enough for UI)
	guess := x / 2
	guess = (guess + x/guess) / 2
	guess = (guess + x/guess) / 2
	return guess
}
