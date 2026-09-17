package usb

import "context"

type StageState string

const (
	StageWaiting StageState = "waiting"
	StageRunning StageState = "running"
	StageDone    StageState = "done"
	StageFailed  StageState = "failed"
)

type Stage struct {
	Name  string
	State StageState
}

// runWorkflow simulates each step and publishes immutable progress snapshots.
func (m DummyManager) runWorkflow(ctx context.Context, result Result, observers []func([]Stage)) (Result, error) {
	stages := make([]Stage, len(result.Log))
	for i, name := range result.Log {
		stages[i] = Stage{Name: name, State: StageWaiting}
	}
	emit := func() {
		for _, observer := range observers {
			if observer != nil {
				observer(append([]Stage(nil), stages...))
			}
		}
	}
	emit()
	for i := range stages {
		stages[i].State = StageRunning
		emit()
		if err := wait(ctx, m.delay()); err != nil {
			stages[i].State = StageFailed
			emit()
			return Result{}, err
		}
		stages[i].State = StageDone
		emit()
	}
	return result, nil
}
