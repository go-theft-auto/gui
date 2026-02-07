package gui

// tableStore is the type-safe store for table state.
// Uses the new FrameStore pattern instead of the old GetState/SetState.
var tableStore = NewFrameStore[TableState]()

// TableFlags control table behavior and appearance.
type TableFlags uint32

const (
	TableFlagsNone TableFlags = 0

	// Features
	TableFlagsResizable       TableFlags = 1 << 0 // Enable column resizing
	TableFlagsSortable        TableFlags = 1 << 1 // Enable sorting (shows sort indicators)
	TableFlagsRowSelect       TableFlags = 1 << 2 // Enable row selection
	TableFlagsScrollY         TableFlags = 1 << 3 // Enable vertical scrolling (requires height)
	TableFlagsStickyHeader    TableFlags = 1 << 4 // Keep header visible when scrolling
	TableFlagsAutoSizeColumns TableFlags = 1 << 5 // Auto-size columns to fit content

	// Borders
	TableFlagsBordersInnerH TableFlags = 1 << 8  // Horizontal borders between rows
	TableFlagsBordersInnerV TableFlags = 1 << 9  // Vertical borders between columns
	TableFlagsBordersOuterH TableFlags = 1 << 10 // Horizontal border on top/bottom
	TableFlagsBordersOuterV TableFlags = 1 << 11 // Vertical border on left/right

	// Convenience
	TableFlagsBordersInner TableFlags = TableFlagsBordersInnerH | TableFlagsBordersInnerV
	TableFlagsBordersOuter TableFlags = TableFlagsBordersOuterH | TableFlagsBordersOuterV
	TableFlagsBorders      TableFlags = TableFlagsBordersInner | TableFlagsBordersOuter

	// Row appearance
	TableFlagsRowBg          TableFlags = 1 << 16 // Alternate row background colors
	TableFlagsHighlightHover TableFlags = 1 << 17 // Highlight hovered row
)

// TableColumnFlags control individual column behavior.
type TableColumnFlags uint32

const (
	TableColumnFlagsNone TableColumnFlags = 0

	// Sizing
	TableColumnFlagsWidthFixed   TableColumnFlags = 1 << 0 // Fixed width column
	TableColumnFlagsWidthStretch TableColumnFlags = 1 << 1 // Stretch to fill available space
	TableColumnFlagsWidthAuto    TableColumnFlags = 1 << 2 // Auto-size to content (default)

	// Behavior
	TableColumnFlagsNoResize TableColumnFlags = 1 << 8 // Disable manual resizing
	TableColumnFlagsNoSort   TableColumnFlags = 1 << 9 // Disable sorting for this column
)

// TableColumn defines a table column.
type TableColumn struct {
	Label     string
	Flags     TableColumnFlags
	InitWidth float32 // Initial/fixed width (0 = auto)
	MinWidth  float32 // Minimum width when resizing
	MaxWidth  float32 // Maximum width when resizing (0 = unlimited)

	// Runtime state (managed by table)
	width float32 // Current computed width
}

// TableState persists table state between frames.
type TableState struct {
	ColumnWidths     []float32 // User-adjusted column widths
	MaxContentWidths []float32 // Max content width per column (for auto-sizing)
	SortColumn       int       // Currently sorted column (-1 = none)
	SortAscending    bool      // Sort direction
	SelectedRow      int       // Selected row index (-1 = none)
	ScrollOffset     float32   // Vertical scroll position
}

// TableOptions configures table behavior.
type TableOptions struct {
	MaxVisibleRows int // Maximum visible rows before scrolling (0 = unlimited)
}

// Table manages table drawing state for the current frame.
type Table struct {
	id      ID
	ctx     *Context
	flags   TableFlags
	columns []TableColumn
	options TableOptions

	// Layout
	startX, startY float32 // Table origin
	width, height  float32 // Table dimensions
	rowHeight      float32 // Height of each row

	// Current state
	currentRow    int
	currentColumn int
	rowStartY     float32

	// Persistent state
	state *TableState

	// Content width tracking (for auto-sizing)
	frameMaxWidths []float32

	// Virtualization support
	clipper    *ListClipper // nil if virtualization not enabled
	totalRows  int          // Total row count for virtualization
	visibleTop float32      // Top of visible area for clipping
}

// TableMaxVisibleRows sets the maximum number of visible rows before scrolling.
func TableMaxVisibleRows(n int) TableOptions {
	return TableOptions{MaxVisibleRows: n}
}

// BeginTable starts a new table. Returns nil if table should be skipped.
// columns define the table structure.
// flags control table behavior.
// width/height specify outer dimensions (0 = auto).
func (ctx *Context) BeginTable(id string, columns []TableColumn, flags TableFlags, width, height float32) *Table {
	return ctx.BeginTableEx(id, columns, flags, width, height, TableOptions{})
}

// BeginTableEx starts a new table with additional options.
func (ctx *Context) BeginTableEx(id string, columns []TableColumn, flags TableFlags, width, height float32, opts TableOptions) *Table {
	tableID := ctx.GetID(id)

	// Get or create persistent state using the new type-safe store
	state := tableStore.Get(tableID, TableState{
		SortColumn:       -1,
		SelectedRow:      -1,
		ColumnWidths:     make([]float32, len(columns)),
		MaxContentWidths: make([]float32, len(columns)),
	})

	// Ensure column widths slice matches column count
	if len(state.ColumnWidths) != len(columns) {
		state.ColumnWidths = make([]float32, len(columns))
	}
	if len(state.MaxContentWidths) != len(columns) {
		state.MaxContentWidths = make([]float32, len(columns))
	}

	// Reset max content widths for this frame (will be updated during rendering)
	newMaxWidths := make([]float32, len(columns))
	copy(newMaxWidths, state.MaxContentWidths) // Keep previous frame's widths for initial sizing

	pos := ctx.ItemPos()

	// Calculate available width
	if width <= 0 {
		width = ctx.currentLayoutWidth()
	}

	// Calculate column widths (use previous frame's content widths for auto-sizing)
	autoSize := flags&TableFlagsAutoSizeColumns != 0
	computedColumns := computeColumnWidths(columns, state.ColumnWidths, state.MaxContentWidths, width, ctx, autoSize)

	t := &Table{
		id:             tableID,
		ctx:            ctx,
		flags:          flags,
		columns:        computedColumns,
		options:        opts,
		startX:         pos.X,
		startY:         pos.Y,
		width:          width,
		height:         height,
		rowHeight:      ctx.lineHeight(),
		currentRow:     -1, // Will be 0 after first TableNextRow
		state:          state,
		frameMaxWidths: make([]float32, len(columns)),
	}

	// Draw outer border if requested
	if flags&TableFlagsBordersOuterH != 0 {
		ctx.DrawList.AddLine(pos.X, pos.Y, pos.X+width, pos.Y, ctx.style.BorderColor, 1)
	}
	if flags&TableFlagsBordersOuterV != 0 {
		ctx.DrawList.AddLine(pos.X, pos.Y, pos.X, pos.Y+height, ctx.style.BorderColor, 1)
		ctx.DrawList.AddLine(pos.X+width, pos.Y, pos.X+width, pos.Y+height, ctx.style.BorderColor, 1)
	}

	return t
}

// computeColumnWidths calculates actual column widths based on flags and constraints.
func computeColumnWidths(columns []TableColumn, savedWidths, maxContentWidths []float32, totalWidth float32, ctx *Context, autoSize bool) []TableColumn {
	result := make([]TableColumn, len(columns))
	copy(result, columns)

	// First pass: calculate fixed and auto-sized columns
	usedWidth := float32(0)
	stretchCount := 0
	stretchWeight := float32(0)

	for i := range result {
		col := &result[i]

		// Use saved width if available (user resized)
		if savedWidths[i] > 0 {
			col.width = savedWidths[i]
			usedWidth += col.width
			continue
		}

		if col.Flags&TableColumnFlagsWidthFixed != 0 && col.InitWidth > 0 {
			// Fixed width
			col.width = col.InitWidth
			usedWidth += col.width
		} else if col.Flags&TableColumnFlagsWidthStretch != 0 {
			// Will be calculated in second pass
			stretchCount++
			weight := col.InitWidth
			if weight <= 0 {
				weight = 1
			}
			stretchWeight += weight
		} else {
			// Auto-size (default or explicit TableColumnFlagsWidthAuto)
			// Use content width if available, otherwise label width
			labelWidth := ctx.MeasureText(col.Label).X + ctx.style.ItemSpacing*2

			// Check if auto-sizing is enabled (per-column flag or table-wide flag)
			useContentWidth := autoSize || col.Flags&TableColumnFlagsWidthAuto != 0
			if useContentWidth && len(maxContentWidths) > i && maxContentWidths[i] > 0 {
				// Use max of label and content width
				contentWidth := maxContentWidths[i] + ctx.style.ItemSpacing*2
				col.width = labelWidth
				if contentWidth > col.width {
					col.width = contentWidth
				}
			} else {
				col.width = labelWidth
			}

			if col.InitWidth > 0 && col.width < col.InitWidth {
				col.width = col.InitWidth
			}
			usedWidth += col.width
		}
	}

	// Second pass: distribute remaining width to stretch columns
	if stretchCount > 0 && stretchWeight > 0 {
		remainingWidth := totalWidth - usedWidth
		if remainingWidth > 0 {
			for i := range result {
				col := &result[i]
				if col.Flags&TableColumnFlagsWidthStretch != 0 && col.width == 0 {
					weight := col.InitWidth
					if weight <= 0 {
						weight = 1
					}
					col.width = remainingWidth * (weight / stretchWeight)

					// Apply min/max constraints
					if col.MinWidth > 0 && col.width < col.MinWidth {
						col.width = col.MinWidth
					}
					if col.MaxWidth > 0 && col.width > col.MaxWidth {
						col.width = col.MaxWidth
					}
				}
			}
		}
	}

	return result
}

// TableHeadersRow renders the header row with column labels.
func (t *Table) TableHeadersRow() {
	ctx := t.ctx
	y := t.startY

	// Draw header background
	ctx.DrawList.AddRect(t.startX, y, t.width, t.rowHeight, ctx.style.HeaderBgColor)

	// Draw column headers
	x := t.startX
	for i, col := range t.columns {
		// Header text
		textColor := ctx.style.HeaderTextColor
		if textColor == 0 {
			textColor = ctx.style.TextColor
		}
		ctx.addText(x+ctx.style.ItemSpacing, y, col.Label, textColor)

		// Sort indicator if sortable
		if t.flags&TableFlagsSortable != 0 && t.state.SortColumn == i {
			indicator := "▲"
			if !t.state.SortAscending {
				indicator = "▼"
			}
			indicatorX := x + col.width - ctx.MeasureText(indicator).X - ctx.style.ItemSpacing
			ctx.addText(indicatorX, y, indicator, textColor)
		}

		// Vertical border between columns
		if t.flags&TableFlagsBordersInnerV != 0 && i < len(t.columns)-1 {
			borderX := x + col.width
			ctx.DrawList.AddLine(borderX, y, borderX, y+t.rowHeight, ctx.style.BorderColor, 1)
		}

		x += col.width
	}

	// Horizontal border below header
	if t.flags&TableFlagsBordersInnerH != 0 {
		ctx.DrawList.AddLine(t.startX, y+t.rowHeight, t.startX+t.width, y+t.rowHeight, ctx.style.BorderColor, 1)
	}

	t.rowStartY = y + t.rowHeight
}

// TableNextRow starts a new row.
func (t *Table) TableNextRow() {
	t.currentRow++
	t.currentColumn = -1

	ctx := t.ctx
	y := t.rowStartY + float32(t.currentRow)*t.rowHeight

	// Create row rect
	rowRect := Rect{X: t.startX, Y: y, W: t.width, H: t.rowHeight}

	// Alternate row background
	if t.flags&TableFlagsRowBg != 0 && t.currentRow%2 == 1 {
		ctx.DrawList.AddRect(t.startX, y, t.width, t.rowHeight, ctx.style.RowBgAltColor)
	}

	// Register row as focusable if row selection is enabled
	// Uses unified RegisterFocusable which handles click-to-focus automatically
	if t.flags&TableFlagsRowSelect != 0 {
		rowID := t.id + ID(t.currentRow+1)*1000 // Generate unique ID per row
		ctx.RegisterFocusable(rowID, "row", rowRect, FocusTypeLeaf)

		// Check if this row has registry focus (set by click or keyboard nav)
		isSelected := ctx.IsRegistryFocused(rowID)

		if isSelected {
			ctx.DrawList.AddRect(t.startX, y, t.width, t.rowHeight, ctx.style.SelectedBgColor)
			ctx.DrawDebugFocusRect(t.startX, y, t.width, t.rowHeight)

			// Auto-scroll: tell parent Scrollable to keep this row visible
			ctx.ScrollTo(y, t.rowHeight)

			// Report focus to parent via the new hierarchical focus system
			ctx.ReportChildFocus(y, t.rowHeight)
			ctx.SetFocusChildIdx(t.currentRow)
		}
	}

	// Horizontal border between rows
	if t.flags&TableFlagsBordersInnerH != 0 && t.currentRow > 0 {
		ctx.DrawList.AddLine(t.startX, y, t.startX+t.width, y, ctx.style.BorderColor, 1)
	}
}

// TableNextColumn moves to the next column and returns the draw position.
// Returns the position where content should be drawn.
func (t *Table) TableNextColumn() Vec2 {
	t.currentColumn++
	if t.currentColumn >= len(t.columns) {
		t.currentColumn = 0
	}
	return t.TableGetColumnPos()
}

// TableSetColumnIndex sets the current column explicitly.
func (t *Table) TableSetColumnIndex(column int) {
	if column >= 0 && column < len(t.columns) {
		t.currentColumn = column
	}
}

// TableGetColumnPos returns the current column's draw position.
func (t *Table) TableGetColumnPos() Vec2 {
	x := t.startX
	for i := 0; i < t.currentColumn && i < len(t.columns); i++ {
		x += t.columns[i].width
	}
	y := t.rowStartY + float32(t.currentRow)*t.rowHeight
	return Vec2{X: x + t.ctx.style.ItemSpacing, Y: y}
}

// TableGetColumnWidth returns the width of the current column.
func (t *Table) TableGetColumnWidth() float32 {
	if t.currentColumn >= 0 && t.currentColumn < len(t.columns) {
		return t.columns[t.currentColumn].width - t.ctx.style.ItemSpacing*2
	}
	return 0
}

// TableText draws text in the current column.
func (t *Table) TableText(text string) {
	pos := t.TableNextColumn()
	col := t.columns[t.currentColumn]

	// Track content width for auto-sizing
	t.trackContentWidth(text)

	// Truncate text if too wide
	maxWidth := col.width - t.ctx.style.ItemSpacing*2
	displayText := t.truncateText(text, maxWidth)

	t.ctx.addText(pos.X, pos.Y, displayText, t.ctx.style.TextColor)
}

// TableTextColored draws colored text in the current column.
func (t *Table) TableTextColored(text string, color uint32) {
	pos := t.TableNextColumn()
	col := t.columns[t.currentColumn]

	// Track content width for auto-sizing
	t.trackContentWidth(text)

	// Truncate text if too wide
	maxWidth := col.width - t.ctx.style.ItemSpacing*2
	displayText := t.truncateText(text, maxWidth)

	t.ctx.addText(pos.X, pos.Y, displayText, color)
}

// trackContentWidth updates the max content width for the current column.
func (t *Table) trackContentWidth(text string) {
	if t.currentColumn >= 0 && t.currentColumn < len(t.frameMaxWidths) {
		textWidth := t.ctx.MeasureText(text).X
		if textWidth > t.frameMaxWidths[t.currentColumn] {
			t.frameMaxWidths[t.currentColumn] = textWidth
		}
	}
}

// truncateText truncates text to fit within maxWidth.
func (t *Table) truncateText(text string, maxWidth float32) string {
	if t.ctx.MeasureText(text).X <= maxWidth {
		return text
	}

	// Iteratively shorten the text until it fits
	runes := []rune(text)
	const ellipsis = ".."

	for len(runes) > 0 {
		truncated := string(runes) + ellipsis
		if t.ctx.MeasureText(truncated).X <= maxWidth {
			return truncated
		}
		runes = runes[:len(runes)-1]
	}

	return ellipsis
}

// TableIsRowHovered returns true if the current row is hovered.
func (t *Table) TableIsRowHovered() bool {
	if t.ctx.Input == nil {
		return false
	}

	y := t.rowStartY + float32(t.currentRow)*t.rowHeight
	rect := Rect{X: t.startX, Y: y, W: t.width, H: t.rowHeight}
	return rect.Contains(Vec2{t.ctx.Input.MouseX, t.ctx.Input.MouseY})
}

// TableIsRowClicked returns true if the current row was clicked.
func (t *Table) TableIsRowClicked() bool {
	return t.TableIsRowHovered() && t.ctx.Input.MouseClicked(MouseButtonLeft)
}

// EndTable finishes the table and advances the cursor.
func (t *Table) EndTable() {
	// Calculate total height
	totalHeight := t.rowHeight // Header
	if t.currentRow >= 0 {
		totalHeight += float32(t.currentRow+1) * t.rowHeight // Data rows
	}

	// Draw bottom border if requested
	if t.flags&TableFlagsBordersOuterH != 0 {
		y := t.startY + totalHeight
		t.ctx.DrawList.AddLine(t.startX, y, t.startX+t.width, y, t.ctx.style.BorderColor, 1)
	}

	// Save content widths for next frame's auto-sizing
	// Always save - individual columns may use auto-sizing even without table flag
	t.state.MaxContentWidths = t.frameMaxWidths

	// State is automatically saved via pointer (no need to call SetState)

	// Advance cursor
	t.ctx.advanceCursor(Vec2{X: t.width, Y: totalHeight})
}

// State returns the table's current state for external manipulation.
func (t *Table) State() *TableState {
	return t.state
}

// Columns returns the computed column definitions.
func (t *Table) Columns() []TableColumn {
	return t.columns
}

// MaxVisibleRows returns the configured max visible rows (0 = unlimited).
func (t *Table) MaxVisibleRows() int {
	return t.options.MaxVisibleRows
}

// BeginTableVirtualized starts a virtualized table for large datasets.
// Unlike BeginTable, this version only renders visible rows for performance.
//
// Parameters:
//   - id: Unique table identifier
//   - columns: Column definitions
//   - flags: Table behavior flags (must include TableFlagsScrollY)
//   - width, height: Table dimensions (height required for virtualization)
//   - totalRows: Total number of rows in the dataset
//
// Usage:
//
//	table := ctx.BeginTableVirtualized("large_table", columns, flags, 0, 400, 10000)
//	if table != nil {
//	    table.TableHeadersRow()
//	    for i := table.FirstVisibleRow(); i < table.LastVisibleRow(); i++ {
//	        table.TableNextRowVirtualized(i)
//	        table.TableText(data[i].Name)
//	        // ... more columns
//	    }
//	    table.EndTable()
//	}
func (ctx *Context) BeginTableVirtualized(id string, columns []TableColumn, flags TableFlags, width, height float32, totalRows int) *Table {
	// Ensure scroll flag is set for virtualization
	flags |= TableFlagsScrollY

	// Start regular table
	t := ctx.BeginTableEx(id, columns, flags, width, height, TableOptions{})
	if t == nil {
		return nil
	}

	// Calculate visible area height (excluding header)
	visibleHeight := height - t.rowHeight
	if visibleHeight <= 0 {
		visibleHeight = height
	}

	// Create clipper for virtualization
	t.clipper = NewListClipper(totalRows, t.rowHeight, visibleHeight, t.state.ScrollOffset)
	t.totalRows = totalRows
	t.visibleTop = t.rowStartY

	return t
}

// FirstVisibleRow returns the first row index that should be rendered.
// Use this with virtualized tables to iterate only over visible rows.
func (t *Table) FirstVisibleRow() int {
	if t.clipper != nil {
		return t.clipper.StartIdx
	}
	return 0
}

// LastVisibleRow returns one past the last row index that should be rendered.
// Use this with virtualized tables to iterate only over visible rows.
func (t *Table) LastVisibleRow() int {
	if t.clipper != nil {
		return t.clipper.EndIdx
	}
	return t.totalRows
}

// TableNextRowVirtualized starts a new row at the specified index for virtualized tables.
// Unlike TableNextRow which auto-increments, this allows sparse row rendering.
// Returns true if the row is visible and should be drawn, false to skip.
func (t *Table) TableNextRowVirtualized(rowIdx int) bool {
	// Check if row should be rendered
	if t.clipper != nil && !t.clipper.ShouldRender(rowIdx) {
		return false
	}

	t.currentRow = rowIdx
	t.currentColumn = -1

	ctx := t.ctx
	y := t.rowStartY + float32(rowIdx)*t.rowHeight - t.state.ScrollOffset

	// Skip if row is outside visible area (belt and suspenders check)
	if t.clipper != nil {
		if y+t.rowHeight < t.visibleTop || y > t.visibleTop+t.height-t.rowHeight {
			return false
		}
	}

	// Alternate row background
	if t.flags&TableFlagsRowBg != 0 && rowIdx%2 == 1 {
		ctx.DrawList.AddRect(t.startX, y, t.width, t.rowHeight, ctx.style.RowBgAltColor)
	}

	// Register row as focusable if row selection is enabled
	rowRect := Rect{X: t.startX, Y: y, W: t.width, H: t.rowHeight}
	if t.flags&TableFlagsRowSelect != 0 {
		rowID := t.id + ID(rowIdx+1)*1000 // Generate unique ID per row
		ctx.RegisterFocusable(rowID, "row", rowRect, FocusTypeLeaf)

		// Sync selection from registry focus (e.g., row was clicked)
		if ctx.IsRegistryFocused(rowID) && rowIdx != t.state.SelectedRow {
			t.state.SelectedRow = rowIdx
		}

		// Check if this row is selected
		isSelected := rowIdx == t.state.SelectedRow

		if isSelected {
			ctx.DrawList.AddRect(t.startX, y, t.width, t.rowHeight, ctx.style.SelectedBgColor)
			ctx.DrawDebugFocusRect(t.startX, y, t.width, t.rowHeight)
		}
	}

	// Horizontal border between rows
	if t.flags&TableFlagsBordersInnerH != 0 && rowIdx > 0 {
		ctx.DrawList.AddLine(t.startX, y, t.startX+t.width, y, ctx.style.BorderColor, 1)
	}

	return true
}

// TableGetColumnPosVirtualized returns the draw position accounting for scroll offset.
// Use this with virtualized tables instead of TableGetColumnPos.
func (t *Table) TableGetColumnPosVirtualized() Vec2 {
	x := t.startX
	for i := 0; i < t.currentColumn && i < len(t.columns); i++ {
		x += t.columns[i].width
	}
	y := t.rowStartY + float32(t.currentRow)*t.rowHeight - t.state.ScrollOffset
	return Vec2{X: x + t.ctx.style.ItemSpacing, Y: y}
}

// TableTextVirtualized draws text in the current column for virtualized tables.
func (t *Table) TableTextVirtualized(text string) {
	t.currentColumn++
	if t.currentColumn >= len(t.columns) {
		t.currentColumn = 0
	}

	pos := t.TableGetColumnPosVirtualized()
	col := t.columns[t.currentColumn]

	// Track content width for auto-sizing
	t.trackContentWidth(text)

	// Truncate text if too wide
	maxWidth := col.width - t.ctx.style.ItemSpacing*2
	displayText := t.truncateText(text, maxWidth)

	t.ctx.addText(pos.X, pos.Y, displayText, t.ctx.style.TextColor)
}

// TableTextColoredVirtualized draws colored text in the current column for virtualized tables.
func (t *Table) TableTextColoredVirtualized(text string, color uint32) {
	t.currentColumn++
	if t.currentColumn >= len(t.columns) {
		t.currentColumn = 0
	}

	pos := t.TableGetColumnPosVirtualized()
	col := t.columns[t.currentColumn]

	// Track content width for auto-sizing
	t.trackContentWidth(text)

	// Truncate text if too wide
	maxWidth := col.width - t.ctx.style.ItemSpacing*2
	displayText := t.truncateText(text, maxWidth)

	t.ctx.addText(pos.X, pos.Y, displayText, color)
}

// IsRowVisibleVirtualized returns true if the row at the given index is currently visible.
func (t *Table) IsRowVisibleVirtualized(rowIdx int) bool {
	if t.clipper == nil {
		return true
	}
	return t.clipper.ShouldRender(rowIdx)
}

// ScrollToRow scrolls the table to make the specified row visible.
func (t *Table) ScrollToRow(rowIdx int) {
	if t.clipper == nil || t.totalRows == 0 {
		return
	}

	visibleHeight := t.height - t.rowHeight // Exclude header
	newScroll := t.clipper.ScrollToItem(rowIdx, t.state.ScrollOffset, visibleHeight)
	t.state.ScrollOffset = newScroll
}

// TotalRows returns the total row count for virtualized tables.
func (t *Table) TotalRows() int {
	return t.totalRows
}

// HandleScrollInput processes mouse wheel input for table scrolling.
// Call this after EndTable if you want custom scroll handling.
func (t *Table) HandleScrollInput() {
	if t.ctx.Input == nil || t.clipper == nil {
		return
	}

	// Check if mouse is over table area
	tableRect := Rect{X: t.startX, Y: t.startY, W: t.width, H: t.height}
	if !tableRect.Contains(Vec2{t.ctx.Input.MouseX, t.ctx.Input.MouseY}) {
		return
	}

	if t.ctx.Input.MouseWheelY != 0 {
		visibleHeight := t.height - t.rowHeight
		maxScroll := t.clipper.MaxScroll(visibleHeight)
		newScroll := t.state.ScrollOffset - t.ctx.Input.MouseWheelY*t.rowHeight*3
		t.state.ScrollOffset = clampf(newScroll, 0, maxScroll)
	}
}
