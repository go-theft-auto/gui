package gui

// FontProvider is the interface for font management in the GUI system.
// It abstracts font loading, caching, and selection, allowing different
// implementations to be injected (e.g., GTA fonts, system fonts, mock fonts for testing).
//
// The GUI package does not depend on any concrete font implementation.
// Instead, applications inject a FontProvider that satisfies this interface.
//
// Example usage:
//
//	// Application code creates a concrete font manager
//	fontMgr := font.NewManager()
//	fontMgr.LoadGTAFonts(gameDir)
//
//	// Inject into GUI context
//	ctx := gui.NewContext()
//	ctx.SetFontProvider(fontMgr)
type FontProvider interface {
	// ActiveFont returns the currently active font for rendering.
	// Returns nil if no font is loaded or active.
	ActiveFont() Font

	// SetActiveFont sets the active font by name.
	// Returns an error if the font is not found.
	SetActiveFont(name string) error
}

// Font is the interface for a single font that can render text.
// It provides methods for measuring text and generating rendering quads.
//
// Implementations should be GPU-optimized, using pre-generated texture atlases
// rather than CPU rasterization at render time.
type Font interface {
	// TextureID returns the OpenGL texture ID for the font atlas.
	// This texture should be bound before rendering glyph quads.
	TextureID() uint32

	// HasGlyph returns true if the font has a glyph for the given rune.
	// This is useful for checking character support before rendering,
	// or for implementing fallback font logic.
	HasGlyph(r rune) bool

	// MeasureText returns the pixel dimensions of the given text at the specified scale.
	// This is used for layout calculations before rendering.
	MeasureText(text string, scale float32) FontVec2

	// GetGlyphQuads generates quads for rendering the given text.
	// Each quad contains screen coordinates and texture coordinates.
	// The returned slice should be used immediately and not stored.
	GetGlyphQuads(text string, x, y, scale float32) []FontGlyphQuad

	// LineHeight returns the line height at the specified scale.
	LineHeight(scale float32) float32
}

// FontVec2 represents a 2D vector returned by font measurement.
// This mirrors the font package's Vec2 to avoid import dependencies.
type FontVec2 struct {
	X, Y float32
}

// FontGlyphQuad represents a single character's rendering quad from a font.
// This mirrors the font package's GlyphQuad to avoid import dependencies.
type FontGlyphQuad struct {
	// Screen coordinates (top-left and bottom-right)
	X0, Y0 float32
	X1, Y1 float32

	// Texture coordinates (top-left and bottom-right)
	U0, V0 float32
	U1, V1 float32
}
