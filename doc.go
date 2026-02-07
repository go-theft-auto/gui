/*
Package gui provides an immediate-mode GUI library inspired by Dear ImGui,
designed as idiomatic Go with a dedicated Context type.

# Overview

This package implements an immediate-mode GUI where the UI is rebuilt every frame.
Unlike retained-mode GUIs, there's no need to manage widget state or handle callbacks.
The UI code is simply called each frame, and widgets return interaction results directly.

# Quick Start

	// Setup
	renderer, _ := opengl.NewRenderer(1920, 1080)
	ui := gui.New(renderer, gui.WithStyle(gui.GTAStyle()))

	// Game loop
	for !window.ShouldClose() {
	    input := pollInput(window)

	    ctx := ui.Begin(input, gui.Vec2{1920, 1080}, deltaTime)

	    ctx.Panel("Menu", gui.Gap(8), gui.Padding(12))(func() {
	        ctx.Text("Hello World")
	        if ctx.Button("Click Me") {
	            // Button was clicked
	        }
	    })

	    ui.End()
	    window.SwapBuffers()
	}

# Keyboard Shortcuts Reference

This section documents all keyboard shortcuts available in the GUI system.

## InputText Widget Shortcuts

Navigation:

	Left Arrow       Move cursor one character left
	Right Arrow      Move cursor one character right
	Ctrl+Left        Move cursor one word left
	Ctrl+Right       Move cursor one word right
	Home             Jump to start of text
	End              Jump to end of text

Selection:

	Shift+Left       Extend selection one character left
	Shift+Right      Extend selection one character right
	Ctrl+Shift+Left  Extend selection one word left
	Ctrl+Shift+Right Extend selection one word right
	Shift+Home       Select from cursor to start
	Shift+End        Select from cursor to end
	Ctrl+A           Select all text

Clipboard Operations:

	Ctrl+C           Copy selected text to clipboard
	Ctrl+X           Cut selected text to clipboard
	Ctrl+V           Paste from clipboard

Undo/Redo:

	Ctrl+Z           Undo last change
	Ctrl+Y           Redo (alternative 1)
	Ctrl+Shift+Z     Redo (alternative 2)

Control:

	Enter            Confirm input and unfocus
	Escape           Cancel and unfocus
	Backspace        Delete character before cursor (or delete selection)
	Delete           Delete character after cursor (or delete selection)

## Scrollable Areas (ListBox, Scrollable, List)

	Mouse Wheel      Scroll vertically
	Shift+Wheel      Scroll horizontally (when enabled)
	Page Up          Scroll up by 80% of viewport height
	Page Down        Scroll down by 80% of viewport height
	Home             Scroll to top (when focused)
	End              Scroll to bottom (when focused)

## ComboBox Widget

	Click            Open/close dropdown menu
	Escape           Close dropdown menu
	Mouse Wheel      Scroll dropdown items
	Type characters  Filter items (when WithSearchable() is set)
	Backspace        Delete filter character

## Slider Widgets (SliderFloat, SliderInt)

	Click+Drag       Adjust value by dragging
	Mouse Wheel      Increment/decrement value (when hovered)

## NumberInput Widgets (NumberInputFloat, NumberInputInt)

	Click+Drag       Adjust value by dragging left/right
	Click (release)  Enter text edit mode (if drag distance < 3px)
	Enter            Confirm text edit
	Escape           Cancel text edit
	0-9, ., -        Input digits/decimal/negative
	Backspace        Delete digit

## Collapsing Headers / Tree Nodes

	Click            Toggle expanded/collapsed state

## Panel Focus (requires PanelRegistry)

	Ctrl+Tab         Cycle to next panel
	Ctrl+Shift+Tab   Cycle to previous panel

# Complete Component List

All components are organized by category. When using the component registry,
components use the "component_*" naming prefix.

## Text Components

	ctx.Text(text string)
	    Draws basic text at current cursor position.
	    Component name: component_text

	ctx.TextColored(text string, color uint32)
	    Draws text with a specific color.
	    Component name: component_text_colored

	ctx.TextDisabled(text string)
	    Draws text with the disabled/grayed out color.

	ctx.TextWrapped(text string, maxWidth float32)
	    Draws text with automatic word wrapping.
	    Use maxWidth=0 for current layout width.
	    Component name: component_text_wrapped

	ctx.LabelText(label, value string)
	    Draws a label and value side by side.

	ctx.BulletText(text string)
	    Draws a bullet point followed by text.

## Button Components

	ctx.Button(label string, opts ...Option) bool
	    Draws a clickable button. Returns true when clicked.
	    Options: WithID, WithDisabled, WithWidth, WithHeight
	    Component name: component_button

	ctx.SmallButton(label string, opts ...Option) bool
	    Draws a smaller button without extra padding.
	    Component name: component_button_small

## Input Components

	ctx.InputText(label string, value *string, opts ...Option) bool
	    Full-featured text input with cursor, selection, clipboard, undo/redo.
	    Returns true when value changes.
	    Options: WithID, WithDisabled, WithWidth
	    Component name: component_input_text

	ctx.SliderFloat(label string, value *float32, min, max float32, opts ...Option) bool
	    Horizontal slider for float values. Returns true when value changes.
	    Options: WithID, WithWidth, WithFormat, WithStep
	    Component name: component_slider

	ctx.SliderInt(label string, value *int, min, max int, opts ...Option) bool
	    Horizontal slider for integer values. Returns true when value changes.
	    Options: WithID, WithWidth, WithFormat, WithStep
	    Component name: component_slider_int

	ctx.NumberInputFloat(label string, value *float32, opts ...Option) bool
	    Numeric input with drag-to-adjust. Click to type, drag to adjust.
	    Options: WithID, WithWidth, WithFormat, WithStep, WithRange,
	             WithDragSpeed, WithPrefix, WithSuffix
	    Component name: component_number_input

	ctx.NumberInputInt(label string, value *int, opts ...Option) bool
	    Integer variant of NumberInputFloat.

	ctx.Checkbox(label string, value *bool, opts ...Option) bool
	    Checkbox with label. Returns true when toggled.
	    Options: WithID, WithDisabled
	    Component name: component_checkbox

	ctx.RadioButton(label string, active bool, opts ...Option) bool
	    Radio button. Returns true when clicked.
	    Options: WithID, WithDisabled
	    Component name: component_radio_button

	ctx.ComboBox(label string, selectedIndex *int, items []string, opts ...Option) bool
	    Dropdown selection widget. Returns true when selection changes.
	    Options: WithID, WithWidth, WithSearchable, WithMaxDropdownHeight
	    Component name: component_combobox

	ctx.ProgressBar(fraction float32, opts ...Option)
	    Displays a progress bar. Fraction should be 0.0 to 1.0.
	    Options: WithWidth, WithHeight
	    Component name: component_progress_bar

## Selection Components

	ctx.Selectable(label string, selected bool, opts ...Option) bool
	    Selectable list item. Returns true when clicked.
	    Options: WithID, WithDisabled
	    Component name: component_selectable

## Layout Components

	ctx.Panel(title string, opts ...LayoutOption) func(func())
	    Container with background and optional title.
	    Options: Gap, GapX, GapY, Padding, PaddingXY, Width, Height, Align, Justify
	    Component name: component_panel

	ctx.CenteredPanel(id string, opts ...LayoutOption) func(func())
	    Panel centered on screen using cached size from previous frame.
	    Solves ImGui's "can't center without knowing size" issue.

	ctx.VStack(opts ...LayoutOption) func(func())
	    Vertical layout container (items stack top to bottom).
	    Options: Gap, GapX, GapY, Padding, Width, Height, Align, Justify
	    Component name: component_vstack

	ctx.HStack(opts ...LayoutOption) func(func())
	    Horizontal layout container (items stack left to right).
	    Options: Gap, GapX, GapY, Padding, Width, Height, Align, Justify
	    Component name: component_hstack

	ctx.Row(contents func())
	    Alias for HStack with default options.

	ctx.ListBox(id string, height float32, opts ...LayoutOption) func(func())
	    Scrollable list area with smooth scrolling.
	    Component name: component_listbox

	ctx.Scrollable(id string, height float32, opts ...Option) func(func())
	    Generic scrollable wrapper for any content.
	    Options: ShowScrollbar, ScrollbarPosition, EnableHorizontal, ClampToContent
	    Component name: component_scrollable

	ctx.List(id string, height float32, opts ...Option) *ListBuilder
	    Advanced list with sections, search filter, and nested widgets.
	    Returns a builder for fluent configuration.
	    Options: ShowScrollbar, WithFilter, WithMultiSelect, DefaultOpen
	    Component name: component_list

## Table Component

	ctx.BeginTable(id string, columns []TableColumn, flags TableFlags, width, height float32) *Table
	    Starts a table. Returns nil if table should be skipped.
	    Component name: component_table

	ctx.BeginTableVirtualized(id string, columns []TableColumn, flags TableFlags, width, height float32, totalRows int) *Table
	    Virtualized table for large datasets (1000+ rows).

	Table methods:
	    t.TableHeadersRow()                    Draw column headers
	    t.TableNextRow()                       Start new row
	    t.TableNextColumn() Vec2               Move to next column
	    t.TableText(text string)               Draw text in current column
	    t.TableTextColored(text, color)        Draw colored text
	    t.TableIsRowHovered() bool             Check if row is hovered
	    t.TableIsRowClicked() bool             Check if row was clicked
	    t.EndTable()                           Finish table

	TableFlags:
	    TableFlagsResizable        Enable column resizing
	    TableFlagsSortable         Enable sorting indicators
	    TableFlagsRowSelect        Enable row selection
	    TableFlagsScrollY          Enable vertical scrolling
	    TableFlagsStickyHeader     Keep header visible when scrolling
	    TableFlagsAutoSizeColumns  Auto-size columns to content
	    TableFlagsBordersInner     Inner borders (H+V)
	    TableFlagsBordersOuter     Outer borders (H+V)
	    TableFlagsBorders          All borders
	    TableFlagsRowBg            Alternate row backgrounds
	    TableFlagsHighlightHover   Highlight hovered row

## Tree/Collapsing Components

	ctx.CollapsingHeader(label string, opts ...Option) bool
	    Collapsible header. Returns true if section is expanded.
	    Options: WithID
	    Component name: component_collapsing_header

	ctx.TreeNode(label string, opts ...Option) bool
	    Tree node with indent. Call TreePop() after contents.
	    Returns true if expanded.
	    Component name: component_tree_node

	ctx.TreePop()
	    End a tree node started with TreeNode().

## Misc Components

	ctx.Separator()
	    Draws a horizontal separator line.
	    Component name: component_separator

	ctx.Spacing(pixels float32)
	    Adds vertical space.

	ctx.Bullet()
	    Draws a bullet point (inline element).

	ctx.Indent(pixels float32)
	    Increases cursor X position.

	ctx.Unindent(pixels float32)
	    Decreases cursor X position.

	ctx.SameLine()
	    Places next widget on same line as previous.

	ctx.Tooltip(text string)
	    Shows tooltip at mouse position.

# Widget Options Reference

Common options available for widgets:

	WithID(id string)              Explicit ID (use in loops)
	WithDisabled(disabled bool)    Disable widget interaction
	WithWidth(width float32)       Set widget width
	WithHeight(height float32)     Set widget height
	WithFormat(format string)      Printf-style format (e.g., "%.2f")
	WithStep(step float32)         Value increment step
	WithRange(min, max float32)    Value range constraints
	WithDragSpeed(speed float32)   Drag sensitivity
	WithPrefix(prefix string)      Text prefix (e.g., "X:")
	WithSuffix(suffix string)      Text suffix (e.g., "px")
	WithSearchable()               Enable typing to filter (ComboBox)
	WithMaxDropdownHeight(h)       Limit dropdown height
	WithColumns(n int)             Multi-column layout
	ShowScrollbar(always bool)     Control scrollbar visibility
	ScrollbarPosition(side)        Scrollbar side (left/right)
	EnableHorizontal()             Enable horizontal scroll
	ClampToContent()               Don't scroll past content
	WithFilter(placeholder)        Enable search filter (List)
	WithMultiSelect()              Allow multiple selection
	DefaultOpen()                  Start sections expanded

# Layout Options Reference

Options for Panel, VStack, HStack, and other layout containers:

	Gap(pixels float32)            Space between all children
	GapX(pixels float32)           Horizontal spacing override
	GapY(pixels float32)           Vertical spacing override
	Padding(pixels float32)        Inner padding on all sides
	PaddingXY(x, y float32)        Separate X/Y padding
	Width(w float32)               Fixed width
	Height(h float32)              Fixed height
	Align(alignment Alignment)     Cross-axis alignment
	Justify(just Justification)    Main-axis alignment

Alignment values: AlignStart, AlignCenter, AlignEnd, AlignStretch
Justification values: JustifyStart, JustifyCenter, JustifyEnd, JustifyBetween

# Spacing Constants

Use these instead of magic numbers:

	SpaceNone  = 0   // No spacing
	SpaceXS    = 2   // Extra small
	SpaceSM    = 4   // Small (default item spacing)
	SpaceMD    = 8   // Medium (default padding)
	SpaceLG    = 12  // Large
	SpaceXL    = 16  // Extra large
	Space2XL   = 24  // 2x extra large
	Space3XL   = 32  // 3x extra large
	Space4XL   = 48  // 4x extra large

# State Types

Widget state types for GetState/SetState:

	ScrollState           Scroll position for ListBox
	InputTextState        Cursor, selection, undo stack for InputText
	TreeNodeState         Expanded state for TreeNode
	CollapsingHeaderState Collapsed state for CollapsingHeader
	SliderState           Drag state for Slider
	ComboBoxState         Open/scroll state for ComboBox
	ScrollableState       Full scroll state for Scrollable
	ListState             Scroll/filter/selection for List
	NumberInputState      Edit/drag state for NumberInput
	TableState            Column widths, sort, selection for Table

# Component Interface

For creating custom components:

	type Component interface {
	    Render(ctx *Context)
	}

	// Register custom component
	gui.RegisterComponent("component_my_widget", func() gui.Component {
	    return &MyWidget{}
	})

	// Use registered component
	gui.RenderComponent(ctx, "component_my_widget", func(c gui.Component) {
	    w := c.(*MyWidget)
	    w.Value = &myValue
	})

# Clipboard Integration

To enable clipboard support, implement ClipboardProvider:

	type ClipboardProvider interface {
	    GetText() string
	    SetText(text string)
	}

	// GLFW example:
	type GLFWClipboard struct {
	    window *glfw.Window
	}

	func (c *GLFWClipboard) GetText() string {
	    return c.window.GetClipboardString()
	}

	func (c *GLFWClipboard) SetText(text string) {
	    c.window.SetClipboardString(text)
	}

	// Register during init:
	gui.SetClipboardProvider(&GLFWClipboard{window: window})

# Text Utilities

For advanced text handling:

	// Wrap text with mode selection
	lines := gui.WrapText(ctx, text, maxWidth, gui.WrapModeAuto)

	// Smart wrap (auto-detects CJK)
	lines := gui.WrapTextSmart(ctx, text, maxWidth)

	// Truncate with ellipsis
	truncated := gui.TruncateText(ctx, text, maxWidth)

	// Measure wrapped text
	size := gui.MeasureWrappedText(ctx, text, maxWidth, gui.WrapModeWord)

WrapMode values: WrapModeWord, WrapModeChar, WrapModeAuto

# Performance Optimizations

Built-in optimizations:

  - sync.Pool for DrawList buffer reuse
  - Batched rendering by texture
  - Pre-allocated glyph buffer for text
  - Per-frame text measurement cache
  - ListClipper for virtualizing large lists
  - Table row virtualization

For large datasets, use:

	// Virtualized table (only renders visible rows)
	table := ctx.BeginTableVirtualized("data", cols, flags, w, h, 10000)
	for i := table.FirstVisibleRow(); i < table.LastVisibleRow(); i++ {
	    if table.TableNextRowVirtualized(i) {
	        table.TableTextVirtualized(data[i].Name)
	    }
	}
	table.EndTable()

	// ListClipper for custom lists
	clipper := gui.NewListClipper(totalItems, itemHeight, visibleHeight, scrollY)
	for i := clipper.StartIdx; i < clipper.EndIdx; i++ {
	    y := clipper.ItemY(i, baseY, scrollY)
	    // Draw item at y
	}

# Differences from Dear ImGui

This implementation addresses known ImGui issues:

  - Layout centering: CenteredPanel uses two-pass layout
  - ID conflicts: Auto-ID generation prevents loop bugs
  - Text wrapping: Built-in TextWrapped with CJK support
  - Hidden state: Explicit StateStore interface
  - Type safety: Go generics instead of void*
  - Memory: sync.Pool instead of manual management
  - InputText: Full cursor, selection, clipboard, undo/redo
  - Virtualization: Built-in ListClipper and table virtualization
  - Smooth scrolling: Interpolated scroll positions
*/
package gui
