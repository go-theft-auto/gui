package gui

// FocusType identifies the kind of focusable widget in the hierarchy.
type FocusType uint8

const (
	// FocusTypeContainer can contain focusable children (panels, groups)
	FocusTypeContainer FocusType = iota

	// FocusTypeLeaf is a terminal focusable element (button, input)
	FocusTypeLeaf

	// FocusTypeSection is a collapsible container
	FocusTypeSection

	// FocusTypeList has indexed children (tables, lists)
	FocusTypeList
)

// String returns a human-readable name for the focus type.
func (t FocusType) String() string {
	switch t {
	case FocusTypeContainer:
		return "Container"
	case FocusTypeLeaf:
		return "Leaf"
	case FocusTypeSection:
		return "Section"
	case FocusTypeList:
		return "List"
	default:
		return "Unknown"
	}
}

// FocusNode represents one level in the focus hierarchy.
// Each node knows its ID, type, and which child (if any) has focus.
type FocusNode struct {
	ID       ID        // Widget ID for state lookup
	Name     string    // Debug-friendly identifier
	Type     FocusType // Widget category
	ChildIdx int       // Which child is focused (-1 = self/none)
	Rect     Rect      // Bounds for hit testing

	// Saved parent focus state (restored when this scope ends)
	savedChildFocusSet    bool
	savedChildFocusY      float32
	savedChildFocusHeight float32
}

// FocusPath tracks the active path from root to the focused leaf widget.
// This enables hierarchical focus tracking where parents know which
// child has focus and where focus is within that child.
//
// Example path for a focused table row inside a scrollable inside a panel:
//
//	[0] Panel      (ChildIdx=0, points to Scrollable)
//	[1] Scrollable (ChildIdx=2, points to Table row)
//	[2] Table      (ChildIdx=5, row index)
type FocusPath struct {
	nodes   []FocusNode
	version uint64 // Incremented on change for dirty checking
}

// NewFocusPath creates an empty focus path.
func NewFocusPath() *FocusPath {
	return &FocusPath{
		nodes: make([]FocusNode, 0, 8),
	}
}

// Clear removes all nodes from the path.
func (fp *FocusPath) Clear() {
	fp.nodes = fp.nodes[:0]
	fp.version++
}

// Push adds a node to the path.
func (fp *FocusPath) Push(node FocusNode) {
	fp.nodes = append(fp.nodes, node)
	fp.version++
}

// Pop removes and returns the last node, or empty node if path is empty.
func (fp *FocusPath) Pop() FocusNode {
	if len(fp.nodes) == 0 {
		return FocusNode{ChildIdx: -1}
	}
	n := len(fp.nodes) - 1
	node := fp.nodes[n]
	fp.nodes = fp.nodes[:n]
	fp.version++
	return node
}

// Depth returns the current depth of the focus path.
func (fp *FocusPath) Depth() int {
	return len(fp.nodes)
}

// At returns the node at the given depth, or empty node if out of range.
func (fp *FocusPath) At(depth int) FocusNode {
	if depth < 0 || depth >= len(fp.nodes) {
		return FocusNode{ChildIdx: -1}
	}
	return fp.nodes[depth]
}

// Leaf returns the deepest (most specific) focused node, or empty if no focus.
func (fp *FocusPath) Leaf() FocusNode {
	if len(fp.nodes) == 0 {
		return FocusNode{ChildIdx: -1}
	}
	return fp.nodes[len(fp.nodes)-1]
}

// Root returns the topmost focused node, or empty if no focus.
func (fp *FocusPath) Root() FocusNode {
	if len(fp.nodes) == 0 {
		return FocusNode{ChildIdx: -1}
	}
	return fp.nodes[0]
}

// Contains returns true if the given ID is anywhere in the focus path.
func (fp *FocusPath) Contains(id ID) bool {
	for _, node := range fp.nodes {
		if node.ID == id {
			return true
		}
	}
	return false
}

// IndexOf returns the depth of the given ID in the path, or -1 if not found.
func (fp *FocusPath) IndexOf(id ID) int {
	for i, node := range fp.nodes {
		if node.ID == id {
			return i
		}
	}
	return -1
}

// Version returns the current version number.
// Incremented each time the path changes, useful for dirty checking.
func (fp *FocusPath) Version() uint64 {
	return fp.version
}

// SetChildIdx updates the ChildIdx of the node at the given depth.
// Returns false if depth is out of range.
func (fp *FocusPath) SetChildIdx(depth int, childIdx int) bool {
	if depth < 0 || depth >= len(fp.nodes) {
		return false
	}
	if fp.nodes[depth].ChildIdx != childIdx {
		fp.nodes[depth].ChildIdx = childIdx
		fp.version++
	}
	return true
}

// Nodes returns a copy of all nodes in the path.
func (fp *FocusPath) Nodes() []FocusNode {
	result := make([]FocusNode, len(fp.nodes))
	copy(result, fp.nodes)
	return result
}

// FocusInfo is returned by EndFocusScope to inform the parent about
// focus within the scope that just ended.
type FocusInfo struct {
	// HasFocusedChild is true if any child within the scope had focus
	HasFocusedChild bool

	// FocusedChildIdx is the index of the focused child within this scope
	// -1 if no child is focused or if the scope itself is focused
	FocusedChildIdx int

	// FocusedChildY is the Y position of the focused child (for auto-scroll)
	// Relative to the scope's content area
	FocusedChildY float32

	// FocusedChildHeight is the height of the focused child
	FocusedChildHeight float32
}
