package gui

import (
	"strings"
	"unicode"
)

// TextWrapMode specifies how text should be wrapped.
type TextWrapMode int

const (
	// WrapModeWord wraps at word boundaries (default for Latin text).
	WrapModeWord TextWrapMode = iota
	// WrapModeChar wraps at character boundaries (for CJK or dense text).
	WrapModeChar
	// WrapModeAuto detects text type and chooses appropriate mode.
	WrapModeAuto
)

// WrapText wraps text to fit within maxWidth using the specified mode.
// Returns a slice of lines.
func WrapText(ctx *Context, text string, maxWidth float32, mode TextWrapMode) []string {
	if maxWidth <= 0 {
		return []string{text}
	}

	// Choose wrap mode if auto
	if mode == WrapModeAuto {
		if containsCJK(text) {
			mode = WrapModeChar
		} else {
			mode = WrapModeWord
		}
	}

	switch mode {
	case WrapModeWord:
		return wrapByWord(ctx, text, maxWidth)
	case WrapModeChar:
		return wrapByChar(ctx, text, maxWidth)
	default:
		return wrapByWord(ctx, text, maxWidth)
	}
}

// wrapByWord wraps text at word boundaries.
func wrapByWord(ctx *Context, text string, maxWidth float32) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}

	var lines []string
	var currentLine string

	for _, word := range words {
		testLine := currentLine
		if testLine != "" {
			testLine += " "
		}
		testLine += word

		width := ctx.MeasureText(testLine).X
		if width > maxWidth && currentLine != "" {
			// Line is too long, start a new line
			lines = append(lines, currentLine)
			currentLine = word
		} else {
			currentLine = testLine
		}
	}

	// Add the last line
	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return lines
}

// wrapByChar wraps text at character boundaries (for CJK text).
func wrapByChar(ctx *Context, text string, maxWidth float32) []string {
	runes := []rune(text)
	if len(runes) == 0 {
		return nil
	}

	var lines []string
	var currentLine []rune

	for _, r := range runes {
		testLine := append(currentLine, r)
		width := ctx.MeasureText(string(testLine)).X

		if width > maxWidth && len(currentLine) > 0 {
			// Line is too long, start a new line
			lines = append(lines, string(currentLine))
			currentLine = []rune{r}
		} else {
			currentLine = testLine
		}
	}

	// Add the last line
	if len(currentLine) > 0 {
		lines = append(lines, string(currentLine))
	}

	return lines
}

// WrapTextSmart wraps text using smart word/character detection.
// Latin text wraps at word boundaries, CJK text wraps at character boundaries.
// Mixed text handles each segment appropriately.
func WrapTextSmart(ctx *Context, text string, maxWidth float32) []string {
	segments := splitByScript(text)
	if len(segments) == 0 {
		return nil
	}

	var lines []string
	var currentLine string

	for _, seg := range segments {
		mode := WrapModeWord
		if seg.isCJK {
			mode = WrapModeChar
		}

		// Wrap this segment
		segLines := WrapText(ctx, seg.text, maxWidth, mode)

		for i, line := range segLines {
			if i == 0 && currentLine != "" {
				// Try to fit first line of segment on current line
				testLine := currentLine + line
				if ctx.MeasureText(testLine).X <= maxWidth {
					currentLine = testLine
					continue
				}
				// Doesn't fit, finalize current line
				lines = append(lines, currentLine)
				currentLine = line
			} else if i == 0 {
				currentLine = line
			} else {
				// Additional lines from segment
				if currentLine != "" {
					lines = append(lines, currentLine)
				}
				currentLine = line
			}
		}
	}

	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return lines
}

// textSegment represents a segment of text with uniform script type.
type textSegment struct {
	text  string
	isCJK bool
}

// splitByScript splits text into segments of CJK and non-CJK characters.
func splitByScript(text string) []textSegment {
	runes := []rune(text)
	if len(runes) == 0 {
		return nil
	}

	var segments []textSegment
	var currentRunes []rune
	currentIsCJK := isCJKRune(runes[0])

	for _, r := range runes {
		runeIsCJK := isCJKRune(r)

		if runeIsCJK != currentIsCJK && len(currentRunes) > 0 {
			segments = append(segments, textSegment{
				text:  string(currentRunes),
				isCJK: currentIsCJK,
			})
			currentRunes = nil
			currentIsCJK = runeIsCJK
		}

		currentRunes = append(currentRunes, r)
	}

	if len(currentRunes) > 0 {
		segments = append(segments, textSegment{
			text:  string(currentRunes),
			isCJK: currentIsCJK,
		})
	}

	return segments
}

// containsCJK returns true if the string contains any CJK characters.
func containsCJK(text string) bool {
	for _, r := range text {
		if isCJKRune(r) {
			return true
		}
	}
	return false
}

// isCJKRune returns true if the rune is a CJK character.
func isCJKRune(r rune) bool {
	return unicode.Is(unicode.Han, r) ||
		unicode.Is(unicode.Hiragana, r) ||
		unicode.Is(unicode.Katakana, r) ||
		unicode.Is(unicode.Hangul, r) ||
		unicode.In(r, unicode.Bopomofo) ||
		unicode.In(r, unicode.Yi)
}

// TruncateText truncates text to fit within maxWidth, adding ellipsis if needed.
func TruncateText(ctx *Context, text string, maxWidth float32) string {
	return TruncateTextWithSuffix(ctx, text, maxWidth, "..")
}

// TruncateTextWithSuffix truncates text and adds a custom suffix.
func TruncateTextWithSuffix(ctx *Context, text string, maxWidth float32, suffix string) string {
	if ctx.MeasureText(text).X <= maxWidth {
		return text
	}

	runes := []rune(text)
	suffixWidth := ctx.MeasureText(suffix).X
	targetWidth := maxWidth - suffixWidth

	for len(runes) > 0 {
		truncated := string(runes) + suffix
		if ctx.MeasureText(string(runes)).X <= targetWidth {
			return truncated
		}
		runes = runes[:len(runes)-1]
	}

	return suffix
}

// TextWidthEllipsis returns text that fits within maxWidth, with ellipsis.
// Unlike TruncateText, this also works with very small widths.
func TextWidthEllipsis(ctx *Context, text string, maxWidth float32) string {
	if maxWidth <= 0 {
		return ""
	}

	if ctx.MeasureText(text).X <= maxWidth {
		return text
	}

	// Try with ".."
	result := TruncateTextWithSuffix(ctx, text, maxWidth, "..")
	if ctx.MeasureText(result).X <= maxWidth {
		return result
	}

	// Fallback to single dot
	result = TruncateTextWithSuffix(ctx, text, maxWidth, ".")
	if ctx.MeasureText(result).X <= maxWidth {
		return result
	}

	// Very small width, return nothing
	return ""
}

// MeasureWrappedText returns the size of text when wrapped to maxWidth.
func MeasureWrappedText(ctx *Context, text string, maxWidth float32, mode TextWrapMode) Vec2 {
	lines := WrapText(ctx, text, maxWidth, mode)
	if len(lines) == 0 {
		return Vec2{}
	}

	lineHeight := ctx.lineHeight()
	maxLineWidth := float32(0)

	for _, line := range lines {
		w := ctx.MeasureText(line).X
		if w > maxLineWidth {
			maxLineWidth = w
		}
	}

	return Vec2{
		X: maxLineWidth,
		Y: float32(len(lines)) * lineHeight,
	}
}
