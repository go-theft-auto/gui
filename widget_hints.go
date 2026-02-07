package gui

import (
	"fmt"
	"strings"
)

// HintKey represents a keyboard key for hint display.
// Use the predefined constants for consistency.
type HintKey string

// Standard hint keys with consistent formatting.
// Uses Unicode arrows which are supported by the built-in font.
const (
	HintKeyUp        HintKey = "↑"
	HintKeyDown      HintKey = "↓"
	HintKeyLeft      HintKey = "←"
	HintKeyRight     HintKey = "→"
	HintKeyUpDown    HintKey = "↑↓"
	HintKeyLeftRight HintKey = "←→"
	HintKeyArrows    HintKey = "←→↑↓"
	HintKeyEnter     HintKey = "Enter"
	HintKeyEscape    HintKey = "Esc"
	HintKeySpace     HintKey = "Space"
	HintKeyTab       HintKey = "Tab"
	HintKeyBackspace HintKey = "Bksp"
	HintKeyDelete    HintKey = "Del"
	HintKeyHome      HintKey = "Home"
	HintKeyEnd       HintKey = "End"
	HintKeyPageUp    HintKey = "PgUp"
	HintKeyPageDown  HintKey = "PgDn"
	HintKeyType      HintKey = "Type"
	HintKeyScroll    HintKey = "Scroll"
	HintKeyClick     HintKey = "Click"
	HintKeyDrag      HintKey = "Drag"
	HintKeyF1        HintKey = "F1"
	HintKeyF2        HintKey = "F2"
	HintKeyF3        HintKey = "F3"
	HintKeyF4        HintKey = "F4"
	HintKeyF5        HintKey = "F5"
)

// HintAction pairs a key with its action description.
type HintAction struct {
	Key    HintKey
	Action string
}

// Hint creates a HintAction for use with HintFooter.
//
// Usage:
//
//	ctx.HintFooter(
//	    gui.Hint(gui.HintKeyUpDown, "Navigate"),
//	    gui.Hint(gui.HintKeyEnter, "Select"),
//	    gui.Hint(gui.HintKeyEscape, "Close"),
//	)
func Hint(key HintKey, action string) HintAction {
	return HintAction{Key: key, Action: action}
}

// HintFooter draws a consistent footer with keyboard hints.
// Automatically adds a separator before the hints.
//
// Usage:
//
//	ctx.HintFooter(
//	    gui.Hint(gui.HintKeyUpDown, "Navigate"),
//	    gui.Hint(gui.HintKeyEnter, "Select"),
//	    gui.Hint(gui.HintKeyEscape, "Close"),
//	)
//
// Renders as: "[↑↓] Navigate  [Enter] Select  [Esc] Close"
func (ctx *Context) HintFooter(hints ...HintAction) {
	if len(hints) == 0 {
		return
	}

	ctx.Separator()

	var parts []string
	for _, h := range hints {
		parts = append(parts, fmt.Sprintf("[%s] %s", h.Key, h.Action))
	}

	text := strings.Join(parts, "  ")
	ctx.TextColored(text, ColorGray)
}

// HintHeader draws a hint at the top of a section (before content).
// Use for instructions like "Type to search..." or "Drag to reorder".
//
// Usage:
//
//	ctx.HintHeader("Type to search...")
func (ctx *Context) HintHeader(text string) {
	ctx.TextColored(text, ColorGray)
}

// HintEmpty draws an empty state message.
// Use when a list or section has no content.
//
// Usage:
//
//	if len(items) == 0 {
//	    ctx.HintEmpty("No items found")
//	}
func (ctx *Context) HintEmpty(text string) {
	if text == "" {
		text = "(none)"
	}
	ctx.TextColored(text, ColorGray)
}

// HintStatus draws a status line showing counts or state.
//
// Usage:
//
//	ctx.HintStatus("%d/%d visible", enabledCount, totalCount)
func (ctx *Context) HintStatus(format string, args ...any) {
	text := fmt.Sprintf(format, args...)
	ctx.TextColored(text, ColorGray)
}

// ScrollHints tracks scroll position and draws indicators.
type ScrollHints struct {
	offset    int
	visible   int
	total     int
	showCount bool
}

// HintScroll creates scroll indicators for a list.
// Call Before() before drawing items and After() after.
//
// Usage:
//
//	scroll := ctx.HintScroll(scrollOffset, visibleItems, totalItems)
//	scroll.Before() // Draws "^ more above" if needed
//	for i := startIdx; i < endIdx; i++ {
//	    // draw items
//	}
//	scroll.After() // Draws "v more below" if needed
func (ctx *Context) HintScroll(offset, visible, total int) *ScrollHints {
	return &ScrollHints{
		offset:    offset,
		visible:   visible,
		total:     total,
		showCount: false,
	}
}

// WithCount enables showing the count of hidden items.
//
// Usage:
//
//	scroll := ctx.HintScroll(offset, visible, total).WithCount()
func (sh *ScrollHints) WithCount() *ScrollHints {
	sh.showCount = true
	return sh
}

// Before draws the "more above" indicator if there are items above the viewport.
func (sh *ScrollHints) Before(ctx *Context) {
	if sh.offset > 0 {
		if sh.showCount {
			ctx.TextColored(fmt.Sprintf("↑ %d more above", sh.offset), ColorGray)
		} else {
			ctx.TextColored("↑ more above", ColorGray)
		}
	}
}

// After draws the "more below" indicator if there are items below the viewport.
func (sh *ScrollHints) After(ctx *Context) {
	remaining := sh.total - sh.offset - sh.visible
	if remaining > 0 {
		if sh.showCount {
			ctx.TextColored(fmt.Sprintf("↓ %d more below", remaining), ColorGray)
		} else {
			ctx.TextColored("↓ more below", ColorGray)
		}
	}
}

// HasMoreAbove returns true if there are items above the viewport.
func (sh *ScrollHints) HasMoreAbove() bool {
	return sh.offset > 0
}

// HasMoreBelow returns true if there are items below the viewport.
func (sh *ScrollHints) HasMoreBelow() bool {
	return sh.total-sh.offset-sh.visible > 0
}

// HintComingSoon draws a "coming soon" placeholder.
func (ctx *Context) HintComingSoon() {
	ctx.TextColored("(coming soon)", ColorGray)
}

// Common hint presets for frequently used patterns.

// HintFooterNav draws navigation hints: [↑↓] Navigate [Enter] Select [Esc] Close
func (ctx *Context) HintFooterNav() {
	ctx.HintFooter(
		Hint(HintKeyUpDown, "Navigate"),
		Hint(HintKeyEnter, "Select"),
		Hint(HintKeyEscape, "Close"),
	)
}

// HintFooterConfirm draws confirmation hints: [Enter] Confirm [Esc] Cancel
func (ctx *Context) HintFooterConfirm() {
	ctx.HintFooter(
		Hint(HintKeyEnter, "Confirm"),
		Hint(HintKeyEscape, "Cancel"),
	)
}

// HintFooterClose draws a simple close hint: [Esc] Close
func (ctx *Context) HintFooterClose() {
	ctx.HintFooter(
		Hint(HintKeyEscape, "Close"),
	)
}

// HintFooterToggle draws toggle hints: [Enter] Toggle [Esc] Close
func (ctx *Context) HintFooterToggle() {
	ctx.HintFooter(
		Hint(HintKeyEnter, "Toggle"),
		Hint(HintKeyEscape, "Close"),
	)
}

// HintFooterSearch draws search hints: [Type] Search [Backspace] Clear [Esc] Close
func (ctx *Context) HintFooterSearch() {
	ctx.HintFooter(
		Hint(HintKeyType, "Search"),
		Hint(HintKeyBackspace, "Clear"),
		Hint(HintKeyEscape, "Close"),
	)
}
