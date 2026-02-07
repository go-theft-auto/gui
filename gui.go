package gui

// Renderer is the interface for rendering GUI draw data.
type Renderer interface {
	Render(dl *DrawList) error
	FontTextureID() uint32
	Resize(width, height int)
}

// GUI manages the immediate mode UI system.
type GUI struct {
	renderer     Renderer
	stateStore   StateStore
	style        Style
	ctx          *Context
	fontProvider FontProvider
}

// GUIOption configures a GUI instance.
type GUIOption func(*GUI)

// WithStyle sets the GUI style.
func WithStyle(style Style) GUIOption {
	return func(g *GUI) { g.style = style }
}

// WithStateStore sets a custom state store.
func WithStateStore(store StateStore) GUIOption {
	return func(g *GUI) { g.stateStore = store }
}

// New creates a new GUI instance.
func New(renderer Renderer, opts ...GUIOption) *GUI {
	g := &GUI{
		renderer:   renderer,
		stateStore: make(MapStateStore),
		style:      DefaultStyle(),
		ctx:        NewContext(),
	}

	for _, opt := range opts {
		opt(g)
	}

	return g
}

// Begin starts a new frame and returns the GUI context.
// Call this at the start of each frame before drawing any UI.
func (g *GUI) Begin(input *InputState, displaySize Vec2, deltaTime float32) *Context {
	ctx := g.ctx

	// Acquire draw lists from the pool
	ctx.DrawList = AcquireDrawList()
	ctx.ForegroundDrawList = AcquireDrawList() // For popups, dropdowns (drawn on top)

	// Set frame state
	ctx.Input = input
	ctx.stateStore = g.stateStore
	ctx.DisplaySize = displaySize
	ctx.DeltaTime = deltaTime
	ctx.SetStyle(g.style)
	ctx.FontTextureID = g.renderer.FontTextureID()

	// Set font provider if available
	if g.fontProvider != nil {
		ctx.SetFontProvider(g.fontProvider)
	}

	// Reset per-frame state
	ctx.Reset(displaySize, deltaTime)

	return ctx
}

// End finishes the frame and renders the UI.
// Call this after all UI drawing is complete.
func (g *GUI) End() error {
	if g.ctx.DrawList == nil {
		return nil
	}

	// Render main draw list
	err := g.renderer.Render(g.ctx.DrawList)
	if err != nil {
		return err
	}

	// Render foreground draw list (popups, dropdowns) on top
	if g.ctx.ForegroundDrawList != nil && len(g.ctx.ForegroundDrawList.CmdBuffer) > 0 {
		err = g.renderer.Render(g.ctx.ForegroundDrawList)
	}

	// Release draw lists back to pool
	ReleaseDrawList(g.ctx.DrawList)
	g.ctx.DrawList = nil
	if g.ctx.ForegroundDrawList != nil {
		ReleaseDrawList(g.ctx.ForegroundDrawList)
		g.ctx.ForegroundDrawList = nil
	}

	return err
}

// Context returns the current GUI context.
// Only valid between Begin() and End() calls.
func (g *GUI) Context() *Context {
	return g.ctx
}

// Style returns the current GUI style.
func (g *GUI) Style() Style {
	return g.style
}

// SetStyle sets the GUI style.
func (g *GUI) SetStyle(style Style) {
	g.style = style
}

// Resize notifies the GUI of a display size change.
func (g *GUI) Resize(width, height int) {
	g.renderer.Resize(width, height)
}

// SetFontProvider sets the font provider for advanced font support.
// The provider will be passed to each frame's Context.
func (g *GUI) SetFontProvider(fp FontProvider) {
	g.fontProvider = fp
}

// FontProvider returns the current font provider, or nil if not set.
func (g *GUI) FontProvider() FontProvider {
	return g.fontProvider
}

// PrepareInputHandling prepares the GUI for input handling by swapping the focus registry buffers.
// CRITICAL: Call this at the START of BeginFrame(), BEFORE any panel HandleInput() is called.
//
// This is necessary because:
//   - HandleInput() runs in BeginFrame() and needs the previous frame's widget registrations
//   - Widgets register themselves during Draw() which happens in EndFrame()
//   - The focus registry uses double-buffering: prevItems (for navigation) and items (being built)
//   - This method swaps the buffers so prevItems contains the last frame's widgets
func (g *GUI) PrepareInputHandling() {
	if g.ctx == nil {
		return
	}
	// Increment frame count here (at start of frame) so the same count is used
	// throughout the frame in both PrepareInputHandling and Context.Reset
	g.ctx.FrameCount++

	if g.ctx.focusRegistry != nil {
		g.ctx.focusRegistry.ResetForFrame(g.ctx.FrameCount)
	}
}
