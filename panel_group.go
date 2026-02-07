package gui

// PanelGroup manages multiple panels in a single tabbed container.
// This allows users to group related panels together, saving screen space
// and enabling organization similar to ImGui's docking system.
//
// Usage:
//
//	group := gui.NewPanelGroup("My Group")
//	group.AddPanel("Tab1", panel1)
//	group.AddPanel("Tab2", panel2)
//
//	// In draw loop:
//	group.Draw(ctx)
//
// Tab switching:
// - Click on tab to switch
// - Ctrl+1-9 to switch to specific tab (when focused)
// - Ctrl+PgUp/PgDown to cycle tabs
type PanelGroup struct {
	// ID is a unique identifier for this group.
	ID string

	// Panels contains the grouped panels in tab order.
	panels []groupedPanel

	// ActiveTab is the index of the currently visible tab.
	ActiveTab int

	// Position and size for the group container.
	DraggablePanel

	// open tracks whether the group is open (visible).
	open bool

	// onClose is called when the group is closed.
	onClose func()
}

// groupedPanel holds a panel with its tab name.
type groupedPanel struct {
	Name  string
	Panel Panel
}

// NewPanelGroup creates a new panel group with the given ID.
func NewPanelGroup(id string) *PanelGroup {
	return &PanelGroup{
		ID:             id,
		panels:         make([]groupedPanel, 0, 4),
		ActiveTab:      0,
		DraggablePanel: *NewDraggablePanel(0, 0),
		open:           true,
	}
}

// AddPanel adds a panel to the group with the given tab name.
func (pg *PanelGroup) AddPanel(name string, panel Panel) {
	pg.panels = append(pg.panels, groupedPanel{
		Name:  name,
		Panel: panel,
	})
}

// RemovePanel removes a panel from the group by name.
// Returns true if the panel was found and removed.
func (pg *PanelGroup) RemovePanel(name string) bool {
	for i, p := range pg.panels {
		if p.Name == name {
			pg.panels = append(pg.panels[:i], pg.panels[i+1:]...)
			// Adjust active tab if needed
			if pg.ActiveTab >= len(pg.panels) && pg.ActiveTab > 0 {
				pg.ActiveTab = len(pg.panels) - 1
			}
			return true
		}
	}
	return false
}

// GetPanel returns a panel by tab name, or nil if not found.
func (pg *PanelGroup) GetPanel(name string) Panel {
	for _, p := range pg.panels {
		if p.Name == name {
			return p.Panel
		}
	}
	return nil
}

// PanelCount returns the number of panels in the group.
func (pg *PanelGroup) PanelCount() int {
	return len(pg.panels)
}

// ActivePanel returns the currently active panel, or nil if empty.
func (pg *PanelGroup) ActivePanel() Panel {
	if pg.ActiveTab < 0 || pg.ActiveTab >= len(pg.panels) {
		return nil
	}
	return pg.panels[pg.ActiveTab].Panel
}

// SetActiveTab sets the active tab by index.
func (pg *PanelGroup) SetActiveTab(index int) {
	if index >= 0 && index < len(pg.panels) {
		pg.ActiveTab = index
	}
}

// SetActiveTabByName sets the active tab by name.
// Returns true if the tab was found and activated.
func (pg *PanelGroup) SetActiveTabByName(name string) bool {
	for i, p := range pg.panels {
		if p.Name == name {
			pg.ActiveTab = i
			return true
		}
	}
	return false
}

// NextTab cycles to the next tab (wraps around).
func (pg *PanelGroup) NextTab() {
	if len(pg.panels) == 0 {
		return
	}
	pg.ActiveTab = (pg.ActiveTab + 1) % len(pg.panels)
}

// PrevTab cycles to the previous tab (wraps around).
func (pg *PanelGroup) PrevTab() {
	if len(pg.panels) == 0 {
		return
	}
	pg.ActiveTab--
	if pg.ActiveTab < 0 {
		pg.ActiveTab = len(pg.panels) - 1
	}
}

// Panel interface implementation

// Open opens the panel group.
func (pg *PanelGroup) Open() {
	pg.open = true
}

// Close closes the panel group.
func (pg *PanelGroup) Close() {
	pg.open = false
	if pg.onClose != nil {
		pg.onClose()
	}
}

// Toggle toggles the group open/closed state.
func (pg *PanelGroup) Toggle() bool {
	pg.open = !pg.open
	if !pg.open && pg.onClose != nil {
		pg.onClose()
	}
	return pg.open
}

// IsOpen returns true if the group is open.
func (pg *PanelGroup) IsOpen() bool {
	return pg.open
}

// CanOpen returns true (groups can always open).
func (pg *PanelGroup) CanOpen() bool {
	return true
}

// SetOnClose sets the callback for when the group closes.
func (pg *PanelGroup) SetOnClose(fn func()) {
	pg.onClose = fn
}

// Draw renders the panel group with tab bar.
func (pg *PanelGroup) Draw(ctx *Context) {
	if !pg.open || len(pg.panels) == 0 {
		return
	}

	// Handle drag
	pg.HandleDrag(ctx)

	// Set cursor position
	ctx.SetCursorPos(pg.Position.X, pg.Position.Y)

	// Draw container panel
	ctx.Panel(pg.ID, Padding(ctx.Style().PanelPadding))(func() {
		// Draw tab bar
		ctx.HStack(Gap(SpaceXS))(func() {
			for i, p := range pg.panels {
				isSelected := i == pg.ActiveTab

				// Calculate tab style
				tabStyle := TabStyle{
					Selected: isSelected,
				}

				if ctx.tabButton(p.Name, tabStyle, WithID(pg.ID+"_tab_"+p.Name)) {
					pg.ActiveTab = i
				}
			}
		})

		// Separator between tabs and content
		ctx.Separator()

		// Draw active panel content
		if pg.ActiveTab >= 0 && pg.ActiveTab < len(pg.panels) {
			activePanel := pg.panels[pg.ActiveTab].Panel
			// The panel draws its own content (we skip its Open check since we manage visibility)
			if activePanel != nil {
				// Call the panel's Draw method but it should handle its own layout
				// For grouped panels, we typically want them to draw as if they're open
				activePanel.Draw(ctx)
			}
		}
	})

	// Update our size based on what was drawn
	// (In a real implementation, we'd measure the content)
}

// HandleInput processes input for the panel group.
func (pg *PanelGroup) HandleInput(input *InputState) bool {
	if !pg.open || input == nil {
		return false
	}

	// Tab switching with Ctrl+PgUp/PgDown
	if input.ModCtrl {
		if input.KeyPressed(KeyPageUp) {
			pg.PrevTab()
			return true
		}
		if input.KeyPressed(KeyPageDown) {
			pg.NextTab()
			return true
		}
	}

	// Escape closes the group
	if input.KeyPressed(KeyEscape) {
		pg.Close()
		return true
	}

	// Route input to active panel
	if activePanel := pg.ActivePanel(); activePanel != nil {
		if activePanel.HandleInput(input) {
			return true
		}
	}

	return false
}

// TabStyle configures the appearance of a tab button.
type TabStyle struct {
	Selected bool
	Closable bool
}

// tabButton draws a single tab button and returns true if clicked.
func (ctx *Context) tabButton(label string, style TabStyle, opts ...Option) bool {
	// Apply options
	o := applyOptions(opts)

	var id ID
	if optID := GetOpt(o, OptID); optID != "" {
		id = ctx.GetID(optID)
	} else {
		id = ctx.GetID(label)
	}

	pos := ctx.ItemPos()

	// Measure text
	textSize := ctx.MeasureText(label)
	paddingX := ctx.style.ButtonPadding
	paddingY := SpaceXS

	w := textSize.X + paddingX*2
	h := textSize.Y + paddingY*2

	rect := Rect{X: pos.X, Y: pos.Y, W: w, H: h}

	// Determine colors based on state
	var bgColor, textColor uint32
	isHovered := ctx.isHovered(id, rect)
	isClicked := ctx.isClicked(id, rect)

	if style.Selected {
		bgColor = ctx.style.SelectedBgColor
		textColor = ctx.style.SelectedTextColor
	} else if isHovered {
		bgColor = ctx.style.HoveredBgColor
		textColor = ctx.style.TextColor
	} else {
		bgColor = ctx.style.ButtonColor
		textColor = ctx.style.TextColor
	}

	// Draw background
	ctx.DrawList.AddRect(pos.X, pos.Y, w, h, bgColor)

	// Draw text centered
	textX := pos.X + (w-textSize.X)/2
	textY := pos.Y + (h-textSize.Y)/2
	ctx.addText(textX, textY, label, textColor)

	// Draw bottom border for selected tab (visual indicator)
	if style.Selected {
		borderY := pos.Y + h - 2
		ctx.DrawList.AddRect(pos.X, borderY, w, 2, ctx.style.FocusColor)
	}

	ctx.advanceCursor(Vec2{X: w, Y: h})

	// Track hover for cursor change
	if isHovered {
		ctx.WantCaptureMouse = true
	}

	return isClicked
}
