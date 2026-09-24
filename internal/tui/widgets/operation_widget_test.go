package widgets

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/micros/microsctl/internal/usb"
)

func TestOperationGuidanceFollowsLifecycle(t *testing.T) {
	for _, operation := range []string{"install", "update"} {
		for _, phase := range []string{"running", "complete", "failed", "stopped"} {
			t.Run(operation+"/"+phase, func(t *testing.T) {
				state := State{Operation: operation, Running: phase == "running"}
				if phase == "complete" {
					state.Result = &usb.Result{}
					state.Progress = 100
				}
				if phase == "failed" {
					state.OperationError = "operation failed"
				}
				got := ansi.Strip(state.OperationWidget(100))
				if strings.Contains(got, "Do not disconnect") != state.Running {
					t.Fatalf("USB warning must only appear during active work: %s", got)
				}
				wantSetup := operation == "install" && phase == "complete"
				for _, text := range []string{"node01 Wi-Fi", "ADmin123", "Esc", "Nodes", "AP mode", "configure it."} {
					if strings.Contains(got, text) != wantSetup {
						t.Fatalf("unexpected setup guidance %q: %s", text, got)
					}
				}
			})
		}
	}
}

func TestInstallCompletionGuidanceWraps(t *testing.T) {
	state := State{Operation: "install", Result: &usb.Result{}, Progress: 100,
		Stages: []usb.Stage{{Name: "Reset", State: usb.StageDone}}}
	lines := strings.Split(ansi.Strip(state.OperationWidget(100)), "\n")
	for i, line := range lines {
		if strings.Contains(line, "1. Connect") {
			if i == 0 || !strings.Contains(lines[i-1], "Reset · done") {
				t.Fatal("setup instructions must immediately follow the final stage")
			}
			for _, instruction := range lines[i : i+3] {
				if !strings.HasPrefix(instruction, "│    ") {
					t.Fatalf("setup instructions must be indented: %s", instruction)
				}
			}
		}
	}
	for i := range lines {
		lines[i] = strings.Trim(lines[i], " │")
	}
	want := "1. Connect to the node01 Wi-Fi network (default password: ADmin123)\n" +
		"2. Press Esc to return to Nodes\n" +
		"3. Open the AP mode device and configure it."
	if !strings.Contains(strings.Join(lines, "\n"), want) {
		t.Fatalf("setup instructions must appear on separate numbered lines: %s", strings.Join(lines, "\n"))
	}
	got := state.OperationWidget(48)
	wrappedText := strings.Join(strings.Fields(strings.ReplaceAll(ansi.Strip(got), "│", " ")), " ")
	if lipgloss.Width(got) != 48 || !strings.Contains(wrappedText, strings.ReplaceAll(want, "\n", " ")) {
		t.Fatalf("setup guidance must fit the panel without truncation: %s", got)
	}
}

func TestUploadDetailReplacesOneLine(t *testing.T) {
	state := State{Running: true, Operation: "install", Stages: []usb.Stage{{
		Name: "Copy resources", State: usb.StageRunning, Detail: "Uploading 1/2 · /first.py",
	}}}
	for _, width := range []int{40, 80} {
		first := state.OperationWidget(width)
		state.Stages[0].Detail = "Uploading 2/2 · /" + strings.Repeat("nested/", 20) + "main.py"
		second := state.OperationWidget(width)
		if strings.Count(second, "Uploading") != 1 || strings.Contains(second, "first.py") || !strings.Contains(second, "…") {
			t.Fatalf("expected one replaced and truncated upload line: %s", second)
		}
		if lipgloss.Height(first) != lipgloss.Height(second) {
			t.Fatal("long upload paths must not wrap or grow the progress panel")
		}
		state.Stages[0].Detail = "Uploading 1/2 · /first.py"
	}
	state.Stages[0].State = usb.StageDone
	if strings.Contains(state.OperationWidget(80), "Uploading") {
		t.Fatal("completed upload must hide the active filename")
	}
}

func TestOperationErrorWrapsWithoutTruncatingCause(t *testing.T) {
	state := State{Operation: "update", OperationError: "copy resource /modules/feature.mpy: verification failed\nbackup saved to /backups/device.zip"}
	got := state.OperationWidget(48)
	for _, text := range []string{"Error:", "feature.mpy", "verification", "device.zip"} {
		if !strings.Contains(got, text) {
			t.Fatalf("missing %q from error: %s", text, got)
		}
	}
	if strings.Contains(got, "…") || strings.Contains(got, "Do not disconnect") {
		t.Fatalf("error was truncated or shows running advice: %s", got)
	}
}
