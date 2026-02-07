package gui

import "fmt"

// RadioGroup draws a group of radio buttons arranged vertically.
// Returns true if the selection changed.
//
// Usage:
//
//	items := []string{"Low", "Medium", "High"}
//	if ctx.RadioGroup("Quality", &selectedIndex, items) {
//	    applyQuality(selectedIndex)
//	}
func (ctx *Context) RadioGroup(label string, selectedIndex *int, items []string, opts ...Option) bool {
	o := applyOptions(opts)

	changed := false
	columns := GetOpt(o, OptColumns)
	optID := GetOpt(o, OptID)

	ctx.VStack(Gap(ctx.style.ItemSpacing))(func() {
		// Draw label if provided
		if label != "" {
			ctx.Text(label)
		}

		// Determine layout based on columns option
		if columns <= 1 {
			// Single column (vertical layout)
			for i, item := range items {
				itemID := fmt.Sprintf("%s_%d", label, i)
				if optID != "" {
					itemID = fmt.Sprintf("%s_%d", optID, i)
				}
				if ctx.RadioButton(item, i == *selectedIndex, WithID(itemID)) {
					*selectedIndex = i
					changed = true
				}
			}
		} else {
			// Multi-column layout
			itemsPerRow := (len(items) + columns - 1) / columns
			for row := range itemsPerRow {
				ctx.HStack(Gap(ctx.style.ItemSpacing * 2))(func() {
					for col := range columns {
						idx := row + col*itemsPerRow
						if idx >= len(items) {
							continue
						}
						itemID := fmt.Sprintf("%s_%d", label, idx)
						if optID != "" {
							itemID = fmt.Sprintf("%s_%d", optID, idx)
						}
						if ctx.RadioButton(items[idx], idx == *selectedIndex, WithID(itemID)) {
							*selectedIndex = idx
							changed = true
						}
					}
				})
			}
		}
	})

	return changed
}

// RadioGroupHorizontal draws a group of radio buttons arranged horizontally.
// Returns true if the selection changed.
//
// Usage:
//
//	items := []string{"On", "Off"}
//	if ctx.RadioGroupHorizontal("Status", &selectedIndex, items) {
//	    applyStatus(selectedIndex)
//	}
func (ctx *Context) RadioGroupHorizontal(label string, selectedIndex *int, items []string, opts ...Option) bool {
	o := applyOptions(opts)

	changed := false
	optID := GetOpt(o, OptID)

	ctx.VStack(Gap(ctx.style.ItemSpacing))(func() {
		// Draw label if provided
		if label != "" {
			ctx.Text(label)
		}

		// Horizontal layout for items
		ctx.HStack(Gap(ctx.style.ItemSpacing * 2))(func() {
			for i, item := range items {
				itemID := fmt.Sprintf("%s_%d", label, i)
				if optID != "" {
					itemID = fmt.Sprintf("%s_%d", optID, i)
				}
				if ctx.RadioButton(item, i == *selectedIndex, WithID(itemID)) {
					*selectedIndex = i
					changed = true
				}
			}
		})
	})

	return changed
}
