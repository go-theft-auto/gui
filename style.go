package gui

// Spacing constants for consistent layout (similar to Tailwind spacing scale).
// Use these instead of raw numbers for maintainability.
const (
	SpaceNone float32 = 0
	SpaceXS   float32 = 2  // Extra small
	SpaceSM   float32 = 4  // Small (default item spacing)
	SpaceMD   float32 = 8  // Medium (default padding)
	SpaceLG   float32 = 12 // Large
	SpaceXL   float32 = 16 // Extra large
	Space2XL  float32 = 24 // 2x extra large
	Space3XL  float32 = 32 // 3x extra large
	Space4XL  float32 = 48 // 4x extra large
)

// Style defines the visual appearance of UI elements.
type Style struct {
	// Colors
	TextColor          uint32
	TextDisabledColor  uint32
	TextHighlightColor uint32

	// Panel colors
	PanelColor           uint32
	PanelBorderColor     uint32
	PanelHeaderBgColor   uint32 // Header background (0 = use ButtonColor)
	PanelHeaderTextColor uint32 // Header text (0 = use TextColor)

	// Button colors
	ButtonColor         uint32
	ButtonHoveredColor  uint32
	ButtonActiveColor   uint32
	ButtonDisabledColor uint32

	// Selection colors
	SelectedBgColor   uint32
	SelectedTextColor uint32
	HoveredBgColor    uint32

	// Input colors
	InputBgColor        uint32
	InputFocusedBgColor uint32
	InputBorderColor    uint32

	// Separator
	SeparatorColor uint32

	// Table colors
	BorderColor     uint32 // General border color (tables, frames)
	HeaderBgColor   uint32 // Table header background
	HeaderTextColor uint32 // Table header text (0 = use TextColor)
	RowBgAltColor   uint32 // Alternate row background

	// Scrollbar
	ScrollbarBgColor     uint32
	ScrollbarGrabColor   uint32
	ScrollbarGrabHovered uint32

	// Slider colors
	SliderTrackColor  uint32 // Background track
	SliderFillColor   uint32 // Filled portion
	SliderGrabColor   uint32 // Handle/grab
	SliderGrabHovered uint32 // Handle when hovered
	SliderGrabActive  uint32 // Handle when dragging

	// Dropdown/ComboBox colors
	DropdownBgColor uint32 // Dropdown menu background
	ComboArrowColor uint32 // Arrow indicator color

	// Focus indicator
	FocusColor uint32

	// Toast notification colors
	ToastInfoColor    uint32
	ToastSuccessColor uint32
	ToastWarningColor uint32
	ToastErrorColor   uint32

	// Font
	FontName string // Font name for use with FontManager (e.g., "font1", "plate")

	// Sizing
	FontScale     float32
	CharWidth     float32
	CharHeight    float32
	ItemSpacing   float32 // Default gap between items
	PanelPadding  float32
	ButtonPadding float32
	InputPadding  float32

	// Border
	BorderSize float32
	Rounding   float32 // Corner rounding (0 = sharp corners)

	// Scrollbar
	ScrollbarSize float32
}

// DefaultStyle returns the default style with sensible defaults.
func DefaultStyle() Style {
	return Style{
		// Text colors
		TextColor:          ColorWhite,
		TextDisabledColor:  ColorGray,
		TextHighlightColor: ColorYellow,

		// Panel
		PanelColor:           RGBA(20, 20, 20, 200),
		PanelBorderColor:     RGBA(80, 80, 80, 255),
		PanelHeaderBgColor:   RGBA(40, 40, 45, 255),
		PanelHeaderTextColor: 0, // Use TextColor

		// Buttons
		ButtonColor:         RGBA(50, 50, 50, 255),
		ButtonHoveredColor:  RGBA(70, 70, 70, 255),
		ButtonActiveColor:   RGBA(90, 90, 90, 255),
		ButtonDisabledColor: RGBA(30, 30, 30, 255),

		// Selection
		SelectedBgColor:   RGBA(50, 100, 150, 255),
		SelectedTextColor: ColorWhite,
		HoveredBgColor:    RGBA(60, 60, 60, 255),

		// Input
		InputBgColor:        RGBA(30, 30, 30, 255),
		InputFocusedBgColor: RGBA(40, 40, 50, 255),
		InputBorderColor:    RGBA(100, 100, 100, 255),

		// Separator
		SeparatorColor: RGBA(80, 80, 80, 255),

		// Table
		BorderColor:     RGBA(80, 80, 80, 255),
		HeaderBgColor:   RGBA(40, 40, 40, 255),
		HeaderTextColor: 0, // Use TextColor
		RowBgAltColor:   RGBA(35, 35, 35, 255),

		// Scrollbar
		ScrollbarBgColor:     RGBA(30, 30, 30, 255),
		ScrollbarGrabColor:   RGBA(80, 80, 80, 255),
		ScrollbarGrabHovered: RGBA(100, 100, 100, 255),

		// Slider
		SliderTrackColor:  RGBA(40, 40, 40, 255),
		SliderFillColor:   RGBA(50, 100, 150, 255),
		SliderGrabColor:   RGBA(100, 100, 100, 255),
		SliderGrabHovered: RGBA(120, 120, 120, 255),
		SliderGrabActive:  RGBA(140, 140, 140, 255),

		// Dropdown
		DropdownBgColor: RGBA(25, 25, 25, 250),
		ComboArrowColor: RGBA(180, 180, 180, 255),

		// Focus indicator
		FocusColor: ColorCyan,

		// Toast notifications
		ToastInfoColor:    RGBA(50, 100, 150, 230),
		ToastSuccessColor: RGBA(50, 130, 80, 230),
		ToastWarningColor: RGBA(180, 130, 40, 230),
		ToastErrorColor:   RGBA(180, 60, 60, 230),

		// Sizing
		FontScale:     1.0,
		CharWidth:     8,
		CharHeight:    8,
		ItemSpacing:   4,
		PanelPadding:  8,
		ButtonPadding: 6,
		InputPadding:  4,

		// Border
		BorderSize: 1,
		Rounding:   0,

		// Scrollbar
		ScrollbarSize: 12,
	}
}

// GTAStyle returns a GTA San Andreas-inspired style.
// Dark theme with cyan/yellow accents reminiscent of the game's menus.
func GTAStyle() Style {
	return Style{
		// Text colors - GTA uses white/yellow text
		TextColor:          ColorWhite,
		TextDisabledColor:  RGBA(128, 128, 128, 255),
		TextHighlightColor: RGBA(255, 200, 0, 255), // GTA yellow

		// Panel - dark semi-transparent
		PanelColor:           RGBA(0, 0, 0, 220),
		PanelBorderColor:     RGBA(100, 100, 100, 255),
		PanelHeaderBgColor:   RGBA(0, 60, 90, 255),   // GTA cyan tinted
		PanelHeaderTextColor: RGBA(255, 200, 0, 255), // GTA yellow

		// Buttons
		ButtonColor:         RGBA(40, 40, 40, 255),
		ButtonHoveredColor:  RGBA(60, 80, 100, 255),
		ButtonActiveColor:   RGBA(0, 150, 200, 255), // Cyan when active
		ButtonDisabledColor: RGBA(30, 30, 30, 150),

		// Selection - GTA style cyan highlight
		SelectedBgColor:   RGBA(0, 120, 180, 255),
		SelectedTextColor: ColorWhite,
		HoveredBgColor:    RGBA(50, 70, 90, 255),

		// Input
		InputBgColor:        RGBA(20, 20, 20, 255),
		InputFocusedBgColor: RGBA(30, 40, 50, 255),
		InputBorderColor:    RGBA(0, 150, 200, 255),

		// Separator
		SeparatorColor: RGBA(0, 150, 200, 128),

		// Table (GTA cyan theme)
		BorderColor:     RGBA(0, 100, 150, 255),
		HeaderBgColor:   RGBA(0, 80, 120, 255),
		HeaderTextColor: ColorWhite,
		RowBgAltColor:   RGBA(20, 30, 40, 255),

		// Scrollbar
		ScrollbarBgColor:     RGBA(20, 20, 20, 255),
		ScrollbarGrabColor:   RGBA(0, 100, 150, 255),
		ScrollbarGrabHovered: RGBA(0, 150, 200, 255),

		// Slider (GTA cyan theme)
		SliderTrackColor:  RGBA(30, 30, 30, 255),
		SliderFillColor:   RGBA(0, 120, 180, 255),
		SliderGrabColor:   RGBA(0, 150, 200, 255),
		SliderGrabHovered: RGBA(0, 180, 230, 255),
		SliderGrabActive:  RGBA(0, 200, 255, 255),

		// Dropdown (GTA style)
		DropdownBgColor: RGBA(10, 10, 10, 250),
		ComboArrowColor: RGBA(0, 180, 230, 255),

		// Focus indicator (GTA cyan)
		FocusColor: RGBA(0, 200, 255, 255),

		// Toast notifications (GTA style)
		ToastInfoColor:    RGBA(0, 80, 120, 230),
		ToastSuccessColor: RGBA(0, 120, 60, 230),
		ToastWarningColor: RGBA(200, 150, 0, 230),
		ToastErrorColor:   RGBA(180, 40, 40, 230),

		// Font
		FontName: "font1", // Use GTA's font1 when loaded

		// Sizing (slightly larger for GTA feel)
		FontScale:     1.5,
		CharWidth:     8,
		CharHeight:    8,
		ItemSpacing:   6,
		PanelPadding:  12,
		ButtonPadding: 8,
		InputPadding:  6,

		// Border
		BorderSize: 1,
		Rounding:   0, // Sharp corners like GTA menus

		// Scrollbar
		ScrollbarSize: 14,
	}
}

// DarkStyle returns a modern dark theme.
func DarkStyle() Style {
	s := DefaultStyle()
	s.PanelColor = RGBA(25, 25, 25, 240)
	s.PanelHeaderBgColor = RGBA(35, 35, 40, 255)
	s.ButtonColor = RGBA(45, 45, 45, 255)
	s.ButtonHoveredColor = RGBA(65, 65, 65, 255)
	s.SelectedBgColor = RGBA(65, 105, 225, 255) // Royal blue
	return s
}

// LightStyle returns a light theme.
func LightStyle() Style {
	return Style{
		TextColor:          RGBA(20, 20, 20, 255),
		TextDisabledColor:  RGBA(150, 150, 150, 255),
		TextHighlightColor: RGBA(0, 100, 200, 255),

		PanelColor:           RGBA(245, 245, 245, 250),
		PanelBorderColor:     RGBA(200, 200, 200, 255),
		PanelHeaderBgColor:   RGBA(220, 220, 225, 255),
		PanelHeaderTextColor: RGBA(40, 40, 40, 255),

		ButtonColor:         RGBA(220, 220, 220, 255),
		ButtonHoveredColor:  RGBA(200, 200, 200, 255),
		ButtonActiveColor:   RGBA(180, 180, 180, 255),
		ButtonDisabledColor: RGBA(230, 230, 230, 255),

		SelectedBgColor:   RGBA(0, 120, 215, 255),
		SelectedTextColor: ColorWhite,
		HoveredBgColor:    RGBA(230, 230, 230, 255),

		InputBgColor:        ColorWhite,
		InputFocusedBgColor: ColorWhite,
		InputBorderColor:    RGBA(150, 150, 150, 255),

		SeparatorColor: RGBA(200, 200, 200, 255),

		// Table
		BorderColor:     RGBA(200, 200, 200, 255),
		HeaderBgColor:   RGBA(230, 230, 230, 255),
		HeaderTextColor: RGBA(20, 20, 20, 255),
		RowBgAltColor:   RGBA(250, 250, 250, 255),

		ScrollbarBgColor:     RGBA(240, 240, 240, 255),
		ScrollbarGrabColor:   RGBA(180, 180, 180, 255),
		ScrollbarGrabHovered: RGBA(160, 160, 160, 255),

		// Slider (light theme)
		SliderTrackColor:  RGBA(220, 220, 220, 255),
		SliderFillColor:   RGBA(0, 120, 215, 255),
		SliderGrabColor:   RGBA(180, 180, 180, 255),
		SliderGrabHovered: RGBA(160, 160, 160, 255),
		SliderGrabActive:  RGBA(140, 140, 140, 255),

		// Dropdown (light theme)
		DropdownBgColor: RGBA(255, 255, 255, 255),
		ComboArrowColor: RGBA(80, 80, 80, 255),

		FontScale:     1.0,
		CharWidth:     8,
		CharHeight:    8,
		ItemSpacing:   4,
		PanelPadding:  8,
		ButtonPadding: 6,
		InputPadding:  4,

		BorderSize: 1,
		Rounding:   0,

		ScrollbarSize: 12,
	}
}
