// Package gui provides an immediate-mode GUI library inspired by Dear ImGui.
// It uses a dedicated Context type (not context.Context) for better performance
// and type safety.
package gui

// Vec2 represents a 2D vector for positions and sizes.
type Vec2 struct {
	X, Y float32
}

// Add returns the sum of two vectors.
func (v Vec2) Add(other Vec2) Vec2 {
	return Vec2{X: v.X + other.X, Y: v.Y + other.Y}
}

// Sub returns the difference of two vectors.
func (v Vec2) Sub(other Vec2) Vec2 {
	return Vec2{X: v.X - other.X, Y: v.Y - other.Y}
}

// Mul returns the vector scaled by a scalar.
func (v Vec2) Mul(s float32) Vec2 {
	return Vec2{X: v.X * s, Y: v.Y * s}
}

// Rect represents a rectangle with position and size.
type Rect struct {
	X, Y float32 // Top-left position
	W, H float32 // Width and height
}

// Contains returns true if the point is inside the rectangle.
func (r Rect) Contains(p Vec2) bool {
	return p.X >= r.X && p.X < r.X+r.W && p.Y >= r.Y && p.Y < r.Y+r.H
}

// Intersects returns true if two rectangles overlap.
func (r Rect) Intersects(other Rect) bool {
	return r.X < other.X+other.W && r.X+r.W > other.X &&
		r.Y < other.Y+other.H && r.Y+r.H > other.Y
}

// Vertex represents a vertex for UI rendering.
// Memory layout matches OpenGL vertex attribute expectations.
type Vertex struct {
	Pos      [2]float32 // Position (x, y)
	TexCoord [2]float32 // Texture coordinates (u, v)
	Color    uint32     // RGBA packed color
}

// DrawCmd represents a single draw command.
// Commands are batched by texture to minimize state changes.
type DrawCmd struct {
	ElemCount    uint32     // Number of indices to draw
	ClipRect     [4]float32 // Clip rectangle (x1, y1, x2, y2)
	TextureID    uint32     // OpenGL texture ID (0 = no texture)
	VertexOffset uint32     // Offset into vertex buffer
	IndexOffset  uint32     // Offset into index buffer
}

// Color constants (RGBA packed as 0xAABBGGRR for OpenGL compatibility)
const (
	ColorWhite       uint32 = 0xFFFFFFFF
	ColorBlack       uint32 = 0xFF000000
	ColorRed         uint32 = 0xFF0000FF
	ColorGreen       uint32 = 0xFF00FF00
	ColorBlue        uint32 = 0xFFFF0000
	ColorYellow      uint32 = 0xFF00FFFF
	ColorCyan        uint32 = 0xFFFFFF00
	ColorMagenta     uint32 = 0xFFFF00FF
	ColorGray        uint32 = 0xFF808080
	ColorDarkGray    uint32 = 0xFF404040
	ColorLightGray   uint32 = 0xFFC0C0C0
	ColorTransparent uint32 = 0x00000000
)

// RGBA creates a packed color from individual components (0-255).
func RGBA(r, g, b, a uint8) uint32 {
	return uint32(a)<<24 | uint32(b)<<16 | uint32(g)<<8 | uint32(r)
}

// RGBAf creates a packed color from float components (0.0-1.0).
func RGBAf(r, g, b, a float32) uint32 {
	return RGBA(
		uint8(clampf(r, 0, 1)*255),
		uint8(clampf(g, 0, 1)*255),
		uint8(clampf(b, 0, 1)*255),
		uint8(clampf(a, 0, 1)*255),
	)
}

// UnpackRGBA extracts RGBA components from a packed color.
func UnpackRGBA(c uint32) (r, g, b, a uint8) {
	return uint8(c), uint8(c >> 8), uint8(c >> 16), uint8(c >> 24)
}

// clampf clamps a float32 value to a range.
func clampf(v, minVal, maxVal float32) float32 {
	if v < minVal {
		return minVal
	}
	if v > maxVal {
		return maxVal
	}
	return v
}

// maxf returns the maximum of two float32 values.
func maxf(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

// minf returns the minimum of two float32 values.
func minf(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}
