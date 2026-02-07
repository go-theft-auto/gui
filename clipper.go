package gui

// ListClipper helps virtualize large lists by calculating the visible item range.
// This is critical for performance with large datasets (1000+ items) where
// rendering all items every frame would cause significant slowdown.
//
// Usage:
//
//	clipper := NewListClipper(totalItems, itemHeight, visibleHeight, scrollY)
//	for i := clipper.StartIdx; i < clipper.EndIdx; i++ {
//	    y := clipper.ItemY(i, baseY, scrollY)
//	    // Draw item at y position
//	}
type ListClipper struct {
	StartIdx   int     // First visible item index (inclusive)
	EndIdx     int     // Last visible item index (exclusive)
	ItemHeight float32 // Height of each item
	TotalItems int     // Total number of items in the list
}

// NewListClipper calculates the visible item range for a scrollable list.
//
// Parameters:
//   - totalItems: Total number of items in the list
//   - itemHeight: Height of each item in pixels
//   - visibleHeight: Height of the visible area in pixels
//   - scrollY: Current vertical scroll offset in pixels
//
// Returns a ListClipper with StartIdx and EndIdx set to the visible range.
func NewListClipper(totalItems int, itemHeight, visibleHeight, scrollY float32) *ListClipper {
	if totalItems == 0 || itemHeight <= 0 {
		return &ListClipper{
			StartIdx:   0,
			EndIdx:     0,
			ItemHeight: itemHeight,
			TotalItems: totalItems,
		}
	}

	// Calculate first visible item
	startIdx := int(scrollY / itemHeight)
	if startIdx < 0 {
		startIdx = 0
	}

	// Calculate how many items fit in the visible area (+2 for partial visibility at top/bottom)
	visibleCount := int(visibleHeight/itemHeight) + 2
	endIdx := startIdx + visibleCount

	// Clamp to valid range
	if startIdx > totalItems {
		startIdx = totalItems
	}
	if endIdx > totalItems {
		endIdx = totalItems
	}

	return &ListClipper{
		StartIdx:   startIdx,
		EndIdx:     endIdx,
		ItemHeight: itemHeight,
		TotalItems: totalItems,
	}
}

// ShouldRender returns true if the item at the given index should be rendered.
// Use this when iterating through all items to skip invisible ones.
func (c *ListClipper) ShouldRender(idx int) bool {
	return idx >= c.StartIdx && idx < c.EndIdx
}

// ItemY calculates the Y position for an item relative to the visible area.
//
// Parameters:
//   - idx: The item index
//   - baseY: The Y position of the list's top edge
//   - scrollY: Current scroll offset
//
// Returns the Y position where the item should be drawn.
func (c *ListClipper) ItemY(idx int, baseY, scrollY float32) float32 {
	return baseY + float32(idx)*c.ItemHeight - scrollY
}

// VisibleCount returns the number of items that should be rendered.
func (c *ListClipper) VisibleCount() int {
	return c.EndIdx - c.StartIdx
}

// ContentHeight returns the total content height (for scrollbar calculations).
func (c *ListClipper) ContentHeight() float32 {
	return float32(c.TotalItems) * c.ItemHeight
}

// MaxScroll returns the maximum valid scroll offset.
func (c *ListClipper) MaxScroll(visibleHeight float32) float32 {
	maxScroll := c.ContentHeight() - visibleHeight
	if maxScroll < 0 {
		return 0
	}
	return maxScroll
}

// ScrollToItem returns the scroll offset needed to make an item visible.
// If the item is already visible, returns the current scroll unchanged.
func (c *ListClipper) ScrollToItem(idx int, currentScroll, visibleHeight float32) float32 {
	if idx < 0 || idx >= c.TotalItems {
		return currentScroll
	}

	itemTop := float32(idx) * c.ItemHeight
	itemBottom := itemTop + c.ItemHeight

	// If item is above visible area, scroll up to it
	if itemTop < currentScroll {
		return itemTop
	}

	// If item is below visible area, scroll down to show it
	if itemBottom > currentScroll+visibleHeight {
		return itemBottom - visibleHeight
	}

	// Item is already visible
	return currentScroll
}
