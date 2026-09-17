package usb

import (
	"context"
	"testing"
	"time"
)

func TestWorkflowStages(t *testing.T) {
	m := DummyManager{Delay: time.Millisecond}
	for _, operation := range []string{"install", "update"} {
		t.Run(operation, func(t *testing.T) {
			var snapshots [][]Stage
			emit := func(stages []Stage) { snapshots = append(snapshots, stages) }
			run := m.Install
			if operation == "update" {
				run = m.Update
			}
			result, err := run(context.Background(), Target{Image: Image{Board: "esp32c3"}}, emit)
			if err != nil {
				t.Fatal(err)
			}
			if len(snapshots) != 1+2*len(result.Log) {
				t.Fatal("missing stage transitions")
			}
			for i := range result.Log {
				if snapshots[0][i].State != StageWaiting || snapshots[1+2*i][i].State != StageRunning || snapshots[2+2*i][i].State != StageDone {
					t.Fatalf("incorrect transitions for stage %d", i)
				}
			}
		})
	}
}

func TestWorkflowCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var last []Stage
	_, err := (DummyManager{}).Install(ctx, Target{}, func(stages []Stage) { last = stages })
	if err != context.Canceled || last[0].State != StageFailed || last[1].State != StageWaiting {
		t.Fatal("cancelled stage should fail without completing later stages")
	}
}
