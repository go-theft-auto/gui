package gui

// MenuDataSource provides items for a modal menu to display.
// The menu doesn't know what the items represent - just how to display them.
type MenuDataSource interface {
	// Count returns the total number of items (after filtering).
	Count() int

	// Label returns the display text for an item at the given index.
	Label(index int) string

	// IsMarked returns true if the item should be marked (e.g., current selection).
	IsMarked(index int) bool

	// Filter applies a search query to the data source.
	// Empty string means show all items.
	Filter(query string)
}

// MenuDelegate receives callbacks from menu interactions.
type MenuDelegate interface {
	// OnSelect is called when the selection changes (for preview).
	OnSelect(index int)

	// OnConfirm is called when the user confirms selection (Enter).
	OnConfirm(index int)

	// OnCancel is called when the user cancels (Escape).
	OnCancel()
}

// ModalMenu is a generic searchable list menu.
// It handles display, scrolling, selection, and search without knowing
// anything about the actual data being displayed.
type ModalMenu struct {
	title      string
	hotkey     string // Keyboard shortcut to display in header
	dataSource MenuDataSource
	delegate   MenuDelegate

	// State
	open         bool
	selectedIdx  int
	scrollOffset int
	searchText   string

	// Configuration
	maxVisible int
	width      float32

	// Draggable support
	drag DraggablePanel
}

// NewModalMenu creates a new modal menu.
func NewModalMenu(title string, width float32, maxVisible int) *ModalMenu {
	m := &ModalMenu{
		title:      title,
		width:      width,
		maxVisible: maxVisible,
	}
	m.drag = *NewDraggablePanel(0, 0)
	return m
}

// SetDataSource sets the data source for the menu.
func (m *ModalMenu) SetDataSource(ds MenuDataSource) {
	m.dataSource = ds
}

// SetDelegate sets the delegate for menu callbacks.
func (m *ModalMenu) SetDelegate(del MenuDelegate) {
	m.delegate = del
}

// SetPosition sets the top-left position of the menu.
func (m *ModalMenu) SetPosition(x, y float32) {
	m.drag.SetPosition(x, y)
}

// Draggable returns the draggable panel for configuration.
func (m *ModalMenu) Draggable() *DraggablePanel {
	return &m.drag
}

// SetHotkey sets the keyboard shortcut to display in the header.
func (m *ModalMenu) SetHotkey(key string) {
	m.hotkey = key
}

// Open opens the menu and resets state.
func (m *ModalMenu) Open() {
	m.open = true
	m.selectedIdx = 0
	m.scrollOffset = 0
	m.searchText = ""
	if m.dataSource != nil {
		m.dataSource.Filter("")
	}
}

// Close closes the menu.
func (m *ModalMenu) Close() {
	m.open = false
}

// Toggle toggles the menu open/closed state.
func (m *ModalMenu) Toggle() {
	if m.open {
		m.Close()
	} else {
		m.Open()
	}
}

// IsOpen returns true if the menu is open.
func (m *ModalMenu) IsOpen() bool {
	return m.open
}

// SelectedIndex returns the currently selected index.
func (m *ModalMenu) SelectedIndex() int {
	return m.selectedIdx
}

// SearchText returns the current search text.
func (m *ModalMenu) SearchText() string {
	return m.searchText
}

// Draw renders the menu using the provided GUI context.
func (m *ModalMenu) Draw(ctx *Context) {
	if !m.open || m.dataSource == nil {
		return
	}

	// Handle drag input
	m.drag.HandleDrag(ctx)

	// Use draggable position
	panelStart := m.drag.Position
	ctx.SetCursorPos(panelStart.X, panelStart.Y)

	listLen := m.dataSource.Count()

	// Validate indices
	m.validateIndices(listLen)

	// Build panel options
	panelOpts := []LayoutOption{Width(m.width), Padding(SpaceLG), Gap(SpaceSM)}
	if m.hotkey != "" {
		panelOpts = append(panelOpts, WithHotkey(m.hotkey))
	}

	ctx.Panel(m.title, panelOpts...)(func() {
		// Search hint
		ctx.HintHeader("Type to search...")

		// Search text display
		if m.searchText != "" {
			ctx.Text("Filter: " + m.searchText)
		}

		ctx.Separator()

		// Item count
		ctx.HintStatus("%s", formatCount(listLen))

		// Render visible items
		endIdx := m.scrollOffset + m.maxVisible
		if endIdx > listLen {
			endIdx = listLen
		}

		for i := m.scrollOffset; i < endIdx; i++ {
			label := m.dataSource.Label(i)
			isSelected := i == m.selectedIdx

			// Mark current item with prefix
			if m.dataSource.IsMarked(i) {
				label = "* " + label
			}

			if ctx.Selectable(label, isSelected, WithID(formatItemID(i))) {
				// Clicked - confirm this item
				if m.delegate != nil {
					m.delegate.OnConfirm(i)
				}
			}
		}

		// Scroll indicators
		scroll := ctx.HintScroll(m.scrollOffset, m.maxVisible, listLen)
		scroll.Before(ctx)
		scroll.After(ctx)

		// Footer
		ctx.HintFooterNav()
	})

	// Update panel size for drag bounds
	panelEnd := ctx.GetCursorPos()
	m.drag.Size = Vec2{
		X: m.width,
		Y: panelEnd.Y - panelStart.Y,
	}

	// Draw snap guides during drag
	m.drag.DrawSnapGuides(ctx)
}

// HandleInput processes input for the menu.
// Returns true if input was consumed.
func (m *ModalMenu) HandleInput(input *InputState) bool {
	if !m.open || m.dataSource == nil {
		return false
	}

	listLen := m.dataSource.Count()
	prevSelectedIdx := m.selectedIdx

	// Navigation with key repeat
	if input.KeyRepeated(KeyUp) && m.selectedIdx > 0 {
		m.selectedIdx--
	}

	if input.KeyRepeated(KeyDown) && m.selectedIdx < listLen-1 {
		m.selectedIdx++
	}

	if input.KeyRepeated(KeyPageUp) {
		m.selectedIdx -= 10
		if m.selectedIdx < 0 {
			m.selectedIdx = 0
		}
	}

	if input.KeyRepeated(KeyPageDown) {
		m.selectedIdx += 10
		if m.selectedIdx >= listLen {
			m.selectedIdx = listLen - 1
		}
	}

	if input.KeyPressed(KeyHome) {
		m.selectedIdx = 0
	}

	if input.KeyPressed(KeyEnd) && listLen > 0 {
		m.selectedIdx = listLen - 1
	}

	// Auto-scroll to keep selection visible
	m.ensureSelectionVisible()

	// Notify delegate if selection changed
	if m.selectedIdx != prevSelectedIdx && m.delegate != nil && listLen > 0 {
		m.delegate.OnSelect(m.selectedIdx)
	}

	// Enter to confirm
	if input.KeyPressed(KeyEnter) && listLen > 0 {
		if m.delegate != nil {
			m.delegate.OnConfirm(m.selectedIdx)
		}
	}

	// Escape to cancel
	if input.KeyPressed(KeyEscape) {
		m.Close()
		if m.delegate != nil {
			m.delegate.OnCancel()
		}
	}

	// Mouse wheel scrolling
	if input.MouseWheelY != 0 {
		scrollLines := int(-input.MouseWheelY * 3)
		newIdx := m.selectedIdx + scrollLines
		if newIdx < 0 {
			newIdx = 0
		}
		if newIdx >= listLen {
			newIdx = listLen - 1
		}
		if newIdx != m.selectedIdx {
			m.selectedIdx = newIdx
			if m.delegate != nil {
				m.delegate.OnSelect(m.selectedIdx)
			}
		}
	}

	// Text input for search
	if input.HasInputChars() {
		for _, ch := range input.InputChars {
			if ch >= 32 && ch < 127 { // Printable ASCII
				m.searchText += string(ch)
				m.dataSource.Filter(m.searchText)
				m.selectedIdx = 0
				m.scrollOffset = 0
			}
		}
	}

	// Backspace to delete search text
	if input.KeyRepeated(KeyBackspace) && len(m.searchText) > 0 {
		m.searchText = m.searchText[:len(m.searchText)-1]
		m.dataSource.Filter(m.searchText)
		m.selectedIdx = 0
		m.scrollOffset = 0
	}

	return true // Menu consumed input
}

// validateIndices ensures selectedIdx and scrollOffset are valid.
func (m *ModalMenu) validateIndices(listLen int) {
	// Ensure scroll offset is valid
	if m.scrollOffset > listLen-m.maxVisible {
		m.scrollOffset = listLen - m.maxVisible
	}
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}

	// Ensure selected index is valid
	if m.selectedIdx >= listLen {
		m.selectedIdx = listLen - 1
	}
	if m.selectedIdx < 0 {
		m.selectedIdx = 0
	}
}

// ensureSelectionVisible adjusts scroll offset to keep selection in view.
func (m *ModalMenu) ensureSelectionVisible() {
	if m.selectedIdx < m.scrollOffset {
		m.scrollOffset = m.selectedIdx
	}
	if m.selectedIdx >= m.scrollOffset+m.maxVisible {
		m.scrollOffset = m.selectedIdx - m.maxVisible + 1
	}
}

// Helper functions to avoid fmt import in this package
func formatCount(n int) string {
	// Simple integer to string without fmt
	if n == 0 {
		return "0 items"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s + " items"
}

func formatItemID(i int) string {
	// Simple integer to string for IDs
	if i == 0 {
		return "item_0"
	}
	s := ""
	n := i
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return "item_" + s
}
