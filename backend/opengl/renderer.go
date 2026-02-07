// Package opengl provides an OpenGL 4.1 backend for the GUI package.
package opengl

import (
	"fmt"
	"unsafe"

	"github.com/go-gl/gl/v4.1-core/gl"

	"github.com/go-theft-auto/gui"
)

// Renderer implements gui rendering using OpenGL.
type Renderer struct {
	shader       uint32
	vao, vbo     uint32
	ebo          uint32
	fontTex      uint32
	projLoc      int32
	texLoc       int32
	useTexLoc    int32
	isRGBATexLoc int32 // Uniform for RGBA vs alpha-only texture mode
	width        int
	height       int

	// Track which textures are RGBA (vs alpha-only)
	rgbaTextures map[uint32]bool
}

// Vertex shader source
const vertexShaderSource = `
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

// Fragment shader source
// Supports two texture modes:
// - Alpha-only (R-channel): For built-in bitmap font and system fonts
// - RGBA: For GTA fonts and plate fonts (full color textures)
const fragmentShaderSource = `
#version 410 core
in vec2 TexCoord;
in vec4 Color;

out vec4 FragColor;

uniform sampler2D fontTexture;
uniform bool useTexture;
uniform bool isRGBATexture;

void main() {
    if (useTexture) {
        vec4 texColor = texture(fontTexture, TexCoord);
        if (isRGBATexture) {
            // RGBA font: use texture color modulated by vertex color
            FragColor = texColor * Color;
        } else {
            // Alpha-only font: R channel is alpha, use vertex color for RGB
            FragColor = vec4(Color.rgb, Color.a * texColor.r);
        }
    } else {
        FragColor = Color;
    }
}
` + "\x00"

// NewRenderer creates a new OpenGL GUI renderer.
func NewRenderer(width, height int) (*Renderer, error) {
	r := &Renderer{
		width:        width,
		height:       height,
		rgbaTextures: make(map[uint32]bool),
	}

	// Create shader program
	var err error
	r.shader, err = createShaderProgram(vertexShaderSource, fragmentShaderSource)
	if err != nil {
		return nil, fmt.Errorf("failed to create shader: %w", err)
	}

	// Get uniform locations
	r.projLoc = gl.GetUniformLocation(r.shader, gl.Str("projection\x00"))
	r.texLoc = gl.GetUniformLocation(r.shader, gl.Str("fontTexture\x00"))
	r.useTexLoc = gl.GetUniformLocation(r.shader, gl.Str("useTexture\x00"))
	r.isRGBATexLoc = gl.GetUniformLocation(r.shader, gl.Str("isRGBATexture\x00"))

	// Create VAO
	gl.GenVertexArrays(1, &r.vao)
	gl.BindVertexArray(r.vao)

	// Create VBO
	gl.GenBuffers(1, &r.vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, r.vbo)

	// Create EBO
	gl.GenBuffers(1, &r.ebo)
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, r.ebo)

	// Vertex layout: Pos (2 floats) + TexCoord (2 floats) + Color (1 uint32)
	stride := int32(unsafe.Sizeof(gui.Vertex{}))

	// Position attribute
	gl.VertexAttribPointerWithOffset(0, 2, gl.FLOAT, false, stride, 0)
	gl.EnableVertexAttribArray(0)

	// TexCoord attribute
	gl.VertexAttribPointerWithOffset(1, 2, gl.FLOAT, false, stride, unsafe.Offsetof(gui.Vertex{}.TexCoord))
	gl.EnableVertexAttribArray(1)

	// Color attribute (normalized uint8x4)
	gl.VertexAttribPointerWithOffset(2, 4, gl.UNSIGNED_BYTE, true, stride, unsafe.Offsetof(gui.Vertex{}.Color))
	gl.EnableVertexAttribArray(2)

	gl.BindVertexArray(0)

	// Create font texture
	r.fontTex = r.createFontTexture()

	return r, nil
}

// FontTextureID returns the OpenGL texture ID for the font atlas.
func (r *Renderer) FontTextureID() uint32 {
	return r.fontTex
}

// RegisterRGBATexture marks a texture as RGBA (vs alpha-only).
// RGBA textures use all four channels for color, while alpha-only textures
// use just the R channel for alpha (tinted by vertex color).
// GTA fonts and plate fonts should be registered as RGBA.
func (r *Renderer) RegisterRGBATexture(textureID uint32) {
	r.rgbaTextures[textureID] = true
}

// UnregisterRGBATexture removes a texture from the RGBA tracking.
// Call this when a font is deleted.
func (r *Renderer) UnregisterRGBATexture(textureID uint32) {
	delete(r.rgbaTextures, textureID)
}

// Resize updates the viewport size.
func (r *Renderer) Resize(width, height int) {
	r.width = width
	r.height = height
}

// Render draws the GUI DrawList.
func (r *Renderer) Render(dl *gui.DrawList) error {
	if dl == nil || len(dl.VtxBuffer) == 0 {
		return nil
	}

	// Finalize the draw list
	dl.Finalize()

	// Save GL state
	var lastProgram int32
	var lastBlendSrc, lastBlendDst int32
	var lastScissorBox [4]int32
	var blendEnabled, depthEnabled, cullEnabled, scissorEnabled bool

	gl.GetIntegerv(gl.CURRENT_PROGRAM, &lastProgram)
	gl.GetIntegerv(gl.BLEND_SRC_ALPHA, &lastBlendSrc)
	gl.GetIntegerv(gl.BLEND_DST_ALPHA, &lastBlendDst)
	gl.GetIntegerv(gl.SCISSOR_BOX, &lastScissorBox[0])
	blendEnabled = gl.IsEnabled(gl.BLEND)
	depthEnabled = gl.IsEnabled(gl.DEPTH_TEST)
	cullEnabled = gl.IsEnabled(gl.CULL_FACE)
	scissorEnabled = gl.IsEnabled(gl.SCISSOR_TEST)

	// Setup render state
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	gl.Disable(gl.CULL_FACE)
	gl.Disable(gl.DEPTH_TEST)
	gl.Enable(gl.SCISSOR_TEST)

	// Use shader
	gl.UseProgram(r.shader)

	// Set projection matrix (orthographic)
	proj := orthoMatrix(0, float32(r.width), float32(r.height), 0, -1, 1)
	gl.UniformMatrix4fv(r.projLoc, 1, false, &proj[0])

	// Bind texture
	gl.ActiveTexture(gl.TEXTURE0)
	gl.Uniform1i(r.texLoc, 0)

	// Bind VAO and upload data
	gl.BindVertexArray(r.vao)

	gl.BindBuffer(gl.ARRAY_BUFFER, r.vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(dl.VtxBuffer)*int(unsafe.Sizeof(gui.Vertex{})),
		gl.Ptr(dl.VtxBuffer), gl.STREAM_DRAW)

	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, r.ebo)
	gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, len(dl.IdxBuffer)*2,
		gl.Ptr(dl.IdxBuffer), gl.STREAM_DRAW)

	// Execute draw commands
	for _, cmd := range dl.CmdBuffer {
		if cmd.ElemCount == 0 {
			continue
		}

		// Set clip rectangle (convert to OpenGL coordinates - Y flipped)
		clipX := int32(cmd.ClipRect[0])
		clipY := int32(float32(r.height) - cmd.ClipRect[3])
		clipW := int32(cmd.ClipRect[2] - cmd.ClipRect[0])
		clipH := int32(cmd.ClipRect[3] - cmd.ClipRect[1])

		// Clamp to screen bounds
		if clipX < 0 {
			clipW += clipX
			clipX = 0
		}
		if clipY < 0 {
			clipH += clipY
			clipY = 0
		}
		if clipW <= 0 || clipH <= 0 {
			continue
		}

		gl.Scissor(clipX, clipY, clipW, clipH)

		// Bind texture if specified
		if cmd.TextureID != 0 {
			gl.BindTexture(gl.TEXTURE_2D, cmd.TextureID)
			gl.Uniform1i(r.useTexLoc, 1)
			// Check if this texture is RGBA or alpha-only
			if r.rgbaTextures[cmd.TextureID] {
				gl.Uniform1i(r.isRGBATexLoc, 1)
			} else {
				gl.Uniform1i(r.isRGBATexLoc, 0)
			}
		} else {
			gl.Uniform1i(r.useTexLoc, 0)
			gl.Uniform1i(r.isRGBATexLoc, 0)
		}

		// Draw
		gl.DrawElementsBaseVertexWithOffset(
			gl.TRIANGLES,
			int32(cmd.ElemCount),
			gl.UNSIGNED_SHORT,
			uintptr(cmd.IndexOffset)*2,
			int32(cmd.VertexOffset),
		)
	}

	// Restore GL state
	gl.UseProgram(uint32(lastProgram))
	gl.BlendFunc(uint32(lastBlendSrc), uint32(lastBlendDst))

	if blendEnabled {
		gl.Enable(gl.BLEND)
	} else {
		gl.Disable(gl.BLEND)
	}
	if depthEnabled {
		gl.Enable(gl.DEPTH_TEST)
	} else {
		gl.Disable(gl.DEPTH_TEST)
	}
	if cullEnabled {
		gl.Enable(gl.CULL_FACE)
	} else {
		gl.Disable(gl.CULL_FACE)
	}
	if scissorEnabled {
		gl.Enable(gl.SCISSOR_TEST)
	} else {
		gl.Disable(gl.SCISSOR_TEST)
	}
	gl.Scissor(lastScissorBox[0], lastScissorBox[1], lastScissorBox[2], lastScissorBox[3])

	gl.BindVertexArray(0)

	return nil
}

// Delete releases OpenGL resources.
func (r *Renderer) Delete() {
	if r.fontTex != 0 {
		gl.DeleteTextures(1, &r.fontTex)
	}
	if r.ebo != 0 {
		gl.DeleteBuffers(1, &r.ebo)
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

// createFontTexture creates a simple bitmap font texture.
func (r *Renderer) createFontTexture() uint32 {
	// Simple 8x8 pixel font for ASCII 32-127
	// Each character is 8x8 pixels, arranged in a 16x6 grid (96 characters)
	const texWidth = 128 // 16 chars * 8 pixels
	const texHeight = 48 // 6 rows * 8 pixels

	data := make([]byte, texWidth*texHeight)

	// Define simple bitmap patterns for common characters
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

	// Fill texture with font data
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

	// Create texture
	var tex uint32
	gl.GenTextures(1, &tex)
	gl.BindTexture(gl.TEXTURE_2D, tex)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RED, texWidth, texHeight, 0, gl.RED, gl.UNSIGNED_BYTE, gl.Ptr(data))
	gl.BindTexture(gl.TEXTURE_2D, 0)

	return tex
}

// createShaderProgram compiles and links a shader program.
func createShaderProgram(vertexSource, fragmentSource string) (uint32, error) {
	// Compile vertex shader
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
		return 0, fmt.Errorf("vertex shader compilation failed: %s", string(log))
	}

	// Compile fragment shader
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
		return 0, fmt.Errorf("fragment shader compilation failed: %s", string(log))
	}

	// Link program
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
		return 0, fmt.Errorf("shader program linking failed: %s", string(log))
	}

	// Cleanup shaders (they're linked into the program now)
	gl.DeleteShader(vertexShader)
	gl.DeleteShader(fragmentShader)

	return program, nil
}

// orthoMatrix creates an orthographic projection matrix.
func orthoMatrix(left, right, bottom, top, near, far float32) [16]float32 {
	return [16]float32{
		2 / (right - left), 0, 0, 0,
		0, 2 / (top - bottom), 0, 0,
		0, 0, -2 / (far - near), 0,
		-(right + left) / (right - left), -(top + bottom) / (top - bottom), -(far + near) / (far - near), 1,
	}
}
