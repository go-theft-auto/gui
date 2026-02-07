// Command gen renders every widget with sample data, captures framebuffer pixels,
// and saves JPEG screenshots to doc/imgs/.
//
// Usage:
//
//	devbox shell
//	go run ./doc/gen/
package main

import (
	"fmt"
	"image"
	"image/jpeg"
	"os"
	"path/filepath"
	"runtime"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"

	"github.com/go-theft-auto/gui"
	"github.com/go-theft-auto/gui/backend/opengl"
)

func init() {
	runtime.LockOSThread()
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// screenshot defines a single widget screenshot to capture.
type screenshot struct {
	name   string                 // filename without extension
	width  int                    // viewport width
	height int                    // viewport height
	draw   func(ctx *gui.Context) // widget drawing function
	frames int                    // extra frames to render (0 = default 2)
}

func run() error {
	if err := glfw.Init(); err != nil {
		return fmt.Errorf("glfw init: %w", err)
	}
	defer glfw.Terminate()

	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
	glfw.WindowHint(glfw.Visible, glfw.False)

	window, err := glfw.CreateWindow(800, 600, "screenshot-gen", nil, nil)
	if err != nil {
		return fmt.Errorf("create window: %w", err)
	}
	window.MakeContextCurrent()

	if err := gl.Init(); err != nil {
		return fmt.Errorf("gl init: %w", err)
	}

	renderer, err := opengl.NewRenderer(800, 600)
	if err != nil {
		return fmt.Errorf("gui renderer: %w", err)
	}
	defer renderer.Delete()

	ui := gui.New(renderer, gui.WithStyle(gui.GTAStyle()))

	outDir := filepath.Join("doc", "imgs")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	shots := buildScreenshots()

	for _, s := range shots {
		if err := capture(window, renderer, ui, s, outDir); err != nil {
			return fmt.Errorf("capture %s: %w", s.name, err)
		}
		fmt.Printf("  %s.jpg (%dx%d)\n", s.name, s.width, s.height)
	}

	fmt.Printf("\nGenerated %d screenshots in %s/\n", len(shots), outDir)
	return nil
}

func capture(_ *glfw.Window, renderer *opengl.Renderer, _ *gui.GUI, s screenshot, outDir string) error {
	// Only update the renderer projection — do NOT call window.SetSize because
	// GLFW processes resizes asynchronously, causing framebuffer/scissor mismatches.
	// The hidden window stays at 800×600 (larger than every screenshot).
	renderer.Resize(s.width, s.height)

	// Fresh GUI per screenshot to avoid state leaking between captures.
	ui := gui.New(renderer, gui.WithStyle(gui.GTAStyle()))

	frames := 2
	if s.frames > 0 {
		frames = s.frames
	}

	for i := 0; i < frames; i++ {
		gl.Viewport(0, 0, int32(s.width), int32(s.height))
		gl.ClearColor(0.12, 0.12, 0.14, 1.0)
		gl.Clear(gl.COLOR_BUFFER_BIT)

		displaySize := gui.Vec2{X: float32(s.width), Y: float32(s.height)}
		ctx := ui.Begin(&gui.InputState{}, displaySize, 1.0/60.0)
		s.draw(ctx)
		if err := ui.End(); err != nil {
			return err
		}
	}

	// Read pixels
	pixels := make([]byte, s.width*s.height*4)
	gl.ReadPixels(0, 0, int32(s.width), int32(s.height), gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(pixels))

	// Flip vertically (OpenGL origin is bottom-left)
	rowLen := s.width * 4
	tmp := make([]byte, rowLen)
	for y := 0; y < s.height/2; y++ {
		top := y * rowLen
		bot := (s.height - 1 - y) * rowLen
		copy(tmp, pixels[top:top+rowLen])
		copy(pixels[top:top+rowLen], pixels[bot:bot+rowLen])
		copy(pixels[bot:bot+rowLen], tmp)
	}

	// Create image
	img := image.NewRGBA(image.Rect(0, 0, s.width, s.height))
	copy(img.Pix, pixels)

	// Encode JPEG
	path := filepath.Join(outDir, s.name+".jpg")
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return jpeg.Encode(f, img, &jpeg.Options{Quality: 90})
}

// buildScreenshots returns the list of all widget screenshots to generate.
func buildScreenshots() []screenshot {
	// Shared state for widgets that need pointers.
	var (
		checked     = true
		unchecked   = false
		radioIdx    = 1
		radioHIdx   = 0
		inputText   = "Hello, world!"
		numFloat    = float32(3.14)
		numInt      = 42
		sliderFloat = float32(0.65)
		sliderInt   = 7
		comboIdx    = 1
		sectionOpen = true
	)

	return []screenshot{
		{
			name: "text", width: 400, height: 200,
			draw: func(ctx *gui.Context) {
				ctx.SetCursorPos(12, 12)
				ctx.VStack(gui.Gap(6))(func() {
					ctx.Text("Plain text")
					ctx.TextColored("Colored text (yellow)", gui.ColorYellow)
					ctx.TextDisabled("Disabled text")
					ctx.TextWrapped("This is wrapped text that will break across lines when it reaches the edge of the available width.", 380)
					ctx.LabelText("Label:", "Value")
					ctx.BulletText("Bullet item")
				})
			},
		},
		{
			name: "button", width: 400, height: 120,
			draw: func(ctx *gui.Context) {
				ctx.SetCursorPos(12, 12)
				ctx.VStack(gui.Gap(8))(func() {
					ctx.Button("Standard Button")
					ctx.HStack(gui.Gap(8))(func() {
						ctx.SmallButton("Small A")
						ctx.SmallButton("Small B")
						ctx.SmallButton("Small C")
					})
				})
			},
		},
		{
			name: "checkbox", width: 300, height: 80,
			draw: func(ctx *gui.Context) {
				ctx.SetCursorPos(12, 12)
				ctx.VStack(gui.Gap(6))(func() {
					ctx.Checkbox("Enabled feature", &checked)
					ctx.Checkbox("Disabled feature", &unchecked)
				})
			},
		},
		{
			name: "radio_button", width: 300, height: 120,
			draw: func(ctx *gui.Context) {
				ctx.SetCursorPos(12, 12)
				ctx.RadioGroup("Quality", &radioIdx, []string{"Low", "Medium", "High"})
			},
		},
		{
			name: "radio_group_horizontal", width: 400, height: 80,
			draw: func(ctx *gui.Context) {
				ctx.SetCursorPos(12, 12)
				ctx.RadioGroupHorizontal("Status", &radioHIdx, []string{"Active", "Idle", "Offline"})
			},
		},
		{
			name: "input_text", width: 400, height: 80,
			draw: func(ctx *gui.Context) {
				ctx.SetCursorPos(12, 12)
				ctx.InputText("Name", &inputText, gui.WithWidth(300))
			},
		},
		{
			name: "number_input", width: 400, height: 100,
			draw: func(ctx *gui.Context) {
				ctx.SetCursorPos(12, 12)
				ctx.VStack(gui.Gap(6))(func() {
					ctx.NumberInputFloat("Position", &numFloat, gui.WithWidth(200))
					ctx.NumberInputInt("Count", &numInt, gui.WithRange(0, 100), gui.WithWidth(200))
				})
			},
		},
		{
			name: "slider", width: 400, height: 100,
			draw: func(ctx *gui.Context) {
				ctx.SetCursorPos(12, 12)
				ctx.VStack(gui.Gap(6))(func() {
					ctx.SliderFloat("Volume", &sliderFloat, 0, 1)
					ctx.SliderInt("Level", &sliderInt, 0, 10)
				})
			},
		},
		{
			name: "combobox", width: 400, height: 80,
			draw: func(ctx *gui.Context) {
				ctx.SetCursorPos(12, 12)
				ctx.ComboBox("Difficulty", &comboIdx, []string{"Easy", "Normal", "Hard", "Nightmare"})
			},
		},
		{
			name: "progress_bar", width: 400, height: 100,
			draw: func(ctx *gui.Context) {
				ctx.SetCursorPos(12, 12)
				ctx.VStack(gui.Gap(8))(func() {
					ctx.ProgressBar(0.25, gui.WithWidth(370))
					ctx.ProgressBar(0.65, gui.WithWidth(370))
					ctx.ProgressBar(1.0, gui.WithWidth(370))
				})
			},
		},
		{
			name: "collapsing_header", width: 400, height: 150,
			draw: func(ctx *gui.Context) {
				ctx.SetCursorPos(12, 12)
				ctx.VStack(gui.Gap(4))(func() {
					if ctx.CollapsingHeader("Open Header") {
						ctx.Text("  Visible content inside header")
					}
					ctx.CollapsingHeader("Closed Header")
				})
			},
		},
		{
			name: "tree_node", width: 400, height: 180,
			draw: func(ctx *gui.Context) {
				ctx.SetCursorPos(12, 12)
				if ctx.TreeNode("Root") {
					ctx.Text("Child 1")
					if ctx.TreeNode("Child 2") {
						ctx.Text("Nested item A")
						ctx.Text("Nested item B")
						ctx.TreePop()
					}
					ctx.TreePop()
				}
			},
		},
		{
			name: "selectable", width: 400, height: 150,
			draw: func(ctx *gui.Context) {
				ctx.SetCursorPos(12, 12)
				ctx.VStack(gui.Gap(2))(func() {
					ctx.Selectable("Infernus", true, gui.WithID("sel_0"))
					ctx.Selectable("Banshee", false, gui.WithID("sel_1"))
					ctx.Selectable("Sultan", false, gui.WithID("sel_2"))
					ctx.Selectable("Cheetah", false, gui.WithID("sel_3"))
				})
			},
		},
		{
			name: "section", width: 400, height: 200, frames: 3,
			draw: func(ctx *gui.Context) {
				ctx.SetCursorPos(12, 12)
				ctx.VStack(gui.Gap(4))(func() {
					ctx.Section("Settings", gui.Open(&sectionOpen))(func() {
						ctx.Text("Volume: 80%")
						ctx.Text("Brightness: 100%")
					})
					ctx.Section("Advanced")(func() {
						ctx.Text("Hidden by default")
					})
				})
			},
		},
		{
			name: "panel", width: 350, height: 250,
			draw: func(ctx *gui.Context) {
				ctx.SetCursorPos(12, 12)
				ctx.Panel("Game Settings", gui.Width(320), gui.Padding(12))(func() {
					ctx.Text("Configure your game")
					ctx.Separator()
					ctx.LabelText("Player:", "CJ")
					ctx.LabelText("Health:", "100%")
					ctx.Button("Apply")
				})
			},
		},
		{
			name: "centered_panel", width: 500, height: 300, frames: 3,
			draw: func(ctx *gui.Context) {
				ctx.CenteredPanel("dialog")(func() {
					ctx.Text("Are you sure you want to exit?")
					ctx.Spacing(12)
					ctx.HStack(gui.Gap(8))(func() {
						ctx.Button("Yes")
						ctx.Button("No")
					})
				})
			},
		},
		{
			name: "scrollable", width: 400, height: 200,
			draw: func(ctx *gui.Context) {
				ctx.SetCursorPos(12, 12)
				ctx.Scrollable("demo_scroll", 170, gui.ShowScrollbar(true))(func() {
					for i := 0; i < 20; i++ {
						ctx.Text(fmt.Sprintf("Line %d: Scrollable content", i+1))
					}
				})
			},
		},
		{
			name: "table", width: 500, height: 250, frames: 3,
			draw: func(ctx *gui.Context) {
				ctx.SetCursorPos(12, 12)
				columns := []gui.TableColumn{
					{Label: "Name", InitWidth: 150},
					{Label: "Type", InitWidth: 100},
					{Label: "Health", Flags: gui.TableColumnFlagsWidthFixed, InitWidth: 80},
				}
				table := ctx.BeginTable("demo_table", columns,
					gui.TableFlagsBorders|gui.TableFlagsRowBg|gui.TableFlagsHighlightHover|gui.TableFlagsStickyHeader,
					470, 220)
				if table != nil {
					table.TableHeadersRow()
					rows := [][3]string{
						{"CJ", "Player", "100"},
						{"Sweet", "NPC", "85"},
						{"Ryder", "NPC", "70"},
						{"Big Smoke", "NPC", "90"},
						{"Cesar", "NPC", "80"},
					}
					for _, row := range rows {
						table.TableNextRow()
						table.TableText(row[0])
						table.TableText(row[1])
						table.TableText(row[2])
					}
					table.EndTable()
				}
			},
		},
		{
			name: "list", width: 400, height: 300, frames: 3,
			draw: func(ctx *gui.Context) {
				ctx.SetCursorPos(12, 12)
				list := ctx.List("demo_list", 270, gui.ShowScrollbar(true))

				list.Section("Vehicles", gui.DefaultOpen()).
					Item("Infernus", true).
					Item("Banshee", false).
					Item("Sultan", false).
					End()

				list.Section("Weapons", gui.DefaultOpen()).
					Item("AK-47", false).
					Item("M4", false).
					Item("Desert Eagle", false).
					End()

				list.End()
			},
		},
		{
			name: "graph", width: 500, height: 200,
			draw: func(ctx *gui.Context) {
				ctx.SetCursorPos(12, 12)
				data := []gui.GraphData{
					{Label: "FPS", Values: []float32{55, 60, 58, 62, 59, 61, 57, 63, 60, 58, 62, 64, 60, 56, 59, 61}, Color: gui.ColorGreen},
					{Label: "Frame Time", Values: []float32{18, 16, 17, 16, 17, 16, 18, 16, 17, 17, 16, 15, 17, 18, 17, 16}, Color: gui.ColorYellow},
				}
				ctx.Graph("demo_graph", data, 170,
					gui.WithGraphGridLines(4),
					gui.WithGraphLegend(),
					gui.WithWidth(470),
				)
			},
		},
		{
			name: "histogram", width: 500, height: 200,
			draw: func(ctx *gui.Context) {
				ctx.SetCursorPos(12, 12)
				bars := []gui.HistogramBar{
					{Label: "Core 0", Value: 75, Color: gui.ColorGreen},
					{Label: "Core 1", Value: 45, Color: gui.ColorCyan},
					{Label: "Core 2", Value: 90, Color: gui.ColorRed},
					{Label: "Core 3", Value: 60, Color: gui.ColorYellow},
					{Label: "Core 4", Value: 30, Color: gui.ColorMagenta},
				}
				ctx.Histogram("demo_hist", bars, 170,
					gui.WithHistogramShowValues(),
					gui.WithWidth(470),
				)
			},
		},
		{
			name: "sequencer", width: 600, height: 200,
			draw: func(ctx *gui.Context) {
				ctx.SetCursorPos(12, 12)
				config := gui.SequencerConfig{
					Duration:    10.0,
					CurrentTime: 3.5,
					Tracks: []gui.SequencerTrack{
						{Name: "Root", Keyframes: []float32{0, 2.5, 5.0, 7.5}, Color: gui.ColorCyan},
						{Name: "Arm_L", Keyframes: []float32{0, 3.0, 6.0}, Color: gui.ColorGreen},
						{Name: "Arm_R", Keyframes: []float32{1.0, 4.0, 8.0}, Color: gui.ColorYellow},
					},
				}
				ctx.Sequencer("demo_seq", config, 170, gui.WithWidth(570))
			},
		},
		{
			name: "toast", width: 500, height: 250,
			draw: func(ctx *gui.Context) {
				ts := &gui.ToastState{
					Toasts: []gui.ToastNotification{
						{Message: "File loaded successfully", Type: gui.ToastTypeInfo, Duration: 3.0, Elapsed: 0.3},
						{Message: "Settings saved", Type: gui.ToastTypeSuccess, Duration: 3.0, Elapsed: 0.3},
						{Message: "Low disk space", Type: gui.ToastTypeWarning, Duration: 3.0, Elapsed: 0.3},
						{Message: "Connection failed", Type: gui.ToastTypeError, Duration: 3.0, Elapsed: 0.3},
					},
				}
				ctx.DrawToasts(ts)
			},
		},
		{
			name: "hint_footer", width: 500, height: 60,
			draw: func(ctx *gui.Context) {
				ctx.SetCursorPos(12, 12)
				ctx.HintFooter(
					gui.Hint(gui.HintKeyUpDown, "Navigate"),
					gui.Hint(gui.HintKeyEnter, "Select"),
					gui.Hint(gui.HintKeyEscape, "Close"),
				)
			},
		},
		{
			name: "separator", width: 400, height: 80,
			draw: func(ctx *gui.Context) {
				ctx.SetCursorPos(12, 12)
				ctx.VStack(gui.Gap(4))(func() {
					ctx.Text("Above separator")
					ctx.Separator()
					ctx.Text("Below separator")
				})
			},
		},
		{
			name: "layout", width: 500, height: 200,
			draw: func(ctx *gui.Context) {
				ctx.SetCursorPos(12, 12)
				ctx.VStack(gui.Gap(8))(func() {
					ctx.Text("VStack + HStack demo:")
					ctx.HStack(gui.Gap(8))(func() {
						ctx.Button("Left")
						ctx.Button("Center")
						ctx.Button("Right")
					})
					ctx.Row(func() {
						ctx.Text("Row A")
						ctx.Text("Row B")
						ctx.Text("Row C")
					})
				})
			},
		},
		{
			name: "modal_menu", width: 450, height: 400,
			draw: func(ctx *gui.Context) {
				ctx.SetCursorPos(12, 12)
				ctx.Panel("Select Vehicle", gui.Width(420), gui.Padding(12))(func() {
					ctx.TextColored("Type to search...", gui.ColorGray)
					ctx.Separator()
					ctx.TextColored("6 items", gui.ColorGray)
					items := []string{"Infernus", "* Banshee", "Sultan", "Cheetah", "Turismo", "Bullet"}
					for i, item := range items {
						ctx.Selectable(item, i == 0, gui.WithID(fmt.Sprintf("mm_%d", i)))
					}
					ctx.Separator()
					ctx.TextColored("[^ v] Navigate  [Enter] Select", gui.ColorGray)
				})
			},
		},
	}
}

// simpleMenuData implements gui.MenuDataSource for the modal menu screenshot.
type simpleMenuData struct {
	items    []string
	filtered []int
	marked   map[int]bool
}

func (d *simpleMenuData) Count() int {
	if d.filtered != nil {
		return len(d.filtered)
	}
	return len(d.items)
}

func (d *simpleMenuData) Label(index int) string {
	if d.filtered != nil {
		if index < len(d.filtered) {
			return d.items[d.filtered[index]]
		}
		return ""
	}
	if index < len(d.items) {
		return d.items[index]
	}
	return ""
}

func (d *simpleMenuData) IsMarked(index int) bool {
	actual := index
	if d.filtered != nil && index < len(d.filtered) {
		actual = d.filtered[index]
	}
	return d.marked[actual]
}

func (d *simpleMenuData) Filter(query string) {
	if query == "" {
		d.filtered = nil
		return
	}
	d.filtered = nil
	for i, item := range d.items {
		if containsIgnoreCase(item, query) {
			d.filtered = append(d.filtered, i)
		}
	}
}

func containsIgnoreCase(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		match := true
		for j := 0; j < len(sub); j++ {
			sc, qc := s[i+j], sub[j]
			if sc >= 'A' && sc <= 'Z' {
				sc += 'a' - 'A'
			}
			if qc >= 'A' && qc <= 'Z' {
				qc += 'a' - 'A'
			}
			if sc != qc {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
