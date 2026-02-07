package gui

// PanelBounds represents the bounds of a panel for snapping calculations.
type PanelBounds struct {
	X, Y, W, H float32
	Name       string
}

// SnapGuide represents a visual snap guide line.
type SnapGuide struct {
	X1, Y1, X2, Y2 float32
	Horizontal     bool
}

// SnapManager handles panel snapping to screen edges and other panels.
type SnapManager struct {
	panels       []PanelBounds
	screenSize   Vec2
	config       SnapConfig
	activeGuides []SnapGuide
}

// NewSnapManager creates a new snap manager.
func NewSnapManager(screenSize Vec2, config SnapConfig) *SnapManager {
	return &SnapManager{
		panels:       make([]PanelBounds, 0, 8),
		screenSize:   screenSize,
		config:       config,
		activeGuides: make([]SnapGuide, 0, 4),
	}
}

// SetScreenSize updates the screen size for edge snapping.
func (sm *SnapManager) SetScreenSize(size Vec2) {
	sm.screenSize = size
}

// SetConfig updates the snap configuration.
func (sm *SnapManager) SetConfig(config SnapConfig) {
	sm.config = config
}

// Clear removes all registered panels.
func (sm *SnapManager) Clear() {
	sm.panels = sm.panels[:0]
	sm.activeGuides = sm.activeGuides[:0]
}

// RegisterPanel adds a panel to the snap manager.
func (sm *SnapManager) RegisterPanel(name string, bounds PanelBounds) {
	bounds.Name = name
	sm.panels = append(sm.panels, bounds)
}

// UpdatePanel updates the bounds of an existing panel.
func (sm *SnapManager) UpdatePanel(name string, bounds PanelBounds) {
	for i := range sm.panels {
		if sm.panels[i].Name == name {
			bounds.Name = name
			sm.panels[i] = bounds
			return
		}
	}
	// Not found, add it
	sm.RegisterPanel(name, bounds)
}

// RemovePanel removes a panel from the snap manager.
func (sm *SnapManager) RemovePanel(name string) {
	for i := range sm.panels {
		if sm.panels[i].Name == name {
			sm.panels = append(sm.panels[:i], sm.panels[i+1:]...)
			return
		}
	}
}

// CalculateSnap calculates the snapped position for a panel being dragged.
// Returns the adjusted position and which edges snapped.
// excluding is the name of the panel being dragged (to avoid self-snapping).
func (sm *SnapManager) CalculateSnap(bounds PanelBounds, excluding string) (Vec2, []SnapGuide) {
	if !sm.config.Enabled {
		return Vec2{X: bounds.X, Y: bounds.Y}, nil
	}

	sm.activeGuides = sm.activeGuides[:0]
	newX := bounds.X
	newY := bounds.Y
	snappedX := false
	snappedY := false

	edgeMargin := sm.config.EdgeMargin
	panelMargin := sm.config.PanelMargin

	// Snap to screen edges
	if edgeMargin > 0 {
		// Left edge
		if absf32(bounds.X) < edgeMargin && !snappedX {
			newX = 0
			snappedX = true
			sm.activeGuides = append(sm.activeGuides, SnapGuide{
				X1: 0, Y1: 0, X2: 0, Y2: sm.screenSize.Y, Horizontal: false,
			})
		}
		// Top edge
		if absf32(bounds.Y) < edgeMargin && !snappedY {
			newY = 0
			snappedY = true
			sm.activeGuides = append(sm.activeGuides, SnapGuide{
				X1: 0, Y1: 0, X2: sm.screenSize.X, Y2: 0, Horizontal: true,
			})
		}
		// Right edge
		rightDist := absf32((bounds.X + bounds.W) - sm.screenSize.X)
		if rightDist < edgeMargin && !snappedX {
			newX = sm.screenSize.X - bounds.W
			snappedX = true
			sm.activeGuides = append(sm.activeGuides, SnapGuide{
				X1: sm.screenSize.X, Y1: 0, X2: sm.screenSize.X, Y2: sm.screenSize.Y, Horizontal: false,
			})
		}
		// Bottom edge
		bottomDist := absf32((bounds.Y + bounds.H) - sm.screenSize.Y)
		if bottomDist < edgeMargin && !snappedY {
			newY = sm.screenSize.Y - bounds.H
			snappedY = true
			sm.activeGuides = append(sm.activeGuides, SnapGuide{
				X1: 0, Y1: sm.screenSize.Y, X2: sm.screenSize.X, Y2: sm.screenSize.Y, Horizontal: true,
			})
		}

		// Snap to screen center
		centerX := sm.screenSize.X / 2
		centerY := sm.screenSize.Y / 2
		panelCenterX := bounds.X + bounds.W/2
		panelCenterY := bounds.Y + bounds.H/2

		if absf32(panelCenterX-centerX) < edgeMargin && !snappedX {
			newX = centerX - bounds.W/2
			snappedX = true
			sm.activeGuides = append(sm.activeGuides, SnapGuide{
				X1: centerX, Y1: 0, X2: centerX, Y2: sm.screenSize.Y, Horizontal: false,
			})
		}
		if absf32(panelCenterY-centerY) < edgeMargin && !snappedY {
			newY = centerY - bounds.H/2
			snappedY = true
			sm.activeGuides = append(sm.activeGuides, SnapGuide{
				X1: 0, Y1: centerY, X2: sm.screenSize.X, Y2: centerY, Horizontal: true,
			})
		}
	}

	// Snap to other panels
	if panelMargin > 0 {
		for _, other := range sm.panels {
			if other.Name == excluding {
				continue
			}

			// Snap to left edge of other panel
			if !snappedX {
				// Our right edge to their left edge
				dist := absf32((bounds.X + bounds.W) - other.X)
				if dist < panelMargin {
					newX = other.X - bounds.W
					snappedX = true
					sm.activeGuides = append(sm.activeGuides, SnapGuide{
						X1: other.X, Y1: minf(bounds.Y, other.Y),
						X2: other.X, Y2: maxf(bounds.Y+bounds.H, other.Y+other.H),
						Horizontal: false,
					})
				}
				// Our left edge to their left edge (alignment)
				dist = absf32(bounds.X - other.X)
				if dist < panelMargin {
					newX = other.X
					snappedX = true
					sm.activeGuides = append(sm.activeGuides, SnapGuide{
						X1: other.X, Y1: minf(bounds.Y, other.Y),
						X2: other.X, Y2: maxf(bounds.Y+bounds.H, other.Y+other.H),
						Horizontal: false,
					})
				}
			}

			// Snap to right edge of other panel
			if !snappedX {
				// Our left edge to their right edge
				dist := absf32(bounds.X - (other.X + other.W))
				if dist < panelMargin {
					newX = other.X + other.W
					snappedX = true
					sm.activeGuides = append(sm.activeGuides, SnapGuide{
						X1: other.X + other.W, Y1: minf(bounds.Y, other.Y),
						X2: other.X + other.W, Y2: maxf(bounds.Y+bounds.H, other.Y+other.H),
						Horizontal: false,
					})
				}
				// Our right edge to their right edge (alignment)
				dist = absf32((bounds.X + bounds.W) - (other.X + other.W))
				if dist < panelMargin {
					newX = other.X + other.W - bounds.W
					snappedX = true
					sm.activeGuides = append(sm.activeGuides, SnapGuide{
						X1: other.X + other.W, Y1: minf(bounds.Y, other.Y),
						X2: other.X + other.W, Y2: maxf(bounds.Y+bounds.H, other.Y+other.H),
						Horizontal: false,
					})
				}
			}

			// Snap to top edge of other panel
			if !snappedY {
				// Our bottom edge to their top edge
				dist := absf32((bounds.Y + bounds.H) - other.Y)
				if dist < panelMargin {
					newY = other.Y - bounds.H
					snappedY = true
					sm.activeGuides = append(sm.activeGuides, SnapGuide{
						X1: minf(bounds.X, other.X), Y1: other.Y,
						X2: maxf(bounds.X+bounds.W, other.X+other.W), Y2: other.Y,
						Horizontal: true,
					})
				}
				// Our top edge to their top edge (alignment)
				dist = absf32(bounds.Y - other.Y)
				if dist < panelMargin {
					newY = other.Y
					snappedY = true
					sm.activeGuides = append(sm.activeGuides, SnapGuide{
						X1: minf(bounds.X, other.X), Y1: other.Y,
						X2: maxf(bounds.X+bounds.W, other.X+other.W), Y2: other.Y,
						Horizontal: true,
					})
				}
			}

			// Snap to bottom edge of other panel
			if !snappedY {
				// Our top edge to their bottom edge
				dist := absf32(bounds.Y - (other.Y + other.H))
				if dist < panelMargin {
					newY = other.Y + other.H
					snappedY = true
					sm.activeGuides = append(sm.activeGuides, SnapGuide{
						X1: minf(bounds.X, other.X), Y1: other.Y + other.H,
						X2: maxf(bounds.X+bounds.W, other.X+other.W), Y2: other.Y + other.H,
						Horizontal: true,
					})
				}
				// Our bottom edge to their bottom edge (alignment)
				dist = absf32((bounds.Y + bounds.H) - (other.Y + other.H))
				if dist < panelMargin {
					newY = other.Y + other.H - bounds.H
					snappedY = true
					sm.activeGuides = append(sm.activeGuides, SnapGuide{
						X1: minf(bounds.X, other.X), Y1: other.Y + other.H,
						X2: maxf(bounds.X+bounds.W, other.X+other.W), Y2: other.Y + other.H,
						Horizontal: true,
					})
				}
			}
		}
	}

	return Vec2{X: newX, Y: newY}, sm.activeGuides
}

// DrawGuides draws the active snap guide lines.
// Call this during a drag operation to show visual feedback.
func (sm *SnapManager) DrawGuides(dl *DrawList, style Style) {
	if len(sm.activeGuides) == 0 {
		return
	}

	guideColor := RGBA(0, 180, 255, 150) // Cyan guide lines

	for _, guide := range sm.activeGuides {
		dl.AddLine(guide.X1, guide.Y1, guide.X2, guide.Y2, guideColor, 1)
	}
}

// ActiveGuides returns the currently active snap guides.
func (sm *SnapManager) ActiveGuides() []SnapGuide {
	return sm.activeGuides
}

// ClearGuides clears the active snap guides.
func (sm *SnapManager) ClearGuides() {
	sm.activeGuides = sm.activeGuides[:0]
}
