package gui

// ToastType defines the type of toast notification.
type ToastType uint8

const (
	ToastTypeInfo ToastType = iota
	ToastTypeSuccess
	ToastTypeWarning
	ToastTypeError
)

// ToastNotification represents a single toast message.
type ToastNotification struct {
	Message  string
	Type     ToastType
	Duration float32 // Total duration in seconds
	Elapsed  float32 // Time elapsed since shown
}

// ToastState holds the state for toast notifications.
// Store this in your application and pass it to DrawToasts.
type ToastState struct {
	Toasts []ToastNotification
}

// DefaultToastDuration is the default duration for toast messages.
const DefaultToastDuration float32 = 3.0

// ToastMaxVisible is the maximum number of visible toasts at once.
const ToastMaxVisible = 5

// Toast adds a toast notification with the specified type and optional duration.
// If duration is not provided, DefaultToastDuration is used.
func (ts *ToastState) Toast(message string, toastType ToastType, duration ...float32) {
	dur := DefaultToastDuration
	if len(duration) > 0 {
		dur = duration[0]
	}

	ts.Toasts = append(ts.Toasts, ToastNotification{
		Message:  message,
		Type:     toastType,
		Duration: dur,
		Elapsed:  0,
	})

	// Limit number of toasts
	if len(ts.Toasts) > ToastMaxVisible*2 {
		ts.Toasts = ts.Toasts[len(ts.Toasts)-ToastMaxVisible:]
	}
}

// ToastInfo adds an info toast.
func (ts *ToastState) ToastInfo(message string) {
	ts.Toast(message, ToastTypeInfo)
}

// ToastSuccess adds a success toast.
func (ts *ToastState) ToastSuccess(message string) {
	ts.Toast(message, ToastTypeSuccess)
}

// ToastWarning adds a warning toast.
func (ts *ToastState) ToastWarning(message string) {
	ts.Toast(message, ToastTypeWarning)
}

// ToastError adds an error toast.
func (ts *ToastState) ToastError(message string) {
	ts.Toast(message, ToastTypeError)
}

// Update advances toast timers and removes expired toasts.
// Call this once per frame with deltaTime.
func (ts *ToastState) Update(deltaTime float32) {
	// Update elapsed time and remove expired toasts
	active := ts.Toasts[:0]
	for i := range ts.Toasts {
		ts.Toasts[i].Elapsed += deltaTime
		if ts.Toasts[i].Elapsed < ts.Toasts[i].Duration {
			active = append(active, ts.Toasts[i])
		}
	}
	ts.Toasts = active
}

// DrawToasts renders all active toast notifications.
// Toasts appear in the bottom-right corner, stacked vertically.
// Call this at the end of your frame, after all other UI.
func (ctx *Context) DrawToasts(ts *ToastState) {
	if ts == nil || len(ts.Toasts) == 0 {
		return
	}

	const (
		toastPaddingX  = float32(12)
		toastPaddingY  = float32(8)
		toastMargin    = float32(10)
		toastGap       = float32(6)
		fadeInDuration = float32(0.15)
		fadeOutStart   = float32(0.7) // Start fade at 70% of duration
	)

	// Start from bottom-right corner
	baseX := ctx.DisplaySize.X - toastMargin
	baseY := ctx.DisplaySize.Y - toastMargin

	// Limit visible toasts
	startIdx := 0
	if len(ts.Toasts) > ToastMaxVisible {
		startIdx = len(ts.Toasts) - ToastMaxVisible
	}

	// Draw toasts from bottom to top
	for i := len(ts.Toasts) - 1; i >= startIdx; i-- {
		toast := &ts.Toasts[i]

		// Calculate opacity (fade in/out)
		opacity := float32(1.0)
		if toast.Elapsed < fadeInDuration {
			// Fade in
			opacity = toast.Elapsed / fadeInDuration
		} else if toast.Elapsed > toast.Duration*fadeOutStart {
			// Fade out
			fadeProgress := (toast.Elapsed - toast.Duration*fadeOutStart) / (toast.Duration * (1 - fadeOutStart))
			opacity = 1.0 - fadeProgress
		}
		if opacity <= 0 {
			continue
		}

		// Measure text size including icon
		icon := ctx.getToastIcon(toast.Type)
		iconWidth := ctx.MeasureText(icon + " ").X
		textSize := ctx.MeasureText(toast.Message)
		toastW := iconWidth + textSize.X + toastPaddingX*2
		toastH := textSize.Y + toastPaddingY*2

		// Position (bottom-right aligned)
		toastX := baseX - toastW
		toastY := baseY - toastH

		// Get background color based on type
		bgColor := ctx.getToastColor(toast.Type)

		// Apply opacity
		r, g, b, _ := UnpackRGBA(bgColor)
		bgColor = RGBA(r, g, b, uint8(float32(230)*opacity))

		// Draw background
		ctx.DrawList.AddRect(toastX, toastY, toastW, toastH, bgColor)

		// Draw border (subtle)
		borderColor := RGBA(255, 255, 255, uint8(float32(60)*opacity))
		ctx.DrawList.AddRectOutline(toastX, toastY, toastW, toastH, borderColor, 1)

		// Draw icon
		iconColor := RGBA(255, 255, 255, uint8(float32(255)*opacity))
		ctx.addText(toastX+toastPaddingX, toastY+toastPaddingY, icon+" ", iconColor)

		// Draw message
		textColor := RGBA(255, 255, 255, uint8(float32(255)*opacity))
		ctx.addText(toastX+toastPaddingX+iconWidth, toastY+toastPaddingY, toast.Message, textColor)

		// Move up for next toast
		baseY -= toastH + toastGap
	}
}

// getToastColor returns the background color for a toast type.
func (ctx *Context) getToastColor(t ToastType) uint32 {
	switch t {
	case ToastTypeSuccess:
		return ctx.style.ToastSuccessColor
	case ToastTypeWarning:
		return ctx.style.ToastWarningColor
	case ToastTypeError:
		return ctx.style.ToastErrorColor
	default:
		return ctx.style.ToastInfoColor
	}
}

// getToastIcon returns the icon character for a toast type.
func (ctx *Context) getToastIcon(t ToastType) string {
	switch t {
	case ToastTypeSuccess:
		return "+"
	case ToastTypeWarning:
		return "!"
	case ToastTypeError:
		return "X"
	default:
		return "i"
	}
}
