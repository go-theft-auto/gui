package gui

// Component is the interface that all GUI components implement.
// This allows users to extend the library with custom components
// without modifying the core package.
//
// Usage (custom component):
//
//	type MyCustomComponent struct {
//	    label string
//	    value *float32
//	}
//
//	func (c *MyCustomComponent) Render(ctx *gui.Context) {
//	    pos := ctx.ItemPos()
//	    ctx.Text(c.label)
//	    // ... custom rendering
//	}
//
//	// Register and use:
//	gui.RegisterComponent("component_custom_slider", func() gui.Component {
//	    return &MyCustomComponent{}
//	})
type Component interface {
	// Render draws the component using the provided context.
	Render(ctx *Context)
}

// ComponentFactory creates a new instance of a component.
type ComponentFactory func() Component

// componentRegistry stores registered component factories.
var componentRegistry = make(map[string]ComponentFactory)

// RegisterComponent registers a custom component with the given name.
// Use the "component_" prefix for custom components to avoid naming conflicts.
//
// Example:
//
//	gui.RegisterComponent("component_color_wheel", func() gui.Component {
//	    return &ColorWheel{}
//	})
func RegisterComponent(name string, factory ComponentFactory) {
	componentRegistry[name] = factory
}

// GetComponent retrieves a component factory by name.
// Returns nil if the component is not registered.
func GetComponent(name string) ComponentFactory {
	return componentRegistry[name]
}

// UnregisterComponent removes a component from the registry.
func UnregisterComponent(name string) {
	delete(componentRegistry, name)
}

// ListComponents returns a list of all registered component names.
func ListComponents() []string {
	names := make([]string, 0, len(componentRegistry))
	for name := range componentRegistry {
		names = append(names, name)
	}
	return names
}

// ComponentWithState is a component that maintains state between frames.
type ComponentWithState interface {
	Component
	// GetState returns the component's current state.
	GetState() any
	// SetState sets the component's state (used for serialization).
	SetState(state any)
}

// ComponentWithID is a component that has a unique identifier.
type ComponentWithID interface {
	Component
	// ID returns the unique identifier for this component instance.
	ID() string
}

// InteractiveComponent is a component that can receive input.
type InteractiveComponent interface {
	Component
	// HandleInput processes input and returns true if the component value changed.
	HandleInput(ctx *Context, input *InputState) bool
}

// BuiltinComponents contains the standard component implementations.
// These use the "component_" prefix naming convention.
var BuiltinComponents = struct {
	// Text components
	Text        string // component_text
	TextWrapped string // component_text_wrapped
	TextColored string // component_text_colored

	// Button components
	Button      string // component_button
	SmallButton string // component_button_small

	// Input components
	InputText   string // component_input_text
	Slider      string // component_slider
	SliderInt   string // component_slider_int
	NumberInput string // component_number_input
	Checkbox    string // component_checkbox
	RadioButton string // component_radio_button
	ComboBox    string // component_combobox

	// Layout components
	Panel      string // component_panel
	ListBox    string // component_listbox
	Scrollable string // component_scrollable
	Table      string // component_table
	VStack     string // component_vstack
	HStack     string // component_hstack

	// Selection components
	Selectable string // component_selectable
	List       string // component_list

	// Misc components
	ProgressBar      string // component_progress_bar
	Separator        string // component_separator
	CollapsingHeader string // component_collapsing_header
	TreeNode         string // component_tree_node
}{
	Text:        "component_text",
	TextWrapped: "component_text_wrapped",
	TextColored: "component_text_colored",

	Button:      "component_button",
	SmallButton: "component_button_small",

	InputText:   "component_input_text",
	Slider:      "component_slider",
	SliderInt:   "component_slider_int",
	NumberInput: "component_number_input",
	Checkbox:    "component_checkbox",
	RadioButton: "component_radio_button",
	ComboBox:    "component_combobox",

	Panel:      "component_panel",
	ListBox:    "component_listbox",
	Scrollable: "component_scrollable",
	Table:      "component_table",
	VStack:     "component_vstack",
	HStack:     "component_hstack",

	Selectable: "component_selectable",
	List:       "component_list",

	ProgressBar:      "component_progress_bar",
	Separator:        "component_separator",
	CollapsingHeader: "component_collapsing_header",
	TreeNode:         "component_tree_node",
}

// ComponentTextWrapper wraps the Text widget as a Component.
type ComponentTextWrapper struct {
	Text  string
	Color uint32 // 0 = use default color
}

// Render implements Component.
func (c *ComponentTextWrapper) Render(ctx *Context) {
	if c.Color != 0 {
		ctx.TextColored(c.Text, c.Color)
	} else {
		ctx.Text(c.Text)
	}
}

// ComponentButtonWrapper wraps the Button widget as a Component.
type ComponentButtonWrapper struct {
	Label   string
	Options []Option
	Clicked bool // Set after Render if button was clicked
}

// Render implements Component.
func (c *ComponentButtonWrapper) Render(ctx *Context) {
	c.Clicked = ctx.Button(c.Label, c.Options...)
}

// HandleInput implements InteractiveComponent.
func (c *ComponentButtonWrapper) HandleInput(ctx *Context, input *InputState) bool {
	return c.Clicked
}

// ComponentSliderWrapper wraps the SliderFloat widget as a Component.
type ComponentSliderWrapper struct {
	Label   string
	Value   *float32
	Min     float32
	Max     float32
	Options []Option
	Changed bool // Set after Render if value changed
}

// Render implements Component.
func (c *ComponentSliderWrapper) Render(ctx *Context) {
	c.Changed = ctx.SliderFloat(c.Label, c.Value, c.Min, c.Max, c.Options...)
}

// HandleInput implements InteractiveComponent.
func (c *ComponentSliderWrapper) HandleInput(ctx *Context, input *InputState) bool {
	return c.Changed
}

// ComponentInputTextWrapper wraps the InputText widget as a Component.
type ComponentInputTextWrapper struct {
	Label   string
	Value   *string
	Options []Option
	Changed bool // Set after Render if value changed
}

// Render implements Component.
func (c *ComponentInputTextWrapper) Render(ctx *Context) {
	c.Changed = ctx.InputText(c.Label, c.Value, c.Options...)
}

// HandleInput implements InteractiveComponent.
func (c *ComponentInputTextWrapper) HandleInput(ctx *Context, input *InputState) bool {
	return c.Changed
}

// ComponentCheckboxWrapper wraps the Checkbox widget as a Component.
type ComponentCheckboxWrapper struct {
	Label   string
	Value   *bool
	Options []Option
	Changed bool // Set after Render if value changed
}

// Render implements Component.
func (c *ComponentCheckboxWrapper) Render(ctx *Context) {
	c.Changed = ctx.Checkbox(c.Label, c.Value, c.Options...)
}

// HandleInput implements InteractiveComponent.
func (c *ComponentCheckboxWrapper) HandleInput(ctx *Context, input *InputState) bool {
	return c.Changed
}

// RegisterBuiltinComponents registers all builtin components.
// Call this during initialization if you want to use the component registry.
func RegisterBuiltinComponents() {
	RegisterComponent(BuiltinComponents.Text, func() Component {
		return &ComponentTextWrapper{}
	})
	RegisterComponent(BuiltinComponents.Button, func() Component {
		return &ComponentButtonWrapper{}
	})
	RegisterComponent(BuiltinComponents.Slider, func() Component {
		return &ComponentSliderWrapper{}
	})
	RegisterComponent(BuiltinComponents.InputText, func() Component {
		return &ComponentInputTextWrapper{}
	})
	RegisterComponent(BuiltinComponents.Checkbox, func() Component {
		return &ComponentCheckboxWrapper{}
	})
}

// RenderComponent renders a registered component by name.
// Returns false if the component is not registered.
func RenderComponent(ctx *Context, name string, config func(Component)) bool {
	factory := GetComponent(name)
	if factory == nil {
		return false
	}

	component := factory()
	if config != nil {
		config(component)
	}
	component.Render(ctx)
	return true
}
