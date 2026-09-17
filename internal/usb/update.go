package usb

import "context"

// Update simulates the state-preserving micrOS USB update flow.
func (m DummyManager) Update(ctx context.Context, target Target, observers ...func([]Stage)) (Result, error) {
	return m.runWorkflow(ctx, Result{
		Operation: "update",
		Target:    target,
		Summary:   "Dummy update completed; node configuration was preserved.",
		Log: []string{
			"Read node_config.json (simulated)",
			"Compared installed and release versions (simulated)",
			"Updated firmware and release resources (simulated)",
			"Restored node_config.json (simulated)",
		},
	}, observers)
}
