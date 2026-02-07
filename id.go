package gui

import "hash/fnv"

// ID uniquely identifies a widget for state persistence.
// IDs are stable across frames for the same widget.
type ID uint64

// GetID generates a stable ID from a string label.
// The ID is unique within the current ID stack context.
// Uses an auto-incrementing counter to differentiate same labels in loops.
func (ctx *Context) GetID(label string) ID {
	ctx.idCounter++

	// Combine: parent ID + label hash + call counter
	// This ensures unique IDs even for:
	// 1. Same label in different parent contexts (parent ID differs)
	// 2. Same label in a loop (counter differs)
	parentID := ID(0)
	if len(ctx.idStack) > 0 {
		parentID = ctx.idStack[len(ctx.idStack)-1]
	}

	h := fnv.New64a()
	h.Write([]byte(label))
	labelHash := h.Sum64()

	// Combine components: parent (32 bits) + counter (16 bits) + label (16 bits)
	return ID(uint64(parentID)<<32 | uint64(ctx.idCounter)<<16 | labelHash&0xFFFF)
}

// GetIDFromInt generates an ID from an integer.
// Useful for items in arrays/slices.
func (ctx *Context) GetIDFromInt(n int) ID {
	ctx.idCounter++

	parentID := ID(0)
	if len(ctx.idStack) > 0 {
		parentID = ctx.idStack[len(ctx.idStack)-1]
	}

	return ID(uint64(parentID)<<32 | uint64(ctx.idCounter)<<16 | uint64(n)&0xFFFF)
}

// PushID pushes an ID onto the stack for nested widgets.
// All GetID calls will be relative to this parent ID.
func (ctx *Context) PushID(label string) {
	ctx.idStack = append(ctx.idStack, ctx.GetID(label))
}

// PushIDInt pushes an integer-based ID onto the stack.
func (ctx *Context) PushIDInt(n int) {
	ctx.idStack = append(ctx.idStack, ctx.GetIDFromInt(n))
}

// PopID removes the last ID from the stack.
func (ctx *Context) PopID() {
	if len(ctx.idStack) > 0 {
		ctx.idStack = ctx.idStack[:len(ctx.idStack)-1]
	}
}

// CurrentID returns the current parent ID (top of stack).
func (ctx *Context) CurrentID() ID {
	if len(ctx.idStack) > 0 {
		return ctx.idStack[len(ctx.idStack)-1]
	}
	return 0
}
