package usb

import (
	"context"
	"strings"
)

// Install simulates a clean USB deployment without opening the serial port.
func (m DummyManager) Install(ctx context.Context, target Target, observers ...func([]Stage)) (Result, error) {
	return m.runWorkflow(ctx, Result{
		Operation: "install",
		Target:    target,
		Summary:   "Dummy install completed; no hardware was changed.",
		Log: []string{
			"Detected bootloader: " + strings.ToUpper(target.Image.Board) + " (simulated)",
			"Erased flash (simulated)",
			"Wrote and verified firmware image (simulated)",
			"Installed micrOS release resources (simulated)",
		},
	}, observers)
}
