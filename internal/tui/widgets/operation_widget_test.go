package widgets

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/micros/microsctl/internal/usb"
)

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
