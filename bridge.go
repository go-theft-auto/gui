package gui

import (
	"unsafe"

	"github.com/go-gl/gl/v4.1-core/gl"
)

// LegacyUI provides a compatibility layer for the old UI interface.
// It wraps the new GUI system but exposes the old-style methods
// (DrawText, DrawRect, DrawLines, Begin, End) that existing code expects.
//
// This allows gradual migration from the old retained-mode UI to the
// new immediate-mode GUI without breaking existing code.
type LegacyUI struct {
	gui      *GUI
	renderer *LegacyRenderer
	width    int
	height   int

	// Input state (managed externally)
	input *InputState
}

// LegacyRenderer implements the low-level OpenGL rendering for LegacyUI.
// It provides immediate drawing calls that bypass the DrawList batching
// for compatibility with the old UI's draw-immediate style.
type LegacyRenderer struct {
	shader    uint32
	vao, vbo  uint32
	fontTex   uint32
	projLoc   int32
	texLoc    int32
	useTexLoc int32
	width     int
	height    int
}

// LegacyVertex matches the old UIVertex layout for compatibility.
type LegacyVertex struct {
	Pos      [2]float32
	TexCoord [2]float32
	Color    [4]float32
}

// Shader sources (same as old ui.go)
const legacyVertexShader = `
#version 410 core
layout (location = 0) in vec2 aPos;
layout (location = 1) in vec2 aTexCoord;
layout (location = 2) in vec4 aColor;

out vec2 TexCoord;
out vec4 Color;

uniform mat4 projection;

void main() {
    gl_Position = projection * vec4(aPos, 0.0, 1.0);
    TexCoord = aTexCoord;
    Color = aColor;
}
` + "\x00"

const legacyFragmentShader = `
#version 410 core
in vec2 TexCoord;
in vec4 Color;

out vec4 FragColor;

uniform sampler2D fontTexture;
uniform bool useTexture;

void main() {
    if (useTexture) {
        float alpha = texture(fontTexture, TexCoord).r;
        FragColor = vec4(Color.rgb, Color.a * alpha);
    } else {
        FragColor = Color;
    }
}
` + "\x00"

// NewLegacyUI creates a new LegacyUI that wraps both the new GUI
// and provides old-style immediate rendering for compatibility.
func NewLegacyUI(width, height int) (*LegacyUI, error) {
	renderer, err := newLegacyRenderer(width, height)
	if err != nil {
		return nil, err
	}

	return &LegacyUI{
		renderer: renderer,
		width:    width,
		height:   height,
		input:    NewInputState(),
	}, nil
}

// Begin starts UI rendering (old API compatibility).
func (u *LegacyUI) Begin() {
	gl.Disable(gl.DEPTH_TEST)
	gl.Disable(gl.CULL_FACE)
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)

	gl.UseProgram(u.renderer.shader)

	// Orthographic projection
	proj := ortho(0, float32(u.width), float32(u.height), 0, -1, 1)
	gl.UniformMatrix4fv(u.renderer.projLoc, 1, false, &proj[0])
}

// End finishes UI rendering (old API compatibility).
func (u *LegacyUI) End() {
	gl.Enable(gl.DEPTH_TEST)
	gl.Enable(gl.CULL_FACE)
}

// DrawText renders text at the specified position (old API compatibility).
func (u *LegacyUI) DrawText(x, y float32, text string, r, g, b, a float32, scale float32) {
	if len(text) == 0 {
		return
	}

	const charWidth = float32(8)
	const charHeight = float32(8)

	charW := charWidth * scale
	charH := charHeight * scale

	vertices := make([]LegacyVertex, 0, len(text)*6)

	for i, ch := range text {
		if ch < 32 || ch > 127 {
			ch = '?'
		}

		idx := int(ch - 32)
		col := float32(idx % 16)
		row := float32(idx / 16)

		// Texture coordinates (16x6 grid of 8x8 chars in 128x48 texture)
		u0 := col * 8 / 128
		v0 := row * 8 / 48
		u1 := (col + 1) * 8 / 128
		v1 := (row + 1) * 8 / 48

		px := x + float32(i)*charW
		py := y

		vertices = append(vertices,
			LegacyVertex{Pos: [2]float32{px, py}, TexCoord: [2]float32{u0, v0}, Color: [4]float32{r, g, b, a}},
			LegacyVertex{Pos: [2]float32{px + charW, py}, TexCoord: [2]float32{u1, v0}, Color: [4]float32{r, g, b, a}},
			LegacyVertex{Pos: [2]float32{px + charW, py + charH}, TexCoord: [2]float32{u1, v1}, Color: [4]float32{r, g, b, a}},
			LegacyVertex{Pos: [2]float32{px, py}, TexCoord: [2]float32{u0, v0}, Color: [4]float32{r, g, b, a}},
			LegacyVertex{Pos: [2]float32{px + charW, py + charH}, TexCoord: [2]float32{u1, v1}, Color: [4]float32{r, g, b, a}},
			LegacyVertex{Pos: [2]float32{px, py + charH}, TexCoord: [2]float32{u0, v1}, Color: [4]float32{r, g, b, a}},
		)
	}

	gl.Uniform1i(u.renderer.useTexLoc, 1)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, u.renderer.fontTex)
	gl.Uniform1i(u.renderer.texLoc, 0)

	gl.BindVertexArray(u.renderer.vao)
	gl.BindBuffer(gl.ARRAY_BUFFER, u.renderer.vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(vertices)*int(unsafe.Sizeof(LegacyVertex{})), gl.Ptr(vertices), gl.STREAM_DRAW)
	gl.DrawArrays(gl.TRIANGLES, 0, int32(len(vertices)))
	gl.BindVertexArray(0)
}

// DrawRect renders a filled rectangle (old API compatibility).
func (u *LegacyUI) DrawRect(x, y, w, h float32, r, g, b, a float32) {
	vertices := []LegacyVertex{
		{Pos: [2]float32{x, y}, Color: [4]float32{r, g, b, a}},
		{Pos: [2]float32{x + w, y}, Color: [4]float32{r, g, b, a}},
		{Pos: [2]float32{x + w, y + h}, Color: [4]float32{r, g, b, a}},
		{Pos: [2]float32{x, y}, Color: [4]float32{r, g, b, a}},
		{Pos: [2]float32{x + w, y + h}, Color: [4]float32{r, g, b, a}},
		{Pos: [2]float32{x, y + h}, Color: [4]float32{r, g, b, a}},
	}

	gl.Uniform1i(u.renderer.useTexLoc, 0)

	gl.BindVertexArray(u.renderer.vao)
	gl.BindBuffer(gl.ARRAY_BUFFER, u.renderer.vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(vertices)*int(unsafe.Sizeof(LegacyVertex{})), gl.Ptr(vertices), gl.STREAM_DRAW)
	gl.DrawArrays(gl.TRIANGLES, 0, 6)
	gl.BindVertexArray(0)
}

// DrawLine draws a line between two points (old API compatibility).
func (u *LegacyUI) DrawLine(x1, y1, x2, y2 float32, r, g, b, a float32) {
	vertices := []LegacyVertex{
		{Pos: [2]float32{x1, y1}, Color: [4]float32{r, g, b, a}},
		{Pos: [2]float32{x2, y2}, Color: [4]float32{r, g, b, a}},
	}

	gl.Uniform1i(u.renderer.useTexLoc, 0)

	gl.BindVertexArray(u.renderer.vao)
	gl.BindBuffer(gl.ARRAY_BUFFER, u.renderer.vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(vertices)*int(unsafe.Sizeof(LegacyVertex{})), gl.Ptr(vertices), gl.STREAM_DRAW)
	gl.DrawArrays(gl.LINES, 0, 2)
	gl.BindVertexArray(0)
}

// DrawLines draws multiple line segments (old API compatibility).
// Each pair of points forms a line segment.
func (u *LegacyUI) DrawLines(points [][2]float32, r, g, b, a float32) {
	if len(points) < 2 {
		return
	}

	vertices := make([]LegacyVertex, len(points))
	for i, p := range points {
		vertices[i] = LegacyVertex{Pos: p, Color: [4]float32{r, g, b, a}}
	}

	gl.Uniform1i(u.renderer.useTexLoc, 0)

	gl.BindVertexArray(u.renderer.vao)
	gl.BindBuffer(gl.ARRAY_BUFFER, u.renderer.vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(vertices)*int(unsafe.Sizeof(LegacyVertex{})), gl.Ptr(vertices), gl.STREAM_DRAW)
	gl.DrawArrays(gl.LINES, 0, int32(len(vertices)))
	gl.BindVertexArray(0)
}

// Resize updates the UI dimensions.
func (u *LegacyUI) Resize(width, height int) {
	u.width = width
	u.height = height
	u.renderer.width = width
	u.renderer.height = height
}

// Delete releases all resources.
func (u *LegacyUI) Delete() {
	if u.renderer != nil {
		u.renderer.delete()
	}
}

// newLegacyRenderer creates the OpenGL resources for legacy rendering.
func newLegacyRenderer(width, height int) (*LegacyRenderer, error) {
	r := &LegacyRenderer{
		width:  width,
		height: height,
	}

	// Create shader program
	var err error
	r.shader, err = compileLegacyShader(legacyVertexShader, legacyFragmentShader)
	if err != nil {
		return nil, err
	}

	// Get uniform locations
	r.projLoc = gl.GetUniformLocation(r.shader, gl.Str("projection\x00"))
	r.texLoc = gl.GetUniformLocation(r.shader, gl.Str("fontTexture\x00"))
	r.useTexLoc = gl.GetUniformLocation(r.shader, gl.Str("useTexture\x00"))

	// Create VAO/VBO
	gl.GenVertexArrays(1, &r.vao)
	gl.GenBuffers(1, &r.vbo)

	gl.BindVertexArray(r.vao)
	gl.BindBuffer(gl.ARRAY_BUFFER, r.vbo)

	stride := int32(unsafe.Sizeof(LegacyVertex{}))

	// Position
	gl.VertexAttribPointerWithOffset(0, 2, gl.FLOAT, false, stride, 0)
	gl.EnableVertexAttribArray(0)

	// TexCoord
	gl.VertexAttribPointerWithOffset(1, 2, gl.FLOAT, false, stride, unsafe.Offsetof(LegacyVertex{}.TexCoord))
	gl.EnableVertexAttribArray(1)

	// Color
	gl.VertexAttribPointerWithOffset(2, 4, gl.FLOAT, false, stride, unsafe.Offsetof(LegacyVertex{}.Color))
	gl.EnableVertexAttribArray(2)

	gl.BindVertexArray(0)

	// Create font texture
	r.fontTex = createLegacyFontTexture()

	return r, nil
}

func (r *LegacyRenderer) delete() {
	if r.fontTex != 0 {
		gl.DeleteTextures(1, &r.fontTex)
	}
	if r.vbo != 0 {
		gl.DeleteBuffers(1, &r.vbo)
	}
	if r.vao != 0 {
		gl.DeleteVertexArrays(1, &r.vao)
	}
	if r.shader != 0 {
		gl.DeleteProgram(r.shader)
	}
}

func compileLegacyShader(vertexSource, fragmentSource string) (uint32, error) {
	vertexShader := gl.CreateShader(gl.VERTEX_SHADER)
	csource, free := gl.Strs(vertexSource)
	gl.ShaderSource(vertexShader, 1, csource, nil)
	free()
	gl.CompileShader(vertexShader)

	var status int32
	gl.GetShaderiv(vertexShader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(vertexShader, gl.INFO_LOG_LENGTH, &logLength)
		log := make([]byte, logLength+1)
		gl.GetShaderInfoLog(vertexShader, logLength, nil, &log[0])
		gl.DeleteShader(vertexShader)
		return 0, &shaderError{msg: "vertex shader: " + string(log)}
	}

	fragmentShader := gl.CreateShader(gl.FRAGMENT_SHADER)
	csource, free = gl.Strs(fragmentSource)
	gl.ShaderSource(fragmentShader, 1, csource, nil)
	free()
	gl.CompileShader(fragmentShader)

	gl.GetShaderiv(fragmentShader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(fragmentShader, gl.INFO_LOG_LENGTH, &logLength)
		log := make([]byte, logLength+1)
		gl.GetShaderInfoLog(fragmentShader, logLength, nil, &log[0])
		gl.DeleteShader(vertexShader)
		gl.DeleteShader(fragmentShader)
		return 0, &shaderError{msg: "fragment shader: " + string(log)}
	}

	program := gl.CreateProgram()
	gl.AttachShader(program, vertexShader)
	gl.AttachShader(program, fragmentShader)
	gl.LinkProgram(program)

	gl.GetProgramiv(program, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetProgramiv(program, gl.INFO_LOG_LENGTH, &logLength)
		log := make([]byte, logLength+1)
		gl.GetProgramInfoLog(program, logLength, nil, &log[0])
		gl.DeleteShader(vertexShader)
		gl.DeleteShader(fragmentShader)
		gl.DeleteProgram(program)
		return 0, &shaderError{msg: "link: " + string(log)}
	}

	gl.DeleteShader(vertexShader)
	gl.DeleteShader(fragmentShader)

	return program, nil
}

type shaderError struct {
	msg string
}

func (e *shaderError) Error() string {
	return e.msg
}

func createLegacyFontTexture() uint32 {
	const texWidth = 128
	const texHeight = 48

	data := make([]byte, texWidth*texHeight)

	font := map[byte][]byte{
		'0':  {0x3C, 0x66, 0x6E, 0x76, 0x66, 0x66, 0x3C, 0x00},
		'1':  {0x18, 0x38, 0x18, 0x18, 0x18, 0x18, 0x7E, 0x00},
		'2':  {0x3C, 0x66, 0x06, 0x1C, 0x30, 0x60, 0x7E, 0x00},
		'3':  {0x3C, 0x66, 0x06, 0x1C, 0x06, 0x66, 0x3C, 0x00},
		'4':  {0x0C, 0x1C, 0x3C, 0x6C, 0x7E, 0x0C, 0x0C, 0x00},
		'5':  {0x7E, 0x60, 0x7C, 0x06, 0x06, 0x66, 0x3C, 0x00},
		'6':  {0x1C, 0x30, 0x60, 0x7C, 0x66, 0x66, 0x3C, 0x00},
		'7':  {0x7E, 0x06, 0x0C, 0x18, 0x30, 0x30, 0x30, 0x00},
		'8':  {0x3C, 0x66, 0x66, 0x3C, 0x66, 0x66, 0x3C, 0x00},
		'9':  {0x3C, 0x66, 0x66, 0x3E, 0x06, 0x0C, 0x38, 0x00},
		'A':  {0x18, 0x3C, 0x66, 0x66, 0x7E, 0x66, 0x66, 0x00},
		'B':  {0x7C, 0x66, 0x66, 0x7C, 0x66, 0x66, 0x7C, 0x00},
		'C':  {0x3C, 0x66, 0x60, 0x60, 0x60, 0x66, 0x3C, 0x00},
		'D':  {0x78, 0x6C, 0x66, 0x66, 0x66, 0x6C, 0x78, 0x00},
		'E':  {0x7E, 0x60, 0x60, 0x7C, 0x60, 0x60, 0x7E, 0x00},
		'F':  {0x7E, 0x60, 0x60, 0x7C, 0x60, 0x60, 0x60, 0x00},
		'G':  {0x3C, 0x66, 0x60, 0x6E, 0x66, 0x66, 0x3E, 0x00},
		'H':  {0x66, 0x66, 0x66, 0x7E, 0x66, 0x66, 0x66, 0x00},
		'I':  {0x7E, 0x18, 0x18, 0x18, 0x18, 0x18, 0x7E, 0x00},
		'J':  {0x3E, 0x0C, 0x0C, 0x0C, 0x0C, 0x6C, 0x38, 0x00},
		'K':  {0x66, 0x6C, 0x78, 0x70, 0x78, 0x6C, 0x66, 0x00},
		'L':  {0x60, 0x60, 0x60, 0x60, 0x60, 0x60, 0x7E, 0x00},
		'M':  {0x63, 0x77, 0x7F, 0x6B, 0x63, 0x63, 0x63, 0x00},
		'N':  {0x66, 0x76, 0x7E, 0x7E, 0x6E, 0x66, 0x66, 0x00},
		'O':  {0x3C, 0x66, 0x66, 0x66, 0x66, 0x66, 0x3C, 0x00},
		'P':  {0x7C, 0x66, 0x66, 0x7C, 0x60, 0x60, 0x60, 0x00},
		'Q':  {0x3C, 0x66, 0x66, 0x66, 0x6A, 0x6C, 0x36, 0x00},
		'R':  {0x7C, 0x66, 0x66, 0x7C, 0x6C, 0x66, 0x66, 0x00},
		'S':  {0x3C, 0x66, 0x60, 0x3C, 0x06, 0x66, 0x3C, 0x00},
		'T':  {0x7E, 0x18, 0x18, 0x18, 0x18, 0x18, 0x18, 0x00},
		'U':  {0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x3C, 0x00},
		'V':  {0x66, 0x66, 0x66, 0x66, 0x66, 0x3C, 0x18, 0x00},
		'W':  {0x63, 0x63, 0x63, 0x6B, 0x7F, 0x77, 0x63, 0x00},
		'X':  {0x66, 0x66, 0x3C, 0x18, 0x3C, 0x66, 0x66, 0x00},
		'Y':  {0x66, 0x66, 0x66, 0x3C, 0x18, 0x18, 0x18, 0x00},
		'Z':  {0x7E, 0x06, 0x0C, 0x18, 0x30, 0x60, 0x7E, 0x00},
		'a':  {0x00, 0x00, 0x3C, 0x06, 0x3E, 0x66, 0x3E, 0x00},
		'b':  {0x60, 0x60, 0x7C, 0x66, 0x66, 0x66, 0x7C, 0x00},
		'c':  {0x00, 0x00, 0x3C, 0x66, 0x60, 0x66, 0x3C, 0x00},
		'd':  {0x06, 0x06, 0x3E, 0x66, 0x66, 0x66, 0x3E, 0x00},
		'e':  {0x00, 0x00, 0x3C, 0x66, 0x7E, 0x60, 0x3C, 0x00},
		'f':  {0x1C, 0x30, 0x30, 0x7C, 0x30, 0x30, 0x30, 0x00},
		'g':  {0x00, 0x00, 0x3E, 0x66, 0x66, 0x3E, 0x06, 0x3C},
		'h':  {0x60, 0x60, 0x7C, 0x66, 0x66, 0x66, 0x66, 0x00},
		'i':  {0x18, 0x00, 0x38, 0x18, 0x18, 0x18, 0x3C, 0x00},
		'j':  {0x0C, 0x00, 0x1C, 0x0C, 0x0C, 0x0C, 0x6C, 0x38},
		'k':  {0x60, 0x60, 0x66, 0x6C, 0x78, 0x6C, 0x66, 0x00},
		'l':  {0x38, 0x18, 0x18, 0x18, 0x18, 0x18, 0x3C, 0x00},
		'm':  {0x00, 0x00, 0x76, 0x7F, 0x6B, 0x6B, 0x63, 0x00},
		'n':  {0x00, 0x00, 0x7C, 0x66, 0x66, 0x66, 0x66, 0x00},
		'o':  {0x00, 0x00, 0x3C, 0x66, 0x66, 0x66, 0x3C, 0x00},
		'p':  {0x00, 0x00, 0x7C, 0x66, 0x66, 0x7C, 0x60, 0x60},
		'q':  {0x00, 0x00, 0x3E, 0x66, 0x66, 0x3E, 0x06, 0x06},
		'r':  {0x00, 0x00, 0x6C, 0x76, 0x60, 0x60, 0x60, 0x00},
		's':  {0x00, 0x00, 0x3E, 0x60, 0x3C, 0x06, 0x7C, 0x00},
		't':  {0x30, 0x30, 0x7C, 0x30, 0x30, 0x30, 0x1C, 0x00},
		'u':  {0x00, 0x00, 0x66, 0x66, 0x66, 0x66, 0x3E, 0x00},
		'v':  {0x00, 0x00, 0x66, 0x66, 0x66, 0x3C, 0x18, 0x00},
		'w':  {0x00, 0x00, 0x63, 0x6B, 0x6B, 0x7F, 0x36, 0x00},
		'x':  {0x00, 0x00, 0x66, 0x3C, 0x18, 0x3C, 0x66, 0x00},
		'y':  {0x00, 0x00, 0x66, 0x66, 0x66, 0x3E, 0x06, 0x3C},
		'z':  {0x00, 0x00, 0x7E, 0x0C, 0x18, 0x30, 0x7E, 0x00},
		' ':  {0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		'.':  {0x00, 0x00, 0x00, 0x00, 0x00, 0x18, 0x18, 0x00},
		',':  {0x00, 0x00, 0x00, 0x00, 0x00, 0x18, 0x18, 0x30},
		':':  {0x00, 0x00, 0x18, 0x18, 0x00, 0x18, 0x18, 0x00},
		';':  {0x00, 0x00, 0x18, 0x18, 0x00, 0x18, 0x18, 0x30},
		'=':  {0x00, 0x00, 0x7E, 0x00, 0x7E, 0x00, 0x00, 0x00},
		'-':  {0x00, 0x00, 0x00, 0x7E, 0x00, 0x00, 0x00, 0x00},
		'+':  {0x00, 0x18, 0x18, 0x7E, 0x18, 0x18, 0x00, 0x00},
		'[':  {0x1C, 0x18, 0x18, 0x18, 0x18, 0x18, 0x1C, 0x00},
		']':  {0x38, 0x18, 0x18, 0x18, 0x18, 0x18, 0x38, 0x00},
		'>':  {0x60, 0x30, 0x18, 0x0C, 0x18, 0x30, 0x60, 0x00},
		'<':  {0x06, 0x0C, 0x18, 0x30, 0x18, 0x0C, 0x06, 0x00},
		'/':  {0x02, 0x06, 0x0C, 0x18, 0x30, 0x60, 0x40, 0x00},
		'\\': {0x40, 0x60, 0x30, 0x18, 0x0C, 0x06, 0x02, 0x00},
		'_':  {0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x7E, 0x00},
		'(':  {0x0C, 0x18, 0x30, 0x30, 0x30, 0x18, 0x0C, 0x00},
		')':  {0x30, 0x18, 0x0C, 0x0C, 0x0C, 0x18, 0x30, 0x00},
		'*':  {0x00, 0x66, 0x3C, 0xFF, 0x3C, 0x66, 0x00, 0x00},
		'|':  {0x18, 0x18, 0x18, 0x18, 0x18, 0x18, 0x18, 0x00},
		'?':  {0x3C, 0x66, 0x06, 0x1C, 0x18, 0x00, 0x18, 0x00},
		'!':  {0x18, 0x18, 0x18, 0x18, 0x18, 0x00, 0x18, 0x00},
		'@':  {0x3C, 0x66, 0x6E, 0x6A, 0x6E, 0x60, 0x3C, 0x00},
		'#':  {0x24, 0x7E, 0x24, 0x24, 0x7E, 0x24, 0x00, 0x00},
		'$':  {0x18, 0x3E, 0x60, 0x3C, 0x06, 0x7C, 0x18, 0x00},
		'%':  {0x62, 0x64, 0x08, 0x10, 0x26, 0x46, 0x00, 0x00},
		'^':  {0x18, 0x3C, 0x66, 0x00, 0x00, 0x00, 0x00, 0x00},
		'&':  {0x38, 0x6C, 0x38, 0x76, 0xDC, 0xCC, 0x76, 0x00},
		'\'': {0x18, 0x18, 0x30, 0x00, 0x00, 0x00, 0x00, 0x00},
		'"':  {0x66, 0x66, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		'`':  {0x30, 0x18, 0x0C, 0x00, 0x00, 0x00, 0x00, 0x00},
		'~':  {0x00, 0x00, 0x76, 0xDC, 0x00, 0x00, 0x00, 0x00},
		'{':  {0x0E, 0x18, 0x18, 0x70, 0x18, 0x18, 0x0E, 0x00},
		'}':  {0x70, 0x18, 0x18, 0x0E, 0x18, 0x18, 0x70, 0x00},
	}

	for ch, pattern := range font {
		if ch < 32 || ch > 127 {
			continue
		}
		idx := int(ch - 32)
		col := idx % 16
		row := idx / 16

		for y := 0; y < 8; y++ {
			for x := 0; x < 8; x++ {
				px := col*8 + x
				py := row*8 + y
				if pattern[y]&(0x80>>x) != 0 {
					data[py*texWidth+px] = 255
				}
			}
		}
	}

	var tex uint32
	gl.GenTextures(1, &tex)
	gl.BindTexture(gl.TEXTURE_2D, tex)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RED, texWidth, texHeight, 0, gl.RED, gl.UNSIGNED_BYTE, gl.Ptr(data))
	gl.BindTexture(gl.TEXTURE_2D, 0)

	return tex
}

func ortho(left, right, bottom, top, near, far float32) [16]float32 {
	return [16]float32{
		2 / (right - left), 0, 0, 0,
		0, 2 / (top - bottom), 0, 0,
		0, 0, -2 / (far - near), 0,
		-(right + left) / (right - left), -(top + bottom) / (top - bottom), -(far + near) / (far - near), 1,
	}
}
