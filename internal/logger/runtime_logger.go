package logger

import (
	"context"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Live emits a short, human-readable activity message to the frontend so the UI
// can show what the agent is doing (reading a file, running a command, etc.).
// Listen on the "live-message" event in the frontend.
func Live(ctx context.Context, message string) {
	if ctx == nil || message == "" {
		return
	}
	runtime.EventsEmit(ctx, "live-message", message)
}
