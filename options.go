package gui

// Option configures a UI widget.
type Option func(*options)

// options holds all widget configuration via the extensions map.
// All options use the unified OptKey system for type safety.
type options struct {
	extensions map[string]any
}

// OptKey is a typed key for widget options.
// All options (built-in and custom) use this system for consistency.
//
// Example:
//
//	// Define option keys (built-in ones are already defined below)
//	var OptCustomThing = gui.NewOptKey("customThing", defaultValue)
//
//	// Set options
//	ctx.MyWidget("id", gui.WithOpt(OptCustomThing, value))
//
//	// Read in widget implementation
//	value := gui.GetOpt(opts, OptCustomThing)
type OptKey[T any] struct {
	name string
	def  T
}

// NewOptKey creates a typed option key with a default value.
// The default is returned when the option is not set.
func NewOptKey[T any](name string, defaultValue T) OptKey[T] {
	return OptKey[T]{name: name, def: defaultValue}
}

// Name returns the key name (useful for debugging).
func (k OptKey[T]) Name() string { return k.name }

// Default returns the default value for this key.
func (k OptKey[T]) Default() T { return k.def }

// WithOpt sets an option value using a typed key.
func WithOpt[T any](key OptKey[T], value T) Option {
	return func(o *options) {
		if o.extensions == nil {
			o.extensions = make(map[string]any)
		}
		o.extensions[key.name] = value
	}
}

// GetOpt retrieves an option value with type safety.
// Returns the key's default value if not set.
func GetOpt[T any](o options, key OptKey[T]) T {
	if o.extensions == nil {
		return key.def
	}
	v, ok := o.extensions[key.name]
	if !ok {
		return key.def
	}
	typed, ok := v.(T)
	if !ok {
		return key.def
	}
	return typed
}

// HasOpt returns true if the option was explicitly set.
func HasOpt[T any](o options, key OptKey[T]) bool {
	if o.extensions == nil {
		return false
	}
	_, ok := o.extensions[key.name]
	return ok
}

// applyOptions applies all options and returns the configuration.
func applyOptions(opts []Option) options {
	var o options
	for _, opt := range opts {
		opt(&o)
	}
	return o
}

// ApplyAndGet applies options and returns a single value.
// Use this in external packages to create custom widgets.
func ApplyAndGet[T any](opts []Option, key OptKey[T]) T {
	return GetOpt(applyOptions(opts), key)
}

// ApplyAndCheck returns the option value and whether it was explicitly set.
func ApplyAndCheck[T any](opts []Option, key OptKey[T]) (T, bool) {
	o := applyOptions(opts)
	return GetOpt(o, key), HasOpt(o, key)
}

// =============================================================================
// Built-in Option Keys
// =============================================================================

// ScrollbarVisibility controls when scrollbars are shown.
type ScrollbarVisibility int

const (
	ScrollbarAuto   ScrollbarVisibility = iota // Show only when content exceeds viewport
	ScrollbarAlways                            // Always show scrollbar
	ScrollbarNever                             // Never show scrollbar
)

// ScrollbarSide controls which side the scrollbar appears on.
type ScrollbarSide int

const (
	ScrollbarRight ScrollbarSide = iota // Scrollbar on right side (default)
	ScrollbarLeft                       // Scrollbar on left side
)

// RangeValue holds min/max range for sliders and number inputs.
type RangeValue struct {
	Min, Max float32
	HasRange bool
}

// FocusValue holds focus Y position and padding for auto-scroll.
type FocusValue struct {
	Y       float32
	Padding float32
	Set     bool
}

// --- Core Options ---
var (
	OptID         = NewOptKey("id", "")
	OptDisabled   = NewOptKey("disabled", false)
	OptFocused    = NewOptKey("focused", false)
	OptForceFocus = NewOptKey("forceFocus", false) // Actually grab keyboard focus
	OptWidth      = NewOptKey[float32]("width", 0)
	OptHeight     = NewOptKey[float32]("height", 0)
)

// --- Slider/NumberInput Options ---
var (
	OptFormat    = NewOptKey("format", "")
	OptStep      = NewOptKey[float32]("step", 0)
	OptRange     = NewOptKey("range", RangeValue{})
	OptDragSpeed = NewOptKey[float32]("dragSpeed", 0)
	OptPrefix    = NewOptKey("prefix", "")
	OptSuffix    = NewOptKey("suffix", "")
)

// --- ComboBox Options ---
var (
	OptSearchable        = NewOptKey("searchable", false)
	OptMaxDropdownHeight = NewOptKey[float32]("maxDropdownHeight", 0)
)

// --- RadioGroup Options ---
var (
	OptColumns = NewOptKey("columns", 0)
)

// --- Scrollable Options ---
var (
	OptScrollbarVisibility = NewOptKey("scrollbarVisibility", ScrollbarAuto)
	OptScrollbarSide       = NewOptKey("scrollbarSide", ScrollbarRight)
	OptHorizontalScroll    = NewOptKey("horizontalScroll", false)
	OptClampToContent      = NewOptKey("clampToContent", false)
	OptFocus               = NewOptKey("focus", FocusValue{})
)

// --- List Options ---
var (
	OptFilterPlaceholder = NewOptKey("filterPlaceholder", "")
	OptMultiSelect       = NewOptKey("multiSelect", false)
	OptDefaultOpen       = NewOptKey("defaultOpen", false)
)

// OpenValue wraps a boolean pointer for controlled section state.
// When Ptr is non-nil, the section is in controlled mode and writes back to it.
type OpenValue struct {
	Ptr *bool // If non-nil, section reads/writes through this pointer
}

// --- Section Options ---
var (
	OptIndentSize = NewOptKey[float32]("indentSize", 0) // Custom indent (0 = use default)
	OptNoIndent   = NewOptKey("noIndent", false)        // Skip indentation entirely
	OptOpen       = NewOptKey("open", OpenValue{})      // Controlled open state via pointer
)

// --- Graph Options ---
var (
	OptGraphYMin      = NewOptKey[float32]("graphYMin", 0)
	OptGraphYMax      = NewOptKey[float32]("graphYMax", 0)
	OptGraphGridLines = NewOptKey("graphGridLines", 0)
	OptGraphLegend    = NewOptKey("graphLegend", false)
)

// --- Histogram Options ---
var (
	OptHistogramYMin       = NewOptKey[float32]("histogramYMin", 0)
	OptHistogramYMax       = NewOptKey[float32]("histogramYMax", 0)
	OptHistogramShowValues = NewOptKey("histogramShowValues", false)
	OptHistogramHorizontal = NewOptKey("histogramHorizontal", false)
)

// --- Sequencer Options ---
var (
	OptSequencerControls = NewOptKey("sequencerControls", false)
)

// =============================================================================
// Convenience Option Functions (wrap WithOpt for common cases)
// =============================================================================

// WithID sets an explicit ID for the widget.
func WithID(id string) Option { return WithOpt(OptID, id) }

// WithDisabled disables the widget (grayed out, no interaction).
func WithDisabled(disabled bool) Option { return WithOpt(OptDisabled, disabled) }

// Focused marks the widget as keyboard-focused (visual highlight).
func Focused() Option { return WithOpt(OptFocused, true) }

// ForceFocus programmatically grabs keyboard focus for the widget.
// Use this when you want a widget to become active on render (e.g., after pressing Enter).
func ForceFocus() Option { return WithOpt(OptForceFocus, true) }

// WithWidth sets a specific width for the widget.
func WithWidth(width float32) Option { return WithOpt(OptWidth, width) }

// WithHeight sets a specific height for the widget.
func WithHeight(height float32) Option { return WithOpt(OptHeight, height) }

// WithFormat sets the display format for numeric values.
func WithFormat(format string) Option { return WithOpt(OptFormat, format) }

// WithStep sets the increment step for value adjustments.
func WithStep(step float32) Option { return WithOpt(OptStep, step) }

// WithRange sets the minimum and maximum values.
func WithRange(minVal, maxVal float32) Option {
	return WithOpt(OptRange, RangeValue{Min: minVal, Max: maxVal, HasRange: true})
}

// WithDragSpeed sets the drag sensitivity (pixels per unit change).
func WithDragSpeed(speed float32) Option { return WithOpt(OptDragSpeed, speed) }

// WithPrefix sets a prefix text displayed before the value.
func WithPrefix(prefix string) Option { return WithOpt(OptPrefix, prefix) }

// WithSuffix sets a suffix text displayed after the value.
func WithSuffix(suffix string) Option { return WithOpt(OptSuffix, suffix) }

// WithSearchable enables typing to filter items in a ComboBox.
func WithSearchable() Option { return WithOpt(OptSearchable, true) }

// WithMaxDropdownHeight limits the maximum height of dropdown menus.
func WithMaxDropdownHeight(height float32) Option { return WithOpt(OptMaxDropdownHeight, height) }

// WithColumns sets the number of columns for multi-column layouts.
func WithColumns(n int) Option { return WithOpt(OptColumns, n) }

// ShowScrollbar controls scrollbar visibility.
func ShowScrollbar(always bool) Option {
	if always {
		return WithOpt(OptScrollbarVisibility, ScrollbarAlways)
	}
	return WithOpt(OptScrollbarVisibility, ScrollbarAuto)
}

// ScrollbarPosition sets which side the scrollbar appears on.
func ScrollbarPosition(side ScrollbarSide) Option { return WithOpt(OptScrollbarSide, side) }

// EnableHorizontal enables horizontal scrolling.
func EnableHorizontal() Option { return WithOpt(OptHorizontalScroll, true) }

// ClampToContent prevents scrolling past content bounds.
func ClampToContent() Option { return WithOpt(OptClampToContent, true) }

// FocusY tracks focus position and auto-scrolls when it changes.
func FocusY(y float32, padding ...float32) Option {
	v := FocusValue{Y: y, Set: true}
	if len(padding) > 0 {
		v.Padding = padding[0]
	}
	return WithOpt(OptFocus, v)
}

// WithFilter enables a search filter input.
func WithFilter(placeholder string) Option { return WithOpt(OptFilterPlaceholder, placeholder) }

// WithMultiSelect enables selecting multiple items in a list.
func WithMultiSelect() Option { return WithOpt(OptMultiSelect, true) }

// DefaultOpen makes sections start in the expanded state.
func DefaultOpen() Option { return WithOpt(OptDefaultOpen, true) }

// IndentSize sets a custom indentation in pixels for Section content.
func IndentSize(px float32) Option { return WithOpt(OptIndentSize, px) }

// NoIndent disables automatic indentation in Section widgets.
func NoIndent() Option { return WithOpt(OptNoIndent, true) }

// Open binds the section's open/closed state to an external boolean variable.
// The section reads from and writes to this variable, making it fully controlled.
// When the user clicks to toggle, the variable is updated automatically.
//
// Usage:
//
//	ctx.Section("Windows", gui.Open(&p.windowsExpanded))(func() {
//	    // content
//	})
func Open(ptr *bool) Option { return WithOpt(OptOpen, OpenValue{Ptr: ptr}) }

// WithGraphYRange sets the Y-axis range for graphs.
func WithGraphYRange(minVal, maxVal float32) Option {
	return func(o *options) {
		WithOpt(OptGraphYMin, minVal)(o)
		WithOpt(OptGraphYMax, maxVal)(o)
	}
}

// WithGraphGridLines sets the number of horizontal grid lines.
func WithGraphGridLines(n int) Option { return WithOpt(OptGraphGridLines, n) }

// WithGraphLegend enables the legend for graphs.
func WithGraphLegend() Option { return WithOpt(OptGraphLegend, true) }

// WithHistogramYRange sets the Y-axis range for histograms.
func WithHistogramYRange(minVal, maxVal float32) Option {
	return func(o *options) {
		WithOpt(OptHistogramYMin, minVal)(o)
		WithOpt(OptHistogramYMax, maxVal)(o)
	}
}

// WithHistogramShowValues shows value text above histogram bars.
func WithHistogramShowValues() Option { return WithOpt(OptHistogramShowValues, true) }

// WithHistogramHorizontal draws horizontal bars instead of vertical.
func WithHistogramHorizontal() Option { return WithOpt(OptHistogramHorizontal, true) }

// WithSequencerControls shows play/pause controls in the sequencer.
func WithSequencerControls() Option { return WithOpt(OptSequencerControls, true) }
