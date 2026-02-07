package gui

// ClipboardProvider abstracts system clipboard access.
// Implement this interface with platform-specific clipboard APIs.
//
// For GLFW:
//
//	type GLFWClipboard struct {
//	    window *glfw.Window
//	}
//
//	func (c *GLFWClipboard) GetText() string {
//	    return c.window.GetClipboardString()
//	}
//
//	func (c *GLFWClipboard) SetText(text string) {
//	    c.window.SetClipboardString(text)
//	}
type ClipboardProvider interface {
	// GetText retrieves text from the system clipboard.
	// Returns empty string if clipboard is empty or contains non-text data.
	GetText() string

	// SetText copies text to the system clipboard.
	SetText(text string)
}

// Global clipboard provider (set by application during initialization).
var clipboardProvider ClipboardProvider

// SetClipboardProvider sets the global clipboard provider.
// Call this during application initialization with a platform-specific implementation.
//
// Example with GLFW:
//
//	gui.SetClipboardProvider(&GLFWClipboard{window: window})
func SetClipboardProvider(cp ClipboardProvider) {
	clipboardProvider = cp
}

// GetClipboardProvider returns the current clipboard provider, or nil if not set.
func GetClipboardProvider() ClipboardProvider {
	return clipboardProvider
}

// ClipboardGetText retrieves text from the clipboard.
// Returns empty string if no clipboard provider is set or clipboard is empty.
func ClipboardGetText() string {
	if clipboardProvider != nil {
		return clipboardProvider.GetText()
	}
	return ""
}

// ClipboardSetText copies text to the clipboard.
// Does nothing if no clipboard provider is set.
func ClipboardSetText(text string) {
	if clipboardProvider != nil {
		clipboardProvider.SetText(text)
	}
}

// ClipboardAvailable returns true if a clipboard provider is configured.
func ClipboardAvailable() bool {
	return clipboardProvider != nil
}
